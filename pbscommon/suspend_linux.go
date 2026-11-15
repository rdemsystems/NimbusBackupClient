//go:build linux

package pbscommon

import (
	"fmt"
	"os/exec"
	"strings"
)

// SleepComputer suspends the computer through systemd (systemctl suspend),
// which the background service may do as root and the user at the computer
// usually may too. A wake alarm for the next backup still wakes it.
func SleepComputer() error {
	return runSystemctl(exec.Command("systemctl", "suspend"))
}

// HibernateComputer hibernates the computer through systemd (systemctl
// hibernate). It needs a swap area large enough to hold the memory.
func HibernateComputer() error {
	return runSystemctl(exec.Command("systemctl", "hibernate"))
}

func runSystemctl(cmd *exec.Cmd) error {
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %v: %s", strings.Join(cmd.Args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// CanHibernate says whether this computer can hibernate (ReadLinuxHibernate).
func CanHibernate() (ok bool, why string) {
	return ReadLinuxHibernate("/")
}
