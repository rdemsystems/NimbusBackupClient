//go:build windows

package pbscommon

import "os"

// replaceFile moves the finished temporary file over the restore target.
//
// Windows will not replace a file that has the read-only attribute (the
// rename fails with "Access is denied"), so a second restore over an earlier
// one, or a Roll back, failed on every read-only file, including the
// read-only .proxmox_backup_client_meta.json in each backup. When the plain
// rename fails and the target is a read-only regular file, the attribute is
// cleared and the rename tried once more; if that still fails the attribute
// is put back, so the old file is left as it was. The attributes recorded in
// the backup are applied to the new file afterwards, as for any other file.
func replaceFile(tmp, dst string) error {
	err := os.Rename(tmp, dst)
	if err == nil {
		return nil
	}
	fi, lerr := os.Lstat(dst)
	if lerr != nil || !fi.Mode().IsRegular() || fi.Mode().Perm()&0o200 != 0 {
		return err // not a read-only file: a different problem
	}
	if os.Chmod(dst, 0o666) != nil {
		return err
	}
	if rerr := os.Rename(tmp, dst); rerr != nil {
		_ = os.Chmod(dst, fi.Mode().Perm())
		return rerr
	}
	return nil
}
