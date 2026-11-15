//go:build linux

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The real alarm timer: as root (the service) it is set and dropped; as a user
// the kernel refuses it, and the message says the service does it. The
// hardware clock's wake alarm goes to a directory of the test's own, so the
// machine's is not touched.
func TestPlatformWakeTimer(t *testing.T) {
	newRTCTest(t, false)
	cancel, err := platformArmWakeTimer(time.Now().Add(time.Hour))
	if os.Geteuid() != 0 {
		if err == nil {
			cancel()
			t.Skip("this user may set wake alarms (CAP_WAKE_ALARM)")
		}
		if !strings.Contains(err.Error(), "background service") {
			t.Fatalf("as a user: %v, want it to name the background service", err)
		}
		return
	}
	if err != nil {
		t.Skipf("as root, no wake alarm on this machine: %v", err)
	}
	cancel()
}

// rtcTest is a stand-in for /sys/class/rtc: rtc0 with an empty wakealarm and,
// when withRTC, a since_epoch of the system's time; without, an empty
// directory, no hardware clock at all.
type rtcTest struct {
	dir   string
	alarm string
	epoch string
}

func newRTCTest(t *testing.T, withRTC bool) *rtcTest {
	t.Helper()
	rt := &rtcTest{dir: t.TempDir()}
	rt.alarm = filepath.Join(rt.dir, "rtc0", "wakealarm")
	rt.epoch = filepath.Join(rt.dir, "rtc0", "since_epoch")
	if withRTC {
		if err := os.MkdirAll(filepath.Dir(rt.alarm), 0o755); err != nil {
			t.Fatal(err)
		}
		rt.write(t, rt.alarm, "")
		rt.write(t, rt.epoch, strconv.FormatInt(time.Now().Unix(), 10))
	}
	oldDir := rtcSysfsDir
	rtcSysfsDir = rt.dir
	resetRTCAlarm()
	t.Cleanup(func() {
		rtcSysfsDir = oldDir
		resetRTCAlarm()
	})
	return rt
}

func resetRTCAlarm() {
	rtcAlarm.mu.Lock()
	defer rtcAlarm.mu.Unlock()
	rtcAlarm.gen, rtcAlarm.at, rtcAlarm.file, rtcAlarm.value, rtcAlarm.set, rtcAlarm.lastErr = 0, time.Time{}, "", 0, false, ""
}

