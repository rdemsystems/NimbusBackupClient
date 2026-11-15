//go:build !windows && !linux

package pbscommon

import "fmt"

// SleepComputer is not supported here (Windows: suspend_windows.go, Linux:
// suspend_linux.go).
func SleepComputer() error {
	return fmt.Errorf("putting the computer to sleep is not supported on this system")
}

// HibernateComputer is not supported here.
func HibernateComputer() error {
	return fmt.Errorf("hibernating the computer is not supported on this system")
}

// CanHibernate: not on this system.
func CanHibernate() (ok bool, why string) {
	return false, HibernateNone
}
