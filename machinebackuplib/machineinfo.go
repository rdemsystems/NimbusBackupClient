package machinebackuplib

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MachineInfo describes the backed-up machine so a "vm" snapshot restores in
// Proxmox VE as a VM that matches it: CPU count, RAM, firmware (BIOS/UEFI),
// NICs, SMBIOS identity and guest OS type. It drives the generated
// qemu-server.conf and is also uploaded as machine-info.json.blob (PVE
// ignores that file; it keeps the full facts for later P2V tooling).
type MachineInfo struct {
	Hostname   string     `json:"hostname"`
	OS         string     `json:"os"`                // runtime.GOOS
	OSName     string     `json:"os_name,omitempty"` // e.g. "Windows Server 2022 Standard (build 20348)"
	OSMajor    uint32     `json:"os_major,omitempty"`
	OSMinor    uint32     `json:"os_minor,omitempty"`
	OSBuild    uint32     `json:"os_build,omitempty"`
	IsServer   bool       `json:"is_server,omitempty"`
	PVEOSType  string     `json:"pve_ostype"`
	CPUs       int        `json:"cpus"` // logical processors
	MemoryMiB  uint64     `json:"memory_mib"`
	Firmware   string     `json:"firmware"` // "uefi", "bios" or "" (unknown)
	SecureBoot bool       `json:"secure_boot,omitempty"`
	TPM        bool       `json:"tpm,omitempty"`
	SMBIOS     SMBIOSInfo `json:"smbios"`
	NICs       []NICInfo  `json:"nics"`
	// BootDisk is the BackupDisk.Index (sataN) of the disk holding the OS,
	// -1 when it could not be determined (the first disk is then used).
	BootDisk    int      `json:"boot_disk"`
	CollectedAt string   `json:"collected_at"`
	Warnings    []string `json:"warnings,omitempty"`
}

