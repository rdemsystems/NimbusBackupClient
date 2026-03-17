package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// handleConfig handles GET and POST for configuration
func (a *App) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		json.NewEncoder(w).Encode(a.config)

	case "POST":
		var config Config
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		a.config = &config
		if err := config.Save(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"message": "Configuration enregistrée",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTestConnection tests the PBS connection
func (a *App) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// TODO: Implement actual PBS connection test
	if a.config.BaseURL == "" {
		http.Error(w, "URL du serveur PBS requis", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "Connexion réussie ! (test simulé)",
	})
}

// handleBackupStart starts a backup
func (a *App) handleBackupStart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Type        string   `json:"type"`        // "directory" or "machine"
		BackupDir   string   `json:"backupDir"`   // for directory type
		DriveLetter string   `json:"driveLetter"` // for machine type
		ExcludeList []string `json:"excludeList"` // for machine type
		BackupID    string   `json:"backupId"`
		UseVSS      bool     `json:"useVss"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Implement actual backup logic
	if err := a.config.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Type == "directory" {
		log.Printf("Directory backup requested: %s (VSS: %v)\n", req.BackupDir, req.UseVSS)
	} else {
		log.Printf("Machine backup requested: %s (VSS: %v, excludes: %d)\n",
			req.DriveLetter, req.UseVSS, len(req.ExcludeList))
	}

	http.Error(w, "Fonctionnalité de backup à implémenter", http.StatusNotImplemented)
}

// handleSnapshotList lists available snapshots
func (a *App) handleSnapshotList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		BackupID string `json:"backupId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Implement actual snapshot listing
	log.Printf("List snapshots requested for: %s\n", req.BackupID)

	// Mock response for now
	snapshots := []map[string]string{
		{
			"id":   "2024-03-17T10:30:00Z",
			"time": "2024-03-17 10:30:00",
			"type": "machine",
		},
	}

	json.NewEncoder(w).Encode(snapshots)
}

// handleSnapshotRestore restores a snapshot
func (a *App) handleSnapshotRestore(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		SnapshotID string `json:"snapshotId"`
		DestPath   string `json:"destPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Implement actual restore logic
	log.Printf("Restore requested: snapshot=%s, dest=%s\n", req.SnapshotID, req.DestPath)

	http.Error(w, "Fonctionnalité de restore à implémenter", http.StatusNotImplemented)
}

// handleInfo returns app info
func (a *App) handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	info := map[string]string{
		"name":    appName,
		"version": appVersion,
		"author":  "RDEM Systems",
		"website": "https://backup.rdem-systems.com",
	}

	json.NewEncoder(w).Encode(info)
}
