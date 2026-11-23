//go:build unix

package pbscommon

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// On every Unix (not only Linux) the archive records the real mode, owner and
// nanosecond mtime, and stores symbolic links.
func TestEntryMetaPosixShape(t *testing.T) {
	if !posixFidelity || !KeepsSymlinks {
		t.Fatal("posixFidelity and KeepsSymlinks must be true on Unix")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(f, 0o640); err != nil {
		t.Fatal(err)
	}
	mt := time.Unix(1700000000, 123456789)
	if err := os.Chtimes(f, mt, mt); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(f)
	if err != nil {
		t.Fatal(err)
	}
	mode, uid, gid, secs, nanos := entryMeta(fi, IFREG)
	st := fi.Sys().(*syscall.Stat_t)
	if mode != IFREG|0o640 || uid != st.Uid || gid != st.Gid {
		t.Errorf("mode %o uid %d gid %d, want %o %d %d", mode, uid, gid, IFREG|0o640, st.Uid, st.Gid)
	}
	// Some file systems keep less than nanosecond precision: compare with
	// what the file system reports rather than with mt.
	if secs != uint64(fi.ModTime().Unix()) || nanos != uint32(fi.ModTime().Nanosecond()) {
		t.Errorf("mtime %d.%09d, want %d.%09d", secs, nanos, fi.ModTime().Unix(), fi.ModTime().Nanosecond())
	}

	l := filepath.Join(dir, "l")
	if err := os.Symlink("f", l); err != nil {
		t.Fatal(err)
	}
	li, err := os.Lstat(l)
	if err != nil {
		t.Fatal(err)
	}
	if lmode, _, _, _, _ := entryMeta(li, IFLNK); lmode&IFLNK != IFLNK {
		t.Errorf("link mode %o lacks IFLNK", lmode)
	}
}
