package app

import (
	"context"
	"sync"

	"github.com/tizbac/proxmoxbackupclient_go/gui/api"
)

// App struct contains the application state
type App struct {
	ctx              context.Context
	config           interface{} // Will be properly typed later
	stopScheduler    chan struct{}
	apiClient        *api.Client
	mode             api.ExecutionMode
	callbacksMap     map[string]interface{}
	callbacksMutex   sync.RWMutex
	isServiceProcess bool
}

// NewAppForService creates an App instance for the Windows Service
func NewAppForService(ctx context.Context) *App {
	return &App{
		ctx:              ctx,
		stopScheduler:    make(chan struct{}),
		callbacksMap:     make(map[string]interface{}),
		isServiceProcess: true,
		mode:             api.ModeStandalone,
	}
}

// CleanupAbandonedJobs removes orphaned job entries
func (a *App) CleanupAbandonedJobs() {
	// TODO: Implement cleanup logic
}

// StartScheduler starts the job scheduler
func (a *App) StartScheduler() {
	// TODO: Implement scheduler logic
}

// StopScheduler stops the job scheduler
func (a *App) StopScheduler() {
	if a.stopScheduler != nil {
		close(a.stopScheduler)
	}
}

// StartBackup starts a backup job
func (a *App) StartBackup(backupType string, backupDirs, driveLetters, excludeList []string, backupID string, useVSS bool) error {
	// TODO: Implement backup logic
	return nil
}

// GetConfigWithHostname returns the configuration with hostname
func (a *App) GetConfigWithHostname() map[string]interface{} {
	// TODO: Implement config retrieval
	return make(map[string]interface{})
}

// GetScheduledJobsForAPI returns all scheduled jobs
func (a *App) GetScheduledJobsForAPI() []map[string]interface{} {
	// TODO: Implement job listing
	return make([]map[string]interface{}, 0)
}

// SaveScheduledJobFromMap creates a new scheduled job
func (a *App) SaveScheduledJobFromMap(job map[string]interface{}) error {
	// TODO: Implement job creation
	return nil
}

// UpdateScheduledJobFromMap updates an existing scheduled job
func (a *App) UpdateScheduledJobFromMap(job map[string]interface{}) error {
	// TODO: Implement job update
	return nil
}

// DeleteScheduledJobFromMap deletes a scheduled job
func (a *App) DeleteScheduledJobFromMap(jobID string) error {
	// TODO: Implement job deletion
	return nil
}
