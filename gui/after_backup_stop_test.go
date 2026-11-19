package main

import (
	"testing"
	"time"
)

// Found on a laptop 2026-11-19: Stop pressed during a run the computer was
// taken as woken for, and the set's "hibernate, only if woken" hibernated it
// under the person who had pressed Stop. A run stopped by hand leaves the
// computer as it is; one the battery rule cancelled still goes on to the
// set's action, and so does a run that failed.
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
