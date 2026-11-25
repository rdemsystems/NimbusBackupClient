//go:build linux

package pbscommon

import (
	"os"
	"os/user"
	"strconv"
)

// PrivateMountNote explains why folder cannot be read when it is on a FUSE
// mount (a desktop share under /run/user/<uid>/gvfs, sshfs...) that only
// another user may enter (see private_mount.go). It is empty otherwise, so
// callers add it only to an error they already have.
func PrivateMountNote(folder string) string {
	info, _ := os.ReadFile("/proc/self/mountinfo")
	m, ok := findPrivateMount(string(info), folder, os.Geteuid())
	if !ok {
		return ""
	}
	name := ""
	if u, err := user.LookupId(strconv.Itoa(m.UID)); err == nil {
		name = u.Username
	}
	return privateMountNote(m, name)
}
