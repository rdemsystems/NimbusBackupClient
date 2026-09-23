package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// legacyDataDirName is the ProgramData folder used by Nimbus Backup releases up
// to 0.3.0, before the data directory became the brand-neutral
// "ProxmoxBackupClient". Installs upgraded from those releases still have their
// config, scheduled jobs and history there.
const legacyDataDirName = "NimbusBackup"

// legacyDataFiles are copied from the legacy folder. Logs and the restore cache
// are deliberately left behind: they are disposable and can be large.
var legacyDataFiles = []string{
	"config.json",
	"scheduled_jobs.json",
	"job_history.json",
	"api-token",
}

// legacyMigrationMarker is created in the new folder once every legacy file has
// been copied. Completion is tracked by this marker rather than by config.json
// existing: if a migration is interrupted and the user then saves a config, the
// remaining files are still copied on the next start (without overwriting the
// new config).
const legacyMigrationMarker = ".migrated-from-NimbusBackup"

// legacyCopiedMarker is written in the LEGACY folder once the copy is done, so
// an admin browsing ProgramData\NimbusBackup sees that it has been copied and
// is no longer used. It also counts as "already copied": the migration never
// runs again, even if the new folder is deleted later (e.g. to reset the app).
const legacyCopiedMarker = "COPIED-TO-ProxmoxBackupClient.txt"

var legacyMigrationOnce sync.Once

// migrateLegacyDataDir copies the pre-0.3 data files from the legacy folder next
// to newDir, until the migration marker exists. It runs at most once per
// process, never overwrites anything and never deletes the legacy folder (so
// downgrading keeps working). The GUI and the service may run it concurrently:
// each file is created with copyFileNoClobber, so whichever process lands a file
// first wins and a late copy can never replace a file already migrated (or
// edited since). Any error other than "legacy file missing" leaves the marker
// unwritten, so the migration is retried on the next start.
func migrateLegacyDataDir(newDir string) {
	legacyMigrationOnce.Do(func() {
		if filepath.Base(newDir) == legacyDataDirName {
			return
		}
		legacyDir := filepath.Join(filepath.Dir(newDir), legacyDataDirName)
		marker := filepath.Join(newDir, legacyMigrationMarker)
		if _, err := os.Stat(marker); err == nil { // #nosec G703 -- path built from the trusted ProgramData env var + fixed names
			return // already migrated
		}
		if _, err := os.Stat(filepath.Join(legacyDir, legacyCopiedMarker)); err == nil { // #nosec G703 -- path built from the trusted ProgramData env var + fixed names
			return // already copied (legacy folder marked)
		}
		if _, err := os.Stat(filepath.Join(legacyDir, "config.json")); err != nil { // #nosec G703 -- path built from the trusted ProgramData env var + fixed names
			return // nothing to migrate (fresh install, or not a Nimbus upgrade)
		}
		// #nosec G703 -- newDir is derived from the trusted ProgramData environment variable
		if err := os.MkdirAll(newDir, 0755); err != nil {
			writeDebugLog(fmt.Sprintf("Legacy data migration: cannot create %s: %v", newDir, err))
			return
		}
		for _, name := range legacyDataFiles {
			src := filepath.Join(legacyDir, name)
			dst := filepath.Join(newDir, name)
			data, err := os.ReadFile(src) // #nosec G304 G703 -- fixed file names under the trusted ProgramData folder
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				writeDebugLog(fmt.Sprintf("Legacy data migration: cannot read %s, will retry next start: %v", src, err))
				return
			}
			if err := copyFileNoClobber(dst, data); err != nil {
				writeDebugLog(fmt.Sprintf("Legacy data migration: cannot write %s, will retry next start: %v", dst, err))
				return
			}
		}
		if err := copyFileNoClobber(marker, []byte(legacyDir+"\n")); err != nil {
			writeDebugLog(fmt.Sprintf("Legacy data migration: cannot write marker %s: %v", marker, err))
			return
		}
		// Best effort: mark the legacy folder as copied. The new-folder marker
		// above is authoritative, so a failure here only costs the hint.
		note := fmt.Sprintf("Copied to %s on %s by version %s.\r\n"+
			"This folder is no longer used; it is kept so a downgrade still finds its data.\r\n",
			newDir, time.Now().Format(time.RFC3339), appVersion)
		if err := copyFileNoClobber(filepath.Join(legacyDir, legacyCopiedMarker), []byte(note)); err != nil {
			writeDebugLog(fmt.Sprintf("Legacy data migration: cannot mark %s as copied: %v", legacyDir, err))
		}
		writeDebugLog(fmt.Sprintf("Legacy data migration: copied data from %s to %s (legacy folder kept)", legacyDir, newDir))
	})
}

// copyFileNoClobber writes data to dst only if dst does not exist yet. The
// content is written and synced to a temp file first, then hard-linked into
// place: the link either creates dst whole or fails because dst exists (which
// is not an error here), so a reader never sees a half-written file and an
// existing file is never replaced — unlike a rename, which overwrites.
func copyFileNoClobber(dst string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".tmp-migrate-"+filepath.Base(dst)+"-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // #nosec G703 -- path built from the trusted ProgramData env var + fixed names
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Link(tmpName, dst); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}
