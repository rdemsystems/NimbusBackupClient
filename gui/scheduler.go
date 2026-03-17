package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Scheduler handles setting up scheduled tasks (cron on Linux, Task Scheduler on Windows)
type Scheduler struct {
	jobManager *JobManager
}

func NewScheduler(jm *JobManager) *Scheduler {
	return &Scheduler{
		jobManager: jm,
	}
}

// ScheduleJob creates a scheduled task for the given job
func (s *Scheduler) ScheduleJob(job *Job) error {
	if runtime.GOOS == "windows" {
		return s.scheduleWindows(job)
	}
	return s.scheduleLinux(job)
}

// UnscheduleJob removes the scheduled task for the given job
func (s *Scheduler) UnscheduleJob(job *Job) error {
	if runtime.GOOS == "windows" {
		return s.unscheduleWindows(job)
	}
	return s.unscheduleLinux(job)
}

// Linux/macOS: Use crontab
func (s *Scheduler) scheduleLinux(job *Job) error {
	// Get current crontab
	currentCron, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		// No crontab exists yet, that's okay
		currentCron = []byte{}
	}

	// Build cron line
	// Get path to GUI binary
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	// Export job config to temporary file
	tmpDir := os.TempDir()
	configPath := filepath.Join(tmpDir, fmt.Sprintf("pbs-job-%s.json", job.ID))
	if err := job.ExportToJSON(configPath); err != nil {
		return err
	}

	cronLine := fmt.Sprintf(
		"%s %s run-job %s # PBS Job: %s\n",
		job.ScheduleCron,
		exePath,
		job.ID,
		job.Name,
	)

	// Append to crontab
	newCron := string(currentCron) + cronLine

	// Write new crontab
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = bytes.NewReader([]byte(newCron))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update crontab: %v", err)
	}

	return nil
}

func (s *Scheduler) unscheduleLinux(job *Job) error {
	// Get current crontab
	currentCron, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return err
	}

	// Remove lines containing job ID
	lines := strings.Split(string(currentCron), "\n")
	filteredLines := []string{}
	for _, line := range lines {
		if !strings.Contains(line, job.ID) {
			filteredLines = append(filteredLines, line)
		}
	}

	// Write new crontab
	newCron := strings.Join(filteredLines, "\n")
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = bytes.NewReader([]byte(newCron))
	return cmd.Run()
}

// Windows: Use Task Scheduler (schtasks)
func (s *Scheduler) scheduleWindows(job *Job) error {
	// Get path to GUI binary
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	// Convert cron to Task Scheduler schedule
	schedule := s.cronToWindowsSchedule(job.ScheduleCron)

	// Create scheduled task
	taskName := fmt.Sprintf("ProxmoxBackup_%s", job.ID)

	cmd := exec.Command(
		"schtasks",
		"/Create",
		"/TN", taskName,
		"/TR", fmt.Sprintf(`"%s" run-job %s`, exePath, job.ID),
		"/SC", schedule.Type,
	)

	// Add additional parameters based on schedule type
	if schedule.Type == "DAILY" {
		cmd.Args = append(cmd.Args, "/ST", schedule.StartTime)
	} else if schedule.Type == "WEEKLY" {
		cmd.Args = append(cmd.Args, "/D", schedule.DayOfWeek, "/ST", schedule.StartTime)
	}

	cmd.Args = append(cmd.Args, "/F") // Force create/overwrite

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create scheduled task: %v\nOutput: %s", err, output)
	}

	return nil
}

func (s *Scheduler) unscheduleWindows(job *Job) error {
	taskName := fmt.Sprintf("ProxmoxBackup_%s", job.ID)

	cmd := exec.Command("schtasks", "/Delete", "/TN", taskName, "/F")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete scheduled task: %v\nOutput: %s", err, output)
	}

	return nil
}

type WindowsSchedule struct {
	Type      string // DAILY, WEEKLY, MONTHLY
	StartTime string // HH:MM
	DayOfWeek string // MON, TUE, etc.
}

// Convert cron expression to Windows Task Scheduler format
func (s *Scheduler) cronToWindowsSchedule(cronExpr string) WindowsSchedule {
	// Simplified cron to Windows conversion
	// Format: MIN HOUR DAY MONTH WEEKDAY

	// Common patterns:
	// 0 2 * * * = Daily at 2:00
	// 0 */6 * * * = Every 6 hours
	// 0 2 * * 0 = Weekly on Sunday at 2:00

	// Default to daily at 2:00 AM
	return WindowsSchedule{
		Type:      "DAILY",
		StartTime: "02:00",
	}
}

// GetCronExpression converts schedule preset to cron expression
func GetCronExpression(preset string) string {
	switch preset {
	case "Toutes les heures":
		return "0 * * * *"
	case "Toutes les 6 heures":
		return "0 */6 * * *"
	case "Quotidien (2h du matin)":
		return "0 2 * * *"
	case "Hebdomadaire (Dimanche 2h)":
		return "0 2 * * 0"
	case "Mensuel (1er jour du mois)":
		return "0 2 1 * *"
	default:
		return "0 2 * * *" // Default daily at 2am
	}
}

