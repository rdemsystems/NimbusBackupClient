package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestMigrateLegacyDataDir(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, legacyDataDirName)
	newDir := filepath.Join(root, "ProxmoxBackupClient")
	if err := os.MkdirAll(legacy, 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"config.json":         `{"baseurl":"https://pbs:8007"}`,
		"scheduled_jobs.json": `[{"id":"job1"}]`,
		"debug-gui.log":       "not migrated",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(legacy, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// A log file created in the new folder before the migration (the logger
	// starts first) must not block it.
	if err := os.MkdirAll(newDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "service-gui.log"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	legacyMigrationOnce = sync.Once{}
	migrateLegacyDataDir(newDir)

	for _, name := range []string{"config.json", "scheduled_jobs.json"} {
		got, err := os.ReadFile(filepath.Join(newDir, name))
		if err != nil {
			t.Fatalf("%s not migrated: %v", name, err)
		}
		if string(got) != files[name] {
			t.Errorf("%s = %q, want %q", name, got, files[name])
		}
	}
	if _, err := os.Stat(filepath.Join(newDir, "debug-gui.log")); !os.IsNotExist(err) {
		t.Errorf("log file should not be migrated")
	}
	if _, err := os.Stat(filepath.Join(legacy, "config.json")); err != nil {
		t.Errorf("legacy folder must be kept: %v", err)
	}

	if _, err := os.Stat(filepath.Join(newDir, legacyMigrationMarker)); err != nil {
		t.Errorf("migration marker not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacy, legacyCopiedMarker)); err != nil {
		t.Errorf("legacy folder not marked as copied: %v", err)
	}

	// Once the legacy folder is marked, deleting the new folder must not
	// trigger a second copy.
	if err := os.RemoveAll(newDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newDir, 0755); err != nil {
		t.Fatal(err)
	}
	legacyMigrationOnce = sync.Once{}
	migrateLegacyDataDir(newDir)
	if _, err := os.Stat(filepath.Join(newDir, "config.json")); !os.IsNotExist(err) {
		t.Errorf("migration ran again although the legacy folder is marked as copied")
	}
	// Restore the migrated state for the no-overwrite check below.
	if err := os.WriteFile(filepath.Join(newDir, legacyMigrationMarker), nil, 0600); err != nil {
		t.Fatal(err)
	}

	// A second migration must never overwrite the (possibly edited) new config.
	if err := os.WriteFile(filepath.Join(newDir, "config.json"), []byte("edited"), 0600); err != nil {
		t.Fatal(err)
	}
	legacyMigrationOnce = sync.Once{}
	migrateLegacyDataDir(newDir)
	if got, _ := os.ReadFile(filepath.Join(newDir, "config.json")); string(got) != "edited" {
		t.Errorf("existing config overwritten: %q", got)
	}
	legacyMigrationOnce = sync.Once{}
}

// An interrupted migration (no marker) whose new folder already has a config
// saved by the user must still copy the missing files, without touching the
// user's config.
func TestMigrateLegacyDataDirResumesWithoutClobbering(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, legacyDataDirName)
	newDir := filepath.Join(root, "ProxmoxBackupClient")
	for _, d := range []string{legacy, newDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(legacy, "config.json"), "legacy")
	write(filepath.Join(legacy, "scheduled_jobs.json"), "jobs")
	write(filepath.Join(newDir, "config.json"), "saved by user")

	legacyMigrationOnce = sync.Once{}
	migrateLegacyDataDir(newDir)
	legacyMigrationOnce = sync.Once{}

	if got, _ := os.ReadFile(filepath.Join(newDir, "config.json")); string(got) != "saved by user" {
		t.Errorf("user config overwritten: %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(newDir, "scheduled_jobs.json")); string(got) != "jobs" {
		t.Errorf("scheduled jobs not migrated: %q", got)
	}
}
