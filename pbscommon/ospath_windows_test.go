//go:build windows

package pbscommon

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestIsReservedName(t *testing.T) {
	for name, want := range map[string]bool{
		"nul": true, "NUL": true, "Nul.txt": true, "con": true, "prn.log": true, "aux": true,
		"com1": true, "COM9": true, "com0": true, "lpt1.txt": true, "LPT9": true, "com\u00b9": true,
		"conin$": true, "CONOUT$": true, "nul .txt": true, "com1.": true,
		"nul2": false, "console": false, "com10": false, "comx": false, "lpt": false, "aux_old": false,
		"auxiliary.txt": false, "connect": false, "x.nul": false, "": false, "com": false,
	} {
		if got := isReservedName(name); got != want {
			t.Errorf("isReservedName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestOSPath(t *testing.T) {
	for _, c := range [][2]string{
		{`C:\data\file.txt`, `C:\data\file.txt`},
		{`C:\data\com1\file.txt`, `\\?\C:\data\com1\file.txt`},
		{`C:\data\nul.txt`, `\\?\C:\data\nul.txt`},
		{`C:/data/aux/x`, `\\?\C:\data\aux\x`},
		{`\\server\share\con\a`, `\\?\UNC\server\share\con\a`},
		{`\\?\C:\data\com1`, `\\?\C:\data\com1`},
		{`\\?\GLOBALROOT\Device\X\nul`, `\\?\GLOBALROOT\Device\X\nul`},
		{``, ``},
	} {
		if got := OSPath(c[0]); got != c[1] {
			t.Errorf("OSPath(%q) = %q, want %q", c[0], got, c[1])
		}
	}
}

// Files and folders named like Windows devices (Git Bash and WSL make them)
// are backed up and restored, not counted as read errors that fail the run.
func TestReservedNamesBackUpAndRestore(t *testing.T) {
	root := t.TempDir()
	// t.TempDir's own clean-up removes by the normal path, which cannot reach
	// these names; remove them in the \\?\ form first (clean-ups run last
	// registered first).
	t.Cleanup(func() { _ = os.RemoveAll(`\\?\` + root) })
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}
	long := func(p string) string { return `\\?\` + p }
	files := map[string]string{
		`com1\inside.txt`: "inside a folder named com1",
		`nul.txt`:         "a file named nul.txt",
		`aux`:             "a file named aux",
		`plain.txt`:       "an ordinary file",
	}
	if err := os.Mkdir(long(filepath.Join(data, "com1")), 0o755); err != nil {
		t.Fatalf("make com1: %v", err)
	}
	for rel, content := range files {
		if err := os.WriteFile(long(filepath.Join(data, rel)), []byte(content), 0o644); err != nil {
			t.Fatalf("make %s: %v", rel, err)
		}
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
		t.Fatalf("device names counted as read errors: %q", a.ReadErrors)
	}
	for _, content := range files {
		if !bytes.Contains(out.Bytes(), []byte(content)) {
			t.Errorf("%q missing from the archive; skipped: %q", content, a.SkippedFiles)
		}
	}

	entries, err := NewPXARReader(out.Bytes()).ListEntries()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Path)
	}
	sort.Strings(names)
	got := strings.Join(names, ",")
	for _, want := range []string{"com1", "com1/inside.txt", "nul.txt", "aux"} {
		if !strings.Contains(","+got+",", ","+want+",") {
			t.Errorf("archive lists %s, want %s in it", got, want)
		}
	}

	check := func(dest, how string, ex []PXARExtractedFile) {
		for _, f := range ex {
			if f.Skipped {
				t.Errorf("%s: %s skipped: %s", how, f.Path, f.SkipReason)
			}
		}
		for rel, content := range files {
			b, err := os.ReadFile(long(filepath.Join(dest, rel)))
			if err != nil || string(b) != content {
				t.Errorf("%s: %s restored as %q, %v", how, rel, b, err)
			}
		}
	}
	dest := filepath.Join(root, "restored")
	ex, err := NewPXARReader(out.Bytes()).ExtractAll(dest)
	if err != nil {
		t.Fatal(err)
	}
	check(dest, "restore", ex)

	dest2 := filepath.Join(root, "restored-parallel")
	ex, err = NewPXARReader(out.Bytes()).ExtractWithRewriterParallel(func(p string) string {
		return filepath.Join(dest2, filepath.FromSlash(p))
	}, nil, false, 4)
	if err != nil {
		t.Fatal(err)
	}
	check(dest2, "parallel restore", ex)
}
