//go:build !windows && !linux

package main

func platformSignedInSessions() (map[string]bool, bool) { return nil, false }
