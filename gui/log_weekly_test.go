package main

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWeekStartIsMonday(t *testing.T) {
	for in, want := range map[string]string{
		"2026-11-16 00:00:00": "20261116", // a Monday
		"2026-11-18 13:45:00": "20261116",
		"2026-11-22 23:59:59": "20261116", // the Sunday after
		"2026-11-23 00:00:01": "20261123",
		"2027-01-01 10:00:00": "20261228", // a week across the new year
	} {
		tm, _ := time.ParseInLocation("2006-01-02 15:04:05", in, time.Local)
		if got := weeklyLogFile(tm); got != "backup-logs-"+want+".log.gz" {
			t.Errorf("%s: %s, want the week of %s", in, got, want)
		}
	}
}

// Run logs of one week go into one file, those of another week into its own,
// and the viewer lists and reads each run as before.
func TestRunLogsFoldIntoWeekFilesAndReadAsBefore(t *testing.T) {
	dir := withLogFolder(t)
	runs := map[string]string{
		"backup-20261116-040100-docs.log.gz":                 "[BACKUP] [2026-11-16 04:01:00] docs run one",
		"backup-20261117-040100-docs.log.gz":                 "[BACKUP] [2026-11-17 04:01:00] docs run two",
		"backup-20261117-040100-docs.log.20261117-050000.gz": "[BACKUP] [2026-11-17 05:00:00] docs run two, rotated part",
		"backup-20261118-120000-photos.log.gz":               "[BACKUP] [2026-11-18 12:00:00] photos run",
		"backup-20261110-040100-docs.log.gz":                 "[BACKUP] [2026-11-10 04:01:00] the week before",
	}
	for n, line := range runs {
		writeLogLines(t, filepath.Join(dir, n), []string{line, "[RESULT] ok"}, true)
	}
	writeLogLines(t, filepath.Join(dir, "backup-20261118-130000-photos.log"), []string{"[BACKUP] [2026-11-18 13:00:00] running"}, false)

	foldRunLogs(dir, "", 0)

	left, _ := os.ReadDir(dir)
	var names []string
	for _, e := range left {
		names = append(names, e.Name())
	}
	got := strings.Join(names, " ")
	if got != "backup-20261118-130000-photos.log backup-logs-20261109.log.gz backup-logs-20261116.log.gz" {
		t.Fatalf("files after folding: %s", got)
	}

	files, err := (&App{}).ListLogFiles()
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]LogFileInfo{}
	for _, f := range files {
		listed[f.Name] = f
	}
	for n, line := range runs {
		f, ok := listed[n]
		if !ok {
			t.Errorf("%s is not listed", n)
			continue
		}
		if f.Kind != "run" || !f.Compressed || f.Size <= 0 {
			t.Errorf("%s listed as %+v", n, f)
		}
		v, err := (&App{}).ReadLog(LogQuery{Folder: f.Folder, Name: n})
		if err != nil {
			t.Errorf("read %s: %v", n, err)
			continue
		}
		if len(v.Lines) != 2 || v.Lines[0] != line {
			t.Errorf("%s reads %q, want only its own two lines", n, v.Lines)
		}
	}
	if listed["backup-logs-20261116.log.gz"].Kind != "backup" {
		t.Errorf("the week file is listed as %+v", listed["backup-logs-20261116.log.gz"])
	}
	// The week file read whole is the week's runs one after another.
	v, err := (&App{}).ReadLog(LogQuery{Name: "backup-logs-20261116.log.gz"})
	if err != nil || v.Matching != 8 {
		t.Errorf("the week read whole: %d lines, %v; want 8", v.Matching, err)
	}

	// Reports finds a run's log in the week file as it found the file.
	at, _ := time.ParseInLocation("2006-01-02 15:04", "2026-11-17 04:30", time.Local)
	if f, err := (&App{}).FindRunLog("docs", at.UnixMilli()); err != nil || f.Name != "backup-20261117-040100-docs.log.gz" {
		t.Errorf("FindRunLog = %q, %v", f.Name, err)
	}
}

// Folding again after the computer stopped between adding a run and removing
// its file does not add it twice.
func TestFoldingTwiceAddsARunOnce(t *testing.T) {
	dir := withLogFolder(t)
	name := "backup-20261116-040100-docs.log.gz"
	writeLogLines(t, filepath.Join(dir, name), []string{"[BACKUP] [2026-11-16 04:01:00] once"}, true)
	data, _ := os.ReadFile(filepath.Join(dir, name))
	foldRunLogs(dir, "", 0)
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil { // as if not removed
		t.Fatal(err)
	}
	foldRunLogs(dir, "", 0)
	scan, err := scanWeeklyLog(filepath.Join(dir, "backup-logs-20261116.log.gz"))
	if err != nil || len(scan.members) != 1 {
		t.Fatalf("the week holds %d runs, %v; want 1", len(scan.members), err)
	}
	if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
		t.Error("the run's own file is still there")
	}
}

