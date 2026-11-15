package pbscommon

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Putting the computer to sleep or into hibernation once a backup has run (a
// Backup Set's "When the backup finishes, the computer", the command line
// tools' -then), beside ShutdownComputer. SleepComputer, HibernateComputer and
// CanHibernate are in suspend_windows.go, suspend_linux.go and
// suspend_other.go.

// Why a computer cannot hibernate, as CanHibernate says it (HibernateCheck.Why).
const (
	// HibernateOff: the computer has sleep states, but hibernation is turned
	// off (Windows: powercfg /hibernate off).
	HibernateOff = "off"
	// HibernateNone: the computer has no hibernate state at all (a VM, or
	// firmware that does not offer one).
	HibernateNone = "none"
	// HibernateKernel: Linux has a hibernate state but the kernel will not use
	// it (sys/power/disk reads "[disabled]", as under Secure Boot's lockdown).
	HibernateKernel = "kernel"
	// HibernateSwap: Linux has no swap space, or less than the memory in use,
	// so the memory has nowhere to be saved.
	HibernateSwap = "swap"
	// HibernateNoResume: Linux has no resume device (resume= on the kernel
	// command line), so the saved memory would not be read back at boot.
	HibernateNoResume = "noresume"
)

// HibernateCheck is what CanHibernate found. Why is "" when OK, else one of
// the Hibernate* codes. For HibernateSwap, SwapBytes is the swap space there
// is and NeededBytes the memory in use that it must hold.
type HibernateCheck struct {
	OK          bool
	Why         string
	SwapBytes   uint64
	NeededBytes uint64
}

// SwapGB says the swap space and the memory to be saved in whole GB (GiB) for
// the messages: needGB rounded up, haveGB rounded down but at least 1 when
// there is any swap, 0 when there is none.
func (c HibernateCheck) SwapGB() (needGB, haveGB uint64) {
	const gib = 1 << 30
	needGB = (c.NeededBytes + gib - 1) / gib
	if needGB < 1 {
		needGB = 1
	}
	if c.SwapBytes > 0 {
		haveGB = c.SwapBytes / gib
		if haveGB < 1 {
			haveGB = 1
		}
	}
	return needGB, haveGB
}

// Reason says in English why the computer cannot hibernate, for the TUI and
// the logs ("" when it can): "no or not enough swap space (needs about 6 GB,
// this computer has none)".
func (c HibernateCheck) Reason() string {
	if c.OK {
		return ""
	}
	switch c.Why {
	case HibernateOff:
		return "hibernation is turned off on this computer"
	case HibernateKernel:
		return "hibernation is turned off in the kernel"
	case HibernateSwap:
		need, have := c.SwapGB()
		has := "none"
		if have > 0 {
			has = fmt.Sprintf("%d GB", have)
		}
		return fmt.Sprintf("no or not enough swap space (needs about %d GB, this computer has %s)", need, has)
	case HibernateNoResume:
		return "no resume device is set (resume= on the kernel command line)"
	}
	return "this computer cannot hibernate"
}

// ReadLinuxHibernate says whether Linux can hibernate, from the files under
// root (the real system's are under "/"):
//   - sys/power/state lists "disk" when the kernel can hibernate at all
//     (else HibernateNone);
//   - sys/power/disk reads "[disabled]" when the kernel may not, as under its
//     lockdown (HibernateKernel);
//   - proc/swaps must list active swap at least as large as the memory in use,
//     MemTotal less MemAvailable from proc/meminfo (HibernateSwap);
//   - sys/power/resume must name a device, not "0:0", for the memory to be
//     read back at boot (HibernateNoResume). When the file is missing this is
//     not checked.
func ReadLinuxHibernate(root string) HibernateCheck {
	read := func(parts ...string) (string, bool) {
		b, err := os.ReadFile(filepath.Join(append([]string{root}, parts...)...)) // #nosec G304 -- a fixed file under root
		return string(b), err == nil
	}
	state, ok := read("sys", "power", "state")
	if !ok {
		return HibernateCheck{Why: HibernateNone}
	}
	disk, _ := read("sys", "power", "disk")
	swaps, _ := read("proc", "swaps")
	meminfo, _ := read("proc", "meminfo")
	resume, hasResume := read("sys", "power", "resume")
	if !hasResume {
		resume = ""
	}
	return ParseLinuxHibernate(state, disk, swaps, meminfo, resume, hasResume)
}

// ParseLinuxHibernate is ReadLinuxHibernate on the files' text. hasResume is
// false when sys/power/resume could not be read: the resume device is then
// not checked.
func ParseLinuxHibernate(state, disk, swaps, meminfo, resume string, hasResume bool) HibernateCheck {
	found := false
	for _, s := range strings.Fields(state) {
		if s == "disk" {
			found = true
		}
	}
	if !found {
		return HibernateCheck{Why: HibernateNone}
	}
	if strings.Contains(disk, "[disabled]") {
		return HibernateCheck{Why: HibernateKernel}
	}
	swap := parseSwapsTotal(swaps)
	total, available := parseMeminfo(meminfo)
	var inUse uint64
	if total > available {
		inUse = total - available
	}
	if swap == 0 || swap < inUse {
		return HibernateCheck{Why: HibernateSwap, SwapBytes: swap, NeededBytes: inUse}
	}
	if hasResume {
		r := strings.TrimSpace(resume)
		if r == "" || r == "0:0" {
			return HibernateCheck{Why: HibernateNoResume}
		}
	}
	return HibernateCheck{OK: true}
}

// ParseSystemdVersion reads the version from the first line of systemctl
// --version ("systemd 255 (255.4-1ubuntu8)"); 0 when it is not there.
func ParseSystemdVersion(out string) int {
	line, _, _ := strings.Cut(out, "\n")
	f := strings.Fields(line)
	if len(f) < 2 || f[0] != "systemd" {
		return 0
	}
	n, err := strconv.Atoi(f[1])
	if err != nil {
		return 0
	}
	return n
}

// parseSwapsTotal adds up the Size column (KiB) of /proc/swaps, in bytes.
func parseSwapsTotal(swaps string) uint64 {
	var total uint64
	sc := bufio.NewScanner(strings.NewReader(swaps))
	first := true
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if first {
			first = false
			if len(f) > 0 && f[0] == "Filename" {
				continue
			}
		}
		if len(f) < 3 {
			continue
		}
		if kib, err := strconv.ParseUint(f[2], 10, 64); err == nil {
			total += kib * 1024
		}
	}
	return total
}

// parseMeminfo reads MemTotal and MemAvailable (kB) from /proc/meminfo, in
// bytes. Without MemAvailable (kernels before 3.14) it falls back to MemFree
// plus Buffers and Cached.
func parseMeminfo(meminfo string) (total, available uint64) {
	vals := map[string]uint64{}
	sc := bufio.NewScanner(strings.NewReader(meminfo))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 2 {
			continue
		}
		if kb, err := strconv.ParseUint(f[1], 10, 64); err == nil {
			vals[strings.TrimSuffix(f[0], ":")] = kb * 1024
		}
	}
	total = vals["MemTotal"]
	if v, ok := vals["MemAvailable"]; ok {
		available = v
	} else {
		available = vals["MemFree"] + vals["Buffers"] + vals["Cached"]
	}
	return total, available
}
