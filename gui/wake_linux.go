//go:build linux

package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// A timer on CLOCK_REALTIME_ALARM wakes the system from suspend when it comes
// due: the kernel sets the hardware clock's wake alarm as the system goes to
// sleep, as systemd timers with WakeSystem= do. It needs CAP_WAKE_ALARM, which
// the background service has (it runs as root); the app on its own does not.
// The timer lives as long as its file: closing it drops the wake.
//
// That is not enough for hibernate. The kernel's alarmtimer programs the
// hardware clock only in its suspend callback, which runs on the way into
// suspend (S3, s2idle) but not on the hibernate path (freeze, then power off):
// a computer hibernated with only the timer set stays off at the wake time.
// Seen on a legacy BIOS laptop with rtc_cmos, kernel 6.14: the timer woke it
// from suspend, not from hibernate, and powering it on resumed it fine. So as
// long as a wake timer is armed, the hardware clock's own wake alarm
// (/sys/class/rtc/rtcN/wakealarm) is kept set to the same time as well, and it
// is cleared when the timer is dropped, if it is still ours. The timer stays:
// it covers suspend as it always has.
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
	gen := armRTCWakeAlarm(at)
	return func() {
		_ = unix.Close(fd)
		dropRTCWakeAlarm(gen)
	}, nil
}

// rtcSysfsDir is where the kernel lists the hardware clocks: a variable so the
// tests can use a directory of their own.
var rtcSysfsDir = "/sys/class/rtc"

// rtcAlarmSlack is how far, in seconds, a wake alarm read back may be from the
// one written and still be taken as ours.
const rtcAlarmSlack = 5

// rtcAlarm is the hardware clock's wake alarm as this process set it.
var rtcAlarm struct {
	mu      sync.Mutex
	gen     uint64    // counts the arms, so an old cancel does not clear a newer alarm
	at      time.Time // the wake wanted; zero when no timer is armed
	file    string    // the wakealarm file written
	value   int64     // the seconds written to it
	set     bool      // value is in file (as far as this process knows)
	lastErr string
}

// armRTCWakeAlarm keeps the hardware clock's wake alarm at at while the wake
// timer is armed; it returns the arm's number for dropRTCWakeAlarm.
func armRTCWakeAlarm(at time.Time) uint64 {
	rtcAlarm.mu.Lock()
	defer rtcAlarm.mu.Unlock()
	rtcAlarm.gen++
	rtcAlarm.at = at
	setRTCWakeAlarmLocked()
	return rtcAlarm.gen
}

// dropRTCWakeAlarm clears the wake alarm set by arm gen, unless a newer arm has
// replaced it.
func dropRTCWakeAlarm(gen uint64) {
	rtcAlarm.mu.Lock()
	defer rtcAlarm.mu.Unlock()
	if gen != rtcAlarm.gen {
		return
	}
	rtcAlarm.at = time.Time{}
	clearRTCWakeAlarmLocked()
}

// platformSyncWakeAlarm makes sure, just before the computer is put to sleep
// or hibernated, that the hardware clock's wake alarm is still the armed wake:
// an alarm that has gone off since (it woke the computer for this run) or
// that something else has changed is written again. It runs synchronously, so
// the alarm is in place before systemctl is called.
func platformSyncWakeAlarm() {
	rtcAlarm.mu.Lock()
	defer rtcAlarm.mu.Unlock()
	if rtcAlarm.at.IsZero() {
		return
	}
	if rtcAlarm.set {
		if cur, ok, err := readWakeAlarm(rtcAlarm.file); err == nil && ok && absDiff(cur, rtcAlarm.value) <= rtcAlarmSlack {
			return
		}
	}
	setRTCWakeAlarmLocked()
}

// setRTCWakeAlarmLocked writes rtcAlarm.at to the wake alarm: 0 first, as the
// kernel refuses a new alarm while one is set, then the time. The time is
// written in the hardware clock's own seconds (its since_epoch plus the time
// left), so it comes out right even on a clock kept in local time.
func setRTCWakeAlarmLocked() {
	file, err := findWakeAlarm()
	if err != nil {
		noteRTCErrorLocked(fmt.Sprintf("[Wake] RTC wake alarm not set (a wake from hibernate may not happen): %v", err))
		return
	}
	left := int64(math.Ceil(time.Until(rtcAlarm.at).Seconds()))
	if left < 1 {
		// Due now: the timer goes off at once, the computer is awake.
		clearRTCWakeAlarmLocked()
		return
	}
	want := rtcNow(filepath.Dir(file)) + left
	cur, ok, err := readWakeAlarm(file)
	if err != nil {
		noteRTCErrorLocked(fmt.Sprintf("[Wake] RTC wake alarm not set: %v", err))
		return
	}
	if ok && absDiff(cur, want) <= rtcAlarmSlack {
		rtcAlarm.file, rtcAlarm.value, rtcAlarm.set, rtcAlarm.lastErr = file, cur, true, ""
		return
	}
	if ok {
		if err := writeWakeAlarm(file, "0"); err != nil {
			noteRTCErrorLocked(fmt.Sprintf("[Wake] RTC wake alarm not set: clearing the old one: %v", err))
			return
		}
	}
	if err := writeWakeAlarm(file, strconv.FormatInt(want, 10)); err != nil {
		rtcAlarm.set = false
		noteRTCErrorLocked(fmt.Sprintf("[Wake] RTC wake alarm not set: %v", err))
		return
	}
	rtcAlarm.file, rtcAlarm.value, rtcAlarm.set, rtcAlarm.lastErr = file, want, true, ""
	writeDebugLog(fmt.Sprintf("[Wake] RTC wake alarm set for %s (%s)", rtcAlarm.at.Format(time.RFC3339), file))
}

