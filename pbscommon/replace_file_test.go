package pbscommon

import (
	"os"
	"path/filepath"
	"testing"
)

// A restore over an earlier restore replaces read-only files: on Windows the
// plain rename is refused for a file with the read-only attribute.
func TestReplaceFileOverReadOnly(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "ro.txt")
	tmp := filepath.Join(dir, "ro.txt.proxmox-1.part")
	if err := os.WriteFile(dst, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dst, 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmp, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(tmp, dst); err != nil {
		t.Fatalf("replaceFile over a read-only file: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "new" {
		t.Fatalf("content = %q, %v; want new", got, err)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("temporary file still there: %v", err)
	}
}

// A failure that has nothing to do with the read-only attribute is returned
// as it is, and the target is left alone.
func TestReplaceFileMissingSource(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(dst, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dst, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dst, 0o644) })
	if err := replaceFile(filepath.Join(dir, "missing.part"), dst); err == nil {
		t.Fatal("want an error for a missing temporary file")
	}
	fi, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o200 != 0 {
		t.Fatalf("target lost its read-only setting: %v", fi.Mode())
	}
	if got, _ := os.ReadFile(dst); string(got) != "old" {
		t.Fatalf("target changed: %q", got)
	}
}
