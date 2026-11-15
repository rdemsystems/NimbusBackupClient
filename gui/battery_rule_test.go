package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"clientcommon"
)

// batteryTest stands in for the battery and for sleeping, and starts each
// test with no held run and no wake.
type batteryTest struct {
	now   clientcommon.BatteryStatus
	slept chan struct{}
	wake  fakeWake
}

func newBatteryTest(t *testing.T) *batteryTest {
	t.Helper()
	useTempConfigDir(t)
	bt := &batteryTest{slept: make(chan struct{}, 4)}
	oldRead, oldSleep, oldDelay := readBattery, sleepComputer, sleepAfterHoldDelay
	readBattery = func() clientcommon.BatteryStatus { return bt.now }
	sleepComputer = func() error { bt.slept <- struct{}{}; return nil }
	sleepAfterHoldDelay = 0
	bt.wake.install(t)
	reset := func() {
		restartBatteryHolds()
		noteResume(time.Time{}, "")
		wokenRun.Lock()
		wokenRun.setID, wokenRun.at, wokenRun.used = "", time.Time{}, false
		wokenRun.Unlock()
	}
	reset()
	t.Cleanup(func() {
		readBattery, sleepComputer, sleepAfterHoldDelay = oldRead, oldSleep, oldDelay
		reset()
	})
	return bt
}

// restartBatteryHolds forgets the held runs in memory, as a restart of the
// service does: they are read back from the file when next looked at.
func restartBatteryHolds() {
	batteryHolds.Lock()
	batteryHolds.m = map[string]batteryHold{}
	batteryHolds.loaded = false
	batteryHolds.Unlock()
}

func onBatteryAt(pct int) clientcommon.BatteryStatus {
	return clientcommon.BatteryStatus{HasBattery: true, OnBattery: true, Percent: pct, Known: true}
}

func onMainsAt(pct int) clientcommon.BatteryStatus {
	return clientcommon.BatteryStatus{HasBattery: true, Percent: pct, Known: true}
}

func batterySet(mode string, start int, cancelBelow int) ScheduledJob {
	return ScheduledJob{ID: "docs", Name: "Documents", Enabled: true, ScheduleMode: "scheduled", BackupType: "directory",
		BackupDirs: []string{"/home/docs"}, OnBattery: mode, BatteryStartPercent: start,
		BatteryCancel: cancelBelow > 0, BatteryCancelBelow: cancelBelow}
}

