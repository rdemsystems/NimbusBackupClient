//go:build !windows && !linux

package main

import "time"

func platformLastUserInput() (time.Time, bool) { return time.Time{}, false }
