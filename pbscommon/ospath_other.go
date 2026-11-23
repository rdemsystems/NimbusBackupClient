//go:build !windows

package pbscommon

// OSPath returns path unchanged: only Windows takes some names (CON, NUL,
// COM1...) for devices (ospath_windows.go).
func OSPath(path string) string { return path }
