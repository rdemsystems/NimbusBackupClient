//go:build unix

package pbscommon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The folder a restore puts the archive's root into gets the root's recorded
// mode and time when the restore creates it, and is left alone when it was
// already there.
func TestTopFolderMetadata(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(src, 0o750); err != nil {
		t.Fatal(err)
	}
	recorded := time.Date(2019, 3, 4, 5, 6, 7, 123456789, time.UTC)
	if err := os.Chtimes(src, recorded, recorded); err != nil {
		t.Fatal(err)
	}
	pxar, _, _ := writePosixArchive(t, src)
	root := os.Geteuid() == 0

	restoreInto := func(top string, includes []string) {
		t.Helper()
		pr := NewPXARReader(pxar)
		pr.SetRestoreOwnership(root)
		rw := func(p string) string {
			if p == "" {
				return top
			}
			return filepath.Join(top, filepath.FromSlash(p))
		}
		res, err := pr.ExtractWithRewriter(rw, includes, true)
		if err != nil {
			t.Fatal(err)
		}
		ApplyDirTimes(res)
		FinishDirs(res, root)
	}
	stat := func(p string) os.FileInfo {
		t.Helper()
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		return fi
	}

	// A new top folder, whole archive: mode and time as recorded.
	top := filepath.Join(t.TempDir(), "restored")
	restoreInto(top, nil)
	fi := stat(top)
	if fi.Mode().Perm() != 0o750 {
		t.Errorf("new top folder: mode %04o, want 0750", fi.Mode().Perm())
	}
	if !fi.ModTime().Equal(recorded) {
		t.Errorf("new top folder: time %v, want %v", fi.ModTime(), recorded)
	}

	// A new top folder, one file selected: the recorded mode, but its time is
	// its own, as for any folder only on the way to the selection.
	top2 := filepath.Join(t.TempDir(), "partial")
	restoreInto(top2, []string{"sub/b.txt"})
	if fi := stat(top2); fi.Mode().Perm() != 0o750 || fi.ModTime().Equal(recorded) {
		t.Errorf("new top folder of a selection: mode %04o time %v, want 0750 and not the recorded time", fi.Mode().Perm(), fi.ModTime())
	}

	// An existing top folder: left as it was.
	top3 := filepath.Join(t.TempDir(), "existing")
	if err := os.Mkdir(top3, 0o711); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(top3, 0o711); err != nil {
		t.Fatal(err)
	}
	before := time.Date(2022, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(top3, before, before); err != nil {
		t.Fatal(err)
	}
	restoreInto(top3, nil)
	fi = stat(top3)
	if fi.Mode().Perm() != 0o711 {
		t.Errorf("existing top folder: mode %04o, want it left at 0711", fi.Mode().Perm())
	}
	// Writing files into it moves its time; the restore must not set the
	// recorded one on a folder it did not create.
	if fi.ModTime().Equal(recorded) {
		t.Errorf("existing top folder got the archive's time")
	}
}
