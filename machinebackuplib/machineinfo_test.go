package machinebackuplib

import (
	"encoding/base64"
	"strings"
	"testing"
)

// smbiosType1 builds a type 1 structure (SMBIOS 2.8 layout, 0x1B bytes)
// followed by its string-set.
func smbiosType1(uuid [16]byte, strs ...string) []byte {
	f := make([]byte, 0x1B)
	f[0], f[1] = 1, 0x1B
	f[4], f[5], f[6], f[7] = 1, 2, 3, 4 // manufacturer, product, version, serial
	copy(f[0x08:0x18], uuid[:])
	f[0x19], f[0x1A] = 5, 6 // SKU, family
	for _, s := range strs {
		f = append(append(f, s...), 0)
	}
	return append(f, 0)
}

func TestParseSMBIOSSystemInfo(t *testing.T) {
	// Type 0 (BIOS) with no strings first: two NULs end its string-set.
	table := append([]byte{0, 4, 0, 0, 0, 0}, smbiosType1(
		[16]byte{0x44, 0x45, 0x4c, 0x4c, 0x35, 0x00, 0x10, 0x4e, 0x80, 0x33, 0xb4, 0xc0, 0x4f, 0x4d, 0x32, 0x33},
		"Dell Inc.", "PowerEdge R640", "To Be Filled By O.E.M.", "ABC1234", "SKU=0716", "PowerEdge")...)
	table = append(table, 127, 4, 0, 0, 0, 0)

	info := parseSMBIOSSystemInfo(table, 2, 8)
	want := SMBIOSInfo{
		UUID:         "4C4C4544-0035-4E10-8033-B4C04F4D3233", // Dell-style, first fields little endian
		Manufacturer: "Dell Inc.",
		Product:      "PowerEdge R640",
		Version:      "", // placeholder dropped
		Serial:       "ABC1234",
		SKU:          "SKU=0716",
		Family:       "PowerEdge",
	}
	if info != want {
		t.Fatalf("got %+v\nwant %+v", info, want)
	}

	// Pre-2.6 tables store the UUID big endian.
	old := parseSMBIOSSystemInfo(smbiosType1([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}), 2, 4)
	if old.UUID != "01020304-0506-0708-090A-0B0C0D0E0F10" {
		t.Errorf("SMBIOS 2.4 UUID = %s", old.UUID)
	}
	var ff [16]byte
	for i := range ff {
		ff[i] = 0xFF
	}
	if u := parseSMBIOSSystemInfo(smbiosType1(ff), 3, 0).UUID; u != "" {
		t.Errorf("unset UUID = %q, want empty", u)
	}
	if got := parseSMBIOSSystemInfo([]byte{1, 0x1B, 0}, 3, 0); got != (SMBIOSInfo{}) {
		t.Errorf("truncated table: %+v", got)
	}
}

func TestPVEOSType(t *testing.T) {
	for _, tc := range []struct {
		os                  string
		major, minor, build uint32
		server              bool
		want                string
	}{
		{"windows", 10, 0, 22631, false, "win11"},
		{"windows", 10, 0, 19045, false, "win10"},
		{"windows", 10, 0, 26100, true, "win11"}, // Server 2025
		{"windows", 10, 0, 20348, true, "win11"}, // Server 2022
		{"windows", 10, 0, 17763, true, "win10"}, // Server 2019
		{"windows", 6, 3, 9600, true, "win8"},    // 2012 R2
		{"windows", 6, 1, 7601, false, "win7"},
		{"linux", 0, 0, 0, false, "l26"},
		{"darwin", 0, 0, 0, false, "other"},
	} {
		mi := &MachineInfo{OS: tc.os, OSMajor: tc.major, OSMinor: tc.minor, OSBuild: tc.build, IsServer: tc.server}
		if got := pveOSType(mi); got != tc.want {
			t.Errorf("%s %d.%d.%d server=%v: %s, want %s", tc.os, tc.major, tc.minor, tc.build, tc.server, got, tc.want)
		}
	}
}

