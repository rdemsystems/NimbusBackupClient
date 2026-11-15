package pbscommon

import (
	"os"
	"path/filepath"
	"strings"
)

// Putting the computer to sleep or into hibernation once a backup has run (a
// Backup Set's "When the backup finishes, the computer", the command line
// tools' -then), beside ShutdownComputer. SleepComputer, HibernateComputer and
// CanHibernate are in suspend_windows.go, suspend_linux.go and
// suspend_other.go.

// Why a computer cannot hibernate, as CanHibernate says it.
const (
	// HibernateOff: the computer has sleep states, but hibernation is turned
	// off (Windows: powercfg /hibernate off; Linux: locked, as under Secure
	// Boot's lockdown).
	HibernateOff = "off"
	// HibernateNone: the computer has no hibernate state at all (a VM, or
	// firmware that does not offer one).
	HibernateNone = "none"
)

// ReadLinuxHibernate says whether Linux can hibernate, from the files under
// root (the real system's are under "/"): sys/power/state lists "disk" when
// the kernel can hibernate, and sys/power/disk reads "[disabled]" when it may
// not (the kernel's lockdown). why is "" when it can, else HibernateNone or
// HibernateOff. Whether the swap is large enough is not known until it tries.
func ReadLinuxHibernate(root string) (ok bool, why string) {
	state, err := os.ReadFile(filepath.Join(root, "sys", "power", "state")) // #nosec G304 -- a fixed file under root
	if err != nil {
		return false, HibernateNone
	}
	disk, _ := os.ReadFile(filepath.Join(root, "sys", "power", "disk")) // #nosec G304 -- a fixed file under root
	return ParseLinuxHibernate(string(state), string(disk))
}

// ParseLinuxHibernate is ReadLinuxHibernate on the two files' text.
func ParseLinuxHibernate(state, disk string) (ok bool, why string) {
	found := false
	for _, s := range strings.Fields(state) {
		if s == "disk" {
			found = true
		}
	}
	if !found {
		return false, HibernateNone
	}
	if strings.Contains(disk, "[disabled]") {
		return false, HibernateOff
	}
	return true, ""
}
