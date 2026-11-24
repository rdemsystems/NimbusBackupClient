//go:build !windows

package pbscommon

import "os"

// replaceFile moves the finished temporary file over the restore target. On
// Unix a rename replaces a read-only file (only the folder's permission
// matters), so nothing else is needed. See replace_file_windows.go.
func replaceFile(tmp, dst string) error {
	return os.Rename(tmp, dst)
}
