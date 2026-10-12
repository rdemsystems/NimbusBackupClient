package main

import "testing"

func TestDebugLogWriterConsumesWholeInput(t *testing.T) {
	in := []byte("first line\n\nsecond line\n")
	n, err := debugLogWriter{}.Write(in)
	if err != nil || n != len(in) {
		t.Fatalf("Write = (%d, %v), want (%d, nil)", n, err, len(in))
	}
}
