package main

import (
	"regexp"
	"strings"
	"testing"
)

// PBS's archive name format (PROXMOX_SAFE_ID_REGEX).
var pbsArchiveName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._\-]*$`)

func TestArchiveNames(t *testing.T) {
	dirs := []string{
		`C:\Users\tacio\Downloads`,
		`C:\Users\tacio\smart`,
		`/`,
		`/backup`,
		`/home/.config`,
		`/srv/Área`,
		`.hidden`,
		`-dash`,
	}
	want := []string{
		"C_Users_tacio_Downloads.pxar.didx",
		"C_Users_tacio_smart.pxar.didx",
		"root.pxar.didx",
		"backup_folder.pxar.didx",
		"home_.config.pxar.didx",
		"srv_rea.pxar.didx",
		"hidden.pxar.didx",
		"dash.pxar.didx",
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

// Folders whose labels collide get distinct names that do not depend on
// their order; the same folder listed twice still gets two names.
func TestArchiveNamesCollisions(t *testing.T) {
	a := ArchiveNames([]string{`D:\Data Files`, `D:\Data-Files`, `C:\x`, `C:\X`})
	b := ArchiveNames([]string{`C:\X`, `D:\Data-Files`, `C:\x`, `D:\Data Files`})
	seen := map[string]bool{}
	for _, n := range a {
		if seen[strings.ToLower(n)] {
			t.Errorf("name %q given twice in %v", n, a)
		}
		seen[strings.ToLower(n)] = true
		if !pbsArchiveName.MatchString(n) {
			t.Errorf("%q is not a valid PBS archive name", n)
		}
	}
	if !strings.HasPrefix(a[0], "D_Data-Files_") || !strings.HasPrefix(a[1], "D_Data-Files_") {
		t.Errorf("colliding labels should keep the label and add a hash: %v", a)
	}
	if a[0] != b[3] || a[1] != b[1] {
		t.Errorf("colliding names moved with the folder order: %v vs %v", a, b)
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
