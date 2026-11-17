package pbscommon

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The snapshot list keeps only the manifest's files. PBS adds any other file
// of the snapshot's folder with no crypt-mode and no size, such as the
// closed index of a folder given up part way (#161); restore must not take
// it for an archive.
func TestListSnapshotsKeepsManifestFilesOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/admin/datastore/store/snapshots") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"backup-type":"host","backup-id":"pc","backup-time":1700000000,"files":[` +
			`{"filename":"a.pxar.didx","crypt-mode":"none","size":10},` +
			`{"filename":"index.json.blob","crypt-mode":"none","size":2},` +
			`{"filename":"b.mpxar.didx"},{"filename":"b.ppxar.didx"}]}]}`))
	}))
	defer srv.Close()
	c := &PBSClient{BaseURL: srv.URL, Datastore: "store"}
	snaps, err := c.ListSnapshots()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range snaps[0].Files {
		names = append(names, f.Filename)
	}
	if got := strings.Join(names, " "); got != "a.pxar.didx index.json.blob" {
		t.Fatalf("files = %s", got)
	}
}

// testCsum is a checksum of the right shape (64 hex digits) for manifests
// built by hand in tests.
var testCsum = strings.Repeat("ab", 32)

// An archive that was given up on leaves the manifest, and the other
// writers still find their own entries afterwards (#161).
func TestDropManifestFile(t *testing.T) {
	c := &PBSClient{WritersManifest: map[uint64]int{}}
	for i, name := range []string{"a.pxar.didx", "b.ppxar.didx", "b.mpxar.didx", "c.pxar.didx"} {
		c.Manifest.Files = append(c.Manifest.Files, File{Filename: name})
		c.WritersManifest[uint64(i+1)] = i
	}

	c.DropManifestFile(2)
	c.DropManifestFile(3)
	c.DropManifestFile(9) // unknown: nothing happens

	var names []string
	for _, f := range c.Manifest.Files {
		names = append(names, f.Filename)
	}
	if got := strings.Join(names, " "); got != "a.pxar.didx c.pxar.didx" {
		t.Fatalf("files = %s", got)
	}
	if _, ok := c.WritersManifest[2]; ok {
		t.Error("writer 2 still mapped")
	}
	for w, want := range map[uint64]string{1: "a.pxar.didx", 4: "c.pxar.didx"} {
		if got := c.Manifest.Files[c.WritersManifest[w]].Filename; got != want {
			t.Errorf("writer %d -> %s, want %s", w, got, want)
		}
	}
}

// A manifest listing an archive that was never closed is not sent: PBS would
// refuse it with "Invalid string length" and remove the whole snapshot.
func TestUploadManifestRefusesUnclosedArchive(t *testing.T) {
	srv, captured := newFakePBSServer(t)
	client := newTestClient(srv.URL, nil)
	client.Manifest.BackupType = "host"
	client.Manifest.BackupID = "unclosed"
	client.Manifest.Files = []File{
		{CryptMode: CryptModeNone, Filename: "a.pxar.didx", Csum: testCsum},
		{CryptMode: CryptModeNone, Filename: "b.pxar.didx"},
	}
	err := client.UploadManifest()
	if err == nil || !strings.Contains(err.Error(), "b.pxar.didx was never closed") {
		t.Fatalf("UploadManifest = %v, want the unclosed archive named", err)
	}
	if len(*captured) != 0 {
		t.Fatalf("%d requests sent, want none", len(*captured))
	}
}
