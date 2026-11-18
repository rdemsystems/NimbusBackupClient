package main

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/salvanza/salvanza_client/gui/api"
)

// #142: a computer woken by the timer for a set's run is kept awake until
// that run takes over, and only that run's start ends the hold.
func TestWakeHoldLastsUntilItsRunTakesOver(t *testing.T) {
	var f fakeSystem
	f.install(t)
	t.Cleanup(func() { releaseWakeHold("docs") })
	holdAwakeForWokenRun("docs", "Documents")
	if f.acquired.Load() != 1 {
		t.Fatalf("the wake did not keep the computer awake (acquired %d)", f.acquired.Load())
	}
	releaseWakeHold("photos")
	if f.released.Load() != 0 {
		t.Fatal("another set's run ended the hold taken for this one")
	}
	releaseWakeHold("docs")
	if f.released.Load() != 1 {
		t.Fatalf("the run did not end the wake's hold (released %d)", f.released.Load())
	}
}

// A woken run that never comes (the set was turned off meanwhile) must not
// keep the computer awake for good.
func TestWakeHoldEndsByItselfWhenNoRunComes(t *testing.T) {
	var f fakeSystem
	f.install(t)
	old := wakeHoldMax
	wakeHoldMax = 20 * time.Millisecond
	t.Cleanup(func() { wakeHoldMax = old; releaseWakeHold("docs") })
	holdAwakeForWokenRun("docs", "Documents")
	deadline := time.Now().Add(2 * time.Second)
	for f.released.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if f.released.Load() != 1 {
		t.Fatal("the hold outlived its time limit")
	}
}

// The scheduler's tick that finds the computer woken by the timer takes the
// hold for the set it was woken for; a wake by hand does not.
func TestATimerWakeTakesTheHold(t *testing.T) {
	newBatteryTest(t)
	var held []string
	old := holdAwakeAfterWake
	holdAwakeAfterWake = func(setID, _ string) { held = append(held, setID) }
	t.Cleanup(func() { holdAwakeAfterWake = old })
	run := time.Date(2026, 11, 17, 16, 0, 0, 0, time.Local)
	wakeState.mu.Lock()
	wakeState.target = wakeTarget{At: run.Add(-wakeLead), Run: run, SetID: "docs", Set: "Documents"}
	wakeState.armed = true
	wakeState.mu.Unlock()

	noteSchedulerTick(run.Add(-5*time.Hour), run.Add(3*time.Hour)) // opened by hand hours later
	if len(held) != 0 {
		t.Fatalf("a wake by hand took the hold: %v", held)
	}
	noteSchedulerTick(run.Add(-5*time.Hour), run.Add(-wakeLead+20*time.Second))
	if len(held) != 1 || held[0] != "docs" {
		t.Fatalf("the timer's wake took the hold for %v, want [docs]", held)
	}
}

// Found 2026-11-17: woken for a 16:00 run, its server out of reach, the run
// started anyway after the wait, ran the "before" application, failed at its
// login and wrote no log, so View log opened the run before it. It is now not
// started: a failed Reports entry that says why, a log of its own, what the
// set does after a run (here: back to sleep), and the computer was held awake
// through the wait and let go before that.
func TestARunWhoseServerIsOutOfReachAfterAWakeIsNotStarted(t *testing.T) {
	at := newAfterBackupTest(t)
	at.now = onMainsAt(100)
	var sys fakeSystem
	sys.install(t)
	woke := time.Now()
	net := newFakeNet(t, woke, 1000)
	noteResume(woke, "after the computer woke")

	a := appWithServer()
	job := afterSet(afterBackupSleep, 0, false)
	job.BackupID = "docs-pc"
	a.executeScheduledJob(job, "scheduled")

	if len(net.dials) < 2 {
		t.Fatalf("the server was tried %d times", len(net.dials))
	}
	h := historyOf(t, a)
	if len(h) != 1 || h[0].Status != "failed" || h[0].MessageKey != MsgScheduledJobError ||
		!strings.Contains(h[0].Message, "did not start") || !strings.Contains(h[0].Message, "192.168.0.242:8007") {
		t.Fatalf("want one failed entry saying the run did not start, got %+v", h)
	}
	if got := at.actedWithin(2 * time.Second); got != afterBackupSleep {
		t.Fatalf("after the run the computer did %q, want %q", got, afterBackupSleep)
	}
	if sys.acquired.Load() != 1 || sys.released.Load() != 1 {
		t.Errorf("the computer was not held awake through the wait and let go (acquired %d, released %d)", sys.acquired.Load(), sys.released.Load())
	}
	f, err := a.FindRunLog("docs-pc", time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("the run left no log of its own: %v", err)
	}
	view, err := a.ReadLog(LogQuery{Name: f.Name})
	if err != nil {
		t.Fatal(err)
	}
	if text := strings.Join(view.Lines, "\n"); !strings.Contains(text, "failed before the backup started") || !strings.Contains(text, "could not be reached") {
		t.Errorf("the run's log does not say what stopped it:\n%s", text)
	}
}

