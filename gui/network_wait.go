package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Waiting for the network after a wake. When the computer wakes from sleep,
// or this program starts with the computer, the network can take a while to
// come back: Wi-Fi has to find its network, join it and get an address. A
// run started in that time failed at once. Found on a laptop 2026-11-15:
// woken with the power button just after it was plugged in, the run held back
// for the battery started on the first scheduler tick, a second after the
// wake, and failed with "A socket operation was attempted to an unreachable
// host".
//
// So an automatic run (scheduled, startup, or a held run released) that
// starts soon after a wake or a start first waits until its PBS server can be
// reached, for at most networkWaitMax. A server still out of reach then is
// down or the network is not coming back: the run is not started, and fails
// saying so (#142: started anyway, it failed at its login with an error that
// did not say the server was never reached, ran the set's "before"
// application for nothing and wrote no log of its own). Run now never waits:
// the user is there and can see whether the network is up. The wait is in the
// run's own goroutine, so other sets' runs and the scheduler's ticks carry on.
// The computer is kept awake while it waits (executeScheduledJob): left
// alone, a computer woken by a timer sleeps again within minutes.

var (
	// networkWaitWindow: a run starting within this long of a wake or a
	// start waits for its server. Later than that the network has had its
	// time, and a server that does not answer is down, not coming up: the
	// run fails at once, as before.
	networkWaitWindow = 5 * time.Minute
	// networkWaitMax is the longest a run waits for its server.
	networkWaitMax = 2 * time.Minute
	// networkRetryEvery is the pause between tries.
	networkRetryEvery = 5 * time.Second
	// networkDialTimeout bounds one try: an address that is not answering
	// at all takes the full time, so it is kept short.
	networkDialTimeout = 3 * time.Second
)

// dialServer tries a TCP connection to addr (host:port) and closes it: the
// cheapest check that the server can be reached, with no login and nothing
// asked of it. A variable, like netNow and netSleep, so the tests run without
// a network or a real clock.
var dialServer = func(ctx context.Context, addr string) error {
	var d net.Dialer
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return c.Close()
}

var (
	netNow   = time.Now
	netSleep = time.Sleep
)

// lastResume is when the computer last woke from sleep, or this program's
// scheduler started, and which, for the log.
var lastResume struct {
	sync.Mutex
	at  time.Time
	why string
}

// noteResume records that the computer woke, or the scheduler started, at
// at; why says which in the log ("after the computer woke").
func noteResume(at time.Time, why string) {
	lastResume.Lock()
	lastResume.at, lastResume.why = at, why
	lastResume.Unlock()
}

// resumedRecently returns why a run starting at now waits for its server
// ("after the computer woke"), or "" when it need not. Wall clock times are
// compared, as in noteSchedulerTick.
func resumedRecently(now time.Time) string {
	lastResume.Lock()
	defer lastResume.Unlock()
	if lastResume.at.IsZero() || now.Round(0).Sub(lastResume.at.Round(0)) > networkWaitWindow {
		return ""
	}
	return lastResume.why
}

// serverHostPort is the host:port a PBS address is reached at: the port
// written in it, or the scheme's own (443 for https), as the backup's HTTP
// client takes it. An address written without a scheme is https, as PBS
// always is. "" when there is no host to try.
func serverHostPort(baseURL string) string {
	s := strings.TrimSpace(baseURL)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	port := u.Port()
	if port == "" {
		port = "443"
		if strings.EqualFold(u.Scheme, "http") {
			port = "80"
		}
	}
	return net.JoinHostPort(u.Hostname(), port)
}

// jobServerAddress is the host:port of the PBS server job backs up to and a
// name for it in the log, or "" when it cannot be told: the run then finds
// out for itself, as before. Only the stored address is read; nothing is
// unlocked or logged in to.
func (a *App) jobServerAddress(job ScheduledJob) (name, addr string) {
	if a.config == nil || a.delegated() {
		return "", ""
	}
	base := ""
	if job.PBSServerID != "" {
		pbs, err := a.config.GetPBSServer(job.PBSServerID)
		if err != nil {
			return "", ""
		}
		base, name = pbs.BaseURL, pbs.Name
	} else {
		base = a.config.EffectivePBS().BaseURL
		if pbs, err := a.config.GetPBSServer(""); err == nil && pbs.BaseURL == base {
			name = pbs.Name
		}
	}
	addr = serverHostPort(base)
	if addr == "" {
		return "", ""
	}
	if name == "" {
		return addr, addr
	}
	return fmt.Sprintf("%s (%s)", name, addr), addr
}

// waitForServerAfterWake holds an automatic run of job, about to start soon
// after the computer woke or this program started, until its PBS server can
// be reached (waitForServer). Run now, and runs at other times, go straight
// on. It returns why the run must not start when the server could not be
// reached in that time, nil otherwise.
func (a *App) waitForServerAfterWake(job ScheduledJob, trigger string) error {
	if trigger == "manual" {
		return nil
	}
	why := resumedRecently(netNow())
	if why == "" {
		return nil
	}
	name, addr := a.jobServerAddress(job)
	if addr == "" {
		return nil
	}
	reachable, waited, err := waitForServer(name, addr, why)
	if reachable {
		return nil
	}
	return fmt.Errorf("the backup did not start: %s could not be reached in the %s %s (%v)", name, formatDuration(waited.Round(time.Second)), why, err)
}

// waitForServer tries addr every networkRetryEvery until it answers or
// networkWaitMax has gone by, and reports whether it answered, how long that
// took and, when it did not, the last try's error. A server that answers at
// once is not logged; one that does not is logged once when the wait begins
// and once when it ends.
func waitForServer(name, addr, why string) (reachable bool, waited time.Duration, lastErr error) {
	start := netNow()
	logged := false
	for {
		ctx, cancel := context.WithTimeout(context.Background(), networkDialTimeout)
		err := dialServer(ctx, addr)
		cancel()
		waited = netNow().Round(0).Sub(start.Round(0))
		if err == nil {
			if logged {
				writeDebugLog(fmt.Sprintf("[Network] %s reachable after %ds", name, int(waited.Round(time.Second)/time.Second)))
			}
			return true, waited, nil
		}
		if !logged {
			writeDebugLog(fmt.Sprintf("[Network] waiting for %s %s (%v)", name, why, err))
			logged = true
		}
		if waited >= networkWaitMax {
			writeDebugLog(fmt.Sprintf("[Network] %s still not reachable after %s: the run is not started (%v)", name, waited.Round(time.Second), err))
			return false, waited, err
		}
		netSleep(networkRetryEvery)
	}
}
