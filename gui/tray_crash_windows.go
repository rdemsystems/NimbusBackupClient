//go:build windows && !service
// +build windows,!service

package main

import (
	"time"

	"github.com/getlantern/systray"
)

// attemptTrayCleanupBeforeCrash gives a panic a short chance to remove the
// tray icon before main's recover handler exits. systray.Quit only posts a
// window message, so this cannot hang, and it does not change the exit code.
func attemptTrayCleanupBeforeCrash() {
	if !trayInitialized {
		return
	}
	systray.Quit()
	waitForTrayExit(300 * time.Millisecond)
}
