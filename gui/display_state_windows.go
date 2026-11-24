//go:build windows

package main

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Watching the screen's state (standby_defer.go). Windows reports it as the
// power setting GUID_CONSOLE_DISPLAY_STATE: 0 off, 1 on, 2 dimmed. Asked
// for through powrprof's PowerSettingRegisterNotification with a callback,
// which needs neither a window nor the service's own control handler, so it
// works the same in the service and in the app. Windows sends the current
// state when the request is made, and each change after it.

var (
	modPowrprof                          = windows.NewLazySystemDLL("powrprof.dll")
	procPowerSettingRegisterNotification = modPowrprof.NewProc("PowerSettingRegisterNotification")
)

// guidConsoleDisplayState is GUID_CONSOLE_DISPLAY_STATE,
// {6FE69556-704A-47A0-8F24-C28D936FDA47}.
var guidConsoleDisplayState = windows.GUID{Data1: 0x6fe69556, Data2: 0x704a, Data3: 0x47a0,
	Data4: [8]byte{0x8f, 0x24, 0xc2, 0x8d, 0x93, 0x6f, 0xda, 0x47}}

const (
	deviceNotifyCallback  = 2      // DEVICE_NOTIFY_CALLBACK
	pbtPowerSettingChange = 0x8013 // PBT_POWERSETTINGCHANGE
)

// powerBroadcastSetting is POWERBROADCAST_SETTING with room for the DWORD
// this setting carries.
type powerBroadcastSetting struct {
	PowerSetting windows.GUID
	DataLength   uint32
	Data         uint32
}

// deviceNotifySubscribeParameters is DEVICE_NOTIFY_SUBSCRIBE_PARAMETERS. It
// is kept for as long as the program runs: Windows holds on to it.
type deviceNotifySubscribeParameters struct {
	Callback uintptr
	Context  uintptr
}

var (
	displayWatchOnce   sync.Once
	displayWatchParams deviceNotifySubscribeParameters
	displayWatchHandle uintptr
)

// displayStateCallback receives each report of the screen's state.
func displayStateCallback(context, kind uintptr, setting *powerBroadcastSetting) uintptr {
	if kind != pbtPowerSettingChange || setting == nil || setting.PowerSetting != guidConsoleDisplayState || setting.DataLength < 4 {
		return 0
	}
	on := setting.Data != 0 // 1 on, 2 dimmed (still on); 0 off
	noteDisplayState(on, time.Now())
	return 0
}

// startDisplayWatch asks Windows for the screen's state, once, on a computer
// with Modern Standby: elsewhere it changes nothing and is not asked for.
func startDisplayWatch() {
	displayWatchOnce.Do(func() {
		if !isModernStandby() {
			return
		}
		if err := registerDisplayWatch(); err != nil {
			writeDebugLog(fmt.Sprintf("[Wake] %v: a run on this Modern Standby computer goes on as before", err))
			return
		}
		writeDebugLog("[Wake] this computer has Modern Standby: an automatic run whose server cannot be reached while the screen is off waits for the screen to come on")
	})
}

// registerDisplayWatch asks Windows to report the screen's state to
// displayStateCallback.
func registerDisplayWatch() error {
	if err := procPowerSettingRegisterNotification.Find(); err != nil {
		return fmt.Errorf("this Windows cannot report the screen's state (%v)", err)
	}
	displayWatchParams.Callback = windows.NewCallback(displayStateCallback)
	r, _, _ := procPowerSettingRegisterNotification.Call(
		uintptr(unsafe.Pointer(&guidConsoleDisplayState)),
		deviceNotifyCallback,
		uintptr(unsafe.Pointer(&displayWatchParams)),
		uintptr(unsafe.Pointer(&displayWatchHandle)))
	if r != 0 {
		return fmt.Errorf("Windows did not take the request for the screen's state (error %d)", r)
	}
	return nil
}
