//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The newest Power-Troubleshooter event 1 in the System log, read through the
// event log API, with wevtutil (System32) to fall back on where a call is
// refused, as the after-backup shutdown keeps shutdown.exe.
var (
	wevtapi        = windows.NewLazySystemDLL("wevtapi.dll")
	procEvtQuery   = wevtapi.NewProc("EvtQuery")
	procEvtNext    = wevtapi.NewProc("EvtNext")
	procEvtRender  = wevtapi.NewProc("EvtRender")
	procEvtClose   = wevtapi.NewProc("EvtClose")
	wakeEventQuery = "*[System[Provider[@Name='Microsoft-Windows-Power-Troubleshooter'] and (EventID=1)]]"
)

const (
	evtQueryChannelPath      = 0x1
	evtQueryReverseDirection = 0x200
	evtRenderEventXML        = 1
	errInsufficientBuffer    = 122
)

func platformLatestWakeEvent() (wakeEvent, error) {
	data, err := latestWakeEventAPI()
	if err != nil {
		var werr error
		if data, werr = latestWakeEventWevtutil(); werr != nil {
			return wakeEvent{}, fmt.Errorf("%v; wevtutil: %v", err, werr)
		}
	}
	return parseWakeEventXML(data)
}

// latestWakeEventAPI renders the newest wake event as XML.
func latestWakeEventAPI() ([]byte, error) {
	if err := wevtapi.Load(); err != nil {
		return nil, err
	}
	path, _ := windows.UTF16PtrFromString("System")
	query, _ := windows.UTF16PtrFromString(wakeEventQuery)
	q, _, err := procEvtQuery.Call(0, uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(query)), evtQueryChannelPath|evtQueryReverseDirection)
	if q == 0 {
		return nil, fmt.Errorf("EvtQuery: %v", err)
	}
	defer func() { _, _, _ = procEvtClose.Call(q) }()
	var ev uintptr
	var returned uint32
	if ok, _, err := procEvtNext.Call(q, 1, uintptr(unsafe.Pointer(&ev)), 1000, 0, uintptr(unsafe.Pointer(&returned))); ok == 0 || returned == 0 {
		return nil, fmt.Errorf("EvtNext: %v", err)
	}
	defer func() { _, _, _ = procEvtClose.Call(ev) }()
	var used, props uint32
	ok, _, err := procEvtRender.Call(0, ev, evtRenderEventXML, 0, 0, uintptr(unsafe.Pointer(&used)), uintptr(unsafe.Pointer(&props)))
	if ok == 0 && err != windows.Errno(errInsufficientBuffer) {
		return nil, fmt.Errorf("EvtRender: %v", err)
	}
	if used < 2 {
		return nil, fmt.Errorf("EvtRender: an empty event")
	}
	buf := make([]uint16, used/2+1)
	if ok, _, err := procEvtRender.Call(0, ev, evtRenderEventXML, uintptr(len(buf)*2), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&used)), uintptr(unsafe.Pointer(&props))); ok == 0 {
		return nil, fmt.Errorf("EvtRender: %v", err)
	}
	return []byte(windows.UTF16ToString(buf)), nil
}

// latestWakeEventWevtutil asks wevtutil for the same event.
func latestWakeEventWevtutil() ([]byte, error) {
	exe := "wevtutil.exe"
	if win := os.Getenv("SystemRoot"); win != "" {
		exe = filepath.Join(win, "System32", "wevtutil.exe") // not whatever PATH finds first
	}
	cmd := exec.Command(exe, "qe", "System", "/q:"+wakeEventQuery, "/c:1", "/rd:true", "/f:xml") // #nosec G204 -- wevtutil from System32, with our own fixed arguments
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no wake event in the System log")
	}
	return out, nil
}
