//go:build service

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// writePIDFile writes this process's ID to path, for an init script on a
// computer without systemd (initscripts/service.sysv), which finds the
// service by it. Written to a temporary file in the same folder and renamed,
// so the script never reads half a number.
func writePIDFile(path string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("could not write the pid file %s: %w", path, err)
	}
	if _, err := tmp.WriteString(strconv.Itoa(os.Getpid()) + "\n"); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("could not write the pid file %s: %w", path, err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("could not write the pid file %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("could not write the pid file %s: %w", path, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("could not write the pid file %s: %w", path, err)
	}
	return nil
}
