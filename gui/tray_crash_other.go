//go:build !windows && !service
// +build !windows,!service

package main

// attemptTrayCleanupBeforeCrash is a no-op off Windows: there is no tray icon.
func attemptTrayCleanupBeforeCrash() {}
