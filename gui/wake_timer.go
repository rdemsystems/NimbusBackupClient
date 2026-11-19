package main

import (
	"fmt"
	"sync"
	"time"
)

// Waking the computer for a Backup Set's scheduled run (issue #140). A set
// with WakeToRun has the system woken from sleep shortly before its next run,
// so a backup scheduled for the night runs at night, not when someone wakes
// the computer in the morning. The scheduler (StartScheduler) keeps one wake
// timer set, for the earliest such run, on every tick; it drops it when it
// stops, as then nothing would run the backup. Once awake, preventSleep keeps
// the computer awake until the backup ends.
//
// The platform files (wake_windows.go, wake_linux.go, wake_other.go) set the
// timer: a waitable timer with resume on Windows, a CLOCK_REALTIME_ALARM timer
// on Linux, with the hardware clock's wake alarm kept set beside it for
// hibernate. It can wake a computer from sleep, and usually from hibernate,
// not from shut down.

// wakeLead is how long before the run the computer is woken: time to resume,
// for the network to come back and for the scheduler's next minute tick.
const wakeLead = 2 * time.Minute

// armWakeTimer sets the system's wake timer for at and returns what cancels
// it. A variable so the tests can stand in for the system.
var armWakeTimer = platformArmWakeTimer

// syncWakeAlarm makes sure, just before the computer is put to sleep or
// hibernated, that the system will wake it for the armed wake: on Linux the
// hardware clock's own wake alarm, which the kernel does not set for
// hibernate (wake_linux.go). A variable so the tests can stand in for it.
var syncWakeAlarm = platformSyncWakeAlarm

// wakeTarget is the run a wake timer is for.
type wakeTarget struct {
	At    time.Time // when to wake (the run less wakeLead, not before now)
	Run   time.Time // the run itself
	SetID string
	Set   string
}

// nextWake is the earliest run of an enabled, scheduled set with WakeToRun
// that is still to come, and when to wake for it; ok is false when there is
// none. A run already due is not woken for (the scheduler is running it):
// while it runs, the set's stored next run is still the one under way, so the
// run after it is worked out from the set's schedules, and the timer for it
// is set at once rather than when the run has finished.
func nextWake(jobs []ScheduledJob, now time.Time) (w wakeTarget, ok bool) {
	for _, j := range jobs {
		if !j.WakeToRun || !j.Enabled || j.ScheduleMode == ScheduleModeManual {
			continue
		}
		run, err := time.Parse(time.RFC3339, j.NextRun)
		if err != nil || !run.After(now) {
			next, _ := nextRunFor(j, now)
			if run, err = time.Parse(time.RFC3339, next); err != nil || !run.After(now) {
				continue
			}
		}
		if !ok || run.Before(w.Run) {
			w, ok = wakeTarget{Run: run, SetID: j.ID, Set: j.Name}, true
		}
	}
	if ok {
		w.At = w.Run.Add(-wakeLead)
		if w.At.Before(now) {
			w.At = now
		}
	}
	return w, ok
}

// wakeState is the timer currently set, if any.
var wakeState struct {
	mu      sync.Mutex
	cancel  func()
	target  wakeTarget
	armed   bool
	lastErr string
}

// updateWakeTimer sets, moves or drops the wake timer to match jobs at now.
// It touches the system only when the wanted time changes, and logs each
// change and each new error once.
func updateWakeTimer(jobs []ScheduledJob, now time.Time) {
	w, want := nextWake(jobs, now)
	wakeState.mu.Lock()
	defer wakeState.mu.Unlock()
	// The same set's same run: keep the timer. (Once its wake time has passed
	// with the computer awake, At would be "now" on every tick.)
	if want && wakeState.armed && wakeState.target.Run.Equal(w.Run) && wakeState.target.SetID == w.SetID {
		return
	}
	if !want && !wakeState.armed {
		return
	}
	dropWakeTimerLocked()
	if !want {
		writeDebugLog("[Wake] no Backup Set needs to wake the computer: wake timer dropped")
		return
	}
	cancel, err := armWakeTimer(w.At)
	if err != nil {
		msg := fmt.Sprintf("[Wake] could not set a wake timer for %q at %s: %v", w.Set, w.At.Format(time.RFC3339), err)
		if msg != wakeState.lastErr {
			writeDebugLog(msg)
			wakeState.lastErr = msg
		}
		return
	}
	wakeState.cancel, wakeState.target, wakeState.armed, wakeState.lastErr = cancel, w, true, ""
	writeDebugLog(fmt.Sprintf("[Wake] wake timer set for %s, for %q at %s", w.At.Format(time.RFC3339), w.Set, w.Run.Format(time.RFC3339)))
	noteModernStandby()
}

