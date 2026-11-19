//go:build !windows

package main

// platformLatestWakeEvent: Linux and the others keep no record of what woke
// the computer that a program can read for every machine, so a start in the
// wake timer's window is taken as the timer's.
func platformLatestWakeEvent() (wakeEvent, error) { return wakeEvent{}, errNoWakeRecord }
