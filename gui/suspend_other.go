//go:build !windows && !linux

package main

import "fmt"

func platformSleepComputer() error {
	return fmt.Errorf("putting the computer to sleep is not supported on this system")
}
