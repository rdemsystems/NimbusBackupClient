package main

// #161: a folder that fails part way in a multi-folder Backup Set must not
// take the snapshot down with it. Its indexes are closed on PBS with what PBS
// was given of them and left out of the manifest, and the shared catalog
// moves past the tables it wrote, so the other folders' archives and catalog
// stay good.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"pbscommon"
)

// closeRecorder is a PBS that takes chunk uploads and assignments, answers
// /dynamic_close with status, and records what it was sent.
type closeRecorder struct {
	mu      sync.Mutex
	status  int
	got     []pbscommon.IndexCloseReq
	uploads []string // digests
	assigns []pbscommon.IndexPutReq
}

func newCloseRecorder(t *testing.T, status int) (*closeRecorder, *pbscommon.PBSClient) {
	t.Helper()
	r := &closeRecorder{status: status}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/dynamic_chunk":
			r.mu.Lock()
			r.uploads = append(r.uploads, req.URL.Query().Get("digest"))
			r.mu.Unlock()
			_, _ = w.Write([]byte(`{"data":null}`))
			return
		case req.Method == http.MethodPut && req.URL.Path == "/dynamic_index":
			var p pbscommon.IndexPutReq
			_ = json.NewDecoder(req.Body).Decode(&p)
			r.mu.Lock()
			r.assigns = append(r.assigns, p)
			r.mu.Unlock()
			_, _ = w.Write([]byte(`{"data":null}`))
			return
		case req.URL.Path != "/dynamic_close":
			http.NotFound(w, req)
			return
		}
		var c pbscommon.IndexCloseReq
		_ = json.NewDecoder(req.Body).Decode(&c)
		r.mu.Lock()
		r.got = append(r.got, c)
		r.mu.Unlock()
		if r.status != http.StatusOK {
			http.Error(w, "dynamic writer close failed", r.status)
			return
		}
		_, _ = w.Write([]byte(`{"data":null}`))
	}))
	t.Cleanup(srv.Close)
	client := &pbscommon.PBSClient{BaseURL: srv.URL, Client: *srv.Client(), WritersManifest: map[uint64]int{}}
	for i, name := range []string{"a.pxar.didx", "b.pxar.didx", "c.pxar.didx"} {
		client.Manifest.Files = append(client.Manifest.Files, pbscommon.File{Filename: name})
		client.WritersManifest[uint64(i+1)] = i
	}
	return r, client
}

func entriesOf(sizes ...uint64) []*chunkEntry {
	var out []*chunkEntry
	var pos uint64
	for i, s := range sizes {
		e := &chunkEntry{offset: pos, size: s}
		e.digest = sha256.Sum256([]byte{byte(i)})
		out = append(out, e)
		pos += s
	}
	return out
}

func manifestNames(c *pbscommon.PBSClient) string {
	var n []string
	for _, f := range c.Manifest.Files {
		n = append(n, f.Filename)
	}
	return strings.Join(n, " ")
}

// Nothing assigned yet (the usual case: chunks are assigned only at EOF):
// PBS closes no index with no chunks, so a one-byte placeholder chunk is
// uploaded and assigned first, the index closed with it, and it leaves the
// manifest.
func TestAbandonClosesEmptyIndex(t *testing.T) {
	rec, client := newCloseRecorder(t, http.StatusOK)
	c := &ChunkState{wrid: 2, open: true, entries: entriesOf(100, 200, 300)}
	if err := c.abandon(client); err != nil {
		t.Fatalf("abandon: %v", err)
	}
	d := sha256.Sum256([]byte{0})
	dh := hex.EncodeToString(d[:])
	if len(rec.uploads) != 1 || rec.uploads[0] != dh {
		t.Fatalf("uploads = %v, want the placeholder %s", rec.uploads, dh)
	}
	if len(rec.assigns) != 1 || rec.assigns[0].WriterID != 2 || len(rec.assigns[0].DigestList) != 1 ||
		rec.assigns[0].DigestList[0] != dh || rec.assigns[0].OffsetList[0] != 0 {
		t.Fatalf("assigns = %+v", rec.assigns)
	}
	// PBS's checksum: SHA-256 over each chunk's end offset (u64 LE) and digest.
	h := sha256.New()
	h.Write([]byte{1, 0, 0, 0, 0, 0, 0, 0})
	h.Write(d[:])
	want := pbscommon.IndexCloseReq{WriterID: 2, ChunkCount: 1, Size: 1, CheckSum: hex.EncodeToString(h.Sum(nil))}
	if len(rec.got) != 1 || rec.got[0] != want {
		t.Fatalf("close requests = %+v, want [%+v]", rec.got, want)
	}
	if got := manifestNames(client); got != "a.pxar.didx c.pxar.didx" {
		t.Fatalf("manifest = %s", got)
	}
	if c.open || !c.closed {
		t.Errorf("open=%v closed=%v after abandon", c.open, c.closed)
	}
	// A second call does nothing.
	if err := c.abandon(client); err != nil || len(rec.got) != 1 {
		t.Errorf("second abandon: err %v, %d requests", err, len(rec.got))
	}
}