func historyOf(t *testing.T, a *App) []JobHistory {
	t.Helper()
	h, err := a.GetJobHistory()
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestBatteryHoldsAScheduledRunOnceAndPluggingInReleasesIt(t *testing.T) {
	bt := newBatteryTest(t)
	a := &App{}
	job := batterySet("above", 30, 10)
	at := time.Date(2026, 11, 15, 2, 0, 30, 0, time.Local)

	bt.now = onBatteryAt(22)
	if _, ok := a.batteryGate(job, "scheduled", at); ok {
		t.Fatal("a run at 22% with a start level of 30% started")
	}
	if _, ok := a.batteryGate(job, "scheduled", at.Add(time.Minute)); ok {
		t.Fatal("the held run started on the next tick, still on battery")
	}
	h := historyOf(t, a)
	if len(h) != 1 || h[0].Status != "held" || h[0].MessageKey != MsgBatteryHeld || h[0].JobID != "docs" {
		t.Fatalf("want one held entry, got %+v", h)
	}
	if p := h[0].MessageParams; p["min"] != float64(30) || p["pct"] != float64(22) {
		t.Errorf("held entry params %v", p)
	}
	if trigger, held := batteryHeld("docs"); !held || trigger != "scheduled" {
		t.Fatal("the run is not recorded as held")
	}

	bt.now = onMainsAt(23)
	plugged := at.Add(40 * time.Minute)
	info, ok := a.batteryGate(job, "scheduled", plugged)
	if !ok || info == nil {
		t.Fatal("plugged in, the held run did not start")
	}
	if !info.heldAt.Equal(at) {
		t.Errorf("held at %s, want %s", info.heldAt, at)
	}
	if _, held := batteryHeld("docs"); held {
		t.Error("the hold was not released")
	}
	entry := JobHistory{Name: job.Name, JobID: job.ID, Status: "success", Message: "Backup completed"}
	a.applyBatteryToHistory(&entry, info)
	if len(entry.Notes) != 1 || entry.Notes[0].Key != NoteBatteryRanLate {
		t.Fatalf("the late run says nothing of its hold: %+v", entry.Notes)
	}
	if entry.Status != "success" || entry.Message != "Backup completed" {
		t.Errorf("a successful run was changed: %+v", entry)
	}
}

// A service restarted while a run is held (an upgrade, say) neither records
// the hold a second time nor forgets when it began.
func TestBatteryHoldIsKeptAcrossARestart(t *testing.T) {
	bt := newBatteryTest(t)
	a := &App{}
	job := batterySet("above", 30, 0)
	at := time.Date(2026, 11, 15, 2, 0, 30, 0, time.Local)
	job.NextRun = at.Add(-30 * time.Second).Format(time.RFC3339)

	bt.now = onBatteryAt(22)
	if _, ok := a.batteryGate(job, "scheduled", at); ok {
		t.Fatal("a run at 22% with a start level of 30% started")
	}

	restartBatteryHolds()
	restarted := at.Add(20 * time.Minute)
	dropStaleBatteryHolds([]ScheduledJob{job}, restarted) // the first tick after the restart
	if trigger, held := batteryHeld(job.ID); !held || trigger != "scheduled" {
		t.Fatal("the restart forgot the held run")
	}
	if _, ok := a.batteryGate(job, "scheduled", restarted); ok {
		t.Fatal("the held run started after the restart, still on battery")
	}
	if h := historyOf(t, a); len(h) != 1 || h[0].Status != "held" {
		t.Fatalf("want the one held entry from before the restart, got %+v", h)
	}

	restartBatteryHolds()
	bt.now = onMainsAt(25)
	info, ok := a.batteryGate(job, "scheduled", at.Add(time.Hour))
	if !ok || info == nil {
		t.Fatal("plugged in, the held run did not start")
	}
	if !info.heldAt.Equal(at) || info.heldPct != 22 {
		t.Fatalf("held at %s (%v%%), want %s (22%%)", info.heldAt, info.heldPct, at)
	}
	entry := JobHistory{Name: job.Name, JobID: job.ID, Status: "success", Message: "Backup completed"}
	a.applyBatteryToHistory(&entry, info)
	if len(entry.Notes) != 1 || entry.Notes[0].Key != NoteBatteryRanLate ||
		entry.Notes[0].Params["time"] != at.Format(time.RFC3339) || entry.Notes[0].Params["pct"] != 22 {
		t.Fatalf("the late run does not say when it was first held: %+v", entry.Notes)
	}

	// Released, the hold is gone from the file too.
	restartBatteryHolds()
	if _, held := batteryHeld(job.ID); held {
		t.Error("the released hold came back after a restart")
	}
	path, err := batteryHoldsPath()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the held runs file is left with no held runs (%v)", err)
	}
}

// A hold no longer due (the set's schedule was changed) is forgotten in the
// file as well, so a restart does not bring it back.
func TestBatteryHoldNoLongerDueIsNotKept(t *testing.T) {
	bt := newBatteryTest(t)
	a := &App{}
	job := batterySet("plugged", 0, 0)
	at := time.Date(2026, 11, 15, 2, 0, 0, 0, time.Local)
	job.NextRun = at.Format(time.RFC3339)
	bt.now = onBatteryAt(50)
	if _, ok := a.batteryGate(job, "scheduled", at.Add(time.Minute)); ok {
		t.Fatal("a set that waits for mains started on battery")
	}
	job.NextRun = at.Add(24 * time.Hour).Format(time.RFC3339)
	dropStaleBatteryHolds([]ScheduledJob{job}, at.Add(2*time.Minute))
	restartBatteryHolds()
	if _, held := batteryHeld(job.ID); held {
		t.Fatal("a hold no longer due came back after a restart")
	}
}

