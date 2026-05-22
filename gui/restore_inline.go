package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"pbscommon"
)

// RestoreMode picks where extracted files land on disk.
//
// Original restores back to the original filesystem location captured in the
// backup metadata sidecar (requires hostname + OS to match). The two Alternate
// modes write under opts.DestPath: Abs preserves the archive's full directory
// layout, Flat strips the longest common prefix of the user's selection so a
// single file lands at dest/<basename>.
type RestoreMode string

const (
	RestoreModeOriginal      RestoreMode = "original"
	RestoreModeAlternateAbs  RestoreMode = "alternate_abs"
	RestoreModeAlternateFlat RestoreMode = "alternate_flat"
)

// RestoreOptions contains all parameters for a restore operation.
//
// IncludePaths is the list of archive-relative paths to extract. Empty means
// "extract everything in the snapshot". Selecting a directory implies all
// descendants. Paths use forward slashes (archive style); backslashes are
// accepted and normalized.
//
// RestoreACLs / RestoreADS / RestoreTimestamps are reserved for the upcoming
// NTFS sidecar work — accepted today so the API surface is stable, but only
// RestoreTimestamps has any effect (always-on: mtime is restored). The other
// two are no-ops until the per-file .nimbus_meta sidecar lands.
type RestoreOptions struct {
	BaseURL         string
	AuthID          string
	Secret          string
	Datastore       string
	Namespace       string
	CertFingerprint string
	BackupID        string
	SnapshotTime    time.Time
	DestPath        string

	// Mode selects the destination policy. Empty defaults to alternate_abs
	// (legacy behaviour: dest + full archive path).
	Mode RestoreMode

	// AllowCrossHost permits an in-place restore even when the snapshot was
	// taken on a different machine. Honored only in RestoreModeOriginal.
	AllowCrossHost bool

	IncludePaths      []string
	Overwrite         bool
	RestoreACLs       bool // reserved — requires NTFS sidecar
	RestoreADS        bool // reserved — requires NTFS sidecar
	RestoreTimestamps bool // mtime is always restored; flag kept for symmetry

	OnProgress func(percent float64, message string)
}

// SnapshotInfo contains information about a backup snapshot.
type SnapshotInfo struct {
	BackupType string
	BackupID   string
	BackupTime time.Time
	Size       int64
	Files      []string
}

// SnapshotEntry is a single file or directory inside a snapshot, suitable for
// driving a tree view in the GUI.
type SnapshotEntry struct {
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    uint64 `json:"size"`
	ModTime int64  `json:"mtime"`
}

// ListSnapshotsInline lists available snapshots from PBS.
// SECURITY: Only lists snapshots from the specified PBS server/datastore/namespace
// to prevent cross-server snapshot access.
func ListSnapshotsInline(baseURL, authID, secret, datastore, namespace, certFingerprint, backupID string) ([]SnapshotInfo, error) {
	writeBackupLog(fmt.Sprintf("Listing snapshots for backup ID: %s on %s/%s/%s", backupID, baseURL, datastore, namespace))

	client := &pbscommon.PBSClient{
		BaseURL:          baseURL,
		CertFingerPrint:  certFingerprint,
		AuthID:           authID,
		Secret:           secret,
		Datastore:        datastore,
		Namespace:        namespace,
		Insecure:         certFingerprint != "",
		CompressionLevel: pbscommon.CompressionFastest,
		Manifest: pbscommon.BackupManifest{
			BackupID: backupID,
		},
	}

	manifests, err := client.ListSnapshots()
	if err != nil {
		writeBackupLog(fmt.Sprintf("Failed to list snapshots: %v", err))
		return nil, fmt.Errorf("failed to list snapshots: %v", err)
	}

	result := make([]SnapshotInfo, 0)
	for _, m := range manifests {
		// Partial match supports split backups: searching "JDS-SRV-1" matches
		// "JDS-SRV-1_D_DATA" or "JDS-SRV-1_PART-A".
		if backupID != "" && !strings.Contains(m.BackupID, backupID) {
			continue
		}

		info := SnapshotInfo{
			BackupType: m.BackupType,
			BackupID:   m.BackupID,
			BackupTime: time.Unix(m.BackupTime, 0),
			Size:       0,
			Files:      make([]string, 0, len(m.Files)),
		}
		for _, f := range m.Files {
			info.Files = append(info.Files, f.Filename)
		}
		result = append(result, info)
	}

	writeBackupLog(fmt.Sprintf("Found %d snapshots", len(result)))
	return result, nil
}

