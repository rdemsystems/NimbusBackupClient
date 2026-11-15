package clientcommon

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"pbscommon"
)

// What the computer does once a backup has run, as -then (or "then" in the
// config file) says. Leave is the default: the computer stays as it is, and
// its own sleep settings apply.
const (
	ThenLeave     = "leave"
	ThenSleep     = "sleep"
	ThenHibernate = "hibernate"
	ThenShutdown  = "shutdown"
	// ThenAfterMax is the longest -then-after, in minutes.
	ThenAfterMax = 240
)

// AfterRun is what the backup tools do once a backup has run: -then and
// -then-after, or "then" and "then-after" in the config file, the app's "When
// the backup finishes, the computer". -shutdown-after ("shutdown-after") is
// the older name for -then shutdown and still works.
type AfterRun struct {
	ShutdownAfter bool   `json:"shutdown-after,omitempty"`
	Then          string `json:"then,omitempty"`
	ThenAfter     int    `json:"then-after,omitempty"` // minutes, 0 at once
}

// Action is what the computer does: "" (it stays as it is), ThenSleep,
// ThenHibernate or ThenShutdown.
func (a AfterRun) Action() string {
	switch t := strings.ToLower(strings.TrimSpace(a.Then)); t {
	case "", ThenLeave:
		if a.ShutdownAfter {
			return ThenShutdown
		}
		return ""
	default:
		return t
	}
}

// Validate refuses an unknown -then, a wait out of range, and -shutdown-after
// with a -then that says something else.
func (a AfterRun) Validate() error {
	switch t := strings.ToLower(strings.TrimSpace(a.Then)); t {
	case "", ThenLeave, ThenSleep, ThenHibernate, ThenShutdown:
		if a.ShutdownAfter && t != "" && t != ThenShutdown {
			return fmt.Errorf("-shutdown-after is -then shutdown, but -then is %s: give one or the other", t)
		}
	default:
		return fmt.Errorf("-then must be leave, sleep, hibernate or shutdown, not %q", a.Then)
	}
	if a.ThenAfter < 0 || a.ThenAfter > ThenAfterMax {
		return fmt.Errorf("-then-after must be a number of minutes from 0 to %d, not %d", ThenAfterMax, a.ThenAfter)
	}
	return nil
}

// AfterRunFlags defines -then, -then-after and -shutdown-after on fs. The
// function it returns, called after parsing, copies the flags given over a, so
// they override the config file and, left out, keep the file's values; it
// returns what is wrong with the result. -then given replaces the file's
// "shutdown-after"; -shutdown-after given alone replaces the file's "then"
// (when it turns shutting down on) or drops a "then" of shutdown (when off).
func AfterRunFlags(fs *flag.FlagSet) func(a *AfterRun) error {
	then := fs.String("then", "", "What the computer does once the backup has run, whether it worked or not, after the e-mail if one is sent: leave (default: it stays as it is, and its own sleep settings apply), sleep, hibernate or shutdown (Windows: through the Windows API, or shutdown.exe if that fails; Linux: systemctl suspend or hibernate, shutdown -h now). Not when the run stops before it starts (wrong options, another backup running). Needs the right to do it: an administrator, SYSTEM or root to shut down")
	after := fs.Int("then-after", 0, fmt.Sprintf("Minutes to wait before -then, 0 to %d (default 0: at once). Ctrl+C during the wait leaves the computer as it is", ThenAfterMax))
	shutdown := fs.Bool("shutdown-after", false, "The same as -then shutdown")
	return func(a *AfterRun) error {
		given := map[string]bool{}
		fs.Visit(func(f *flag.Flag) { given[f.Name] = true })
		switch {
		case given["then"]:
			a.Then = *then
			a.ShutdownAfter = given["shutdown-after"] && *shutdown
		case given["shutdown-after"]:
			a.ShutdownAfter = *shutdown
			if *shutdown || a.Action() == ThenShutdown {
				a.Then = ""
			}
		}
		if given["then-after"] {
			a.ThenAfter = *after
		}
		return a.Validate()
	}
}

// The ways ActAfterRun acts on the computer, and its wait; tests replace them.
var (
	shutdownComputer  = pbscommon.ShutdownComputer
	sleepComputer     = pbscommon.SleepComputer
	hibernateComputer = pbscommon.HibernateComputer
	waitBeforeThen    = time.Sleep
)

// ActAfterRun does what a asks once the backup has run and its e-mail has
// gone: waits -then-after minutes, then puts the computer to sleep, hibernates
// it or turns it off. Call it just before the tool exits. It says what it
// does. A failure is reported on stderr and does not change the exit code: the
// backup's result is what the exit code reports.
func ActAfterRun(a AfterRun) {
	action := a.Action()
	if action == "" {
		return
	}
	what := map[string]string{ThenSleep: "Putting the computer to sleep", ThenHibernate: "Hibernating the computer", ThenShutdown: "Turning the computer off"}[action]
	if what == "" {
		fmt.Fprintf(os.Stderr, "Leaving the computer as it is: -then %q is not known.\n", a.Then)
		return
	}
	if a.ThenAfter > 0 {
		fmt.Printf("%s in %d minute(s), as -then asks (Ctrl+C leaves it as it is).\n", what, a.ThenAfter)
		waitBeforeThen(time.Duration(a.ThenAfter) * time.Minute)
	}
	fmt.Printf("%s, as -then asks.\n", what)
	var err error
	switch action {
	case ThenSleep:
		err = sleepComputer()
	case ThenHibernate:
		err = hibernateComputer()
	case ThenShutdown:
		var method string
		method, err = shutdownComputer("Shutting down after a backup, as -then shutdown asks.")
		if err == nil {
			fmt.Println("Shutdown requested through " + method + ".")
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot do -then %s: %v\n", action, err)
	}
}
