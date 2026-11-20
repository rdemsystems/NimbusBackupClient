//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// netInfoScript asks Windows for the connected networks (their names as
// Settings, Network shows them, and the adapter each is on), whether the
// internet connection is metered (Windows Runtime's connection cost) and
// the networks in the network list with when each was last connected (the
// list's DateLastConnected, a SYSTEMTIME in local time). It prints one line
// of JSON. Nothing is changed.
const netInfoScript = `$ErrorActionPreference = 'SilentlyContinue'
$cur = @(Get-NetConnectionProfile | ForEach-Object {
  $a = Get-NetAdapter -InterfaceIndex $_.InterfaceIndex
  [pscustomobject]@{ name = [string]$_.Name; ifType = [int]$a.InterfaceType; desc = [string]$a.InterfaceDescription }
})
$cost = ''
try {
  $null = [Windows.Networking.Connectivity.NetworkInformation, Windows.Networking.Connectivity, ContentType = WindowsRuntime]
  $p = [Windows.Networking.Connectivity.NetworkInformation]::GetInternetConnectionProfile()
  if ($p) { $cost = [string]$p.GetConnectionCost().NetworkCostType }
} catch { }
$known = @(Get-ChildItem 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\NetworkList\Profiles' | ForEach-Object {
  $v = Get-ItemProperty $_.PSPath
  $b = $v.DateLastConnected
  $t = ''
  if ($b -and $b.Length -ge 14) {
    $t = '{0:D4}-{1:D2}-{2:D2}T{3:D2}:{4:D2}:{5:D2}' -f [BitConverter]::ToUInt16($b, 0), [BitConverter]::ToUInt16($b, 2), [BitConverter]::ToUInt16($b, 6), [BitConverter]::ToUInt16($b, 8), [BitConverter]::ToUInt16($b, 10), [BitConverter]::ToUInt16($b, 12)
  }
  [pscustomobject]@{ name = [string]$v.ProfileName; nameType = [int]$v.NameType; last = $t }
})
[pscustomobject]@{ current = $cur; cost = $cost; known = $known } | ConvertTo-Json -Depth 4 -Compress`

// netInfoTimeout bounds one reading: PowerShell's first start can be slow.
var netInfoTimeout = 30 * time.Second

func readNetworkInfoSystem() NetworkInfo {
	exe := "powershell.exe"
	if win := os.Getenv("SystemRoot"); win != "" {
		exe = filepath.Join(win, "System32", "WindowsPowerShell", "v1.0", "powershell.exe") // not whatever PATH finds first
	}
	ctx, cancel := context.WithTimeout(context.Background(), netInfoTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", netInfoScript) // #nosec G204 -- PowerShell from System32 with our own fixed script
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		writeDebugLog(fmt.Sprintf("[Network] the networks could not be read: %v", err))
		return NetworkInfo{}
	}
	info, err := parseWindowsNetInfo(out)
	if err != nil {
		writeDebugLog(fmt.Sprintf("[Network] the networks could not be read: %v", err))
		return NetworkInfo{}
	}
	return info
}

// windowsNetInfo is the script's output. PowerShell 5.1 writes a list of one
// as the item itself, so the lists are read either way (oneOrMany).
type windowsNetInfo struct {
	Current json.RawMessage `json:"current"`
	Cost    string          `json:"cost"`
	Known   json.RawMessage `json:"known"`
}

type windowsNetConn struct {
	Name   string `json:"name"`
	IfType int    `json:"ifType"`
	Desc   string `json:"desc"`
}

type windowsKnownNet struct {
	Name     string `json:"name"`
	NameType int    `json:"nameType"`
	Last     string `json:"last"`
}

// oneOrMany reads a JSON list, or one item standing for a list of one.
func oneOrMany[T any](raw json.RawMessage) ([]T, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return nil, nil
	}
	if strings.HasPrefix(s, "[") {
		var out []T
		err := json.Unmarshal(raw, &out)
		return out, err
	}
	var one T
	if err := json.Unmarshal(raw, &one); err != nil {
		return nil, err
	}
	return []T{one}, nil
}

// parseWindowsNetInfo reads the script's output.
func parseWindowsNetInfo(out []byte) (NetworkInfo, error) {
	var w windowsNetInfo
	if err := json.Unmarshal(stripUTF8BOM(out), &w); err != nil {
		return NetworkInfo{}, err
	}
	cur, err := oneOrMany[windowsNetConn](w.Current)
	if err != nil {
		return NetworkInfo{}, err
	}
	known, err := oneOrMany[windowsKnownNet](w.Known)
	if err != nil {
		return NetworkInfo{}, err
	}
	info := NetworkInfo{Supported: true, Current: []NetConn{}, Known: []KnownNetwork{}}
	for _, c := range cur {
		if c.Name == "" {
			continue
		}
		info.Current = append(info.Current, NetConn{Name: c.Name, Kind: windowsConnKind(c.IfType, c.Desc)})
	}
	switch strings.ToLower(w.Cost) {
	case "unrestricted":
		info.MeteredKnown = true
	case "fixed", "variable":
		info.MeteredKnown, info.Metered = true, true
	}
	for _, k := range known {
		if k.Name == "" {
			continue
		}
		kn := KnownNetwork{Name: k.Name, Kind: windowsConnKind(k.NameType, "")}
		if t, err := time.ParseInLocation("2006-01-02T15:04:05", k.Last, time.Local); err == nil && t.Year() > 1601 {
			kn.LastSeen = t.Format(time.RFC3339)
		}
		info.Known = append(info.Known, kn)
	}
	return info, nil
}

// windowsConnKind is the kind of a connection from its interface type (IANA
// ifType, as Get-NetAdapter and the network list give it) and the adapter's
// description: a VPN adapter is often a virtual Ethernet one.
func windowsConnKind(ifType int, desc string) string {
	d := strings.ToLower(desc)
	for _, v := range []string{"vpn", "wireguard", "tap-windows", "openvpn", "wintun", "anyconnect", "fortinet", "globalprotect"} {
		if strings.Contains(d, v) {
			return connVPN
		}
	}
	switch ifType {
	case 6:
		return connWired
	case 71:
		return connWifi
	case 23, 53, 131:
		return connVPN
	case 243, 244:
		return connMobile
	}
	return connOther
}
