//go:build linux

package machinebackuplib

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func collectPlatformInfo(mi *MachineInfo) {
	mi.OSName = linuxOSName()
	mi.MemoryMiB = linuxMemTotalMiB()
	if _, err := os.Stat("/sys/firmware/efi"); err == nil {
		mi.Firmware = "uefi"
		// efivarfs: 4 attribute bytes, then the value (1 = Secure Boot on).
		if b, err := os.ReadFile("/sys/firmware/efi/efivars/SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c"); err == nil && len(b) >= 5 && b[4] == 1 {
			mi.SecureBoot = true
		}
	} else {
		mi.Firmware = "bios"
	}
	if _, err := os.Stat("/sys/class/tpm/tpm0"); err == nil {
		mi.TPM = true
	}
	read := func(name string) string {
		b, err := os.ReadFile("/sys/class/dmi/id/" + name)
		if err != nil {
			return ""
		}
		return cleanSMBIOSString(string(b))
	}
	mi.SMBIOS = SMBIOSInfo{
		UUID:         normalizeUUID(read("product_uuid")), // root only
		Manufacturer: read("sys_vendor"),
		Product:      read("product_name"),
		Version:      read("product_version"),
		Serial:       read("product_serial"),
		SKU:          read("product_sku"),
		Family:       read("product_family"),
	}
}

func linuxOSName() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), "PRETTY_NAME="); ok {
			return strings.Trim(v, `"'`)
		}
	}
	return "Linux"
}

func linuxMemTotalMiB() uint64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kb, err := strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				return kb / 1024
			}
		}
	}
	return 0
}

// systemDiskIndex returns the index (in devices) of the backed-up disk that
// holds the root filesystem, or -1.
func systemDiskIndex(devices []string, disks []BackupDisk) int {
	var st unix.Stat_t
	if err := unix.Stat("/", &st); err != nil {
		return -1
	}
	disk := blockDiskForDev(unix.Major(uint64(st.Dev)), unix.Minor(uint64(st.Dev)))
	if disk == "" {
		return -1
	}
	for i, dev := range devices {
		p, err := filepath.EvalSymlinks(dev)
		if err != nil {
			p = dev
		}
		if filepath.Base(p) != disk {
			continue
		}
		for _, d := range disks {
			if d.Index == i {
				return i
			}
		}
	}
	return -1
}

// blockDiskForDev resolves a block device number to its whole-disk kernel
// name (sda, nvme0n1), following device-mapper / LVM / md to their first
// underlying device. "" when it cannot (e.g. btrfs anonymous devices).
func blockDiskForDev(maj, min uint32) string {
	sys, err := filepath.EvalSymlinks(fmt.Sprintf("/sys/dev/block/%d:%d", maj, min))
	if err != nil {
		return ""
	}
	for depth := 0; depth < 8; depth++ {
		if slaves, _ := filepath.Glob(filepath.Join(sys, "slaves", "*")); len(slaves) > 0 {
			next, err := filepath.EvalSymlinks(slaves[0])
			if err != nil {
				return ""
			}
			sys = next
			continue
		}
		if _, err := os.Stat(filepath.Join(sys, "partition")); err == nil {
			return filepath.Base(filepath.Dir(sys))
		}
		return filepath.Base(sys)
	}
	return ""
}
