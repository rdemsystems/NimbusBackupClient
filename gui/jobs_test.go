package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJobManagerAddJob(t *testing.T) {
	tmpDir := t.TempDir()
	jm := &JobManager{
		Jobs:     []*Job{},
		filePath: filepath.Join(tmpDir, "jobs.json"),
	}

	job := &Job{
		Name:        "Test Job",
		Description: "Test backup job",
		Enabled:     true,
		PBSConfig: Config{
			BaseURL:   "https://pbs.example.com:8007",
			AuthID:    "test@pbs!token",
			Secret:    "secret",
			Datastore: "backup",
		},
		Folders:    []string{"/tmp/test"},
		Schedule:   "Daily",
		KeepLast:   7,
		KeepDaily:  14,
		KeepWeekly: 8,
	}

	if err := jm.AddJob(job); err != nil {
		t.Fatalf("AddJob() error = %v", err)
	}

	if len(jm.Jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(jm.Jobs))
	}

	if job.ID == "" {
		t.Error("Job ID should be auto-generated")
	}

	if job.Created.IsZero() {
		t.Error("Job Created timestamp should be set")
	}

	// Verify file was created
	if _, err := os.Stat(jm.filePath); os.IsNotExist(err) {
		t.Error("Jobs file should be created")
	}
}

func TestJobManagerUpdateJob(t *testing.T) {
	tmpDir := t.TempDir()
	jm := &JobManager{
		Jobs:     []*Job{},
		filePath: filepath.Join(tmpDir, "jobs.json"),
	}

	job := &Job{
		ID:          "test-id",
		Name:        "Original Name",
		Description: "Original description",
		Created:     time.Now(),
	}

	jm.Jobs = append(jm.Jobs, job)

	updatedJob := &Job{
		Name:        "Updated Name",
		Description: "Updated description",
	}

	if err := jm.UpdateJob("test-id", updatedJob); err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}

	if jm.Jobs[0].Name != "Updated Name" {
		t.Errorf("Name = %v, want 'Updated Name'", jm.Jobs[0].Name)
	}

	// Created should be preserved
	if jm.Jobs[0].Created.IsZero() {
		t.Error("Created timestamp should be preserved")
	}
}

func TestJobManagerDeleteJob(t *testing.T) {
	tmpDir := t.TempDir()
	jm := &JobManager{
		Jobs:     []*Job{},
		filePath: filepath.Join(tmpDir, "jobs.json"),
	}

	jm.Jobs = append(jm.Jobs, &Job{ID: "job1", Name: "Job 1"})
	jm.Jobs = append(jm.Jobs, &Job{ID: "job2", Name: "Job 2"})
	jm.Jobs = append(jm.Jobs, &Job{ID: "job3", Name: "Job 3"})

	if err := jm.DeleteJob("job2"); err != nil {
		t.Fatalf("DeleteJob() error = %v", err)
	}

	if len(jm.Jobs) != 2 {
		t.Errorf("Expected 2 jobs after delete, got %d", len(jm.Jobs))
	}

	// Verify job2 is gone
	for _, job := range jm.Jobs {
		if job.ID == "job2" {
			t.Error("Job2 should be deleted")
		}
	}
}

func TestJobManagerGetEnabledJobs(t *testing.T) {
	jm := &JobManager{
		Jobs: []*Job{
			{ID: "job1", Name: "Job 1", Enabled: true},
			{ID: "job2", Name: "Job 2", Enabled: false},
			{ID: "job3", Name: "Job 3", Enabled: true},
		},
	}

	enabled := jm.GetEnabledJobs()

	if len(enabled) != 2 {
		t.Errorf("Expected 2 enabled jobs, got %d", len(enabled))
	}

	for _, job := range enabled {
		if !job.Enabled {
			t.Errorf("GetEnabledJobs returned disabled job %s", job.ID)
		}
	}
}

func TestJobExportToJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "job-export.json")

	job := &Job{
		ID:          "test-job",
		Name:        "Test Job",
		Description: "Test export",
		Enabled:     true,
		Created:     time.Now(),
		PBSConfig: Config{
			BaseURL:   "https://pbs.example.com:8007",
			AuthID:    "test@pbs!token",
			Secret:    "secret",
			Datastore: "backup",
		},
	}

	if err := job.ExportToJSON(filePath); err != nil {
		t.Fatalf("ExportToJSON() error = %v", err)
	}

	// Verify file exists and has content
	stat, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Exported file not found: %v", err)
	}

	if stat.Size() == 0 {
		t.Error("Exported file is empty")
	}
}

func TestGetCronExpression(t *testing.T) {
	tests := []struct {
		preset   string
		expected string
	}{
		{"Toutes les heures", "0 * * * *"},
		{"Toutes les 6 heures", "0 */6 * * *"},
		{"Quotidien (2h du matin)", "0 2 * * *"},
		{"Hebdomadaire (Dimanche 2h)", "0 2 * * 0"},
		{"Mensuel (1er jour du mois)", "0 2 1 * *"},
		{"Unknown", "0 2 * * *"}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.preset, func(t *testing.T) {
			result := GetCronExpression(tt.preset)
			if result != tt.expected {
				t.Errorf("GetCronExpression(%s) = %s, want %s", tt.preset, result, tt.expected)
			}
		})
	}
}
