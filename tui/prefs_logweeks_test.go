package tui

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

// The weeks of run logs kept come from the service's settings; a service
// from before the setting sends none and keeps 52.
func TestPrefsLogWeeksFromConfig(t *testing.T) {
	if got := prefsFromConfig(map[string]interface{}{}).LogWeeks; got != 52 {
		t.Errorf("without the setting: %d, want 52", got)
	}
	if got := prefsFromConfig(map[string]interface{}{"log_retention_weeks": float64(0)}).LogWeeks; got != 0 {
		t.Errorf("keep all: %d, want 0", got)
	}
	if got := prefsFromConfig(map[string]interface{}{"log_retention_weeks": float64(13)}).LogWeeks; got != 13 {
		t.Errorf("13 weeks: %d", got)
	}
}

// The Backup options page shows the setting and saves a change.
func TestPreferencesSaveLogWeeks(t *testing.T) {
	h := newHarness(t, 130, 44)
	h.sim.InjectKey(tcell.KeyF9, 0, tcell.ModNone)
	h.waitFor("Keep run logs, weeks (0 = all)")
	// the preferences are read: the field shows the demo's 52
	deadline := time.Now().Add(5 * time.Second)
	for {
		var got string
		h.onUI(func() { got = h.a.pf.logWeeks.GetText() })
		if got == "52" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the field shows %q, want 52", got)
		}
		time.Sleep(50 * time.Millisecond)
	}
	h.onUI(func() { h.a.pf.logWeeks.SetText("0") })
	h.onUI(h.a.pf.save)
	h.waitFor("preferences saved")
	if p, err := h.a.be.Prefs(); err != nil || p.LogWeeks != 0 {
		t.Errorf("saved: %+v %v", p.LogWeeks, err)
	}
}
