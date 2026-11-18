package main

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"sync"
	"time"
)

// A log for a run that failed before it started (#142). A run's own log is
// opened by the backup itself (RunBackupInline), so a scheduled run that
// failed before it got there (its PBS server out of reach after a wake, the
// login refused, a disk or the encryption key not found) left no log of its
// own, and its "View log" in Reports and the Message Log opened the log of the
// run before it, which had completed. Such a run now gets a short log of its
// own saying what stopped it.

// runLogStarts is when each backup ID's run last opened its own log.
var runLogStarts struct {
	sync.Mutex
	at map[string]time.Time
}

// noteRunLogStarted records that a run of backupID opened its log now.
func noteRunLogStarted(backupID string) {
	runLogStarts.Lock()
	defer runLogStarts.Unlock()
	if runLogStarts.at == nil {
		runLogStarts.at = map[string]time.Time{}
	}
	runLogStarts.at[sanitizeForFilename(backupID)] = time.Now()
}

// runLogStartedSince reports whether a run of backupID opened its log at or
// after since.
func runLogStartedSince(backupID string, since time.Time) bool {
	runLogStarts.Lock()
	defer runLogStarts.Unlock()
	at, ok := runLogStarts.at[sanitizeForFilename(backupID)]
	return ok && !at.Before(since)
}

// runLogIDFor is the backup ID a run's log is named after: the set's, or this
// computer's name when the set has none, as the run itself uses.
func runLogIDFor(backupID string) string {
	if backupID != "" {
		return backupID
	}
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "backup"
}

// writeNotStartedRunLog writes the log of set's run (backup ID backupID) that
// failed with cause before it started.
func writeNotStartedRunLog(backupID, set, trigger string, cause error) {
	id := runLogIDFor(backupID)
	logger := StartBackupRunLog(id)
	if logger == nil {
		return
	}
	noteRunLogStarted(id)
	userName := "<unknown>"
	if cu, err := user.Current(); err == nil {
		userName = cu.Username
	}
	for _, line := range []string{
		fmt.Sprintf("==== starting version %s - user %s - %s/%s ====", appVersion, userName, runtime.GOOS, runtime.GOARCH),
		fmt.Sprintf("Backup Set %q, %s run", set, trigger),
		fmt.Sprintf("[ERROR] The run failed before the backup started: %v", cause),
	} {
		writeLogToLogger(logger, "BACKUP", line)
	}
	EndBackupRunLog(logger)
}
