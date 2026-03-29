//go:build service
// +build service

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var (
	backupLogger  *RotatingLogger
	serviceLogger *RotatingLogger
	logDir        string
)

func init() {
	// Setup log directory for SERVICE
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = "C:\\ProgramData"
	}
	logDir = filepath.Join(programData, "NimbusBackup")
	// #nosec G703 -- ProgramData is a trusted Windows system environment variable
	_ = os.MkdirAll(logDir, 0700)

	// Initialize rotating loggers
	var err error
	backupLogger, err = NewRotatingLogger(
		filepath.Join(logDir, "backup-service.log"),
		MaxLogSize,
		MaxLogFiles,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create backup logger: %v\n", err)
	}

	serviceLogger, err = NewRotatingLogger(
		filepath.Join(logDir, "service-service.log"),
		MaxLogSize,
		MaxLogFiles,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create service logger: %v\n", err)
	}
}

// writeDebugLog writes to service log (scheduler, general operations)
func writeDebugLog(message string) {
	writeLogToLogger(serviceLogger, "SERVICE", message)
}

// writeBackupLog writes to backup log (backup operations)
func writeBackupLog(message string) {
	writeLogToLogger(backupLogger, "BACKUP", message)
}

func writeLogToLogger(logger *RotatingLogger, prefix string, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] [%s] %s\n", prefix, timestamp, message)

	// Fallback to stderr if logger is not initialized
	if logger == nil {
		fmt.Fprint(os.Stderr, logLine)
		return
	}

	// Write to rotating logger
	if err := logger.Write(logLine); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write log: %v\n", err)
		fmt.Fprint(os.Stderr, logLine)
	}

	// Also write to stderr for console output
	fmt.Fprint(os.Stderr, logLine)
}
