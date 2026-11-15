package clientcommon

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSupply makes one /sys/class/power_supply/<name> folder under root.
func writeSupply(t *testing.T, root, name string, files map[string]string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for k, v := range files {
		if err := os.WriteFile(filepath.Join(dir, k), []byte(v+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReadSysfsBattery(t *testing.T) {
	t.Run("no supplies at all: no battery", func(t *testing.T) {
		if got := readSysfsBattery(filepath.Join(t.TempDir(), "missing")); got != (BatteryStatus{}) {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("desktop with mains only", func(t *testing.T) {
		root := t.TempDir()
		writeSupply(t, root, "AC", map[string]string{"type": "Mains", "online": "1"})
		if got := readSysfsBattery(root); got.HasBattery {
			t.Fatalf("a computer with no battery read as having one: %+v", got)
		}
	})
	t.Run("laptop unplugged", func(t *testing.T) {
		root := t.TempDir()
		writeSupply(t, root, "AC", map[string]string{"type": "Mains", "online": "0"})
		writeSupply(t, root, "BAT0", map[string]string{"type": "Battery", "capacity": "22", "status": "Discharging"})
		want := BatteryStatus{HasBattery: true, OnBattery: true, Percent: 22, Known: true}
		if got := readSysfsBattery(root); got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})
	t.Run("laptop on a USB-C charger", func(t *testing.T) {
		root := t.TempDir()
		writeSupply(t, root, "AC", map[string]string{"type": "Mains", "online": "0"})
		writeSupply(t, root, "ucsi-source-psy-1", map[string]string{"type": "USB", "online": "1"})
		writeSupply(t, root, "BAT0", map[string]string{"type": "Battery", "capacity": "60", "status": "Charging"})
		if got := readSysfsBattery(root); !got.HasBattery || got.OnBattery || got.Percent != 60 {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("no adapter listed: the battery's status says", func(t *testing.T) {
		root := t.TempDir()
		writeSupply(t, root, "BAT1", map[string]string{"type": "Battery", "capacity": "80", "status": "Discharging"})
		if got := readSysfsBattery(root); !got.OnBattery {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("two batteries average, a mouse battery is not the computer's", func(t *testing.T) {
		root := t.TempDir()
		writeSupply(t, root, "AC", map[string]string{"type": "Mains", "online": "0"})
		writeSupply(t, root, "BAT0", map[string]string{"type": "Battery", "capacity": "40"})
		writeSupply(t, root, "BAT1", map[string]string{"type": "Battery", "capacity": "20"})
		writeSupply(t, root, "hidpp_battery_0", map[string]string{"type": "Battery", "scope": "Device", "capacity": "5"})
		if got := readSysfsBattery(root); got.Percent != 30 || !got.Known {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("only a mouse battery: no battery", func(t *testing.T) {
		root := t.TempDir()
		writeSupply(t, root, "hidpp_battery_0", map[string]string{"type": "Battery", "scope": "Device", "capacity": "5"})
		if got := readSysfsBattery(root); got.HasBattery {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("capacity unreadable", func(t *testing.T) {
		root := t.TempDir()
		writeSupply(t, root, "AC", map[string]string{"type": "Mains", "online": "0"})
		writeSupply(t, root, "BAT0", map[string]string{"type": "Battery"})
		if got := readSysfsBattery(root); !got.OnBattery || got.Known {
			t.Fatalf("got %+v", got)
		}
	})
}

func TestBatteryRule(t *testing.T) {
	low := BatteryStatus{HasBattery: true, OnBattery: true, Percent: 22, Known: true}
	high := BatteryStatus{HasBattery: true, OnBattery: true, Percent: 80, Known: true}
	mains := BatteryStatus{HasBattery: true, OnBattery: false, Percent: 5, Known: true}
	desktop := BatteryStatus{}
	unknown := BatteryStatus{HasBattery: true, OnBattery: true}

	above := BatteryRule{Mode: BatteryAbove, StartPercent: 30}
	if hold, min := above.Holds(low); !hold || min != 30 {
		t.Errorf("above 30 at 22%%: hold %v min %d", hold, min)
	}
	if hold, _ := above.Holds(high); hold {
		t.Error("above 30 at 80% held")
	}
	if hold, _ := above.Holds(mains); hold {
		t.Error("held on mains")
	}
	if hold, _ := above.Holds(desktop); hold {
		t.Error("held with no battery")
	}
	if hold, _ := above.Holds(unknown); hold {
		t.Error("held with an unknown charge")
	}
	if hold, min := (BatteryRule{Mode: BatteryAbove}).Holds(low); !hold || min != DefaultBatteryStartPercent {
		t.Errorf("above with no level: hold %v min %d", hold, min)
	}
	plugged := BatteryRule{Mode: BatteryPlugged}
	if hold, min := plugged.Holds(high); !hold || min != 0 {
		t.Errorf("plugged on battery: hold %v min %d", hold, min)
	}
	if hold, _ := plugged.Holds(unknown); !hold {
		t.Error("plugged with an unknown charge not held")
	}
	if hold, _ := (BatteryRule{}).Holds(low); hold {
		t.Error("the zero rule (start anyway) held")
	}

	cancel := BatteryRule{CancelBelow: 10}
	if !cancel.Cancels(BatteryStatus{HasBattery: true, OnBattery: true, Percent: 9, Known: true}) {
		t.Error("9% under a 10% level not cancelled")
	}
	if cancel.Cancels(BatteryStatus{HasBattery: true, OnBattery: true, Percent: 10, Known: true}) {
		t.Error("10% cancelled under a 10% level")
	}
	if cancel.Cancels(BatteryStatus{HasBattery: true, OnBattery: false, Percent: 3, Known: true}) {
		t.Error("cancelled on mains")
	}
	if (BatteryRule{}).Cancels(BatteryStatus{HasBattery: true, OnBattery: true, Percent: 1, Known: true}) {
		t.Error("cancelled with no cancel level")
	}
}

func TestBatteryRuleValidate(t *testing.T) {
	for _, r := range []BatteryRule{{}, {Mode: "anyway"}, {Mode: BatteryPlugged, CancelBelow: 50}, {Mode: BatteryAbove, StartPercent: 30, CancelBelow: 10}, {Mode: BatteryAbove}} {
		if err := r.Validate(); err != nil {
			t.Errorf("%+v: %v", r, err)
		}
	}
	for _, r := range []BatteryRule{{Mode: "sometimes"}, {Mode: BatteryAbove, StartPercent: 4}, {Mode: BatteryAbove, StartPercent: 96},
		{CancelBelow: 100}, {CancelBelow: -1}, {Mode: BatteryAbove, StartPercent: 30, CancelBelow: 30}} {
		if err := r.Validate(); err == nil {
			t.Errorf("%+v accepted", r)
		}
	}
}

func TestParseOnBattery(t *testing.T) {
	cases := []struct {
		in    string
		mode  string
		start int
		bad   bool
	}{
		{"run", BatteryRunAnyway, 0, false},
		{"", BatteryRunAnyway, 0, false},
		{"plugged", BatteryPlugged, 0, false},
		{"above:40", BatteryAbove, 40, false},
		{"above:40%", BatteryAbove, 40, false},
		{"above", BatteryAbove, DefaultBatteryStartPercent, false},
		{"above:2", "", 0, true},
		{"above:x", "", 0, true},
		{"never", "", 0, true},
	}
	for _, c := range cases {
		mode, start, err := ParseOnBattery(c.in)
		if (err != nil) != c.bad || mode != c.mode || start != c.start {
			t.Errorf("%q: got %q %d %v", c.in, mode, start, err)
		}
	}
}
