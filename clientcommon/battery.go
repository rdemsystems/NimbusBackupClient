package clientcommon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// A Backup Set's "On battery" rule, and the command line's -on-battery and
// -cancel-below: whether a backup starts when a laptop runs on its battery,
// and when a running one is cancelled because the charge has dropped. On mains
// power, and on a computer with no battery (a desktop, a VM), a backup always
// starts and is never cancelled for the battery.

// BatteryStatus is the computer's battery as the system reports it.
type BatteryStatus struct {
	// HasBattery: the computer has a battery of its own (a laptop). False on
	// a desktop or a VM, and when the system cannot say.
	HasBattery bool
	// OnBattery: it runs on that battery, with no mains power.
	OnBattery bool
	// Percent is the charge, 0 to 100, when Known.
	Percent int
	Known   bool
}

// ReadBattery reads the battery now (Windows: GetSystemPowerStatus; Linux:
// /sys/class/power_supply; other systems: no battery).
func ReadBattery() BatteryStatus { return platformReadBattery() }

// The start choices of a battery rule.
const (
	BatteryRunAnyway = ""        // start whatever the charge (also "anyway" and "run")
	BatteryAbove     = "above"   // start only at StartPercent or more
	BatteryPlugged   = "plugged" // start only on mains power
)

// DefaultBatteryStartPercent is the start level a rule with BatteryAbove and
// no level of its own uses (and what a new Backup Set starts with).
const DefaultBatteryStartPercent = 30

// BatteryRule is when a backup starts on battery and when it is cancelled.
type BatteryRule struct {
	Mode         string // BatteryRunAnyway, BatteryAbove or BatteryPlugged
	StartPercent int    // for BatteryAbove, 5 to 95
	CancelBelow  int    // cancel a running backup below this charge; 0 never
}

// Normal is the rule with its start choice spelled one way ("anyway" and
// "run" are BatteryRunAnyway) and a level for BatteryAbove.
func (r BatteryRule) Normal() BatteryRule {
	switch strings.ToLower(strings.TrimSpace(r.Mode)) {
	case "", "anyway", "run":
		r.Mode = BatteryRunAnyway
	case BatteryAbove:
		r.Mode = BatteryAbove
		if r.StartPercent == 0 {
			r.StartPercent = DefaultBatteryStartPercent
		}
	case BatteryPlugged:
		r.Mode = BatteryPlugged
	}
	if r.CancelBelow < 0 {
		r.CancelBelow = 0
	}
	return r
}

// Validate says what is wrong with the rule, or nil.
func (r BatteryRule) Validate() error {
	switch strings.ToLower(strings.TrimSpace(r.Mode)) {
	case "", "anyway", "run", BatteryPlugged:
	case BatteryAbove:
		if r.StartPercent != 0 && (r.StartPercent < 5 || r.StartPercent > 95) {
			return fmt.Errorf("the battery start level must be 5 to 95%%, not %d%%", r.StartPercent)
		}
	default:
		return fmt.Errorf("unknown battery choice %q (use anyway, above or plugged)", r.Mode)
	}
	if r.CancelBelow < 0 || r.CancelBelow > 99 {
		return fmt.Errorf("the battery cancel level must be 0 to 99%%, not %d%%", r.CancelBelow)
	}
	n := r.Normal()
	if n.Mode == BatteryAbove && n.CancelBelow > 0 && n.CancelBelow >= n.StartPercent {
		return fmt.Errorf("the battery cancel level (%d%%) must be lower than the start level (%d%%)", n.CancelBelow, n.StartPercent)
	}
	return nil
}

// Holds reports whether a backup must wait for mains power with the battery
// as b, and the charge it needs (0 when it waits for mains whatever the
// charge). A charge that cannot be read holds a backup only when the rule
// waits for mains: below a level cannot be told, so it starts.
func (r BatteryRule) Holds(b BatteryStatus) (hold bool, min int) {
	if !b.HasBattery || !b.OnBattery {
		return false, 0
	}
	n := r.Normal()
	switch n.Mode {
	case BatteryPlugged:
		return true, 0
	case BatteryAbove:
		if b.Known && b.Percent < n.StartPercent {
			return true, n.StartPercent
		}
	}
	return false, 0
}

// Cancels reports whether a running backup is to be cancelled with the
// battery as b: on battery, with a known charge below CancelBelow.
func (r BatteryRule) Cancels(b BatteryStatus) bool {
	return r.CancelBelow > 0 && b.HasBattery && b.OnBattery && b.Known && b.Percent < r.CancelBelow
}

// ParseOnBattery reads -on-battery: run, above:N or plugged.
func ParseOnBattery(s string) (mode string, start int, err error) {
	v := strings.ToLower(strings.TrimSpace(s))
	switch {
	case v == "" || v == "run" || v == "anyway":
		return BatteryRunAnyway, 0, nil
	case v == BatteryPlugged:
		return BatteryPlugged, 0, nil
	case v == BatteryAbove:
		return BatteryAbove, DefaultBatteryStartPercent, nil
	case strings.HasPrefix(v, BatteryAbove+":"):
		n, cerr := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(v, BatteryAbove+":"), "%"))
		if cerr != nil || n < 5 || n > 95 {
			return "", 0, fmt.Errorf("-on-battery above:N needs N from 5 to 95, not %q", s)
		}
		return BatteryAbove, n, nil
	}
	return "", 0, fmt.Errorf("-on-battery must be run, above:N or plugged, not %q", s)
}

// readSysfsBattery reads the power supplies under root (on Linux
// /sys/class/power_supply): a "Battery" of the system (not a mouse's, whose
// scope is "Device") with its capacity and status, and the "Mains" and "USB"
// adapters with online. The computer is on battery when it has adapters and
// none is online, or, with no adapter listed, when a battery is discharging.
// With several batteries the charge is their average.
func readSysfsBattery(root string) BatteryStatus {
	entries, err := os.ReadDir(root)
	if err != nil {
		return BatteryStatus{}
	}
	read := func(dir, name string) string {
		b, err := os.ReadFile(filepath.Join(root, dir, name))
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
	var (
		st                       BatteryStatus
		adapters, online         int
		discharging              bool
		percentSum, percentCount int
	)
	for _, e := range entries {
		name := e.Name()
		switch read(name, "type") {
		case "Battery":
			if strings.EqualFold(read(name, "scope"), "Device") || read(name, "present") == "0" {
				continue
			}
			st.HasBattery = true
			if n, err := strconv.Atoi(read(name, "capacity")); err == nil && n >= 0 && n <= 100 {
				percentSum += n
				percentCount++
			}
			if strings.EqualFold(read(name, "status"), "Discharging") {
				discharging = true
			}
		case "Mains", "USB", "USB_C", "USB_PD":
			adapters++
			if read(name, "online") == "1" {
				online++
			}
		}
	}
	if !st.HasBattery {
		return BatteryStatus{}
	}
	if adapters > 0 {
		st.OnBattery = online == 0
	} else {
		st.OnBattery = discharging
	}
	if percentCount > 0 {
		st.Percent, st.Known = percentSum/percentCount, true
	}
	return st
}
