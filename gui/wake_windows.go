//go:build windows

package main

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// A waitable timer with resume set wakes the system from sleep when it comes
// due, as Task Scheduler's "Wake the computer to run this task" does. It lives
// as long as its handle: the scheduler's process holds it, and cancelling or
// closing it (or the process ending) drops the wake.
var (
	kernel32Wake            = windows.NewLazySystemDLL("kernel32.dll")
	procCreateWaitableTimer = kernel32Wake.NewProc("CreateWaitableTimerW")
	procSetWaitableTimer    = kernel32Wake.NewProc("SetWaitableTimer")
	procCancelWaitableTimer = kernel32Wake.NewProc("CancelWaitableTimer")

	powrprof                 = windows.NewLazySystemDLL("powrprof.dll")
	procPowerGetActiveScheme = powrprof.NewProc("PowerGetActiveScheme")
	procPowerReadACValue     = powrprof.NewProc("PowerReadACValueIndex")
	procPowerReadDCValue     = powrprof.NewProc("PowerReadDCValueIndex")
	procLocalFree            = kernel32Wake.NewProc("LocalFree")
	procSetLastError         = kernel32Wake.NewProc("SetLastError")
)

// The power plan's Sleep > "Allow wake timers" setting (RTCWAKE): 0 disable,
// 1 enable, 2 important wake timers only (the system's own; ours is not one).
var (
	guidSubSleep       = windows.GUID{Data1: 0x238c9fa8, Data2: 0x0aad, Data3: 0x41ed, Data4: [8]byte{0x83, 0xf4, 0x97, 0xbe, 0x24, 0x2c, 0x8f, 0x20}}
	guidAllowWakeTimer = windows.GUID{Data1: 0xbd3b718a, Data2: 0x0680, Data3: 0x4d9d, Data4: [8]byte{0x8a, 0xb2, 0xe1, 0xd2, 0xb4, 0xac, 0x80, 0x6d}}
)

const errorNotSupported = 50 // ERROR_NOT_SUPPORTED: the timer is set, but cannot wake this system

func platformArmWakeTimer(at time.Time) (func(), error) {
	h, _, err := procCreateWaitableTimer.Call(0, 1, 0) // manual reset, unnamed
	if h == 0 {
		return nil, fmt.Errorf("CreateWaitableTimer: %v", err)
	}
	handle := windows.Handle(h)
	// An absolute due time is a positive FILETIME (100 ns since 1601, UTC).
	due := windows.NsecToFiletime(at.UnixNano())
	dueTime := int64(due.HighDateTime)<<32 | int64(due.LowDateTime)
	// SetWaitableTimer succeeds but sets ERROR_NOT_SUPPORTED when the system
	// cannot resume: clear the last error first, on this one OS thread, so a
	// stale one is not taken for that.
	runtime.LockOSThread()
	_, _, _ = procSetLastError.Call(0)
	ok, _, err := procSetWaitableTimer.Call(h, uintptr(unsafe.Pointer(&dueTime)), 0, 0, 0, 1) // fResume = TRUE
	runtime.UnlockOSThread()
	if ok == 0 {
		_ = windows.CloseHandle(handle)
		return nil, fmt.Errorf("SetWaitableTimer: %v", err)
	}
	if errno, isErrno := err.(windows.Errno); isErrno && errno == errorNotSupported {
		_, _, _ = procCancelWaitableTimer.Call(h)
		_ = windows.CloseHandle(handle)
		return nil, fmt.Errorf("this computer does not support waking from sleep by a timer")
	}
	return func() {
		_, _, _ = procCancelWaitableTimer.Call(h)
		_ = windows.CloseHandle(handle)
	}, nil
}

// readWakeTimerSetting reads Allow wake timers from the active power plan, on
// mains (ac) and on battery; ok is false when it cannot be read.
func readWakeTimerSetting() (ac, dc uint32, ok bool) {
	var scheme *windows.GUID
	if r, _, _ := procPowerGetActiveScheme.Call(0, uintptr(unsafe.Pointer(&scheme))); r != 0 || scheme == nil {
		return 0, 0, false
	}
	defer func() { _, _, _ = procLocalFree.Call(uintptr(unsafe.Pointer(scheme))) }() // PowerGetActiveScheme allocated it
	r1, _, _ := procPowerReadACValue.Call(0, uintptr(unsafe.Pointer(scheme)), uintptr(unsafe.Pointer(&guidSubSleep)), uintptr(unsafe.Pointer(&guidAllowWakeTimer)), uintptr(unsafe.Pointer(&ac)))
	r2, _, _ := procPowerReadDCValue.Call(0, uintptr(unsafe.Pointer(scheme)), uintptr(unsafe.Pointer(&guidSubSleep)), uintptr(unsafe.Pointer(&guidAllowWakeTimer)), uintptr(unsafe.Pointer(&dc)))
	return ac, dc, r1 == 0 && r2 == 0
}

var procGetPwrCapabilities = powrprof.NewProc("GetPwrCapabilities")

// canWakeByTimer reads SYSTEM_POWER_CAPABILITIES: a sleep state to wake from
// (S1 to S4, or Modern Standby) and a clock that can wake it (a wake alarm,
// or RtcWake deeper than working). A VM without sleep states has neither.
// modern is Modern Standby (AoAc) with no classic sleep (S1 to S3): there
// Windows itself decides whether a program's timer may wake the computer,
// and often it does not, above all on battery. ok is false when the
// capabilities cannot be read.
func canWakeByTimer() (can, modern, ok bool) {
	var caps [128]byte // SYSTEM_POWER_CAPABILITIES is 76 bytes
	if r, _, _ := procGetPwrCapabilities.Call(uintptr(unsafe.Pointer(&caps[0]))); r == 0 {
		return false, false, false
	}
	const (
		offS1, offS2, offS3, offS4 = 3, 4, 5, 6
		offWakeAlarm, offAoAc      = 19, 20
		offRtcWake                 = 64 // SYSTEM_POWER_STATE, after BatteryScale[3]
	)
	classic := caps[offS1] != 0 || caps[offS2] != 0 || caps[offS3] != 0
	sleeps := classic || caps[offS4] != 0 || caps[offAoAc] != 0
	rtcWake := uint32(caps[offRtcWake]) | uint32(caps[offRtcWake+1])<<8 | uint32(caps[offRtcWake+2])<<16 | uint32(caps[offRtcWake+3])<<24
	clock := caps[offWakeAlarm] != 0 || rtcWake >= 2 || caps[offAoAc] != 0 // 2 = PowerSystemSleeping1
	return sleeps && clock, caps[offAoAc] != 0 && !classic, true
}

func platformWakeTimerInfo() WakeTimerInfo {
	can, modern, ok := canWakeByTimer()
	if ok && !can {
		return WakeTimerInfo{Note: "no sleep state a timer can wake this computer from (a VM, or the firmware does not offer one)"}
	}
	ac, dc, ok := readWakeTimerSetting()
	if !ok {
		// No such setting (some editions): the system decides; assume it allows.
		return WakeTimerInfo{Supported: true, OnMains: true, OnBattery: true, ModernStandby: modern}
	}
	return WakeTimerInfo{Supported: true, OnMains: ac == 1, OnBattery: dc == 1, ModernStandby: modern,
		Note: fmt.Sprintf("Allow wake timers: on mains %d, on battery %d (0 off, 1 on, 2 important only); Modern Standby %v", ac, dc, modern)}
}
