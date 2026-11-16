package main

// Tests of #159: a folder backup's size estimate leaves out what the backup
// leaves out, so the progress total is what the archive writer reads.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pbscommon"
)

// sizeTree makes a folder of files with the given sizes under root.
func sizeTree(t *testing.T, root string, files map[string]int) {
	t.Helper()
	for p, n := range files {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(strings.Repeat("x", n)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// archiveReads writes root as the archive writer does for a folder
// backup (backupDirectory: ExcludeList, the folder as ExcludeRoot), with
// nowhere to send it, and returns the size and number of the files it read.
func archiveReads(t *testing.T, root string, excludes []string) (uint64, uint64) {
	t.Helper()
	var size, files uint64
	a := &pbscommon.PXARArchive{ArchiveName: "test.pxar.didx", ExcludeList: excludes, ExcludeRoot: root}
	a.WriteCB = func([]byte) error { return nil }
	a.OnFile = func(p string) {
		fi, err := os.Stat(p)
		if err != nil {
			t.Errorf("archive read %s: %v", p, err)
			return
		}
		size += uint64(fi.Size())
		files++
	}
	if _, err := a.WriteDir(root, "", true); err != nil {
		t.Fatal(err)
	}
	return size, files
}

func TestSizeEstimateLeavesOutExclusions(t *testing.T) {
	root := t.TempDir()
	sizeTree(t, root, map[string]int{
		"keep.txt":                    100, // kept
		"docs/a.txt":                  200, // kept
		"docs/b.tmp":                  300, // *.tmp: a file pattern
		"cache/big.bin":               5000,
		"cache/keep.pst":              50,  // !*.pst, but inside the excluded folder cache: not looked into
		"src/cache/obj.bin":           70,  // cache matches a folder of that name anywhere
		"logs/2026/x.log":             400, // logs/**/*.log
		"logs/2026/d/y.log":           500, // ** crosses any number of folders
		"logs/z.txt":                  60,  // kept
		"dist":                        80,  // **/dist/ matches folders only: this file is kept
		"sub/dist/out.js":             90,  // a folder called dist: left out
		"build/out.o":                 700, // build/
		"skipme/s.dat":                111, // an absolute pattern, anchored to the folder backed up
		"a/old.bak":                   33,  // *.bak
		"a/important.bak":             44,  // brought back by !important.bak
		"pagefile.sys":                999, // a system file every backup leaves out
		"System Volume Information/x": 13,
	})
	excludes := []string{
		"*.tmp",
		"cache",
		"!*.pst",
		"logs/**/*.log",
		"**/dist/",
		"build/",
		filepath.Join(root, "skipme"),
		"*.bak",
		"!important.bak",
	}

	size, files, err := calculateDirSizeFilesCtx(context.Background(), root, excludes, root)
	if err != nil {
		t.Fatal(err)
	}
	wantSize, wantFiles := uint64(100+200+60+80+44), uint64(5)
	if size != wantSize || files != wantFiles {
		t.Errorf("estimate %d bytes in %d files, want %d in %d", size, files, wantSize, wantFiles)
	}
	aSize, aFiles := archiveReads(t, root, excludes)
	if size != aSize || files != aFiles {
		t.Errorf("estimate %d bytes in %d files, the archive read %d in %d", size, files, aSize, aFiles)
	}

	// Without exclusions, everything but the system folder and file.
	size, files, err = calculateDirSizeFilesCtx(context.Background(), root, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	aSize, aFiles = archiveReads(t, root, nil)
	if size != aSize || files != aFiles {
		t.Errorf("no exclusions: estimate %d bytes in %d files, the archive read %d in %d", size, files, aSize, aFiles)
	}
	if want := uint64(100 + 200 + 300 + 5000 + 50 + 70 + 400 + 500 + 60 + 80 + 90 + 700 + 111 + 33 + 44); size != want {
		t.Errorf("no exclusions: estimate %d bytes, want %d", size, want)
	}
}

// The split analysis sizes each top-level folder on its own, with the
// patterns anchored to the folder the user chose (calculateDirSizeCtx with
// excludeRoot).
func TestSizeEstimateAnchoredToChosenFolder(t *testing.T) {
	root := t.TempDir()
	sizeTree(t, root, map[string]int{
		"logs/2026/x.log": 400,
		"logs/z.txt":      60,
	})
	size, err := calculateDirSizeCtx(context.Background(), filepath.Join(root, "logs"), []string{"logs/**/*.log"}, root)
	if err != nil {
		t.Fatal(err)
	}
	if size != 60 {
		t.Errorf("logs sized %d bytes, want 60", size)
	}
}
