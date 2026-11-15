//go:build !windows && !linux

package main

import (
	"fmt"
	"time"
)

func platformArmWakeTimer(time.Time) (func(), error) {
	return nil, fmt.Errorf("waking the computer for a backup is not supported on this system")
}

func platformWakeTimerInfo() WakeTimerInfo {
	return WakeTimerInfo{Note: "waking the computer for a backup is not supported on this system"}
}