// Some batches assigned before the folder failed: the close gives exactly
// what PBS holds, as PBS checks count, size and checksum.
func TestAbandonClosesWithAssignedEntries(t *testing.T) {
	rec, client := newCloseRecorder(t, http.StatusOK)
	c := &ChunkState{wrid: 3, open: true, entries: entriesOf(100, 200, 300), assigned: 2}
	if err := c.abandon(client); err != nil {
		t.Fatalf("abandon: %v", err)
	}
	_, _, csum := c.indexListN(2)
	want := pbscommon.IndexCloseReq{WriterID: 3, ChunkCount: 2, Size: 300, CheckSum: csum}
	if len(rec.got) != 1 || rec.got[0] != want {
		t.Fatalf("close requests = %+v, want [%+v]", rec.got, want)
	}
	if got := manifestNames(client); got != "a.pxar.didx b.pxar.didx" {
		t.Fatalf("manifest = %s", got)
	}
}

// Closed already (the folder failed after its EOF): not closed again, only
// left out of the manifest.
func TestAbandonClosedIndexOnlyLeavesManifest(t *testing.T) {
	rec, client := newCloseRecorder(t, http.StatusOK)
	c := &ChunkState{wrid: 1, open: true, closed: true}
	if err := c.abandon(client); err != nil {
		t.Fatalf("abandon: %v", err)
	}
	if len(rec.got) != 0 {
		t.Fatalf("%d close requests, want none", len(rec.got))
	}
	if got := manifestNames(client); got != "b.pxar.didx c.pxar.didx" {
		t.Fatalf("manifest = %s", got)
	}
}

// PBS refuses the close: the error comes back and the entry stays, so the
// run says so instead of sending a manifest PBS would refuse.
func TestAbandonRefusedClose(t *testing.T) {
	_, client := newCloseRecorder(t, http.StatusBadRequest)
	c := &ChunkState{wrid: 2, open: true}
	if err := c.abandon(client); err == nil {
		t.Fatal("abandon succeeded on a refused close")
	}
	if got := manifestNames(client); got != "a.pxar.didx b.pxar.didx c.pxar.didx" {
		t.Fatalf("manifest = %s", got)
	}
	leftOpen := &archiveLeftOpenError{cause: errors.New("source went away"), close: errors.New("HTTP 400")}
	if !strings.Contains(leftOpen.Error(), "no snapshot can be kept") || !errors.Is(leftOpen, leftOpen.cause) {
		t.Errorf("archiveLeftOpenError = %q", leftOpen.Error())
	}
}

// Never created: nothing to do.
func TestAbandonNotOpen(t *testing.T) {
	rec, client := newCloseRecorder(t, http.StatusOK)
	if err := (&ChunkState{}).abandon(client); err != nil || len(rec.got) != 0 {
		t.Fatalf("abandon of an index never opened: err %v, %d requests", err, len(rec.got))
	}
}

