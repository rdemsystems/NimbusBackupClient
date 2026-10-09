//go:build windows
// +build windows

package snapshot

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/st-matskevich/go-vss"
)

// shadowIDRe matches a bare VSS shadow-copy GUID (8-4-4-4-12 hex).
var shadowIDRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// normalizeShadowID strips optional braces and validates a VSS shadow ID,
// returning the bare GUID or "" if the name is not shadow-id-shaped.
func normalizeShadowID(name string) string {
	s := strings.TrimSpace(name)
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	if !shadowIDRe.MatchString(s) {
		return ""
	}
	return s
}

func SymlinkSnapshot(symlinkPath string, id string, deviceObjectPath string) (string, error) {

	snapshotSymLinkFolder := symlinkPath + "\\" + id + "\\"

	snapshotSymLinkFolder = filepath.Clean(snapshotSymLinkFolder)
	os.RemoveAll(snapshotSymLinkFolder)
	if err := os.MkdirAll(snapshotSymLinkFolder, 0700); err != nil {
		return "", fmt.Errorf("failed to create snapshot symlink folder for snapshot: %s, err: %s", id, err)
	}

	os.Remove(snapshotSymLinkFolder)

	fmt.Println("Symlink from: ", deviceObjectPath, " to: ", snapshotSymLinkFolder)

	if err := os.Symlink(deviceObjectPath, snapshotSymLinkFolder); err != nil {
		return "", fmt.Errorf("failed to create symlink from: %s to: %s, error: %s", deviceObjectPath, snapshotSymLinkFolder, err)
	}

	return snapshotSymLinkFolder, nil
}

func getAppDataFolder() (string, error) {
	// Get information about the current user
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}

	// Construct the path to the application data folder
	appDataFolder := filepath.Join(currentUser.HomeDir, "AppData", "Roaming", "PBSBackupGO")

	// Create the folder if it doesn't exist
	err = os.MkdirAll(appDataFolder, os.ModePerm)
	if err != nil {
		return "", err
	}

	return appDataFolder, nil
}

