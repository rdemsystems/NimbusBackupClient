//go:build !windows && !linux

package clientcommon

// Other systems: no battery is read, so nothing is held back or cancelled.
func platformReadBattery() BatteryStatus { return BatteryStatus{} }
