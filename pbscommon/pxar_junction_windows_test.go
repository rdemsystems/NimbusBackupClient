//go:build windows

package pbscommon

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Junctions are skipped as links, not opened: with Go 1.23 and later they
// come back from Lstat as os.ModeIrregular, not os.ModeSymlink, and were
// opened as files. Windows' own deny-listed ones (Documents\My Music and
// friends) then failed with "Access is denied" and counted as read errors,
// so a backup of Documents ended partial (seen on a laptop, 2026-10-31).
func TestJunctionsSkippedNotReadErrors(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	data := filepath.Join(root, "data")
	for _, d := range []string{target, data} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(target, "inside.txt"), []byte("reached through a junction"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "keep.txt"), []byte("kept"), 0o644); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(data, "plain link")
	denied := filepath.Join(data, "My Music")
	for _, j := range []string{plain, denied} {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", j, target).CombinedOutput(); err != nil {
			t.Fatalf("mklink /J %s: %v: %s", j, err, out)
		}
	}
	// As Windows sets up Documents\My Music: everyone denied listing it.
	if out, err := exec.Command("icacls", denied, "/deny", "*S-1-1-0:(RD)", "/L").CombinedOutput(); err != nil {
		t.Fatalf("icacls: %v: %s", err, out)
	}
	defer exec.Command("icacls", denied, "/remove:d", "*S-1-1-0", "/L").Run()

	if fi, err := os.Lstat(denied); err == nil {
		t.Logf("Lstat of a junction: mode %v (symlink %v, irregular %v)", fi.Mode(), fi.Mode()&os.ModeSymlink != 0, fi.Mode()&os.ModeIrregular != 0)
	}

	var out bytes.Buffer
	a := &PXARArchive{ArchiveName: "test.pxar.didx"}
	a.WriteCB = func(b []byte) error { out.Write(b); return nil }
	a.CatalogWriteCB = func(b []byte) error { return nil }
	a.Create()
	if _, err := a.WriteDir(data, "", true); err != nil {
		t.Fatal(err)
	}
	if len(a.ReadErrors) != 0 {
		t.Errorf("junctions counted as read errors: %q", a.ReadErrors)
	}
	skipped := strings.Join(a.SkippedFiles, "\n")
	for _, j := range []string{plain, denied} {
		if !strings.Contains(skipped, "Junction point (skipped): "+j) {
			t.Errorf("%s not skipped as a junction; skipped: %q", j, a.SkippedFiles)
		}
	}
	if bytes.Contains(out.Bytes(), []byte("reached through a junction")) {
		t.Error("a junction's target was backed up through it")
	}
	if !bytes.Contains(out.Bytes(), []byte("kept")) {
		t.Error("the folder's own file is missing")
	}
}

// Windows has real symbolic links as well as junctions (mklink and mklink /D,
// made by Developer Mode, Git, WSL...). Both are skipped, each under its own
// name, so the list of skipped items does not call a symbolic link a junction.
func TestSymlinksAndJunctionsNamedApart(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	data := filepath.Join(root, "data")
	for _, d := range []string{target, data} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	tfile := filepath.Join(target, "t.txt")
	if err := os.WriteFile(tfile, []byte("reached through a link"), 0o644); err != nil {
		t.Fatal(err)
	}
	junction := filepath.Join(data, "junction")
	dirLink := filepath.Join(data, "dir link")
	fileLink := filepath.Join(data, "file link.txt")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", junction, target).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J: %v: %s", err, out)
	}
	// A symbolic link needs administrator rights or Developer Mode.
	if out, err := exec.Command("cmd", "/c", "mklink", "/D", dirLink, target).CombinedOutput(); err != nil {
		t.Skipf("cannot make a symbolic link here (administrator or Developer Mode needed): %v: %s", err, out)
	}
	if out, err := exec.Command("cmd", "/c", "mklink", fileLink, tfile).CombinedOutput(); err != nil {
		t.Fatalf("mklink: %v: %s", err, out)
	}

	var out bytes.Buffer
	a := &PXARArchive{ArchiveName: "test.pxar.didx"}
	a.WriteCB = func(b []byte) error { out.Write(b); return nil }
	a.CatalogWriteCB = func(b []byte) error { return nil }
	a.Create()
	if _, err := a.WriteDir(data, "", true); err != nil {
		t.Fatal(err)
	}
	if len(a.ReadErrors) != 0 {
		t.Errorf("links counted as read errors: %q", a.ReadErrors)
	}
	want := map[string]string{
		junction: "Junction point (skipped): ",
		dirLink:  "Symbolic link (skipped): ",
		fileLink: "Symbolic link (skipped): ",
	}
	for p, prefix := range want {
		found := false
		for _, s := range a.SkippedFiles {
			if s == prefix+p {
				found = true
			}
		}
		if !found {
			t.Errorf("%s not listed as %q; skipped: %q", p, prefix, a.SkippedFiles)
		}
	}
	if bytes.Contains(out.Bytes(), []byte("reached through a link")) {
		t.Error("a link's target was backed up through it")
	}
}

// An irregular entry that is not a link (here: a file whose reparse tag is
// not a name surrogate cannot be made without a driver, so a plain file and
// a plain folder stand in) is not taken for a link.
func TestIsLinkEntryPlainEntries(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{f, dir} {
		fi, err := os.Lstat(p)
		if err != nil {
			t.Fatal(err)
		}
		if isLinkEntry(p, fi) {
			t.Errorf("%s taken for a link", p)
		}
	}
}
