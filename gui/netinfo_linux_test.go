//go:build linux

package main

import (
	"errors"
	"strings"
	"testing"
)

func TestSplitNmcli(t *testing.T) {
	got := splitNmcli(`Caf\:e \\ net:802-11-wireless:wlp2s0`)
	if len(got) != 3 || got[0] != `Caf:e \ net` || got[1] != "802-11-wireless" || got[2] != "wlp2s0" {
		t.Fatalf("got %q", got)
	}
}

// The networks from nmcli: the active connections (not loopback), the
// device's metered state, and the saved connections with when last active.
func TestReadNetworkInfoNmcli(t *testing.T) {
	old := runNmcli
	t.Cleanup(func() { runNmcli = old })
	runNmcli = func(args ...string) ([]byte, error) {
		a := strings.Join(args, " ")
		switch {
		case strings.Contains(a, "--active"):
			return []byte("Home:802-11-wireless:wlan0\nlo:loopback:lo\n"), nil
		case strings.Contains(a, "GENERAL.METERED"):
			return []byte("GENERAL.METERED:yes (guessed)\n"), nil
		case strings.Contains(a, "TIMESTAMP"):
			return []byte("Home:802-11-wireless:1763670000\nWired connection 1:802-3-ethernet:0\nlo:loopback:0\n"), nil
		}
		return nil, errors.New("unexpected " + a)
	}
	info := readNetworkInfoSystem()
	if !info.Supported || len(info.Current) != 1 || info.Current[0].Name != "Home" || info.Current[0].Kind != connWifi {
		t.Fatalf("current %+v", info.Current)
	}
	if !info.Metered || !info.MeteredKnown {
		t.Error("a metered connection was not seen")
	}
	if len(info.Known) != 2 || info.Known[0].LastSeen == "" || info.Known[1].Kind != connWired || info.Known[1].LastSeen != "" {
		t.Errorf("known %+v", info.Known)
	}

	runNmcli = func(...string) ([]byte, error) { return nil, errors.New("nmcli: not found") }
	if readNetworkInfoSystem().Supported {
		t.Error("without nmcli the networks are reported as known")
	}
}
