//go:build windows

package main

import (
	"testing"
	"time"
)

// Windows takes the request for the screen's state and reports the current
// state at once, through the callback (on any Windows computer, Modern
// Standby or not: only startDisplayWatch is limited to Modern Standby).
func TestDisplayWatchReportsTheScreen(t *testing.T) {
	standbyTest(t, false)
	if err := registerDisplayWatch(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		standbyState.Lock()
		known, off := standbyState.known, standbyState.off
		standbyState.Unlock()
		if known {
			t.Logf("Windows reports the screen %s", map[bool]string{true: "off", false: "on"}[off])
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("Windows did not report the screen's state within 10 s")
}
