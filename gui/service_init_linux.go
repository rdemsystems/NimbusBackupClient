//go:build linux && !service

package main

// The background service on a Linux computer without systemd: SysV init
// (Devuan, older Debian; /etc/init.d), Slackware (/etc/rc.d/rc.<name>, started
// from rc.local) and OpenRC (Alpine, Gentoo). The app writes the init script
// (initscripts/), turns it on at boot, and starts, stops and removes the
// service with it, as it does with systemctl on a systemd computer
// (service_control_linux.go). The service writes its own pid file, which is
// how the app and the scripts tell that it runs.

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"pbscommon"
)

//go:embed initscripts/service.sysv
var sysvScriptTemplate string

//go:embed initscripts/service.openrc
var openrcScriptTemplate string

// The service managers the app knows.
const (
	initSystemd   = "systemd"
	initSysV      = "sysv"
	initSlackware = "slackware"
	initOpenRC    = "openrc"
)

// initRoot is "/" except in the tests, which give a folder standing in for
// the computer's root.
var initRoot = "/"

func initPath(p string) string { return filepath.Join(initRoot, p) }

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// detectInit names the service manager this computer was started with, or ""
// when it has none the app can use.
func detectInit() string {
	switch {
	case dirExists(initPath("/run/systemd/system")):
		return initSystemd
	case dirExists(initPath("/run/openrc")):
		return initOpenRC
	case fileExists(initPath("/etc/slackware-version")) && dirExists(initPath("/etc/rc.d")):
		return initSlackware
	case dirExists(initPath("/etc/init.d")):
		return initSysV
	}
	return ""
}

// noInitMessage is shown when the computer has no service manager the app
// can install the background service with.
const noInitMessage = "This computer has no service manager the background service can be installed with (systemd, SysV init, Slackware's rc.d or OpenRC). The app runs the Backup Sets itself while it is open."

// errNoInit is the same as an error, for the buttons that cannot act.
var errNoInit = errors.New("this computer has no service manager the background service can be installed with (systemd, SysV init, Slackware's rc.d or OpenRC)")

// initService is the background service under a service manager other than
// systemd.
type initService struct {
	kind string
	name string // linuxServiceUnit
}

func newInitService(kind string) initService {
	return initService{kind: kind, name: linuxServiceUnit}
}

// scriptPath is where its init script goes.
func (s initService) scriptPath() string {
	if s.kind == initSlackware {
		return initPath("/etc/rc.d/rc." + s.name)
	}
	return initPath("/etc/init.d/" + s.name)
}

func (s initService) pidFile() string {
	if dirExists(initPath("/run")) {
		return initPath("/run/" + s.name + ".pid")
	}
	return initPath("/var/run/" + s.name + ".pid")
}

// script is the init script for the service program exe.
func (s initService) script(exe string) string {
	t := sysvScriptTemplate
	if s.kind == initOpenRC {
		t = openrcScriptTemplate
	}
	r := strings.NewReplacer("@NAME@", s.name, "@EXE@", exe, "@TITLE@", product().Title)
	return r.Replace(t)
}

var scriptExeRe = regexp.MustCompile(`(?m)^(?:EXE|command)="([^"]+)"`)

// exePath is the service program the installed script starts ("" when it
// cannot be read).
func (s initService) exePath() string {
	b, err := os.ReadFile(s.scriptPath())
	if err != nil {
		return ""
	}
	if m := scriptExeRe.FindSubmatch(b); m != nil {
		return string(m[1])
	}
	return ""
}

func (s initService) installed() bool { return fileExists(s.scriptPath()) }

// pid is the running service's process, read from its pid file and checked
// against the program the script starts.
func (s initService) pid() (int, bool) {
	b, err := os.ReadFile(s.pidFile())
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	if err := syscall.Kill(pid, 0); err != nil && !errors.Is(err, syscall.EPERM) {
		return 0, false
	}
	// /proc/<pid>/exe is root's to read for root's process; the command line
	// is readable by everyone, so the app run without administrator rights
	// still sees the service running.
	exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		cmd, cerr := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if cerr != nil {
			return 0, false
		}
		exe, _, _ = strings.Cut(string(cmd), "\x00")
	}
	exe = strings.TrimSuffix(exe, " (deleted)")
	if want := s.exePath(); want != "" && filepath.Clean(exe) != filepath.Clean(want) {
		return 0, false
	}
	return pid, true
}

