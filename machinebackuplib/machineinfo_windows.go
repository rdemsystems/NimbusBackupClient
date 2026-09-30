//go:build windows

package machinebackuplib

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	procGlobalMemoryStatusEx   = modkernel32.NewProc("GlobalMemoryStatusEx")
	procGetFirmwareType        = modkernel32.NewProc("GetFirmwareType")
	procGetSystemFirmwareTable = modkernel32.NewProc("GetSystemFirmwareTable")
	modtbs                     = windows.NewLazySystemDLL("tbs.dll")
	procTbsiGetDeviceInfo      = modtbs.NewProc("Tbsi_GetDeviceInfo")
)

// MEMORYSTATUSEX
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

// TPM_DEVICE_INFO
type tpmDeviceInfo struct {
	StructVersion    uint32
	TPMVersion       uint32
	TPMInterfaceType uint32
	TPMImpRevision   uint32
}

func collectPlatformInfo(mi *MachineInfo) {
	// RtlGetVersion is not subject to the manifest-based version lie of GetVersionEx.
	v := windows.RtlGetVersion()
	mi.OSMajor, mi.OSMinor, mi.OSBuild = v.MajorVersion, v.MinorVersion, v.BuildNumber
	mi.IsServer = v.ProductType != 1 // VER_NT_WORKSTATION
	mi.OSName = windowsProductName(mi)

	var ms memoryStatusEx
	ms.Length = uint32(unsafe.Sizeof(ms))
	if r, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms))); r != 0 {
		mi.MemoryMiB = ms.TotalPhys / (1024 * 1024)
	}

	// GetFirmwareType (Windows 8+): FirmwareTypeBios = 1, FirmwareTypeUefi = 2.
	if procGetFirmwareType.Find() == nil {
		var ft uint32
		if r, _, _ := procGetFirmwareType.Call(uintptr(unsafe.Pointer(&ft))); r != 0 {
			switch ft {
			case 1:
				mi.Firmware = "bios"
			case 2:
				mi.Firmware = "uefi"
			}
		}
	}
	if mi.Firmware == "uefi" {
		if k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\SecureBoot\State`, registry.QUERY_VALUE); err == nil {
			if val, _, err := k.GetIntegerValue("UEFISecureBootEnabled"); err == nil && val == 1 {
				mi.SecureBoot = true
			}
			_ = k.Close()
		}
	}

	// Tbsi_GetDeviceInfo returns TBS_SUCCESS (0) only when a TPM is present.
	if procTbsiGetDeviceInfo.Find() == nil {
		var info tpmDeviceInfo
		if r, _, _ := procTbsiGetDeviceInfo.Call(uintptr(unsafe.Sizeof(info)), uintptr(unsafe.Pointer(&info))); r == 0 {
			mi.TPM = true
		}
	}

	mi.SMBIOS = windowsSMBIOS()
}

// windowsProductName is e.g. "Windows Server 2022 Standard (build 20348)".
// Windows 11 still reports "Windows 10" in ProductName: fix it from the build.
func windowsProductName(mi *MachineInfo) string {
	name := "Windows"
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE); err == nil {
		if s, _, err := k.GetStringValue("ProductName"); err == nil && s != "" {
			name = s
		}
		_ = k.Close()
	}
	if !mi.IsServer && mi.OSBuild >= 22000 {
		name = strings.Replace(name, "Windows 10", "Windows 11", 1)
	}
	return fmt.Sprintf("%s (build %d)", name, mi.OSBuild)
}

// windowsSMBIOS reads the raw SMBIOS table (GetSystemFirmwareTable 'RSMB').
func windowsSMBIOS() SMBIOSInfo {
	const rsmb = 0x52534D42 // 'RSMB'
	size, _, _ := procGetSystemFirmwareTable.Call(rsmb, 0, 0, 0)
	if size < 8 || size > 1<<20 {
		return SMBIOSInfo{}
	}
	buf := make([]byte, size)
	n, _, _ := procGetSystemFirmwareTable.Call(rsmb, 0, uintptr(unsafe.Pointer(&buf[0])), size)
	if n < 8 || n > size {
		return SMBIOSInfo{}
	}
	buf = buf[:n]
	// RawSMBIOSData: Used20CallingMethod, SMBIOSMajorVersion, SMBIOSMinorVersion,
	// DmiRevision (1 byte each), Length (DWORD), then the structure table.
	table := buf[8:]
	if tlen := binary.LittleEndian.Uint32(buf[4:8]); int(tlen) < len(table) {
		table = table[:tlen]
	}
	return parseSMBIOSSystemInfo(table, buf[1], buf[2])
}

// systemDiskIndex returns the backed-up disk (PhysicalDriveN index) holding
// the Windows system drive, or -1.
func systemDiskIndex(devices []string, disks []BackupDisk) int {
	sysDrive := strings.ToUpper(os.Getenv("SystemDrive"))
	if sysDrive == "" {
		sysDrive = "C:"
	}
	vols, err := enumVolumeDiskOffset()
	if err != nil {
		return -1
	}
	for _, v := range vols {
		for _, l := range v.Letters {
			if !strings.EqualFold(strings.TrimRight(l, `\`), sysDrive) {
				continue
			}
			for _, d := range disks {
				if d.Index == int(v.DiskNumber) {
					return d.Index
				}
			}
		}
	}
	return -1
}
