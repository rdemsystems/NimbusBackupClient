//go:build unix

package pbscommon

import (
	"os"
	"path/filepath"
	"testing"
)

// Restoring one file from ~/.ssh into a home folder that is already there:
// that folder keeps its own mode and owner, for root too. (The ~/.ssh the
// restore creates on the way is made 0755: a selective restore walks only
// the selection, so it never reads the ancestors' metadata. Known gap,
// testing/restore-matrix/README.md.)
func TestFinishDirsSelectedFile(t *testing.T) {
	src := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(src, "home", ".ssh"), 0o700))
	must(os.WriteFile(filepath.Join(src, "home", ".ssh", "id_ed25519"), []byte("key"), 0o600))
	must(os.Chmod(filepath.Join(src, "home", ".ssh"), 0o700))
	must(os.Chmod(filepath.Join(src, "home"), 0o750))
	pxar, _, _ := writePosixArchive(t, src)

	dest := t.TempDir()
	must(os.Mkdir(filepath.Join(dest, "home"), 0o711))
	must(os.Chmod(filepath.Join(dest, "home"), 0o711))
	pr := NewPXARReader(pxar)
	root := os.Geteuid() == 0
	pr.SetRestoreOwnership(root)
	res, err := pr.ExtractFiltered(dest, []string{"home/.ssh/id_ed25519"}, false)
	if err != nil {
		t.Fatal(err)
	}
	FinishDirs(res, root)

	for _, c := range []struct {
		path string
		want os.FileMode
	}{
		// Already there: left alone.
		{"home", 0o711},
		{"home/.ssh/id_ed25519", 0o600},
	} {
		fi, err := os.Stat(filepath.Join(dest, filepath.FromSlash(c.path)))
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != c.want {
			t.Errorf("%s: mode %04o, want %04o", c.path, fi.Mode().Perm(), c.want)
		}
	}
}

// A user's restore removes the bits the archive removed and adds none: a
// 0777 folder (a Windows backup records every folder so) stays 0755, and a
// 0700 one is made 0700. Setuid, setgid and sticky are not applied.
func TestApplyDirMode(t *testing.T) {
	for _, c := range []struct {
		start    os.FileMode
		recorded uint32
		want     os.FileMode
	}{
		{0o755, 0o700, 0o700},
		{0o755, 0o777, 0o755},
		{0o755, 0o1777, 0o755},
		{0o755, 0o2750, 0o750},
		{0o755, 0o500, 0o500},
		{0o700, 0o755, 0o700},
	} {
		d := filepath.Join(t.TempDir(), "d")
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(d, c.start); err != nil {
			t.Fatal(err)
		}
		ApplyDirMode(d, uint32(IFDIR)|c.recorded)
		fi, err := os.Stat(d)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != c.want || fi.Mode()&(os.ModeSticky|os.ModeSetgid|os.ModeSetuid) != 0 {
			t.Errorf("start %04o, recorded %04o: got %v, want %04o", c.start, c.recorded, fi.Mode(), c.want)
		}
		_ = os.Chmod(d, 0o755)
	}
}
