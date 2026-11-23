package pbscommon

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// A tool that PATH does not find is still found in the system folders, as
// modprobe in /usr/sbin for a user whose PATH has no sbin folder.
func TestSystemToolOutsidePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Linux and other Unix systems only")
	}
	dir := t.TempDir()
	tool := filepath.Join(dir, "salvanza-test-tool")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := systemToolDirs
	systemToolDirs = []string{filepath.Join(dir, "missing"), dir}
	defer func() { systemToolDirs = old }()
	t.Setenv("PATH", filepath.Join(dir, "empty"))

	if got := SystemTool("salvanza-test-tool"); got != tool {
		t.Errorf("SystemTool = %q, want %q", got, tool)
	}
	if got := SystemTool("no-such-tool-here"); got != "no-such-tool-here" {
		t.Errorf("a tool found nowhere = %q, want the bare name", got)
	}
	t.Setenv("PATH", dir)
	if got := SystemTool("salvanza-test-tool"); got != tool {
		t.Errorf("found on PATH = %q, want %q", got, tool)
	}
}
