//go:build !service

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Preferences > Message Log > Keep backup run logs: 52 weeks unless set, 0
// keeps them all, saved and read back by the window; out of range refused.
func TestSetLogRetentionPersists(t *testing.T) {
	strictHome(t)
	noSpeedLimits(t)
	a := &App{config: &Config{PBSServers: map[string]*PBSServer{}}}
	if m := a.GetConfigWithHostname(); m["log_retention_weeks"] != DefaultLogRetentionWeeks {
		t.Errorf("unset: %v, want the default", m["log_retention_weeks"])
	}
	if err := a.SetLogRetention(0); err != nil {
		t.Fatal(err)
	}
	if c := LoadConfig(); c.LogRetentionWeeks == nil || *c.LogRetentionWeeks != 0 {
		t.Errorf("saved: %v, want 0 (keep all)", c.LogRetentionWeeks)
	}
	if m := a.GetConfigWithHostname(); m["log_retention_weeks"] != 0 {
		t.Errorf("keep all reads back as %v", m["log_retention_weeks"])
	}
	if logRetention.Load() != 0 {
		t.Errorf("applied: %d", logRetention.Load())
	}
	for _, bad := range []int{-1, MaxLogRetentionWeeks + 1} {
		if err := a.SetLogRetention(bad); err == nil {
			t.Errorf("SetLogRetention(%d) accepted", bad)
		}
	}
	if err := a.SetLogRetention(13); err != nil {
		t.Fatal(err)
	}
	if c := LoadConfig(); c.LogRetentionWeeks == nil || *c.LogRetentionWeeks != 13 {
		t.Errorf("saved: %v, want 13", c.LogRetentionWeeks)
	}
}

// A hand-edited value out of range counts as the default.
func TestLogRetentionSettingOutOfRange(t *testing.T) {
	for _, w := range []int{-5, MaxLogRetentionWeeks + 1} {
		w := w
		if got := logRetentionSetting(&Config{LogRetentionWeeks: &w}); got != DefaultLogRetentionWeeks {
			t.Errorf("%d in config.json reads as %d", w, got)
		}
	}
}

// Before the settings are read nothing is removed, and with keep all nothing
// is removed either; with a number of weeks the older weeks go.
func TestFoldPrunesByTheSetting(t *testing.T) {
	dir := withLogFolder(t)
	ancient := "backup-logs-" + weekStart(time.Now().AddDate(-3, 0, 0)).Format("20060102") + ".log.gz"
	if err := os.WriteFile(filepath.Join(dir, ancient), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, keep := range []int64{-1, 0} {
		logRetention.Store(keep)
		foldRunLogs(dir, "")
		if _, err := os.Stat(filepath.Join(dir, ancient)); err != nil {
			t.Fatalf("with %d the 3-year-old week was removed", keep)
		}
	}
	logRetention.Store(52)
	foldRunLogs(dir, "")
	if _, err := os.Stat(filepath.Join(dir, ancient)); !os.IsNotExist(err) {
		t.Error("with 52 weeks the 3-year-old week is still there")
	}
}

// The service runs SetLogRetention for the app; an app working through the
// service hands the change to it and takes the setting back.
func TestSetLogRetentionThroughService(t *testing.T) {
	if _, ok := apiCalls["SetLogRetention"]; !ok {
		t.Fatal("SetLogRetention is not on the service's list")
	}
	noSpeedLimits(t)
	svc := strictServiceApp(t)
	if _, err := svc.CallForAPI("SetLogRetention", json.RawMessage(`["4"]`)); err == nil {
		t.Error("SetLogRetention with a string accepted")
	}
	if _, err := svc.CallForAPI("SetLogRetention", json.RawMessage(`[4]`)); err != nil {
		t.Fatal(err)
	}
	if w := svc.config.LogRetentionWeeks; w == nil || *w != 4 {
		t.Errorf("service setting: %v", w)
	}

	p := newStrictPair(t)
	if err := p.app.SetLogRetention(0); err != nil {
		t.Fatal(err)
	}
	p.service(func(s *App) {
		if w := s.config.LogRetentionWeeks; w == nil || *w != 0 {
			t.Errorf("service setting: %v", w)
		}
	})
	if w := p.app.config.LogRetentionWeeks; w == nil || *w != 0 {
		t.Errorf("app's copy: %v", w)
	}
	p.saw(t, "POST /call/SetLogRetention", "GET /config")
}