func CreateVSSSnapshot(paths []string, needFiles bool, backup_callback func(sn map[string]SnapShot) error) (retErr error) {

	// One Snapshotter per volume: go-vss rejects reuse of a single Snapshotter
	// for a second volume ("snapshotter is already in use"), which made every
	// whole-machine backup of a disk with >= 2 mounted volumes fail. Each
	// successful snapshot is held until the backup callback has consumed them,
	// then released together.
	snapshotters := make([]*vss.Snapshotter, 0, len(paths))
	// IDs of the shadows this call actually created, in creation order.
	createdIDs := make([]string, 0, len(paths))
	var leases []shadowLease
	defer func() {
		// Released last (defers run in reverse), once the shadows are gone.
		defer func() {
			for _, l := range leases {
				l.release()
			}
		}()
		for _, s := range snapshotters {
			if err := s.Release(); err != nil {
				fmt.Printf("⚠️  VSS: releasing snapshot: %v\n", err)
			}
		}
		// Cancelled or failed backup: Release() ends the writer session but
		// does not itself remove the shadow copy, so explicitly delete the
		// shadows we just created instead of leaving them (and their symlink
		// markers) behind until the next startup VSSCleanup. Only ever our own
		// IDs, and only on the error path — never a blanket `delete shadows
		// /all`.
		if retErr != nil && len(createdIDs) > 0 {
			deleteShadowsBestEffort(createdIDs)
		}
	}()
	snapshots := make(map[string]SnapShot)
	// One shadow copy per volume, shared by every path on it: the folders of
	// a volume are then frozen at the same instant (and VSS is not asked for
	// several copies of the same volume).
	byVolume := make(map[string]SnapShot)

	for _, path := range paths {
		path, _ = filepath.Abs(path)
		volName := filepath.VolumeName(path)
		volName += "\\"
		subPath := path[len(volName):] //Strp C:\, 3 chars or whatever it is

		appDataFolder, err := getAppDataFolder()
		if err != nil {
			fmt.Println("Error:", err)
			return err
		}

		if shared, ok := byVolume[strings.ToUpper(volName)]; ok {
			snapshots[path] = SnapShot{FullPath: filepath.Join(appDataFolder, "VSS", shared.Id, subPath), Id: shared.Id, ObjectPath: shared.ObjectPath, Valid: true}
			continue
		}

		fmt.Print("Creating VSS Snapshot...")

		// Check VSS writers status before creating snapshot
		checkWritersCmd := exec.Command("vssadmin", "list", "writers")
		writersOutput, _ := checkWritersCmd.CombinedOutput()
		writersStatus := string(writersOutput)

		// Log warnings for writers with known errors
		hasWriterWarnings := false
		if strings.Contains(writersStatus, "System Writer") && strings.Contains(writersStatus, "Last error") {
			fmt.Println("⚠️  WARNING: System Writer has errors - system state may not be fully captured")
			hasWriterWarnings = true
		}
		if strings.Contains(writersStatus, "NTDS") && (strings.Contains(writersStatus, "Last error") || strings.Contains(writersStatus, "0x800423f4")) {
			fmt.Println("⚠️  WARNING: NTDS Writer refuses to participate - Active Directory state will not be captured")
			hasWriterWarnings = true
		}
		if strings.Contains(writersStatus, "Dhcp") && strings.Contains(writersStatus, "Last error") {
			fmt.Println("⚠️  WARNING: DHCP Jet Writer has errors - DHCP configuration may not be captured")
			hasWriterWarnings = true
		}

		if hasWriterWarnings {
			fmt.Println("         → Backup will continue with available writers only")
			fmt.Println("         → File-level backup will work normally")
		}

		sn := &vss.Snapshotter{}
		snapshot, err := sn.CreateSnapshot(volName, false, 180)
		// VSS creates one shadow copy set at a time on the whole host: "already
		// in progress" means another requester (another backup, a restore
		// point) is creating one. Wait for it and retry; never delete shadows
		// or restart the VSS service, which destroyed every shadow copy on the
		// host (restore points, other backup tools, our other runs).
		for attempt := 0; err != nil && isShadowAlreadyInProgress(err) && attempt < len(vssBusyRetryWaits); attempt++ {
			wait := vssBusyRetryWaits[attempt]
			fmt.Printf("⚠️  VSS busy (another shadow copy is being created): %v — retrying in %s (%d/%d)\n", err, wait, attempt+1, len(vssBusyRetryWaits))
			time.Sleep(wait)
			// Do NOT Release the failed Snapshotter: go-vss already aborted and
			// released its components on the failed CreateSnapshot (without
			// nil-ing them), so a second Release would touch freed COM state.
			sn = &vss.Snapshotter{}
			snapshot, err = sn.CreateSnapshot(volName, false, 180)
		}
		if err != nil && isShadowAlreadyInProgress(err) {
			return fmt.Errorf("VSS is still busy with another shadow copy (another backup, or a VSS context stuck since a crash): %w. Retry later; if no other backup is running, restarting the \"Volume Shadow Copy\" service clears a stuck context", err)
		}
		if err != nil {
			errMsg := err.Error()
			// Check if error is ONLY due to writer failures (0x80070005 = Access Denied, 0x800423f4 = Non-retryable)
			if strings.Contains(errMsg, "0x80070005") || strings.Contains(errMsg, "0x800423f4") {
				// These are writer-specific errors, not snapshot creation errors
				// Log but DON'T fail - the snapshot might still be usable for file backup
				fmt.Printf("⚠️  VSS Writers error during snapshot creation: %v\n", err)
				fmt.Println("         → Attempting to use snapshot anyway for file-level backup")
				// snapshot might still be valid even with writer errors - check below
			} else {
				// Other errors are critical
				return fmt.Errorf("VSS snapshot creation failed: %v", err)
			}
		}

		// Verify snapshot was actually created
		if snapshot == nil || snapshot.Id == "" {
			return fmt.Errorf("VSS snapshot creation failed: no valid snapshot created")
		}

		// Snapshot is valid: track it so it is released after the backup callback.
		snapshotters = append(snapshotters, sn)

		fmt.Printf("✓ Snapshot created: %s\n", snapshot.Id)
		if hasWriterWarnings {
			fmt.Println("  Note: Snapshot created despite writer warnings - file-level backup will proceed")
		}

		_, err = SymlinkSnapshot(filepath.Join(appDataFolder, "VSS"), snapshot.Id, snapshot.DeviceObjectPath)

		if err != nil {
			return err
		}

		snapshots[path] = SnapShot{FullPath: filepath.Join(appDataFolder, "VSS", snapshot.Id, subPath), Id: snapshot.Id, ObjectPath: snapshot.DeviceObjectPath, Valid: true}
		byVolume[strings.ToUpper(volName)] = snapshots[path]
		createdIDs = append(createdIDs, snapshot.Id)
		// Lease: held open until this call returns, so VSSCleanup in another
		// process of ours (a GUI started during a CLI backup) knows the shadow
		// is in use and leaves it alone.
		if lease, lerr := holdShadowLease(filepath.Join(appDataFolder, "VSS", snapshot.Id)); lerr == nil {
			leases = append(leases, lease)
		} else {
			fmt.Printf("⚠️  VSS: could not create the in-use marker for %s: %v\n", snapshot.Id, lerr)
		}

	}

	return backup_callback(snapshots)

}

