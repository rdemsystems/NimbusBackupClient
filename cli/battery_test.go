package main

import (
	"flag"
	"strings"
	"testing"

	"clientcommon"
)

func TestBatteryArgs(t *testing.T) {
	cases := []struct {
		args              []string
		onBattery, cancel string
		info              bool
	}{
		{[]string{"-backupdir", "/data"}, "", "", false},
		{[]string{"-on-battery", "above:40", "-cancel-below", "10", "-backupdir", "/data"}, "above:40", "10", false},
		{[]string{"--on-battery=plugged", "-cancel-below=5"}, "plugged", "5", false},
		{[]string{"-backupdir", "/data", "-help"}, "", "", true},
		{[]string{"-version"}, "", "", true},
		// After "--" nothing is an option.
		{[]string{"--", "-on-battery", "plugged"}, "", "", false},
	}
	for _, c := range cases {
		on, below, info := batteryArgs(c.args)
		if on != c.onBattery || below != c.cancel || info != c.info {
			t.Errorf("%v: got %q %q %v", c.args, on, below, info)
		}
	}
}

func TestBatteryRuleFromArgs(t *testing.T) {
	rule, _, err := batteryRuleFromArgs([]string{"-on-battery", "above:40", "-cancel-below", "10"})
	if err != nil || rule.Mode != clientcommon.BatteryAbove || rule.StartPercent != 40 || rule.CancelBelow != 10 {
		t.Fatalf("got %+v, %v", rule, err)
	}
	if rule, _, err := batteryRuleFromArgs(nil); err != nil || rule != (clientcommon.BatteryRule{}) {
		t.Fatalf("no options: %+v, %v (want start anyway, never cancel)", rule, err)
	}
	for _, bad := range [][]string{
		{"-on-battery", "sometimes"},
		{"-cancel-below", "x"},
		{"-cancel-below", "100"},
		{"-on-battery", "above:30", "-cancel-below", "30"},
	} {
		if _, _, err := batteryRuleFromArgs(bad); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
}

func TestBatteryHoldMessage(t *testing.T) {
	low := clientcommon.BatteryStatus{HasBattery: true, OnBattery: true, Percent: 22, Known: true}
	if msg := batteryHoldMessage(clientcommon.BatteryRule{Mode: clientcommon.BatteryAbove, StartPercent: 30}, low); !strings.Contains(msg, "22%") || !strings.Contains(msg, "30%") {
		t.Errorf("message %q", msg)
	}
	if msg := batteryHoldMessage(clientcommon.BatteryRule{Mode: clientcommon.BatteryPlugged}, low); msg == "" {
		t.Error("plugged on battery was not held")
	}
	if msg := batteryHoldMessage(clientcommon.BatteryRule{}, low); msg != "" {
		t.Errorf("run anyway held: %q", msg)
	}
	mains := clientcommon.BatteryStatus{HasBattery: true, Percent: 3, Known: true}
	if msg := batteryHoldMessage(clientcommon.BatteryRule{Mode: clientcommon.BatteryPlugged}, mains); msg != "" {
		t.Errorf("held on mains: %q", msg)
	}
}

// The backup's own parsing accepts the two options once they are defined.
func TestDefineBatteryFlags(t *testing.T) {
	fs := flag.NewFlagSet("backup-folder", flag.ContinueOnError)
	defineBatteryFlags(fs)
	if err := fs.Parse([]string{"-on-battery", "above:40", "-cancel-below", "10"}); err != nil {
		t.Fatal(err)
	}
}
