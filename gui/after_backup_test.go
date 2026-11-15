package main

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// afterBackupTest stands in for sleeping, hibernating and shutting down (and
// the battery and the wake timer, through batteryTest), shortens the waits,
// and starts and ends each test with nothing waiting.
type afterBackupTest struct {
	*batteryTest
	acted chan string
}

func newAfterBackupTest(t *testing.T) *afterBackupTest {
	t.Helper()
	at := &afterBackupTest{batteryTest: newBatteryTest(t), acted: make(chan string, 8)}
	sleepComputer = func() error { at.acted <- afterBackupSleep; return nil }
	oldHib, oldShut, oldMinute, oldPoll := hibernateComputer, shutdownComputerNow, afterBackupMinute, afterBackupPoll
	hibernateComputer = func() error { at.acted <- afterBackupHibernate; return nil }
	shutdownComputerNow = func() (string, error) { at.acted <- afterBackupShutdown; return "a test", nil }
	afterBackupMinute, afterBackupPoll = 20*time.Millisecond, 5*time.Millisecond
	dropAll := func() {
		afterBackups.Lock()
		for _, p := range afterBackups.bySet {
			p.cancel()
		}
		afterBackups.bySet = map[string]*pendingAfterBackup{}
		afterBackups.Unlock()
	}
	dropAll()
	t.Cleanup(func() {
		dropAll()
		hibernateComputer, shutdownComputerNow, afterBackupMinute, afterBackupPoll = oldHib, oldShut, oldMinute, oldPoll
	})
	return at
}

// actedWithin returns the action carried out within d, "" for none.
func (at *afterBackupTest) actedWithin(d time.Duration) string {
	select {
	case a := <-at.acted:
		return a
	case <-time.After(d):
		return ""
	}
}

func afterSet(choice string, wait int, onlyIfWoken bool) ScheduledJob {
	return ScheduledJob{ID: "docs", Name: "Documents", Enabled: true, ScheduleMode: "scheduled", BackupType: "directory",
		BackupDirs: []string{"/home/docs"}, AfterBackup: choice, AfterBackupWait: wait, AfterBackupOnlyIfWoken: onlyIfWoken,
		ShutdownAfter: choice == afterBackupShutdown}
}

// A set saved with "Turn off the computer after the backup" shuts down, at
// once, after any run, as before; ShutdownAfter is still written, true only
// for a shutdown, for an older app or service reading the file.
func TestAfterBackupMigratesShutdownAfter(t *testing.T) {
	var old ScheduledJob
	if err := json.Unmarshal([]byte(`{"id":"old","name":"Old set","enabled":true,"scheduleMode":"scheduled","shutdownAfter":true}`), &old); err != nil {
		t.Fatal(err)
	}
	normalizeJobSchedules(&old)
	if old.AfterBackup != afterBackupShutdown || old.AfterBackupWait != 0 || old.AfterBackupOnlyIfWoken || !old.ShutdownAfter {
		t.Fatalf("an older shutdown set became %q, %d minutes, only if woken %v, shutdownAfter %v", old.AfterBackup, old.AfterBackupWait, old.AfterBackupOnlyIfWoken, old.ShutdownAfter)
	}
	if action, _ := afterBackupFor(old, time.Time{}); action != afterBackupShutdown {
		t.Errorf("an older shutdown set does %q after a run it was not woken for", action)
	}

	sleep := afterSet(afterBackupSleep, 10, true)
	normalizeJobSchedules(&sleep)
	data, _ := json.Marshal(sleep)
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	if m["afterBackup"] != "sleep" || m["afterBackupWait"] != float64(10) || m["afterBackupOnlyIfWoken"] != true {
		t.Errorf("written as %v %v %v", m["afterBackup"], m["afterBackupWait"], m["afterBackupOnlyIfWoken"])
	}
	if _, set := m["shutdownAfter"]; set {
		t.Error("a set that sleeps is written with shutdownAfter")
	}

	// An older app that keeps the fields it does not know turned the
	// shutdown off: the set no longer shuts down.
	off := afterSet(afterBackupShutdown, 5, false)
	off.ShutdownAfter = false
	normalizeJobSchedules(&off)
	if off.AfterBackup != afterBackupStay || off.AfterBackupWait != 0 {
		t.Errorf("shutdownAfter turned off left %q after %d", off.AfterBackup, off.AfterBackupWait)
	}

	// "leave" is "stays as it is".
	leave := afterSet("leave", 0, false)
	normalizeJobSchedules(&leave)
	if leave.AfterBackup != afterBackupStay || leave.ShutdownAfter {
		t.Errorf("leave became %q", leave.AfterBackup)
	}
}