func (s initService) running() bool {
	_, ok := s.pid()
	return ok
}

// slackMarker opens the lines the app adds to Slackware's rc.local files.
func (s initService) slackMarker() string {
	return "# " + s.name + ": " + product().Title + " background service (added by the app)"
}

func (s initService) slackLines(action string) string {
	p := "/etc/rc.d/rc." + s.name
	return s.slackMarker() + "\nif [ -x " + p + " ]; then\n  " + p + " " + action + "\nfi\n"
}

// enabled reports whether the service starts with the computer.
func (s initService) enabled() bool {
	switch s.kind {
	case initOpenRC:
		// The run level's entry is a link to the script: looked at itself.
		_, err := os.Lstat(initPath("/etc/runlevels/default/" + s.name))
		return err == nil
	case initSlackware:
		fi, err := os.Stat(s.scriptPath())
		if err != nil || fi.Mode()&0o111 == 0 {
			return false
		}
		b, err := os.ReadFile(initPath("/etc/rc.d/rc.local"))
		return err == nil && strings.Contains(string(b), s.slackMarker())
	default:
		for _, lvl := range []string{"2", "3", "4", "5"} {
			if m, _ := filepath.Glob(initPath("/etc/rc" + lvl + ".d/S*" + s.name)); len(m) > 0 {
				return true
			}
		}
		return false
	}
}

// runTool and haveTool are variables so the tests never run a real
// update-rc.d or rc-update against the test computer's own /etc.
var (
	runTool  = runSystemTool
	haveTool = haveSystemTool
)

// runSystemTool runs a system tool (found also outside the PATH) and returns
// its error with what it printed.
func runSystemTool(name string, args ...string) error {
	cmd := exec.Command(pbscommon.SystemTool(name), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), msg)
	}
	return nil
}

func haveSystemTool(name string) bool { return pbscommon.SystemTool(name) != name }

