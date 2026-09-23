//go:build windows || service

package main

import (
	"os"
	"path/filepath"
	"strings"
)

// defaultServiceName is the SCM name used when the executable name cannot be
// resolved. It matches the default brand's installer (Product.wxs ExeName).
const defaultServiceName = "ProxmoxBackupClient"

// serviceExeSuffix is appended to the brand's ExeName to name the service
// binary ($(var.ServiceExeName) = "<ExeName>SVC" in installer/wix).
const serviceExeSuffix = "SVC"

// serviceNameFromExecutable returns the Windows service name for this binary.
// The MSI registers the service under the brand's ExeName (ProductBody.wxi:
// ServiceInstall Name="$(var.ExeName)", e.g. "NimbusBackup" — the name every
// Nimbus install has used), and the service binary is "<ExeName>SVC.exe". So
// strip the extension and the "SVC" suffix from os.Executable, which keeps
// `-service start|stop|uninstall` pointed at the service the installer
// actually registered, whatever the brand.
func serviceNameFromExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		return defaultServiceName
	}
	base := filepath.Base(exe)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if len(name) > len(serviceExeSuffix) && strings.EqualFold(name[len(name)-len(serviceExeSuffix):], serviceExeSuffix) {
		name = name[:len(name)-len(serviceExeSuffix)]
	}
	if name == "" {
		return defaultServiceName
	}
	return name
}
