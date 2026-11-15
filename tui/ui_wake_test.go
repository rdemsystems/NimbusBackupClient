package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// openScheduleStep opens the first set (Documents) at its Schedule step.
func openScheduleStep(h *harness) {
	h.key(tcell.KeyEnter, 0)
	h.waitFor("Edit Backup Set")
	h.key(tcell.KeyPgDn, 0)
	h.waitFor("Repeat every")
}

// With a power plan that does not let timers wake a laptop on battery, the
// Schedule step says so while Wake is ticked, and its button has the service
// turn wake timers on.
func TestEditorOffersToEnableWakeTimers(t *testing.T) {
	be := newDemoBackend()
	be.wake = WakeInfo{OnMains: true, Alert: "battery"}
	h := newHarnessWith(t, be, 130, 44)
	openScheduleStep(h)
	if !strings.Contains(h.screen(), "Enable wake timers on battery") {
		t.Fatalf("no button to enable wake timers:\n%s", h.screen())
	}
	if strings.Contains(h.screen(), "This computer's power") {
		t.Errorf("the alert shows while Wake is not ticked:\n%s", h.screen())
	}
	// down to Wake: unit, every, at, days, wake
	for i := 0; i < 5; i++ {
		h.key(tcell.KeyTab, 0)
	}
	h.typ(' ')
	h.waitFor("This computer's power")
	// on to the button: battery, start, cancel, below (the texts are passed
	// over), then the first button
	for i := 0; i < 5; i++ {
		h.key(tcell.KeyTab, 0)
	}
	h.key(tcell.KeyEnter, 0)
	h.waitFor("wake timers turned on")
	be.mu.Lock()
	asked := strings.Join(be.wakeEnabled, ",")
	be.mu.Unlock()
	if asked != "battery" {
		t.Errorf("EnableWakeTimers asked for %q, want battery", asked)
	}
	s := h.screen()
	if strings.Contains(s, "Enable wake timers on battery") || strings.Contains(s, "This computer's power") {
		t.Errorf("the alert stays after wake timers were turned on:\n%s", s)
	}
}

// No alert where the power plan lets timers wake the computer.
func TestEditorHasNoWakeAlertWhenTimersMayWake(t *testing.T) {
	h := newHarness(t, 130, 44)
	openScheduleStep(h)
	if strings.Contains(h.screen(), "Enable wake timers") {
		t.Errorf("a wake timer button with nothing to enable:\n%s", h.screen())
	}
}
