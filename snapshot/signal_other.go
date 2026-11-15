//go:build !linux

package snapshot

import "os"

// CleanupOnSignal does nothing here: a Windows shadow copy of a backup
// context goes with the process that made it, and other systems take no
// snapshots.
func CleanupOnSignal() {}

// ExitWithCleanup ends the process with code; for the same reason there is
// nothing to remove first.
func ExitWithCleanup(code int) { os.Exit(code) }
