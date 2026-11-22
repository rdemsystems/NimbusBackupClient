package main

import (
	"encoding/json"
	"testing"
	"time"
)

func boolp(b bool) *bool { return &b }

// A set saved before each schedule had its own wake choice: every schedule
// gets the set's old choice, the old flag stays as it was, and a schedule
// that already has its own is never changed.
func TestScheduleWakeMovedFromTheSetOnce(t *testing.T) {
	old := ScheduledJob{ID: "a", Enabled: true, ScheduleMode: ScheduleModeScheduled, WakeToRun: true,
		Schedules: []Schedule{{ID: "s1", Enabled: true, Unit: "days", Every: 1, At: "02:00"}, {ID: "s2", Enabled: true, Unit: "days", Every: 1, At: "13:00"}}}
	normalizeJobSchedules(&old)
	for _, s := range old.Schedules {
		if s.Wake == nil || !*s.Wake {
			t.Fatalf("schedule %s: wake %v, want copied on from the set", s.ID, s.Wake)
		}
	}
	if !old.WakeToRun {
		t.Error("the set-wide flag went off: an older app would stop waking the computer")
	}

	// Now one line is turned off by the editor: the other keeps waking, the
	// set-wide flag stays on (any line wakes), and loading again changes nothing.
	old.Schedules[1].Wake = boolp(false)
	normalizeJobSchedules(&old)
	if !*old.Schedules[0].Wake || *old.Schedules[1].Wake || !old.WakeToRun {
		t.Fatalf("after editing: s1 %v s2 %v set %v; want true false true", *old.Schedules[0].Wake, *old.Schedules[1].Wake, old.WakeToRun)
	}

	// Both lines off: the set-wide flag is written off, and it is never read
	// back into the lines (they have their own choice).
	old.Schedules[0].Wake = boolp(false)
	normalizeJobSchedules(&old)
	if old.WakeToRun {
		t.Error("no line wakes but the set-wide flag is on")
	}
	old.WakeToRun = true // an older app turned it on again
	normalizeJobSchedules(&old)
	if *old.Schedules[0].Wake || *old.Schedules[1].Wake {
		t.Error("the old set-wide flag was read back over the lines' own choices")
	}

	// A set that did not wake: its lines get "off".
	off := ScheduledJob{ID: "b", Enabled: true, ScheduleMode: ScheduleModeScheduled,
		Schedules: []Schedule{{ID: "s1", Enabled: true, Unit: "days", Every: 1, At: "02:00"}}}
	normalizeJobSchedules(&off)
	if off.Schedules[0].Wake == nil || *off.Schedules[0].Wake || off.WakeToRun {
		t.Fatalf("a set that did not wake: line %v, set %v", off.Schedules[0].Wake, off.WakeToRun)
	}
}

// The wake timer is for the next run of a line that wakes, not the set's next run.
func TestNextWakeIsForALineThatWakes(t *testing.T) {
	now := time.Date(2026, 11, 23, 10, 0, 0, 0, time.Local)
	j := ScheduledJob{ID: "a", Name: "a", Enabled: true, ScheduleMode: ScheduleModeScheduled,
		Schedules: []Schedule{
			{ID: "noon", Enabled: true, Unit: "days", Every: 1, At: "13:00", Wake: boolp(false)},
			{ID: "night", Enabled: true, Unit: "days", Every: 1, At: "02:00", Wake: boolp(true)},
		}}
	setNextRun(&j, now)
	w, ok := nextWake([]ScheduledJob{j}, now)
	want := time.Date(2026, 11, 24, 2, 0, 0, 0, time.Local)
	if !ok || !w.Run.Equal(want) {
		t.Fatalf("wakes for %v (%v), want the 02:00 line at %s, not the 13:00 one that does not wake", w.Run, ok, want)
	}
	j.Schedules[1].Enabled = false
	if _, ok := nextWake([]ScheduledJob{j}, now); ok {
		t.Error("a wake with the only waking line turned off")
	}
}

