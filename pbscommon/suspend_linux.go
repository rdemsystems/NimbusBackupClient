//go:build linux

package pbscommon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
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
// With no resume device set it asks logind as well: systemd 255 and later,
// on a computer started through UEFI, records where the memory was saved in
// an EFI variable and reads it back at boot without resume=, and its
// CanHibernate says "yes" only when that works.
func CanHibernate() HibernateCheck {
	c := ReadLinuxHibernate("/")
	if c.Why == HibernateNoResume && logindCanHibernateWithoutResume() {
		return HibernateCheck{OK: true}
	}
	return c
}

func logindCanHibernateWithoutResume() bool {
	if _, err := os.Stat("/sys/firmware/efi"); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "systemctl", "--version").Output()
	if err != nil || ParseSystemdVersion(string(out)) < 255 {
		return false
	}
	out, err = exec.CommandContext(ctx, "busctl", "call", "org.freedesktop.login1", "/org/freedesktop/login1",
		"org.freedesktop.login1.Manager", "CanHibernate").Output()
	return err == nil && strings.TrimSpace(string(out)) == `s "yes"`
}
