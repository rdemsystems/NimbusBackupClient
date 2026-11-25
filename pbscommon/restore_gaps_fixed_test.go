//go:build unix

package pbscommon

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// entryWith is a PXAR_ENTRY body with an owner and sub-second time.
func entryWith(mode uint64, uid, gid uint32, secs uint64, nanos uint32) []byte {
	out := make([]byte, 40)
	binary.LittleEndian.PutUint64(out[0:8], mode)
	binary.LittleEndian.PutUint32(out[16:20], uid)
	binary.LittleEndian.PutUint32(out[20:24], gid)
	binary.LittleEndian.PutUint64(out[24:32], secs)
	binary.LittleEndian.PutUint32(out[32:36], nanos)
	return out
}

// placeholderArchive is an archive in the shape Windows (and Linux clients
// before 2.0) write: every entry 0777, owned by 1000:1000, whole seconds.
// rootNanos lets a test give the top folder a real sub-second time.
func placeholderArchive(t *testing.T, rootNanos uint32) []byte {
	t.Helper()
	var b []byte
	b = append(b, pxarSection(t, PXAR_ENTRY, entryWith(IFDIR|0o777, 1000, 1000, 1600000000, rootNanos))...)
	b = append(b, pxarSection(t, PXAR_FILENAME, []byte("docs\x00"))...)
	b = append(b, pxarSection(t, PXAR_ENTRY, entryWith(IFDIR|0o777, 1000, 1000, 1600000000, 0))...)
	b = append(b, pxarSection(t, PXAR_FILENAME, []byte("report.txt\x00"))...)
	b = append(b, pxarSection(t, PXAR_ENTRY, entryWith(IFREG|0o777, 1000, 1000, 1600000000, 0))...)
	b = append(b, pxarSection(t, PXAR_PAYLOAD, []byte("from a Windows backup\n"))...)
	b = append(b, pxarSection(t, PXAR_GOODBYE, nil)...)
	// The writer's own meta file has a real mode, 0444.
	b = append(b, pxarSection(t, PXAR_FILENAME, []byte("meta.json\x00"))...)
	b = append(b, pxarSection(t, PXAR_ENTRY, entryWith(IFREG|0o444, 1000, 1000, 1600000000, 0))...)
	b = append(b, pxarSection(t, PXAR_PAYLOAD, []byte("{}"))...)
	b = append(b, pxarSection(t, PXAR_GOODBYE, nil)...)
	return b
}

func TestRecordsNoPermissions(t *testing.T) {
	if !NewPXARReader(placeholderArchive(t, 0)).RecordsNoPermissions() {
		t.Error("an archive in the Windows shape was not recognised")
	}
	if NewPXARReader(placeholderArchive(t, 123)).RecordsNoPermissions() {
		t.Error("a top folder with a sub-second time was taken for a placeholder")
	}
	if NewPXARReader(standardPXAR(t)).RecordsNoPermissions() {
		t.Error("an archive with real modes (0755, root) was taken for a placeholder")
	}
	if NewPXARReader(nil).RecordsNoPermissions() {
		t.Error("an empty archive was taken for a placeholder")
	}
}

