package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"machinebackuplib"
	"retry"
	"snapshot"
)

// runOneSnapshotBackup backs up every folder of opts into ONE snapshot of
// one backup group (opts.BackupID, or the hostname): one archive per folder,
// named by ArchiveNames, sharing the snapshot's catalog. This is the layout
// of proxmox-backup-client with several archives, so prune keeps or removes
// a run as a whole, and PVE or the official client can restore each folder.
//
// The folders are written one after the other in a single session. A folder
// whose root cannot be read is left out and the run is partial; any other
// error ends the session, which is retried once after a lost session, like
// a folder of the other layout.
func runOneSnapshotBackup(opts BackupOptions) error {
	startTime := time.Now()

	if !opts.lockHeld {
		backupLock := getBackupLock(opts.BaseURL, opts.Datastore)
		writeBackupLog(fmt.Sprintf("[Backup Lock] Waiting for lock on %s/%s (prevents concurrent backups)", opts.BaseURL, opts.Datastore))
		if opts.OnProgress != nil {
			opts.OnProgress(0, "Waiting for previous backup to complete...")
		}
		backupLock.Lock()
		writeBackupLog(fmt.Sprintf("[Backup Lock] ✓ Lock acquired for %s/%s - starting backup", opts.BaseURL, opts.Datastore))
		defer func() {
			backupLock.Unlock()
			writeBackupLog(fmt.Sprintf("[Backup Lock] ✓ Lock released for %s/%s", opts.BaseURL, opts.Datastore))
		}()
	}

	if opts.BackupID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unnamed-backup"
		}
		opts.BackupID = hostname
	}
	if opts.BackupType == "" {
		opts.BackupType = "host"
	}

	progress := func(pct float64, msg string) {
		writeBackupLog(fmt.Sprintf("Backup progress: %.1f%% - %s", pct*100, msg))
		if opts.OnProgress != nil {
			opts.OnProgress(pct, msg)
		}
	}

	dirs := opts.BackupObjects
	names := ArchiveNames(dirs)
	run := &dirRun{}
	writeBackupLog(fmt.Sprintf("[Snapshot] Backing up %d folder(s) into one snapshot of %s", len(dirs), opts.BackupID))

	// A root that cannot be read would fail after its archive index is
	// created, which leaves the whole snapshot unfinishable: check first.
	var ready []int
	for i, dir := range dirs {
		if _, err := os.ReadDir(dir); err != nil {
			run.fail(dir, fmt.Sprintf("Backup failed for %s: cannot read the folder: %v", dir, err), err)
			continue
		}
		writeBackupLog(fmt.Sprintf("[Snapshot] %s -> %s", dir, names[i]))
		ready = append(ready, i)
	}
	if len(ready) == 0 {
		return run.report(opts, 0, startTime, progress)
	}

	progress(0.05, "Connecting to PBS...")
	client := newDirBackupClient(opts)
	// Also on a panic: an open session holds the backup group lock. Close
	// can be called again.
	defer client.Close()

	cancelled := func() bool { return opts.Ctx != nil && opts.Ctx.Err() != nil }

	// session writes the ready folders, reading each from paths[dir] (its VSS
	// shadow copy path) when paths is set, else from a VSS snapshot of its
	// own (useVSS) or the live folder. committed tells that PBS has the
	// snapshot, even if releasing the VSS snapshot fails afterwards.
	var archived []uint64
	committed := false
	session := func(paths map[string]string, useVSS bool) error {
		// The counters of an abandoned attempt do not describe this snapshot.
		archived = archived[:0]
		run.newchunk.Store(0)
		run.reusechunk.Store(0)
		run.failedchunk.Store(0)
		s, err := openSnapshotSession(client, &run.newchunk, &run.reusechunk, &run.failedchunk, opts.ExcludeList)
		if err != nil {
			return err
		}
		var doneBytes uint64
		for n, i := range ready {
			if cancelled() {
				return machinebackuplib.ErrCancelled
			}
			dir := dirs[i]
			writeBackupLog(fmt.Sprintf("Starting backup of directory %d/%d: %s", n+1, len(ready), dir))
			// Each folder fills its share of 5-95% of the progress bar; the
			// statistics count the folders done before it.
			base, share := 0.05+0.9*float64(n)/float64(len(ready)), 0.9/float64(len(ready))
			folderProgress := func(pct float64, msg string) { progress(base+share*pct, msg) }
			var folderStats func(*BackupProgressStats)
			if opts.OnStats != nil {
				before := doneBytes
				folderStats = func(st *BackupProgressStats) {
					total := *st
					total.Percent = base + share*st.Percent
					total.BytesDone += before
					if total.BytesTotal > 0 {
						total.BytesTotal += before
					}
					opts.OnStats(&total)
				}
			}

			var size uint64
			switch {
			case paths != nil:
				path, ok := paths[dir]
				if !ok {
					return fmt.Errorf("no VSS snapshot was taken for %s", dir)
				}
				size, err = s.writeArchive(names[i], path, dir, true, folderProgress, folderStats)
			case useVSS:
				err = snapshot.CreateVSSSnapshot([]string{dir}, true, func(snaps map[string]snapshot.SnapShot) error {
					path := ""
					for _, snap := range snaps {
						path = snap.FullPath
						break
					}
					if path == "" {
						return fmt.Errorf("no VSS snapshot was taken for %s", dir)
					}
					var e error
					size, e = s.writeArchive(names[i], path, dir, true, folderProgress, folderStats)
					return e
				})
			default:
				size, err = s.writeArchive(names[i], dir, dir, false, folderProgress, folderStats)
			}
			if err != nil {
				return fmt.Errorf("%s: %w", dir, err)
			}
			archived = append(archived, size)
			doneBytes += size
			progress(base+share, fmt.Sprintf("Folder %d/%d archived: %s", n+1, len(ready), dir))
		}
		// Last chance to stop: past this point the snapshot is committed.
		if cancelled() {
			return machinebackuplib.ErrCancelled
		}
		if err := s.close(); err != nil {
			return err
		}
		finishCfg := retry.DefaultConfig()
		finishCfg.MaxAttempts = 5
		finishCtx, finishCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer finishCancel()
		if err := retry.DoWithJitter(finishCtx, finishCfg, retry.DefaultRetryable, func() error {
			return client.Finish()
		}); err != nil {
			return err
		}
		committed = true
		return nil
	}

	// Under VSS, one snapshot set for every folder, so they are frozen
	// together; when the set cannot be taken (a volume refuses VSS), one
	// snapshot per folder in turn.
	attempt := func() error {
		if !opts.UseVSS {
			return session(nil, false)
		}
		readyDirs := make([]string, 0, len(ready))
		for _, i := range ready {
			readyDirs = append(readyDirs, dirs[i])
		}
		called := false
		err := snapshot.CreateVSSSnapshot(readyDirs, true, func(snaps map[string]snapshot.SnapShot) error {
			called = true
			paths := make(map[string]string, len(readyDirs))
			for _, dir := range readyDirs {
				if sn, ok := snaps[dir]; ok {
					paths[dir] = sn.FullPath
				} else if abs, aerr := filepath.Abs(dir); aerr == nil {
					if sn, ok := snaps[abs]; ok {
						paths[dir] = sn.FullPath
					}
				}
			}
			return session(paths, false)
		})
		if err != nil && !called {
			writeBackupLog(fmt.Sprintf("[VSS] One snapshot for every folder failed (%v): falling back to one snapshot per folder", err))
			return session(nil, true)
		}
		return err
	}

	var err error
	for try := 1; try <= maxDirAttempts; try++ {
		if err = attempt(); err == nil || committed {
			break
		}
		// Release the backup group lock the unfinished session holds. [B-4]
		client.Close()
		if errors.Is(err, machinebackuplib.ErrCancelled) {
			writeBackupLog("Cancellation requested — the snapshot is not committed")
			if opts.OnComplete != nil {
				opts.OnComplete(false, "Backup cancelled by user")
			}
			return machinebackuplib.ErrCancelled
		}
		if !isFatalSessionError(err) || try == maxDirAttempts {
			break
		}
		writeBackupLog(fmt.Sprintf("Snapshot of %s: session lost on attempt %d/%d: %v", opts.BackupID, try, maxDirAttempts, err))
		if !waitForGroupLock(opts, progress) {
			return machinebackuplib.ErrCancelled
		}
		writeBackupLog("Wait complete, retrying the snapshot with a fresh connection")
	}
	run.collect(client)

	switch {
	case committed:
		for n, i := range ready {
			run.totalSize.Add(archived[n])
			run.succeed(dirs[i])
		}
		writeBackupLog(fmt.Sprintf("Snapshot of %d folder(s) finalized", len(ready)))
		if err != nil {
			// PBS has the snapshot, but the VSS snapshot could not be
			// released: never a plain success.
			msg := fmt.Sprintf("VSS snapshot of the folders failed: %v", err)
			writeBackupLog(msg)
			run.dirErrors = append(run.dirErrors, msg)
		}
	default:
		// Nothing of the snapshot was committed: every folder failed.
		for _, i := range ready {
			run.fail(dirs[i], fmt.Sprintf("Backup failed for %s: %v", dirs[i], err), err)
		}
	}
	return run.report(opts, client.Manifest.BackupTime, startTime, progress)
}
