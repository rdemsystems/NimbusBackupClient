package main

import (
	"errors"
	"testing"
	"time"
)

// Tests never read this computer's own event log: a wake there could be
// anything. Each test that needs a record of a wake sets one.
func init() {
	latestWakeEvent = func() (wakeEvent, error) { return wakeEvent{}, errNoWakeRecord }
}

// fakeWakeRecords stands in for the system's record of wakes, and for the
// clock while looking for one.
func fakeWakeRecords(t *testing.T, events ...func() (wakeEvent, error)) *int {
	t.Helper()
	calls := 0
	oldEvent, oldNow, oldSleep := latestWakeEvent, wakeSourceNow, wakeSourceSleep
	clock := time.Date(2026, 11, 19, 22, 27, 50, 0, time.UTC)
	latestWakeEvent = func() (wakeEvent, error) {
		i := calls
		calls++
		if i >= len(events) {
			i = len(events) - 1
		}
		return events[i]()
	}
	wakeSourceNow = func() time.Time { return clock }
	wakeSourceSleep = func(d time.Duration) { clock = clock.Add(d) }
	t.Cleanup(func() { latestWakeEvent, wakeSourceNow, wakeSourceSleep = oldEvent, oldNow, oldSleep })
	return &calls
}

func record(ev wakeEvent) func() (wakeEvent, error) {
	return func() (wakeEvent, error) { return ev, nil }
}

// As DMXCONTROL recorded a wake by the app's timer (2026-11-18 11:20), and
// as a laptop recorded the power button.
const timerWakeXML = `<Event xmlns='http://schemas.microsoft.com/win/2004/08/events/event'><System><Provider Name='Microsoft-Windows-Power-Troubleshooter'/><EventID>1</EventID></System><EventData>
<Data Name='SleepTime'>2026-11-18T11:14:41.7616351Z</Data><Data Name='WakeTime'>2026-11-18T11:20:00.5035631Z</Data>
<Data Name='TargetState'>4</Data><Data Name='WakeSourceType'>6</Data><Data Name='WakeSourceTextLength'>16</Data>
<Data Name='WakeSourceText'>salvanza-svc.exe</Data><Data Name='WakeTimerOwnerLength'>63</Data>
<Data Name='WakeTimerOwner'>\Device\HarddiskVolume2\Program Files\Salvanza\salvanza-svc.exe</Data><Data Name='WakeTimerContext'></Data></EventData></Event>`

const buttonWakeXML = `<Event xmlns='http://schemas.microsoft.com/win/2004/08/events/event'><EventData>
<Data Name='WakeTime'>2026-11-15T16:06:17.0874216Z</Data><Data Name='WakeSourceType'>1</Data>
<Data Name='WakeSourceText'></Data><Data Name='WakeTimerOwner'></Data></EventData></Event>`

func TestParseWakeEventXML(t *testing.T) {
	ev, err := parseWakeEventXML([]byte(timerWakeXML))
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 11, 18, 11, 20, 0, 503563100, time.UTC); !ev.WakeTime.Equal(want) || ev.SourceType != 6 ||
		ev.SourceText != "salvanza-svc.exe" || baseName(ev.TimerOwner) != "salvanza-svc.exe" {
		t.Fatalf("timer wake read as %+v", ev)
	}
	if v := judgeWake(ev, "SALVANZA-SVC.EXE"); !v.known || !v.ours {
		t.Errorf("the app's own timer wake judged %+v", v)
	}

	ev, err = parseWakeEventXML([]byte(buttonWakeXML))
	if err != nil {
		t.Fatal(err)
	}
	if v := judgeWake(ev, "salvanza-svc.exe"); !v.known || v.ours || v.what != "the power button" {
		t.Errorf("a power button wake judged %+v", v)
	}
	if _, err := parseWakeEventXML([]byte(`<Event><EventData></EventData></Event>`)); err == nil {
		t.Error("an event with no wake time was read")
	}
}

