//go:build service
// +build service

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/kardianos/service"
)

func main() {
	// -last-input: run by the service in a signed-in session, to ask when the
	// computer was last used there (user_input.go). Before anything is logged.
	runLastInputArg(os.Args)
	// -apply-settings <file>: the installer's CONFIGFILE (settings_deploy.go).
	if path, ok := applySettingsArg(os.Args[1:]); ok {
		os.Exit(runApplySettings(path))
	}
	// -list-sets, -undelete, -rollback: a command-line run, not the service.
	if runSetCLI(os.Args[1:]) {
		return
	}
	writeDebugLog(serviceExeBase() + " starting...")
	startStallWatchdog()

	// Command-line flags for service control
	svcFlag := flag.String("service", "", "Control the system service: install, uninstall, start, stop, restart")
	// -pidfile: started by an init script on a computer without systemd
	// (SysV, Slackware), which finds the service by the process it names.
	pidFile := flag.String("pidfile", "", "Write the service's process ID to this file (init scripts without systemd)")
	flag.Parse()
	if *pidFile != "" {
		if err := writePIDFile(*pidFile); err != nil {
			log.Fatal(err)
		}
		defer os.Remove(*pidFile)
	}

	// Service configuration. The name matches what the installers and
	// Preferences > Run as Service use (on Windows the service program's own
	// name, salvanza-svc; on Linux the packaged unit), so "-service install"
	// does not make a second one.
	svcConfig := &service.Config{
		Name:        serviceName(),
		DisplayName: product().Title + " Service",
		Description: "Runs scheduled backups to Proxmox Backup Server with VSS support",
	}
	if runtime.GOOS == "linux" {
		svcConfig.Name = linuxServiceUnit
		svcConfig.Description = "Runs scheduled backups to Proxmox Backup Server"
	}

	backupSvc := &BackupService{}
	s, err := service.New(backupSvc, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Handle service control commands (install, uninstall, start, stop)
	if len(*svcFlag) != 0 {
		err := service.Control(s, *svcFlag)
		if err != nil {
			log.Printf("Valid actions: %q\n", service.ControlAction)
			log.Fatal(err)
		}
		writeDebugLog(fmt.Sprintf("Service control action '%s' completed", *svcFlag))
		return
	}

	// Run service
	writeDebugLog("Starting service...")
	err = s.Run()
	if err != nil {
		log.Fatal(err)
	}
}
