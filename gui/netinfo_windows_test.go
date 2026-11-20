//go:build windows

package main

import "testing"

// PowerShell 5.1 writes a list of one as the item itself; both forms are read.
func TestParseWindowsNetInfo(t *testing.T) {
	out := []byte(`{"current":{"name":"Home","ifType":71,"desc":"Intel(R) Dual Band Wireless-AC 8265"},"cost":"Unrestricted",` +
		`"known":[{"name":"Home","nameType":71,"last":"2026-11-20T19:23:05"},{"name":"Network 2","nameType":6,"last":""},{"name":"Office VPN","nameType":23,"last":"2026-11-17T08:00:00"}]}`)
	info, err := parseWindowsNetInfo(out)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Supported || len(info.Current) != 1 || info.Current[0] != (NetConn{Name: "Home", Kind: connWifi}) {
		t.Fatalf("current %+v", info.Current)
	}
	if !info.MeteredKnown || info.Metered {
		t.Errorf("metered %v known %v, want not metered", info.Metered, info.MeteredKnown)
	}
	if len(info.Known) != 3 || info.Known[1].Kind != connWired || info.Known[1].LastSeen != "" || info.Known[2].Kind != connVPN || info.Known[0].LastSeen == "" {
		t.Errorf("known %+v", info.Known)
	}

	metered, err := parseWindowsNetInfo([]byte(`{"current":[{"name":"Phone","ifType":71,"desc":"x"},{"name":"Work","ifType":6,"desc":"WireGuard Tunnel"}],"cost":"Fixed","known":null}`))
	if err != nil {
		t.Fatal(err)
	}
	if !metered.Metered || len(metered.Current) != 2 || metered.Current[1].Kind != connVPN {
		t.Errorf("got %+v", metered)
	}
}