// A restore of such an archive gives files the fixed mode, never 0777, and
// no recorded owner; folders stay as created, for root and for a user.
func TestFixedPermissionsRestore(t *testing.T) {
	root := os.Geteuid() == 0
	dest := t.TempDir()
	pr := NewPXARReader(placeholderArchive(t, 0))
	pr.SetRestoreOwnership(root)
	pr.SetFixedPermissions(0o644)
	res, err := pr.ExtractAll(dest)
	if err != nil {
		t.Fatal(err)
	}
	FinishDirs(res, root)

	f, err := os.Stat(filepath.Join(dest, "docs", "report.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if f.Mode().Perm() != 0o644 {
		t.Errorf("file mode %04o, want 0644 (the archive records 0777)", f.Mode().Perm())
	}
	if m, err := os.Stat(filepath.Join(dest, "meta.json")); err != nil || m.Mode().Perm() != 0o444 {
		t.Errorf("a file with a real recorded mode (0444): %v, %v; want it kept", m, err)
	}
	d, err := os.Stat(filepath.Join(dest, "docs"))
	if err != nil {
		t.Fatal(err)
	}
	if d.Mode().Perm()&0o022 != 0 {
		t.Errorf("folder mode %04o is writable by others (the archive records 0777)", d.Mode().Perm())
	}
	me := uint32(os.Geteuid())
	for _, fi := range []os.FileInfo{f, d} {
		if uid := fi.Sys().(*syscall.Stat_t).Uid; uid != me {
			t.Errorf("%s owned by %d, want the restoring account %d (the archive records 1000)", fi.Name(), uid, me)
		}
	}
	for _, r := range res {
		if r.IsDir && !r.FixedPerms {
			t.Errorf("folder %s not marked FixedPerms", r.Path)
		}
	}
}

// Restoring one file creates the folders on the way to it with their
// recorded permissions, through the archive index (no full walk), and
// leaves a folder that was already there as it is.
func TestSelectionParentsGetRecordedModes(t *testing.T) {
	src := t.TempDir()
	ssh := filepath.Join(src, "home", ".ssh")
	if err := os.MkdirAll(ssh, 0o700); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(ssh, "id_ed25519")
	if err := os.WriteFile(key, []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}
	for p, m := range map[string]os.FileMode{ssh: 0o700, filepath.Join(src, "home"): 0o750, key: 0o600} {
		if err := os.Chmod(p, m); err != nil {
			t.Fatal(err)
		}
	}
	pxar, _, _ := writePosixArchive(t, src)
	sel := []string{"home/.ssh/id_ed25519"}
	if _, ok := NewPXARReader(pxar).resolveIncludeSpans(sel); !ok {
		t.Fatal("the selection does not resolve through the index; the test would not cover the fast path")
	}
	root := os.Geteuid() == 0

	restore := func(dest string) {
		t.Helper()
		pr := NewPXARReader(pxar)
		pr.SetRestoreOwnership(root)
		res, err := pr.ExtractFiltered(dest, sel, true)
		if err != nil {
			t.Fatal(err)
		}
		FinishDirs(res, root)
	}
	perm := func(p string) os.FileMode {
		t.Helper()
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		return fi.Mode().Perm()
	}

	dest := t.TempDir()
	restore(dest)
	for p, want := range map[string]os.FileMode{"home": 0o750, "home/.ssh": 0o700, "home/.ssh/id_ed25519": 0o600} {
		if got := perm(filepath.Join(dest, p)); got != want {
			t.Errorf("%s: %04o, want %04o", p, got, want)
		}
	}

	// home already there with its own mode: left alone; .ssh created.
	dest2 := t.TempDir()
	if err := os.Mkdir(filepath.Join(dest2, "home"), 0o751); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dest2, "home"), 0o751); err != nil {
		t.Fatal(err)
	}
	restore(dest2)
	if got := perm(filepath.Join(dest2, "home")); got != 0o751 {
		t.Errorf("existing home: %04o, want it left at 0751", got)
	}
	if got := perm(filepath.Join(dest2, "home", ".ssh")); got != 0o700 {
		t.Errorf("created .ssh under an existing home: %04o, want 0700", got)
	}
}

func TestSelectionAncestors(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a", "b", "c.txt"), []byte("c"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(src, "a"), 0o711); err != nil {
		t.Fatal(err)
	}
	pxar, _, _ := writePosixArchive(t, src)
	got, err := NewPXARReader(pxar).selectionAncestors("a/b/c.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Path != "a" || got[1].Path != "a/b" || !got[0].IsDir {
		t.Fatalf("ancestors = %+v, want a and a/b", got)
	}
	if got[0].Mode&0o777 != 0o711 {
		t.Errorf("a recorded %04o, want 0711", got[0].Mode&0o777)
	}
	if _, err := NewPXARReader(pxar).selectionAncestors("a/missing/c.txt"); err == nil {
		t.Error("a missing folder on the way resolved")
	}
}