func TestAfterBackupIsValidated(t *testing.T) {
	for _, j := range []ScheduledJob{afterSet("reboot", 0, false), afterSet(afterBackupSleep, 241, false), afterSet(afterBackupSleep, -1, false)} {
		if err := validateScheduledJob(j); err == nil {
			t.Errorf("%q after %d accepted", j.AfterBackup, j.AfterBackupWait)
		}
	}
	for _, j := range []ScheduledJob{afterSet("", 0, false), afterSet(afterBackupHibernate, 240, true), afterSet(afterBackupShutdown, 0, false)} {
		if err := validateScheduledJob(j); err != nil {
			t.Errorf("%q after %d refused: %v", j.AfterBackup, j.AfterBackupWait, err)
		}
	}
}

func TestAfterBackupDecision(t *testing.T) {
	woken := time.Date(2026, 11, 16, 1, 58, 0, 0, time.Local)
	for _, c := range []struct {
		job   ScheduledJob
		woken time.Time
		want  string
		why   bool
	}{
		{afterSet("", 0, false), time.Time{}, "", false},
		{afterSet("", 0, true), woken, "", false},
		{afterSet(afterBackupSleep, 0, false), time.Time{}, afterBackupSleep, false},
		{afterSet(afterBackupHibernate, 30, false), woken, afterBackupHibernate, false},
		{afterSet(afterBackupShutdown, 0, false), time.Time{}, afterBackupShutdown, false},
		// only if this backup woke the computer: Run now, a run at startup or
		// one while the computer was awake are not woken for
		{afterSet(afterBackupSleep, 0, true), time.Time{}, "", true},
		{afterSet(afterBackupSleep, 0, true), woken, afterBackupSleep, false},
		{afterSet(afterBackupShutdown, 0, true), woken, afterBackupShutdown, false},
	} {
		got, why := afterBackupFor(c.job, c.woken)
		if got != c.want || (why != "") != c.why {
			t.Errorf("%q, only if woken %v, woken %v: %q (%q), want %q", c.job.AfterBackup, c.job.AfterBackupOnlyIfWoken, !c.woken.IsZero(), got, why, c.want)
		}
	}
}

// A run held back for the battery never started: nothing follows it.
func TestAfterBackupNotAfterAHeldRun(t *testing.T) {
	at := newAfterBackupTest(t)
	a := &App{}
	job := afterSet(afterBackupSleep, 0, false)
	job.OnBattery = "plugged"
	at.now = onBatteryAt(60)
	a.executeScheduledJob(job, "scheduled")
	if h := historyOf(t, a); len(h) != 1 || h[0].Status != "held" {
		t.Fatalf("want one held entry, got %+v", h)
	}
	if n := len(listAfterBackups()); n != 0 {
		t.Fatalf("%d actions waiting after a held run", n)
	}
	if got := at.actedWithin(100 * time.Millisecond); got != "" {
		t.Fatalf("the computer did %q after a held run", got)
	}
}

