package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// A snapshot holding one archive per folder, as runOneSnapshotBackup writes it.
func archivesMeta() *BackupMeta {
	host, _ := os.Hostname()
	return &BackupMeta{
		BackupID: "pc1",
		Hostname: host,
		OS:       runtime.GOOS,
		Archives: []ArchiveOrigin{
			{Archive: "Downloads.pxar.didx", Label: "Downloads", OriginalPath: filepath.Join(string(filepath.Separator), "home", "u", "Downloads")},
			{Archive: "smart.pxar.didx", Label: "smart", OriginalPath: filepath.Join(string(filepath.Separator), "srv", "smart")},
		},
	}
}

func identity(p string) string { return p }

func TestRestoreArchiveJobsLegacy(t *testing.T) {
	jobs, err := restoreArchiveJobs(nil, []string{"a/b"}, identity)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].archive != legacyArchiveName || !reflect.DeepEqual(jobs[0].includes, []string{"a/b"}) {
		t.Fatalf("legacy snapshot: got %+v", jobs)
	}
	if jobs[0].rewriter("a/b") != "a/b" {
		t.Error("legacy snapshot: the rewriter was changed")
	}
}

func TestRestoreArchiveJobsSplitsSelection(t *testing.T) {
	meta := archivesMeta()

	// No selection: every archive, whole.
	jobs, err := restoreArchiveJobs(meta, nil, identity)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 || jobs[0].includes != nil || jobs[1].includes != nil {
		t.Fatalf("no selection: got %+v", jobs)
	}

	// Files of one archive, and another archive whole by its label.
	jobs, err = restoreArchiveJobs(meta, []string{"Downloads/x.iso", "Downloads/sub", "smart"}, identity)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2: %+v", len(jobs), jobs)
	}
	if jobs[0].archive != "Downloads.pxar.didx" || !reflect.DeepEqual(jobs[0].includes, []string{"x.iso", "sub"}) {
		t.Errorf("Downloads job: %+v", jobs[0])
	}
	if jobs[1].archive != "smart.pxar.didx" || jobs[1].includes != nil {
		t.Errorf("smart job: %+v", jobs[1])
	}
	// Paths inside an archive land under its label for the full rewriter.
	if got := jobs[0].rewriter("x.iso"); got != "Downloads/x.iso" {
		t.Errorf("rewriter(x.iso) = %q, want Downloads/x.iso", got)
	}
	if got := jobs[1].rewriter(""); got != "smart" {
		t.Errorf("rewriter(root) = %q, want smart", got)
	}

	// Only one archive selected: the other is not opened.
	jobs, _ = restoreArchiveJobs(meta, []string{"smart/z.txt"}, identity)
	if len(jobs) != 1 || jobs[0].archive != "smart.pxar.didx" {
		t.Errorf("one archive selected: %+v", jobs)
	}

	if _, err := restoreArchiveJobs(meta, []string{"other/f"}, identity); err == nil {
		t.Error("a path outside every archive was accepted")
	}
}

// In place, each archive goes back to its own folder.
func TestPathRewriterOriginalWithArchives(t *testing.T) {
	meta := archivesMeta()
	rw, err := buildPathRewriter(RestoreOptions{Mode: RestoreModeOriginal}, meta)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := rw("Downloads/sub/y.txt"), filepath.Join(meta.Archives[0].OriginalPath, "sub", "y.txt"); got != want {
		t.Errorf("rw(Downloads/sub/y.txt) = %q, want %q", got, want)
	}
	if got := rw("smart"); got != meta.Archives[1].OriginalPath {
		t.Errorf("rw(smart) = %q, want %q", got, meta.Archives[1].OriginalPath)
	}
	if got := rw("unknown/f"); got != "" {
		t.Errorf("rw(unknown/f) = %q, want it dropped", got)
	}

	// An archive whose folder is unknown blocks an in-place restore of it,
	// not of the others.
	meta.Archives[1].OriginalPath = ""
	rw, err = buildPathRewriter(RestoreOptions{Mode: RestoreModeOriginal}, meta)
	if err != nil {
		t.Fatal(err)
	}
	if got := rw("smart/z.txt"); got != "" {
		t.Errorf("rw(smart/z.txt) = %q, want it dropped", got)
	}
	jobs, _ := restoreArchiveJobs(meta, []string{"Downloads/x.iso"}, rw)
	if err := checkArchiveOrigins(meta, jobs); err != nil {
		t.Errorf("restoring Downloads only: %v", err)
	}
	jobs, _ = restoreArchiveJobs(meta, nil, rw)
	if err := checkArchiveOrigins(meta, jobs); err == nil {
		t.Error("in place accepted an archive whose folder is unknown")
	}
}

// Alternate location: each archive lands in a folder named after its label.
func TestPathRewriterAlternateWithArchives(t *testing.T) {
	dest := t.TempDir()
	rw, err := buildPathRewriter(RestoreOptions{Mode: RestoreModeAlternateAbs, DestPath: dest}, archivesMeta())
	if err != nil {
		t.Fatal(err)
	}
	jobs, _ := restoreArchiveJobs(archivesMeta(), []string{"smart/z.txt"}, rw)
	if got, want := jobs[0].rewriter("z.txt"), filepath.Join(dest, "smart", "z.txt"); got != want {
		t.Errorf("rewriter(z.txt) = %q, want %q", got, want)
	}
}

func TestJoinOriginPathWithArchives(t *testing.T) {
	meta := archivesMeta()
	meta.OS = "linux"
	meta.Archives[0].OriginalPath = "/home/u/Downloads"
	if got := joinOriginPath(meta, "Downloads/sub/y.txt"); got != "/home/u/Downloads/sub/y.txt" {
		t.Errorf("joinOriginPath = %q", got)
	}
	if got := joinOriginPath(meta, "Downloads"); got != "/home/u/Downloads" {
		t.Errorf("joinOriginPath(label) = %q", got)
	}
	if got := joinOriginPath(meta, "nope/f"); got != "" {
		t.Errorf("joinOriginPath(unknown) = %q, want empty", got)
	}
}
