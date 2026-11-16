//go:build windows

package snapshot

import (
	"testing"

	"golang.org/x/sys/windows"
)

// The descriptor handed to CoInitializeSecurity is valid, absolute, and lets
// in exactly the accounts named: COM refuses an invalid one (E_INVALIDARG), and
// the VSS writers' callbacks are then turned away (VSS event 8194, issue 160).
func TestCOMCallersDescriptor(t *testing.T) {
	sd, err := comCallersDescriptor()
	if err != nil {
		t.Fatal(err)
	}
	if !sd.IsValid() {
		t.Fatal("not valid")
	}
	ctl, _, err := sd.Control()
	if err != nil {
		t.Fatal(err)
	}
	if ctl&windows.SE_SELF_RELATIVE != 0 {
		t.Error("self-relative; CoInitializeSecurity takes the absolute form")
	}
	if owner, _, err := sd.Owner(); err != nil || owner == nil || !owner.IsWellKnown(windows.WinLocalSystemSid) {
		t.Errorf("owner %v, %v; want SYSTEM", owner, err)
	}
	if group, _, err := sd.Group(); err != nil || group == nil {
		t.Errorf("group %v, %v", group, err)
	}
	if s := sd.String(); s != comCallersSDDL {
		t.Errorf("descriptor %s, want %s", s, comCallersSDDL)
	}
}
