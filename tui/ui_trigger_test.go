package tui

import "testing"

// A set's server trigger and a schedule that waits for its server show in
// the "When to run" column.
func TestScheduleTextTriggers(t *testing.T) {
	daily := map[string]interface{}{"id": "s1", "enabled": true, "unit": "days", "every": 1.0, "at": "22:00", "ifUnreachable": "wait"}
	m := map[string]interface{}{
		"scheduleMode":  "scheduled",
		"schedules":     []interface{}{daily},
		"serverTrigger": map[string]interface{}{"enabled": true, "everyHours": 12.0},
	}
	if got, want := scheduleText(m), "daily 22:00 (waits for the server), when the server answers, at most every 12h"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// Only the server trigger on: it alone, not "all schedules off".
	daily["enabled"] = false
	if got, want := scheduleText(m), "when the server answers, at most every 12h"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// A trigger turned off counts for nothing.
	m["serverTrigger"] = map[string]interface{}{"enabled": false, "everyHours": 12.0}
	if got, want := scheduleText(m), "all schedules off"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