// install writes the init script for exe and turns it on at boot.
func (s initService) install(exe string) error {
	path := s.scriptPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := atomicWriteFile(path, []byte(s.script(exe)), 0o755); err != nil {
		return fmt.Errorf("could not write %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		return err
	}
	return s.enable()
}

func (s initService) enable() error {
	switch s.kind {
	case initOpenRC:
		return runTool("rc-update", "add", s.name, "default")
	case initSlackware:
		if err := os.Chmod(s.scriptPath(), 0o755); err != nil {
			return err
		}
		if err := addSlackLines(initPath("/etc/rc.d/rc.local"), s.slackMarker(), s.slackLines("start")); err != nil {
			return err
		}
		return addSlackLines(initPath("/etc/rc.d/rc.local_shutdown"), s.slackMarker(), s.slackLines("stop"))
	default:
		switch {
		case haveTool("update-rc.d"):
			return runTool("update-rc.d", s.name, "defaults")
		case haveTool("chkconfig"):
			return runTool("chkconfig", "--add", s.name)
		}
		return s.linkRunlevels()
	}
}

// linkRunlevels starts the service in run levels 2 to 5 and stops it in 0,
// 1 and 6 by hand, where no tool does it.
func (s initService) linkRunlevels() error {
	made := false
	for lvl, prefix := range map[string]string{"0": "K20", "1": "K20", "6": "K20", "2": "S80", "3": "S80", "4": "S80", "5": "S80"} {
		dir := initPath("/etc/rc" + lvl + ".d")
		if !dirExists(dir) {
			continue
		}
		link := filepath.Join(dir, prefix+s.name)
		_ = os.Remove(link)
		if err := os.Symlink("../init.d/"+s.name, link); err != nil {
			return err
		}
		made = true
	}
	if !made {
		return errors.New("no /etc/rcN.d folders to start the service from at boot")
	}
	return nil
}

func (s initService) disable() error {
	switch s.kind {
	case initOpenRC:
		if s.enabled() {
			return runTool("rc-update", "del", s.name, "default")
		}
		return nil
	case initSlackware:
		if err := removeSlackLines(initPath("/etc/rc.d/rc.local"), s.slackMarker()); err != nil {
			return err
		}
		return removeSlackLines(initPath("/etc/rc.d/rc.local_shutdown"), s.slackMarker())
	default:
		switch {
		case haveTool("update-rc.d"):
			return runTool("update-rc.d", "-f", s.name, "remove")
		case haveTool("chkconfig"):
			return runTool("chkconfig", "--del", s.name)
		}
		for _, lvl := range []string{"0", "1", "2", "3", "4", "5", "6"} {
			m, _ := filepath.Glob(initPath("/etc/rc" + lvl + ".d/[SK][0-9][0-9]" + s.name))
			for _, l := range m {
				_ = os.Remove(l)
			}
		}
		return nil
	}
}

// addSlackLines appends lines (opening with marker) to a Slackware rc file,
// once, making the file if there is none.
func addSlackLines(path, marker, lines string) error {
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if strings.Contains(string(b), marker) {
		return nil
	}
	s := string(b)
	if s == "" {
		s = "#!/bin/sh\n"
	}
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	s += "\n" + lines
	if err := atomicWriteFile(path, []byte(s), 0o755); err != nil {
		return fmt.Errorf("could not write %s: %w", path, err)
	}
	return os.Chmod(path, 0o755)
}

// removeSlackLines takes out the lines addSlackLines added.
func removeSlackLines(path, marker string) error {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		if lines[i] == marker {
			// The marker, "if [ -x ... ]; then", the command and "fi".
			i += 3
			if len(out) > 0 && out[len(out)-1] == "" {
				out = out[:len(out)-1]
			}
			continue
		}
		out = append(out, lines[i])
	}
	if len(out) == len(lines) {
		return nil
	}
	if err := atomicWriteFile(path, []byte(strings.Join(out, "\n")), 0o755); err != nil {
		return fmt.Errorf("could not write %s: %w", path, err)
	}
	return os.Chmod(path, 0o755)
}

// start starts the service with its script and waits until it runs.
func (s initService) start() error {
	var err error
	if s.kind == initOpenRC {
		err = runTool("rc-service", s.name, "start")
	} else {
		err = runTool(s.scriptPath(), "start")
	}
	if err != nil {
		return err
	}
	return s.wait(true, 30*time.Second)
}

// stop stops the service with its script and waits until it has stopped.
func (s initService) stop() error {
	var err error
	if s.kind == initOpenRC {
		err = runTool("rc-service", s.name, "stop")
	} else {
		err = runTool(s.scriptPath(), "stop")
	}
	if err != nil {
		return err
	}
	return s.wait(false, 100*time.Second)
}

func (s initService) wait(running bool, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for s.running() != running {
		if time.Now().After(deadline) {
			if running {
				return fmt.Errorf("the service did not start within %s (see /var/log/%s)", timeout, s.name)
			}
			return fmt.Errorf("the service did not stop within %s", timeout)
		}
		time.Sleep(300 * time.Millisecond)
	}
	return nil
}

// remove stops the service, turns it off at boot and deletes its script.
func (s initService) remove() error {
	if s.running() {
		if err := s.stop(); err != nil {
			return err
		}
	}
	if err := s.disable(); err != nil {
		return err
	}
	if err := os.Remove(s.scriptPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// status fills in what the service manager knows about the service.
func (s initService) status(st *ServiceStatus) {
	st.Name = s.name + " (" + s.kind + ")"
	if !s.installed() {
		return
	}
	st.Installed = true
	st.State = "stopped"
	if s.running() {
		st.State = "running"
	}
	st.StartType = "manual"
	if s.enabled() {
		st.StartType = "boot"
	}
	st.ExePath = s.exePath()
	st.OtherExe = st.ExePath != "" && filepath.Clean(st.ExePath) != filepath.Clean(st.ExpectedExe)
}
