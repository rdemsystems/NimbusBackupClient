package main

import (
	"fmt"
	"sync"
	"time"
)

// Keeping the computer awake from a wake until its run has started (#142).
//
// A computer woken by a program's timer, with no one at it, goes back to
// sleep on its own after a short while: Windows' "System unattended sleep
// timeout" is 2 minutes by default. The wake timer goes off wakeLead (2
// minutes) before the run, and the run may then wait for its server
// (network_wait.go), so the computer could fall asleep again before the run
// began or while it waited, and the run went on only when something next woke
// it. Found on a Modern Standby laptop 2026-11-17: woken at 16:01 for a 16:00
// run, the 2 minute wait for the server ended at 17:35, and the 03:00 runs
// started half an hour late.
//
// So when the scheduler finds the computer woken by the timer for a set, it
// keeps it awake (preventSleep) until that set's run takes over with its own
// hold (executeScheduledJob), or wakeHoldMax has gone by, whichever is first.
// The time limit is there for a run that never comes (the set was edited or
// turned off in the meantime): the computer must not be kept awake for good.

// wakeHoldMax is the longest the computer is kept awake for a woken run that
// has not started: the lead to the run, the longest wait for its server and a
// few minutes more for the scheduler's ticks.
var wakeHoldMax = wakeLead + networkWaitMax + 3*time.Minute

// holdAwakeAfterWake keeps the computer awake for setID's woken run. A
// variable so the tests that wake a fake computer do not take the real
// system's request (TestMain sets it to do nothing).
var holdAwakeAfterWake = holdAwakeForWokenRun

var wakeHold struct {
	mu    sync.Mutex
	setID string
	end   func()
	timer *time.Timer
	gen   int
}

// holdAwakeForWokenRun keeps the computer awake for set's run, which the
// computer was woken for, until releaseWakeHold(setID) or wakeHoldMax.
func holdAwakeForWokenRun(setID, set string) {
	end, err := preventSleep(product().Title + ": woken for a backup")
	wakeHold.mu.Lock()
	defer wakeHold.mu.Unlock()
	releaseWakeHoldLocked()
	wakeHold.gen++
	gen := wakeHold.gen
	wakeHold.setID, wakeHold.end = setID, end
	wakeHold.timer = time.AfterFunc(wakeHoldMax, func() {
		wakeHold.mu.Lock()
		defer wakeHold.mu.Unlock()
		if wakeHold.gen == gen && wakeHold.end != nil {
			writeDebugLog(fmt.Sprintf("[Wake] the run of %q did not start within %s of the wake: the computer may sleep again", set, wakeHoldMax))
			releaseWakeHoldLocked()
		}
	})
	if err != nil {
		writeDebugLog(fmt.Sprintf("[Wake] %v: the computer may go back to sleep before the run of %q starts", err, set))
		return
	}
	writeDebugLog(fmt.Sprintf("[Wake] keeping the computer awake until the run of %q starts", set))
}

// releaseWakeHold ends the hold taken for setID's woken run, if that is the
// one held: its run has its own hold now.
func releaseWakeHold(setID string) {
	wakeHold.mu.Lock()
	defer wakeHold.mu.Unlock()
	if wakeHold.end != nil && wakeHold.setID == setID {
		releaseWakeHoldLocked()
	}
}

func releaseWakeHoldLocked() {
	if wakeHold.timer != nil {
		wakeHold.timer.Stop()
	}
	if wakeHold.end != nil {
		wakeHold.end()
	}
	wakeHold.setID, wakeHold.end, wakeHold.timer = "", nil, nil
}