// withSnapshotReader opens a snapshot archive over a LAZY chunk-backed reader and
// hands it to fn. Chunks are fetched from PBS on demand (with an LRU cache) as fn
// reads, instead of downloading and reassembling the whole archive into a temp
// file first. This removes the "free %TEMP% space == archive size" requirement
// and lets selective restore skip the chunks of files the caller did not select.
// The PBS reader session stays open for the duration of fn.
func withSnapshotReader(opts RestoreOptions, archiveName, logTag string, progress func(done, total int), fn func(*pbscommon.PXARReader) error) error {
	if archiveName == "" {
		archiveName = "backup.pxar.didx"
	}
	if opts.BaseURL == "" || opts.AuthID == "" || opts.Secret == "" {
		return fmt.Errorf("PBS connection parameters required")
	}
	if opts.BackupID == "" {
		return fmt.Errorf("backup ID required")
	}
	if opts.Datastore == "" {
		return fmt.Errorf("datastore required")
	}

	client := &pbscommon.PBSClient{
		BaseURL:          opts.BaseURL,
		CertFingerPrint:  opts.CertFingerprint,
		AuthID:           opts.AuthID,
		Secret:           opts.Secret,
		Datastore:        opts.Datastore,
		Namespace:        opts.Namespace,
		Insecure:         opts.CertFingerprint != "",
		CompressionLevel: pbscommon.CompressionFastest,
		Manifest: pbscommon.BackupManifest{
			BackupID:   opts.BackupID,
			BackupTime: opts.SnapshotTime.Unix(),
		},
	}
	client.Connect(true, "host")
	defer client.Close()

	ra, size, err := client.NewDIDXReaderAt(archiveName, 64, func(fetched, total int) {
		if fetched == total || fetched%32 == 0 {
			writeBackupLog(fmt.Sprintf("%s: fetched %d/%d chunks of %s", logTag, fetched, total, archiveName))
		}
		if progress != nil {
			progress(fetched, total)
		}
	})
	if err != nil {
		writeBackupLog(fmt.Sprintf("Failed to open snapshot reader (%s): %v", logTag, err))
		return fmt.Errorf("failed to open snapshot archive: %w", err)
	}

	return fn(pbscommon.NewPXARReaderAt(ra, size))
}

// buildSnapshotCacheKey is the canonical cache key for a given snapshot.
// Centralized so list/meta/restore all hit the same envelope.
func buildSnapshotCacheKey(opts RestoreOptions) snapshotCacheKey {
	return snapshotCacheKey{
		PBSID:      opts.BaseURL,
		Datastore:  opts.Datastore,
		Namespace:  opts.Namespace,
		BackupType: "host", // this client only ever creates host-type snapshots
		BackupID:   opts.BackupID,
		SnapshotAt: opts.SnapshotTime.Unix(),
	}
}

