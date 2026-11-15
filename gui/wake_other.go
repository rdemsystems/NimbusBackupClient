//go:build !windows && !linux

package main

import (
	"fmt"
	"time"
)

func platformArmWakeTimer(time.Time) (func(), error) {
	return nil, fmt.Errorf("waking the computer for a backup is not supported on this system")
}

func platformSyncWakeAlarm() {}

func platformWakeTimerInfo() WakeTimerInfo {
	return WakeTimerInfo{Note: "waking the computer for a backup is not supported on this system"}
}

func platformOpenWakePlan() (wakePlan, wakePlanWriter, error) {
	return nil, nil, fmt.Errorf("there is no power plan setting for wake timers on this system")
}
