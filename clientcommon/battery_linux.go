//go:build linux

package clientcommon

func platformReadBattery() BatteryStatus {
	return readSysfsBattery("/sys/class/power_supply")
}