// ListSnapshotContentsInline downloads a snapshot's PXAR archive and returns
// its tree of entries (files + directories) without extracting anything to disk.
// Used by the GUI to power the restore navigation tree.
//
// Results are cached locally per snapshot — a snapshot's contents are immutable
// once written, so the cache never goes stale, only ages out. Set forceRefresh
// to bypass the cache (e.g. for a manual "Reload" button).
//
// As a side effect, the snapshot's `.nimbus_backup_meta.json` sidecar is parsed
// and cached too, so a subsequent ReadSnapshotMetaInline call is free.
//
// archiveName defaults to "backup.pxar.didx" when empty.
func ListSnapshotContentsInline(opts RestoreOptions, archiveName string, forceRefresh bool) ([]SnapshotEntry, error) {
	if archiveName == "" {
		archiveName = "backup.pxar.didx"
	}
	writeBackupLog(fmt.Sprintf("Listing contents: backupID=%s snapshot=%s archive=%s force=%v",
		opts.BackupID, opts.SnapshotTime.Format(time.RFC3339), archiveName, forceRefresh))

	cacheKey := buildSnapshotCacheKey(opts)
	if !forceRefresh {
		if cached, ok := loadSnapshotTreeCache(cacheKey); ok {
			writeBackupLog(fmt.Sprintf("Restore cache hit: %d entries (skipping download)", len(cached.Entries)))
			return cached.Entries, nil
		}
	}

	var result []SnapshotEntry
	err := withSnapshotReader(opts, archiveName, "Listing", nil, func(reader *pbscommon.PXARReader) error {
		entries, lerr := reader.ListEntries()
		if lerr != nil {
			return fmt.Errorf("failed to parse archive: %v", lerr)
		}

		result = make([]SnapshotEntry, 0, len(entries))
		for _, e := range entries {
			result = append(result, SnapshotEntry{
				Path:    e.Path,
				IsDir:   e.IsDir,
				Size:    e.Size,
				ModTime: e.ModTime,
			})
		}
		writeBackupLog(fmt.Sprintf("Listed %d entries in snapshot", len(result)))

		// Pull the meta sidecar from the same archive so the next
		// GetSnapshotMeta call doesn't re-download multi-GB of data.
		meta := tryReadBackupMeta(reader)

		// Best-effort cache write — a failure here just means the next listing
		// pays the assembly cost again.
		if werr := saveSnapshotTreeCache(cacheKey, result, meta); werr != nil {
			writeBackupLog(fmt.Sprintf("Restore cache write failed: %v", werr))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// tryReadBackupMeta extracts the .nimbus_backup_meta.json sidecar from an
// already-parsed archive. Returns nil on any failure (legacy snapshots,
// corrupted JSON, missing file) — meta is informational, never fatal.
func tryReadBackupMeta(reader *pbscommon.PXARReader) *BackupMeta {
	raw, err := reader.ReadVirtualFile(BackupMetaFilename)
	if err != nil {
		// os.ErrNotExist is expected for legacy snapshots created before the
		// sidecar shipped — log at debug volume only.
		writeBackupLog(fmt.Sprintf("Backup meta sidecar not available: %v", err))
		return nil
	}
	var meta BackupMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		writeBackupLog(fmt.Sprintf("Backup meta sidecar malformed: %v", err))
		return nil
	}
	return &meta
}

// ReadSnapshotMetaInline returns the .nimbus_backup_meta.json sidecar stored
// at the root of a snapshot, or nil with a non-nil error when no sidecar is
// present (legacy snapshots created before the sidecar shipped).
//
// Hits the local cache first — if ListSnapshotContentsInline has run for this
// snapshot, the meta is already there and no download is performed. Otherwise
// the archive is downloaded + assembled (same cost as a listing).
func ReadSnapshotMetaInline(opts RestoreOptions, forceRefresh bool) (*BackupMeta, error) {
	if opts.BaseURL == "" || opts.AuthID == "" || opts.Secret == "" {
		return nil, fmt.Errorf("PBS connection parameters required")
	}
	if opts.BackupID == "" {
		return nil, fmt.Errorf("backup ID required")
	}
	if opts.Datastore == "" {
		return nil, fmt.Errorf("datastore required")
	}

	cacheKey := buildSnapshotCacheKey(opts)
	if !forceRefresh {
		if cached, ok := loadSnapshotTreeCache(cacheKey); ok {
			if cached.Meta != nil {
				writeBackupLog("Snapshot meta: cache hit")
				return cached.Meta, nil
			}
			// Cache exists but predates meta capture (or this snapshot has no
			// sidecar). Don't bother re-downloading just to confirm — caller
			// can pass forceRefresh=true explicitly if they want to retry.
			writeBackupLog("Snapshot meta: cache hit without meta — no sidecar in this snapshot")
			return nil, nil
		}
	}

	writeBackupLog(fmt.Sprintf("Snapshot meta: downloading archive for backupID=%s snapshot=%s",
		opts.BackupID, opts.SnapshotTime.Format(time.RFC3339)))

	var meta *BackupMeta
	err := withSnapshotReader(opts, "backup.pxar.didx", "Meta", nil, func(reader *pbscommon.PXARReader) error {
		meta = tryReadBackupMeta(reader)

		// While we have the archive on disk, parse the tree too and refresh the
		// cache so a later list call avoids the same download.
		entries, lerr := reader.ListEntries()
		if lerr == nil {
			result := make([]SnapshotEntry, 0, len(entries))
			for _, e := range entries {
				result = append(result, SnapshotEntry{
					Path: e.Path, IsDir: e.IsDir, Size: e.Size, ModTime: e.ModTime,
				})
			}
			if werr := saveSnapshotTreeCache(cacheKey, result, meta); werr != nil {
				writeBackupLog(fmt.Sprintf("Restore cache write failed: %v", werr))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return meta, nil
}

// buildPathRewriter validates the requested mode against the snapshot metadata
// (when needed) and returns a rewriter that maps archive paths to filesystem
// paths on this host.
//
// Validation rules:
//   - original: requires a meta sidecar with OriginalPath. OS must match
//     runtime.GOOS (no Windows-on-Linux). Hostname must match unless
//     opts.AllowCrossHost is set.
//   - alternate_abs / alternate_flat: requires opts.DestPath.
//   - flat with no IncludePaths is equivalent to abs — flat needs a selection
//     to derive a common root from.
func buildPathRewriter(opts RestoreOptions, meta *BackupMeta) (pbscommon.PathRewriter, error) {
	mode := opts.Mode
	if mode == "" {
		mode = RestoreModeAlternateAbs
	}

	switch mode {
	case RestoreModeOriginal:
		if meta == nil {
			return nil, fmt.Errorf("restauration in-place impossible : ce snapshot n'a pas de métadonnées (.nimbus_backup_meta.json absent), choisissez « autre emplacement »")
		}
		if meta.OriginalPath == "" {
			return nil, fmt.Errorf("restauration in-place impossible : le chemin d'origine n'est pas renseigné dans les métadonnées")
		}
		if meta.OS != "" && meta.OS != runtime.GOOS {
			return nil, fmt.Errorf("restauration in-place impossible : sauvegarde faite sur %s, machine actuelle %s", meta.OS, runtime.GOOS)
		}
		if !opts.AllowCrossHost {
			localHost, err := os.Hostname()
			if err != nil {
				return nil, fmt.Errorf("impossible de lire le hostname local : %w", err)
			}
			if meta.Hostname != "" && !equalHostnames(meta.Hostname, localHost) {
				return nil, fmt.Errorf("restauration in-place bloquée : sauvegarde de %q, machine actuelle %q — cochez « forcer cross-host » si l'intention est délibérée", meta.Hostname, localHost)
			}
		}
		// Materialize the original root once, with native separators.
		root := filepath.Clean(meta.OriginalPath)
		return func(archivePath string) string {
			if archivePath == "" {
				return root
			}
			return filepath.Join(root, filepath.FromSlash(archivePath))
		}, nil

	case RestoreModeAlternateAbs:
		if opts.DestPath == "" {
			return nil, fmt.Errorf("dossier de destination requis")
		}
		dest := opts.DestPath
		return func(archivePath string) string {
			return filepath.Join(dest, filepath.FromSlash(archivePath))
		}, nil

	case RestoreModeAlternateFlat:
		if opts.DestPath == "" {
			return nil, fmt.Errorf("dossier de destination requis")
		}
		// Empty selection means "restore everything" — flat is meaningless,
		// fall back to abs so the user gets a sensible result instead of
		// hundreds of files colliding at the dest root.
		if len(opts.IncludePaths) == 0 {
			dest := opts.DestPath
			return func(archivePath string) string {
				return filepath.Join(dest, filepath.FromSlash(archivePath))
			}, nil
		}
		prefix := commonAncestorDir(pbscommon.NormalizeIncludes(opts.IncludePaths))
		dest := opts.DestPath
		return func(archivePath string) string {
			// Drop archive entries above the selection root (e.g. parent
			// directory entries the walker emits as scaffolding). The walker
			// also matches ancestors of includes for mkdir scaffolding —
			// skip those so they don't pollute the flat root.
			if archivePath == "" {
				return ""
			}
			if prefix == "" {
				return filepath.Join(dest, filepath.FromSlash(archivePath))
			}
			if archivePath == prefix {
				return ""
			}
			rel := strings.TrimPrefix(archivePath, prefix+"/")
			if rel == archivePath {
				// Not a descendant of the common prefix → ancestor scaffolding,
				// drop silently.
				return ""
			}
			return filepath.Join(dest, filepath.FromSlash(rel))
		}, nil

	default:
		return nil, fmt.Errorf("mode de restauration inconnu : %q", string(mode))
	}
}

// commonAncestorDir returns the longest path prefix shared by all includes,
// using forward-slash archive convention. Returns "" when the includes have
// no common directory (e.g. "a/x.txt" and "b/y.txt").
func commonAncestorDir(includes []string) string {
	if len(includes) == 0 {
		return ""
	}
	if len(includes) == 1 {
		// Single selection: parent dir is the common root. A single file
		// `a/b/c.txt` becomes flat as `c.txt`; a single dir `a/b/c` becomes
		// flat as `c/...` because the trim prefix is `a/b`.
		return parentDir(includes[0])
	}
	parts := strings.Split(includes[0], "/")
	for _, p := range includes[1:] {
		ps := strings.Split(p, "/")
		max := len(parts)
		if len(ps) < max {
			max = len(ps)
		}
		i := 0
		for i < max && parts[i] == ps[i] {
			i++
		}
		parts = parts[:i]
		if len(parts) == 0 {
			return ""
		}
	}
	return strings.Join(parts, "/")
}

func parentDir(archivePath string) string {
	i := strings.LastIndex(archivePath, "/")
	if i < 0 {
		return ""
	}
	return archivePath[:i]
}

// equalHostnames compares hostnames tolerant of case and trailing dot/domain
// (so "WIN-A" == "win-a" == "WIN-A.local").
func equalHostnames(a, b string) bool {
	norm := func(s string) string {
		s = strings.ToLower(s)
		if dot := strings.Index(s, "."); dot >= 0 {
			s = s[:dot]
		}
		return s
	}
	return norm(a) == norm(b)
}

// RestoreSnapshotInline restores a snapshot from PBS.
// SECURITY: Only restores from the configured PBS server/datastore/namespace.
// Snapshots from other servers will fail with HTTP 404.
//
// When opts.IncludePaths is non-empty, only the matching files and directories
// are extracted. Otherwise the whole snapshot is restored.
func RestoreSnapshotInline(opts RestoreOptions) error {
	mode := opts.Mode
	if mode == "" {
		mode = RestoreModeAlternateAbs
	}
	writeBackupLog(fmt.Sprintf("Starting restore: snapshot=%s, mode=%s, dest=%s, includes=%d, overwrite=%v, allowCrossHost=%v from %s/%s/%s",
		opts.SnapshotTime.Format("2006-01-02T15:04:05Z"), mode, opts.DestPath, len(opts.IncludePaths), opts.Overwrite, opts.AllowCrossHost,
		opts.BaseURL, opts.Datastore, opts.Namespace))

	progress := func(pct float64, msg string) {
		writeBackupLog(fmt.Sprintf("Restore progress: %.1f%% - %s", pct*100, msg))
		if opts.OnProgress != nil {
			opts.OnProgress(pct, msg)
		}
	}

	if opts.BaseURL == "" || opts.AuthID == "" || opts.Secret == "" {
		return fmt.Errorf("PBS connection parameters required")
	}
	if opts.BackupID == "" {
		return fmt.Errorf("backup ID required")
	}
	if opts.Datastore == "" {
		return fmt.Errorf("datastore required for security")
	}
	// DestPath is required for alternate modes only — original reads the
	// target from the backup metadata sidecar.
	if mode != RestoreModeOriginal && opts.DestPath == "" {
		return fmt.Errorf("destination path required")
	}

	// In-place restores need the sidecar up front to validate before we burn
	// time on the multi-GB download. Cheap if the snapshot was listed first.
	var meta *BackupMeta
	if mode == RestoreModeOriginal {
		var err error
		meta, err = ReadSnapshotMetaInline(opts, false)
		if err != nil {
			return fmt.Errorf("lecture du sidecar pour restauration in-place: %w", err)
		}
		// Validate immediately so the user sees the cross-host / cross-OS
		// refusal before any chunk is downloaded.
		if _, err := buildPathRewriter(opts, meta); err != nil {
			return err
		}
		// In-place implies overwrite. Files in OriginalPath are by definition
		// candidates for replacement; skipping them would be confusing.
		opts.Overwrite = true
		writeBackupLog(fmt.Sprintf("In-place target: %s (host=%s, os=%s)", meta.OriginalPath, meta.Hostname, meta.OS))
	}

	progress(0.05, "Preparing restore...")

	// Build the rewriter before downloading so a misconfiguration (missing
	// dest, cross-host refusal) fails fast. For alternate modes meta is unused;
	// for in-place, meta was read above before download.
	rewriter, err := buildPathRewriter(opts, meta)
	if err != nil {
		return err
	}

	progress(0.20, "Downloading backup archive...")
	// AssembleDIDXToFile downloads the .didx index and reassembles the actual
	// PXAR stream chunk-by-chunk into a temp file (bounded memory), then we walk
	// it from disk and stream each file payload to its destination.
	var extracted []pbscommon.PXARExtractedFile
	err = withSnapshotReader(opts, "backup.pxar.didx", "Restore", func(done, total int) {
		// Map chunk progress to the 0.20–0.80 portion of the overall bar.
		if total == 0 {
			return
		}
		pct := 0.20 + 0.60*(float64(done)/float64(total))
		progress(pct, fmt.Sprintf("Downloading archive (%d/%d chunks)", done, total))
	}, func(reader *pbscommon.PXARReader) error {
		progress(0.85, "Extracting files...")
		var eerr error
		extracted, eerr = reader.ExtractWithRewriter(rewriter, opts.IncludePaths, opts.Overwrite)
		if eerr != nil {
			writeBackupLog(fmt.Sprintf("PXAR extraction failed: %v", eerr))
			return fmt.Errorf("failed to extract archive: %v", eerr)
		}
		return nil
	})
	if err != nil {
		return err
	}

	successCount := 0
	skipCount := 0
	dirCount := 0
	for _, f := range extracted {
		if f.Skipped {
			skipCount++
			writeBackupLog(fmt.Sprintf("SKIPPED: %s - %s", f.Path, f.SkipReason))
		} else if f.IsDir {
			dirCount++
		} else {
			successCount++
		}
	}

	writeBackupLog(fmt.Sprintf("Extraction complete: %d files, %d dirs, %d skipped",
		successCount, dirCount, skipCount))
	progress(0.95, fmt.Sprintf("Extracted %d files", successCount))

	if opts.RestoreACLs || opts.RestoreADS {
		// Reserved options — sidecar metadata isn't written by the backup yet.
		// Log the request so it shows up in support transcripts.
		writeBackupLog("NOTE: ACL/ADS restore requested but not yet implemented (NTFS sidecar pending)")
	}

	progress(1.0, "Restore completed")

	if skipCount > 0 {
		return fmt.Errorf("restore completed with %d skipped files (see logs)", skipCount)
	}
	return nil
}
