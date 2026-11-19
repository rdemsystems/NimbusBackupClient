package main

import (
	"testing"
	"time"
)

// schedulerOn marks the scheduler as running in this process for one test,
// as it is in the service.
func schedulerOn(t *testing.T, on bool) {
	t.Helper()
	schedulerMu.Lock()
	old := schedulerRunning
	schedulerRunning = on
	schedulerMu.Unlock()
	t.Cleanup(func() {
		schedulerMu.Lock()
		schedulerRunning = old
		schedulerMu.Unlock()
	})
}

func wakingSet(at string) ScheduledJob {
	return ScheduledJob{ID: "nl", Name: "New Laptop", BackupID: "new-laptop", BackupType: "directory",
		BackupDirs: []string{`C:\Data`}, ScheduleMode: "scheduled", WakeToRun: true,
		Schedules: []Schedule{{ID: "s1", Enabled: true, Unit: "days", Every: 1, At: at}}}
}

// lastArmed is the wake time set last, zero for none.
func lastArmed(f *fakeWake) time.Time {
	if len(f.armed) == 0 {
		return time.Time{}
	}
	return f.armed[len(f.armed)-1]
}

// Found on a laptop 2026-11-19, three times: a set saved with a new time and
// the computer shut down within the minute kept the old wake time, as the
// timer was moved only at the scheduler's next tick. Saving, changing,
// turning off and deleting a set now moves or drops the timer at once.
func TestSavingASetSetsItsWakeTimerAtOnce(t *testing.T) {
	deployTestDir(t)
	var f fakeWake
	f.install(t)
	schedulerOn(t, true)
	a := &App{}

	job := wakingSet("03:00")
	if err := a.SaveScheduledJob(job); err != nil {
		t.Fatal(err)
	}
	jobs, _ := a.GetScheduledJobs()
	run, err := time.Parse(time.RFC3339, jobs[0].NextRun)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.armed) != 1 || !lastArmed(&f).Equal(run.Add(-wakeLead)) {
		t.Fatalf("after saving: armed %v, want one timer at %s", f.armed, run.Add(-wakeLead))
	}

	job.Schedules[0].At = "04:30"
	if err := a.UpdateScheduledJob(job); err != nil {
		t.Fatal(err)
	}
	jobs, _ = a.GetScheduledJobs()
	run, _ = time.Parse(time.RFC3339, jobs[0].NextRun)
	if len(f.armed) != 2 || f.canceled != 1 || !lastArmed(&f).Equal(run.Add(-wakeLead)) {
		t.Fatalf("after changing the time: armed %v, cancelled %d, want the timer moved to %s", f.armed, f.canceled, run.Add(-wakeLead))
	}

	if err := a.SetScheduledJobEnabled("nl", false); err != nil {
		t.Fatal(err)
	}
	if f.canceled != 2 || wakeState.armed {
		t.Fatalf("after turning the set off: cancelled %d, still armed %v", f.canceled, wakeState.armed)
	}
	if err := a.SetScheduledJobEnabled("nl", true); err != nil {
		t.Fatal(err)
	}
	if len(f.armed) != 3 || !wakeState.armed {
		t.Fatalf("after turning it on again: armed %v", f.armed)
	}
	if err := a.DeleteScheduledJob("nl"); err != nil {
		t.Fatal(err)
	}
	if f.canceled != 3 || wakeState.armed {
		t.Fatalf("after deleting the set: cancelled %d, still armed %v", f.canceled, wakeState.armed)
	}
}

// Where the scheduler does not run in this process (the app, with the
// service doing the scheduling), saving a set sets no timer of its own.
func TestSavingASetWithoutTheSchedulerSetsNoTimer(t *testing.T) {
	deployTestDir(t)
	var f fakeWake
	f.install(t)
	schedulerOn(t, false)
	if err := (&App{}).SaveScheduledJob(wakingSet("03:00")); err != nil {
		t.Fatal(err)
	}
	if len(f.armed) != 0 {
		t.Fatalf("a timer was set where the scheduler does not run: %v", f.armed)
	}
}
