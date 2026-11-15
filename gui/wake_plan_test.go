package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"clientcommon"
)

// fakeWakePlan stands in for the active power plan: its Allow wake timers
// values, what it refuses, and what was done to it.
type fakeWakePlan struct {
	ac, dc  uint32
	refuse  error // every Write and Activate fails with it
	held    bool  // writes succeed but change nothing, as under a policy
	readErr error
	did     []string
	closed  bool
}

func (p *fakeWakePlan) Name() string {
	return "Balanced"
}

func (p *fakeWakePlan) Read() (ac, dc uint32, err error) {
	return p.ac, p.dc, p.readErr
}

func (p *fakeWakePlan) Close() {
	p.closed = true
}

func (p *fakeWakePlan) Write(onMains bool, v uint32) error {
	if p.refuse != nil {
		return p.refuse
	}
	p.did = append(p.did, wakeStep(onMains, v))
	if !p.held {
		p.set(onMains, v)
	}
	return nil
}

func (p *fakeWakePlan) set(onMains bool, v uint32) {
	if onMains {
		p.ac = v
	} else {
		p.dc = v
	}
}

func (p *fakeWakePlan) Activate() error {
	if p.refuse != nil {
		return p.refuse
	}
	p.did = append(p.did, "activate")
	return nil
}

func wakeStep(onMains bool, v uint32) string {
	if onMains {
		return fmt.Sprintf("ac=%d", v)
	}
	return fmt.Sprintf("dc=%d", v)
}

// fakeWakeFallback writes the same plan another way, as powercfg does.
type fakeWakeFallback struct {
	plan   *fakeWakePlan
	refuse error
	did    []string
}

func (f *fakeWakeFallback) Write(onMains bool, v uint32) error {
	if f.refuse != nil {
		return f.refuse
	}
	f.did = append(f.did, wakeStep(onMains, v))
	f.plan.set(onMains, v)
	return nil
}

func (f *fakeWakeFallback) Activate() error {
	if f.refuse != nil {
		return f.refuse
	}
	f.did = append(f.did, "activate")
	return nil
}

// installFakeWakePlan makes plan the computer's active power plan, on a
// Windows laptop, for the rest of the test: the window's view of it comes
// from its values, as the system's would.
func installFakeWakePlan(t *testing.T, plan *fakeWakePlan, fallback wakePlanWriter) {
	t.Helper()
	oldOpen, oldInfo, oldGOOS, oldBattery := openWakePlan, readWakeTimerInfo, wakeGOOS, readBattery
	openWakePlan = func() (wakePlan, wakePlanWriter, error) { plan.closed = false; return plan, fallback, nil }
	readWakeTimerInfo = func() WakeTimerInfo {
		return WakeTimerInfo{Supported: true, OnMains: plan.ac == 1, OnBattery: plan.dc == 1, planRead: true}
	}
	wakeGOOS = "windows"
	readBattery = func() clientcommon.BatteryStatus {
		return clientcommon.BatteryStatus{HasBattery: true, Percent: 80, Known: true}
	}
	t.Cleanup(func() { openWakePlan, readWakeTimerInfo, wakeGOOS, readBattery = oldOpen, oldInfo, oldGOOS, oldBattery })
}

