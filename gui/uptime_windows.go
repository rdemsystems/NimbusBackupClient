//go:build windows
// +build windows

package main

import (
	"time"

	"golang.org/x/sys/windows"
)

// systemUptime is how long ago the machine booted.
func systemUptime() (time.Duration, error) {
	return windows.DurationSinceBoot(), nil
}
