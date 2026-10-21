//go:build !windows && !linux
// +build !windows,!linux

package main

import (
	"errors"
	"time"
)

// systemUptime is how long ago the machine booted; not known here.
func systemUptime() (time.Duration, error) {
	return 0, errors.New("system uptime not available on this platform")
}
