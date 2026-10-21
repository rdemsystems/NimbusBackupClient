//go:build windows || service
// +build windows service

package main

import (
	"fmt"
	"time"
)

// Only the service runs these jobs, so this lives in the files the service
// is built from (every Windows build, and the service build elsewhere).

// runStartupJobsAtBoot runs the jobs set to run at machine startup, once per
// boot. The service calls it when it starts, which is what "at machine
// startup" means. The GUI used to run them each time it opened, which ran
// the job again at every launch.
func (a *App) runStartupJobsAtBoot() {
	uptime, err := systemUptime()
	if !isMachineStartup(uptime, err) {
		writeDebugLog(fmt.Sprintf("Startup jobs not run: the machine has been up for %s, the service was restarted, not booted",
			uptime.Round(time.Second)))
		return
	}
	a.HandleStartupRun()
}