func TestOnlyOurOwnTimerCountsAsOurWake(t *testing.T) {
	for _, c := range []struct {
		what string
		ev   wakeEvent
		ours bool
	}{
		{"the app's timer", wakeEvent{SourceType: 6, SourceText: "salvanza-svc.exe", TimerOwner: `\Device\HarddiskVolume3\Program Files\Salvanza\salvanza-svc.exe`}, true},
		{"no owner path: the text names it", wakeEvent{SourceType: 6, SourceText: "salvanza-svc.exe"}, true},
		{"Windows Update's timer", wakeEvent{SourceType: 6, SourceText: "MoUsoCoreWorker.exe"}, false},
		{"a Fast Startup start by hand", wakeEvent{SourceType: 0}, false},
		{"the power button", wakeEvent{SourceType: 1}, false},
		{"not a timer at all", wakeEvent{SourceType: 5, SourceText: "salvanza-svc.exe"}, false},
		{"another file", wakeEvent{SourceType: 6, TimerOwner: `C:\Other\salvanza-svc.exe.bak`}, false},
	} {
		if v := judgeWake(c.ev, "salvanza-svc.exe"); v.ours != c.ours || !v.known {
			t.Errorf("%s: judged %+v, want ours=%v", c.what, v, c.ours)
		}
	}
	if judgeWake(wakeEvent{SourceType: 6, SourceText: "salvanza-svc.exe"}, "").ours {
		t.Error("a wake counted as ours with our own name unknown")
	}
}

// Windows writes the record a few seconds after the wake: the newest record
// is the last sleep's until then, and is looked for again.
func TestTheRecordOfAWakeIsWaitedFor(t *testing.T) {
	asleep := time.Date(2026, 11, 19, 22, 24, 11, 0, time.UTC)
	calls := fakeWakeRecords(t,
		record(wakeEvent{WakeTime: asleep.Add(-time.Hour), SourceType: 6, SourceText: "salvanza-svc.exe"}),
		record(wakeEvent{WakeTime: asleep.Add(-time.Hour), SourceType: 6, SourceText: "salvanza-svc.exe"}),
		record(wakeEvent{WakeTime: asleep.Add(3*time.Minute + 39*time.Second), SourceType: 0}),
	)
	v := whatWokeTheComputer(asleep)
	if !v.known || v.ours || *calls != 3 {
		t.Fatalf("judged %+v after %d looks, want the third record: not ours", v, *calls)
	}
}

// No record within the wait, or none kept at all: the wake is not known, and
// is taken as the timer's as before.
func TestNoRecordOfAWakeIsNotAVerdict(t *testing.T) {
	asleep := time.Date(2026, 11, 19, 22, 24, 11, 0, time.UTC)
	calls := fakeWakeRecords(t, func() (wakeEvent, error) { return wakeEvent{}, errors.New("EvtQuery: access denied") })
	if v := whatWokeTheComputer(asleep); v.known {
		t.Fatalf("judged %+v with no record", v)
	}
	if want := int(wakeSourceWait/wakeSourceEvery) + 1; *calls != want {
		t.Errorf("looked %d times, want %d (every %s for %s)", *calls, want, wakeSourceEvery, wakeSourceWait)
	}
	calls = fakeWakeRecords(t, func() (wakeEvent, error) { return wakeEvent{}, errNoWakeRecord })
	if v := whatWokeTheComputer(asleep); v.known || *calls != 1 {
		t.Fatalf("a system with no records was asked %d times, judged %+v", *calls, v)
	}
}

// Found on a laptop 2026-11-19: it did not wake from a Fast Startup
// shutdown, was switched on with the power button 2m50s after its wake timer
// was due, and was taken as woken for the set. Windows' record says the
// power button (here: a source it could not tell), so it is not; a record
// that names the app's own timer is.
func TestAStartByHandInTheTimersWindowIsNotItsWake(t *testing.T) {
	newBatteryTest(t)
	var held []string
	old := holdAwakeAfterWake
	holdAwakeAfterWake = func(setID, _ string) { held = append(held, setID) }
	t.Cleanup(func() { holdAwakeAfterWake = old })
	run := time.Date(2026, 11, 19, 22, 27, 0, 0, time.Local)
	wakeState.mu.Lock()
	wakeState.target = wakeTarget{At: run.Add(-wakeLead), Run: run, SetID: "nl", Set: "New Laptop"}
	wakeState.armed = true
	wakeState.mu.Unlock()
	asleep := time.Date(2026, 11, 19, 22, 24, 11, 0, time.Local)
	woke := time.Date(2026, 11, 19, 22, 27, 50, 0, time.Local)

	fakeWakeRecords(t, record(wakeEvent{WakeTime: woke, SourceType: 0}))
	noteSchedulerTick(asleep, woke)
	if len(held) != 0 {
		t.Fatalf("a start by hand was taken as the timer's wake: %v", held)
	}
	if !takeWoken(ScheduledJob{ID: "nl"}, "scheduled", woke).IsZero() {
		t.Fatal("the set's run was told the computer was woken for it")
	}

	fakeWakeRecords(t, record(wakeEvent{WakeTime: woke, SourceType: wakeSourceTimer, SourceText: ownExeName()}))
	noteSchedulerTick(asleep, woke)
	if len(held) != 1 || held[0] != "nl" {
		t.Fatalf("the app's own timer wake was missed: %v", held)
	}
}
