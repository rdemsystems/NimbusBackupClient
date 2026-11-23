//go:build windows

package pbscommon

import (
	"os"
	"path/filepath"
	"strings"
)

// OSPath returns the form of path to hand to Windows. Windows takes a name
// such as CON, NUL, COM1 or LPT1 (also with an extension, as nul.txt or
// com1.log) as a device wherever it appears in a path, so a file or folder of
// that name, made by Git Bash, WSL or a Linux program on a shared disk,
// cannot be reached by its normal path: "Cannot stat directory ...
// GetFileAttributesEx ... cannot find the file". Its \\?\ form turns that
// translation off and reaches the real entry. Any other path comes back
// unchanged, so logs and messages keep the paths people know.
//
// The path is made absolute with plain string handling: filepath.Abs asks
// Windows (GetFullPathName), which turns such a path into the device itself.
func OSPath(path string) string {
	if path == "" || strings.HasPrefix(path, `\\?\`) || strings.HasPrefix(path, `\\.\`) ||
		strings.HasPrefix(path, `//?/`) || strings.HasPrefix(path, `//./`) {
		return path
	}
	if !hasReservedName(path) {
		return path
	}
	p := filepath.Clean(path)
	if !filepath.IsAbs(p) {
		wd, err := os.Getwd()
		if err != nil {
			return path
		}
		p = filepath.Join(wd, p)
	}
	if strings.HasPrefix(p, `\\`) {
		return `\\?\UNC\` + p[2:]
	}
	return `\\?\` + p
}

// hasReservedName reports whether any element of path is a Windows device
// name (see isReservedName).
func hasReservedName(path string) bool {
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '\\' || path[i] == '/' {
			if i > start && isReservedName(path[start:i]) {
				return true
			}
			start = i + 1
		}
	}
	return false
}

// isReservedName reports whether Windows takes name for a device: CON, PRN,
// AUX, NUL, COM0 to COM9, LPT0 to LPT9 (also with a superscript 1, 2 or 3),
// CONIN$ and CONOUT$, in any case, alone or before an extension, and with
// trailing spaces or dots ignored ("nul", "NUL.txt", "com1 .log").
func isReservedName(name string) bool {
	if i := strings.IndexByte(name, '.'); i >= 0 {
		name = name[:i]
	}
	name = strings.TrimRight(name, " ")
	switch strings.ToUpper(name) {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}
	if len(name) < 4 {
		return false
	}
	head := strings.ToUpper(name[:3])
	if head != "COM" && head != "LPT" {
		return false
	}
	switch name[3:] {
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "\u00b9", "\u00b2", "\u00b3":
		return true
	}
	return false
}
