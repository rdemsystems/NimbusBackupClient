package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeNet stands in for the network and the clock: the first failures tries
// find the server out of reach, and a pause moves the clock on at once.
type fakeNet struct {
	now      time.Time
	failures int
	dials    []string
}

func newFakeNet(t *testing.T, now time.Time, failures int) *fakeNet {
	t.Helper()
	f := &fakeNet{now: now, failures: failures}
	oldDial, oldNow, oldSleep := dialServer, netNow, netSleep
	dialServer = func(_ context.Context, addr string) error {
		f.dials = append(f.dials, addr)
		if len(f.dials) <= f.failures {
			return errors.New("connectex: A socket operation was attempted to an unreachable host")
		}
		return nil
	}
	netNow = func() time.Time { return f.now }
	netSleep = func(d time.Duration) { f.now = f.now.Add(d) }
	noteResume(time.Time{}, "")
	t.Cleanup(func() {
		dialServer, netNow, netSleep = oldDial, oldNow, oldSleep
		noteResume(time.Time{}, "")
	})
	return f
}

func appWithServer() *App {
	return &App{config: &Config{BaseURL: "https://192.168.0.242:8007"}}
}

// Found on a laptop: woken with the power button, the held run started a
// second after the wake and failed, the network not yet back. It now waits
// for its server.
func TestNetworkWaitAfterWakeUntilTheServerAnswers(t *testing.T) {
	woke := time.Date(2020, 3, 1, 7, 40, 0, 0, time.Local)
	start := woke.Add(time.Second)
	f := newFakeNet(t, start, 3)
	noteSchedulerTick(woke.Add(-8*time.Hour), woke) // asleep overnight
	if err := appWithServer().waitForServerAfterWake(batterySet("plugged", 0, 0), "scheduled"); err != nil {
		t.Fatalf("the server answered, yet the run is not to start: %v", err)
	}
	if len(f.dials) != 4 || f.dials[0] != "192.168.0.242:8007" {
		t.Fatalf("tries %v, want four to 192.168.0.242:8007", f.dials)
	}
	if waited := f.now.Sub(start); waited != 3*networkRetryEvery {
		t.Errorf("waited %s, want %s", waited, 3*networkRetryEvery)
	}
}

// A server still out of reach after two minutes: the wait ends with the last
// try's error.
func TestNetworkWaitGivesUpAfterTwoMinutes(t *testing.T) {
	start := time.Date(2020, 3, 1, 7, 40, 0, 0, time.Local)
	f := newFakeNet(t, start, 1000)
	reachable, waited, err := waitForServer("192.168.0.242:8007", "192.168.0.242:8007", "after the computer woke")
	if reachable || err == nil {
		t.Fatalf("an unreachable server was reported reachable (err %v)", err)
	}
	if waited != networkWaitMax || f.now.Sub(start) != networkWaitMax {
		t.Errorf("waited %s (clock moved %s), want %s", waited, f.now.Sub(start), networkWaitMax)
	}
	if want := int(networkWaitMax/networkRetryEvery) + 1; len(f.dials) != want {
		t.Errorf("%d tries, want %d", len(f.dials), want)
	}
}

// #142: a run woken for whose server is still out of reach after the wait is
// not started, and says why: it used to start anyway and fail at its login
// with "no such host", which did not say the server was never reached.
func TestNetworkWaitUnreachableAfterWakeStopsTheRun(t *testing.T) {
	woke := time.Date(2020, 3, 1, 16, 1, 39, 0, time.Local)
	newFakeNet(t, woke, 1000)
	noteSchedulerTick(woke.Add(-2*time.Hour), woke)
	err := appWithServer().waitForServerAfterWake(batterySet("", 0, 0), "scheduled")
	if err == nil {
		t.Fatal("the run was let start with its server out of reach")
	}
	for _, want := range []string{"did not start", "192.168.0.242:8007", "2m 0s after the computer woke", "unreachable host"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%q does not say %q", err, want)
		}
	}
}

func TestNetworkNoWaitWhenTheServerAnswersAtOnce(t *testing.T) {
	start := time.Date(2020, 3, 1, 7, 40, 0, 0, time.Local)
	f := newFakeNet(t, start, 0)
	noteResume(start, "after the service started")
	if err := appWithServer().waitForServerAfterWake(batterySet("", 0, 0), "startup"); err != nil {
		t.Fatalf("a server that answered stopped the run: %v", err)
	}
	if len(f.dials) != 1 || !f.now.Equal(start) {
		t.Fatalf("tries %v, clock moved %s", f.dials, f.now.Sub(start))
	}
}

// Run now, a run long after the wake, and a run with no wake or start before
// it go straight on, whatever the server.
func TestNetworkNoWaitForRunNowOrLongAfterAWake(t *testing.T) {
	woke := time.Date(2020, 3, 1, 7, 40, 0, 0, time.Local)
	f := newFakeNet(t, woke.Add(10*time.Second), 1000)
	job := batterySet("", 0, 0)
	a := appWithServer()

	goesOn := func(what string, err error) {
		t.Helper()
		if err != nil {
			t.Errorf("%s: the run was stopped: %v", what, err)
		}
	}
	goesOn("nothing woke or started yet", a.waitForServerAfterWake(job, "scheduled"))
	noteResume(woke, "after the computer woke")
	goesOn("Run now", a.waitForServerAfterWake(job, "manual"))
	goesOn("no server to try", (&App{}).waitForServerAfterWake(job, "scheduled"))
	f.now = woke.Add(networkWaitWindow + time.Second)
	goesOn("long after the wake", a.waitForServerAfterWake(job, "scheduled"))
	if len(f.dials) != 0 {
		t.Fatalf("tried the server %d times", len(f.dials))
	}
}

func TestServerHostPort(t *testing.T) {
	for _, c := range [][2]string{
		{"https://192.168.0.242:8007", "192.168.0.242:8007"},
		{"https://pbs.example.com", "pbs.example.com:443"},
		{"http://pbs.example.com", "pbs.example.com:80"},
		{"pbs.example.com:8007", "pbs.example.com:8007"},
		{"https://[2001:db8::1]:8007/", "[2001:db8::1]:8007"},
		{"", ""},
		{"https://", ""},
	} {
		if got := serverHostPort(c[0]); got != c[1] {
			t.Errorf("serverHostPort(%q) = %q, want %q", c[0], got, c[1])
		}
	}
}

func TestJobServerAddressIsTheSetsServer(t *testing.T) {
	a := &App{config: &Config{DefaultPBSID: "main", PBSServers: map[string]*PBSServer{
		"main": {ID: "main", Name: "Main", BaseURL: "https://pbs.example.com"},
		"off":  {ID: "off", Name: "Offsite", BaseURL: "https://offsite.example.net:8007"},
	}}}
	job := batterySet("", 0, 0)
	job.PBSServerID = "off"
	if name, addr := a.jobServerAddress(job); addr != "offsite.example.net:8007" || name != "Offsite (offsite.example.net:8007)" {
		t.Errorf("the set's server: %q %q", name, addr)
	}
	job.PBSServerID = ""
	if name, addr := a.jobServerAddress(job); addr != "pbs.example.com:443" || name != "Main (pbs.example.com:443)" {
		t.Errorf("the default server: %q %q", name, addr)
	}
}
