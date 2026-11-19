//go:build service && windows

package main

import (
	"fmt"
	"os"
	"strconv"
)

// runLastInputArg: -last-input is the service program run by the service
// in a signed-in session (user_input_windows.go). It prints the milliseconds
// since that session's last keyboard or mouse input and exits, 0 told, 1 not.
func runLastInputArg(args []string) {
	if len(args) != 2 || args[1] != lastInputArg {
		return
	}
	ms, err := sessionIdleMillis()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(strconv.FormatUint(uint64(ms), 10))
	os.Exit(0)
}
