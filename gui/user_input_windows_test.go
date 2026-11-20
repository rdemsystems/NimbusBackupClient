//go:build windows

package main

import "testing"

// The helper started as a standard user printed a warning before the number
// (it could not open the service's log): the last line is the answer.
func TestParseLastInputOutput(t *testing.T) {
	for _, c := range []struct {
		out  string
		ms   uint32
		fail bool
	}{
		{"159890\r\n", 159890, false},
		{"Failed to create backup logger: open C:\\ProgramData\\Salvanza\\backup-service.log: Access is denied.\r\n42\r\n", 42, false},
		{"", 0, true},
		{"Access is denied.\r\n", 0, true},
	} {
		ms, err := parseLastInputOutput(c.out)
		if (err != nil) != c.fail || ms != c.ms {
			t.Errorf("%q: %d, %v; want %d, failure %v", c.out, ms, err, c.ms, c.fail)
		}
	}
}
