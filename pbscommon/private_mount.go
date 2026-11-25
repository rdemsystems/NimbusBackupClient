package pbscommon

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Private desktop mounts (Linux).
//
// A share opened from a Linux desktop's file manager (Files, Nemo, Dolphin:
// smb://, sftp://, a phone over MTP) is mounted by gvfs under
// /run/user/<uid>/gvfs, as a FUSE mount that only that user may enter. Linux
// refuses every other account, root included, unless the mount was made with
// allow_other, and the mount is gone while the user is logged out. The
// background service runs as root, so a Backup Set with such a folder failed
// with a bare "permission denied" (found on linuxclient 2026-11-25). The same
// holds for any FUSE mount a user made without allow_other (sshfs, rclone).
//
// PrivateMountNote says what is going on and what to do instead, for a folder
// the backup could not open. It is empty for any other folder.

// gvfsPath matches a folder under a user's gvfs mount: /run/user/<uid>/gvfs.
var gvfsPath = regexp.MustCompile(`^/run/user/(\d+)/gvfs(/|$)`)

// privateMount is a FUSE mount that only its owner may enter.
type privateMount struct {
	MountPoint string
	UID        int
}

// findPrivateMount looks path up in mountinfo (the text of
// /proc/self/mountinfo) and reports the private FUSE mount it is on, if it is
// on one that euid may not enter. A folder under /run/user/<uid>/gvfs counts
// even when nothing is mounted there (its user is logged out).
func findPrivateMount(mountinfo, path string, euid int) (privateMount, bool) {
	path = filepath.Clean(path)
	best := ""
	var bestFS, bestSuper string
	sc := bufio.NewScanner(strings.NewReader(mountinfo))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		mp, fstype, super, ok := parseMountinfoLine(sc.Text())
		if !ok || !pathWithin(path, mp) || len(mp) <= len(best) {
			continue
		}
		best, bestFS, bestSuper = mp, fstype, super
	}
	if best != "" && strings.HasPrefix(bestFS, "fuse") {
		uid := -1
		allowOther := false
		for _, o := range strings.Split(bestSuper, ",") {
			switch {
			case o == "allow_other":
				allowOther = true
			case strings.HasPrefix(o, "user_id="):
				if n, err := strconv.Atoi(strings.TrimPrefix(o, "user_id=")); err == nil {
					uid = n
				}
			}
		}
		if !allowOther && uid >= 0 && uid != euid {
			return privateMount{MountPoint: best, UID: uid}, true
		}
		if allowOther || uid == euid {
			return privateMount{}, false
		}
	}
	if m := gvfsPath.FindStringSubmatch(path); m != nil {
		uid, _ := strconv.Atoi(m[1])
		if uid != euid {
			return privateMount{MountPoint: "/run/user/" + m[1] + "/gvfs", UID: uid}, true
		}
	}
	return privateMount{}, false
}

// parseMountinfoLine reads one /proc/self/mountinfo line: the mount point
// (field 5) and, after the " - " separator, the file system type and the
// super block options.
func parseMountinfoLine(line string) (mountPoint, fstype, superOpts string, ok bool) {
	pre, post, found := strings.Cut(line, " - ")
	if !found {
		return "", "", "", false
	}
	f := strings.Fields(pre)
	g := strings.Fields(post)
	if len(f) < 5 || len(g) < 1 {
		return "", "", "", false
	}
	mountPoint = unescapeMountinfo(f[4])
	fstype = g[0]
	if len(g) >= 3 {
		superOpts = g[2]
	}
	return mountPoint, fstype, superOpts, true
}

// unescapeMountinfo turns the octal escapes of mountinfo (\040 for a space,
// \011, \012, \134) back into their characters.
func unescapeMountinfo(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			if n, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// pathWithin reports whether path is dir or inside it.
func pathWithin(path, dir string) bool {
	if dir == "/" {
		return strings.HasPrefix(path, "/")
	}
	return path == dir || strings.HasPrefix(path, dir+"/")
}

// privateMountNote is the explanation for a folder on m, with the owner's
// name from userName (empty when it is not known). The caller's message
// names the folder.
func privateMountNote(m privateMount, userName string) string {
	who := userName
	if who == "" {
		who = fmt.Sprintf("user %d", m.UID)
	}
	return fmt.Sprintf("This folder is on a share or drive that the desktop mounted for %s only (%s). "+
		"Linux lets no other account into such a mount, root and the background service included, "+
		"and it is only there while %s is logged in. "+
		"Mount the share for the whole computer instead, as \"Backing up network shares on Linux\" in the manual describes.",
		who, m.MountPoint, who)
}