// The action waits its minutes, then for the other backups, then acts once;
// while it waits it is listed for the window's notice.
func TestAfterBackupWaitsThenForOtherBackups(t *testing.T) {
	at := newAfterBackupTest(t)
	a := &App{}
	runningJobsMutex.Lock()
	runningJobs["other"] = true
	runningJobsMutex.Unlock()
	defer func() {
		runningJobsMutex.Lock()
		delete(runningJobs, "other")
		runningJobsMutex.Unlock()
	}()

	start := time.Now()
	a.runPostBackupActions(afterSet(afterBackupSleep, 2, false), true, "done", nil, &batteryRunInfo{})
	list := a.GetAfterBackupPending()
	if len(list) != 1 || list[0].Set != "Documents" || list[0].Action != afterBackupSleep || list[0].WaitingForOthers {
		t.Fatalf("listed %+v", list)
	}
	if got := at.actedWithin(150 * time.Millisecond); got != "" {
		t.Fatalf("the computer did %q while another backup was running", got)
	}
	if list = a.GetAfterBackupPending(); len(list) != 1 || !list[0].WaitingForOthers {
		t.Fatalf("after the wait, listed %+v; want it waiting for the other backups", list)
	}

	runningJobsMutex.Lock()
	delete(runningJobs, "other")
	runningJobsMutex.Unlock()
	if got := at.actedWithin(2 * time.Second); got != afterBackupSleep {
		t.Fatalf("did %q once the other backup ended, want sleep", got)
	}
	if time.Since(start) < 2*afterBackupMinute {
		t.Error("acted before its wait was over")
	}
	if n := len(a.GetAfterBackupPending()); n != 0 {
		t.Errorf("%d still listed after acting", n)
	}
	if got := at.actedWithin(50 * time.Millisecond); got != "" {
		t.Errorf("acted twice (%q)", got)
	}
}

// Cancel leaves the computer as it is; a second set's action is dropped once
// the first is carried out.
func TestAfterBackupCancelAndOneActionOnly(t *testing.T) {
	at := newAfterBackupTest(t)
	a := &App{}
	afterBackupMinute = time.Hour
	a.startAfterBackup(afterSet(afterBackupShutdown, 1, false), nil)
	list := a.GetAfterBackupPending()
	if len(list) != 1 {
		t.Fatalf("listed %+v", list)
	}
	if err := a.CancelAfterBackup(list[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := a.CancelAfterBackup(list[0].ID); err == nil {
		t.Error("cancelled twice")
	}
	if n := len(a.GetAfterBackupPending()); n != 0 {
		t.Errorf("%d listed after Cancel", n)
	}

	afterBackupMinute = 20 * time.Millisecond
	other := afterSet(afterBackupHibernate, 50, false)
	other.ID, other.Name = "photos", "Photos"
	a.startAfterBackup(other, nil)
	woken := &batteryRunInfo{wokenAt: time.Now()}
	a.startAfterBackup(afterSet(afterBackupSleep, 0, true), woken)
	if got := at.actedWithin(2 * time.Second); got != afterBackupSleep {
		t.Fatalf("did %q, want sleep", got)
	}
	if n := len(a.GetAfterBackupPending()); n != 0 {
		t.Errorf("%d still waiting after the computer went to sleep", n)
	}
	if got := at.actedWithin(1500 * time.Millisecond); got != "" {
		t.Errorf("then also did %q", got)
	}
}

// A failure is logged, not fatal.
func TestAfterBackupFailureIsReported(t *testing.T) {
	newAfterBackupTest(t)
	a := &App{}
	sleepComputer = func() error { return errors.New("refused") }
	a.carryOutAfterBackup(afterSet(afterBackupSleep, 0, false), afterBackupSleep)
	log, err := a.GetMessageLog()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range log {
		if e.MessageKey == MsgLogAfterBackupFailed && e.MessageParams["error"] == "refused" {
			found = true
		}
	}
	if !found {
		t.Errorf("no failure line in %+v", log)
	}
}

// Before sleep and hibernation the system's wake alarm is brought in line with
// the wake timer, before the computer is told to go (on Linux the kernel does
// not set the hardware clock for hibernate).
func TestAfterBackupSyncsWakeAlarmFirst(t *testing.T) {
	newAfterBackupTest(t)
	oldSync := syncWakeAlarm
	t.Cleanup(func() { syncWakeAlarm = oldSync })
	for _, action := range []string{afterBackupSleep, afterBackupHibernate} {
		var steps []string
		syncWakeAlarm = func() { steps = append(steps, "sync") }
		sleepComputer = func() error { steps = append(steps, afterBackupSleep); return nil }
		hibernateComputer = func() error { steps = append(steps, afterBackupHibernate); return nil }
		(&App{}).carryOutAfterBackup(afterSet(action, 0, false), action)
		if len(steps) != 2 || steps[0] != "sync" || steps[1] != action {
			t.Errorf("%s: steps %v, want [sync %s]", action, steps, action)
		}
	}
}
