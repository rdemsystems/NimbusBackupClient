package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"clientcommon"
	"snapshot"
)

// -on-battery and -cancel-below, for backup-folder and backup-disk: the
// command line's form of a Backup Set's "On battery" rule. They are read
// here, before the backup's own options, so a run that may not start on
// battery stops at once (exit code 4, before any password prompt or
// connection), and a run that may is watched while it goes on. They are also
// defined on the backup's flag set, so its parsing accepts them and its
// -help lists them.

const (
	exitBatteryHeld      = 4 // not started: on battery, and -on-battery does not let it start
	exitBatteryCancelled = 5 // cancelled: the charge dropped below -cancel-below
)

// batteryCheckEvery is how often a running backup's -cancel-below is checked.
var batteryCheckEvery = 30 * time.Second

// batteryFlagNames are the two options, as given without their dash.
var batteryFlagNames = map[string]bool{"on-battery": true, "cancel-below": true}

// batteryArgs reads -on-battery and -cancel-below from args as the flag
// package would (-x v, -x=v, --x v), up to "--", and whether args ask for
// -help or -version (no backup runs then). Other options' values are not
// known here, so every argument up to "--" is looked at. It does not change
// args.
func batteryArgs(args []string) (onBattery, cancelBelow string, info bool) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			continue
		}
		name := strings.TrimLeft(a, "-")
		value, hasValue := "", false
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			name, value, hasValue = name[:eq], name[eq+1:], true
		}
		switch name {
		case "help", "h", "version":
			info = true
			continue
		}
		if !batteryFlagNames[name] {
			continue
		}
		if !hasValue && i+1 < len(args) {
			i++
			value = args[i]
		}
		if name == "on-battery" {
			onBattery = value
		} else {
			cancelBelow = value
		}
	}
	return onBattery, cancelBelow, info
}

// batteryRuleFromArgs is the rule args give, checked.
func batteryRuleFromArgs(args []string) (rule clientcommon.BatteryRule, info bool, err error) {
	on, below, info := batteryArgs(args)
	mode, start, err := clientcommon.ParseOnBattery(on)
	if err != nil {
		return rule, info, err
	}
	rule = clientcommon.BatteryRule{Mode: mode, StartPercent: start}
	if below != "" {
		n, cerr := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(below), "%"))
		if cerr != nil || n < 0 || n > 99 {
			return rule, info, fmt.Errorf("-cancel-below must be a whole number from 0 to 99 (0 never cancels), not %q", below)
		}
		rule.CancelBelow = n
	}
	return rule, info, rule.Validate()
}

// defineBatteryFlags puts the two options on fs (the backup's flag set), for
// its parsing and its -help. Their values there are not read: batteryArgs
// has read them.
func defineBatteryFlags(fs *flag.FlagSet) {
	fs.String("on-battery", "run", "On a laptop running on its battery: run (start anyway), above:N (start only if the charge is at least N%, 5 to 95) or plugged (do not start). A run that does not start prints why and ends with exit code 4. On mains, or with no battery, the backup always runs")
	fs.Int("cancel-below", 0, "Cancel the backup if, on battery, the charge drops below this percentage (checked every 30 seconds; 0 never cancels; lower than N of above:N). Nothing from a cancelled run is saved; it ends with exit code 5")
}

// batteryHoldMessage says why a backup does not start, or "" when it may.
func batteryHoldMessage(rule clientcommon.BatteryRule, b clientcommon.BatteryStatus) string {
	hold, min := rule.Holds(b)
	if !hold {
		return ""
	}
	if min > 0 {
		return fmt.Sprintf("Not started: the computer is on battery at %d%%, below the required %d%% (-on-battery above:%d). Run it again when the computer is plugged in.", b.Percent, min, min)
	}
	return "Not started: the computer is on battery, and -on-battery plugged waits for the computer to be plugged in. Run it again when it is."
}

// startBatteryRule stops here, with exit code 4, a backup the battery does
// not let start, and watches -cancel-below while one runs.
func startBatteryRule(rule clientcommon.BatteryRule) {
	if msg := batteryHoldMessage(rule, clientcommon.ReadBattery()); msg != "" {
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(exitBatteryHeld)
	}
	if rule.CancelBelow <= 0 {
		return
	}
	go func() {
		t := time.NewTicker(batteryCheckEvery)
		defer t.Stop()
		for range t.C {
			b := clientcommon.ReadBattery()
			if !rule.Cancels(b) {
				continue
			}
			fmt.Fprintf(os.Stderr, "\nCancelled: on battery, the charge dropped below -cancel-below %d%% (now %d%%). Nothing from this run was saved; the next run starts it again in full.\n", rule.CancelBelow, b.Percent)
			snapshot.ExitWithCleanup(exitBatteryCancelled)
		}
	}()
}