// RunAtStartup moves into ComputerEvents once and is then only written from it.
func TestComputerEventsTakeOverRunAtStartup(t *testing.T) {
	old := ScheduledJob{ID: "a", Enabled: true, ScheduleMode: ScheduleModeScheduled, RunAtStartup: true}
	normalizeJobSchedules(&old)
	if old.ComputerEvents == nil || !old.ComputerEvents.Enabled || !old.ComputerEvents.Starts {
		t.Fatalf("RunAtStartup not moved over: %+v", old.ComputerEvents)
	}
	// Starts unticked in the editor: RunAtStartup is written off, and stays
	// off even though an older copy of the set said true.
	old.ComputerEvents.Starts, old.ComputerEvents.Wakes = false, true
	old.RunAtStartup = true
	normalizeJobSchedules(&old)
	if old.ComputerEvents.Starts || old.RunAtStartup {
		t.Errorf("starts %v, runAtStartup %v: the old flag came back", old.ComputerEvents.Starts, old.RunAtStartup)
	}
	// The trigger deleted (nil) with RunAtStartup sent as false stays deleted.
	old.ComputerEvents, old.RunAtStartup = nil, false
	normalizeJobSchedules(&old)
	if old.ComputerEvents != nil {
		t.Error("a deleted trigger came back")
	}
	// A manual set never runs at startup for an older app.
	m := ScheduledJob{ID: "m", ScheduleMode: ScheduleModeManual, ComputerEvents: &ComputerEvents{Enabled: true, Starts: true}}
	normalizeJobSchedules(&m)
	if m.RunAtStartup {
		t.Error("a manual set written with runAtStartup on")
	}
	// Written to the file under its own key, beside runAtStartup.
	b, _ := json.Marshal(ScheduledJob{ComputerEvents: &ComputerEvents{Enabled: true, Wakes: true, IdleMinutes: 15}})
	var back map[string]interface{}
	_ = json.Unmarshal(b, &back)
	ce, _ := back["computerEvents"].(map[string]interface{})
	if ce["wakes"] != true || ce["idleMinutes"] != float64(15) {
		t.Errorf("written as %v", back["computerEvents"])
	}
}

func TestComputerEventsValidated(t *testing.T) {
	if err := validateComputerEvents(&ComputerEvents{IdleMinutes: -1}); err == nil {
		t.Error("negative idle minutes accepted")
	}
	if err := validateComputerEvents(&ComputerEvents{IdleMinutes: maxIdleMinutes + 1}); err == nil {
		t.Error("idle beyond a day accepted")
	}
	if err := validateComputerEvents(&ComputerEvents{IdleMinutes: 15}); err != nil {
		t.Error(err)
	}
}

// startedRuns records the event runs started, instead of backing up.
func startedRuns(t *testing.T) *[]string {
	got := &[]string{}
	old := startEventBackup
	startEventBackup = func(_ *App, job ScheduledJob, trigger string) { *got = append(*got, job.ID+":"+trigger) }
	eventState.Lock()
	eventState.sessions, eventState.idleFired, eventState.started = nil, map[string]time.Time{}, map[string]time.Time{}
	eventState.Unlock()
	t.Cleanup(func() { startEventBackup = old })
	return got
}

func eventsSet(id string, e ComputerEvents) ScheduledJob {
	e.Enabled = true
	return ScheduledJob{ID: id, Name: id, Enabled: true, ScheduleMode: ScheduleModeScheduled, ComputerEvents: &e}
}

func TestWakeEventStartsARunOnceAndNotForOurOwnWake(t *testing.T) {
	got := startedRuns(t)
	a := &App{}
	now := time.Now()
	wakes := eventsSet("w", ComputerEvents{Wakes: true})
	starts := eventsSet("s", ComputerEvents{Starts: true}) // starts only: no run on a wake
	a.checkComputerEvents([]ScheduledJob{wakes, starts}, now, false)
	if len(*got) != 0 {
		t.Fatalf("runs without a wake: %v", *got)
	}
	a.checkComputerEvents([]ScheduledJob{wakes, starts}, now, true)
	if len(*got) != 1 || (*got)[0] != "w:"+triggerWake {
		t.Fatalf("on a wake: %v, want one run of w", *got)
	}
	// A second wake minutes later starts no second run (eventMinGap).
	a.checkComputerEvents([]ScheduledJob{wakes}, now.Add(5*time.Minute), true)
	if len(*got) != 1 {
		t.Fatalf("a second wake within %s started another run: %v", eventMinGap, *got)
	}

	// Woken by this app's own timer for one of the set's times: that run
	// follows; the wake event starts nothing.
	*got = nil
	other := eventsSet("o", ComputerEvents{Wakes: true})
	wokenRun.Lock()
	wokenRun.setID, wokenRun.at, wokenRun.used = "o", now, false
	wokenRun.Unlock()
	t.Cleanup(func() { wokenRun.Lock(); wokenRun.setID = ""; wokenRun.Unlock() })
	a.checkComputerEvents([]ScheduledJob{other}, now, true)
	if len(*got) != 0 {
		t.Errorf("a wake for the set's own time started an extra run: %v", *got)
	}
}

