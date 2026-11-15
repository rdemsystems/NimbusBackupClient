//go:build windows

package clientcommon

import "testing"

func TestBatteryFromPowerStatus(t *testing.T) {
	cases := []struct {
		name string
		in   systemPowerStatus
		want BatteryStatus
	}{
		{"no system battery", systemPowerStatus{ACLineStatus: 1, BatteryFlag: 128, BatteryLifePercent: 255}, BatteryStatus{}},
		{"battery state unknown", systemPowerStatus{ACLineStatus: 255, BatteryFlag: 255, BatteryLifePercent: 255}, BatteryStatus{}},
		{"on battery", systemPowerStatus{ACLineStatus: 0, BatteryFlag: 2, BatteryLifePercent: 22}, BatteryStatus{HasBattery: true, OnBattery: true, Percent: 22, Known: true}},
		{"charging", systemPowerStatus{ACLineStatus: 1, BatteryFlag: 8, BatteryLifePercent: 70}, BatteryStatus{HasBattery: true, Percent: 70, Known: true}},
		{"charge unknown", systemPowerStatus{ACLineStatus: 0, BatteryFlag: 1, BatteryLifePercent: 255}, BatteryStatus{HasBattery: true, OnBattery: true}},
	}
	for _, c := range cases {
		if got := batteryFromPowerStatus(c.in); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}
