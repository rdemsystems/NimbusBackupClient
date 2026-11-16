//go:build windows

package snapshot

import (
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// COM security for the VSS requestor (16/11/2026, issue 160).
//
// While a shadow copy is made, VSS has each writer call back into the
// requestor (IVssWriterCallback). Those calls come from the writer's own
// process and account: the System Writer, for one, runs in Cryptographic
// Services as NETWORK SERVICE. Nothing here called CoInitializeSecurity, so
// COM gave the process its default access rights, which let in SYSTEM,
// administrators and the process's own account, and turned the System Writer
// away: every snapshot left VSS event 8194 ("Unexpected error querying for the
// IVssWriterCallback interface. hr = 0x80070005, Access is denied") in the
// Application log, about 2,400 of them on the Windows test VM since September.
// The snapshot was still made, without that writer's metadata.
//
// Microsoft's guidance for requestors ("Security Considerations for
// Requestors", and its vshadow sample) is to call CoInitializeSecurity once,
// before COM is used for anything else, letting in the accounts writers run
// as. setCOMSecurity does that the first time initCOM starts COM. In the
// service, which uses COM only here, that is first. Where COM is already in
// use (the desktop app: its window is a COM control), Windows refuses the call
// (RPC_E_TOO_LATE) and nothing changes; the log says so once.

const (
	rpcCAuthnLevelPktPrivacy = 6          // RPC_C_AUTHN_LEVEL_PKT_PRIVACY
	rpcCImpLevelImpersonate  = 3          // RPC_C_IMP_LEVEL_IMPERSONATE: WMI (vss_wmi_windows.go) needs it
	eoacDynamicCloaking      = 0x40       // EOAC_DYNAMIC_CLOAKING
	rpcETooLate              = 0x80010119 // RPC_E_TOO_LATE: security was set already, or COM is in use
)

// comCallersSDDL: who may call into this process's COM objects (the VSS
// callback): SYSTEM, administrators, backup operators, and the two service
// accounts writers run as (NETWORK SERVICE, LOCAL SERVICE). CCDC is SDDL's
// name for 0x3, COM_RIGHTS_EXECUTE | COM_RIGHTS_EXECUTE_LOCAL: calls from this
// computer only (written so, it reads back the same). COM requires an owner
// and a group.
const comCallersSDDL = "O:SYG:SYD:(A;;CCDC;;;SY)(A;;CCDC;;;BA)(A;;CCDC;;;BO)(A;;CCDC;;;NS)(A;;CCDC;;;LS)"

var (
	comSecurityOnce sync.Once
	// comSecurityResult is what CoInitializeSecurity answered (for the tests
	// and the log): 0 once set, rpcETooLate where COM was in use already.
	comSecurityResult uintptr
	// comCallersSD keeps the descriptor handed to COM alive for the life of
	// the process, should COM refer to it rather than copy it.
	comCallersSD *windows.SECURITY_DESCRIPTOR
	// comCallersRel holds the owner, group and DACL comCallersSD points to.
	comCallersRel *windows.SECURITY_DESCRIPTOR

	procCoInitializeSecurity = windows.NewLazySystemDLL("ole32.dll").NewProc("CoInitializeSecurity")
)

// comCallersDescriptor builds comCallersSDDL as the absolute security
// descriptor CoInitializeSecurity takes. Not with ToAbsolute: the
// golang.org/x/sys this module is built with returns a descriptor that is not
// valid (IsValid false), and COM refuses it with E_INVALIDARG (seen on Server
// 2022). Its owner, group and DACL are set on a new descriptor instead; they
// point into the self-relative one, which is kept with it.
func comCallersDescriptor() (*windows.SECURITY_DESCRIPTOR, error) {
	rel, err := windows.SecurityDescriptorFromString(comCallersSDDL)
	if err != nil {
		return nil, err
	}
	owner, _, err := rel.Owner()
	if err != nil {
		return nil, fmt.Errorf("owner: %w", err)
	}
	group, _, err := rel.Group()
	if err != nil {
		return nil, fmt.Errorf("group: %w", err)
	}
	dacl, _, err := rel.DACL()
	if err != nil {
		return nil, fmt.Errorf("DACL: %w", err)
	}
	abs, err := windows.NewSecurityDescriptor()
	if err != nil {
		return nil, err
	}
	if err := abs.SetOwner(owner, false); err != nil {
		return nil, fmt.Errorf("owner: %w", err)
	}
	if err := abs.SetGroup(group, false); err != nil {
		return nil, fmt.Errorf("group: %w", err)
	}
	if err := abs.SetDACL(dacl, true, false); err != nil {
		return nil, fmt.Errorf("DACL: %w", err)
	}
	if !abs.IsValid() {
		return nil, fmt.Errorf("the descriptor built from %s is not valid", comCallersSDDL)
	}
	comCallersRel = rel
	return abs, nil
}

// setCOMSecurity sets the process's COM security (see above). COM must be
// initialised on the calling thread. Run it once (comSecurityOnce).
func setCOMSecurity() {
	sd, err := comCallersDescriptor()
	if err != nil {
		fmt.Printf("VSS: COM security not set (security descriptor: %v); writers that run as another account may log VSS event 8194\n", err)
		return
	}
	comCallersSD = sd
	hr, _, _ := procCoInitializeSecurity.Call(
		uintptr(unsafe.Pointer(sd)),
		^uintptr(0), // cAuthSvc -1: COM picks the authentication services
		0, 0,
		rpcCAuthnLevelPktPrivacy,
		rpcCImpLevelImpersonate,
		0,
		eoacDynamicCloaking,
		0,
	)
	comSecurityResult = hr
	switch uint32(hr) {
	case 0:
		fmt.Println("VSS: COM security set for the writers' callbacks")
	case rpcETooLate:
		fmt.Println("VSS: COM was already in use in this process, so its security could not be set; writers that run as another account may log VSS event 8194")
	default:
		fmt.Printf("VSS: COM security not set (CoInitializeSecurity 0x%08X); writers that run as another account may log VSS event 8194\n", uint32(hr))
	}
}