func TestBuildQemuConfig(t *testing.T) {
	mi := &MachineInfo{
		Hostname: "SRV_FILES 01", OS: "windows", OSName: "Windows Server 2022 Standard (build 20348)",
		PVEOSType: "win11", CPUs: 8, MemoryMiB: 32768, Firmware: "uefi", SecureBoot: true, TPM: true,
		SMBIOS:   SMBIOSInfo{UUID: "4C4C4544-0035-4E10-8033-B4C04F4D3233", Manufacturer: "Dell Inc.", Serial: "ABC1234"},
		NICs:     []NICInfo{{Name: "Ethernet", MAC: "B4:96:91:00:11:22", Up: true}, {Name: "Ethernet 2", MAC: "B4:96:91:00:11:23"}},
		BootDisk: 1, CollectedAt: "2026-09-30T12:00:00+02:00",
	}
	conf := BuildQemuConfig(mi, 105, []BackupDisk{{Index: 0, Size: 1 << 40}, {Index: 1, Size: 256 << 30}})
	for _, want := range []string{
		"bios: ovmf\n",
		"boot: order=sata1\n",
		"cores: 8\n",
		"memory: 32768\n",
		"name: SRV-FILES-01\n",
		"net0: e1000e=B4:96:91:00:11:22,bridge=vmbr0\n",
		"net1: e1000e=B4:96:91:00:11:23,bridge=vmbr0\n",
		"ostype: win11\n",
		"sata0: local:105/vm-105-disk-0.raw,discard=on,size=1099511627776\n",
		"sata1: local:105/vm-105-disk-1.raw,discard=on,size=274877906944\n",
		"smbios1: uuid=4c4c4544-0035-4e10-8033-b4c04f4d3233,manufacturer=" + base64.StdEncoding.EncodeToString([]byte("Dell Inc.")) +
			",serial=" + base64.StdEncoding.EncodeToString([]byte("ABC1234")) + ",base64=1\n",
		"EFI Disk with pre-enrolled keys",
		"TPM State v2.0",
	} {
		if !strings.Contains(conf, want) {
			t.Errorf("config lacks %q:\n%s", want, conf)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(conf), "\n") {
		if strings.HasPrefix(line, "#") {
			if strings.Contains(line, "%") {
				t.Errorf("description line contains %%: %q", line)
			}
			continue
		}
		if !strings.Contains(line, ": ") {
			t.Errorf("malformed config line %q", line)
		}
	}
	injected := BuildQemuConfig(&MachineInfo{Hostname: "h", OS: "linux", PVEOSType: "l26", BootDisk: -1,
		SMBIOS: SMBIOSInfo{Manufacturer: "Evil\nbios: seabios"}}, 100, nil)
	if strings.Contains(injected, "\nbios: seabios") {
		t.Errorf("newline in SMBIOS data escaped the description:\n%s", injected)
	}
	if strings.Contains(conf, "efidisk0") || strings.Contains(conf, "tpmstate0") {
		t.Error("config references volumes absent from the backup")
	}

	// Unknown hardware: safe defaults, generated SMBIOS UUID, PVE-picked MAC.
	bare := BuildQemuConfig(&MachineInfo{Hostname: "", OS: "linux", PVEOSType: "l26", BootDisk: -1}, 100, []BackupDisk{{Index: 0, Size: 1 << 30}})
	for _, want := range []string{"boot: order=sata0\n", "cores: 2\n", "memory: 2048\n", "name: restored-machine\n", "net0: virtio,bridge=vmbr0\n", "smbios1: uuid="} {
		if !strings.Contains(bare, want) {
			t.Errorf("bare config lacks %q:\n%s", want, bare)
		}
	}
	if strings.Contains(bare, "bios: ovmf") || strings.Contains(bare, "base64=1") {
		t.Errorf("bare config:\n%s", bare)
	}
}

func TestSanitizeVMNameAndUUID(t *testing.T) {
	for in, want := range map[string]string{"WIN-DC01": "WIN-DC01", "srv_files": "srv-files", "-é-": "restored-machine", strings.Repeat("a", 70): strings.Repeat("a", 63)} {
		if got := sanitizeVMName(in); got != want {
			t.Errorf("sanitizeVMName(%q) = %q, want %q", in, got, want)
		}
	}
	if normalizeUUID("00000000-0000-0000-0000-000000000000") != "" || normalizeUUID("ffffffff-ffff-ffff-ffff-ffffffffffff") != "" {
		t.Error("unset UUID not dropped")
	}
	if got := normalizeUUID("4c4c4544-0035-4e10-8033-b4c04f4d3233"); got != "4C4C4544-0035-4E10-8033-B4C04F4D3233" {
		t.Errorf("normalizeUUID = %s", got)
	}
}