// SMBIOSInfo is the SMBIOS "System Information" (type 1) of the machine.
type SMBIOSInfo struct {
	UUID         string `json:"uuid,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
	Product      string `json:"product,omitempty"`
	Version      string `json:"version,omitempty"`
	Serial       string `json:"serial,omitempty"`
	SKU          string `json:"sku,omitempty"`
	Family       string `json:"family,omitempty"`
}

// NICInfo is one physical network adapter.
type NICInfo struct {
	Name string `json:"name"`
	MAC  string `json:"mac"`
	Up   bool   `json:"up"`
	// LocallyAdministered MACs are set by software (virtual adapters, but
	// also overridden or randomized MACs on real ones).
	LocallyAdministered bool `json:"locally_administered,omitempty"`
}

// CollectMachineInfo gathers the machine facts for the backed-up disks.
// devices are cfg.BackupDevices (used to locate the boot disk on Linux).
// Every probe is best-effort: a failed one leaves its field empty and the
// generated VM config falls back to safe defaults.
func CollectMachineInfo(devices []string, disks []BackupDisk) *MachineInfo {
	mi := &MachineInfo{
		OS:          runtime.GOOS,
		CPUs:        runtime.NumCPU(),
		BootDisk:    -1,
		CollectedAt: time.Now().Format(time.RFC3339),
	}
	mi.Hostname, _ = os.Hostname()
	collectPlatformInfo(mi)
	mi.NICs = collectNICs()
	mi.PVEOSType = pveOSType(mi)
	if idx := systemDiskIndex(devices, disks); idx >= 0 {
		mi.BootDisk = idx
	} else if len(disks) > 0 {
		mi.Warnings = append(mi.Warnings, "could not determine the system disk; boot order set to the first disk")
	}
	for _, d := range disks {
		if d.Index > 5 {
			mi.Warnings = append(mi.Warnings, fmt.Sprintf("disk sata%d: Proxmox VE only supports sata0-sata5, move it to another bus after restore", d.Index))
		}
	}
	return mi
}

// pveOSType maps the guest OS to the Proxmox VE "ostype" value.
func pveOSType(mi *MachineInfo) string {
	switch mi.OS {
	case "linux":
		return "l26"
	case "windows":
	default:
		return "other"
	}
	switch {
	case mi.OSMajor >= 10:
		// Windows 11 / Server 2022 / Server 2025 → win11;
		// Windows 10 / Server 2016 / Server 2019 → win10.
		if (!mi.IsServer && mi.OSBuild >= 22000) || (mi.IsServer && mi.OSBuild >= 20348) {
			return "win11"
		}
		return "win10"
	case mi.OSMajor == 6 && mi.OSMinor >= 2: // 8, 8.1, 2012, 2012 R2
		return "win8"
	case mi.OSMajor == 6 && mi.OSMinor == 1: // 7, 2008 R2
		return "win7"
	case mi.OSMajor == 6: // Vista, 2008
		return "w2k8"
	case mi.OSMajor == 5 && mi.OSMinor >= 2: // 2003
		return "w2k3"
	case mi.OSMajor == 5:
		return "wxp"
	default:
		return "win10" // unknown Windows: the most compatible modern choice
	}
}

// virtualNICMarkers identifies virtual/tunnel adapters that must not become
// VM NICs (matched case-insensitively against the interface name).
var virtualNICMarkers = []string{
	"loopback", "vethernet", "hyper-v", "virtualbox", "vmware", "vmnet", "tap", "tun",
	"wireguard", "wg", "tailscale", "zerotier", "openvpn", "bluetooth", "docker",
	"veth", "virbr", "vmbr", "br-", "wsl", "npcap", "isatap", "teredo", "6to4",
}

// collectNICs lists physical adapters: a 6-byte MAC and no virtual-adapter
// name. Locally administered MACs are kept (real adapters can have one) but
// sorted after factory MACs, so the 4-NIC cap drops them first.
func collectNICs() []NICInfo {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	nics := make([]NICInfo, 0)
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagLoopback != 0 || len(ifc.HardwareAddr) != 6 {
			continue
		}
		lname := strings.ToLower(ifc.Name)
		virtual := false
		for _, m := range virtualNICMarkers {
			if strings.Contains(lname, m) {
				virtual = true
				break
			}
		}
		if virtual {
			continue
		}
		nics = append(nics, NICInfo{
			Name:                ifc.Name,
			MAC:                 strings.ToUpper(ifc.HardwareAddr.String()),
			Up:                  ifc.Flags&net.FlagUp != 0,
			LocallyAdministered: ifc.HardwareAddr[0]&0x02 != 0,
		})
	}
	// Factory MACs first, then connected adapters, then by name, for a
	// stable net0..netN order.
	sort.SliceStable(nics, func(i, j int) bool {
		if nics[i].LocallyAdministered != nics[j].LocallyAdministered {
			return !nics[i].LocallyAdministered
		}
		if nics[i].Up != nics[j].Up {
			return nics[i].Up
		}
		return nics[i].Name < nics[j].Name
	})
	return nics
}

// parseSMBIOSSystemInfo extracts the type 1 (System Information) structure
// from a raw SMBIOS structure table. major/minor is the SMBIOS version: from
// 2.6 on, the first three UUID fields are little endian.
func parseSMBIOSSystemInfo(table []byte, major, minor byte) SMBIOSInfo {
	var info SMBIOSInfo
	pos := 0
	for pos+4 <= len(table) {
		typ := table[pos]
		length := int(table[pos+1])
		if length < 4 || pos+length > len(table) {
			break
		}
		formatted := table[pos : pos+length]
		// Unformatted string-set: NUL-terminated strings, ended by an extra NUL.
		strs := []string{}
		end := pos + length
		for end < len(table) {
			start := end
			for end < len(table) && table[end] != 0 {
				end++
			}
			if end == start { // empty string: end of the string-set
				end++
				break
			}
			strs = append(strs, string(table[start:end]))
			end++
		}
		if len(strs) == 0 && end < len(table) && table[end] == 0 {
			end++ // no strings: the set is two NULs
		}
		str := func(off int) string {
			if off >= len(formatted) {
				return ""
			}
			n := int(formatted[off])
			if n == 0 || n > len(strs) {
				return ""
			}
			return cleanSMBIOSString(strs[n-1])
		}
		if typ == 1 {
			info.Manufacturer = str(0x04)
			info.Product = str(0x05)
			info.Version = str(0x06)
			info.Serial = str(0x07)
			if len(formatted) >= 0x19 {
				info.UUID = formatSMBIOSUUID(formatted[0x08:0x18], major > 2 || (major == 2 && minor >= 6))
			}
			info.SKU = str(0x19)
			info.Family = str(0x1A)
			return info
		}
		if typ == 127 { // end-of-table
			break
		}
		pos = end
	}
	return info
}

// formatSMBIOSUUID renders a 16-byte SMBIOS UUID, or "" when the firmware
// left it unset (all 0x00 or all 0xFF).
func formatSMBIOSUUID(b []byte, littleEndian bool) string {
	if len(b) != 16 {
		return ""
	}
	allZero, allFF := true, true
	for _, v := range b {
		allZero = allZero && v == 0
		allFF = allFF && v == 0xFF
	}
	if allZero || allFF {
		return ""
	}
	var u [16]byte
	copy(u[:], b)
	if littleEndian {
		binary.BigEndian.PutUint32(u[0:4], binary.LittleEndian.Uint32(b[0:4]))
		binary.BigEndian.PutUint16(u[4:6], binary.LittleEndian.Uint16(b[4:6]))
		binary.BigEndian.PutUint16(u[6:8], binary.LittleEndian.Uint16(b[6:8]))
	}
	return strings.ToUpper(fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16]))
}

// smbiosPlaceholders are values firmware vendors leave in unset SMBIOS fields.
var smbiosPlaceholders = []string{
	"to be filled by o.e.m.", "default string", "not specified", "not applicable",
	"system product name", "system manufacturer", "system version", "system serial number",
	"none", "n/a", "0", "0123456789",
}

// cleanSMBIOSString trims a SMBIOS string and drops vendor placeholders.
func cleanSMBIOSString(s string) string {
	s = strings.TrimSpace(strings.Trim(s, "\x00"))
	for _, p := range smbiosPlaceholders {
		if strings.EqualFold(s, p) {
			return ""
		}
	}
	return s
}

// normalizeUUID upper-cases a textual UUID and drops unset ones (all 0 / all F).
func normalizeUUID(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	hexOnly := strings.ReplaceAll(s, "-", "")
	if len(hexOnly) != 32 || strings.Trim(hexOnly, "0") == "" || strings.Trim(hexOnly, "F") == "" {
		return ""
	}
	return s
}

// sanitizeVMName makes a Proxmox VE VM name (a DNS name: letters, digits,
// '-' and '.', not starting or ending with '-' or '.').
func sanitizeVMName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	s := strings.Trim(b.String(), "-.")
	if len(s) > 63 {
		s = strings.Trim(s[:63], "-.")
	}
	if s == "" {
		s = "restored-machine"
	}
	return s
}

// BuildQemuConfig renders the qemu-server.conf stored in a "vm" snapshot, so
// PVE restores a VM sized and identified like the source machine. Devices
// that need a storage volume PVE cannot take from the backup (EFI vars disk,
// TPM state) are not declared: the notes at the top of the config (shown as
// the VM description in PVE) say what to add before the first boot.
func BuildQemuConfig(mi *MachineInfo, vmid int64, disks []BackupDisk) string {
	var notes []string
	notes = append(notes, fmt.Sprintf("Created by Nimbus Backup from machine %s", mi.Hostname))
	if mi.OSName != "" {
		notes = append(notes, "Source OS: "+mi.OSName)
	}
	if mi.SMBIOS.Manufacturer != "" || mi.SMBIOS.Product != "" {
		notes = append(notes, strings.TrimSpace("Source hardware: "+mi.SMBIOS.Manufacturer+" "+mi.SMBIOS.Product))
	}
	notes = append(notes, fmt.Sprintf("Backed up on %s with %d CPUs and %d MiB RAM: adjust to the target host if needed.", mi.CollectedAt, mi.CPUs, mi.MemoryMiB))
	if mi.Firmware == "uefi" {
		keys := ""
		if mi.SecureBoot {
			keys = " with pre-enrolled keys (Secure Boot was on)"
		}
		notes = append(notes, "UEFI boot: add an EFI Disk"+keys+" before the first start (Hardware > Add > EFI Disk). Without it PVE boots with temporary EFI variables.")
	}
	if mi.TPM {
		notes = append(notes, "The source had a TPM: add a TPM State v2.0 (Hardware > Add > TPM State). BitLocker volumes may ask for their recovery key on first boot.")
	}
	if len(mi.NICs) > 0 {
		notes = append(notes, "Network cards keep the source MAC addresses: tick Unique when restoring if the source machine is still online.")
	}
	for _, w := range mi.Warnings {
		notes = append(notes, "Warning: "+w)
	}

	var b strings.Builder
	for _, n := range notes {
		// One line per note (a CR/LF from SMBIOS or a hostname would otherwise
		// start an uncommented config line); PVE URI-decodes description
		// lines, so keep '%' out of them.
		n = strings.NewReplacer("\r", " ", "\n", " ", "%", " percent").Replace(n)
		b.WriteString("#" + n + "\n")
	}

	boot := -1
	if mi.BootDisk >= 0 {
		for _, d := range disks {
			if d.Index == mi.BootDisk {
				boot = d.Index
			}
		}
	}
	if boot < 0 && len(disks) > 0 {
		boot = disks[0].Index
	}

	if mi.Firmware == "uefi" {
		b.WriteString("bios: ovmf\n")
	}
	if boot >= 0 {
		fmt.Fprintf(&b, "boot: order=sata%d\n", boot)
	}
	cpus := mi.CPUs
	if cpus < 1 {
		cpus = 2
	}
	fmt.Fprintf(&b, "cores: %d\n", cpus)
	b.WriteString("machine: q35\n")
	mem := mi.MemoryMiB
	if mem < 512 {
		mem = 2048
	}
	fmt.Fprintf(&b, "memory: %d\n", mem)
	fmt.Fprintf(&b, "name: %s\n", sanitizeVMName(mi.Hostname))

	// Windows ships an e1000e driver in-box (virtio needs extra drivers);
	// older Windows only has e1000. Linux has virtio in-kernel.
	model := "virtio"
	if mi.OS == "windows" {
		model = "e1000e"
		if mi.PVEOSType == "win7" || mi.PVEOSType == "w2k8" || mi.PVEOSType == "w2k3" || mi.PVEOSType == "wxp" {
			model = "e1000"
		}
	}
	nics := mi.NICs
	if len(nics) > 4 {
		nics = nics[:4]
	}
	if len(nics) == 0 {
		fmt.Fprintf(&b, "net0: %s,bridge=vmbr0\n", model) // PVE picks a MAC
	}
	for i, n := range nics {
		fmt.Fprintf(&b, "net%d: %s=%s,bridge=vmbr0\n", i, model, n.MAC)
	}
	b.WriteString("numa: 0\n")
	b.WriteString("onboot: 0\n")
	fmt.Fprintf(&b, "ostype: %s\n", mi.PVEOSType)
	for _, d := range disks {
		fmt.Fprintf(&b, "sata%d: local:%d/vm-%d-disk-%d.raw,discard=on,size=%d\n", d.Index, vmid, vmid, d.Index, d.Size)
	}
	b.WriteString("scsihw: virtio-scsi-single\n")
	b.WriteString("smbios1: " + smbios1Value(mi.SMBIOS) + "\n")
	b.WriteString("sockets: 1\n")
	fmt.Fprintf(&b, "vmgenid: %s\n", uuid.New().String())
	return b.String()
}

// smbios1Value builds the PVE smbios1 option: the source UUID (a fresh one
// if unknown) and, base64-encoded as PVE requires for free text, the
// manufacturer/product/serial/... strings.
func smbios1Value(s SMBIOSInfo) string {
	id := s.UUID
	if id == "" {
		id = uuid.New().String()
	}
	parts := []string{"uuid=" + strings.ToLower(id)}
	enc := func(k, v string) {
		if v != "" {
			parts = append(parts, k+"="+base64.StdEncoding.EncodeToString([]byte(v)))
		}
	}
	enc("manufacturer", s.Manufacturer)
	enc("product", s.Product)
	enc("version", s.Version)
	enc("serial", s.Serial)
	enc("sku", s.SKU)
	enc("family", s.Family)
	if len(parts) > 1 {
		parts = append(parts, "base64=1")
	}
	return strings.Join(parts, ",")
}
