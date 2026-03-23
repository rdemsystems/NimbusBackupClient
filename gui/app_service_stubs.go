//go:build service
// +build service

// Stubs for service compilation
// These methods are required by api.BackupHandler interface
// Full implementations are in main.go (GUI mode)

package main

import (
	"fmt"
	"os"
)

// GetConfigWithHostname returns the configuration with hostname
func (a *App) GetConfigWithHostname() map[string]interface{} {
	hostname, _ := os.Hostname()
	result := map[string]interface{}{
		"hostname": hostname,
	}

	if a.config != nil {
		result["server"] = a.config.Server
		result["datastore"] = a.config.Datastore
		result["fingerprint"] = a.config.Fingerprint
		result["backup-id"] = a.config.BackupID
		result["encryption-key"] = a.config.EncryptionKey
	}

	return result
}

// StartBackup starts a backup job
// This is a stub that delegates to the appropriate implementation
func (a *App) StartBackup(backupType string, backupDirs, driveLetters, excludeList []string, backupID string, useVSS bool) error {
	writeDebugLog(fmt.Sprintf("StartBackup stub called: type=%s, dirs=%v, id=%s, vss=%v", backupType, backupDirs, backupID, useVSS))

	// For service mode, we need the real implementation
	// This will be overridden by the full implementation in backup.go
	return fmt.Errorf("StartBackup not fully implemented in service stub")
}
