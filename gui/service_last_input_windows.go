//go:build service && windows

package main

import (
	"fmt"
	"os"
	"strconv"
)

// isLastInputHelper: this program was started as the -last-input helper.
func isLastInputHelper(args []string) bool { return len(args) == 2 && args[1] == lastInputArg }

// runLastInputArg: -last-input is the service program run by the service
// in a signed-in session (user_input_windows.go). It prints the milliseconds
// since that session's last keyboard or mouse input and exits, 0 told, 1 not.
func runLastInputArg(args []string) {
	if !isLastInputHelper(args) {
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
