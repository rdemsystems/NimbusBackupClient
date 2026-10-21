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

// Live round trip of a job with two folders in one snapshot (issue #15):
// one backup group, one archive per folder, browsed, searched and restored
// by our readers, archive by archive and whole. run.sh (test 12) then
// restores each archive with the official client, from E2E_OUT/archives.txt
// ("<archive> <folder>" per line).
//
// Environment: as TestE2ELiveInlineEncrypted, without E2E_KEYFILE.
func TestE2ELiveOneSnapshot(t *testing.T) {
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

	// A second folder next to the suite's source tree, of several chunks
	// (24 MiB that do not repeat), so that its own previous index is what
	// lets the second run reuse them.
	src2 := filepath.Join(out, "second")
	big := make([]byte, 24<<20)
	for i, x := 0, uint32(2463534242); i < len(big); i++ {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		big[i] = byte(x)
	}
	for name, body := range map[string]string{"notes.txt": "second folder\n", "deeper/inside/file.bin": string(big)} {
		p := filepath.Join(src2, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dirs := []string{src, src2}
	names := ArchiveNames(dirs)

	backup := func() *BackupStatus {
		var rs *BackupStatus
		err := RunBackupInline(BackupOptions{
			BaseURL: baseURL, AuthID: authID, Secret: secret, Datastore: store, CertFingerprint: fp,
			BackupObjects: dirs, BackupID: backupID, BackupType: "host", Kind: "directory",
			Compression: "fastest", OneSnapshot: true,
			OnResult: func(s *BackupStatus) { rs = s },
		})
		if err != nil || rs == nil || !rs.Success() {
			t.Fatalf("one-snapshot backup: %v %+v", err, rs)
		}
		return rs
	}

	// 1. Two runs: the second reuses the first's chunks, archive by archive.
	// Only the first chunk of each archive changes (its metadata file holds
	// the backup time), so a second archive not finding its previous index
	// would send its ~6 chunks again.
	first := backup()
	time.Sleep(time.Second) // snapshots are named by the second
	rs := backup()
	t.Logf("first run %d new / %d reused chunks, second run %d new / %d reused", first.NewChunks, first.ReusedChunks, rs.NewChunks, rs.ReusedChunks)
	if rs.NewChunks > 4 || rs.ReusedChunks < 4 {
		t.Errorf("second run sent %d new chunks and reused %d: an archive did not find its previous index", rs.NewChunks, rs.ReusedChunks)
	}
	snapTime := time.Unix(rs.BackupTime, 0).UTC()
	t.Logf("snapshot host/%s/%s, archives %v", backupID, snapTime.Format(time.RFC3339), names)

	// 2. One group, holding every archive.
	snaps, err := ListSnapshotsInline(baseURL, authID, secret, "", "", store, "", fp, backupID)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	var latest *SnapshotInfo
	for i := range snaps {
		if snaps[i].BackupID != backupID {
			t.Errorf("unexpected backup group %s: the folders were split into groups", snaps[i].BackupID)
		}
		if latest == nil || snaps[i].BackupTime.After(latest.BackupTime) {
			latest = &snaps[i]
		}
	}
	if latest == nil {
		t.Fatal("no snapshot listed")
	}
	for _, n := range names {
		found := false
		for _, f := range latest.Files {
			found = found || f == n
		}
		if !found {
			t.Errorf("snapshot files %v lack %s", latest.Files, n)
		}
	}

	opts := RestoreOptions{
		BaseURL: baseURL, AuthID: authID, Secret: secret, Datastore: store, CertFingerprint: fp,
		BackupID: backupID, SnapshotTime: snapTime, Mode: RestoreModeAlternateAbs, Overwrite: true,
	}

	// 3. The catalog lists each archive under its label, with its origin.
	entries, meta, ok := listSnapshotViaCatalog(opts, nil)
	if !ok || meta == nil || len(meta.Archives) != 2 {
		t.Fatalf("catalog: ok=%v meta=%+v", ok, meta)
	}
	for i, a := range meta.Archives {
		if a.Archive != names[i] || a.OriginalPath != dirs[i] {
			t.Errorf("archive %d: %+v, want %s from %s", i, a, names[i], dirs[i])
		}
	}
	if !hasEntrySuffix(entries, archiveLabel(names[0])+"/subdir/nested/deep.txt") ||
		!hasEntrySuffix(entries, archiveLabel(names[1])+"/deeper/inside/file.bin") {
		t.Errorf("catalog entries lack a folder's files (%d entries)", len(entries))
	}

	// 4. Search finds a file of the second archive, with its real origin.
	res, err := SearchFilesInline(SearchOptions{
		BaseURL: baseURL, AuthID: authID, Secret: secret, Datastore: store, CertFingerprint: fp,
		HostPrefix: backupID, Query: "file.bin", Mode: SearchModeName,
		From: snapTime.Add(-time.Hour), To: snapTime.Add(time.Hour), AssembleMissing: true,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res.Hits) == 0 || res.Hits[0].OriginPath != filepath.Join(src2, "deeper", "inside", "file.bin") {
		t.Errorf("search hits %+v; want file.bin from %s", res.Hits, src2)
	}

	// 5. Whole snapshot: each folder lands in <dest>/<label>.
	opts.DestPath = filepath.Join(out, "restore-multi-all")
	if err := os.MkdirAll(opts.DestPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RestoreSnapshotInline(opts); err != nil {
		t.Fatalf("restore: %v", err)
	}
	for i, d := range dirs {
		assertSameTree(t, d, filepath.Join(opts.DestPath, archiveLabel(names[i])), "whole snapshot, "+names[i])
	}

	// 6. One file of the second archive only.
	one := opts
	one.DestPath = filepath.Join(out, "restore-multi-one")
	if err := os.MkdirAll(one.DestPath, 0o755); err != nil {
		t.Fatal(err)
	}
	one.IncludePaths = []string{archiveLabel(names[1]) + "/notes.txt"}
	if err := RestoreSnapshotInline(one); err != nil {
		t.Fatalf("selective restore: %v", err)
	}
	// Exactly that file, under its archive's folder.
	got := treeOf(t, one.DestPath)
	sum := sha256.Sum256([]byte("second folder\n"))
	want := map[string][]byte{archiveLabel(names[1]): nil, archiveLabel(names[1]) + "/notes.txt": sum[:]}
	if len(got) != len(want) {
		t.Errorf("selective restore wrote %d entries, want %d: %v", len(got), len(want), got)
	}
	for p, w := range want {
		if g, ok := got[p]; !ok || !bytes.Equal(g, w) {
			t.Errorf("selective restore: %s missing or different", p)
		}
	}

	var list strings.Builder
	for i, n := range names {
		list.WriteString(n + " " + dirs[i] + "\n")
	}
	if err := os.WriteFile(filepath.Join(out, "archives.txt"), []byte(list.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// assertSameTree checks that got holds exactly src's files and folders, byte
// for byte, besides the metadata file the GUI adds at an archive's root.
func assertSameTree(t *testing.T, src, got, label string) {
	t.Helper()
	want, have := treeOf(t, src), treeOf(t, got)
	delete(have, BackupMetaFilename)
	for p, w := range want {
		g, ok := have[p]
		switch {
		case !ok:
			t.Errorf("%s: %s missing from the restore", label, p)
		case !bytes.Equal(g, w):
			t.Errorf("%s: %s differs from the source", label, p)
		}
	}
	for p := range have {
		if _, ok := want[p]; !ok {
			t.Errorf("%s: unexpected %s in the restore", label, p)
		}
	}
	t.Logf("%s: %d entries identical", label, len(want))
}
