//go:build windows

package snapshot

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Opt-in (PBS_VSS_VOL, run as SYSTEM as the service does): one shadow copy
// made the way the backups make them, then the Application log is read for
// VSS event 8194 ("Unexpected error querying for the IVssWriterCallback
// interface ... Access is denied", issue 160) logged during it. With COM
// security set (vss_comsecurity_windows.go) the writers, the System Writer
// among them, can call back, and there are none.
//
// PBS_VSS_EXPECT_8194=1 turns the check round, to show a build without the fix
// does log them (the repro).
func TestLiveVSSWritersCanCallBack(t *testing.T) {
	vol := os.Getenv("PBS_VSS_VOL")
	if vol == "" {
		t.Skip("set PBS_VSS_VOL (and run as SYSTEM)")
	}
	start := time.Now()
	time.Sleep(1100 * time.Millisecond) // the event log keeps whole seconds
	if err := CreateVSSSnapshot([]string{vol}, false, func(map[string]SnapShot) error { return nil }); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second) // let VSS write its events
	t.Logf("CoInitializeSecurity answered 0x%08X", uint32(comSecurityResult))

	n := vssEventsSince(t, 8194, start)
	t.Logf("VSS event 8194 during the snapshot: %d", n)
	if os.Getenv("PBS_VSS_EXPECT_8194") == "1" {
		if n == 0 {
			t.Errorf("expected VSS event 8194 without the fix, saw none")
		}
		return
	}
	if comSecurityResult != 0 {
		t.Errorf("CoInitializeSecurity answered 0x%08X, want S_OK in a process that used COM for nothing else", uint32(comSecurityResult))
	}
	if n != 0 {
		t.Errorf("VSS logged event 8194 %d times during the snapshot: a writer could not call back", n)
	}
}

// vssEventsSince counts the VSS events with that ID in the Application log
// since start.
func vssEventsSince(t *testing.T, id int, start time.Time) int {
	t.Helper()
	ms := time.Since(start).Milliseconds() + 1000
	q := fmt.Sprintf("*[System[Provider[@Name='VSS'] and EventID=%d and TimeCreated[timediff(@SystemTime) <= %d]]]", id, ms)
	out, err := exec.Command("wevtutil", "qe", "Application", "/q:"+q, "/f:text", "/c:500").CombinedOutput()
	if err != nil {
		t.Fatalf("wevtutil: %v: %s", err, out)
	}
	return countEventIDs(string(out), id)
}

// countEventIDs counts the "Event ID: <id>" lines of wevtutil's text output.
func countEventIDs(out string, id int) int {
	return strings.Count(out, "Event ID: "+strconv.Itoa(id))
}
