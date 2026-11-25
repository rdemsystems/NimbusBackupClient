package pbscommon

import (
	"strings"
	"testing"
)

// Lines as /proc/self/mountinfo has them on linuxclient (Mint 21.3) with a
// desktop SFTP share open, plus an sshfs mount made with allow_other and a
// mount point with a space.
const testMountinfo = `22 1 252:0 / / rw,relatime shared:1 - ext4 /dev/vda1 rw,errors=remount-ro
26 22 0:24 / /run rw,nosuid,nodev,noexec,relatime shared:5 - tmpfs tmpfs rw,size=401812k,mode=755
512 26 0:47 / /run/user/1000 rw,nosuid,nodev,relatime shared:280 - tmpfs tmpfs rw,size=401808k,nr_inodes=100452,mode=700,uid=1000,gid=1000
530 512 0:49 / /run/user/1000/gvfs rw,nosuid,nodev,relatime shared:290 - fuse.gvfsd-fuse gvfsd-fuse rw,user_id=1000,group_id=1000
540 22 0:51 / /mnt/shared\040files rw,relatime shared:300 - fuse.sshfs nas:/data rw,user_id=1000,group_id=1000,allow_other
550 22 0:52 / /home/alice/remote rw,relatime shared:310 - fuse.sshfs nas:/alice rw,user_id=1001,group_id=1001
`

func TestFindPrivateMount(t *testing.T) {
	cases := []struct {
		name   string
		path   string
		euid   int
		want   bool
		mount  string
		ownerU int
	}{
		{"desktop share, read by root", "/run/user/1000/gvfs/sftp:host=nas,user=a/home/a/docs", 0, true, "/run/user/1000/gvfs", 1000},
		{"desktop share, read by its user", "/run/user/1000/gvfs/smb-share:server=nas,share=data", 1000, false, "", 0},
		{"gvfs path while its user is logged out", "/run/user/1002/gvfs/smb-share:server=nas,share=data/x", 0, true, "/run/user/1002/gvfs", 1002},
		{"sshfs with allow_other", "/mnt/shared files/reports", 0, false, "", 0},
		{"sshfs without allow_other, another user's", "/home/alice/remote/projects", 0, true, "/home/alice/remote", 1001},
		{"ordinary folder", "/home/mickbeeby/Documents", 0, false, "", 0},
		{"the gvfs folder itself", "/run/user/1000/gvfs", 0, true, "/run/user/1000/gvfs", 1000},
		{"a name that only starts like gvfs", "/run/user/1000/gvfsx", 0, false, "", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, ok := findPrivateMount(testMountinfo, c.path, c.euid)
			if ok != c.want {
				t.Fatalf("private = %v, want %v (%+v)", ok, c.want, m)
			}
			if ok && (m.MountPoint != c.mount || m.UID != c.ownerU) {
				t.Fatalf("mount = %+v, want %s owned by %d", m, c.mount, c.ownerU)
			}
		})
	}
}

func TestPrivateMountNoteText(t *testing.T) {
	note := privateMountNote(privateMount{MountPoint: "/run/user/1000/gvfs", UID: 1000}, "mick")
	for _, want := range []string{"for mick only (/run/user/1000/gvfs)", "root and the background service", "while mick is logged in", "Backing up network shares on Linux"} {
		if !strings.Contains(note, want) {
			t.Errorf("note lacks %q: %s", want, note)
		}
	}
	if anon := privateMountNote(privateMount{MountPoint: "/m", UID: 1234}, ""); !strings.Contains(anon, "for user 1234 only") {
		t.Errorf("unknown owner: %s", anon)
	}
}

func TestUnescapeMountinfo(t *testing.T) {
	for in, want := range map[string]string{
		`/mnt/a\040b`:    "/mnt/a b",
		`/plain`:         "/plain",
		`/tab\011x\134y`: "/tab\tx\\y",
		`/trailing\04`:   `/trailing\04`,
		`/not\999octal`:  `/not\999octal`,
	} {
		if got := unescapeMountinfo(in); got != want {
			t.Errorf("unescape(%q) = %q, want %q", in, got, want)
		}
	}
}
