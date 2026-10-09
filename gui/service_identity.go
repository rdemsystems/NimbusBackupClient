package main

import "strings"

// serviceIdentityForExeBase maps a service executable's base name (as the WiX
// installer names it: $(var.ServiceExeName).exe, i.e. "<ExeName>SVC") to the
// (SCM name, display name) pair the service has to register under.
//
// The SCM name is the brand's ExeName, the executable base name without its
// "SVC" suffix, because installer/wix/ProductBody.wxi registers the service
// as Name="$(var.ExeName)": that is the name every Nimbus Backup MSI up to
// 0.3.0 registered ("NimbusBackup"), which scripts and monitoring target.
// Registering anything else makes StartServiceCtrlDispatcher fail with
// ERROR_FAILED_SERVICE_CONTROLLER_CONNECT (1061) and Windows reports "service
// failed to start".
//
// The display name mirrors the installer's DisplayName="$(var.ProductName) Service".
func serviceIdentityForExeBase(exeBase string) (name, displayName string) {
	name = strings.TrimSuffix(exeBase, "SVC")
	if name == "" {
		name = exeBase
	}
	return name, ResolveBrand(name).Title + " Service"
}
