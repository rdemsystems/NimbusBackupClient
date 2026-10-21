//go:build linux
// +build linux

package main

import (
	"time"

	"golang.org/x/sys/unix"
)

// systemUptime is how long ago the machine booted.
func systemUptime() (time.Duration, error) {
	var info unix.Sysinfo_t
	if err := unix.Sysinfo(&info); err != nil {
		return 0, err
	}
	return time.Duration(info.Uptime) * time.Second, nil
}
