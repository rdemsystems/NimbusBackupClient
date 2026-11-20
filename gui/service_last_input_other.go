//go:build service && !windows

package main

// runLastInputArg: -last-input is for Windows only (logind and the desktop
// tell on Linux).
func runLastInputArg([]string) {}

func isLastInputHelper([]string) bool { return false }
