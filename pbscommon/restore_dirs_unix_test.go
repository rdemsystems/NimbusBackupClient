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

// A user restores a tree with read-only folders (0555, as Go's module cache
// is), then restores it again, or rolls back: the folders the first restore
// made read-only must not stop the second one. Both extraction paths, a
// whole restore and one file selected below them; before each restore over
// the first one, the files are changed, so that it must write them again.
// The folders end as recorded each time.
func TestRestoreAgainIntoReadOnlyFolders(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes into any folder; this is about an ordinary user's restore")
	}
	src := t.TempDir()
	unlockOnCleanup(t, src)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	content := map[string]string{"ro/ro.txt": "ro", "ro/deep/d.txt": "deep", "nowrite/n.txt": "n"}
	must(os.MkdirAll(filepath.Join(src, "ro", "deep"), 0o755))
	must(os.MkdirAll(filepath.Join(src, "nowrite"), 0o755))
	must(os.WriteFile(filepath.Join(src, "ro", "ro.txt"), []byte("ro"), 0o444))
	must(os.WriteFile(filepath.Join(src, "ro", "deep", "d.txt"), []byte("deep"), 0o444))
	must(os.WriteFile(filepath.Join(src, "nowrite", "n.txt"), []byte("n"), 0o600))
	must(os.Chmod(filepath.Join(src, "ro", "deep"), 0o555))
	must(os.Chmod(filepath.Join(src, "ro"), 0o555))
	must(os.Chmod(filepath.Join(src, "nowrite"), 0o500))
	pxar, _, _ := writePosixArchive(t, src)

	dest := t.TempDir()
	unlockOnCleanup(t, dest)
	want := map[string]os.FileMode{"ro": 0o555, "ro/deep": 0o555, "nowrite": 0o500}
	// change rewrites files of an earlier restore, through their read-only
	// folders, as someone editing them would.
	change := func(paths ...string) {
		t.Helper()
		for _, p := range paths {
			f := filepath.Join(dest, filepath.FromSlash(p))
			d := filepath.Dir(f)
			fi, err := os.Stat(d)
			must(err)
			must(os.Chmod(d, 0o700))
			must(os.Chmod(f, 0o600))
			must(os.WriteFile(f, []byte("changed"), 0o600))
			must(os.Chmod(d, fi.Mode().Perm()))
		}
	}
	modes := func(step string) {
		t.Helper()
		for p, m := range want {
			fi, err := os.Stat(filepath.Join(dest, filepath.FromSlash(p)))
			if err != nil {
				t.Fatalf("%s: %v", step, err)
			}
			if fi.Mode().Perm() != m {
				t.Errorf("%s: %s is %04o, want %04o", step, p, fi.Mode().Perm(), m)
			}
		}
	}
	check := func(step string, res []PXARExtractedFile, err error, overwrite bool) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", step, err)
		}
		for _, f := range res {
			if f.Skipped && (overwrite || !f.Expected) {
				t.Errorf("%s: %s: %s", step, f.Path, f.SkipReason)
			}
		}
		FinishDirs(res, false)
		modes(step)
		for p, c := range content {
			b, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(p)))
			if err != nil || string(b) != c {
				t.Errorf("%s: %s = %q, %v", step, p, b, err)
			}
		}
	}
	all := []string{"ro/ro.txt", "ro/deep/d.txt", "nowrite/n.txt"}
	rewriter := func(p string) string { return filepath.Join(dest, filepath.FromSlash(p)) }

	res, err := NewPXARReader(pxar).ExtractAll(dest)
	check("first restore", res, err, false)
	change(all...)
	res, err = NewPXARReader(pxar).ExtractFiltered(dest, nil, true)
	check("restore again", res, err, true)
	change(all...)
	res, err = NewPXARReader(pxar).ExtractWithRewriterParallel(rewriter, nil, true, 4)
	check("restore again, parallel", res, err, true)
	change("ro/deep/d.txt")
	res, err = NewPXARReader(pxar).ExtractFiltered(dest, []string{"ro/deep/d.txt"}, true)
	check("one file below them", res, err, true)
	change("ro/deep/d.txt")
	res, err = NewPXARReader(pxar).ExtractWithRewriterParallel(rewriter, []string{"ro/deep/d.txt"}, true, 4)
	check("one file below them, parallel", res, err, true)
	// Without overwrite: nothing replaced, folders unchanged.
	res, err = NewPXARReader(pxar).ExtractFiltered(dest, nil, false)
	check("restore again without overwrite", res, err, false)

	// A restore that does not apply folder modes (the text-mode window's)
	// puts the folders it opened back with RelockDirs.
	change(all...)
	res, err = NewPXARReader(pxar).ExtractFiltered(dest, nil, true)
	must(err)
	opened := 0
	for _, f := range res {
		if f.Unlocked {
			opened++
		}
	}
	if opened != len(want) {
		t.Errorf("RelockDirs: %d folders marked opened, want %d", opened, len(want))
	}
	RelockDirs(res)
	modes("RelockDirs")
}

// A folder the extraction opened gets the recorded bits its mode before the
// restore had, never more.
func TestApplyDirModeUnlocked(t *testing.T) {
	for _, c := range []struct {
		prev     os.FileMode
		recorded uint32
		want     os.FileMode
	}{
		{0o555, 0o555, 0o555},
		{0o555, 0o755, 0o555},
		{0o500, 0o755, 0o500},
		{0o555, 0o700, 0o500},
	} {
		d := filepath.Join(t.TempDir(), "d")
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(d, c.prev|0o300); err != nil {
			t.Fatal(err)
		}
		applyDirMode(d, uint32(IFDIR)|c.recorded, c.prev, true)
		fi, err := os.Stat(d)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != c.want {
			t.Errorf("prev %04o, recorded %04o: got %04o, want %04o", c.prev, c.recorded, fi.Mode().Perm(), c.want)
		}
		_ = os.Chmod(d, 0o755)
	}
}
