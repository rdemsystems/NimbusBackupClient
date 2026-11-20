package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// The networks this computer is on and has been on, for the triggers that
// run only on some networks or not on metered connections (server_trigger.go).
// Windows names a network as Settings, Network does (the Wi-Fi network's name,
// "Network 2" for a wired one unless renamed), from its network list; Linux
// by NetworkManager's connection name. A computer where they cannot be read
// (no NetworkManager, another system) reports Supported false, and the
// network choices then let every run go ahead: a set must not stop backing up
// because the system does not say which network it is on.

// Connection kinds.
const (
	connWired  = "wired"
	connWifi   = "wifi"
	connVPN    = "vpn"
	connMobile = "mobile"
	connOther  = "other"
)

// connectionKinds are the kinds a trigger may be limited to.
var connectionKinds = map[string]bool{connWired: true, connWifi: true, connVPN: true, connMobile: true}

// NetConn is one network the computer is connected to now.
type NetConn struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// KnownNetwork is a network this computer has joined, with when it was last
// connected (RFC 3339, "" when not known).
type KnownNetwork struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	LastSeen  string `json:"lastSeen,omitempty"`
	Connected bool   `json:"connected"`
}

// NetworkInfo is what the system says about the computer's networks.
type NetworkInfo struct {
	Supported    bool           `json:"supported"`
	Current      []NetConn      `json:"current"`
	Metered      bool           `json:"metered"`
	MeteredKnown bool           `json:"meteredKnown"`
	Known        []KnownNetwork `json:"known"`
}

// readNetworkInfo asks the system (netinfo_windows.go, netinfo_linux.go). A
// variable so the tests stand in for it.
var readNetworkInfo = readNetworkInfoSystem

// netInfoMaxAge is how long a reading is used again: the scheduler asks every
// minute for each set with a network choice, and asking costs a process.
var netInfoMaxAge = 60 * time.Second

var netInfoCache struct {
	sync.Mutex
	at   time.Time
	info NetworkInfo
}

// currentNetworkInfo returns the networks, read again when the last reading
// is older than maxAge.
func currentNetworkInfo(maxAge time.Duration) NetworkInfo {
	netInfoCache.Lock()
	defer netInfoCache.Unlock()
	now := time.Now()
	if !netInfoCache.at.IsZero() && now.Sub(netInfoCache.at) < maxAge {
		return netInfoCache.info
	}
	info := readNetworkInfo()
	// Connected networks are always in the known list, marked as such.
	seen := map[string]bool{}
	for i := range info.Known {
		for _, c := range info.Current {
			if strings.EqualFold(c.Name, info.Known[i].Name) {
				info.Known[i].Connected = true
			}
		}
		seen[strings.ToLower(info.Known[i].Name)] = true
	}
	for _, c := range info.Current {
		if !seen[strings.ToLower(c.Name)] {
			info.Known = append(info.Known, KnownNetwork{Name: c.Name, Kind: c.Kind, LastSeen: now.Format(time.RFC3339), Connected: true})
			seen[strings.ToLower(c.Name)] = true
		}
	}
	netInfoCache.at, netInfoCache.info = now, info
	return info
}

// GetNetworkInfo tells the window the computer's networks now and the ones
// it has joined, for a trigger's network choices.
func (a *App) GetNetworkInfo() NetworkInfo {
	if a.delegated() {
		var out NetworkInfo
		if err := a.apiClient.Call("GetNetworkInfo", &out); err == nil {
			return out
		}
	}
	info := currentNetworkInfo(5 * time.Second)
	if info.Current == nil {
		info.Current = []NetConn{}
	}
	if info.Known == nil {
		info.Known = []KnownNetwork{}
	}
	return info
}

// Why a network choice stops a run (networkAllows).
const (
	netBlockNone       = ""
	netBlockNetwork    = "network"    // not on one of the chosen networks
	netBlockConnection = "connection" // not a chosen kind of connection
	netBlockMetered    = "metered"    // the connection is metered
)

// networkAllows reports whether a run limited to networks (names; empty =
// any), connections (kinds; empty = any) and not metered connections may go
// ahead on the networks in info, and when not, why. With Supported false
// every run goes ahead.
func networkAllows(info NetworkInfo, networks, connections []string, notMetered bool) (bool, string) {
	if !info.Supported {
		return true, netBlockNone
	}
	if len(networks) > 0 {
		on := false
		for _, c := range info.Current {
			for _, n := range networks {
				if strings.EqualFold(strings.TrimSpace(n), c.Name) {
					on = true
				}
			}
		}
		if !on {
			return false, netBlockNetwork
		}
	}
	if len(connections) > 0 {
		on := false
		for _, c := range info.Current {
			for _, k := range connections {
				if k == c.Kind {
					on = true
				}
			}
		}
		if !on {
			return false, netBlockConnection
		}
	}
	if notMetered && info.MeteredKnown && info.Metered {
		return false, netBlockMetered
	}
	return true, netBlockNone
}

// netNames puts the connected networks in words for a log line: "Home
// (wifi), Office (wired)", or "no network".
func netNames(info NetworkInfo) string {
	if !info.Supported {
		return "networks not known"
	}
	if len(info.Current) == 0 {
		return "no network"
	}
	var parts []string
	for _, c := range info.Current {
		parts = append(parts, fmt.Sprintf("%s (%s)", c.Name, c.Kind))
	}
	return strings.Join(parts, ", ")
}
