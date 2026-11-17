package main

import (
	"bytes"
	"context"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pbscommon"
)

// TestFailedFolderKeepsSnapshotLive is #161 against the isolated test PBS: a
// Backup Set of three folders whose middle one fails part way (as when Windows
// deletes the VSS snapshot under it) must end as a partial backup that keeps
// the other two folders: snapshot finished, both archives restorable, both in
// the catalog, the failed folder in neither. Before the fix PBS refused the
// manifest ("Invalid string length") and removed the whole snapshot.
// Run for the legacy single-stream archive and the split (metadata) one.
// PBS_FAILFOLDER_VSS=1 reads the folders from a VSS snapshot (Windows, as an
// administrator), as the report's Backup Set did.
//
//	set PBS_LIVE_TEST=1 (and the PBS_LIVE_ variables)
//	go test -run TestFailedFolderKeepsSnapshotLive -v .
func TestFailedFolderKeepsSnapshotLive(t *testing.T) {
	if os.Getenv("PBS_LIVE_TEST") == "" {
		t.Skip("set PBS_LIVE_TEST=1 to run this opt-in live integration test")
	}
	liveServer(t)
	for _, mode := range []string{"", pbscommon.ChangeDetectionMetadata} {
		name := mode
		if name == "" {
			name = "legacy"
		}
		t.Run(name, func(t *testing.T) { failedFolderLive(t, mode) })
	}
}

func failedFolderLive(t *testing.T, mode string) {
	root := t.TempDir()
	rng := rand.New(rand.NewSource(161))
	want := map[string][]byte{}
	mk := func(folder string) string {
		dir := filepath.Join(root, folder)
		for _, f := range []struct {
			rel  string
			size int
		}{{"top.bin", 64 << 10}, {"sub/one.bin", 6 << 20}, {"sub/two.bin", 6 << 20}, {"zz/last.bin", 128 << 10}} {
			p := filepath.Join(dir, filepath.FromSlash(f.rel))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			data := make([]byte, f.size)
			rng.Read(data)
			if err := os.WriteFile(p, data, 0o644); err != nil {
				t.Fatal(err)
			}
			want[folder+"/"+f.rel] = data
		}
		return dir
	}
	dirA, dirB, dirC := mk("folderA"), mk("folderB"), mk("folderC")

	// folderB fails part way: after 3 MB of its archive (legacy), or after
	// the first write of its metadata stream (split, where the file data goes
	// to the payload stream and the metadata stream stays small).
	failAt := uint64(3 << 20)
	if mode != "" {
		failAt = 1
	}
	errGone := errors.New("source became unavailable (test)")
	folderFailHook = func(folder string, written uint64) error {
		if folder == dirB && written >= failAt {
			return errGone
		}
		return nil
	}
	t.Cleanup(func() { folderFailHook = nil })

	backupID := "failed-folder-161-" + strings.ReplaceAll(strings.ToLower(t.Name()[strings.LastIndex(t.Name(), "/")+1:]), "_", "-")
	started := time.Now()
	var status *BackupStatus
	err := RunBackupInline(BackupOptions{
		Ctx:                 context.Background(),
		BaseURL:             e2eBaseURL,
		AuthID:              e2eAuthID,
		Secret:              e2eSecret,
		Datastore:           e2eStore,
		CertFingerprint:     e2eFP,
		BackupObjects:       []string{dirA, dirB, dirC},
		BackupID:            backupID,
		BackupType:          "host",
		Kind:                "directory",
		Compression:         "fastest",
		ChangeDetectionMode: mode,
		UseVSS:              os.Getenv("PBS_FAILFOLDER_VSS") != "", // Windows, as an administrator
		OnResult:            func(s *BackupStatus) { status = s },
	})
	if status == nil {
		t.Fatalf("no result (err %v)", err)
	}
	t.Logf("outcome %s: %s (err %v)", status.Outcome, status.Message, err)
	if status.Outcome != OutcomePartial {
		t.Fatalf("outcome %s, want partial: %s", status.Outcome, status.Message)
	}
	for _, d := range status.Directories {
		if (d.Path == dirB) == d.OK {
			t.Errorf("directory %s OK=%v (%s)", d.Path, d.OK, d.Error)
		}
	}

	// The snapshot was finished.
	client := &pbscommon.PBSClient{BaseURL: e2eBaseURL, CertFingerPrint: e2eFP, AuthID: e2eAuthID, Secret: e2eSecret, Datastore: e2eStore, Insecure: true}
	client.Connect(true, "host")
	snaps, err := client.ListSnapshots()
	client.Close()
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	var snapTime int64
	for _, s := range snaps {
		if s.BackupID == backupID && s.BackupTime >= started.Unix()-5 && s.BackupTime > snapTime {
			snapTime = s.BackupTime
		}
	}
	if snapTime == 0 {
		t.Fatalf("no snapshot of %s since %s: the run's snapshot was not kept", backupID, started.Format(time.RFC3339))
	}

	ropts := RestoreOptions{
		Ctx: context.Background(), BaseURL: e2eBaseURL, AuthID: e2eAuthID, Secret: e2eSecret,
		Datastore: e2eStore, CertFingerprint: e2eFP, BackupID: backupID,
		SnapshotTime: time.Unix(snapTime, 0).UTC(),
	}

	// The catalog lists A's and C's files, and nothing of B.
	entries, _, ok := listSnapshotViaCatalog(ropts, nil)
	if !ok {
		t.Fatal("the snapshot's catalog could not be read")
	}
	listed := map[string]bool{}
	for _, e := range entries {
		listed[e.Path] = true
		if strings.Contains(e.Path, "folderB") {
			t.Errorf("catalog lists %s of the failed folder", e.Path)
		}
	}
	for rel := range want {
		if strings.HasPrefix(rel, "folderB/") {
			continue
		}
		found := false
		for p := range listed {
			if strings.HasSuffix(filepath.ToSlash(p), rel) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("catalog lacks %s (%d entries)", rel, len(entries))
		}
	}

	// Everything restores, with the right content, and B is not there.
	dest := t.TempDir()
	ropts.DestPath = dest
	if err := RestoreSnapshotInline(ropts); err != nil {
		t.Fatalf("RestoreSnapshotInline: %v", err)
	}
	restored := map[string]bool{}
	_ = filepath.Walk(dest, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dest, p)
		rel = filepath.ToSlash(rel)
		for key, data := range want {
			if strings.HasSuffix(rel, key) {
				got, rerr := os.ReadFile(p)
				if rerr != nil || !bytes.Equal(got, data) {
					t.Errorf("restored %s differs (read err %v)", rel, rerr)
				}
				restored[key] = true
			}
		}
		if strings.Contains(rel, "folderB") {
			t.Errorf("restored %s of the failed folder", rel)
		}
		return nil
	})
	for key := range want {
		if !strings.HasPrefix(key, "folderB/") && !restored[key] {
			t.Errorf("%s was not restored", key)
		}
	}
}
