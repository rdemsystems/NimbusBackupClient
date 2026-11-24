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
// usually may too. A wake alarm for the next backup still wakes it. Without
// systemd: through elogind (loginctl suspend), as Devuan and Slackware have
// it, else the kernel's own /sys/power/state.
func SleepComputer() error {
	return suspendTo("suspend", "mem")
}

// HibernateComputer hibernates the computer through systemd (systemctl
// hibernate), elogind or /sys/power/state, as SleepComputer. It needs a swap
// area large enough to hold the memory.
func HibernateComputer() error {
	return suspendTo("hibernate", "disk")
}

// powerStateFile is the kernel's switch for sleep and hibernate. A variable
// for the tests.
var powerStateFile = "/sys/power/state"

// systemdBooted reports whether systemd started this computer. A variable
// for the tests.
var systemdBooted = func() bool {
	fi, err := os.Stat("/run/systemd/system")
	return err == nil && fi.IsDir()
}

// suspendTo puts the computer to sleep (verb suspend, kernel state mem) or
// into hibernate (hibernate, disk) with whatever this computer has.
func suspendTo(verb, kernelState string) error {
	if systemdBooted() {
		return runSystemctl(exec.Command(SystemTool("systemctl"), verb))
	}
	if loginctl := SystemTool("loginctl"); loginctl != "loginctl" {
		if err := runSystemctl(exec.Command(loginctl, verb)); err == nil {
			return nil
		}
	}
	// The kernel's own switch, root only: what is written there must be one
	// the kernel offers, as /sys/power/state lists them.
	offered, err := os.ReadFile(powerStateFile)
	if err != nil {
		return fmt.Errorf("this computer has neither systemd nor elogind, and %s cannot be read: %v", powerStateFile, err)
	}
	if !strings.Contains(" "+strings.TrimSpace(string(offered))+" ", " "+kernelState+" ") {
		return fmt.Errorf("this computer's kernel does not offer %q (it offers %q)", kernelState, strings.TrimSpace(string(offered)))
	}
	if err := os.WriteFile(powerStateFile, []byte(kernelState), 0o644); err != nil {
		return fmt.Errorf("writing %q to %s: %v", kernelState, powerStateFile, err)
	}
	return nil
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
