//go:build linux

package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"pbscommon"
)

// systemd-logind keeps a block lock for as long as the program it starts runs.
// That program is cat on a pipe we hold: closing the pipe ends it, and so does
// the service dying, so a lock cannot outlive the service. Without systemd,
// elogind (Devuan, Slackware) has the same tool as elogind-inhibit.
func platformAcquireSleepLock(reason string) (func(), error) {
	inhibit := ""
	for _, name := range []string{"systemd-inhibit", "elogind-inhibit"} {
		if p := pbscommon.SystemTool(name); p != name {
			inhibit = p
			break
		}
	}
	if inhibit == "" {
		return nil, fmt.Errorf("neither systemd-inhibit nor elogind-inhibit is installed")
	}
	cat, err := exec.LookPath("cat")
	if err != nil {
		return nil, fmt.Errorf("cat is not installed")
	}
	cmd := exec.Command(inhibit, "--what=sleep:idle", "--who="+product().Title, "--why="+reason, "--mode=block", cat) // #nosec G204 -- fixed program names from LookPath, reason is our own text
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	tool := filepath.Base(inhibit)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%s: %w", tool, err)
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	// If logind refuses (no systemd, no permission) systemd-inhibit exits at once.
	select {
	case <-done:
		return nil, fmt.Errorf("%s exited: logind did not grant the lock", tool)
	case <-time.After(300 * time.Millisecond):
	}
	return func() {
		_ = stdin.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}, nil
}
