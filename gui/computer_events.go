package main

import (
	"fmt"
	"sync"
	"time"
)

// A Backup Set's "When something happens on this computer" trigger (#157):
// the set runs when the computer starts, wakes from sleep or hibernate,
// someone signs in, or nobody has used the keyboard or mouse for N minutes.
//
// Sets saved before it existed have RunAtStartup only: normalizeJobSchedules
// moves that into ComputerEvents.Starts, and from then on writes RunAtStartup
// back from it (mirrorComputerEvents), so an older app or service reading the
// file still runs the set when the computer starts. RunAtStartup is never
// read again once ComputerEvents exists.
//
// Runs these events start are like scheduled ones: the set's On battery rule
// applies (a run it would hold back is simply not started) and a set runs once
// at a time (runningJobs). eventMinGap stops two events close together (the
// computer starts, then someone signs in) starting two backups.

// History Trigger values of the runs these events start ("startup" is older).
const (
	triggerWake   = "wake"
	triggerSignIn = "signin"
	triggerIdle   = "idle"
)

// ComputerEvents is a set's "When something happens on this computer" trigger.
type ComputerEvents struct {
	Enabled bool `json:"enabled"`
	// Starts: when the computer starts (the service, or the app scheduling on
	// its own, starts with it).
	Starts bool `json:"starts,omitempty"`
	// Wakes: when the computer wakes from sleep or hibernate. A wake this
	// app's own wake timer made for one of the set's times runs that time's
	// run instead (the schedule), not a second one.
	Wakes bool `json:"wakes,omitempty"`
	// SignIn: when someone signs in (a new signed-in session appears).
	SignIn bool `json:"signIn,omitempty"`
	// IdleMinutes over 0: when nobody has used the keyboard or mouse for that
	// many minutes; once each time the computer goes idle.
	IdleMinutes int `json:"idleMinutes,omitempty"`
}

const (
	maxIdleMinutes = 24 * 60
	// eventMinGap: after one of these events started a run of a set, no
	// other event starts one for this long.
	eventMinGap = 15 * time.Minute
)

// computerEventsOn reports whether job's events trigger may start runs.
func computerEventsOn(job ScheduledJob) bool {
	return job.Enabled && job.ScheduleMode != ScheduleModeManual && job.ComputerEvents != nil && job.ComputerEvents.Enabled
}

// validateComputerEvents checks a set's events trigger before it is saved.
func validateComputerEvents(e *ComputerEvents) error {
	if e == nil {
		return nil
	}
	if e.IdleMinutes < 0 || e.IdleMinutes > maxIdleMinutes {
		return fmt.Errorf("when something happens on this computer: idle for 1 to %d minutes", maxIdleMinutes)
	}
	return nil
}

// migrateComputerEvents moves a set saved before ComputerEvents existed to it
// (RunAtStartup becomes Starts). A set that has ComputerEvents keeps it.
func migrateComputerEvents(job *ScheduledJob) {
	if job.ComputerEvents == nil && job.RunAtStartup {
		job.ComputerEvents = &ComputerEvents{Enabled: true, Starts: true}
	}
}

// mirrorComputerEvents writes RunAtStartup from ComputerEvents, for an older
// app or service reading the file.
func mirrorComputerEvents(job *ScheduledJob) {
	job.RunAtStartup = job.ScheduleMode != ScheduleModeManual && job.ComputerEvents != nil &&
		job.ComputerEvents.Enabled && job.ComputerEvents.Starts
}

// ---- per schedule wake

// scheduleWakes reports whether a run of s wakes the computer. A schedule
// saved before each had its own choice (Wake nil) follows the set's old
// WakeToRun; normalizeJobSchedules gives every schedule its own.
func scheduleWakes(job ScheduledJob, s Schedule) bool {
	if s.Wake != nil {
		return *s.Wake
	}
	return job.WakeToRun
}

// migrateScheduleWake gives every schedule its own wake choice, copied from
// the set's old WakeToRun, and never changes one a schedule already has.
// WakeToRun is then written as "on if any schedule wakes", so an older app
// reading the file still wakes the computer for the set.
func migrateScheduleWake(job *ScheduledJob) {
	wakes := false
	for i := range job.Schedules {
		if job.Schedules[i].Wake == nil {
			v := job.WakeToRun
			job.Schedules[i].Wake = &v
		}
		if *job.Schedules[i].Wake {
			wakes = true
		}
	}
	if len(job.Schedules) > 0 || job.ScheduleMode == ScheduleModeManual {
		job.WakeToRun = wakes && job.ScheduleMode != ScheduleModeManual
	}
}

// ---- watching for the events

// eventState is what the scheduler has seen of the events between ticks.
var eventState = struct {
	sync.Mutex
	// sessions signed in at the last look; nil before the first look.
	sessions map[string]bool
	// idleFired: per set, when its idle trigger last started a run; it starts
	// none again until someone has used the computer since.
	idleFired map[string]time.Time
	// started: per set, when an event last started a run (eventMinGap).
	started map[string]time.Time
}{idleFired: map[string]time.Time{}, started: map[string]time.Time{}}

