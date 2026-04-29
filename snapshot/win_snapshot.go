//go:build windows
// +build windows

package snapshot

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/st-matskevich/go-vss"
)

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

func CreateVSSSnapshot(paths []string, backup_callback func(sn map[string]SnapShot) error) error {

	sn := vss.Snapshotter{}
	defer sn.Release()
	snapshots := make(map[string]SnapShot)

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

		snapshot, err := sn.CreateSnapshot(volName, false, 180)
		if err != nil && isShadowAlreadyInProgress(err) {
			// IVssBackupComponents stuck from a previous crashed run.
			// vssadmin delete shadows alone won't release it — we have to bounce
			// the VSS service to drop the orphaned context, then retry once.
			fmt.Printf("⚠️  VSS busy: %v\n", err)
			fmt.Println("         → Resetting VSS service state and retrying once...")
			sn.Release()
			if resetErr := vssForceReset(); resetErr != nil {
				fmt.Printf("         → VSS reset failed: %v\n", resetErr)
			}
			sn = vss.Snapshotter{}
			snapshot, err = sn.CreateSnapshot(volName, false, 180)
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

		fmt.Printf("✓ Snapshot created: %s\n", snapshot.Id)
		if hasWriterWarnings {
			fmt.Println("  Note: Snapshot created despite writer warnings - file-level backup will proceed")
		}

		_, err = SymlinkSnapshot(filepath.Join(appDataFolder, "VSS"), snapshot.Id, snapshot.DeviceObjectPath)

		if err != nil {
			return err
		}

		snapshots[path] = SnapShot{FullPath: filepath.Join(appDataFolder, "VSS", snapshot.Id, subPath), Id: snapshot.Id, ObjectPath: snapshot.DeviceObjectPath, Valid: true}

	}

	return backup_callback(snapshots)

}

// VSSCleanup removes all orphaned VSS snapshots and resets the VSS service
// state. This prevents shadow copies from accumulating after crashes or abnormal
// terminations, and clears any "shadow copy creation already in progress" lock
// held by a stuck IVssBackupComponents context from a previous run.
func VSSCleanup() error {
	// List all shadows first to check if cleanup is needed
	listCmd := exec.Command("vssadmin", "list", "shadows")
	output, err := listCmd.CombinedOutput()
	if err != nil {
		// If vssadmin fails, log but don't block service startup
		fmt.Printf("Warning: Failed to list VSS shadows: %v\n", err)
		return nil
	}

	// Only delete if there are actually shadows present
	if len(output) > 0 && !strings.Contains(string(output), "No items found") {
		fmt.Println("VSS Cleanup: Removing orphaned shadow copies...")

		// Delete all shadow copies
		// Note: This is safe because Nimbus creates snapshots only during backup
		// and releases them immediately after. Any remaining snapshots are orphans.
		deleteCmd := exec.Command("vssadmin", "delete", "shadows", "/all", "/quiet")
		deleteOutput, err := deleteCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Warning: VSS cleanup failed: %v - %s\n", err, string(deleteOutput))
		} else {
			fmt.Println("VSS Cleanup: Successfully removed orphaned snapshots")
		}
	} else {
		fmt.Println("VSS Cleanup: No orphaned snapshots found")
	}

	// Restart the VSS service to drop any stuck IVssBackupComponents context
	// from a previously crashed Nimbus process. Without this, vssadmin can
	// report 0 shadows yet the next backup still fails with
	// "VSS_START - shadow copy creation is already in progress".
	if err := restartVSSService(); err != nil {
		fmt.Printf("Warning: VSS service restart failed: %v\n", err)
	}

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

// vssForceReset is the aggressive recovery used mid-backup when a snapshot
// attempt returns "already in progress". It deletes orphan shadows then
// bounces the VSS service so the next CreateSnapshot starts from a clean
// state.
func vssForceReset() error {
	deleteCmd := exec.Command("vssadmin", "delete", "shadows", "/all", "/quiet")
	if out, err := deleteCmd.CombinedOutput(); err != nil {
		fmt.Printf("VSS reset: delete shadows warning: %v - %s\n", err, string(out))
	}
	return restartVSSService()
}

// restartVSSService bounces the Windows Volume Shadow Copy service. Safe at
// service startup and during error recovery because Nimbus is the only VSS
// consumer on backup-dedicated hosts, and stopping VSS just discards any
// in-flight shadow context (which is exactly what we want when it's stuck).
func restartVSSService() error {
	fmt.Println("VSS Cleanup: Restarting VSS service to clear stuck state...")
	stopCmd := exec.Command("net", "stop", "VSS")
	if out, err := stopCmd.CombinedOutput(); err != nil {
		// "service is not started" is fine — we'll start it next.
		if !strings.Contains(string(out), "not started") &&
			!strings.Contains(strings.ToLower(string(out)), "n'est pas démarr") {
			fmt.Printf("VSS service stop warning: %v - %s\n", err, string(out))
		}
	}
	startCmd := exec.Command("net", "start", "VSS")
	if out, err := startCmd.CombinedOutput(); err != nil {
		// "already started" is fine.
		if strings.Contains(string(out), "already been started") ||
			strings.Contains(strings.ToLower(string(out)), "déjà été démarr") {
			return nil
		}
		return fmt.Errorf("net start VSS failed: %v - %s", err, string(out))
	}
	fmt.Println("VSS Cleanup: VSS service restarted")
	return nil
}
