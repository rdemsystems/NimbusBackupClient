// +build windows

package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	trayInitialized = false
	menuShow        *systray.MenuItem
	menuQuit        *systray.MenuItem

	// trayExitedCh is closed once systray's message loop has removed the icon
	// (see onExit). Code about to os.Exit waits on it, with its own bound and
	// its own exit code.
	trayExitedCh   = make(chan struct{})
	trayExitedOnce sync.Once
)

// markTrayExited closes trayExitedCh exactly once. Called from onExit.
func markTrayExited() {
	trayExitedOnce.Do(func() { close(trayExitedCh) })
}

// waitForTrayExit blocks until the tray icon is removed or timeout elapses.
func waitForTrayExit(timeout time.Duration) {
	select {
	case <-trayExitedCh:
	case <-time.After(timeout):
	}
}

// preventCloseToTray reports whether closing the main window should be
// intercepted and turned into a hide-to-tray. On Windows the tray keeps the
// app alive, so we swallow the close.
func (a *App) preventCloseToTray() bool {
	return true
}

// SetupSystemTray initializes the system tray icon and menu
func (a *App) SetupSystemTray() {
	if trayInitialized {
		return
	}

	writeDebugLog("Setting up system tray")

	// Setup tray in goroutine to avoid blocking
	go func() {
		systray.Run(onReady(a), onExit)
	}()

	trayInitialized = true
}

func onReady(a *App) func() {
	return func() {
		// Set tray icon from embedded PNG data (icon.go)
		systray.SetIcon(TrayIconData)
		systray.SetTitle("Proxmox Backup Client")
		systray.SetTooltip("Proxmox Backup Client - Scheduled backups active")

		// Add menu items
		menuShow = systray.AddMenuItem("🖥️ Show window", "Open the Proxmox Backup Client interface")
		systray.AddSeparator()

		menuStatus := systray.AddMenuItem("📊 Backup status", "See scheduled backup status")
		menuStatus.Disable() // For display only

		systray.AddSeparator()
		menuQuit = systray.AddMenuItem("❌ Quit", "Close Proxmox Backup Client")

		// Handle menu item clicks
		go func() {
			for {
				select {
				case <-menuShow.ClickedCh:
					writeDebugLog("Tray: Show window clicked")
					// Show the main window
					runtime.WindowShow(a.ctx)
					runtime.WindowUnminimise(a.ctx)
				case <-menuQuit.ClickedCh:
					writeDebugLog("Tray: Quit clicked")
					// Ask systray to remove the icon and Wails to shut down.
					systray.Quit()
					runtime.Quit(a.ctx)
					// Exit once the icon is confirmed gone (bounded in case the
					// message loop is stuck). The old blind 2 s timer could exit
					// before Shell_NotifyIcon(NIM_DELETE) ran, leaving a ghost icon.
					go func() {
						waitForTrayExit(3 * time.Second)
						writeDebugLog("Tray: exiting after Quit")
						os.Exit(0)
					}()
				}
			}
		}()

		writeDebugLog("System tray initialized")
	}
}

// onExit is called by systray's message loop only after it has removed the
// icon with Shell_NotifyIcon(NIM_DELETE). systray never sets the icon's GUID,
// so Windows cannot clean it up after the process is gone: an os.Exit that
// does not wait for this leaves a ghost icon until it is hovered or Explorer
// restarts.
//
// Adapted from CJInvent/NimbusBackupClient 9a7097e.
func onExit() {
	writeDebugLog("System tray exiting (icon removed)")
	markTrayExited()
}

// MinimizeToTray hides the window and minimizes to tray
func (a *App) MinimizeToTray() {
	writeDebugLog("Minimizing to tray")
	runtime.WindowHide(a.ctx)
}

// ShowFromTray shows the window from tray
func (a *App) ShowFromTray() {
	writeDebugLog("Showing from tray")
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
}

// UpdateTrayTooltip updates the tray icon tooltip (e.g., with next backup time)
func (a *App) UpdateTrayTooltip(message string) {
	if !trayInitialized {
		return
	}
	systray.SetTooltip(fmt.Sprintf("Proxmox Backup Client - %s", message))
}