// modernStandbyNoted: the service log has said once, since the start, that
// waking is best effort on this computer.
var modernStandbyNoted sync.Once

// noteModernStandby says once in the service log, the first time a wake timer
// is set, when the computer uses Modern Standby: whoever reads the log of a
// run that started late or not at all then knows why (#142).
func noteModernStandby() {
	modernStandbyNoted.Do(func() {
		if readWakeTimerInfo().ModernStandby {
			writeDebugLog("[Wake] this computer uses Modern Standby (S0 low power idle): Windows decides whether to honour the wake timer, " +
				"the request to keep the computer awake and the request to sleep, so waking for a backup is best effort here. " +
				"The reliable way is to turn Modern Standby off and hibernate the computer (see Waking the computer in the manual)")
		}
	})
}

// dropWakeTimer cancels the wake timer, if one is set (the scheduler stops).
func dropWakeTimer() {
	wakeState.mu.Lock()
	defer wakeState.mu.Unlock()
	if wakeState.armed {
		dropWakeTimerLocked()
		writeDebugLog("[Wake] scheduler stopped: wake timer dropped")
	}
}

func dropWakeTimerLocked() {
	if wakeState.cancel != nil {
		wakeState.cancel()
	}
	wakeState.cancel, wakeState.target, wakeState.armed = nil, wakeTarget{}, false
}

// refreshWakeTimer reads the Backup Sets and updates the wake timer; the
// scheduler calls it on every tick and when it starts.
func (a *App) refreshWakeTimer() {
	jobs, err := a.GetScheduledJobs()
	if err != nil {
		return
	}
	updateWakeTimer(jobs, time.Now())
}

// wakeTimerAfterSetsChanged sets the wake timer again at once when Backup
// Sets were saved, changed, turned on or off, deleted or imported, where the
// scheduler runs in this process. Found on a laptop 2026-11-19: the timer was
// moved only at the scheduler's next tick, up to a minute later, so a set
// saved and the computer shut down at once kept the old wake time.
func (a *App) wakeTimerAfterSetsChanged() {
	schedulerMu.Lock()
	running := schedulerRunning
	schedulerMu.Unlock()
	if running {
		a.refreshWakeTimer()
	}
}

// WakeTimerInfo is what the app shows under a set's Wake option.
type WakeTimerInfo struct {
	// Supported: this computer's system can set a wake timer at all.
	Supported bool `json:"supported"`
	// OnMains and OnBattery: whether the power plan lets a timer wake the
	// computer (Windows' "Allow wake timers"); both true where there is no
	// such setting.
	OnMains   bool `json:"onMains"`
	OnBattery bool `json:"onBattery"`
	// ModernStandby: Windows on a computer with Modern Standby (S0 low power
	// idle) and no classic sleep. Windows decides there whether a program's
	// timer may wake it, and often does not, above all on battery, so the app
	// says it may not wake rather than promising it.
	ModernStandby bool   `json:"modernStandby"`
	Note          string `json:"note"`
	// HasBattery: the computer has a battery of its own (a laptop).
	HasBattery bool `json:"hasBattery"`
	// Alert is the question the app asks under the Wake option when the
	// power plan stops a timer waking the computer (wakeAlertFor): "all",
	// "battery" or "" for none.
	Alert string `json:"alert"`
	// planRead: the power plan's setting was read, not assumed. The alert
	// offers to change only a setting that is really there.
	planRead bool
}

// GetWakeTimerInfo tells the window whether waking for a backup can work on
// this computer, and what stops it.
func (a *App) GetWakeTimerInfo() WakeTimerInfo {
	return wakeTimerInfoNow()
}
