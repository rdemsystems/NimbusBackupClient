//go:build !linux

package pbscommon

// PrivateMountNote is Linux only (private_mount_linux.go): elsewhere there
// is no such mount to explain.
func PrivateMountNote(string) string { return "" }
