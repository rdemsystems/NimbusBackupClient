//go:build linux

package pbscommon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Without systemd or elogind, sleep and hibernate write the kernel's own
// switch, and only a state the kernel offers.
func TestSuspendWithoutSystemd(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(dir, "state")
	if err := os.WriteFile(state, []byte("freeze mem disk\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldFile, oldBooted, oldDirs := powerStateFile, systemdBooted, systemToolDirs
	powerStateFile, systemdBooted = state, func() bool { return false }
	systemToolDirs = []string{filepath.Join(dir, "none")}
	t.Setenv("PATH", filepath.Join(dir, "empty")) // no loginctl
	t.Cleanup(func() { powerStateFile, systemdBooted, systemToolDirs = oldFile, oldBooted, oldDirs })

	// The kernel's switch is a file here: writing replaces what it offered.
	if err := HibernateComputer(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(state); string(b) != "disk" {
		t.Errorf("wrote %q, want disk", b)
	}
	if err := os.WriteFile(state, []byte("freeze\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SleepComputer(); err == nil || !strings.Contains(err.Error(), "does not offer") {
		t.Errorf("sleep with only freeze offered: %v, want does not offer", err)
	}
}
