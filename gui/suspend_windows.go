//go:build windows

package main

import "fmt"

var procSetSuspendState = powrprof.NewProc("SetSuspendState")

// platformSleepComputer puts the computer to sleep, as the Start menu's Sleep
// does: SetSuspendState(FALSE, FALSE, FALSE) is sleep, not hibernate, and
// leaves wake events on, so the wake timer for the set's next run still wakes
// it. The call needs SeShutdownPrivilege, which SYSTEM (the service) and an
// interactive user hold but Windows keeps disabled until asked.
func platformSleepComputer() error {
	if err := enablePrivileges("SeShutdownPrivilege"); err != nil {
		return err
	}
	if r, _, err := procSetSuspendState.Call(0, 0, 0); r == 0 {
		return fmt.Errorf("SetSuspendState: %v", err)
	}
	return nil
}
