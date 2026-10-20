package main

import (
	"bytes"
	"crypto/sha256"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pbscommon"
)

// Live round trip of the GUI/service code path against a real PBS, run by the
// end-to-end suite (testing/e2e/run.sh, test 11) and skipped otherwise.
//
// The CLI tests of the suite never exercise RunBackupInline / ChunkState, the
// writer the GUI and the service use: it indexed encrypted chunks under the
// bare sha256 (rdemsystems/NimbusBackupClient#13) and nothing caught it. This
// test backs up a folder with that writer and a key file, then reads the
// snapshot back through every GUI read path: catalog, search and restore. The
// suite then restores the same snapshot with the official client.
//
// Environment (set by run.sh): E2E_LIVE=1, PBS_BASEURL, PBS_FINGERPRINT,
// PBS_AUTHID, PBS_TOKEN_SECRET, PBS_DATASTORE, E2E_SRC (folder to back up),
// E2E_KEYFILE (unprotected key file), E2E_OUT (scratch folder), E2E_BACKUP_ID,
// and optionally E2E_PLAIN_ID: an unencrypted backup group, restored with the
// key loaded.
func TestE2ELiveInlineEncrypted(t *testing.T) {
	if os.Getenv("E2E_LIVE") != "1" {
		t.Skip("live end-to-end test: run through testing/e2e/run.sh")
	}
	env := func(name string) string {
		v := os.Getenv(name)
		if v == "" {
			t.Fatalf("%s is not set", name)
		}
		return v
	}
	baseURL, fp := env("PBS_BASEURL"), env("PBS_FINGERPRINT")
	authID, secret, store := env("PBS_AUTHID"), env("PBS_TOKEN_SECRET"), env("PBS_DATASTORE")
	src, out, backupID := env("E2E_SRC"), env("E2E_OUT"), env("E2E_BACKUP_ID")

	keyCfg, err := pbscommon.LoadKeyConfig(env("E2E_KEYFILE"))
	if err != nil {
		t.Fatalf("LoadKeyConfig: %v", err)
	}
	crypt, err := keyCfg.CryptConfig(nil)
	if err != nil {
		t.Fatalf("CryptConfig: %v", err)
	}

	// 1. Encrypted backup with the GUI/service writer.
	var rs *BackupStatus
	err = RunBackupInline(BackupOptions{
		BaseURL: baseURL, AuthID: authID, Secret: secret, Datastore: store, CertFingerprint: fp,
		BackupObjects: []string{src}, BackupID: backupID, BackupType: "host", Kind: "directory",
		Compression: "fastest", Crypt: crypt,
		OnResult: func(s *BackupStatus) { rs = s },
	})
	if err != nil || rs == nil || !rs.Success() {
		t.Fatalf("encrypted backup: %v %+v", err, rs)
	}
	snapTime := time.Unix(rs.BackupTime, 0).UTC()
	t.Logf("encrypted snapshot host/%s/%s", backupID, snapTime.Format(time.RFC3339))

	opts := RestoreOptions{
		BaseURL: baseURL, AuthID: authID, Secret: secret, Datastore: store, CertFingerprint: fp,
		BackupID: backupID, SnapshotTime: snapTime, Crypt: crypt,
		Mode: RestoreModeAlternateAbs, Overwrite: true,
	}

	// 2. Catalog: the browse tree and the search read it. ListSnapshotContents
	// falls back to walking the data archive when the catalog cannot be read,
	// which hid the "no key" failure of #13, so the catalog is read directly.
	entries, _, ok := listSnapshotViaCatalog(opts, nil)
	if !ok {
		t.Fatal("the catalog of the encrypted snapshot could not be read")
	}
	if !hasEntrySuffix(entries, "subdir/nested/deep.txt") {
		t.Errorf("catalog has %d entries but not subdir/nested/deep.txt", len(entries))
	}

	// 3. Search.
	res, err := SearchFilesInline(SearchOptions{
		BaseURL: baseURL, AuthID: authID, Secret: secret, Datastore: store, CertFingerprint: fp,
		HostPrefix: backupID, Query: "deep.txt", Mode: SearchModeName,
		From: snapTime.Add(-time.Hour), To: snapTime.Add(time.Hour),
		AssembleMissing: true, Crypt: crypt,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res.Hits) == 0 || res.SnapshotsSkipped != 0 {
		t.Errorf("search: %d hits, %d snapshots searched, %d skipped; want a hit and no skip",
			len(res.Hits), res.SnapshotsSearched, res.SnapshotsSkipped)
	}

	// 4. Restore, compared with the source.
	opts.DestPath = filepath.Join(out, "restore-gui-enc")
	if err := os.MkdirAll(opts.DestPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RestoreSnapshotInline(opts); err != nil {
		t.Fatalf("restore with the key: %v", err)
	}
	assertTreeRestored(t, src, opts.DestPath, "encrypted snapshot")

	// 5. Without the key the restore must fail, not write ciphertext.
	noKey := opts
	noKey.Crypt = nil
	noKey.DestPath = filepath.Join(out, "restore-gui-enc-nokey")
	if err := os.MkdirAll(noKey.DestPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RestoreSnapshotInline(noKey); err == nil {
		t.Error("restoring the encrypted snapshot without the key succeeded")
	}

	// 6. An unencrypted snapshot restored with the key loaded (the GUI loads
	// the server's key for every read, older plain snapshots included).
	if plainID := os.Getenv("E2E_PLAIN_ID"); plainID != "" {
		snaps, err := ListSnapshotsInline(baseURL, authID, secret, "", "", store, "", fp, plainID)
		if err != nil {
			t.Fatalf("listing %s: %v", plainID, err)
		}
		// The listing matches backup IDs partially (e2e-dir also lists
		// e2e-dir-enc): keep the latest snapshot of exactly plainID.
		var latest *SnapshotInfo
		for i := range snaps {
			if snaps[i].BackupID == plainID && (latest == nil || snaps[i].BackupTime.After(latest.BackupTime)) {
				latest = &snaps[i]
			}
		}
		if latest == nil {
			t.Fatalf("no snapshot of %s among %d listed", plainID, len(snaps))
		}
		plain := opts
		plain.BackupID, plain.SnapshotTime = plainID, latest.BackupTime
		plain.DestPath = filepath.Join(out, "restore-plain-withkey")
		if err := os.MkdirAll(plain.DestPath, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := RestoreSnapshotInline(plain); err != nil {
			t.Fatalf("restoring the unencrypted %s with the key loaded: %v", plainID, err)
		}
		assertTreeRestored(t, src, plain.DestPath, "unencrypted snapshot read with the key loaded")
	}
}

func hasEntrySuffix(entries []SnapshotEntry, suffix string) bool {
	for _, e := range entries {
		if strings.HasSuffix(e.Path, suffix) {
			return true
		}
	}
	return false
}

// assertTreeRestored checks that every file and folder of src is in the restore
// below dest, byte-identical. The restore lands under dest at the archive path,
// so its root is the folder holding src's hello.txt; the metadata file the GUI
// adds at the archive root is the only extra entry allowed.
func assertTreeRestored(t *testing.T, src, dest, label string) {
	t.Helper()
	root := ""
	_ = filepath.WalkDir(dest, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "hello.txt" && root == "" {
			root = filepath.Dir(p)
		}
		return nil
	})
	if root == "" {
		t.Fatalf("%s: hello.txt not found under %s", label, dest)
	}

	want := treeOf(t, src)
	got := treeOf(t, root)
	delete(got, BackupMetaFilename)
	for p, w := range want {
		g, ok := got[p]
		switch {
		case !ok:
			t.Errorf("%s: %s missing from the restore", label, p)
		case !bytes.Equal(g, w):
			t.Errorf("%s: %s differs from the source", label, p)
		}
	}
	for p := range got {
		if _, ok := want[p]; !ok {
			t.Errorf("%s: unexpected %s in the restore", label, p)
		}
	}
	t.Logf("%s: %d entries identical", label, len(want))
}

// treeOf maps every entry below root to the sha256 of its content (files) or
// to nil (folders).
func treeOf(t *testing.T, root string) map[string][]byte {
	t.Helper()
	tree := map[string][]byte{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == root {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			tree[rel] = nil
			return nil
		}
		data, err := os.ReadFile(p) // #nosec G304 -- test fixture paths
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		tree[rel] = sum[:]
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	return tree
}
