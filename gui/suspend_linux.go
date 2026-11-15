//go:build linux

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// platformSleepComputer suspends the computer through systemd (systemctl
// suspend), which the background service may do as root. The wake alarm for
// the set's next run (wake_linux.go) still wakes it.
func platformSleepComputer() error {
	if out, err := exec.Command("systemctl", "suspend").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl suspend: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