// clearRTCWakeAlarmLocked clears the wake alarm this process set, only if it
// is still its own: another program's alarm is left alone.
func clearRTCWakeAlarmLocked() {
	if !rtcAlarm.set {
		return
	}
	rtcAlarm.set = false
	cur, ok, err := readWakeAlarm(rtcAlarm.file)
	switch {
	case err != nil:
		noteRTCErrorLocked(fmt.Sprintf("[Wake] RTC wake alarm not cleared: %v", err))
	case !ok:
		// It has gone off, or something else cleared it.
	case absDiff(cur, rtcAlarm.value) > rtcAlarmSlack:
		writeDebugLog("[Wake] RTC wake alarm left as it is: another program has set it since")
	default:
		if err := writeWakeAlarm(rtcAlarm.file, "0"); err != nil {
			noteRTCErrorLocked(fmt.Sprintf("[Wake] RTC wake alarm not cleared: %v", err))
			return
		}
		rtcAlarm.lastErr = ""
		writeDebugLog("[Wake] RTC wake alarm cleared")
	}
}

func noteRTCErrorLocked(msg string) {
	if msg != rtcAlarm.lastErr {
		writeDebugLog(msg)
		rtcAlarm.lastErr = msg
	}
}

// findWakeAlarm is the wakealarm file of the hardware clock that has one:
// rtc0 when it does, else the first other.
func findWakeAlarm() (string, error) {
	first := filepath.Join(rtcSysfsDir, "rtc0", "wakealarm")
	if _, err := os.Stat(first); err == nil {
		return first, nil
	}
	matches, _ := filepath.Glob(filepath.Join(rtcSysfsDir, "rtc*", "wakealarm"))
	sort.Strings(matches)
	if len(matches) == 0 {
		return "", fmt.Errorf("no hardware clock with a wake alarm (%s/rtc*/wakealarm)", rtcSysfsDir)
	}
	return matches[0], nil
}

// readWakeAlarm reads the alarm set, in seconds; ok is false when none is.
func readWakeAlarm(file string) (secs int64, ok bool, err error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return 0, false, err
	}
	s := strings.TrimSpace(string(b))
	if s == "" || s == "0" {
		return 0, false, nil
	}
	secs, err = strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("%s reads %q", file, s)
	}
	return secs, true, nil
}

func writeWakeAlarm(file, value string) error {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(value); err != nil {
		_ = f.Close()
		return fmt.Errorf("writing %s to %s: %w", value, file, err)
	}
	return f.Close()
}

// rtcNow is the hardware clock's time in seconds (its since_epoch), or the
// system's when it cannot be read.
func rtcNow(dir string) int64 {
	if b, err := os.ReadFile(filepath.Join(dir, "since_epoch")); err == nil {
		if n, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64); err == nil {
			return n
		}
	}
	return time.Now().Unix()
}

func absDiff(a, b int64) int64 {
	if a > b {
		return a - b
	}
	return b - a
}

func platformWakeTimerInfo() WakeTimerInfo {
	// The hardware clock must be able to wake the system (an rtc with an alarm).
	if _, err := findWakeAlarm(); err != nil {
		return WakeTimerInfo{Note: "no wake-capable hardware clock (/sys/class/rtc/rtc*/wakealarm)"}
	}
	// The setting lives in the service; the app cannot set the timer itself,
	// but the service will, so it is supported either way.
	return WakeTimerInfo{Supported: true, OnMains: true, OnBattery: true}
}

// platformOpenWakePlan: Linux has no power plan setting that stops a timer
// waking the computer, so there is nothing to turn on.
func platformOpenWakePlan() (wakePlan, wakePlanWriter, error) {
	return nil, nil, fmt.Errorf("there is no power plan setting for wake timers on this system")
}
