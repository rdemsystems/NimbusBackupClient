package main

import (
	"errors"
	"testing"
	"time"
)

func wakeJob(id string, wake, enabled bool, next string) ScheduledJob {
	return ScheduledJob{ID: id, Name: "set " + id, WakeToRun: wake, Enabled: enabled, ScheduleMode: "scheduled", NextRun: next}
}

func TestNextWakeIsTheEarliestRunOfASetThatWakes(t *testing.T) {
	now := time.Date(2026, 11, 14, 22, 0, 0, 0, time.UTC)
	jobs := []ScheduledJob{
		wakeJob("late", true, true, "2026-11-15T03:00:00Z"),
		wakeJob("early", true, true, "2026-11-15T01:30:00Z"),
		wakeJob("no-wake", false, true, "2026-11-14T23:00:00Z"), // earlier, but does not wake
		wakeJob("off", true, false, "2026-11-14T23:10:00Z"),     // disabled
		wakeJob("past", true, true, "2026-11-14T21:00:00Z"),     // overdue: the scheduler runs it now
		wakeJob("bad", true, true, "tonight"),                   // unreadable
		{ID: "manual", WakeToRun: true, Enabled: true, ScheduleMode: ScheduleModeManual, NextRun: "2026-11-14T23:20:00Z"},
	}
	w, ok := nextWake(jobs, now)
	if !ok || w.SetID != "early" {
		t.Fatalf("nextWake = %+v, %v; want the set early", w, ok)
	}
	if want := time.Date(2026, 11, 15, 1, 28, 0, 0, time.UTC); !w.At.Equal(want) {
		t.Errorf("wakes at %s, want %s (2 minutes before)", w.At, want)
	}
	if _, ok := nextWake(jobs[2:], now); ok {
		t.Error("a wake for sets that do not wake, are off, overdue, unreadable or manual")
	}
}

// While a run is under way the set's stored next run is that run; the timer
// for the one after is set at once (found on DMXCONTROL: a minute without one).
func TestNextWakeLooksPastARunUnderWay(t *testing.T) {
	now := time.Date(2026, 11, 14, 14, 40, 34, 0, time.Local)
	j := wakeJob("ten", true, true, time.Date(2026, 11, 14, 14, 40, 0, 0, time.Local).Format(time.RFC3339))
	j.Schedules = []Schedule{{ID: "s1", Enabled: true, Unit: "minutes", Every: 10}}
	w, ok := nextWake([]ScheduledJob{j}, now)
	if want := time.Date(2026, 11, 14, 14, 50, 0, 0, time.Local); !ok || !w.Run.Equal(want) {
		t.Fatalf("nextWake = %+v, %v; want the run at %s", w, ok, want)
	}
}

func TestNextWakeIsNotInThePast(t *testing.T) {
	now := time.Date(2026, 11, 14, 22, 0, 0, 0, time.UTC)
	w, ok := nextWake([]ScheduledJob{wakeJob("soon", true, true, "2026-11-14T22:01:00Z")}, now)
	if !ok || !w.At.Equal(now) {
		t.Fatalf("a run in one minute wakes at %s, want now (%s)", w.At, now)
	}
}

// fakeWake stands in for the system's wake timer.
type fakeWake struct {
	armed    []time.Time
	canceled int
	fail     error
}

func (f *fakeWake) install(t *testing.T) {
	t.Helper()
	old := armWakeTimer
	armWakeTimer = func(at time.Time) (func(), error) {
		if f.fail != nil {
			return nil, f.fail
		}
		f.armed = append(f.armed, at)
		return func() { f.canceled++ }, nil
	}
	t.Cleanup(func() {
		armWakeTimer = old
		wakeState.mu.Lock()
		wakeState.cancel, wakeState.target, wakeState.armed, wakeState.lastErr = nil, wakeTarget{}, false, ""
		wakeState.mu.Unlock()
	})
}

func TestWakeTimerIsSetOnceAndMovedWhenTheRunChanges(t *testing.T) {
	var f fakeWake
	f.install(t)
	now := time.Date(2026, 11, 14, 22, 0, 0, 0, time.UTC)
	jobs := []ScheduledJob{wakeJob("a", true, true, "2026-11-15T02:00:00Z")}

	updateWakeTimer(jobs, now)
	updateWakeTimer(jobs, now.Add(time.Minute)) // the scheduler's next tick: nothing changed
	// Past the wake time with the computer awake: still the same run, kept.
	updateWakeTimer(jobs, time.Date(2026, 11, 15, 1, 59, 0, 0, time.UTC))
	if len(f.armed) != 1 || f.canceled != 0 {
		t.Fatalf("armed %d times, cancelled %d; want one timer", len(f.armed), f.canceled)
	}

	jobs[0].NextRun = "2026-11-15T04:00:00Z" // the set was edited
	updateWakeTimer(jobs, now.Add(2*time.Minute))
	if len(f.armed) != 2 || f.canceled != 1 || !f.armed[1].Equal(time.Date(2026, 11, 15, 3, 58, 0, 0, time.UTC)) {
		t.Fatalf("after a change: armed %v, cancelled %d", f.armed, f.canceled)
	}

	jobs[0].WakeToRun = false // the option was turned off
	updateWakeTimer(jobs, now.Add(3*time.Minute))
	if f.canceled != 2 || len(f.armed) != 2 {
		t.Fatalf("after turning it off: armed %d, cancelled %d", len(f.armed), f.canceled)
	}
}

func TestWakeTimerDroppedWhenTheSchedulerStops(t *testing.T) {
	var f fakeWake
	f.install(t)
	updateWakeTimer([]ScheduledJob{wakeJob("a", true, true, "2099-01-01T02:00:00Z")}, time.Now())
	dropWakeTimer()
	dropWakeTimer() // a second stop does nothing more
	if f.canceled != 1 {
		t.Fatalf("cancelled %d times, want 1", f.canceled)
	}
}

func TestWakeTimerThatCannotBeSetIsTriedAgain(t *testing.T) {
	f := fakeWake{fail: errors.New("needs the background service")}
	f.install(t)
	jobs := []ScheduledJob{wakeJob("a", true, true, "2099-01-01T02:00:00Z")}
	updateWakeTimer(jobs, time.Now())
	if wakeState.armed {
		t.Fatal("a timer the system refused counts as set")
	}
	f.fail = nil
	updateWakeTimer(jobs, time.Now())
	if len(f.armed) != 1 || !wakeState.armed {
		t.Fatalf("not set once the system allows it: %v", f.armed)
	}
}
