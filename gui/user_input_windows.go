//go:build windows

package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// wtsActive is WTS_CONNECTSTATE_CLASS WTSActive: someone signed in at it.
	wtsActive = 0
	// lastInputArg is the command the service runs in a user's session
	// (service_last_input_windows.go).
	lastInputArg = "-last-input"
)

var (
	user32Input          = windows.NewLazySystemDLL("user32.dll")
	procGetLastInputInfo = user32Input.NewProc("GetLastInputInfo")
	procGetTickCount     = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetTickCount")
)

// sessionIdleMillis is how long ago this session's last keyboard or mouse
// input was (GetLastInputInfo, which counts for the session the program runs
// in only).
func sessionIdleMillis() (uint32, error) {
	var info struct{ cbSize, dwTime uint32 }
	info.cbSize = uint32(unsafe.Sizeof(info))
	if ok, _, err := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&info))); ok == 0 {
		return 0, fmt.Errorf("GetLastInputInfo: %v", err)
	}
	now, _, _ := procGetTickCount.Call()
	return uint32(now) - info.dwTime, nil // wraps as the tick count does
}

// platformLastUserInput: in a signed-in session (the app scheduling on its
// own) it asks here; in the service (session 0) it asks in each active
// signed-in session through runInSession.
func platformLastUserInput() (time.Time, bool) {
	var own uint32
	if err := windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &own); err == nil && own != 0 {
		ms, err := sessionIdleMillis()
		if err != nil {
			return time.Time{}, false
		}
		return time.Now().Add(-time.Duration(ms) * time.Millisecond), true
	}
	var sessions *windows.WTS_SESSION_INFO
	var count uint32
	if err := windows.WTSEnumerateSessions(0, 0, 1, &sessions, &count); err != nil {
		writeDebugLog(fmt.Sprintf("[After backup] could not list the signed-in sessions: %v", err))
		return time.Time{}, false
	}
	defer windows.WTSFreeMemory(uintptr(unsafe.Pointer(sessions)))
	var latest time.Time
	known := false
	for _, s := range unsafe.Slice(sessions, count) {
		if s.SessionID == 0 || s.State != wtsActive {
			continue
		}
		ms, err := idleMillisInSession(s.SessionID)
		if err != nil {
			writeDebugLog(fmt.Sprintf("[After backup] could not ask session %d when it was last used: %v", s.SessionID, err))
			continue
		}
		at := time.Now().Add(-time.Duration(ms) * time.Millisecond)
		if !known || at.After(latest) {
			latest, known = at, true
		}
	}
	return latest, known
}

// idleMillisInSession runs this program with -last-input in the session, as
// its signed-in user, with no window, and reads what it prints.
func idleMillisInSession(session uint32) (uint32, error) {
	var token windows.Token
	if err := windows.WTSQueryUserToken(session, &token); err != nil {
		return 0, fmt.Errorf("WTSQueryUserToken: %w", err)
	}
	defer token.Close()
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}
	sa := windows.SecurityAttributes{InheritHandle: 1}
	sa.Length = uint32(unsafe.Sizeof(sa))
	var r, w windows.Handle
	if err := windows.CreatePipe(&r, &w, &sa, 0); err != nil {
		return 0, fmt.Errorf("CreatePipe: %w", err)
	}
	if err := windows.SetHandleInformation(r, windows.HANDLE_FLAG_INHERIT, 0); err != nil {
		_ = windows.CloseHandle(r)
		_ = windows.CloseHandle(w)
		return 0, err
	}
	out := os.NewFile(uintptr(r), "last-input")
	defer out.Close()
	desktop, _ := windows.UTF16PtrFromString(`winsta0\default`)
	si := windows.StartupInfo{Desktop: desktop, Flags: windows.STARTF_USESTDHANDLES | windows.STARTF_USESHOWWINDOW,
		ShowWindow: windows.SW_HIDE, StdOutput: w, StdErr: w}
	si.Cb = uint32(unsafe.Sizeof(si))
	app, _ := windows.UTF16PtrFromString(exe)
	cmd, _ := windows.UTF16PtrFromString(windows.EscapeArg(exe) + " " + lastInputArg)
	var pi windows.ProcessInformation
	err = windows.CreateProcessAsUser(token, app, cmd, nil, nil, true, windows.CREATE_NO_WINDOW, nil, nil, &si, &pi)
	_ = windows.CloseHandle(w) // the child has its own copy
	if err != nil {
		return 0, fmt.Errorf("CreateProcessAsUser: %w", err)
	}
	defer func() { _ = windows.CloseHandle(pi.Process) }()
	defer func() { _ = windows.CloseHandle(pi.Thread) }()
	if ev, _ := windows.WaitForSingleObject(pi.Process, 10000); ev != windows.WAIT_OBJECT_0 {
		_ = windows.TerminateProcess(pi.Process, 1)
		return 0, fmt.Errorf("it did not answer within 10 seconds")
	}
	data, err := io.ReadAll(io.LimitReader(out, 4096))
	if err != nil {
		return 0, err
	}
	var code uint32
	if err := windows.GetExitCodeProcess(pi.Process, &code); err == nil && code != 0 {
		return 0, fmt.Errorf("it said: %s", strings.TrimSpace(string(data)))
	}
	ms, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("it printed %q", strings.TrimSpace(string(data)))
	}
	return uint32(ms), nil
}