// Which alert, if any, goes under the Wake option.
func TestWakeAlertFor(t *testing.T) {
	read := func(ac, dc uint32, battery bool) WakeTimerInfo {
		return WakeTimerInfo{Supported: true, OnMains: ac == 1, OnBattery: dc == 1, HasBattery: battery, planRead: true}
	}
	assumed := read(1, 1, true)
	assumed.planRead = false
	assumed.OnBattery = false
	unsupported := read(0, 0, true)
	unsupported.Supported = false
	cases := []struct {
		name string
		goos string
		info WakeTimerInfo
		want string
	}{
		{"on everywhere", "windows", read(1, 1, true), ""},
		{"off on battery, laptop", "windows", read(1, 0, true), wakeAlertBattery},
		{"important only on battery, laptop", "windows", read(1, 2, true), wakeAlertBattery},
		{"off on battery, desktop", "windows", read(1, 0, false), ""},
		{"off on mains and battery, laptop", "windows", read(0, 0, true), wakeAlertAll},
		{"off on mains, desktop", "windows", read(0, 0, false), wakeAlertAll},
		{"important only on mains", "windows", read(2, 1, true), wakeAlertAll},
		{"setting not read", "windows", assumed, ""},
		{"no sleep a timer can wake from", "windows", unsupported, ""},
		{"linux", "linux", read(0, 0, true), ""},
		{"other system", "darwin", read(1, 0, true), ""},
	}
	for _, c := range cases {
		if got := wakeAlertFor(c.goos, c.info); got != c.want {
			t.Errorf("%s: alert %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSetWakeTimersOnBatteryOnly(t *testing.T) {
	plan := &fakeWakePlan{ac: 1, dc: 0}
	if err := setWakeTimers(plan, nil, false); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(plan.did, " "); got != "dc=1 activate" {
		t.Errorf("did %q, want the battery value written and the plan applied", got)
	}
	if plan.ac != 1 || plan.dc != 1 {
		t.Errorf("plan now ac %d dc %d", plan.ac, plan.dc)
	}
}

func TestSetWakeTimersOnMainsAndBattery(t *testing.T) {
	plan := &fakeWakePlan{ac: 0, dc: 2}
	if err := setWakeTimers(plan, nil, true); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(plan.did, " "); got != "ac=1 dc=1 activate" {
		t.Errorf("did %q", got)
	}
}

// A call the system refuses is done by the fallback (powercfg) instead.
func TestSetWakeTimersFallsBack(t *testing.T) {
	plan := &fakeWakePlan{ac: 1, dc: 0, refuse: errors.New("access denied")}
	fb := &fakeWakeFallback{plan: plan}
	if err := setWakeTimers(plan, fb, false); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(fb.did, " "); got != "dc=1 activate" {
		t.Errorf("the fallback did %q", got)
	}
	if plan.dc != 1 {
		t.Errorf("battery value %d after the fallback", plan.dc)
	}
}

func TestSetWakeTimersReportsBothFailures(t *testing.T) {
	plan := &fakeWakePlan{ac: 1, dc: 0, refuse: errors.New("access denied")}
	err := setWakeTimers(plan, &fakeWakeFallback{plan: plan, refuse: errors.New("powercfg exit 1")}, false)
	if err == nil || !strings.Contains(err.Error(), "access denied") || !strings.Contains(err.Error(), "powercfg exit 1") {
		t.Errorf("error %v, want both reasons", err)
	}
	if plan.dc != 0 {
		t.Errorf("the plan changed although every write failed")
	}
}

// A change that does not take (a policy holds the setting) is an error.
func TestSetWakeTimersChecksTheChangeTook(t *testing.T) {
	plan := &fakeWakePlan{ac: 1, dc: 0, held: true}
	if err := setWakeTimers(plan, nil, false); err == nil || !strings.Contains(err.Error(), "still does not allow") {
		t.Errorf("error %v, want one saying the setting did not change", err)
	}
	plan = &fakeWakePlan{ac: 1, dc: 0, readErr: errors.New("unreadable")}
	if err := setWakeTimers(plan, nil, false); err == nil || !strings.Contains(err.Error(), "unreadable") {
		t.Errorf("error %v, want the read back failure", err)
	}
}

// Pressing the button: the plan changes, the Message Log says so, and the
// answer has no alert left.
func TestEnableWakeTimersOnBattery(t *testing.T) {
	useTempConfigDir(t)
	plan := &fakeWakePlan{ac: 1, dc: 0}
	installFakeWakePlan(t, plan, nil)
	a := &App{}
	if got := a.GetWakeTimerInfo(); got.Alert != wakeAlertBattery || !got.HasBattery {
		t.Fatalf("before: %+v, want the battery alert", got)
	}
	info, err := a.EnableWakeTimers(wakeAlertBattery)
	if err != nil {
		t.Fatal(err)
	}
	if !info.OnBattery || !info.OnMains || info.Alert != "" {
		t.Errorf("after: %+v, want wake timers on and no alert", info)
	}
	if !plan.closed {
		t.Error("the plan was not closed")
	}
	if got := wakeLogLines(t, a); len(got) != 1 || got[0].MessageKey != MsgWakeTimersEnabledBattery || got[0].MessageParams["plan"] != "Balanced" ||
		got[0].Message != "Wake timers on battery were enabled in the power plan 'Balanced' at the user's request." {
		t.Errorf("Message Log: %+v", got)
	}
}

func TestEnableWakeTimersEverywhere(t *testing.T) {
	useTempConfigDir(t)
	plan := &fakeWakePlan{ac: 0, dc: 0}
	installFakeWakePlan(t, plan, nil)
	a := &App{}
	if got := a.GetWakeTimerInfo(); got.Alert != wakeAlertAll {
		t.Fatalf("before: %+v, want the alert for wake timers off", got)
	}
	info, err := a.EnableWakeTimers(wakeAlertAll)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ac != 1 || plan.dc != 1 || info.Alert != "" {
		t.Errorf("plan ac %d dc %d, answer %+v", plan.ac, plan.dc, info)
	}
	if got := wakeLogLines(t, a); len(got) != 1 || got[0].MessageKey != MsgWakeTimersEnabled {
		t.Errorf("Message Log: %+v", got)
	}
}

// Nothing is changed or logged for an unknown scope or a change that fails,
// and the reason comes back for the alert to show.
func TestEnableWakeTimersRefusals(t *testing.T) {
	useTempConfigDir(t)
	plan := &fakeWakePlan{ac: 1, dc: 0}
	installFakeWakePlan(t, plan, nil)
	a := &App{}
	if _, err := a.EnableWakeTimers("sideways"); err == nil {
		t.Error("an unknown scope was accepted")
	}
	if len(plan.did) != 0 {
		t.Errorf("the plan was touched: %v", plan.did)
	}
	plan.refuse = errors.New("access denied")
	if _, err := a.EnableWakeTimers(wakeAlertBattery); err == nil || !strings.Contains(err.Error(), "access denied") {
		t.Errorf("error %v, want the reason", err)
	}
	if got := wakeLogLines(t, a); len(got) != 0 {
		t.Errorf("logged although nothing changed: %+v", got)
	}
}

// wakeLogLines are the Message Log's lines about wake timers.
func wakeLogLines(t *testing.T, a *App) []MessageLogEntry {
	t.Helper()
	log, err := a.GetMessageLog()
	if err != nil {
		t.Fatal(err)
	}
	var out []MessageLogEntry
	for _, e := range log {
		if e.MessageKey == MsgWakeTimersEnabled || e.MessageKey == MsgWakeTimersEnabledBattery {
			out = append(out, e)
		}
	}
	return out
}
