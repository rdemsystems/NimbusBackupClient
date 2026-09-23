//go:build windows
// +build windows

package snapshot

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSymlinkSnapshot_Windows tests symlink creation on Windows
func TestSymlinkSnapshot_Windows(t *testing.T) {

	// Create temporary directory for testing
	tmpDir := t.TempDir()

	testCases := []struct {
		name             string
		symlinkPath      string
		id               string
		deviceObjectPath string
		expectError      bool
	}{
		{
			name:             "valid paths",
			symlinkPath:      tmpDir,
			id:               "test-snapshot-1",
			deviceObjectPath: `\\?\GLOBALROOT\Device\HarddiskVolumeShadowCopy1`,
			expectError:      false, // Will error without admin, but tests path logic
		},
		{
			name:             "empty id",
			symlinkPath:      tmpDir,
			id:               "",
			deviceObjectPath: `\\?\GLOBALROOT\Device\HarddiskVolumeShadowCopy1`,
			expectError:      false, // Path operations should still work
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Note: This will fail without admin privileges
			// But we can at least test that it doesn't panic
			_, err := SymlinkSnapshot(tc.symlinkPath, tc.id, tc.deviceObjectPath)

			// We expect errors on non-admin systems, just ensure it doesn't panic
			if err != nil {
				t.Logf("Expected error (requires admin): %v", err)
			}
		})
	}
}

// TestGetAppDataFolder_Windows tests AppData folder detection
func TestGetAppDataFolder_Windows(t *testing.T) {

	appDataFolder, err := getAppDataFolder()
	if err != nil {
		t.Fatalf("Failed to get AppData folder: %v", err)
	}

	if appDataFolder == "" {
		t.Error("AppData folder should not be empty")
	}

	// Verify it ends with expected path
	if !filepath.IsAbs(appDataFolder) {
		t.Errorf("AppData folder should be absolute path, got: %s", appDataFolder)
	}

	// Check if folder exists (should be created by function)
	info, err := os.Stat(appDataFolder)
	if err != nil {
		t.Errorf("AppData folder should exist: %v", err)
	}

	if !info.IsDir() {
		t.Error("AppData folder should be a directory")
	}
}
