//go:build !windows && !linux

package main

// readNetworkInfoSystem: networks are not read on this system, so network
// choices let every run go ahead (netinfo.go).
func readNetworkInfoSystem() NetworkInfo { return NetworkInfo{} }
