//go:build windows

package main

import (
	"os"
	"testing"
)

// What this Windows reports: whether a timer can wake it, and the power
// plan's Allow wake timers. WAKE_EXPECT=yes|no checks the answer for the
// machine it runs on (a VM without sleep states: no).
func TestWindowsWakeCapabilities(t *testing.T) {
	can, modern, ok := canWakeByTimer()
	if !ok {
		t.Fatal("GetPwrCapabilities could not be read")
	}
	info := platformWakeTimerInfo()
	t.Logf("WAKE can wake by timer: %v; Modern Standby only: %v; info %+v", can, modern, info)
	if info.ModernStandby != modern {
		t.Errorf("WakeTimerInfo.ModernStandby %v, capabilities say %v", info.ModernStandby, modern)
	}
	switch os.Getenv("WAKE_EXPECT") {
	case "yes":
		if !can || !info.Supported {
			t.Errorf("this machine should be wakeable by a timer")
		}
	case "no":
		if can || info.Supported {
			t.Errorf("this machine has no sleep state a timer could wake it from")
		}
	}
}
