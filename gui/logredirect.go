package main

// logredirect.go routes Go's standard log package into the rotating log the
// rest of the client writes to (writeDebugLog).
//
// A Windows service has no console: whatever a process started by the SCM
// writes to os.Stderr is discarded. The stdlib log package defaults to
// os.Stderr, so before this file every log.Printf in the service process
// (machinebackuplib's progress and padding notes, log.Fatal in
// service_main.go, third-party packages) was written nowhere.
//
// No recursion: writeLogToLogger writes to the rotating logger and to
// os.Stderr with fmt, and never calls the stdlib log package.
//
// Adapted from CJInvent/NimbusBackupClient 729abbd.

import (
	"bytes"
	"io"
	"log"
)

// debugLogWriter adapts writeDebugLog to io.Writer so it can back log.SetOutput.
type debugLogWriter struct{}

// Write forwards each line to the rotating log. A multi-line message (a
// formatted error, a panic dump) arrives as one Write; splitting keeps every
// line on its own timestamped row.
func (debugLogWriter) Write(p []byte) (int, error) {
	for _, line := range bytes.Split(bytes.TrimRight(p, "\n"), []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		writeDebugLog(string(line))
	}
	// Always report the whole input as consumed: a short count would make the
	// log package report an error about the logger itself.
	return len(p), nil
}

// redirectStdlibLog points the standard log package at the rotating log. It is
// the first statement of both main() functions; anything logged before it
// goes to stderr. The flags are cleared because writeLogToLogger already
// stamps every line with its own prefix and timestamp.
func redirectStdlibLog() {
	log.SetFlags(0)
	log.SetOutput(debugLogWriter{})
}

var _ io.Writer = debugLogWriter{}
