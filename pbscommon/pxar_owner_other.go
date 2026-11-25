//go:build !unix

package pbscommon

// Windows: owners and Unix modes are not applied; symbolic links are not
// restored (extractSymlink).

import "os"

func applyModeOwner(path string, mode, uid, gid uint32, owner bool) {
	_ = os.Chmod(path, os.FileMode(mode&0o777))
}

func chownLink(path string, uid, gid uint32) {}

func setLinkTime(path string, secs int64, nanos uint32) {}

func ApplyDirMetadata(path string, mode, uid, gid uint32) {}

func ApplyDirMode(path string, mode uint32) {}

func applyDirMode(path string, mode uint32, prev os.FileMode, unlocked bool) {}

func dirUnlockApplies() bool { return false }

func openDirForRestore(dir string) (os.FileMode, bool) { return 0, false }