// deleteShadowsBestEffort removes shadow copies this process created during a
// backup that ended in an error or was cancelled by the user, plus the
// <appData>/VSS/<id> marker symlinks pointing at them.
//
// Best-effort by design: `vssadmin delete shadows /shadow={id}` may need
// `/for=<volume>` on some Windows versions and the shadow may already be gone
// (a normal release leaves a dangling marker). Either way we log and continue —
// a leftover shadow is cleaned up by VSSCleanup() on the next start, and we
// never fall back to `/all`, which would wipe other applications' shadows.
func deleteShadowsBestEffort(ids []string) {
	appData, appDataErr := getAppDataFolder()
	for _, id := range ids {
		norm := normalizeShadowID(id)
		if norm == "" {
			continue // not a shadow-id-shaped entry
		}
		deleteCmd := exec.Command("vssadmin", "delete", "shadows", "/shadow={"+norm+"}", "/quiet")
		if out, err := deleteCmd.CombinedOutput(); err != nil {
			fmt.Printf("⚠️  VSS: could not delete shadow %s after cancelled/failed backup: %v - %s\n", norm, err, strings.TrimSpace(string(out)))
			continue
		}
		fmt.Printf("✓ VSS: deleted shadow %s after cancelled/failed backup\n", norm)
		if appDataErr == nil {
			_ = os.Remove(filepath.Join(appData, "VSS", id))
		}
	}
}

