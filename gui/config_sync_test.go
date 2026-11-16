package main

// Tests of #158: settings the installer merges into config.json while the
// service runs (-apply-settings) are not written over by the service's older
// copy the next time it changes a setting.

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/salvanza/salvanza_client/gui/api"
)

const syncFingerprint = "AB:CD:EF:01:23:45:67:89:AB:CD:EF:01:23:45:67:89:AB:CD:EF:01:23:45:67:89:AB:CD:EF:01:23:45:67:89"

// installerMerge merges a settings file whose server pbs1 has the token
// secret into the service's folder, as the installer's own program
// (-apply-settings) does: with an App of its own that reads config.json.
func installerMerge(t *testing.T, secret string) {
	t.Helper()
	b := deployBundle(t)
	b.ScheduledJobs = nil
	b.Config.PBSServers["pbs1"].Name = "Office PBS" // an edit needs one
	b.Config.PBSServers["pbs1"].Secret = secret
	merger := &App{config: LoadConfig()}
	if _, _, err := merger.applySettingsFile(writeBundle(t, b)); err != nil {
		t.Fatal(err)
	}
}

// syncedService is the service after an install with token A and a second
// install with token B, behind its API, and a client of it. Requests run
// under the returned lock, and the test reads the service's settings under
// it: the race detector sees no ordering through the loopback socket.
func syncedService(t *testing.T) (*App, *sync.Mutex, *api.Client) {
	t.Helper()
	deployTestDir(t)
	installerMerge(t, "token-A")
	svc := &App{isServiceProcess: true, config: LoadConfig()} // the service starts with A
	installerMerge(t, "token-B")                              // the MSI run again with a changed file
	if got := svc.config.PBSServers["pbs1"].Secret; got != "token-A" {
		t.Fatalf("the service should still hold token-A in memory, holds %q", got)
	}

	tokenPath := filepath.Join(t.TempDir(), "api-token")
	tok, err := api.EnsureToken(tokenPath)
	if err != nil {
		t.Fatal(err)
	}
	mu := &sync.Mutex{}
	h := api.NewServer("127.0.0.1:0", svc, tok, "test").Handler()
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		h.ServeHTTP(w, r)
	}))
	t.Cleanup(hs.Close)
	return svc, mu, api.NewClientAt(hs.URL, tokenPath)
}

// onDisk is server pbs1 as config.json holds it.
func onDisk(t *testing.T) *PBSServer {
	t.Helper()
	s, err := LoadConfig().GetPBSServer("pbs1")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// The case of the report: a fingerprint pinned through the service's API
// (POST /pbs/fingerprint, as the app's connection test does) after the merge.
func TestInstallerMergeSurvivesFingerprintPin(t *testing.T) {
	svc, mu, client := syncedService(t)
	if err := client.PinFingerprint("pbs1", syncFingerprint); err != nil {
		t.Fatal(err)
	}
	s := onDisk(t)
	if s.Secret != "token-B" {
		t.Errorf("config.json holds token %q after the pin, want the merged token-B", s.Secret)
	}
	if s.CertFingerprint != syncFingerprint {
		t.Errorf("config.json holds fingerprint %q, want %q", s.CertFingerprint, syncFingerprint)
	}
	mu.Lock()
	defer mu.Unlock()
	if got := svc.config.PBSServers["pbs1"].Secret; got != "token-B" {
		t.Errorf("the service holds token %q, want token-B", got)
	}
}

// Every other change through the service's API: a setting (POST /call/...).
func TestInstallerMergeSurvivesSettingCall(t *testing.T) {
	_, _, client := syncedService(t)
	if err := client.Call("SetSMTPSettings", nil, "mail.example", "587", "", "", false, "backup@example"); err != nil {
		t.Fatal(err)
	}
	if s := onDisk(t); s.Secret != "token-B" {
		t.Errorf("config.json holds token %q after the e-mail account was set, want token-B", s.Secret)
	}
	if got := LoadConfig().SMTPHost; got != "mail.example" {
		t.Errorf("e-mail server on disk is %q, want mail.example", got)
	}
}

// changedOnDisk: not after this program's own load or save, yes after
// another program's write, and never for settings built in memory.
func TestConfigChangedOnDisk(t *testing.T) {
	deployTestDir(t)
	c := LoadConfig() // no file yet
	if c.changedOnDisk() {
		t.Error("no file, nothing written: reported changed")
	}
	c.SMTPHost = "mail.one"
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	if c.changedOnDisk() {
		t.Error("reported changed after its own save")
	}
	other := LoadConfig()
	if other.changedOnDisk() {
		t.Error("reported changed right after loading")
	}
	other.SMTPHost = "mail.two"
	if err := other.Save(); err != nil {
		t.Fatal(err)
	}
	if !c.changedOnDisk() {
		t.Error("not reported changed after another writer saved")
	}

	if (&Config{}).changedOnDisk() {
		t.Error("settings built in memory reported changed")
	}

	// A file that has gone is not a change to reload: the settings in memory
	// would be lost.
	path, err := getConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if other.changedOnDisk() {
		t.Error("a removed file reported as changed")
	}
}

// SyncConfigFromDisk reloads only when the file changed, and the reloaded
// settings are the merged ones.
func TestSyncConfigFromDisk(t *testing.T) {
	deployTestDir(t)
	installerMerge(t, "token-A")
	svc := &App{isServiceProcess: true, config: LoadConfig()}
	if svc.SyncConfigFromDisk() {
		t.Error("reloaded although nothing changed")
	}
	installerMerge(t, "token-B")
	if !svc.SyncConfigFromDisk() {
		t.Fatal("did not reload after the merge")
	}
	if got := svc.config.PBSServers["pbs1"].Secret; got != "token-B" {
		t.Errorf("reloaded token %q, want token-B", got)
	}
	if svc.SyncConfigFromDisk() {
		t.Error("reloaded a second time with nothing new")
	}

	// Settings built in memory (the app's copy of the service's, tests) are
	// left alone.
	mem := &App{isServiceProcess: true, config: &Config{SMTPHost: "kept"}}
	if mem.SyncConfigFromDisk() || mem.config.SMTPHost != "kept" {
		t.Error("settings built in memory were replaced")
	}

	// Where secrets are sealed, the reload went through the sealed file: the
	// token is not in it as plain text, yet the service has it.
	path, _ := getConfigPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if canSealSecrets && strings.Contains(string(raw), "token-B") {
		t.Error("config.json holds the token in plain text")
	}
}
