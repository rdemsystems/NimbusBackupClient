package main

import "sync"

// configSyncMu keeps two requests from reloading the settings at once.
var configSyncMu sync.Mutex

// SyncConfigFromDisk reloads this program's settings when another program
// has written config.json since this one last read or wrote it, and says
// whether it did.
//
// The service loads its settings when it starts and keeps them in memory.
// Re-running the installer with CONFIGFILE= merges a settings file into
// config.json from a program of its own (-apply-settings), while the service
// runs on. The service's next change of a setting (a fingerprint pinned by a
// connection test, a speed limit, a server edited) then saved its old copy
// over the merged settings, and the merge was lost without a word (#158).
// The service's API asks this before every request (api.ConfigSyncer), so a
// request reads and changes the settings as they are on disk. A backup
// reloads them when it starts as before (ReloadConfig).
func (a *App) SyncConfigFromDisk() bool {
	configSyncMu.Lock()
	defer configSyncMu.Unlock()
	if a.config == nil || a.delegated() || !a.config.changedOnDisk() {
		return false
	}
	writeDebugLog("config.json was changed by another program (the installer merging a settings file, say): settings reloaded from it")
	a.ReloadConfig()
	return true
}
