//go:build linux

package main

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// A timer on CLOCK_REALTIME_ALARM wakes the system from suspend when it comes
// due: the kernel sets the hardware clock's wake alarm as the system goes to
// sleep, as systemd timers with WakeSystem= do. It needs CAP_WAKE_ALARM, which
// the background service has (it runs as root); the app on its own does not.
// The timer lives as long as its file: closing it drops the wake.
func platformArmWakeTimer(at time.Time) (func(), error) {
	fd, err := unix.TimerfdCreate(unix.CLOCK_REALTIME_ALARM, unix.TFD_CLOEXEC)
	if err != nil {
		if err == unix.EPERM {
			return nil, fmt.Errorf("waking the computer needs the background service (CAP_WAKE_ALARM): %w", err)
		}
		return nil, fmt.Errorf("timerfd_create(CLOCK_REALTIME_ALARM): %w", err)
	}
	spec := unix.ItimerSpec{Value: unix.NsecToTimespec(at.UnixNano())}
	if err := unix.TimerfdSettime(fd, unix.TFD_TIMER_ABSTIME, &spec, nil); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("timerfd_settime: %w", err)
	}
	return func() { _ = unix.Close(fd) }, nil
}

func platformWakeTimerInfo() WakeTimerInfo {
	// The hardware clock must be able to wake the system (an rtc with an alarm).
	if _, err := os.Stat("/sys/class/rtc/rtc0/wakealarm"); err != nil {
		return WakeTimerInfo{Note: "no wake-capable hardware clock (/sys/class/rtc/rtc0/wakealarm)"}
	}
	// The setting lives in the service; the app cannot set the timer itself,
	// but the service will, so it is supported either way.
	return WakeTimerInfo{Supported: true, OnMains: true, OnBattery: true}
}
