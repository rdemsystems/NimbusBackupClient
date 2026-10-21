package main

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"pbscommon"
	"security"
)

// Retry policy for session-lost failures: PBS keeps the BackupGroup lock until
// it detects the dead TCP connection server-side (observed ~16 min in prod).
// Any retry sooner than that hits 400 "while creating locked backup group".
// We wait sessionLostRetryWait between attempts so the lock has time to expire.
const (
	maxDirAttempts       = 2
	sessionLostRetryWait = 25 * time.Minute
)

// newDirBackupClient is the PBS client of a directory backup run.
func newDirBackupClient(opts BackupOptions) *pbscommon.PBSClient {
	// Debug: log connection parameters with sanitized credentials
	writeBackupLog(fmt.Sprintf("PBS Connection: URL=%s, AuthID=%s, Secret=%s, Datastore=%s, BackupID=%s",
		security.SanitizeURL(opts.BaseURL),
		opts.AuthID,
		security.SanitizeSecret(opts.Secret),
		opts.Datastore,
		opts.BackupID))

	// Parse compression level (default to fastest if empty or invalid)
	compressionLevel := pbscommon.ParseCompressionLevel(opts.Compression)
	writeBackupLog(fmt.Sprintf("[DEBUG] Compression level: %s", compressionLevel))

	return &pbscommon.PBSClient{
		BaseURL:          opts.BaseURL,
		CertFingerPrint:  opts.CertFingerprint,
		AuthID:           opts.AuthID,
		Secret:           opts.Secret,
		Ticket:           opts.Ticket,
		CSRFToken:        opts.CSRFToken,
		Datastore:        opts.Datastore,
		Namespace:        opts.Namespace,
		Insecure:         opts.CertFingerprint != "",
		CompressionLevel: compressionLevel,
		Crypt:            opts.Crypt,
		Manifest: pbscommon.BackupManifest{
			BackupID: opts.BackupID,
		},
	}
}

// waitForGroupLock waits sessionLostRetryWait for PBS to release the backup
// group lock of a lost session, showing the countdown. It returns false when
// the user cancelled meanwhile (already reported through OnComplete).
func waitForGroupLock(opts BackupOptions, progress func(float64, string)) bool {
	writeBackupLog(fmt.Sprintf("Waiting %s for PBS to release backup group lock before retry...",
		sessionLostRetryWait))

	waitUntil := time.Now().Add(sessionLostRetryWait)
	for {
		if opts.Ctx != nil && opts.Ctx.Err() != nil {
			writeBackupLog("Cancellation requested during session-lost wait — aborting")
			if opts.OnComplete != nil {
				opts.OnComplete(false, "Backup cancelled by user")
			}
			return false
		}
		remaining := time.Until(waitUntil)
		if remaining <= 0 {
			return true
		}
		progress(0, fmt.Sprintf("PBS session lost, waiting %s before retrying (PBS lock being released)...",
			remaining.Round(time.Second)))
		sleepFor := 30 * time.Second
		if remaining < sleepFor {
			sleepFor = remaining
		}
		time.Sleep(sleepFor)
	}
}

// dirRun accumulates the result of a directory backup run, folder by folder,
// whatever the layout (a snapshot per folder, or one for all of them).
type dirRun struct {
	newchunk, reusechunk, failedchunk, totalSize atomic.Uint64

	dirErrors      []string
	dirResults     []DirResult
	successfulDirs int
	// Run-level accumulators: client.{ReadErrors,SkippedFiles,ExcludedFiles} are
	// reset by each Connect(), so they are captured after each session before
	// the next one wipes them — otherwise a read error in an earlier folder of a
	// multi-folder run / split bin is lost and the run can falsely report
	// verified_success (v2-H-02 / M-01).
	allReadErrors, allSkipped, allExcluded []string
}

func (r *dirRun) collect(client *pbscommon.PBSClient) {
	r.allReadErrors = append(r.allReadErrors, client.ReadErrors...)
	r.allSkipped = append(r.allSkipped, client.SkippedFiles...)
	r.allExcluded = append(r.allExcluded, client.ExcludedFiles...)
}

func (r *dirRun) fail(dir, errMsg string, err error) {
	writeBackupLog(errMsg)
	r.dirErrors = append(r.dirErrors, errMsg)
	r.dirResults = append(r.dirResults, DirResult{Path: dir, OK: false, Error: err.Error()})
}

func (r *dirRun) succeed(dir string) {
	r.dirResults = append(r.dirResults, DirResult{Path: dir, OK: true})
	r.successfulDirs++
}

