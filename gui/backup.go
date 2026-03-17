package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// BackupRunner handles the execution of backup operations
type BackupRunner struct {
	config  *Config
	cmd     *exec.Cmd
	onProgress func(progress float64, status string)
	onComplete func(success bool, message string)
}

func NewBackupRunner(config *Config) *BackupRunner {
	return &BackupRunner{
		config: config,
	}
}

func (br *BackupRunner) Start() error {
	if err := br.config.Validate(); err != nil {
		return err
	}

	// Build command arguments
	args := []string{
		"-baseurl", br.config.BaseURL,
		"-authid", br.config.AuthID,
		"-secret", br.config.Secret,
		"-datastore", br.config.Datastore,
		"-backupdir", br.config.BackupDir,
	}

	if br.config.CertFingerprint != "" {
		args = append(args, "-certfingerprint", br.config.CertFingerprint)
	}

	if br.config.Namespace != "" {
		args = append(args, "-namespace", br.config.Namespace)
	}

	if br.config.BackupID != "" {
		args = append(args, "-backup-id", br.config.BackupID)
	}

	if !br.config.UseVSS {
		args = append(args, "-novss")
	}

	// Find the directorybackup binary
	// Assuming it's in the same directory or in PATH
	binaryPath := "./directorybackup"
	if _, err := exec.LookPath("proxmoxbackupgo"); err == nil {
		binaryPath = "proxmoxbackupgo"
	}

	br.cmd = exec.Command(binaryPath, args...)

	// Capture stdout and stderr
	stdout, err := br.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := br.cmd.StderrPipe()
	if err != nil {
		return err
	}

	// Start the command
	if err := br.cmd.Start(); err != nil {
		return err
	}

	// Monitor output in goroutines
	go br.monitorOutput(stdout)
	go br.monitorOutput(stderr)

	// Wait for completion in background
	go func() {
		err := br.cmd.Wait()
		success := err == nil
		message := "Backup completed successfully"
		if !success {
			message = fmt.Sprintf("Backup failed: %v", err)
		}

		if br.onComplete != nil {
			br.onComplete(success, message)
		}
	}()

	return nil
}

func (br *BackupRunner) Stop() error {
	if br.cmd != nil && br.cmd.Process != nil {
		return br.cmd.Process.Kill()
	}
	return nil
}

func (br *BackupRunner) monitorOutput(reader io.Reader) {
	scanner := bufio.NewScanner(reader)

	// Regular expressions to parse output
	progressRe := regexp.MustCompile(`Progress:\s*(\d+)%`)
	speedRe := regexp.MustCompile(`Speed:\s*([\d.]+)\s*([KMG]B/s)`)
	chunksRe := regexp.MustCompile(`Chunks:\s*(\d+)\s*new,\s*(\d+)\s*reused`)

	for scanner.Scan() {
		line := scanner.Text()

		// Parse progress
		if matches := progressRe.FindStringSubmatch(line); len(matches) > 1 {
			if progress, err := strconv.ParseFloat(matches[1], 64); err == nil {
				if br.onProgress != nil {
					br.onProgress(progress/100.0, line)
				}
			}
		}

		// Parse other stats
		if speedRe.MatchString(line) || chunksRe.MatchString(line) {
			if br.onProgress != nil {
				br.onProgress(-1, line) // -1 means keep current progress, just update status
			}
		}

		// General status update
		if strings.Contains(line, "Uploading") ||
		   strings.Contains(line, "Processing") ||
		   strings.Contains(line, "Snapshot") {
			if br.onProgress != nil {
				br.onProgress(-1, line)
			}
		}
	}
}

// SetProgressCallback sets the callback for progress updates
func (br *BackupRunner) SetProgressCallback(callback func(float64, string)) {
	br.onProgress = callback
}

// SetCompleteCallback sets the callback for completion
func (br *BackupRunner) SetCompleteCallback(callback func(bool, string)) {
	br.onComplete = callback
}
