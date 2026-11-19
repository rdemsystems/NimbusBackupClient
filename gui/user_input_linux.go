//go:build linux

package main

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// platformLastUserInput asks logind about each active graphical session
// (X11, Wayland): one not idle is in use now; an idle one was last used when
// it went idle. Text sessions (a console, ssh) do not say when they are idle,
// so they are left out: a terminal left open must not keep the computer
// from sleeping after every backup.
func platformLastUserInput() (time.Time, bool) {
	out, err := exec.Command("loginctl", "list-sessions", "--no-legend").Output() // #nosec G204 -- fixed arguments
	if err != nil {
		return time.Time{}, false
	}
	var latest time.Time
	known := false
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		props, err := exec.Command("loginctl", "show-session", fields[0], "-p", "Active", "-p", "Type", "-p", "Class", "-p", "IdleHint", "-p", "IdleSinceHint").Output() // #nosec G204 -- a session ID logind listed
		if err != nil {
			continue
		}
		if at, ok := sessionLastUsed(string(props), time.Now()); ok && (!known || at.After(latest)) {
			latest, known = at, true
		}
	}
	return latest, known
}

// sessionLastUsed reads loginctl show-session's Key=Value lines: when an
// active graphical user session was last used (now when it is not idle).
func sessionLastUsed(props string, now time.Time) (time.Time, bool) {
	p := map[string]string{}
	for _, l := range strings.Split(props, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(l), "="); ok {
			p[k] = v
		}
	}
	if p["Active"] != "yes" || p["Class"] != "user" || (p["Type"] != "x11" && p["Type"] != "wayland" && p["Type"] != "mir") {
		return time.Time{}, false
	}
	if p["IdleHint"] != "yes" {
		return now, true
	}
	us, err := strconv.ParseInt(p["IdleSinceHint"], 10, 64)
	if err != nil || us <= 0 {
		return time.Time{}, false
	}
	return time.UnixMicro(us), true
}
