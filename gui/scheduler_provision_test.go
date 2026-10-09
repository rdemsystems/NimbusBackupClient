package main

import (
	"testing"
	"time"
)

func TestSlugifyJobID(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Nightly C:", "provisioned-nightly-c"},
		{"  Backup  Users  ", "provisioned-backup-users"},
		{"Sauvegarde éà#1", "provisioned-sauvegarde-1"},
		{"", "job"},
		{"!!!", "job"},
		{"already-slug", "provisioned-already-slug"},
	}
	for _, c := range cases {
		if got := slugifyJobID(c.in); got != c.want {
			t.Errorf("slugifyJobID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMergeProvisionedJobs_AppendsAndComputesNextRun(t *testing.T) {
	prov := []ScheduledJob{
		{Name: "Nightly", ScheduleTime: "02:30", Enabled: true, BackupDirs: []string{"C:"}},
	}
	got := mergeProvisionedJobs(nil, prov)
	if len(got) != 1 {
		t.Fatalf("want 1 job, got %d", len(got))
	}
	j := got[0]
	if j.ID != "provisioned-nightly" {
		t.Errorf("ID not derived from name: %q", j.ID)
	}
	if !validRFC3339(j.NextRun) {
		t.Errorf("nextRun should be auto-computed for an enabled job, got %q", j.NextRun)
	}
}

func TestMergeProvisionedJobs_DisabledOrNoTimeGetsNoNextRun(t *testing.T) {
	prov := []ScheduledJob{
		{ID: "a", Name: "disabled", ScheduleTime: "02:30", Enabled: false},
		{ID: "b", Name: "no-time", Enabled: true},
	}
	got := mergeProvisionedJobs(nil, prov)
	for _, j := range got {
		if j.NextRun != "" {
			t.Errorf("job %q should have empty nextRun, got %q", j.ID, j.NextRun)
		}
	}
}

func TestMergeProvisionedJobs_UpsertPreservesHistoryAndNextRun(t *testing.T) {
	stored := time.Now().Add(6 * time.Hour).Format(time.RFC3339)
	existing := []ScheduledJob{
		{ID: "nightly", Name: "Nightly", ScheduleTime: "02:30", Enabled: true,
			NextRun: stored, LastRun: "2026-07-01T02:30:00Z", BackupID: "old"},
	}
	// Same ID, same schedule, changed backup-id: should replace fields but keep
	// LastRun and the still-valid NextRun (idempotent re-provisioning).
	prov := []ScheduledJob{
		{ID: "nightly", Name: "Nightly", ScheduleTime: "02:30", Enabled: true, BackupID: "new"},
	}
	got := mergeProvisionedJobs(existing, prov)
	if len(got) != 1 {
		t.Fatalf("upsert should not duplicate; got %d jobs", len(got))
	}
	if got[0].BackupID != "new" {
		t.Errorf("provisioned fields not applied: BackupID=%q", got[0].BackupID)
	}
	if got[0].LastRun != "2026-07-01T02:30:00Z" {
		t.Errorf("LastRun history not preserved: %q", got[0].LastRun)
	}
	if got[0].NextRun != stored {
		t.Errorf("valid NextRun should be preserved on unchanged schedule: got %q want %q", got[0].NextRun, stored)
	}
}

func TestMergeProvisionedJobs_ScheduleChangeRecomputesNextRun(t *testing.T) {
	stored := time.Now().Add(6 * time.Hour).Format(time.RFC3339)
	existing := []ScheduledJob{
		{ID: "nightly", Name: "Nightly", ScheduleTime: "02:30", Enabled: true, NextRun: stored},
	}
	prov := []ScheduledJob{
		{ID: "nightly", Name: "Nightly", ScheduleTime: "04:00", Enabled: true},
	}
	got := mergeProvisionedJobs(existing, prov)
	if got[0].NextRun == stored {
		t.Errorf("nextRun should be recomputed when schedule time changes")
	}
	if !validRFC3339(got[0].NextRun) {
		t.Errorf("recomputed nextRun invalid: %q", got[0].NextRun)
	}
}

func TestMergeProvisionedJobs_LeavesGUIJobsUntouched(t *testing.T) {
	existing := []ScheduledJob{
		{ID: "1720000000", Name: "GUI job", ScheduleTime: "12:00", Enabled: true},
	}
	prov := []ScheduledJob{
		{ID: "provisioned-nightly", Name: "Nightly", ScheduleTime: "02:30", Enabled: true},
	}
	got := mergeProvisionedJobs(existing, prov)
	if len(got) != 2 {
		t.Fatalf("want both GUI and provisioned job, got %d", len(got))
	}
	if got[0].ID != "1720000000" {
		t.Errorf("GUI job should be preserved in place, got first ID %q", got[0].ID)
	}
}

func TestMergeProvisionedJobs_DoesNotMutateInputs(t *testing.T) {
	existing := []ScheduledJob{{ID: "nightly", ScheduleTime: "02:30", Enabled: true, NextRun: ""}}
	prov := []ScheduledJob{{ID: "nightly", ScheduleTime: "02:30", Enabled: true}}
	_ = mergeProvisionedJobs(existing, prov)
	if existing[0].NextRun != "" {
		t.Errorf("mergeProvisionedJobs mutated its existing input: %q", existing[0].NextRun)
	}
}