// VSSCleanup removes orphaned VSS snapshots left by a previously crashed Proxmox Backup Client
// run. It deletes ONLY shadow copies Proxmox Backup Client created — recorded as the subfolder
// names of the <appData>/VSS symlink directory — never `vssadmin delete shadows
// /all`, which destroyed EVERY shadow copy on the host (other backup tools, DCs,
// SQL/Exchange) on each service start (audit v2-H-05).
//
// Best-effort by design: the worst case here is that an orphan remains until a
// later run, which is far safer than wiping other applications' shadow copies.
//
// WINDOWS-VERIFY: the exact `vssadmin delete shadows /shadow={id}` form may
// require `/for=<volume>` on some Windows versions; we do not persist the source
// volume. The robust long-term path is the VSS API DeleteSnapshots(by ID). If the
// invocation is rejected, the orphan simply remains (no collateral damage).
func VSSCleanup() error {
	appData, err := getAppDataFolder()
	if err != nil {
		fmt.Printf("VSS Cleanup: cannot resolve app data folder: %v\n", err)
		return nil
	}
	vssDir := filepath.Join(appData, "VSS")
	entries, err := os.ReadDir(vssDir)
	if err != nil {
		// No VSS symlink directory ⇒ no Proxmox Backup Client-created shadows to clean.
		return nil
	}

	for _, e := range entries {
		id := normalizeShadowID(e.Name())
		if id == "" {
			continue // not a shadow-id-shaped entry
		}
		marker := filepath.Join(vssDir, e.Name())

		// If the symlink no longer resolves, the shadow was already released (a
		// normal-backup leftover, not a live orphan): just drop the stale marker so
		// these don't accumulate and slow every startup.
		if _, statErr := os.Stat(marker); statErr != nil {
			_ = os.Remove(marker)
			continue
		}

		// Another process of ours (e.g. a CLI backup running while this GUI
		// starts) is still reading this shadow: not an orphan.
		if shadowInUse(marker) {
			fmt.Printf("VSS Cleanup: shadow %s is in use by another backup, kept\n", id)
			continue
		}

		// Live symlink, no live owner ⇒ a genuine orphan from a crash.
		fmt.Printf("VSS Cleanup: removing orphaned Proxmox Backup Client shadow %s...\n", id)
		deleteCmd := exec.Command("vssadmin", "delete", "shadows", "/shadow={"+id+"}", "/quiet")
		if out, derr := deleteCmd.CombinedOutput(); derr != nil {
			// Keep the marker so a later run retries; never fall back to /all.
			fmt.Printf("VSS Cleanup: could not delete shadow %s (best-effort, will retry): %v - %s\n", id, derr, string(out))
			continue
		}
		fmt.Printf("VSS Cleanup: removed Proxmox Backup Client shadow %s\n", id)
		_ = os.Remove(marker)
	}

	// NOTE: we deliberately do NOT bounce the Windows VSS service here.
	// `net stop/start VSS` affects EVERY VSS consumer on the host — on a Domain
	// Controller, or a machine running third-party backup software (Veritas Backup
	// Exec, Windows Server Backup, SQL/Exchange agents), restarting VSS can abort
	// their in-flight snapshots and corrupt their backup state. Doing it on every
	// service startup is especially hostile and runs even when no backup is due.
	// A busy VSS ("shadow copy creation already in progress") is waited for and
	// retried by CreateVSSSnapshot, never forced.
	return nil
}

// isShadowAlreadyInProgress detects the "shadow copy creation is already in
// progress" error returned by go-vss / Windows VSS when a previous
// IVssBackupComponents context is still held.
func isShadowAlreadyInProgress(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// Windows surfaces this either as VSS_E_BAD_STATE (0x8004230f) or as a
	// human-readable message containing "already in progress".
	return strings.Contains(msg, "already in progress") ||
		strings.Contains(msg, "0x8004230f")
}

// vssBusyRetryWaits are the waits between CreateSnapshot attempts while VSS
// reports that another shadow copy set is being created.
var vssBusyRetryWaits = []time.Duration{30 * time.Second, 60 * time.Second, 120 * time.Second}

// shadowLease marks a shadow copy as in use by this process: <marker>.lock is
// held open without FILE_SHARE_DELETE, so another process cannot delete it
// while we live. When the process dies, Windows closes the handle and the
// lease can be removed, which is how VSSCleanup tells an orphan from a shadow
// another run of ours is still reading.
type shadowLease struct {
	handle syscall.Handle
	path   string
}

func holdShadowLease(marker string) (shadowLease, error) {
	path := marker + ".lock"
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return shadowLease{}, err
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ|syscall.GENERIC_WRITE, syscall.FILE_SHARE_READ, nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return shadowLease{}, err
	}
	return shadowLease{handle: h, path: path}, nil
}

func (l shadowLease) release() {
	_ = syscall.CloseHandle(l.handle)
	_ = os.Remove(l.path)
}

// shadowInUse reports whether another live process holds the lease of the
// shadow behind marker. A lease left by a dead process is removed here.
func shadowInUse(marker string) bool {
	err := os.Remove(marker + ".lock")
	return err != nil && !os.IsNotExist(err)
}

// DetectControl reports whether a Linux block-snapshot control module is
// available. Always false on this platform.
func DetectControl() (SnapControl, bool) {
	return SnapControl{}, false
}
