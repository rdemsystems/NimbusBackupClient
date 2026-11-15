package pbscommon

import (
	"os"
	"path/filepath"
	"testing"
)

// Linux can hibernate when sys/power/state lists "disk" and sys/power/disk is
// not "[disabled]"; read from a stand-in for /sys.
func TestReadLinuxHibernate(t *testing.T) {
	for _, c := range []struct {
		name        string
		state, disk string // "" leaves the file out
		ok          bool
		why         string
	}{
		{"can", "freeze mem disk\n", "[platform] shutdown reboot suspend test_resume\n", true, ""},
		{"no disk state", "freeze mem\n", "[platform] shutdown\n", false, HibernateNone},
		{"locked down", "freeze mem disk\n", "[disabled]\n", false, HibernateOff},
		{"no disk file", "freeze mem disk\n", "", true, ""},
		{"no sysfs", "", "", false, HibernateNone},
		{"disk only as part of a word", "freeze mem diskette\n", "", false, HibernateNone},
	} {
		root := t.TempDir()
		power := filepath.Join(root, "sys", "power")
		if err := os.MkdirAll(power, 0o755); err != nil {
			t.Fatal(err)
		}
		if c.state != "" {
			if err := os.WriteFile(filepath.Join(power, "state"), []byte(c.state), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if c.disk != "" {
			if err := os.WriteFile(filepath.Join(power, "disk"), []byte(c.disk), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		ok, why := ReadLinuxHibernate(root)
		if ok != c.ok || why != c.why {
			t.Errorf("%s: %v %q, want %v %q", c.name, ok, why, c.ok, c.why)
		}
	}
}
