//go:build linux

package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

// The real alarm timer: as root (the service) it is set and dropped; as a user
// the kernel refuses it, and the message says the service does it.
func TestPlatformWakeTimer(t *testing.T) {
	cancel, err := platformArmWakeTimer(time.Now().Add(time.Hour))
	if os.Geteuid() != 0 {
		if err == nil {
			cancel()
			t.Skip("this user may set wake alarms (CAP_WAKE_ALARM)")
		}
		if !strings.Contains(err.Error(), "background service") {
			t.Fatalf("as a user: %v, want it to name the background service", err)
		}
		return
	}
	if err != nil {
		t.Skipf("as root, no wake alarm on this machine: %v", err)
	}
	cancel()
}
