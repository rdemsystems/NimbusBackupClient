package main

import (
	"fmt"
	"sync"
	"time"
)

// Waiting for the screen on a Modern Standby computer (#142).
//
// On a computer with Modern Standby (S0 low power idle) a wake timer wakes
// the processor but never turns the screen on, and Windows puts the computer
// back into standby at once: services get a second of work every 30 seconds,
// and unless networking in standby is on, there is no network. A run started
// then waited minutes or hours for its server and failed (Sean's laptop,
// 2026-11-16: woken at 04:01, the run's server could not be reached, its
// retry ran at 06:01 and its snapshot took 38 minutes). No program can bring
// such a computer out of standby; only the person at it can.
//
// So on such a computer, with its screen off, an automatic run does not wait
// the usual two minutes for its server (network_wait.go). It checks briefly
// (standbyCheckMax): with networking in standby on, the server answers and
// the run goes on as on any computer. Otherwise the run is not started and
// not recorded as failed: the set waits until the screen comes on (someone
// uses the computer, or it is woken properly), and then runs once, with a
// line in Reports saying why it ran late. On a computer with classic sleep,
// or when the screen's state is not known, nothing changes.

// standbyCheckMax is the longest an automatic run on a Modern Standby
// computer with its screen off tries its server before it waits for the
// screen instead.
var standbyCheckMax = 30 * time.Second

// standbyWaitMax is the longest a set waits for the screen. A computer whose
// screen never comes on (a Modern Standby mini PC run without a monitor, as
// far as Windows can tell) would otherwise wait for good, silently: after
// this the run goes ahead as on any computer, and a failure is recorded and
// e-mailed as usual.
var standbyWaitMax = 12 * time.Hour

// isModernStandby reports whether this computer has Modern Standby rather
// than classic sleep. A variable for the tests.
var isModernStandby = func() bool { return readWakeTimerInfo().ModernStandby }

// standbyState is the screen's state as Windows reports it (display_state_*.go)
// and the sets waiting for the screen to come on.
var standbyState struct {
	sync.Mutex
	known   bool      // a report of the screen's state has come in
	off     bool      // the screen is off
	since   time.Time // when it last changed
	waiting map[string]standbyWait
}

// standbyWait is a set waiting for the screen: since when, and its name for
// the log.
type standbyWait struct {
	since time.Time
	set   string
	// released: the screen has come on, or the set has waited
	// standbyWaitMax (expired); the next run of the set takes it
	// (takeStandbyWait) for its line in Reports.
	released bool
	expired  bool
}

// noteDisplayState is called with each report of the screen's state. When
// the screen comes on, runs starting now wait for their server as after any
// wake, and the sets waiting for the screen are let go: the scheduler's next
// tick runs them, once.
func noteDisplayState(on bool, at time.Time) {
	standbyState.Lock()
	wasOff := standbyState.known && standbyState.off
	changed := !standbyState.known || standbyState.off == on
	standbyState.known, standbyState.off = true, !on
	if changed {
		standbyState.since = at
	}
	var released []string
	if on {
		for id, w := range standbyState.waiting {
			if !w.released {
				w.released = true
				standbyState.waiting[id] = w
				released = append(released, fmt.Sprintf("%q (waiting since %s)", w.set, clock(w.since)))
			}
		}
	}
	standbyState.Unlock()
	if on && wasOff {
		noteResume(at, "after the screen came on")
	}
	for _, r := range released {
		writeDebugLog(fmt.Sprintf("[Wake] the screen came on: the run of %s starts now", r))
	}
}

// inStandby reports whether automatic runs must not count on the network:
// a Modern Standby computer whose screen is off.
func inStandby() bool {
	standbyState.Lock()
	known, off := standbyState.known, standbyState.off
	standbyState.Unlock()
	return known && off && isModernStandby()
}

// waitingForScreen reports whether setID's run waits for the screen to come
// on: the scheduler leaves it until then, or until it has waited
// standbyWaitMax.
func waitingForScreen(setID string) bool {
	standbyState.Lock()
	w, ok := standbyState.waiting[setID]
	if !ok || w.released {
		standbyState.Unlock()
		return false
	}
	if netNow().Round(0).Sub(w.since.Round(0)) < standbyWaitMax {
		standbyState.Unlock()
		return true
	}
	w.released, w.expired = true, true
	standbyState.waiting[setID] = w
	standbyState.Unlock()
	writeDebugLog(fmt.Sprintf("[Wake] Backup Set %s has waited %d hours for the screen to come on (since %s): the run goes ahead, and its result is recorded", w.set, int(standbyWaitMax/time.Hour), clock(w.since)))
	return false
}

// waitForScreen makes setID's run wait for the screen to come on.
func waitForScreen(setID, set string, at time.Time) {
	standbyState.Lock()
	defer standbyState.Unlock()
	if standbyState.waiting == nil {
		standbyState.waiting = map[string]standbyWait{}
	}
	if w, ok := standbyState.waiting[setID]; ok && !w.released {
		return
	}
	standbyState.waiting[setID] = standbyWait{since: at, set: set}
}

// takeStandbyWait returns, once, when setID's run started waiting for the
// screen, if it did and has been let go: ok when the screen came on or the
// wait ran out, expired for the second (the run must then go ahead whatever
// the screen does).
func takeStandbyWait(setID string) (since time.Time, ok, expired bool) {
	standbyState.Lock()
	defer standbyState.Unlock()
	w, found := standbyState.waiting[setID]
	if !found || !w.released {
		return time.Time{}, false, false
	}
	delete(standbyState.waiting, setID)
	return w.since, true, w.expired
}

// standbyCheck decides, for an automatic run of job about to start, whether
// it can go on: false when the computer is in Modern Standby with its screen
// off and the run's server does not answer within standbyCheckMax. The run
// then waits for the screen (and the reason is logged once).
func (a *App) standbyCheck(job ScheduledJob, trigger string) bool {
	if trigger == "manual" || !inStandby() {
		return true
	}
	name, addr := a.jobServerAddress(job)
	if addr == "" {
		return true
	}
	reachable, _, err := waitForServerUpTo(name, addr, "in Modern Standby", standbyCheckMax)
	if reachable {
		writeDebugLog(fmt.Sprintf("[Wake] Backup Set %s: the computer is in Modern Standby with its screen off, and %s answers: the run goes on", job.Name, name))
		return true
	}
	waitForScreen(job.ID, job.Name, netNow())
	writeDebugLog(fmt.Sprintf("[Wake] Backup Set %s: the computer is in Modern Standby with its screen off and %s cannot be reached (%v): Windows does not let a program bring it out of standby, so the run waits until the screen comes on", job.Name, name, err))
	return false
}
