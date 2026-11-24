//go:build linux && !service

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// fakeRoot makes dirs under a temporary folder standing in for the
// computer's root, and records the tools run instead of running them.
func fakeRoot(t *testing.T, dirs ...string) (root string, ran *[]string) {
	t.Helper()
	root = t.TempDir()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var calls []string
	oldRoot, oldRun, oldHave := initRoot, runTool, haveTool
	initRoot = root
	runTool = func(name string, args ...string) error {
		calls = append(calls, strings.TrimSpace(filepath.Base(name)+" "+strings.Join(args, " ")))
		return nil
	}
	haveTool = func(string) bool { return false }
	t.Cleanup(func() { initRoot, runTool, haveTool = oldRoot, oldRun, oldHave })
	return root, &calls
}

func TestDetectInit(t *testing.T) {
	for _, c := range []struct {
		want  string
		dirs  []string
		files []string
	}{
		{initSystemd, []string{"run/systemd/system", "etc/init.d"}, nil},
		{initOpenRC, []string{"run/openrc", "etc/init.d"}, nil},
		{initSlackware, []string{"etc/rc.d", "etc/init.d"}, []string{"etc/slackware-version"}},
		{initSysV, []string{"etc/init.d"}, nil},
		{"", []string{"etc"}, nil},
	} {
		root, _ := fakeRoot(t, c.dirs...)
		for _, f := range c.files {
			if err := os.WriteFile(filepath.Join(root, f), []byte("Slackware 15.0\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if got := detectInit(); got != c.want {
			t.Errorf("dirs %v files %v: detectInit = %q, want %q", c.dirs, c.files, got, c.want)
		}
	}
}

// The scripts name the service program and the service, with nothing left
// to fill in.
func TestInitScripts(t *testing.T) {
	for _, kind := range []string{initSysV, initSlackware, initOpenRC} {
		s := newInitService(kind)
		got := s.script("/usr/libexec/salvanza/salvanza-svc")
		if strings.Contains(got, "@EXE@") || strings.Contains(got, "@NAME@") || strings.Contains(got, "@TITLE@") {
			t.Errorf("%s script still has a placeholder:\n%s", kind, got)
		}
		if !strings.Contains(got, `"/usr/libexec/salvanza/salvanza-svc"`) {
			t.Errorf("%s script does not name the service program", kind)
		}
	}
	if !strings.HasPrefix(newInitService(initOpenRC).script("/x"), "#!/sbin/openrc-run") {
		t.Error("the OpenRC script is not an openrc-run script")
	}
}

// Slackware: the script is rc.<name>, started from rc.local and stopped from
// rc.local_shutdown; removing takes the lines out again and leaves the rest.
func TestSlackwareInstallAndRemove(t *testing.T) {
	root, _ := fakeRoot(t, "etc/rc.d")
	local := filepath.Join(root, "etc/rc.d/rc.local")
	if err := os.WriteFile(local, []byte("#!/bin/sh\n/usr/local/bin/mine\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := newInitService(initSlackware)
	if err := s.install("/usr/libexec/salvanza/salvanza-svc"); err != nil {
		t.Fatal(err)
	}
	if !s.installed() || !s.enabled() {
		t.Fatalf("installed %v enabled %v", s.installed(), s.enabled())
	}
	if s.scriptPath() != filepath.Join(root, "etc/rc.d/rc."+linuxServiceUnit) {
		t.Errorf("script at %s", s.scriptPath())
	}
	if got := s.exePath(); got != "/usr/libexec/salvanza/salvanza-svc" {
		t.Errorf("exePath %q", got)
	}
	// Installing twice adds the lines once.
	if err := s.install("/usr/libexec/salvanza/salvanza-svc"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(local)
	if n := strings.Count(string(b), s.slackMarker()); n != 1 {
		t.Errorf("rc.local has the lines %d times:\n%s", n, b)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "etc/rc.d/rc.local_shutdown")); !strings.Contains(string(b), "rc."+linuxServiceUnit+" stop") {
		t.Errorf("rc.local_shutdown does not stop the service:\n%s", b)
	}
	if err := s.remove(); err != nil {
		t.Fatal(err)
	}
	if s.installed() {
		t.Error("the script is still there")
	}
	if b, _ := os.ReadFile(local); string(b) != "#!/bin/sh\n/usr/local/bin/mine\n" {
		t.Errorf("rc.local after removing:\n%q", b)
	}
}

// SysV without update-rc.d or chkconfig: the run level links by hand.
func TestSysVInstallLinksRunlevels(t *testing.T) {
	root, _ := fakeRoot(t, "etc/init.d", "etc/rc0.d", "etc/rc2.d", "etc/rc3.d", "etc/rc6.d")
	s := newInitService(initSysV)
	if err := s.install("/opt/salvanza/salvanza-svc"); err != nil {
		t.Fatal(err)
	}
	if !s.enabled() {
		t.Fatal("not enabled")
	}
	if target, err := os.Readlink(filepath.Join(root, "etc/rc2.d/S80"+linuxServiceUnit)); err != nil || target != "../init.d/"+linuxServiceUnit {
		t.Errorf("rc2.d link %q %v", target, err)
	}
	if _, err := os.Lstat(filepath.Join(root, "etc/rc0.d/K20"+linuxServiceUnit)); err != nil {
		t.Error("no stop link in rc0.d")
	}
	if err := s.remove(); err != nil {
		t.Fatal(err)
	}
	if s.enabled() || s.installed() {
		t.Error("still enabled or installed after removing")
	}
}

// SysV with update-rc.d, and OpenRC: the tools are asked.
func TestInitToolsAsked(t *testing.T) {
	_, ran := fakeRoot(t, "etc/init.d")
	haveTool = func(name string) bool { return name == "update-rc.d" }
	s := newInitService(initSysV)
	if err := s.install("/x"); err != nil {
		t.Fatal(err)
	}
	if err := s.disable(); err != nil {
		t.Fatal(err)
	}
	want := []string{"update-rc.d " + linuxServiceUnit + " defaults", "update-rc.d -f " + linuxServiceUnit + " remove"}
	if strings.Join(*ran, "|") != strings.Join(want, "|") {
		t.Errorf("ran %q, want %q", *ran, want)
	}

	root, ran := fakeRoot(t, "run/openrc", "etc/init.d", "etc/runlevels/default")
	o := newInitService(initOpenRC)
	if err := o.install("/x"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/init.d/"+linuxServiceUnit, filepath.Join(root, "etc/runlevels/default", linuxServiceUnit)); err != nil {
		t.Fatal(err)
	}
	if !o.enabled() {
		t.Error("OpenRC: not seen as enabled")
	}
	if err := o.disable(); err != nil {
		t.Fatal(err)
	}
	want = []string{"rc-update add " + linuxServiceUnit + " default", "rc-update del " + linuxServiceUnit + " default"}
	if strings.Join(*ran, "|") != strings.Join(want, "|") {
		t.Errorf("ran %q, want %q", *ran, want)
	}
}

// The service is running when its pid file names a live process running the
// program the script starts; a stale pid file, or one naming another
// program, is not.
func TestInitServiceRunning(t *testing.T) {
	root, _ := fakeRoot(t, "run", "etc/init.d", "etc/rc2.d")
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	s := newInitService(initSysV)
	if err := s.install(self); err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(root, "run", linuxServiceUnit+".pid")
	if s.running() {
		t.Error("running with no pid file")
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !s.running() {
		t.Error("not running, though the pid file names this process")
	}
	var st ServiceStatus
	st.ExpectedExe = self
	s.status(&st)
	if !st.Installed || st.State != "running" || st.OtherExe {
		t.Errorf("status %+v", st)
	}
	if err := s.install("/somewhere/else"); err != nil {
		t.Fatal(err)
	}
	if s.running() {
		t.Error("running, though the pid file names another program")
	}
	if err := os.WriteFile(pidFile, []byte("999999999\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if s.running() {
		t.Error("running with a stale pid file")
	}
}