// A week file whose last run was cut off part way (the computer stopped while
// it was added) loses only that part: the next run is added after the whole
// ones and every run can be read.
func TestACutOffEndIsDroppedBeforeTheNextRun(t *testing.T) {
	dir := withLogFolder(t)
	writeLogLines(t, filepath.Join(dir, "backup-20261116-040100-docs.log.gz"), []string{"[BACKUP] [2026-11-16 04:01:00] first"}, true)
	foldRunLogs(dir, "", 0)
	week := filepath.Join(dir, "backup-logs-20261116.log.gz")
	whole, _ := os.Stat(week)
	f, err := os.OpenFile(week, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte{0x1f, 0x8b, 8, 0, 0, 0}) // the start of a part, and no more
	_ = f.Close()
	writeLogLines(t, filepath.Join(dir, "backup-20261117-040100-docs.log.gz"), []string{"[BACKUP] [2026-11-17 04:01:00] second"}, true)
	foldRunLogs(dir, "", 0)
	scan, err := scanWeeklyLog(week)
	if err != nil || len(scan.members) != 2 || scan.goodEnd != scan.size || scan.members[1].Offset != whole.Size() {
		t.Fatalf("after the cut-off end: %+v, %v", scan, err)
	}
	r, ok := openWeeklyRunLog(dir, "backup-20261117-040100-docs.log.gz")
	if !ok {
		t.Fatal("the second run cannot be opened")
	}
	b, _ := io.ReadAll(r)
	_ = r.Close()
	if !strings.Contains(string(b), "second") || strings.Contains(string(b), "first") {
		t.Errorf("the second run reads %q", b)
	}
	// and the whole week reads with any gzip reader
	wf, _ := os.Open(week)
	defer func() { _ = wf.Close() }()
	zr, err := gzip.NewReader(wf)
	if err != nil {
		t.Fatal(err)
	}
	all, err := io.ReadAll(zr)
	if err != nil || !strings.Contains(string(all), "first") || !strings.Contains(string(all), "second") {
		t.Errorf("the week read whole: %q, %v", all, err)
	}
}

// A run log still being compressed (no gzip end yet) is left for later.
func TestAnUnfinishedRunLogIsLeft(t *testing.T) {
	dir := withLogFolder(t)
	name := "backup-20261116-040100-docs.log.gz"
	writeLogLines(t, filepath.Join(dir, name), []string{strings.Repeat("[BACKUP] [2026-11-16 04:01:00] x\n", 200)}, true)
	data, _ := os.ReadFile(filepath.Join(dir, name))
	if err := os.WriteFile(filepath.Join(dir, name), data[:len(data)-8], 0o600); err != nil {
		t.Fatal(err)
	}
	foldRunLogs(dir, "", 0)
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Errorf("the unfinished file was taken: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "backup-logs-20261116.log.gz")); !os.IsNotExist(err) {
		t.Error("an unfinished run log was added to the week")
	}
}

// With a prefix only that run's files are folded (the end of one run while
// another runs on).
func TestFoldOneRun(t *testing.T) {
	dir := withLogFolder(t)
	writeLogLines(t, filepath.Join(dir, "backup-20261116-040100-docs.log.gz"), []string{"a"}, true)
	writeLogLines(t, filepath.Join(dir, "backup-20261116-040100-docs.log.20261116-050000.gz"), []string{"b"}, true)
	writeLogLines(t, filepath.Join(dir, "backup-20261116-040100-docs2.log.gz"), []string{"c"}, true)
	foldRunLogs(dir, "backup-20261116-040100-docs.log", 0)
	if _, err := os.Stat(filepath.Join(dir, "backup-20261116-040100-docs2.log.gz")); err != nil {
		t.Errorf("another run's file was folded: %v", err)
	}
	scan, _ := scanWeeklyLog(filepath.Join(dir, "backup-logs-20261116.log.gz"))
	if len(scan.members) != 2 {
		t.Errorf("the week holds %d parts, want the run's 2", len(scan.members))
	}
}

// Weeks older than those kept are removed; this week and the ones kept stay.
func TestPruneWeeklyLogs(t *testing.T) {
	dir := withLogFolder(t)
	for _, n := range []string{"backup-logs-20261109.log.gz", "backup-logs-20261102.log.gz", "backup-logs-20261026.log.gz", "backup-logs-20261116.log.gz", "backup-gui.log"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ := os.ReadDir(dir)
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.Local)
	pruneWeeklyLogs(dir, entries, 3, now) // this week and the 2 before
	left, _ := os.ReadDir(dir)
	var names []string
	for _, e := range left {
		names = append(names, e.Name())
	}
	if got := strings.Join(names, " "); got != "backup-gui.log backup-logs-20261102.log.gz backup-logs-20261109.log.gz backup-logs-20261116.log.gz" {
		t.Errorf("left: %s", got)
	}
	pruneWeeklyLogs(dir, left, 0, now) // 0 keeps all
	if again, _ := os.ReadDir(dir); len(again) != len(left) {
		t.Error("0 weeks removed files")
	}
}
