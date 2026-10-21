package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alphadose/haxmap"
	"pbscommon"
	"retry"
)

// legacyArchiveName is the one archive of a snapshot of the original layout
// (one folder per snapshot). A snapshot holding several folders names each
// archive after its folder instead (ArchiveNames).
const legacyArchiveName = "backup.pxar.didx"

// archiveLabel is the name a folder's archive is shown under: its file name
// without ".pxar.didx" (C_Users_tacio_Downloads.pxar.didx -> C_Users_tacio_Downloads).
func archiveLabel(archiveName string) string {
	return strings.TrimSuffix(archiveName, ".pxar.didx")
}

// archiveBlobName is the blob an archive's per-folder metadata (NTFS ACLs,
// status sidecar) is stored in. The archive of the original layout keeps the
// bare names; in a snapshot of several archives each has its own, prefixed
// with its label.
func archiveBlobName(archiveName, blob string) string {
	if archiveName == legacyArchiveName || archiveName == "" {
		return blob
	}
	return archiveLabel(archiveName) + "." + blob
}

// snapshotSession is one PBS backup session writing one snapshot: one or more
// folder archives, which share the snapshot's catalog, then the manifest.
type snapshotSession struct {
	client                            *pbscommon.PBSClient
	newchunk, reusechunk, failedchunk *atomic.Uint64
	knownChunks                       *haxmap.Map[string, bool]
	catalog                           *pbscommon.SnapshotCatalog
	catalogChunk                      ChunkState
	excludeList                       []string
}

// openSnapshotSession starts the backup session on client and creates the
// snapshot's catalog index.
func openSnapshotSession(client *pbscommon.PBSClient, newchunk, reusechunk, failedchunk *atomic.Uint64, excludeList []string) (*snapshotSession, error) {
	client.Connect(false, "host")
	s := &snapshotSession{
		client:      client,
		newchunk:    newchunk,
		reusechunk:  reusechunk,
		failedchunk: failedchunk,
		knownChunks: haxmap.New[string, bool](),
		excludeList: excludeList,
	}
	s.catalogChunk.Init(newchunk, reusechunk, failedchunk, s.knownChunks, nil, &atomic.Uint64{}, nil, "")
	var err error
	if s.catalogChunk.wrid, err = client.CreateDynamicIndex("catalog.pcat1.didx"); err != nil {
		return nil, err
	}
	s.catalog = &pbscommon.SnapshotCatalog{WriteCB: func(b []byte) error {
		return s.catalogChunk.HandleData(b, client)
	}}
	return s, nil
}