// catalogRun writes folders into one shared catalog stream as backupReal
// does, with the folders in fail failing once failAfter bytes of their
// archive are written, and returns the finished catalog.
func catalogRun(t *testing.T, folders []string, fail map[string]bool, failAfter int, recordThenFail string) []byte {
	t.Helper()
	var stream bytes.Buffer
	s := &sharedCatalogCoordinator{}
	errGone := errors.New("source became unavailable")
	for _, dir := range folders {
		a := &pbscommon.PXARArchive{ArchiveName: filepath.Base(dir) + ".pxar.didx"}
		written := 0
		a.WriteCB = func(b []byte) error {
			if fail[dir] && written+len(b) > failAfter {
				return errGone
			}
			written += len(b)
			return nil
		}
		s.prepare(a, nil)
		a.CatalogWriteCB = func(b []byte) error { _, err := stream.Write(b); return err }
		before := stream.Len()
		cd, err := a.WriteDir(dir, "", true)
		if fail[dir] {
			if err == nil {
				t.Fatalf("%s did not fail", dir)
			}
			if stream.Len() == before {
				t.Fatalf("%s failed before writing any catalog table: the test proves nothing", dir)
			}
			s.abandon(a)
			continue
		}
		if err != nil {
			t.Fatalf("WriteDir %s: %v", dir, err)
		}
		s.record(cd, a)
		if dir == recordThenFail {
			s.abandon(a) // failed after its walk (0 bytes, or its index did not close)
		}
	}
	if uint64(stream.Len()) != s.pos {
		t.Fatalf("catalog position %d, stream has %d bytes", s.pos, stream.Len())
	}
	if err := pbscommon.WriteSharedCatalogRoot(func(b []byte) error { _, err := stream.Write(b); return err }, s.pos, s.entries); err != nil {
		t.Fatal(err)
	}
	return stream.Bytes()
}

// makeFolder makes name with two subfolders: s1 with small files, whose
// catalog table is written first, and s2 with a file large enough to fail on.
func makeFolder(t *testing.T, root, name string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	for _, f := range []struct {
		path string
		size int
	}{
		{"s1/one.txt", 10}, {"s1/two.txt", 20},
		{"s2/big.bin", 3 << 20},
		{"top.txt", 30},
	} {
		p := filepath.Join(dir, filepath.FromSlash(f.path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, bytes.Repeat([]byte{'x'}, f.size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func catalogPaths(t *testing.T, cat []byte) map[string]bool {
	t.Helper()
	entries, err := pbscommon.ParseCatalog(cat)
	if err != nil {
		t.Fatalf("ParseCatalog: %v", err)
	}
	got := map[string]bool{}
	for _, e := range entries {
		got[e.Path] = true
	}
	return got
}

func checkCatalog(t *testing.T, got map[string]bool, kept, gone []string) {
	t.Helper()
	defer func() {
		if t.Failed() {
			t.Logf("catalog paths: %v", got)
		}
	}()
	for _, k := range kept {
		a := k + ".pxar.didx"
		for _, p := range []string{a, a + "/top.txt", a + "/s1/one.txt", a + "/s2/big.bin"} {
			if !got[p] {
				t.Errorf("catalog lacks %s", p)
			}
		}
	}
	for _, g := range gone {
		for p := range got {
			if strings.HasPrefix(p, g+".pxar") {
				t.Errorf("catalog lists %s of the failed folder", p)
			}
		}
	}
}

// The middle folder fails after writing catalog tables: the others are
// found in the catalog, the failed one is not.
func TestSharedCatalogSkipsFailedMiddleFolder(t *testing.T) {
	root := t.TempDir()
	a, b, c := makeFolder(t, root, "fa"), makeFolder(t, root, "fb"), makeFolder(t, root, "fc")
	cat := catalogRun(t, []string{a, b, c}, map[string]bool{b: true}, 1<<20, "")
	checkCatalog(t, catalogPaths(t, cat), []string{"fa", "fc"}, []string{"fb"})
}

// The first folder fails after writing the magic header and some tables:
// the next one does not write the header again.
func TestSharedCatalogSkipsFailedFirstFolder(t *testing.T) {
	root := t.TempDir()
	a, b := makeFolder(t, root, "fa"), makeFolder(t, root, "fb")
	cat := catalogRun(t, []string{a, b}, map[string]bool{a: true}, 1<<20, "")
	checkCatalog(t, catalogPaths(t, cat), []string{"fb"}, []string{"fa"})
}

// A folder that failed after its walk was recorded (0 bytes, or its index
// would not close) is taken out of the catalog's root again.
func TestSharedCatalogDropsRecordedFailedFolder(t *testing.T) {
	root := t.TempDir()
	a, b, c := makeFolder(t, root, "fa"), makeFolder(t, root, "fb"), makeFolder(t, root, "fc")
	cat := catalogRun(t, []string{a, b, c}, nil, 0, b)
	checkCatalog(t, catalogPaths(t, cat), []string{"fa", "fc"}, []string{"fb"})
}
