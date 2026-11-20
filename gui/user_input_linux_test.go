//go:build linux

package main

import (
	"strconv"
	"testing"
	"time"
)

func TestLogindSessionsThatCount(t *testing.T) {
	for _, c := range []struct {
		what  string
		props string
		ok    bool
	}{
		{"a Wayland desktop", "Active=yes\nType=wayland\nClass=user\nName=mick\nUser=1000\nIdleHint=no\nIdleSinceHint=0\n", true},
		{"an X11 desktop", "Active=yes\nType=x11\nClass=user\nName=mick\nUser=1000\n", true},
		{"an ssh session", "Active=yes\nType=tty\nClass=user\nName=mick\nUser=1000\n", false},
		{"the login screen", "Active=yes\nType=wayland\nClass=greeter\nName=gdm\nUser=42\n", false},
		{"systemd's manager", "Active=yes\nType=unspecified\nClass=manager\nName=mick\nUser=1000\n", false},
		{"a desktop switched away from", "Active=no\nType=wayland\nClass=user\nName=mick\nUser=1000\n", false},
	} {
		s, ok := parseLogindSession(c.props)
		if ok != c.ok {
			t.Errorf("%s: counted %v, want %v", c.what, ok, c.ok)
		}
		if ok && (s.name != "mick" || s.uid != 1000) {
			t.Errorf("%s: read as %+v", c.what, s)
		}
	}
}

// Found on a Fedora VM 2026-11-20: a GNOME desktop untouched for 12 days had
// IdleHint "no" (its screen blank delay is off), and GNOME's idle monitor
// said 1057815653 ms. The desktop's answer comes first; without one, logind's
// last change of IdleHint.
func TestLastUsedOfALinuxDesktop(t *testing.T) {
	now := time.Date(2026, 11, 20, 0, 38, 0, 0, time.UTC)
	changed := time.Date(2026, 11, 7, 18, 34, 0, 0, time.UTC)
	props := "Active=yes\nType=wayland\nClass=user\nName=mick\nUser=1000\nIdleHint=no\nIdleSinceHint=" + strconv.FormatInt(changed.UnixMicro(), 10) + "\n"
	s, _ := parseLogindSession(props)

	idle := parseMutterIdle("(uint64 1057815653,)\n")
	if idle != 1057815653*time.Millisecond {
		t.Fatalf("GNOME's idle time read as %s", idle)
	}
	if at, ok := s.lastUsed(idle, now); !ok || !at.Equal(now.Add(-idle)) {
		t.Errorf("with GNOME's answer: %s %v", at, ok)
	}
	if at, ok := s.lastUsed(-1, now); !ok || !at.Equal(changed) {
		t.Errorf("without it, logind's last change: %s %v, want %s", at, ok, changed)
	}
	s.idleSince = time.Time{}
	if _, ok := s.lastUsed(-1, now); ok {
		t.Error("a session with nothing to go by was taken as known")
	}
	for _, bad := range []string{"", "Error: GDBus.Error:org.freedesktop.DBus.Error.ServiceUnknown", "(uint32 5,)"} {
		if parseMutterIdle(bad) >= 0 {
			t.Errorf("%q read as an idle time", bad)
		}
	}
}
