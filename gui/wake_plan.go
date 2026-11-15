package main

import (
	"fmt"
	"runtime"
)

// Turning on "Allow wake timers" in the power plan, at the user's request
// (#140). A Backup Set that wakes the computer does nothing while the power
// plan does not let timers wake it, and many laptops come with them off on
// battery, or allowed for "important" (the system's own) timers only. So the
// app asks, under the set's Wake option, whether to turn them on: nothing
// changes unless the user presses the button that does. The service makes the
// change (the plan belongs to the computer, and the app may not be elevated),
// reads it back to be sure it took and notes it in the Message Log.

// The alerts: the plan does not let a timer wake the computer at all, or not
// while it runs on battery.
const (
	wakeAlertAll     = "all"
	wakeAlertBattery = "battery"
)

// wakePlanWriter changes Allow wake timers in the active power plan.
type wakePlanWriter interface {
	// Write sets it on mains (onMains) or on battery to value (1 is on).
	Write(onMains bool, value uint32) error
	// Activate applies the plan again, so that what was written takes effect.
	Activate() error
}

// wakePlan is the active power plan as the write path sees it: the system's
// own calls on Windows, a fake in the tests.
type wakePlan interface {
	wakePlanWriter
	// Name is the plan's name as Power Options shows it.
	Name() string
	// Read reads Allow wake timers on mains (ac) and on battery (dc).
	Read() (ac, dc uint32, err error)
	Close()
}

// openWakePlan opens the active power plan, with a second way of writing it
// to fall back on (powercfg on Windows; nil when there is none). Variables
// so the tests can stand in for the system.
var (
	openWakePlan      = platformOpenWakePlan
	readWakeTimerInfo = platformWakeTimerInfo
	wakeGOOS          = runtime.GOOS
)

// wakeTimerInfoNow is what the window shows under the Wake option, with the
// alert to put there, if any.
func wakeTimerInfoNow() WakeTimerInfo {
	info := readWakeTimerInfo()
	info.HasBattery = readBattery().HasBattery
	info.Alert = wakeAlertFor(wakeGOOS, info)
	return info
}

// wakeAlertFor is the alert for info on the system goos. "all": the power
// plan does not let a timer wake the computer at all, battery or not.
// "battery": it does on mains but not on battery, on a computer that has a
// battery (on a desktop the battery value never applies). "" otherwise. Only
// on Windows, the one system with such a setting, and only when the setting
// was read: where it is missing the system decides, and there is nothing for
// the user to turn on.
func wakeAlertFor(goos string, info WakeTimerInfo) string {
	if goos != "windows" || !info.Supported || !info.planRead {
		return ""
	}
	switch {
	case !info.OnMains:
		return wakeAlertAll
	case !info.OnBattery && info.HasBattery:
		return wakeAlertBattery
	}
	return ""
}

// EnableWakeTimers turns on Allow wake timers in the active power plan at the
// user's request, the first button of the alert under the Wake option: on
// battery for scope "battery", on mains and on battery for "all". It answers
// with what the computer allows now. With the service, the service does it:
// the app may not be elevated.
func (a *App) EnableWakeTimers(scope string) (WakeTimerInfo, error) {
	if a.delegated() {
		var info WakeTimerInfo
		err := a.apiClient.Call("EnableWakeTimers", &info, scope)
		if !isUnknownCall(err) {
			return info, err
		}
		// A service from before this call: try here, which works when the
		// app is elevated or the system lets the user change the plan.
	}
	return enableWakeTimers(scope)
}

// enableWakeTimers changes the plan for EnableWakeTimers, in this process.
func enableWakeTimers(scope string) (WakeTimerInfo, error) {
	if scope != wakeAlertBattery && scope != wakeAlertAll {
		return WakeTimerInfo{}, fmt.Errorf("wake timers: %q is neither %q nor %q", scope, wakeAlertBattery, wakeAlertAll)
	}
	plan, fallback, err := openWakePlan()
	if err != nil {
		return WakeTimerInfo{}, fmt.Errorf("the active power plan could not be opened: %w", err)
	}
	defer plan.Close()
	name := plan.Name()
	mains := scope == wakeAlertAll
	if err := setWakeTimers(plan, fallback, mains); err != nil {
		writeDebugLog(fmt.Sprintf("[Wake] turning on wake timers (%s) in the power plan %q: %v", scope, name, err))
		return WakeTimerInfo{}, err
	}
	if mains {
		LogMessage("App", "info", fmt.Sprintf("Wake timers were enabled in the power plan '%s' at the user's request.", name), "",
			MsgWakeTimersEnabled, msgParams{"plan": name})
	} else {
		LogMessage("App", "info", fmt.Sprintf("Wake timers on battery were enabled in the power plan '%s' at the user's request.", name), "",
			MsgWakeTimersEnabledBattery, msgParams{"plan": name})
	}
	writeDebugLog(fmt.Sprintf("[Wake] wake timers (%s) turned on in the power plan %q at the user's request", scope, name))
	return wakeTimerInfoNow(), nil
}

// setWakeTimers turns Allow wake timers on in plan: on battery, and on mains
// too when mains is set. A step the plan's own call refuses is tried again
// through fallback (when there is one). It reads the setting back and fails
// unless the change took: a policy set for the computer, for one, can hold it.
func setWakeTimers(plan wakePlan, fallback wakePlanWriter, mains bool) error {
	step := func(what string, do func(w wakePlanWriter) error) error {
		err := do(plan)
		if err == nil {
			return nil
		}
		if fallback == nil {
			return fmt.Errorf("%s: %w", what, err)
		}
		if err2 := do(fallback); err2 != nil {
			return fmt.Errorf("%s: %v; the fallback failed too: %w", what, err, err2)
		}
		writeDebugLog(fmt.Sprintf("[Wake] %s: %v; done by the fallback instead", what, err))
		return nil
	}
	if mains {
		if err := step("turning on wake timers on mains", func(w wakePlanWriter) error { return w.Write(true, 1) }); err != nil {
			return err
		}
	}
	if err := step("turning on wake timers on battery", func(w wakePlanWriter) error { return w.Write(false, 1) }); err != nil {
		return err
	}
	if err := step("applying the power plan", func(w wakePlanWriter) error { return w.Activate() }); err != nil {
		return err
	}
	ac, dc, err := plan.Read()
	if err != nil {
		return fmt.Errorf("reading the power plan back: %w", err)
	}
	if dc != 1 || (mains && ac != 1) {
		return fmt.Errorf("the power plan still does not allow wake timers (on mains %d, on battery %d, where 1 is on): a policy set for this computer may be holding the setting", ac, dc)
	}
	return nil
}
