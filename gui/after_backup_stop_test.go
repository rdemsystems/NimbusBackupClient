package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"pbscommon"
)

// Found on a laptop 2026-11-19: Stop pressed during a run the computer was
// taken as woken for, and the set's "hibernate, only if woken" hibernated it
// under the person who had pressed Stop. A run stopped by hand leaves the
// computer as it is, unless the set says otherwise; one the battery rule
// cancelled still goes on to the set's action, and so does a run that failed.
func TestAfterBackupNotAfterARunStoppedByHand(t *testing.T) {
	at := newAfterBackupTest(t)
	a := &App{}
	job := afterSet(afterBackupHibernate, 0, true)
	woken := func() *batteryRunInfo { return &batteryRunInfo{wokenAt: time.Now()} }

	a.runPostBackupActions(job, false, "Backup cancelled by user", nil, woken())
	if got := at.actedWithin(300 * time.Millisecond); got != "" {
		t.Fatalf("after a Stop the computer did %q, want it left as it is", got)
	}

	battery := woken()
	battery.cancelled = true
	a.runPostBackupActions(job, false, "The computer was woken for this backup at 03:00 and the backup started (battery 40%), but it was automatically cancelled at 03:20", nil, battery)
	if got := at.actedWithin(2 * time.Second); got != afterBackupHibernate {
		t.Fatalf("after a battery cancel the computer did %q, want %q", got, afterBackupHibernate)
	}

	a.runPostBackupActions(job, false, "Error: PBS authentication failed", nil, woken())
	if got := at.actedWithin(2 * time.Second); got != afterBackupHibernate {
		t.Fatalf("after a failed run the computer did %q, want %q", got, afterBackupHibernate)
	}

	job.AfterBackupEvenIfStopped = true // the set asks for it after a Stop too
	a.runPostBackupActions(job, false, "Backup cancelled by user", nil, woken())
	if got := at.actedWithin(2 * time.Second); got != afterBackupHibernate {
		t.Fatalf("a set that acts even after a Stop did %q, want %q", got, afterBackupHibernate)
	}
}

func TestStoppedByHand(t *testing.T) {
	battery := &batteryRunInfo{cancelled: true}
	for _, c := range []struct {
		success bool
		message string
		run     *batteryRunInfo
		want    bool
	}{
		{false, "Backup cancelled by user", nil, true},
		{false, "backup cancelled by user", &batteryRunInfo{}, true},
		{false, "Backup cancelled by user", battery, false},
		{false, "Error: no such host", nil, false},
		{true, "Backup completed", nil, false},
	} {
		if got := stoppedByHand(c.success, c.message, c.run); got != c.want {
			t.Errorf("stoppedByHand(%v, %q, battery cancel %v) = %v, want %v", c.success, c.message, c.run != nil && c.run.cancelled, got, c.want)
		}
	}
}

// fakeUserInput stands in for the keyboard and mouse of anyone signed in.
func fakeUserInput(t *testing.T, at time.Time, known bool) {
	t.Helper()
	old := lastUserInput
	lastUserInput = func() (time.Time, bool) { return at, known }
	t.Cleanup(func() { lastUserInput = old })
}

// Someone who used the computer after it woke for the backup (or after the
// backup began) is at it: the computer stays as it is, unless the set says
// otherwise. Use before then, or none known, does not count.
func TestAfterBackupNotWhenSomeoneUsesTheComputer(t *testing.T) {
	at := newAfterBackupTest(t)
	a := &App{}
	job := afterSet(afterBackupHibernate, 0, true)
	woke := time.Now().Add(-10 * time.Minute)
	run := func() *batteryRunInfo { return &batteryRunInfo{wokenAt: woke, startedAt: woke.Add(2 * time.Minute)} }

	fakeUserInput(t, woke.Add(5*time.Minute), true)
	a.startAfterBackup(job, run())
	if got := at.actedWithin(300 * time.Millisecond); got != "" {
		t.Fatalf("with someone at the computer it did %q, want it left as it is", got)
	}
	// The decision is logged by the after-backup step's own goroutine, which
	// under a loaded test machine may come after the 300 ms above: wait for
	// the line rather than look once (failed once in a full local CI run with
	// every lane busy, 25/11/2026; 20 runs on their own all passed).
	for deadline := time.Now().Add(5 * time.Second); !messageLogHas(t, a, MsgLogAfterBackupInUse); time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Error("the Message Log does not say why the computer stayed as it is")
			break
		}
	}

	job.AfterBackupEvenIfInUse = true
	a.startAfterBackup(job, run())
	if got := at.actedWithin(2 * time.Second); got != afterBackupHibernate {
		t.Fatalf("a set that acts even when the computer is in use did %q", got)
	}

	job.AfterBackupEvenIfInUse = false
	fakeUserInput(t, woke.Add(-time.Hour), true) // last used before it slept
	a.startAfterBackup(job, run())
	if got := at.actedWithin(2 * time.Second); got != afterBackupHibernate {
		t.Fatalf("use before the wake kept it from acting (did %q)", got)
	}

	fakeUserInput(t, time.Time{}, false) // nobody signed in, or no way to tell
	a.startAfterBackup(job, run())
	if got := at.actedWithin(2 * time.Second); got != afterBackupHibernate {
		t.Fatalf("with no one known to be at it the computer did %q", got)
	}
}

