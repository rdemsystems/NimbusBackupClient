package main

import "time"

// Whether someone has used the computer since a moment: what keeps the
// computer as it is after a backup when someone is at it (after_backup.go,
// AfterBackupEvenIfInUse). The service cannot see the keyboard and mouse of
// a signed-in user: on Windows it runs in session 0, and the last input time
// Windows reports for other sessions is filled in only for Remote Desktop
// (found on a laptop 2026-11-19: 0 for the console session). So the service
// runs its own program for a moment in each signed-in session with
// -last-input, which asks there and prints how long ago the last input was
// (user_input_windows.go, service_last_input_windows.go); on Linux logind
// says when a graphical session went idle (user_input_linux.go).

// lastUserInput is the latest keyboard or mouse use of anyone signed in;
// known is false when it cannot be told (nobody signed in, or no way to ask).
// Tests replace it.
var lastUserInput = platformLastUserInput

// usedSince reports whether someone used the computer after since, and when.
func usedSince(since time.Time) (time.Time, bool) {
	at, known := lastUserInput()
	if !known || since.IsZero() || !at.After(since) {
		return at, false
	}
	return at, true
}