func TestBatteryPluggedRuleHoldsWhateverTheCharge(t *testing.T) {
	bt := newBatteryTest(t)
	a := &App{}
	bt.now = onBatteryAt(95)
	if _, ok := a.batteryGate(batterySet("plugged", 0, 0), "startup", time.Now()); ok {
		t.Fatal("a set that waits for mains started on battery")
	}
	h := historyOf(t, a)
	if len(h) != 1 || h[0].MessageKey != MsgBatteryHeldPlugged || h[0].Trigger != "startup" {
		t.Fatalf("got %+v", h)
	}
	if trigger, _ := batteryHeld("docs"); trigger != "startup" {
		t.Errorf("held by %q, want startup (tried again on every tick)", trigger)
	}
}

func TestBatteryRunNowIsNeverHeld(t *testing.T) {
	bt := newBatteryTest(t)
	a := &App{}
	bt.now = onBatteryAt(5)
	info, ok := a.batteryGate(batterySet("plugged", 0, 0), "manual", time.Now())
	if !ok || info == nil {
		t.Fatal("Run now was held back for the battery")
	}
	if h := historyOf(t, a); len(h) != 0 {
		t.Errorf("Run now recorded %+v", h)
	}
}

func TestBatteryNoBatteryIsNeverHeldOrCancelled(t *testing.T) {
	bt := newBatteryTest(t)
	a := &App{}
	bt.now = clientcommon.BatteryStatus{} // a desktop or a VM
	job := batterySet("plugged", 0, 50)
	if _, ok := a.batteryGate(job, "scheduled", time.Now()); !ok {
		t.Fatal("held on a computer with no battery")
	}
	if job.batteryRule().Cancels(bt.now) {
		t.Error("cancelled on a computer with no battery")
	}
}

// A set saved before the rule existed has none of its fields: it starts
// whatever the charge and is never cancelled, as before.
func TestBatteryExistingSetStartsAnyway(t *testing.T) {
	bt := newBatteryTest(t)
	a := &App{}
	var job ScheduledJob
	if err := json.Unmarshal([]byte(`{"id":"old","name":"Old set","enabled":true,"scheduleMode":"scheduled"}`), &job); err != nil {
		t.Fatal(err)
	}
	rule := job.batteryRule()
	if rule.Mode != clientcommon.BatteryRunAnyway || rule.CancelBelow != 0 {
		t.Fatalf("an older set's rule is %+v", rule)
	}
	bt.now = onBatteryAt(3)
	if _, ok := a.batteryGate(job, "scheduled", time.Now()); !ok {
		t.Fatal("an older set was held back")
	}
	if rule.Cancels(bt.now) {
		t.Fatal("an older set would be cancelled")
	}
	if err := validateScheduledJob(job); err != nil {
		t.Fatalf("an older set no longer validates: %v", err)
	}
}

func TestBatteryRuleIsValidated(t *testing.T) {
	bad := batterySet("above", 30, 30)
	if err := validateScheduledJob(bad); err == nil {
		t.Error("a cancel level equal to the start level was accepted")
	}
	off := batterySet("above", 30, 0)
	off.BatteryCancelBelow = 50 // the box is unticked: the level is kept, not used
	if err := validateScheduledJob(off); err != nil {
		t.Errorf("an unticked cancel level was checked: %v", err)
	}
	if err := validateScheduledJob(batterySet("whenever", 0, 0)); err == nil {
		t.Error("an unknown battery choice was accepted")
	}
}

func TestBatteryWokenAndHeldGoesBackToSleep(t *testing.T) {
	bt := newBatteryTest(t)
	a := &App{}
	job := batterySet("above", 30, 0)
	job.WakeToRun = true
	job.Schedules = []Schedule{{ID: "s1", Enabled: true, Unit: "days", Every: 1, At: "02:00"}}
	// The run fell due half a minute ago (the wake timer is refreshed with
	// the real time, so the times here are the real ones).
	run := time.Now().Truncate(time.Second).Add(-30 * time.Second)
	job.NextRun = run.Format(time.RFC3339)
	path, err := getScheduledJobsPath()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal([]ScheduledJob{job})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	// The wake timer was set for 01:58; the computer slept from 23:00.
	wakeState.mu.Lock()
	wakeState.target = wakeTarget{At: run.Add(-wakeLead), Run: run, SetID: job.ID, Set: job.Name}
	wakeState.armed = true
	wakeState.mu.Unlock()
	asleep := run.Add(-3 * time.Hour)
	woke := run.Add(-wakeLead + 5*time.Second)
	noteSchedulerTick(asleep, woke)

	bt.now = onBatteryAt(18)
	if _, ok := a.batteryGate(job, "scheduled", time.Now()); ok {
		t.Fatal("the woken run started at 18%")
	}
	h := historyOf(t, a)
	if len(h) != 1 || h[0].Status != "skipped" || h[0].MessageKey != MsgBatterySkippedWoken {
		t.Fatalf("want one skipped entry, got %+v", h)
	}
	select {
	case <-bt.slept:
	case <-time.After(5 * time.Second):
		t.Fatal("the computer was not put back to sleep")
	}
	if len(bt.wake.armed) == 0 {
		t.Error("no wake timer was set for the next run before sleeping")
	}
	// The wake is taken once: a later run is not "woken for".
	if !takeWoken(job, "scheduled", run.Add(time.Minute)).IsZero() {
		t.Error("the wake was used twice")
	}
}

