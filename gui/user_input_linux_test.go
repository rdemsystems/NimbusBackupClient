//go:build linux

package main

import (
	"strconv"
	"testing"
	"time"
)

func TestSessionLastUsedFromLogind(t *testing.T) {
	now := time.Date(2026, 11, 19, 23, 10, 0, 0, time.UTC)
	idleSince := time.Date(2026, 11, 19, 22, 40, 0, 0, time.UTC)
	for _, c := range []struct {
		what  string
		props string
		at    time.Time
		known bool
	}{
		{"a desktop in use", "Active=yes\nType=wayland\nClass=user\nIdleHint=no\nIdleSinceHint=0\n", now, true},
		{"a desktop gone idle", "Active=yes\nType=x11\nClass=user\nIdleHint=yes\nIdleSinceHint=" + itoa(idleSince.UnixMicro()) + "\n", idleSince, true},
		{"an ssh session", "Active=yes\nType=tty\nClass=user\nIdleHint=no\n", time.Time{}, false},
		{"the login screen", "Active=yes\nType=wayland\nClass=greeter\nIdleHint=no\n", time.Time{}, false},
		{"a desktop switched away from", "Active=no\nType=wayland\nClass=user\nIdleHint=no\n", time.Time{}, false},
		{"idle with no time", "Active=yes\nType=x11\nClass=user\nIdleHint=yes\nIdleSinceHint=0\n", time.Time{}, false},
	} {
		at, known := sessionLastUsed(c.props, now)
		if known != c.known || !at.Equal(c.at) {
			t.Errorf("%s: %s %v, want %s %v", c.what, at, known, c.at, c.known)
		}
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
