//go:build !service

package main

import "testing"

// An app working through the service has the service turn wake timers on
// (the app may not be elevated), and the service logs it. The TUI reads the
// alert from the service too.
func TestEnableWakeTimersThroughService(t *testing.T) {
	plan := &fakeWakePlan{ac: 1, dc: 0}
	installFakeWakePlan(t, plan, nil)
	p := newStrictPair(t)
	// Both ends live in this one process: without this the service's Message
	// Log line would be sent back to itself over HTTP, as the app's are.
	messageSink.Store((func(MessageLogEntry) error)(nil))

	var seen WakeTimerInfo
	if err := p.client.Call("GetWakeTimerInfo", &seen); err != nil {
		t.Fatal(err)
	}
	if seen.Alert != wakeAlertBattery {
		t.Errorf("the service's view: %+v, want the battery alert", seen)
	}
	info, err := p.app.EnableWakeTimers(wakeAlertBattery)
	if err != nil {
		t.Fatal(err)
	}
	if info.Alert != "" || !info.OnBattery || plan.dc != 1 {
		t.Errorf("answer %+v, plan dc %d", info, plan.dc)
	}
	p.saw(t, "POST /call/GetWakeTimerInfo", "POST /call/EnableWakeTimers")
	p.service(func(s *App) {
		if got := wakeLogLines(t, s); len(got) != 1 || got[0].MessageKey != MsgWakeTimersEnabledBattery {
			t.Errorf("the service's Message Log: %+v", got)
		}
	})
}
