//go:build service && !windows

package main

// runLastInputArg: -last-input is for Windows only (logind tells on Linux).
func runLastInputArg([]string) {}