func (rt *rtcTest) write(t *testing.T, file, s string) {
	t.Helper()
	if err := os.WriteFile(file, []byte(s+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// read is the wakealarm's seconds, 0 when none is set.
func (rt *rtcTest) read(t *testing.T) int64 {
	t.Helper()
	n, ok, err := readWakeAlarm(rt.alarm)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		return 0
	}
	return n
}

func near(a, b int64) bool { return absDiff(a, b) <= 2 }

// Arming sets the hardware clock's alarm to the wake time; dropping clears it.
func TestRTCWakeAlarmSetAndCleared(t *testing.T) {
	rt := newRTCTest(t, true)
	at := time.Now().Add(time.Hour)
	gen := armRTCWakeAlarm(at)
	if got := rt.read(t); !near(got, at.Unix()) {
		t.Fatalf("wakealarm %d, want about %d", got, at.Unix())
	}
	dropRTCWakeAlarm(gen)
	if got := rt.read(t); got != 0 {
		t.Fatalf("after the drop wakealarm is %d, want it cleared", got)
	}
}

// A hardware clock kept in local time (an hour ahead here) gets the time in
// its own seconds: the wake still comes an hour from now.
func TestRTCWakeAlarmClockInLocalTime(t *testing.T) {
	rt := newRTCTest(t, true)
	rtc := time.Now().Unix() + 3600
	rt.write(t, rt.epoch, strconv.FormatInt(rtc, 10))
	armRTCWakeAlarm(time.Now().Add(time.Hour))
	if got := rt.read(t); !near(got, rtc+3600) {
		t.Fatalf("wakealarm %d, want about %d", got, rtc+3600)
	}
}

// Another program's alarm, set over ours since, is not cleared on the drop.
func TestRTCWakeAlarmOtherProgramsKept(t *testing.T) {
	rt := newRTCTest(t, true)
	gen := armRTCWakeAlarm(time.Now().Add(time.Hour))
	other := time.Now().Add(5 * time.Hour).Unix()
	rt.write(t, rt.alarm, strconv.FormatInt(other, 10))
	dropRTCWakeAlarm(gen)
	if got := rt.read(t); got != other {
		t.Fatalf("wakealarm %d, want the other program's %d left", got, other)
	}
}

// An alarm already there is replaced by the wake (the kernel needs 0 first).
func TestRTCWakeAlarmReplacesOld(t *testing.T) {
	rt := newRTCTest(t, true)
	rt.write(t, rt.alarm, strconv.FormatInt(time.Now().Add(9*time.Hour).Unix(), 10))
	at := time.Now().Add(time.Hour)
	armRTCWakeAlarm(at)
	if got := rt.read(t); !near(got, at.Unix()) {
		t.Fatalf("wakealarm %d, want about %d", got, at.Unix())
	}
}

// A drop from an older arm does not clear the newer one's alarm.
func TestRTCWakeAlarmOldDropIgnored(t *testing.T) {
	rt := newRTCTest(t, true)
	first := armRTCWakeAlarm(time.Now().Add(time.Hour))
	at := time.Now().Add(2 * time.Hour)
	armRTCWakeAlarm(at)
	dropRTCWakeAlarm(first)
	if got := rt.read(t); !near(got, at.Unix()) {
		t.Fatalf("wakealarm %d, want the newer %d kept", got, at.Unix())
	}
}

// Before sleep or hibernate an alarm that has gone off (or been changed) is
// written again; one that is still right is left; with no wake armed nothing
// is written.
func TestSyncWakeAlarm(t *testing.T) {
	rt := newRTCTest(t, true)
	platformSyncWakeAlarm()
	if got := rt.read(t); got != 0 {
		t.Fatalf("nothing armed, yet wakealarm is %d", got)
	}
	at := time.Now().Add(time.Hour)
	armRTCWakeAlarm(at)
	rt.write(t, rt.alarm, "")
	platformSyncWakeAlarm()
	if got := rt.read(t); !near(got, at.Unix()) {
		t.Fatalf("after the sync wakealarm %d, want about %d", got, at.Unix())
	}
	rt.write(t, rt.alarm, "12345")
	platformSyncWakeAlarm()
	if got := rt.read(t); !near(got, at.Unix()) {
		t.Fatalf("after a change and the sync wakealarm %d, want about %d", got, at.Unix())
	}
}

// With no hardware clock that has a wake alarm, arming carries on (the timer
// still covers suspend) and the Wake option says why it cannot work.
func TestRTCWakeAlarmMissing(t *testing.T) {
	newRTCTest(t, false)
	gen := armRTCWakeAlarm(time.Now().Add(time.Hour))
	platformSyncWakeAlarm()
	dropRTCWakeAlarm(gen)
	if info := platformWakeTimerInfo(); info.Supported {
		t.Errorf("no rtc, yet supported: %+v", info)
	}
}

// The clock with a wake alarm is used when rtc0 has none.
func TestFindWakeAlarmOtherClock(t *testing.T) {
	rt := newRTCTest(t, false)
	if err := os.MkdirAll(filepath.Join(rt.dir, "rtc0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(rt.dir, "rtc1"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(rt.dir, "rtc1", "wakealarm")
	rt.write(t, want, "")
	got, err := findWakeAlarm()
	if err != nil || got != want {
		t.Fatalf("findWakeAlarm = %q, %v; want %q", got, err, want)
	}
	if info := platformWakeTimerInfo(); !info.Supported {
		t.Errorf("rtc1 has a wake alarm, yet not supported: %+v", info)
	}
}
