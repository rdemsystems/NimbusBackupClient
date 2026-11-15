//go:build windows

package pbscommon

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	powrprofSuspend        = windows.NewLazySystemDLL("powrprof.dll")
	procSetSuspendState    = powrprofSuspend.NewProc("SetSuspendState")
	procGetPwrCapabilities = powrprofSuspend.NewProc("GetPwrCapabilities")
)

// SleepComputer puts the computer to sleep, as the Start menu's Sleep does:
// SetSuspendState(FALSE, FALSE, FALSE) is sleep, not hibernate, and leaves
// wake events on, so a wake timer for the next backup still wakes it. The call
// needs SeShutdownPrivilege, which SYSTEM (the service) and an interactive
// user hold but Windows keeps disabled until asked.
func SleepComputer() error {
	return setSuspendState(false)
}

// HibernateComputer hibernates the computer: SetSuspendState(TRUE, FALSE,
// FALSE), with wake events left on, so a wake timer can usually still wake it.
func HibernateComputer() error {
	return setSuspendState(true)
}

func setSuspendState(hibernate bool) error {
	if err := enableShutdownPrivilege(); err != nil {
		return err
	}
	h := uintptr(0)
	if hibernate {
		h = 1
	}
	if r, _, err := procSetSuspendState.Call(h, 0, 0); r == 0 {
		return fmt.Errorf("SetSuspendState: %v", err)
	}
	return nil
}

// CanHibernate reads SYSTEM_POWER_CAPABILITIES: SystemS4 is the hibernate
// state, HiberFilePresent whether hibernation is turned on (powercfg
// /hibernate on). With hibernation off Windows may report neither, so a
// computer that has any sleep state is taken to have hibernation turned off
// rather than missing; one with none (a VM) cannot hibernate at all.
func CanHibernate() HibernateCheck {
	var caps [128]byte // SYSTEM_POWER_CAPABILITIES is 76 bytes
	if r, _, _ := procGetPwrCapabilities.Call(uintptr(unsafe.Pointer(&caps[0]))); r == 0 {
		return HibernateCheck{Why: HibernateNone}
	}
	const (
		offS1, offS2, offS3, offS4 = 3, 4, 5, 6
		offHiberFile, offAoAc      = 8, 20
	)
	if caps[offS4] != 0 && caps[offHiberFile] != 0 {
		return HibernateCheck{OK: true}
	}
	if caps[offS1] != 0 || caps[offS2] != 0 || caps[offS3] != 0 || caps[offS4] != 0 || caps[offAoAc] != 0 {
		return HibernateCheck{Why: HibernateOff}
	}
	return HibernateCheck{Why: HibernateNone}
}
