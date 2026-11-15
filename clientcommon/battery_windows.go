//go:build windows

package clientcommon

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGetSystemPowerStatus = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetSystemPowerStatus")

// systemPowerStatus is SYSTEM_POWER_STATUS.
type systemPowerStatus struct {
	ACLineStatus       byte      // 0 offline (on battery), 1 online, 255 unknown
	BatteryFlag        byte      // 128 no system battery, 255 unknown
	BatteryLifePercent byte      // 0 to 100, 255 unknown
	_                  byte      // SystemStatusFlag
	_                  [2]uint32 // BatteryLifeTime, BatteryFullLifeTime
}

func platformReadBattery() BatteryStatus {
	var s systemPowerStatus
	if r, _, _ := procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&s))); r == 0 {
		return BatteryStatus{}
	}
	return batteryFromPowerStatus(s)
}

// batteryFromPowerStatus reads SYSTEM_POWER_STATUS. A battery flag of 128 is
// no battery (a desktop, a VM); 255, unknown, counts as none too, so nothing
// is held back on a guess.
func batteryFromPowerStatus(s systemPowerStatus) BatteryStatus {
	if s.BatteryFlag&128 != 0 || s.BatteryFlag == 255 {
		return BatteryStatus{}
	}
	st := BatteryStatus{HasBattery: true, OnBattery: s.ACLineStatus == 0}
	if s.BatteryLifePercent <= 100 {
		st.Percent, st.Known = int(s.BatteryLifePercent), true
	}
	return st
}
