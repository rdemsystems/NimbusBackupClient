//go:build linux

package main

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// On Linux the networks are NetworkManager's connections, read with nmcli:
// the active ones, whether the device each is on is metered, and every saved
// connection with when it was last active. Without nmcli (another network
// manager, a server) the networks are not known (netinfo.go).

var netInfoTimeout = 15 * time.Second

// runNmcli runs nmcli with args in its terse form; a variable for the tests.
var runNmcli = func(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netInfoTimeout)
	defer cancel()
	return exec.CommandContext(ctx, "nmcli", append([]string{"-t"}, args...)...).Output() // #nosec G204 -- nmcli with our own fixed arguments
}

func readNetworkInfoSystem() NetworkInfo {
	active, err := runNmcli("-f", "NAME,TYPE,DEVICE", "connection", "show", "--active")
	if err != nil {
		writeDebugLog(fmt.Sprintf("[Network] the networks could not be read (nmcli): %v", err))
		return NetworkInfo{}
	}
	info := NetworkInfo{Supported: true, Current: []NetConn{}, Known: []KnownNetwork{}}
	for _, line := range strings.Split(string(active), "\n") {
		f := splitNmcli(line)
		if len(f) < 3 || f[0] == "" {
			continue
		}
		kind := nmConnKind(f[1])
		if f[1] == "loopback" {
			continue
		}
		info.Current = append(info.Current, NetConn{Name: f[0], Kind: kind})
		if f[2] == "" {
			continue
		}
		if out, err := runNmcli("-f", "GENERAL.METERED", "device", "show", f[2]); err == nil {
			v := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(out)), "GENERAL.METERED:")))
			switch {
			case strings.HasPrefix(v, "yes"):
				info.MeteredKnown, info.Metered = true, true
			case strings.HasPrefix(v, "no"):
				info.MeteredKnown = true
			}
		}
	}
	if saved, err := runNmcli("-f", "NAME,TYPE,TIMESTAMP", "connection", "show"); err == nil {
		for _, line := range strings.Split(string(saved), "\n") {
			f := splitNmcli(line)
			if len(f) < 3 || f[0] == "" || f[1] == "loopback" {
				continue
			}
			kn := KnownNetwork{Name: f[0], Kind: nmConnKind(f[1])}
			if secs, err := strconv.ParseInt(f[2], 10, 64); err == nil && secs > 0 {
				kn.LastSeen = time.Unix(secs, 0).Format(time.RFC3339)
			}
			info.Known = append(info.Known, kn)
		}
	}
	return info
}

// splitNmcli splits one line of nmcli's terse output at its colons, where a
// colon or backslash in a value is written with a backslash before it.
func splitNmcli(line string) []string {
	line = strings.TrimRight(line, "\r")
	if line == "" {
		return nil
	}
	var out []string
	var b strings.Builder
	esc := false
	for _, r := range line {
		switch {
		case esc:
			b.WriteRune(r)
			esc = false
		case r == '\\':
			esc = true
		case r == ':':
			out = append(out, b.String())
			b.Reset()
		default:
			b.WriteRune(r)
		}
	}
	return append(out, b.String())
}

// nmConnKind is the kind of a NetworkManager connection type.
func nmConnKind(t string) string {
	switch t {
	case "802-3-ethernet", "ethernet":
		return connWired
	case "802-11-wireless", "wifi":
		return connWifi
	case "vpn", "wireguard":
		return connVPN
	case "gsm", "cdma":
		return connMobile
	}
	return connOther
}
