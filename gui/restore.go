package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RestoreManager handles file restore operations using pbsnbd
type RestoreManager struct {
	config *Config
}

// BackupSnapshot represents a backup snapshot available for restore
type BackupSnapshot struct {
	Type      string    // "host", "vm", "ct"
	ID        string    // backup-id or vm-id
	Timestamp time.Time
	Files     []BackupFile
}

// BackupFile represents a file in a backup snapshot
type BackupFile struct {
	Name string // e.g., "root.pxar.didx", "drive-scsi0.img.fidx"
	Size int64
	Type string // "pxar" (directory), "fidx" (disk image), "blob"
}

func NewRestoreManager(config *Config) *RestoreManager {
	return &RestoreManager{
		config: config,
	}
}

// ListSnapshots lists available backup snapshots from PBS
func (rm *RestoreManager) ListSnapshots() ([]BackupSnapshot, error) {
	// TODO: Implement actual PBS API call to list snapshots
	// For now, return mock data

	snapshots := []BackupSnapshot{
		{
			Type:      "host",
			ID:        "web-server-01",
			Timestamp: time.Now().Add(-24 * time.Hour),
			Files: []BackupFile{
				{Name: "root.pxar.didx", Size: 5 * 1024 * 1024 * 1024, Type: "pxar"},
			},
		},
		{
			Type:      "host",
			ID:        "web-server-01",
			Timestamp: time.Now().Add(-48 * time.Hour),
			Files: []BackupFile{
				{Name: "root.pxar.didx", Size: 4800 * 1024 * 1024, Type: "pxar"},
			},
		},
	}

	return snapshots, nil
}

// RestoreFile restores a specific file or directory from a backup
func (rm *RestoreManager) RestoreFile(snapshot BackupSnapshot, file BackupFile, targetPath string) error {
	if rm.config.BaseURL == "" || rm.config.AuthID == "" || rm.config.Secret == "" {
		return fmt.Errorf("PBS configuration incomplete")
	}

	// Build the backup path
	// Format: type/id/timestamp/filename
	timestamp := snapshot.Timestamp.Format("2006-01-02T15:04:05Z")
	backupPath := fmt.Sprintf("%s/%s/%s/%s",
		snapshot.Type,
		snapshot.ID,
		timestamp,
		file.Name,
	)

	// For PXAR files (directories), we need to extract
	if file.Type == "pxar" {
		return rm.restorePXAR(backupPath, targetPath)
	}

	// For FIDX files (disk images), we use NBD
	if file.Type == "fidx" || strings.HasSuffix(file.Name, ".fidx") {
		return rm.mountNBD(backupPath)
	}

	return fmt.Errorf("unsupported file type: %s", file.Type)
}

// restorePXAR extracts files from a PXAR archive
func (rm *RestoreManager) restorePXAR(backupPath, targetPath string) error {
	// TODO: Implement PXAR extraction
	// This requires either:
	// 1. PBS HTTP API call to download and extract
	// 2. Mount via NBD and use pxar tool
	// 3. Direct PXAR reader implementation

	return fmt.Errorf("PXAR restore not yet implemented - use PBS web UI or CLI")
}

// mountNBD mounts a disk image via NBD for file browsing
func (rm *RestoreManager) mountNBD(backupPath string) error {
	// Find pbsnbd binary
	nbdBinary := "./pbsnbd"
	if _, err := exec.LookPath("pbsnbd"); err == nil {
		nbdBinary = "pbsnbd"
	}

	// Build command
	args := []string{
		"-baseurl", rm.config.BaseURL,
		"-authid", rm.config.AuthID,
		"-secret", rm.config.Secret,
		"-datastore", rm.config.Datastore,
	}

	if rm.config.CertFingerprint != "" {
		args = append(args, "-certfingerprint", rm.config.CertFingerprint)
	}

	if rm.config.Namespace != "" {
		args = append(args, "-namespace", rm.config.Namespace)
	}

	args = append(args, "-path", backupPath)

	cmd := exec.Command(nbdBinary, args...)

	// This will block and mount the NBD device
	// The user needs to manually unmount it later
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start NBD: %v", err)
	}

	return nil
}

// BrowseBackup opens an interactive terminal UI to browse backups
func (rm *RestoreManager) BrowseBackup() error {
	// Find pbsnbd binary
	nbdBinary := "./pbsnbd"
	if _, err := exec.LookPath("pbsnbd"); err == nil {
		nbdBinary = "pbsnbd"
	}

	// Build command (without -path to get interactive UI)
	args := []string{
		"-baseurl", rm.config.BaseURL,
		"-authid", rm.config.AuthID,
		"-secret", rm.config.Secret,
		"-datastore", rm.config.Datastore,
	}

	if rm.config.CertFingerprint != "" {
		args = append(args, "-certfingerprint", rm.config.CertFingerprint)
	}

	if rm.config.Namespace != "" {
		args = append(args, "-namespace", rm.config.Namespace)
	}

	cmd := exec.Command(nbdBinary, args...)

	// Run interactively
	return cmd.Run()
}

// GetSnapshotInfo gets detailed info about a specific snapshot
func (rm *RestoreManager) GetSnapshotInfo(snapshot BackupSnapshot) (string, error) {
	// TODO: Implement PBS API call to get snapshot manifest

	info := fmt.Sprintf(`Backup Snapshot Information
============================

Type: %s
ID: %s
Timestamp: %s
Age: %s

Files:
`,
		snapshot.Type,
		snapshot.ID,
		snapshot.Timestamp.Format("2006-01-02 15:04:05"),
		time.Since(snapshot.Timestamp).Round(time.Hour),
	)

	for _, file := range snapshot.Files {
		info += fmt.Sprintf("  - %s (%s)\n", file.Name, formatBytes(file.Size))
	}

	return info, nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
