package main

import (
	"testing"
	"time"
)

// standbyTest sets up a computer with or without Modern Standby and forgets
// the screen's state and any waiting sets afterwards.
func standbyTest(t *testing.T, modern bool) {
	t.Helper()
	old := isModernStandby
	isModernStandby = func() bool { return modern }
	reset := func() {
		standbyState.Lock()
		standbyState.known, standbyState.off, standbyState.waiting = false, false, nil
		standbyState.Unlock()
	}
	reset()
	t.Cleanup(func() { isModernStandby = old; reset() })
}

// On a computer with classic sleep nothing changes: no check, the run goes on.
func TestStandbyClassicSleepUnchanged(t *testing.T) {
	standbyTest(t, false)
	f := newFakeNet(t, time.Date(2020, 3, 1, 4, 1, 0, 0, time.Local), 1000)
	noteDisplayState(false, f.now)
	if !appWithServer().standbyCheck(batterySet("plugged", 0, 0), "scheduled") {
		t.Fatal("a run on a computer with classic sleep was held for the screen")
	}
	if len(f.dials) != 0 {
		t.Errorf("tried the server %d times, want none", len(f.dials))
	}
}

// With the screen's state not known (Windows did not report it), a Modern
// Standby computer goes on as before.
func TestStandbyUnknownScreenUnchanged(t *testing.T) {
	standbyTest(t, true)
	f := newFakeNet(t, time.Date(2020, 3, 1, 4, 1, 0, 0, time.Local), 1000)
	if !appWithServer().standbyCheck(batterySet("plugged", 0, 0), "scheduled") || len(f.dials) != 0 {
		t.Fatal("a run was held although the screen's state is not known")
	}
}

// Screen on: no check either.
func TestStandbyScreenOnUnchanged(t *testing.T) {
	standbyTest(t, true)
	f := newFakeNet(t, time.Date(2020, 3, 1, 9, 0, 0, 0, time.Local), 1000)
	noteDisplayState(true, f.now)
	if !appWithServer().standbyCheck(batterySet("plugged", 0, 0), "scheduled") || len(f.dials) != 0 {
		t.Fatal("a run was held with the screen on")
	}
}

// Networking in standby on (Sean's laptop once it was turned on): the server
// answers and the run goes on, at once.
func TestStandbyServerAnswersRunGoesOn(t *testing.T) {
	standbyTest(t, true)
	f := newFakeNet(t, time.Date(2020, 3, 1, 3, 0, 0, 0, time.Local), 0)
	noteDisplayState(false, f.now.Add(-time.Hour))
	set := batterySet("plugged", 0, 0)
	if !appWithServer().standbyCheck(set, "scheduled") {
		t.Fatal("the server answered, yet the run waits for the screen")
	}
	if waitingForScreen(set.ID) {
		t.Error("the set is marked as waiting for the screen")
	}
}

// Woken in standby with no network: the run waits for the screen (no failed
// run, no retry every minute), then runs once when the screen comes on, and
// says why it ran late.
func TestStandbyNoNetworkWaitsForTheScreen(t *testing.T) {
	standbyTest(t, true)
	woke := time.Date(2020, 3, 1, 4, 1, 0, 0, time.Local)
	f := newFakeNet(t, woke, 1000)
	noteDisplayState(false, woke.Add(-5*time.Hour))
	set := batterySet("plugged", 0, 0)
	a := appWithServer()
	if a.standbyCheck(set, "scheduled") {
		t.Fatal("the server never answered, yet the run goes on")
	}
	if took := f.now.Sub(woke); took > standbyCheckMax+networkRetryEvery {
		t.Errorf("the check took %s, want at most %s", took, standbyCheckMax)
	}
	if !waitingForScreen(set.ID) {
		t.Fatal("the set does not wait for the screen")
	}
	if _, ok, _ := takeStandbyWait(set.ID); ok {
		t.Error("the wait was taken before the screen came on")
	}
	// Run now is not held by it.
	if !a.standbyCheck(set, "manual") {
		t.Error("Run now was held for the screen")
	}

	// The screen comes on: the set is let go, runs now wait for their server
	// as after a wake, and the next run takes the wait for its line in Reports.
	on := time.Date(2020, 3, 1, 8, 30, 0, 0, time.Local)
	f.now = on
	noteDisplayState(true, on)
	if waitingForScreen(set.ID) {
		t.Error("the set still waits after the screen came on")
	}
	if why := resumedRecently(on.Add(time.Minute)); why != "after the screen came on" {
		t.Errorf("runs now wait for their server %q, want after the screen came on", why)
	}
	// It began waiting when the short check ended.
	since, ok, expired := takeStandbyWait(set.ID)
	if !ok || expired || since.Before(woke) || since.After(woke.Add(standbyCheckMax+networkRetryEvery)) {
		t.Errorf("the wait taken = %v, %v, expired %v; want since about %v", since, ok, expired, woke.Add(standbyCheckMax))
	}
	if _, ok, _ := takeStandbyWait(set.ID); ok {
		t.Error("the wait was taken twice")
	}
}

// A screen that never comes on (a mini PC without a monitor): after
// standbyWaitMax the run goes ahead, so a failure is recorded, not hidden.
func TestStandbyWaitRunsOut(t *testing.T) {
	standbyTest(t, true)
	start := time.Date(2020, 3, 1, 4, 1, 0, 0, time.Local)
	f := newFakeNet(t, start, 1000)
	noteDisplayState(false, start.Add(-time.Hour))
	waitForScreen("s1", "Set", start)
	f.now = start.Add(standbyWaitMax - time.Minute)
	if !waitingForScreen("s1") {
		t.Fatal("let go before the wait ran out")
	}
	f.now = start.Add(standbyWaitMax)
	if waitingForScreen("s1") {
		t.Fatal("still waiting after standbyWaitMax")
	}
	since, ok, expired := takeStandbyWait("s1")
	if !ok || !expired || !since.Equal(start) {
		t.Errorf("taken %v %v expired %v, want since %v and expired", since, ok, expired, start)
	}
}

// A second check of a set already waiting keeps the first time it waited.
func TestStandbyWaitKeepsItsStart(t *testing.T) {
	standbyTest(t, true)
	first := time.Date(2020, 3, 1, 4, 1, 0, 0, time.Local)
	waitForScreen("s1", "Set", first)
	waitForScreen("s1", "Set", first.Add(time.Hour))
	noteDisplayState(true, first.Add(4*time.Hour))
	if since, ok, _ := takeStandbyWait("s1"); !ok || !since.Equal(first) {
		t.Errorf("since %v %v, want %v", since, ok, first)
	}
}
