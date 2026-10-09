package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigDirs(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want []string
	}{
		{"none", Config{}, nil},
		{"single legacy", Config{BackupSourceDir: "/a"}, []string{"/a"}},
		{"list only", Config{BackupSourceDirs: []string{"/a", "/b"}}, []string{"/a", "/b"}},
		{"both, dedup, order", Config{BackupSourceDir: "/a", BackupSourceDirs: []string{"/b", "/a", ""}}, []string{"/a", "/b"}},
	}
	for _, c := range cases {
		if got := c.cfg.Dirs(); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Dirs() = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDirListFlag(t *testing.T) {
	var f dirListFlag
	for _, v := range []string{"/a", "/b"} {
		if err := f.Set(v); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual([]string(f), []string{"/a", "/b"}) || f.String() != "/a,/b" {
		t.Errorf("dirListFlag = %v", f)
	}
}

func TestConfigDirsWithKeyFile(t *testing.T) {
	raw := `{
		"baseurl": "https://pbs:8007",
		"authid": "user@pbs!tok",
		"secret": "s",
		"datastore": "ds",
		"backupdirs": ["/data/a", "/data/b", "/data/c"],
		"keyfile": "/etc/pbs/key.json",
		"keyfilepassphrase": "hunter2"
	}`
	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	if want := []string{"/data/a", "/data/b", "/data/c"}; !reflect.DeepEqual(cfg.Dirs(), want) {
		t.Errorf("Dirs() = %v, want %v", cfg.Dirs(), want)
	}
	// One key serves every directory: it is a single run-wide setting, not a per-directory one.
	if cfg.KeyFile != "/etc/pbs/key.json" || cfg.KeyFilePassphrase != "hunter2" {
		t.Errorf("key settings lost: %q %q", cfg.KeyFile, cfg.KeyFilePassphrase)
	}
	if !cfg.valid() {
		t.Error("config with several backupdirs and a keyfile should be valid")
	}
}

func TestConfigExcludePatterns(t *testing.T) {
	file := filepath.Join(t.TempDir(), "exclude.txt")
	content := "# caches\n*.tmp\n\n  node_modules  \r\nlogs/*.log\n"
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var cfg Config
	if err := json.Unmarshal([]byte(`{"exclude":["Thumbs.db"],"exclude-from":`+jsonString(file)+`}`), &cfg); err != nil {
		t.Fatal(err)
	}
	got, err := cfg.ExcludePatterns()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Thumbs.db", "*.tmp", "node_modules", "logs/*.log"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExcludePatterns() = %q, want %q", got, want)
	}

	cfg.ExcludeFrom = filepath.Join(t.TempDir(), "missing.txt")
	if _, err := cfg.ExcludePatterns(); err == nil {
		t.Error("a missing exclusion file must be an error, not an unfiltered backup")
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