func TestBatteryWokenDetectionNeedsAGapAndTheTimer(t *testing.T) {
	newBatteryTest(t)
	job := batterySet("", 0, 0)
	run := time.Date(2026, 11, 16, 2, 0, 0, 0, time.Local)
	wakeState.mu.Lock()
	wakeState.target = wakeTarget{At: run.Add(-wakeLead), Run: run, SetID: job.ID, Set: job.Name}
	wakeState.armed = true
	wakeState.mu.Unlock()

	noteSchedulerTick(run.Add(-wakeLead-time.Minute), run.Add(-wakeLead)) // awake: a minute apart
	if !takeWoken(job, "scheduled", run).IsZero() {
		t.Fatal("woken without a gap")
	}
	noteSchedulerTick(run.Add(-5*time.Hour), run.Add(-4*time.Hour)) // slept, but the timer was not due then
	if !takeWoken(job, "scheduled", run).IsZero() {
		t.Fatal("woken by a timer due outside the gap")
	}
	// Slept through the timer (Modern Standby ignored it) and opened by hand
	// hours later: never taken as woken for the run, so never put back to sleep.
	noteSchedulerTick(run.Add(-5*time.Hour), run.Add(3*time.Hour))
	if !takeWoken(job, "scheduled", run.Add(3*time.Hour)).IsZero() {
		t.Fatal("woken by hand long after the timer was due counted as woken by the timer")
	}
	noteSchedulerTick(run.Add(-5*time.Hour), run.Add(-wakeLead+time.Second))
	if !takeWoken(job, "startup", run).IsZero() {
		t.Error("a startup run counted as woken for")
	}
	if takeWoken(job, "scheduled", run).IsZero() {
		t.Fatal("the run the timer woke the computer for is not woken for")
	}
}

func TestBatteryCancelThreshold(t *testing.T) {
	bt := newBatteryTest(t)
	a := &App{}
	job := batterySet("above", 30, 10)
	bt.now = onBatteryAt(60)
	start := time.Now()
	info, ok := a.batteryGate(job, "manual", start)
	if !ok {
		t.Fatal("not started")
	}
	r := newBackupRun(job.Name, "directory", "pc")
	defer r.end()
	rule := job.batteryRule()

	bt.now = onBatteryAt(12)
	if a.checkBatteryCancel(job, rule, info, time.Now()) {
		t.Fatal("cancelled at 12% with a level of 10%")
	}
	bt.now = onMainsAt(4)
	if a.checkBatteryCancel(job, rule, info, time.Now()) {
		t.Fatal("cancelled although plugged back in")
	}
	r.queued.Store(true)
	bt.now = onBatteryAt(9)
	if a.checkBatteryCancel(job, rule, info, time.Now()) {
		t.Fatal("a run still waiting its turn was cancelled")
	}
	r.queued.Store(false)
	if !a.checkBatteryCancel(job, rule, info, time.Now()) {
		t.Fatal("not cancelled at 9%")
	}
	select {
	case <-r.ctx.Done():
	default:
		t.Fatal("the run was not stopped")
	}

	entry := JobHistory{Name: job.Name, JobID: job.ID, Status: "cancelled", Message: "Backup cancelled by user", MessageKey: MsgBackupCancelled}
	a.applyBatteryToHistory(&entry, info)
	if entry.Status != "cancelled" || entry.MessageKey != MsgBatteryCancelled || entry.logLine == nil || entry.logLine.Level != "error" {
		t.Fatalf("cancelled entry %+v", entry)
	}
	if entry.MessageParams["min"] != 10 || entry.MessageParams["startPct"] != 60 {
		t.Errorf("params %v", entry.MessageParams)
	}
	if len(entry.Notes) != 0 {
		t.Errorf("a first battery cancel has notes: %+v", entry.Notes)
	}
}