// messageLogHas: the Message Log holds a line with key.
func messageLogHas(t *testing.T, a *App, key MessageKey) bool {
	t.Helper()
	log, err := a.GetMessageLog()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range log {
		if m.MessageKey == key {
			return true
		}
	}
	return false
}

// While an action waits, the computer is kept awake; if it slept anyway (put
// to sleep by hand, a lid closed) the action is dropped, or the next wake
// would carry it out on whoever woke the computer.
func TestAfterBackupDroppedWhenTheComputerSleptMeanwhile(t *testing.T) {
	at := newAfterBackupTest(t)
	a := &App{}
	var mu sync.Mutex
	held, let := 0, 0
	oldKeep := keepAwakeForAfterBackup
	keepAwakeForAfterBackup = func(string, string) func() {
		mu.Lock()
		held++
		mu.Unlock()
		return func() { mu.Lock(); let++; mu.Unlock() }
	}
	// The wall clock: its second reading, after the first step of the wait,
	// is three hours on, as if the computer slept meanwhile.
	clock, reads := time.Now(), 0
	oldNow := afterBackupNow
	afterBackupNow = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		reads++
		if reads == 2 {
			clock = clock.Add(3 * time.Hour)
		}
		clock = clock.Add(time.Millisecond)
		return clock
	}
	t.Cleanup(func() { keepAwakeForAfterBackup, afterBackupNow = oldKeep, oldNow })

	a.startAfterBackup(afterSet(afterBackupShutdown, 1, false), nil) // waits a (test) minute
	if got := at.actedWithin(500 * time.Millisecond); got != "" {
		t.Fatalf("after sleeping through the wait the computer did %q, want nothing", got)
	}
	waitForAfterBackup("docs")
	mu.Lock()
	defer mu.Unlock()
	if held != 1 || let != 1 {
		t.Errorf("kept awake %d times, let go %d times, want once each", held, let)
	}
}

// Found on a VM 2026-11-20 with hibernation off: a set told to hibernate
// logged "hibernates now" and did nothing, without a word. It now says why,
// and does not try.
func TestAfterBackupHibernateThatCannotIsReported(t *testing.T) {
	at := newAfterBackupTest(t)
	a := &App{}
	canHibernate = func() pbscommon.HibernateCheck { return pbscommon.HibernateCheck{Why: pbscommon.HibernateOff} }
	a.carryOutAfterBackup(afterSet(afterBackupHibernate, 0, false), afterBackupHibernate)
	if got := at.actedWithin(100 * time.Millisecond); got != "" {
		t.Fatalf("it tried to %s anyway", got)
	}
	log, err := a.GetMessageLog()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range log {
		if e.MessageKey == MsgLogAfterBackupFailed && strings.Contains(fmt.Sprint(e.MessageParams["error"]), "hibernation is turned off") {
			found = true
		}
		if e.MessageKey == MsgLogAfterBackupHibernate {
			t.Error("the Message Log says it hibernates, though it cannot")
		}
	}
	if !found {
		t.Errorf("no line saying why it did not hibernate in %+v", log)
	}
}

func TestAfterBackupStayClearsItsOptions(t *testing.T) {
	j := afterSet(afterBackupStay, 0, false)
	j.AfterBackupEvenIfStopped, j.AfterBackupEvenIfInUse = true, true
	normalizeAfterBackup(&j)
	if j.AfterBackupEvenIfStopped || j.AfterBackupEvenIfInUse {
		t.Errorf("a set that stays as it is kept %v %v", j.AfterBackupEvenIfStopped, j.AfterBackupEvenIfInUse)
	}
}
