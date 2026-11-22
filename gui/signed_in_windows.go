//go:build windows

package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// platformSignedInSessions lists the sessions someone is signed in at and
// using now (WTSActive): a desktop at the screen or over Remote Desktop. A
// session that is only disconnected is left out, so connecting to it again
// counts as signing in. Session 0 (services) is never one.
func platformSignedInSessions() (map[string]bool, bool) {
	var sessions *windows.WTS_SESSION_INFO
	var count uint32
	if err := windows.WTSEnumerateSessions(0, 0, 1, &sessions, &count); err != nil {
		writeDebugLog(fmt.Sprintf("[Trigger] could not list the signed-in sessions: %v", err))
		return nil, false
	}
	defer windows.WTSFreeMemory(uintptr(unsafe.Pointer(sessions)))
	out := map[string]bool{}
	for _, s := range unsafe.Slice(sessions, count) {
		if s.SessionID == 0 || s.State != wtsActive {
			continue
		}
		out[fmt.Sprint(s.SessionID)] = true
	}
	return out, true
}