// A run that fails before the backup starts for another reason (here its PBS
// server is not in the settings) gets a log of its own too.
func TestARunThatFailsBeforeItStartsGetsItsOwnLog(t *testing.T) {
	var sys fakeSystem
	sys.install(t)
	// As the service runs a set: a standalone App in the service's process.
	a := &App{config: &Config{}, isServiceProcess: true, mode: api.ModeStandalone}
	job := afterSet("", 0, false)
	job.BackupID = "early-fail"
	job.PBSServerID = "no-such-server"
	a.executeScheduledJob(job, "manual")

	h := historyOf(t, a)
	if len(h) == 0 || h[0].Status != "failed" {
		t.Fatalf("want a failed entry, got %+v", h)
	}
	f, err := a.FindRunLog("early-fail", time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("the failed run left no log of its own: %v", err)
	}
	view, err := a.ReadLog(LogQuery{Name: f.Name})
	if err != nil {
		t.Fatal(err)
	}
	if text := strings.Join(view.Lines, "\n"); !strings.Contains(text, "failed before the backup started") {
		t.Errorf("the run's log does not say what stopped it:\n%s", text)
	}
}

// Found on a laptop 2026-11-18: woken by the timer at its very second, the
// first tick after the resume read the clock a moment before the timer was
// due, so the wake was missed (no hold, and the set's "only if woken" after
// action did nothing). A tick just before the timer still counts; one well
// before it does not.
func TestATickJustBeforeTheTimerStillCountsAsItsWake(t *testing.T) {
	newBatteryTest(t)
	var held []string
	old := holdAwakeAfterWake
	holdAwakeAfterWake = func(setID, _ string) { held = append(held, setID) }
	t.Cleanup(func() { holdAwakeAfterWake = old })
	run := time.Date(2026, 11, 18, 11, 6, 0, 0, time.Local)
	wakeState.mu.Lock()
	wakeState.target = wakeTarget{At: run.Add(-wakeLead), Run: run, SetID: "b", Set: "wake-142-b"}
	wakeState.armed = true
	wakeState.mu.Unlock()
	asleep := time.Date(2026, 11, 18, 10, 56, 42, 0, time.Local)

	noteSchedulerTick(asleep, run.Add(-wakeLead-2*wakeEarly)) // woke by hand minutes before
	if len(held) != 0 {
		t.Fatalf("a wake well before the timer took the hold: %v", held)
	}
	noteSchedulerTick(asleep, run.Add(-wakeLead-50*time.Millisecond))
	if len(held) != 1 || held[0] != "b" {
		t.Fatalf("the wake a moment before the timer was missed: %v", held)
	}
}

// On a Modern Standby computer the service log says once, not at every wake
// timer, that waking is best effort.
func TestModernStandbyIsNotedOnce(t *testing.T) {
	calls := 0
	old := readWakeTimerInfo
	readWakeTimerInfo = func() WakeTimerInfo { calls++; return WakeTimerInfo{Supported: true, ModernStandby: true} }
	modernStandbyNoted = sync.Once{}
	t.Cleanup(func() { readWakeTimerInfo = old; modernStandbyNoted = sync.Once{} })
	noteModernStandby()
	noteModernStandby()
	if calls != 1 {
		t.Errorf("the power capabilities were read %d times, want once", calls)
	}
}

// A run that opens its own log is not given a second one.
func TestRunLogStartedSince(t *testing.T) {
	before := time.Now()
	noteRunLogStarted("some pc")
	if !runLogStartedSince("some pc", before) {
		t.Error("a run log opened after the run began was not seen")
	}
	if runLogStartedSince("some pc", time.Now().Add(time.Second)) {
		t.Error("a run log opened before the run began counted as its own")
	}
	if runLogStartedSince("another pc", before) {
		t.Error("another backup ID's log counted")
	}
}