// writeArchive archives the folder backupdir (the path read: a VSS shadow
// copy path, or originalPath itself) as archiveName, with its NTFS metadata
// and status blobs. It returns the archived byte count. Any error leaves the
// session unusable: the archive's index stays open, so PBS would refuse to
// finish the snapshot.
func (s *snapshotSession) writeArchive(archiveName, backupdir, originalPath string, vssUsed bool, progress func(float64, string), onStats func(*BackupProgressStats)) (uint64, error) {
	client := s.client

	// Start background scan to calculate total size (drives the progress %). This
	// is non-blocking — the backup streams in parallel — but bound it with the same
	// runaway guard so a whole-drive root can't leave a goroutine walking forever
	// (junctions are already skipped); a partial size just makes the % approximate.
	totalSize := &atomic.Uint64{}
	go func() {
		writeBackupLog(fmt.Sprintf("Starting background size calculation for: %s", backupdir))
		ctx, cancel := context.WithTimeout(context.Background(), analysisSizeBudget)
		defer cancel()
		size, err := calculateDirSizeCtx(ctx, backupdir)
		if err != nil {
			writeBackupLog(fmt.Sprintf("WARNING: Size calculation had errors: %v", err))
		}
		totalSize.Store(size)
		writeBackupLog(fmt.Sprintf("Total size calculated: %d MB", size/(1024*1024)))
	}()

	archive := &pbscommon.PXARArchive{}
	archive.ArchiveName = archiveName
	archive.Catalog = s.catalog
	archive.ExcludeList = s.excludeList
	archive.ExcludeRoot = originalPath // logical root for VSS-safe absolute-pattern matching

	// Inject backup metadata into the PXAR archive root
	hostname, _ := os.Hostname()
	metaJSON, err := GenerateBackupMeta(client.Manifest.BackupID, originalPath, hostname, vssUsed)
	if err != nil {
		writeBackupLog(fmt.Sprintf("WARNING: Failed to generate backup metadata: %v", err))
	} else {
		archive.VirtualFiles = map[string][]byte{
			BackupMetaFilename: metaJSON,
		}
	}

	// NTFS metadata collector: captures owner/ACLs/attrs on Windows, or every
	// extended attribute (including POSIX ACLs, which the kernel itself
	// stores as xattrs) on Linux, for every entry during the walk; no-op on
	// other platforms. The collected data is serialized after the walk and
	// uploaded as a blob in the same backup session (see below, after Eof).
	ntfsCollector := NewNTFSMetaCollector(backupdir, hostname)
	archive.MetaCollector = ntfsCollector

	previousDidx, err := client.DownloadPreviousToBytes(archive.ArchiveName)
	if err != nil {
		// This is normal for first backup - no previous backup exists
		writeBackupLog(fmt.Sprintf("No previous %s found (first backup?): %v", archive.ArchiveName, err))
		previousDidx = []byte{}
	} else {
		writeBackupLog(fmt.Sprintf("Downloaded previous DIDX %s: %d bytes", archive.ArchiveName, len(previousDidx)))
	}

	for _, shahash := range pbscommon.ParsePreviousDIDXChunkDigests(previousDidx) {
		s.knownChunks.Set(shahash, true)
	}

	writeBackupLog(fmt.Sprintf("Known chunks: %d", s.knownChunks.Len()))

	pxarChunk := ChunkState{}
	pxarChunk.Init(s.newchunk, s.reusechunk, s.failedchunk, s.knownChunks, progress, totalSize, onStats, backupdir)

	pxarChunk.wrid, err = client.CreateDynamicIndex(archive.ArchiveName)
	if err != nil {
		return 0, err
	}

	archive.WriteCB = func(b []byte) error {
		return pxarChunk.HandleData(b, client)
	}

	if _, err = archive.WriteDir(backupdir, "", true); err != nil {
		return 0, fmt.Errorf("failed to write directory archive: %w", err)
	}

	// Map VSS shadow-copy paths back to the original logical root so the status
	// lists are meaningful to the user (no-op for non-VSS backups, where
	// backupdir == originalPath). Reused for both the aggregate status and sidecar.
	logicalSkipped := toLogicalPaths(archive.SkippedFiles, backupdir, originalPath)
	logicalExcluded := toLogicalPaths(archive.ExcludedFiles, backupdir, originalPath)
	logicalReadErrors := toLogicalPaths(archive.ReadErrors, backupdir, originalPath)

	// Collect skipped files from archive
	if len(logicalSkipped) > 0 {
		writeBackupLog(fmt.Sprintf("Backup completed with %d skipped files/directories", len(logicalSkipped)))
		client.SkippedFiles = append(client.SkippedFiles, logicalSkipped...)
	}

	// Genuine read errors / content instability drive the outcome (v2-H-02).
	if len(logicalReadErrors) > 0 {
		client.ReadErrors = append(client.ReadErrors, logicalReadErrors...)
	}

	// Collect files excluded by user policy (H-04), kept distinct from errors.
	if len(logicalExcluded) > 0 {
		writeBackupLog(fmt.Sprintf("%d files/directories excluded by user policy", len(logicalExcluded)))
		client.ExcludedFiles = append(client.ExcludedFiles, logicalExcluded...)
	}

	// Guard: if WriteDir produced 0 data, the backup dir was effectively empty or inaccessible
	if pxarChunk.pos == 0 && len(pxarChunk.currentChunk) == 0 {
		return 0, fmt.Errorf("backup produced 0 bytes for %s — directory may be empty, inaccessible, or all files were excluded", backupdir)
	}

	if err = pxarChunk.EOF(client); err != nil {
		return 0, err
	}

	// Serialize NTFS metadata collected during the walk and upload it as a
	// blob in the same backup session. The blob is listed in the PBS manifest
	// alongside the archive and catalog.pcat1.didx so restore tools can
	// fetch it without extracting the whole archive. Best-effort: a failure
	// here does NOT fail the backup — the file data is already safe.
	if aclBytes, aclErr := ntfsCollector.Finalize(); aclErr != nil {
		writeBackupLog(fmt.Sprintf("WARNING: failed to serialize NTFS metadata: %v", aclErr))
	} else if len(aclBytes) > 0 {
		entries, uniqueSDDLs, metaErrs := ntfsCollector.Stats()
		writeBackupLog(fmt.Sprintf("NTFS metadata: %d entries, %d unique SDDLs, %d errors, %d bytes gzipped",
			entries, uniqueSDDLs, metaErrs, len(aclBytes)))
		streams, byName, eas, dropped := ntfsCollector.StreamStats()
		if streams+byName+eas > 0 {
			writeBackupLog(fmt.Sprintf("Alternate data streams: %d recorded, %d by name only (too large or unreadable); extended attributes: %d",
				streams, byName, eas))
		}
		if dropped > 0 {
			writeBackupLog(fmt.Sprintf("WARNING: NTFS metadata too large for one PBS blob: the content of %d alternate data streams was left out (permissions and attributes kept)", dropped))
		}
		if upErr := client.UploadBlob(archiveBlobName(archiveName, BackupAclsFilename), aclBytes); upErr != nil {
			writeBackupLog(fmt.Sprintf("WARNING: failed to upload NTFS metadata blob: %v", upErr))
		}
	}

	// Persist a per-snapshot status sidecar (files excluded by policy + files
	// skipped on read errors) as a manifest blob, so the GUI can list them without
	// restoring the archive. Best-effort: a failure here does NOT fail the backup.
	sidecar := &BackupSidecar{
		FormatVersion:    1,
		BackupID:         client.Manifest.BackupID,
		Directory:        originalPath,
		GeneratedAt:      time.Now().Unix(),
		ExcludedByPolicy: excludedToIssues(logicalExcluded),
		SkippedReadError: skippedToIssues(logicalReadErrors),
	}
	if sidecarBytes, sErr := json.Marshal(sidecar); sErr != nil {
		writeBackupLog(fmt.Sprintf("WARNING: failed to serialize status sidecar: %v", sErr))
	} else if upErr := client.UploadBlob(archiveBlobName(archiveName, BackupStatusFilename), sidecarBytes); upErr != nil {
		writeBackupLog(fmt.Sprintf("WARNING: failed to upload status sidecar: %v", upErr))
	}

	// pxarChunk.pos is the true archived byte count for this directory (more
	// accurate than the background size estimate, which only drives the %).
	return pxarChunk.pos, nil
}

// close ends the catalog listing every archive written and uploads the
// manifest. The caller then finishes the session (client.Finish).
func (s *snapshotSession) close() error {
	if err := s.catalog.Close(); err != nil {
		return err
	}
	if err := s.catalogChunk.EOF(s.client); err != nil {
		return err
	}

	// Upload manifest with retry
	retryConfig := retry.DefaultConfig()
	retryConfig.MaxAttempts = 5
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := retry.DoWithJitter(ctx, retryConfig, retry.DefaultRetryable, func() error {
		return s.client.UploadManifest()
	})
	if err != nil {
		return fmt.Errorf("failed to upload manifest after retries: %w", err)
	}
	return nil
}