func TestSignInEventOnlyForANewSession(t *testing.T) {
	got := startedRuns(t)
	sessions := map[string]bool{"1": true}
	old := signedInSessions
	signedInSessions = func() (map[string]bool, bool) { return sessions, true }
	t.Cleanup(func() { signedInSessions = old })
	a := &App{}
	set := eventsSet("si", ComputerEvents{SignIn: true})
	now := time.Now()
	a.checkComputerEvents([]ScheduledJob{set}, now, false) // first look: already signed in
	if len(*got) != 0 {
		t.Fatalf("a session signed in before the first look counted: %v", *got)
	}
	sessions = map[string]bool{"1": true, "2": true}
	a.checkComputerEvents([]ScheduledJob{set}, now.Add(time.Minute), false)
	if len(*got) != 1 || (*got)[0] != "si:"+triggerSignIn {
		t.Fatalf("a new sign-in: %v", *got)
	}
}

func TestIdleEventOncePerIdleSpell(t *testing.T) {
	got := startedRuns(t)
	now := time.Now()
	input := now.Add(-20 * time.Minute)
	old := lastUserInput
	lastUserInput = func() (time.Time, bool) { return input, true }
	t.Cleanup(func() { lastUserInput = old })
	a := &App{}
	set := eventsSet("id", ComputerEvents{IdleMinutes: 30})
	a.checkComputerEvents([]ScheduledJob{set}, now, false)
	if len(*got) != 0 {
		t.Fatal("ran after 20 minutes idle, set to 30")
	}
	a.checkComputerEvents([]ScheduledJob{set}, now.Add(11*time.Minute), false)
	if len(*got) != 1 || (*got)[0] != "id:"+triggerIdle {
		t.Fatalf("after 31 minutes idle: %v", *got)
	}
	// Still idle an hour on: no second run in the same idle spell.
	a.checkComputerEvents([]ScheduledJob{set}, now.Add(70*time.Minute), false)
	if len(*got) != 1 {
		t.Fatalf("a second run in one idle spell: %v", *got)
	}
	// Someone used it, then left it idle again: runs again.
	input = now.Add(80 * time.Minute)
	a.checkComputerEvents([]ScheduledJob{set}, now.Add(115*time.Minute), false)
	if len(*got) != 2 {
		t.Fatalf("a new idle spell after use: %v", *got)
	}
	// Nobody signed in (input unknown): no run.
	lastUserInput = func() (time.Time, bool) { return time.Time{}, false }
	a.checkComputerEvents([]ScheduledJob{eventsSet("x", ComputerEvents{IdleMinutes: 5})}, now.Add(10*time.Hour), false)
	if len(*got) != 2 {
		t.Errorf("ran with nobody signed in: %v", *got)
	}
}

// A run of a set the event trigger starts waits for a due scheduled run, and
// a turned-off trigger or manual set starts nothing.
func TestEventRunsRespectScheduleAndSwitches(t *testing.T) {
	got := startedRuns(t)
	a := &App{}
	now := time.Now()
	due := eventsSet("due", ComputerEvents{Wakes: true})
	due.NextRun = now.Add(-time.Minute).Format(time.RFC3339)
	off := eventsSet("off", ComputerEvents{Wakes: true})
	off.ComputerEvents.Enabled = false
	manual := eventsSet("man", ComputerEvents{Wakes: true})
	manual.ScheduleMode = ScheduleModeManual
	a.checkComputerEvents([]ScheduledJob{due, off, manual}, now, true)
	if len(*got) != 0 {
		t.Errorf("runs started: %v", *got)
	}
}
