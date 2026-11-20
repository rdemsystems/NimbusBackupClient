//go:build linux

package main

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// platformLastUserInput: for each active graphical session (X11, Wayland)
// logind lists, when it was last used. GNOME's own idle monitor is asked
// first, as the session's user: it counts from the last key or pointer move.
// logind's IdleHint is no help alone: GNOME sets it only when the screen
// blank delay is on, and with it off (found on a Fedora VM 2026-11-20: idle
// 12 days, IdleHint "no") the session would look in use for good. Without an
// answer from the desktop, the time logind last saw the session go idle, or
// come back from idle, is used. Text sessions (a console, ssh) are left out:
// a terminal left open must not keep the computer from sleeping after every
// backup.
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
		props, err := exec.Command("loginctl", "show-session", fields[0], "-p", "Active", "-p", "Type", "-p", "Class", "-p", "Name", "-p", "User", "-p", "IdleHint", "-p", "IdleSinceHint").Output() // #nosec G204 -- a session ID logind listed
		if err != nil {
			continue
		}
		s, ok := parseLogindSession(string(props))
		if !ok {
			continue
		}
		at, ok := s.lastUsed(desktopIdle(s), time.Now())
		if ok && (!known || at.After(latest)) {
			latest, known = at, true
		}
	}
	return latest, known
}

// logindSession is what loginctl show-session says of an active graphical
// user session.
type logindSession struct {
	name      string
	uid       int
	idleHint  bool
	idleSince time.Time // when IdleHint last changed; zero when never
}

// parseLogindSession reads loginctl show-session's Key=Value lines; ok only
// for an active graphical user session.
func parseLogindSession(props string) (logindSession, bool) {
	p := map[string]string{}
	for _, l := range strings.Split(props, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(l), "="); ok {
			p[k] = v
		}
	}
	if p["Active"] != "yes" || p["Class"] != "user" || (p["Type"] != "x11" && p["Type"] != "wayland" && p["Type"] != "mir") {
		return logindSession{}, false
	}
	s := logindSession{name: p["Name"], idleHint: p["IdleHint"] == "yes"}
	s.uid, _ = strconv.Atoi(p["User"])
	if us, err := strconv.ParseInt(p["IdleSinceHint"], 10, 64); err == nil && us > 0 {
		s.idleSince = time.UnixMicro(us)
	}
	return s, true
}

// lastUsed is when the session was last used: from the desktop's idle time
// (idle >= 0) when it answered, else from logind's last change of IdleHint.
func (s logindSession) lastUsed(idle time.Duration, now time.Time) (time.Time, bool) {
	if idle >= 0 {
		return now.Add(-idle), true
	}
	if !s.idleSince.IsZero() {
		return s.idleSince, true
	}
	return time.Time{}, false
}

var mutterIdle = regexp.MustCompile(`uint64 (\d+)`)

// desktopIdle asks GNOME's idle monitor how long the session has been idle,
// as its user, on its session bus; -1 when it does not answer (not GNOME, no
// gdbus).
func desktopIdle(s logindSession) time.Duration {
	if s.uid <= 0 || s.name == "" {
		return -1
	}
	bus := "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/" + strconv.Itoa(s.uid) + "/bus"
	args := []string{"gdbus", "call", "--session", "--dest", "org.gnome.Mutter.IdleMonitor",
		"--object-path", "/org/gnome/Mutter/IdleMonitor/Core", "--method", "org.gnome.Mutter.IdleMonitor.GetIdletime"}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	switch {
	case os.Getuid() == 0:
		cmd = exec.CommandContext(ctx, "runuser", append([]string{"-u", s.name, "--", "env", bus}, args...)...) // #nosec G204 -- a user logind listed, fixed arguments
	case os.Getuid() == s.uid:
		cmd = exec.CommandContext(ctx, "env", append([]string{bus}, args...)...) // #nosec G204 -- fixed arguments
	default:
		return -1 // another user's session: only root may ask it
	}
	out, err := cmd.Output()
	if err != nil {
		return -1
	}
	return parseMutterIdle(string(out))
}

// parseMutterIdle reads gdbus's "(uint64 1057815653,)": milliseconds idle.
func parseMutterIdle(out string) time.Duration {
	m := mutterIdle.FindStringSubmatch(out)
	if m == nil {
		return -1
	}
	ms, err := strconv.ParseUint(m[1], 10, 63)
	if err != nil {
		return -1
	}
	return time.Duration(ms) * time.Millisecond
}
