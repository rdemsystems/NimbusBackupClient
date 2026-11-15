package clientcommon

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"testing"
	"time"
)

// -then, -then-after and -shutdown-after given override the config file;
// left out, the file's values stand. -shutdown-after is -then shutdown.
func TestAfterRunFlags(t *testing.T) {
	for _, c := range []struct {
		file   string
		args   []string
		action string
		wait   int
		bad    bool
	}{
		{`{}`, nil, "", 0, false},
		{`{}`, []string{"-then", "sleep"}, ThenSleep, 0, false},
		{`{}`, []string{"-then", "hibernate", "-then-after", "15"}, ThenHibernate, 15, false},
		{`{}`, []string{"-then", "leave"}, "", 0, false},
		{`{}`, []string{"-then", "SHUTDOWN"}, ThenShutdown, 0, false},
		// the older name
		{`{}`, []string{"-shutdown-after"}, ThenShutdown, 0, false},
		{`{"shutdown-after": true}`, nil, ThenShutdown, 0, false},
		{`{"shutdown-after": true}`, []string{"-shutdown-after=false"}, "", 0, false},
		{`{"shutdown-after": false}`, []string{"-shutdown-after"}, ThenShutdown, 0, false},
		{`{"then": "shutdown"}`, []string{"-shutdown-after=false"}, "", 0, false},
		{`{"then": "sleep"}`, []string{"-shutdown-after=false"}, ThenSleep, 0, false},
		{`{"then": "sleep"}`, []string{"-shutdown-after"}, ThenShutdown, 0, false},
		{`{"shutdown-after": true}`, []string{"-then", "sleep"}, ThenSleep, 0, false},
		{`{"shutdown-after": true}`, []string{"-then", "leave"}, "", 0, false},
		{`{}`, []string{"-shutdown-after", "-then", "shutdown"}, ThenShutdown, 0, false},
		// the file's own
		{`{"then": "sleep", "then-after": 30}`, nil, ThenSleep, 30, false},
		{`{"then": "sleep", "then-after": 30}`, []string{"-then-after", "0"}, ThenSleep, 0, false},
		// refused
		{`{}`, []string{"-shutdown-after", "-then", "sleep"}, "", 0, true},
		{`{}`, []string{"-then", "reboot"}, "", 0, true},
		{`{}`, []string{"-then", "sleep", "-then-after", "241"}, "", 0, true},
		{`{}`, []string{"-then-after", "-1"}, "", 0, true},
		{`{"then": "standby"}`, nil, "", 0, true},
	} {
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		apply := AfterRunFlags(fs)
		if err := fs.Parse(c.args); err != nil {
			t.Fatal(err)
		}
		var a AfterRun
		if err := json.Unmarshal([]byte(c.file), &a); err != nil {
			t.Fatal(err)
		}
		err := apply(&a)
		if c.bad {
			if err == nil {
				t.Errorf("file %s, args %v: accepted", c.file, c.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("file %s, args %v: %v", c.file, c.args, err)
			continue
		}
		if a.Action() != c.action || a.ThenAfter != c.wait {
			t.Errorf("file %s, args %v: %q after %d, want %q after %d", c.file, c.args, a.Action(), a.ThenAfter, c.action, c.wait)
		}
	}
}

// ActAfterRun does only what it is told, after the wait, and a refusal is
// reported, not fatal.
func TestActAfterRun(t *testing.T) {
	var calls []string
	var waited time.Duration
	var fail error
	oldShut, oldSleep, oldHib, oldWait := shutdownComputer, sleepComputer, hibernateComputer, waitBeforeThen
	shutdownComputer = func(string) (string, error) { calls = append(calls, "shutdown"); return "a test", fail }
	sleepComputer = func() error { calls = append(calls, "sleep"); return fail }
	hibernateComputer = func() error { calls = append(calls, "hibernate"); return fail }
	waitBeforeThen = func(d time.Duration) { waited += d }
	t.Cleanup(func() {
		shutdownComputer, sleepComputer, hibernateComputer, waitBeforeThen = oldShut, oldSleep, oldHib, oldWait
	})

	ActAfterRun(AfterRun{})
	ActAfterRun(AfterRun{Then: ThenLeave, ThenAfter: 5})
	if len(calls) != 0 || waited != 0 {
		t.Fatalf("acted (%v) or waited (%s) when told to leave the computer", calls, waited)
	}
	ActAfterRun(AfterRun{ShutdownAfter: true})
	ActAfterRun(AfterRun{Then: ThenSleep, ThenAfter: 10})
	ActAfterRun(AfterRun{Then: ThenHibernate})
	if len(calls) != 3 || calls[0] != "shutdown" || calls[1] != "sleep" || calls[2] != "hibernate" {
		t.Errorf("calls %v", calls)
	}
	if waited != 10*time.Minute {
		t.Errorf("waited %s, want 10 minutes", waited)
	}
	fail = errors.New("refused")
	ActAfterRun(AfterRun{Then: ThenShutdown}) // must return, not exit
	if len(calls) != 4 {
		t.Errorf("%d requests, want 4", len(calls))
	}
}
