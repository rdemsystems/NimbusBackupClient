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