// report logs the run's result, delivers it through OnComplete/OnResult and
// returns the run's error: nil only for a success.
func (r *dirRun) report(opts BackupOptions, backupTime int64, startTime time.Time, progress func(float64, string)) error {
	// If NO directory was backed up successfully, fail the whole backup
	if r.successfulDirs == 0 {
		errMsg := fmt.Sprintf("All %d directories failed:\n%s", len(opts.BackupObjects), strings.Join(r.dirErrors, "\n"))
		writeBackupLog(errMsg)
		status := &BackupStatus{
			Outcome:          OutcomeFailed,
			BackupID:         opts.BackupID,
			BackupTime:       backupTime,
			DurationSec:      time.Since(startTime).Seconds(),
			TotalBytes:       r.totalSize.Load(),
			NewChunks:        r.newchunk.Load(),
			ReusedChunks:     r.reusechunk.Load(),
			FailedChunks:     r.failedchunk.Load(),
			Directories:      r.dirResults,
			ExcludedByPolicy: excludedToIssues(r.allExcluded),
			SkippedReadError: skippedToIssues(r.allReadErrors),
			Message:          errMsg,
		}
		if opts.OnComplete != nil {
			opts.OnComplete(false, errMsg)
		}
		if opts.OnResult != nil {
			opts.OnResult(status)
		}
		return fmt.Errorf("%s", errMsg)
	}

	// Calculate backup duration and size
	duration := time.Since(startTime)
	totalSizeMB := float64(r.totalSize.Load()) / (1024 * 1024)

	// Build completion message with duration, size, and chunk stats
	failed := r.failedchunk.Load()
	partial := len(r.dirErrors) > 0
	var completionMsg, progressMsg string
	switch {
	case partial:
		completionMsg = fmt.Sprintf("⚠️  Backup partiel en %s: %d/%d dossiers OK, %.1f MB (%d new, %d reused chunks)\nErreurs:\n%s",
			formatDuration(duration), r.successfulDirs, len(opts.BackupObjects), totalSizeMB, r.newchunk.Load(), r.reusechunk.Load(), strings.Join(r.dirErrors, "\n"))
		progressMsg = fmt.Sprintf("Backup partiel : %d/%d dossiers OK", r.successfulDirs, len(opts.BackupObjects))
	case failed > 0:
		completionMsg = fmt.Sprintf("⚠️  Backup completed with errors in %s: %.1f MB backed up (%d new, %d reused, %d FAILED chunks)",
			formatDuration(duration), totalSizeMB, r.newchunk.Load(), r.reusechunk.Load(), failed)
		progressMsg = fmt.Sprintf("Backup completed with %d failed chunks", failed)
	default:
		completionMsg = fmt.Sprintf("Backup completed in %s: %.1f MB backed up (%d new, %d reused chunks)",
			formatDuration(duration), totalSizeMB, r.newchunk.Load(), r.reusechunk.Load())
		progressMsg = "Backup completed"
	}

	progress(1.0, progressMsg)

	if len(r.allSkipped) > 0 {
		completionMsg += fmt.Sprintf("\n⚠️  %d files/folders skipped (access denied or junction points)", len(r.allSkipped))
		writeBackupLog(fmt.Sprintf("=== SKIPPED FILES/DIRECTORIES (%d) ===", len(r.allSkipped)))

		// Log first 50 skipped files in detail
		maxLog := 50
		if len(r.allSkipped) < maxLog {
			maxLog = len(r.allSkipped)
		}
		for i := 0; i < maxLog; i++ {
			writeBackupLog(fmt.Sprintf("  [%d] %s", i+1, r.allSkipped[i]))
		}
		if len(r.allSkipped) > 50 {
			writeBackupLog(fmt.Sprintf("  ... and %d more (see full list in GUI)", len(r.allSkipped)-50))
		}
		writeBackupLog("=== END SKIPPED FILES ===")
	}

	writeBackupLog(completionMsg)

	// Determine the authoritative 4-level outcome (v2-H-02 / F-01):
	//   failed  — a chunk upload failed (would corrupt the index) or no dir committed
	//   partial — some directories failed, OR files were unreadable / changed during
	//             read (genuine read errors, not expected system auto-excludes)
	//   success_with_policy_exclusions — complete except files the user excluded
	//   verified_success — fully complete
	hasReadIssues := len(r.allReadErrors) > 0
	var outcome BackupOutcome
	switch {
	case failed > 0:
		outcome = OutcomeFailed
	case partial:
		outcome = OutcomePartial
	case hasReadIssues:
		outcome = OutcomePartial
	case len(r.allExcluded) > 0:
		outcome = OutcomeSuccessWithExclusions
	default:
		outcome = OutcomeVerifiedSuccess
	}

	status := &BackupStatus{
		Outcome:          outcome,
		BackupID:         opts.BackupID,
		BackupTime:       backupTime,
		DurationSec:      duration.Seconds(),
		TotalBytes:       r.totalSize.Load(),
		NewChunks:        r.newchunk.Load(),
		ReusedChunks:     r.reusechunk.Load(),
		FailedChunks:     failed,
		Directories:      r.dirResults,
		ExcludedByPolicy: excludedToIssues(r.allExcluded),
		SkippedReadError: skippedToIssues(r.allReadErrors),
		Message:          completionMsg,
	}

	// One machine-greppable result line for support (pairs with the start-of-run
	// target log): outcome label + the structured counters behind completionMsg.
	writeBackupLog(fmt.Sprintf("[RESULT] outcome=%s dirs_ok=%d/%d new=%d reused=%d failed=%d read_errors=%d excluded=%d bytes=%d duration=%s",
		status.Outcome, r.successfulDirs, len(opts.BackupObjects), status.NewChunks, status.ReusedChunks,
		status.FailedChunks, len(status.SkippedReadError), len(status.ExcludedByPolicy), status.TotalBytes, duration))

	// Additive (choice A): OnComplete keeps its (success, message) contract for
	// existing consumers; OnResult carries the full structured status for the
	// sidecar (Group 1) and rich history.
	if opts.OnComplete != nil {
		opts.OnComplete(status.Success(), completionMsg)
	}
	if opts.OnResult != nil {
		opts.OnResult(status)
	}

	// Contract (Group 0 / audit H-03): a partial or failed backup must NOT return a
	// nil error, otherwise the API fallback and the scheduler record it as success.
	if !status.Success() {
		return fmt.Errorf("%s", completionMsg)
	}
	return nil
}
