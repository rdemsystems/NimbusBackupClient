package machinebackuplib

import (
	"strings"
	"testing"
)

func TestBuildQemuConfigUsesVMIDForEveryDisk(t *testing.T) {
	mi := &MachineInfo{Hostname: "testhost", OS: "linux", PVEOSType: "l26", BootDisk: -1}
	cfg := BuildQemuConfig(mi, 107, []BackupDisk{{Index: 0, Size: 1 << 30}, {Index: 1, Size: 2 << 30}})
	for _, want := range []string{
		"name: testhost",
		"sata0: local:107/vm-107-disk-0.raw,cache=writeback,discard=on,size=1073741824",
		"sata1: local:107/vm-107-disk-1.raw,cache=writeback,discard=on,size=2147483648",
		"#qmdump#map:sata0:drive-sata0::raw:",
		"#qmdump#map:sata1:drive-sata1::raw:",
		"vmgenid: ",
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("config missing %q\n%s", want, cfg)
		}
	}
}

func TestBuildQemuConfigUEFI(t *testing.T) {
	mi := &MachineInfo{Hostname: "h", OS: "windows", PVEOSType: "win11", BootDisk: -1}
	disks := []BackupDisk{{Index: 0, Size: 1 << 30}}
	if out := BuildQemuConfig(mi, 107, disks); strings.Contains(out, "bios:") {
		t.Errorf("BIOS disk must not get a bios line\n%s", out)
	}
	// Firmware detection failed but the boot disk is GPT: still OVMF.
	disks[0].GPT = true
	if out := BuildQemuConfig(mi, 107, disks); !strings.Contains(out, "\nbios: ovmf\n") || strings.Contains(out, "efidisk") {
		t.Errorf("GPT boot disk should get bios: ovmf and no efidisk\n%s", out)
	}
	// OVMF follows the boot disk, not the lowest-index one.
	mi.BootDisk = 1
	two := []BackupDisk{{Index: 0, Size: 1 << 30}, {Index: 1, Size: 1 << 30, GPT: true}}
	if out := BuildQemuConfig(mi, 107, two); !strings.Contains(out, "boot: order=sata1\n") || !strings.Contains(out, "\nbios: ovmf\n") {
		t.Errorf("GPT boot disk sata1 should get bios: ovmf\n%s", out)
	}
	mi.BootDisk = -1
	disks[0].GPT = false
	mi.Firmware = "uefi"
	if out := BuildQemuConfig(mi, 107, disks); !strings.Contains(out, "\nbios: ovmf\n") {
		t.Errorf("UEFI firmware should get bios: ovmf\n%s", out)
	}
}

func TestGPTDetection(t *testing.T) {
	mk := func(off int) []byte {
		b := make([]byte, 8192)
		copy(b[off:], "EFI PART")
		return b
	}
	if !gptSignatureIn(mk(512)) || !gptSignatureIn(mk(4096)) {
		t.Error("GPT header not found")
	}
	if gptSignatureIn(make([]byte, 8192)) {
		t.Error("MBR/blank disk reported as GPT")
	}
	if !diskIsGPT([]BackupDisk{{Index: 1}, {Index: 0, GPT: true}}, 0) || diskIsGPT([]BackupDisk{{Index: 0}, {Index: 1, GPT: true}}, 0) {
		t.Error("diskIsGPT must look at the requested disk")
	}
}
