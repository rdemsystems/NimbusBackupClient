//go:build unix

package pbscommon

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// applyModeOwner sets ownership (when owner is true and the process is root)
// and then the permission bits, in that order because chown clears setuid and
// setgid. With owner false only the plain 0777 bits are applied, which is what
// restores have always done.
func applyModeOwner(path string, mode, uid, gid uint32, owner bool) {
	if !owner {
		_ = os.Chmod(path, os.FileMode(mode&0o777))
		return
	}
	if os.Geteuid() == 0 {
		_ = os.Lchown(path, int(uid), int(gid))
	}
	_ = os.Chmod(path, posixFileMode(mode))
}

// posixFileMode converts st_mode permission, setuid, setgid and sticky bits
// into an os.FileMode.
func posixFileMode(mode uint32) os.FileMode {
	m := os.FileMode(mode & 0o777)
	if mode&uint32(ISUID) != 0 {
		m |= os.ModeSetuid
	}
	if mode&uint32(ISGID) != 0 {
		m |= os.ModeSetgid
	}
	if mode&uint32(ISVTX) != 0 {
		m |= os.ModeSticky
	}
	return m
}

// chownLink sets a symlink's own owner without following it.
func chownLink(path string, uid, gid uint32) {
	if os.Geteuid() == 0 {
		_ = os.Lchown(path, int(uid), int(gid))
	}
}

// setLinkTime sets a symlink's own mtime without following it. utimensat
// with AT_SYMLINK_NOFOLLOW exists on every Unix, but its flag value differs
// (0x100 on Linux, 0x200 on FreeBSD, ...), so it goes through x/sys.
func setLinkTime(path string, secs int64, nanos uint32) {
	ts := []unix.Timespec{
		{Sec: secs, Nsec: int64(nanos)},
		{Sec: secs, Nsec: int64(nanos)},
	}
	_ = unix.UtimesNanoAt(unix.AT_FDCWD, path, ts, unix.AT_SYMLINK_NOFOLLOW)
}

// ApplyDirMetadata applies a restored folder's recorded owner and full mode
// bits. Run after the folder's contents are written, because a restrictive
// mode would otherwise block the restore itself.
func ApplyDirMetadata(path string, mode, uid, gid uint32) {
	applyModeOwner(path, mode, uid, gid, true)
}

// ApplyDirMode applies a folder's recorded permission bits for a restore that
// does not run as root: the bits the archive removed are removed, none is
// added (so the folder is never more open than at its creation, 0755 under
// the usual umask), and setuid, setgid and sticky are not applied, as for
// files (the folder keeps the ones it has).
func ApplyDirMode(path string, mode uint32) {
	applyDirMode(path, mode, 0, false)
}

// applyDirMode is ApplyDirMode for a folder the restore opened
// (openDirForRestore) when unlocked is true: the recorded bits are then taken
// from prev, its mode before the restore, not from its opened mode.
func applyDirMode(path string, mode uint32, prev os.FileMode, unlocked bool) {
	fi, err := os.Lstat(path)
	if err != nil || !fi.IsDir() {
		return
	}
	cur := fi.Mode().Perm()
	base := cur
	if unlocked {
		base = prev.Perm()
	}
	if want := base & os.FileMode(mode&0o777); want != cur {
		_ = os.Chmod(path, want|fi.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky))
	}
}

// dirUnlockApplies: only a restore that does not run as root can be stopped
// by a folder's permissions.
func dirUnlockApplies() bool { return os.Geteuid() != 0 }

// openDirForRestore gives the owner write and search permission (u+wx) on a
// folder the restoring user owns and lacks them on, and returns its mode
// before, special bits included. It does nothing to a folder of another
// account, which the user could not change anyway.
func openDirForRestore(dir string) (os.FileMode, bool) {
	euid := os.Geteuid()
	if euid == 0 {
		return 0, false
	}
	fi, err := os.Lstat(dir)
	if err != nil || !fi.IsDir() || fi.Mode().Perm()&0o300 == 0o300 {
		return 0, false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || uint64(st.Uid) != uint64(euid) {
		return 0, false
	}
	prev := fi.Mode() & (os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
	if os.Chmod(dir, prev|0o300) != nil {
		return 0, false
	}
	return prev, true
}