// signedInSessions lists the sessions someone is signed in at (keys that stay
// the same while the session lasts); known is false when it cannot be read.
// A variable so the tests can stand in for the system.
var signedInSessions = platformSignedInSessions

// newSignIns compares the sessions signed in now with the last look: true
// when one has appeared. The first look only notes them (those already
// signed in when the scheduler started are not new).
func newSignIns(now map[string]bool) bool {
	eventState.Lock()
	defer eventState.Unlock()
	prev := eventState.sessions
	eventState.sessions = now
	if prev == nil {
		return false
	}
	for k := range now {
		if !prev[k] {
			return true
		}
	}
	return false
}

// checkComputerEvents starts a run of each set whose events trigger fired
// since the last tick: resumed is true when the computer woke between them.
// Called on every scheduler tick.
func (a *App) checkComputerEvents(jobs []ScheduledJob, now time.Time, resumed bool) {
	var wantSignIn, wantIdle bool
	for _, j := range jobs {
		if computerEventsOn(j) {
			wantSignIn = wantSignIn || j.ComputerEvents.SignIn
			wantIdle = wantIdle || j.ComputerEvents.IdleMinutes > 0
		}
	}
	signedIn := false
	if wantSignIn {
		if s, ok := signedInSessions(); ok {
			signedIn = newSignIns(s)
		}
	}
	var lastInput time.Time
	inputKnown := false
	if wantIdle {
		lastInput, inputKnown = lastUserInput()
	}
	for _, job := range jobs {
		if !computerEventsOn(job) {
			continue
		}
		e := job.ComputerEvents
		trigger, why := "", ""
		switch {
		case resumed && e.Wakes && wokenForSet(job.ID, now):
			// This app woke the computer for one of the set's times: that
			// time's run follows in a moment, as scheduled.
			writeDebugLog(fmt.Sprintf("[Trigger] %s: the computer woke for one of the set's times: that run follows, not a second one", job.Name))
		case resumed && e.Wakes:
			trigger, why = triggerWake, "the computer woke"
		case signedIn && e.SignIn:
			trigger, why = triggerSignIn, "someone signed in"
		case e.IdleMinutes > 0 && inputKnown && idleDue(job.ID, e.IdleMinutes, lastInput, now):
			trigger, why = triggerIdle, fmt.Sprintf("nobody has used the computer for %d minutes", e.IdleMinutes)
		}
		if trigger == "" {
			continue
		}
		a.startEventRun(job, trigger, why, now)
	}
}

// idleDue reports whether the set's idle trigger should fire: nobody has used
// the computer for minutes, and it has not fired since they last did.
func idleDue(setID string, minutes int, lastInput, now time.Time) bool {
	if now.Sub(lastInput) < time.Duration(minutes)*time.Minute {
		return false
	}
	eventState.Lock()
	defer eventState.Unlock()
	if fired, ok := eventState.idleFired[setID]; ok && !lastInput.After(fired) {
		return false
	}
	eventState.idleFired[setID] = now
	return true
}

// startEventRun starts a run of job for an event, unless the set is running,
// a scheduled run of it is due (that run goes first), an event started one
// within eventMinGap, or its On battery rule would hold it back.
func (a *App) startEventRun(job ScheduledJob, trigger, why string, now time.Time) {
	if next, err := time.Parse(time.RFC3339, job.NextRun); err == nil && !now.Before(next) {
		writeDebugLog(fmt.Sprintf("[Trigger] %s: %s, but a scheduled run is due: that run goes first", job.Name, why))
		return
	}
	runningJobsMutex.Lock()
	running := runningJobs[job.ID]
	runningJobsMutex.Unlock()
	if running {
		return
	}
	eventState.Lock()
	recent := now.Sub(eventState.started[job.ID]) < eventMinGap
	if !recent {
		eventState.started[job.ID] = now
	}
	eventState.Unlock()
	if recent {
		writeDebugLog(fmt.Sprintf("[Trigger] %s: %s, but an event started a backup less than %s ago", job.Name, why, formatDuration(eventMinGap)))
		return
	}
	if hold, _ := job.batteryRule().Holds(readBattery()); hold {
		writeDebugLog(fmt.Sprintf("[Trigger] %s: %s, but the On battery rule holds the run back", job.Name, why))
		return
	}
	writeDebugLog(fmt.Sprintf("[Trigger] %s: %s: starting a backup", job.Name, why))
	startEventBackup(a, job, trigger)
}

// startEventBackup starts the run; a variable so the tests can see it.
var startEventBackup = func(a *App, job ScheduledJob, trigger string) { go a.executeScheduledJob(job, trigger) }