func TestBatteryTwoCancelsInARowSayToPlugIn(t *testing.T) {
	bt := newBatteryTest(t)
	a := &App{}
	job := batterySet("", 0, 10)
	if err := a.appendJobHistory(JobHistory{ID: "1", Name: job.Name, JobID: job.ID, Status: "cancelled", MessageKey: MsgBatteryCancelled}); err != nil {
		t.Fatal(err)
	}
	// A hold between the two is not a run.
	if err := a.appendJobHistory(JobHistory{ID: "2", Name: job.Name, JobID: job.ID, Status: "held", MessageKey: MsgBatteryHeld}); err != nil {
		t.Fatal(err)
	}
	bt.now = onBatteryAt(40)
	info, _ := a.batteryGate(job, "scheduled", time.Now())
	info.cancelled, info.cancelAt, info.cancelPct, info.cancelMin = true, time.Now(), 9, 10
	entry := JobHistory{Name: job.Name, JobID: job.ID, Status: "cancelled", Message: "Backup cancelled by user"}
	a.applyBatteryToHistory(&entry, info)
	if len(entry.Notes) != 1 || entry.Notes[0].Key != NoteBatteryTwice || entry.logLine.MessageKey != MsgLogBatteryCancelledTwice {
		t.Fatalf("the second cancel in a row does not say so: %+v %+v", entry.Notes, entry.logLine)
	}
	if job.batteryRule().CancelBelow != 10 {
		t.Error("the rule changed by itself")
	}

	// A successful run between them: the next cancel is a first again.
	if err := a.appendJobHistory(JobHistory{ID: "3", Name: job.Name, JobID: job.ID, Status: "success"}); err != nil {
		t.Fatal(err)
	}
	again := JobHistory{Name: job.Name, JobID: job.ID, Status: "cancelled"}
	a.applyBatteryToHistory(&again, info)
	if len(again.Notes) != 0 {
		t.Errorf("a cancel after a good run says twice in a row: %+v", again.Notes)
	}
}

func TestBatteryStartAndEndOnASuccessfulRun(t *testing.T) {
	bt := newBatteryTest(t)
	a := &App{}
	bt.now = onBatteryAt(80)
	info, _ := a.batteryGate(batterySet("", 0, 0), "scheduled", time.Now())
	bt.now = onBatteryAt(64)
	entry := JobHistory{Status: "success"}
	a.applyBatteryToHistory(&entry, info)
	if len(entry.Notes) != 1 || entry.Notes[0].Key != NoteBatteryStartEnd ||
		entry.Notes[0].Params["startPct"] != 80 || entry.Notes[0].Params["endPct"] != 64 {
		t.Fatalf("notes %+v", entry.Notes)
	}
	if entry.Notes[0].Text != "On battery: 80% at start, 64% at the end." {
		t.Errorf("text %q", entry.Notes[0].Text)
	}

	bt.now = onMainsAt(100)
	info, _ = a.batteryGate(batterySet("", 0, 0), "scheduled", time.Now())
	plain := JobHistory{Status: "success"}
	a.applyBatteryToHistory(&plain, info)
	if len(plain.Notes) != 0 {
		t.Errorf("a run on mains has battery notes: %+v", plain.Notes)
	}
}

func TestLastRunOutcomeIgnoresHeldRuns(t *testing.T) {
	jobs := []ScheduledJob{{ID: "docs", Name: "Documents"}}
	hist := []JobHistory{
		{JobID: "docs", Status: "held", Timestamp: "2026-11-15T03:00:00Z"},
		{JobID: "docs", Status: "success", Timestamp: "2026-11-14T02:00:00Z"},
	}
	failed, ended := lastRunOutcome(jobs, hist)
	if failed || ended != "2026-11-14T02:00:00Z" {
		t.Fatalf("failed %v, ended %q", failed, ended)
	}
}
