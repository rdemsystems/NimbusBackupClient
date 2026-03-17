package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Job represents a configured backup job
type Job struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	Created     time.Time `json:"created"`
	LastRun     time.Time `json:"last_run,omitempty"`

	// PBS Config
	PBSConfig Config `json:"pbs_config"`

	// Backup sources
	Folders []string `json:"folders"`
	Disks   []string `json:"disks,omitempty"`

	// Exclusions
	Exclusions []string `json:"exclusions"`

	// Schedule
	Schedule     string `json:"schedule"`       // cron format or preset
	ScheduleCron string `json:"schedule_cron"`  // actual cron expression

	// Retention
	KeepLast    int `json:"keep_last"`
	KeepDaily   int `json:"keep_daily"`
	KeepWeekly  int `json:"keep_weekly"`
	KeepMonthly int `json:"keep_monthly"`

	// Advanced
	Compression     string `json:"compression"`
	ChunkSize       string `json:"chunk_size"`
	BandwidthLimit  int    `json:"bandwidth_limit"`
	ParallelUploads int    `json:"parallel_uploads"`
}

// JobManager manages multiple backup jobs
type JobManager struct {
	Jobs     []*Job
	filePath string
}

func NewJobManager() (*JobManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".proxmox-backup-guardian")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, err
	}

	jm := &JobManager{
		Jobs:     []*Job{},
		filePath: filepath.Join(configDir, "jobs.json"),
	}

	// Load existing jobs
	_ = jm.Load()

	return jm, nil
}

func (jm *JobManager) Load() error {
	data, err := os.ReadFile(jm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No jobs file yet, that's okay
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &jm.Jobs)
}

func (jm *JobManager) Save() error {
	data, err := json.MarshalIndent(jm.Jobs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(jm.filePath, data, 0600)
}

func (jm *JobManager) AddJob(job *Job) error {
	// Generate ID if not set
	if job.ID == "" {
		job.ID = fmt.Sprintf("job_%d", time.Now().Unix())
	}

	job.Created = time.Now()
	jm.Jobs = append(jm.Jobs, job)

	return jm.Save()
}

func (jm *JobManager) UpdateJob(id string, updatedJob *Job) error {
	for i, job := range jm.Jobs {
		if job.ID == id {
			updatedJob.ID = id
			updatedJob.Created = job.Created
			jm.Jobs[i] = updatedJob
			return jm.Save()
		}
	}
	return fmt.Errorf("job not found: %s", id)
}

func (jm *JobManager) DeleteJob(id string) error {
	for i, job := range jm.Jobs {
		if job.ID == id {
			jm.Jobs = append(jm.Jobs[:i], jm.Jobs[i+1:]...)
			return jm.Save()
		}
	}
	return fmt.Errorf("job not found: %s", id)
}

func (jm *JobManager) GetJob(id string) (*Job, error) {
	for _, job := range jm.Jobs {
		if job.ID == id {
			return job, nil
		}
	}
	return nil, fmt.Errorf("job not found: %s", id)
}

func (jm *JobManager) GetEnabledJobs() []*Job {
	enabled := []*Job{}
	for _, job := range jm.Jobs {
		if job.Enabled {
			enabled = append(enabled, job)
		}
	}
	return enabled
}

// ExportToINI exports a job to INI format (compatible with directorybackup config)
func (j *Job) ExportToINI(filePath string) error {
	ini := fmt.Sprintf(`# Proxmox Backup Guardian Job: %s
# Generated: %s

[pbs]
baseurl = %s
certfingerprint = %s
authid = %s
secret = %s
datastore = %s
namespace = %s

[backup]
folders = %s
backup-id = %s
usevss = %t

[exclusions]
patterns = %s

[schedule]
cron = %s

[retention]
keep-last = %d
keep-daily = %d
keep-weekly = %d
keep-monthly = %d

[advanced]
compression = %s
chunk-size = %s
bandwidth-limit = %d
parallel-uploads = %d
`,
		j.Name,
		time.Now().Format(time.RFC3339),
		j.PBSConfig.BaseURL,
		j.PBSConfig.CertFingerprint,
		j.PBSConfig.AuthID,
		j.PBSConfig.Secret,
		j.PBSConfig.Datastore,
		j.PBSConfig.Namespace,
		joinStrings(j.Folders, ","),
		j.PBSConfig.BackupID,
		j.PBSConfig.UseVSS,
		joinStrings(j.Exclusions, ","),
		j.ScheduleCron,
		j.KeepLast,
		j.KeepDaily,
		j.KeepWeekly,
		j.KeepMonthly,
		j.Compression,
		j.ChunkSize,
		j.BandwidthLimit,
		j.ParallelUploads,
	)

	return os.WriteFile(filePath, []byte(ini), 0600)
}

// ExportToJSON exports a job to JSON format
func (j *Job) ExportToJSON(filePath string) error {
	data, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0600)
}

func joinStrings(items []string, sep string) string {
	result := ""
	for i, item := range items {
		if i > 0 {
			result += sep
		}
		result += item
	}
	return result
}
