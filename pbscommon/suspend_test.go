package pbscommon

import (
	"os"
	"path/filepath"
	"testing"
)

const testSwapsHead = "Filename\t\t\t\tType\t\tSize\t\tUsed\t\tPriority\n"

const testSwaps8G = testSwapsHead + "/dev/nvme0n1p3                          partition\t8388604\t\t0\t\t-2\n"

const testSwaps1G = testSwapsHead + "/swapfile                               file\t\t1048572\t\t0\t\t-2\n"

const testSwapsTwo = testSwapsHead + "/swapfile                               file\t\t2097148\t\t0\t\t-2\n" +
	"/dev/zram0                              partition\t4194300\t\t0\t\t100\n"

// 16 GiB of memory, 10.5 GiB of it available: 5.5 GiB in use.
const testMeminfo = "MemTotal:       16777216 kB\nMemFree:         2097152 kB\nMemAvailable:   11010048 kB\nBuffers:          524288 kB\nCached:          8388608 kB\n"

// The same before MemAvailable (kernels before 3.14): free, buffers and cache.
const testMeminfoOld = "MemTotal:       16777216 kB\nMemFree:         2097152 kB\nBuffers:          524288 kB\nCached:          8388608 kB\n"

const testState = "freeze mem disk\n"

const testDisk = "[platform] shutdown reboot suspend test_resume\n"

const reasonNoSwap = "no or not enough swap space (needs about 6 GB, this computer has none)"

const reasonSmallSwap = "no or not enough swap space (needs about 6 GB, this computer has 1 GB)"

const reasonNoResume = "no resume device is set (resume= on the kernel command line)"

const reasonNone = "this computer cannot hibernate"

// Linux can hibernate when sys/power/state lists "disk", sys/power/disk is
// not "[disabled]", proc/swaps holds at least the memory in use from
// proc/meminfo, and sys/power/resume names a device; read from a stand-in for
// / ("-" leaves a file out).
func TestReadLinuxHibernate(t *testing.T) {
	// why is "" when it can hibernate.
	for _, c := range []struct{ name, state, disk, swaps, meminfo, resume, why, reason string }{
		{"can", testState, testDisk, testSwaps8G, testMeminfo, "259:3\n", "", ""},
		{"two swap areas add up", testState, testDisk, testSwapsTwo, testMeminfo, "259:3\n", "", ""},
		{"no disk state", "freeze mem\n", testDisk, testSwaps8G, testMeminfo, "259:3\n", HibernateNone, reasonNone},
		{"locked down", testState, "[disabled]\n", testSwaps8G, testMeminfo, "259:3\n", HibernateKernel, "hibernation is turned off in the kernel"},
		{"no disk file", testState, "-", testSwaps8G, testMeminfo, "259:3\n", "", ""},
		{"no sysfs", "-", "-", "-", "-", "-", HibernateNone, reasonNone},
		{"disk only as part of a word", "freeze mem diskette\n", "-", testSwaps8G, testMeminfo, "259:3\n", HibernateNone, reasonNone},
		{"no swap", testState, testDisk, testSwapsHead, testMeminfo, "259:3\n", HibernateSwap, reasonNoSwap},
		{"no swaps file", testState, testDisk, "-", testMeminfo, "259:3\n", HibernateSwap, reasonNoSwap},
		{"swap too small", testState, testDisk, testSwaps1G, testMeminfo, "259:3\n", HibernateSwap, reasonSmallSwap},
		{"swap too small, old meminfo", testState, testDisk, testSwaps1G, testMeminfoOld, "259:3\n", HibernateSwap, reasonSmallSwap},
		{"no resume device", testState, testDisk, testSwaps8G, testMeminfo, "0:0\n", HibernateNoResume, reasonNoResume},
		{"empty resume file", testState, testDisk, testSwaps8G, testMeminfo, "\n", HibernateNoResume, reasonNoResume},
		{"no resume file", testState, testDisk, testSwaps8G, testMeminfo, "-", "", ""},
		{"swap checked before resume", testState, testDisk, testSwapsHead, testMeminfo, "0:0\n", HibernateSwap, reasonNoSwap},
		{"kernel checked before swap", testState, "[disabled]\n", testSwapsHead, testMeminfo, "0:0\n", HibernateKernel, "hibernation is turned off in the kernel"},
	} {
		root := t.TempDir()
		for _, f := range []struct{ dir, name, text string }{
			{filepath.Join(root, "sys", "power"), "state", c.state},
			{filepath.Join(root, "sys", "power"), "disk", c.disk},
			{filepath.Join(root, "sys", "power"), "resume", c.resume},
			{filepath.Join(root, "proc"), "swaps", c.swaps},
			{filepath.Join(root, "proc"), "meminfo", c.meminfo},
		} {
			if err := os.MkdirAll(f.dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if f.text == "-" {
				continue
			}
			if err := os.WriteFile(filepath.Join(f.dir, f.name), []byte(f.text), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		got := ReadLinuxHibernate(root)
		if got.OK != (c.why == "") || got.Why != c.why || got.Reason() != c.reason {
			t.Errorf("%s: %v %q %q, want %q %q", c.name, got.OK, got.Why, got.Reason(), c.why, c.reason)
		}
	}
}

// The swap message rounds what is needed up and what there is down, but
// never says there is 0 GB of swap when there is some.
func TestHibernateSwapGB(t *testing.T) {
	const gib = 1 << 30
	for _, c := range []struct{ swap, need, wantNeed, wantHave uint64 }{
		{0, 5*gib + gib/2, 6, 0},
		{gib / 2, gib / 4, 1, 1},
		{5*gib + gib/2, 5*gib + gib*3/4, 6, 5},
		{0, 0, 1, 0},
	} {
		need, have := HibernateCheck{Why: HibernateSwap, SwapBytes: c.swap, NeededBytes: c.need}.SwapGB()
		if need != c.wantNeed || have != c.wantHave {
			t.Errorf("swap %d need %d: %d %d, want %d %d", c.swap, c.need, need, have, c.wantNeed, c.wantHave)
		}
	}
}

func TestHibernateReasonWindows(t *testing.T) {
	if r := (HibernateCheck{Why: HibernateOff}).Reason(); r != "hibernation is turned off on this computer" {
		t.Errorf("off: %q", r)
	}
	if r := (HibernateCheck{Why: HibernateNone}).Reason(); r != reasonNone {
		t.Errorf("none: %q", r)
	}
	if r := (HibernateCheck{OK: true}).Reason(); r != "" {
		t.Errorf("ok: %q", r)
	}
}

func TestParseSystemdVersion(t *testing.T) {
	for _, c := range []struct {
		in   string
		want int
	}{
		{"systemd 255 (255.4-1ubuntu8)\n+PAM +AUDIT\n", 255},
		{"systemd 249 (249.11-0ubuntu3.12)\n", 249},
		{"", 0},
		{"not systemd\n", 0},
	} {
		if got := ParseSystemdVersion(c.in); got != c.want {
			t.Errorf("%q: %d, want %d", c.in, got, c.want)
		}
	}
}
