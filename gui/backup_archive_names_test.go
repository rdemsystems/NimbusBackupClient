package main

import (
	"regexp"
	"testing"
)

// PBS's archive name format (PROXMOX_SAFE_ID_REGEX).
var pbsArchiveName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._\-]*$`)

func TestArchiveNames(t *testing.T) {
	dirs := []string{
		`C:\Users\tacio\Downloads`,
		`C:\Users\tacio\smart`,
		`C:\Users\tacio\smart`, // same folder twice
		`D:\Data Files`,
		`D:\Data-Files`, // same label as the one above
		`C:\Users\TACIO\Downloads`, // Windows paths are case-insensitive
		`/`,
		`/backup`,
		`/home/.config`,
		`/srv/Área`,
	}
	want := []string{
		"C_Users_tacio_Downloads.pxar.didx",
		"C_Users_tacio_smart.pxar.didx",
		"C_Users_tacio_smart_2.pxar.didx",
		"D_Data-Files.pxar.didx",
		"D_Data-Files_2.pxar.didx",
		"C_Users_TACIO_Downloads_2.pxar.didx",
		"root.pxar.didx",
		"backup_folder.pxar.didx",
		"home_.config.pxar.didx",
		"srv_rea.pxar.didx",
	}
	got := ArchiveNames(dirs)
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ArchiveNames(%q) = %q, want %q", dirs[i], got[i], want[i])
		}
		if !pbsArchiveName.MatchString(got[i]) {
			t.Errorf("%q is not a valid PBS archive name", got[i])
		}
	}
}

// A folder's archive name does not depend on the folders around it, so a
// job that gains or loses a folder keeps the others' incremental chain.
func TestArchiveNamesStable(t *testing.T) {
	a := ArchiveNames([]string{`C:\one`, `C:\two`})
	b := ArchiveNames([]string{`C:\zero`, `C:\two`, `C:\one`})
	if a[0] != b[2] || a[1] != b[1] {
		t.Errorf("names moved with the folder list: %v vs %v", a, b)
	}
}

// An archive name starting with a dot or a dash is refused by PBS.
func TestArchiveNamesLeadingCharacter(t *testing.T) {
	for _, dir := range []string{`.hidden`, `-dash`, `..`} {
		name := ArchiveNames([]string{dir})[0]
		if !pbsArchiveName.MatchString(name) {
			t.Errorf("ArchiveNames(%q) = %q, not a valid PBS archive name", dir, name)
		}
	}
}
