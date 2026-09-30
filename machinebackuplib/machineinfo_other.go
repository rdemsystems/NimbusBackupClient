//go:build !windows && !linux

package machinebackuplib

func collectPlatformInfo(mi *MachineInfo) {}

func systemDiskIndex(devices []string, disks []BackupDisk) int { return -1 }
