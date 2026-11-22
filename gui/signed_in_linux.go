//go:build linux

package main

import (
	"os/exec"
	"strings"
)

// platformSignedInSessions lists the user sessions logind has at a screen of
// the computer (graphical, or a text console), active or not. Sessions over
// the network (ssh) are left out: someone checking a machine remotely is not
// someone sitting down to use it.
func platformSignedInSessions() (map[string]bool, bool) {
	out, err := exec.Command("loginctl", "list-sessions", "--no-legend").Output() // #nosec G204 -- fixed arguments
	if err != nil {
		return nil, false
	}
	sessions := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		props, err := exec.Command("loginctl", "show-session", fields[0], "-p", "Class", "-p", "Type", "-p", "Remote").Output() // #nosec G204 -- a session ID logind listed
		if err != nil {
			continue
		}
		if localUserSession(string(props)) {
			sessions[fields[0]] = true
		}
	}
	return sessions, true
}

// localUserSession reads loginctl show-session's Key=Value lines: a user's
// session at the computer itself.
func localUserSession(props string) bool {
	p := map[string]string{}
	for _, l := range strings.Split(props, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(l), "="); ok {
			p[k] = v
		}
	}
	if p["Class"] != "user" || p["Remote"] == "yes" {
		return false
	}
	switch p["Type"] {
	case "x11", "wayland", "mir", "tty":
		return true
	}
	return false
}
