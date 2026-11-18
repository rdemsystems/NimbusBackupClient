package tui

// WakeInfo is what the service says about waking the computer for a backup
// (the app's WakeTimerInfo). Alert is the question the app asks under a
// set's Wake option: "all" when the power plan does not let a timer wake the
// computer at all, "battery" when not on battery on a computer that has one,
// "" when there is nothing to ask (and always on a system other than
// Windows).
type WakeInfo struct {
	OnMains   bool   `json:"onMains"`
	OnBattery bool   `json:"onBattery"`
	Alert     string `json:"alert"`
	// ModernStandby: Windows on a computer with Modern Standby (S0 low power
	// idle), where waking for a backup is best effort (modernStandbyText).
	ModernStandby bool `json:"modernStandby"`
}

// modernStandbyText is the app's statement under Wake on a Modern Standby
// computer (wakeModernStandbyTitle, wakeModernStandby), shortened to fit
// beside the form's labels.
const modernStandbyText = "Modern Standby: waking for a backup is best effort. " +
	"Salvanza sends the wake, keep awake and sleep requests a backup needs, but Windows decides whether to honour them, " +
	"and often does not. The only reliable way is to turn Modern Standby off and use hibernate, which wakes on any computer " +
	"(manual: \"Waking the computer\")."

// wakeAlertText is the alert's text and the label of the button that turns
// wake timers on, as the app words them; "" for no alert.
func wakeAlertText(alert string) (text, button string) {
	switch alert {
	case "all":
		return "This computer's power plan does not let timers wake it at all, so the backup will not wake it. Enable wake timers?",
			"Enable wake timers"
	case "battery":
		return "This computer's power plan does not let timers wake it while on battery, so the backup will not wake it then. Enable wake timers on battery?",
			"Enable wake timers on battery"
	}
	return "", ""
}

func (b *serviceBackend) WakeTimers() (WakeInfo, error) {
	var out WakeInfo
	err := b.c.Call("GetWakeTimerInfo", &out)
	return out, err
}

// EnableWakeTimers has the service change the power plan: the window may not
// be elevated, and the plan belongs to the computer.
func (b *serviceBackend) EnableWakeTimers(scope string) (WakeInfo, error) {
	var out WakeInfo
	err := b.c.Call("EnableWakeTimers", &out, scope)
	return out, err
}

// The demo computer's power plan lets timers wake it (no alert) unless a
// test says otherwise (demoBackend.wake).
func (d *demoBackend) WakeTimers() (WakeInfo, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.wake, nil
}

func (d *demoBackend) EnableWakeTimers(scope string) (WakeInfo, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.wake.OnBattery = true
	if scope == "all" {
		d.wake.OnMains = true
	}
	d.wake.Alert = ""
	d.wakeEnabled = append(d.wakeEnabled, scope)
	return d.wake, nil
}
