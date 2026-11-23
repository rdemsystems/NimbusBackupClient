package pbscommon

import (
	"os"
	"os/exec"
	"path/filepath"
)

// systemToolDirs are where Linux keeps tools such as modprobe. An ordinary
// user's PATH often has neither sbin folder (Debian, Raspberry Pi OS), and
// neither does the PATH of a root shell opened with `su` rather than `su -`.
var systemToolDirs = []string{"/usr/sbin", "/sbin", "/usr/bin", "/bin"}

// SystemTool returns the program to run for a system tool (modprobe, ...):
// what PATH finds, else the first of the usual system folders that has it,
// else the bare name, so the error then names the tool.
func SystemTool(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	for _, d := range systemToolDirs {
		p := filepath.Join(d, name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return p
		}
	}
	return name
}
