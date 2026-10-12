//go:build !service
// +build !service

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// webviewUserDataDir picks the WebView2 user-data folder (Windows only; the
// option is ignored elsewhere).
//
// It used to be %APPDATA%\ProxmoxBackupClient (Roaming). For a domain account
// whose roaming profile is not provisioned on the machine, typically when the
// GUI is elevated as another user, WebView2 cannot create that folder and the
// window never opens. Local AppData is always provisioned per profile and is
// Microsoft's recommended location.
//
// An existing Roaming folder is kept: it holds the frontend's localStorage
// (the chosen language among others), which moving would reset.
//
// Adapted from CJInvent/NimbusBackupClient 756d388.
func webviewUserDataDir() string {
	if roaming := os.Getenv("APPDATA"); roaming != "" {
		legacy := filepath.Join(roaming, "ProxmoxBackupClient")
		if st, err := os.Stat(legacy); err == nil && st.IsDir() { // #nosec G703 -- path built from the user's APPDATA/LOCALAPPDATA/TEMP env vars + fixed names
			return legacy
		}
	}

	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.Getenv("APPDATA")
	}
	if base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "ProxmoxBackupClient", "WebView2")
	// Pre-create it so a failure is logged with its path instead of ending in
	// WebView2's own dead-end dialog.
	if err := os.MkdirAll(dir, 0o700); err != nil { // #nosec G703 -- path built from the user's APPDATA/LOCALAPPDATA/TEMP env vars + fixed names
		writeDebugLog(fmt.Sprintf("WebView2 data dir %s cannot be created: %v; using the temp dir", dir, err))
		dir = filepath.Join(os.TempDir(), "ProxmoxBackupClient", "WebView2")
		_ = os.MkdirAll(dir, 0o700) // #nosec G703 -- path built from the user's APPDATA/LOCALAPPDATA/TEMP env vars + fixed names
	}
	writeDebugLog(fmt.Sprintf("WebView2 user-data path: %s", dir))
	return dir
}
