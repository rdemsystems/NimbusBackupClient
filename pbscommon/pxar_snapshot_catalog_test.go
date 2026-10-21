package pbscommon

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// makeTree creates the given files (slash paths) under a new temp dir.
func makeTree(t *testing.T, files ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, f := range files {
		p := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("content of "+f), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func catalogPaths(entries []CatalogEntry) []string {
	var out []string
	for _, e := range entries {
		out = append(out, e.Path)
	}
	sort.Strings(out)
	return out
}

// One archive written through a SnapshotCatalog gives byte for byte the
// catalog the single-archive writer produces.
func TestSnapshotCatalogSingleArchiveMatchesLegacy(t *testing.T) {
	root := makeTree(t, "top.txt", "a/b/one.txt", "a/c/two.txt")
	_, legacy := writeTestArchive(t, root, false)

	var c bytes.Buffer
	cat := &SnapshotCatalog{WriteCB: func(b []byte) error { c.Write(b); return nil }}
	archive := &PXARArchive{ArchiveName: "backup.pxar.didx", Catalog: cat}
	archive.WriteCB = func([]byte) error { return nil }
	archive.Create()
	if _, err := archive.WriteDir(root, "", true); err != nil {
		t.Fatalf("WriteDir: %v", err)
	}
	if err := cat.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !bytes.Equal(c.Bytes(), legacy) {
		t.Errorf("shared catalog differs from the single-archive one (%d vs %d bytes)", c.Len(), len(legacy))
	}
}

// Several archives share one catalog whose root table lists each of them,
// and each archive's entries are relative to its own root.
func TestSnapshotCatalogSeveralArchives(t *testing.T) {
	roots := []struct {
		name  string
		dir   string
		paths []string
	}{
		{"Downloads.pxar.didx", makeTree(t, "x.iso", "sub/y.txt"), []string{"sub", "sub/y.txt", "x.iso"}},
		{"smart.pxar.didx", makeTree(t, "z.txt"), []string{"z.txt"}},
	}

	var c bytes.Buffer
	cat := &SnapshotCatalog{WriteCB: func(b []byte) error { c.Write(b); return nil }}
	for _, r := range roots {
		var p bytes.Buffer
		archive := &PXARArchive{ArchiveName: r.name, Catalog: cat}
		archive.WriteCB = func(b []byte) error { p.Write(b); return nil }
		archive.Create()
		if _, err := archive.WriteDir(r.dir, "", true); err != nil {
			t.Fatalf("WriteDir %s: %v", r.name, err)
		}
	}
	if err := cat.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	archives, err := ParseCatalogArchives(c.Bytes())
	if err != nil {
		t.Fatalf("ParseCatalogArchives: %v", err)
	}
	if len(archives) != len(roots) {
		t.Fatalf("got %d archives, want %d", len(archives), len(roots))
	}
	for i, r := range roots {
		if archives[i].Name != r.name {
			t.Errorf("archive %d is %q, want %q", i, archives[i].Name, r.name)
		}
		got := catalogPaths(archives[i].Entries)
		if len(got) != len(r.paths) {
			t.Errorf("%s: entries %v, want %v", r.name, got, r.paths)
			continue
		}
		for j := range got {
			if got[j] != r.paths[j] {
				t.Errorf("%s: entries %v, want %v", r.name, got, r.paths)
				break
			}
		}
	}

	if _, err := ParseCatalog(c.Bytes()); err == nil {
		t.Error("ParseCatalog accepted a catalog of two archives")
	}
}

// A snapshot catalog with no archive written is still a valid, empty one.
func TestSnapshotCatalogEmpty(t *testing.T) {
	var c bytes.Buffer
	cat := &SnapshotCatalog{WriteCB: func(b []byte) error { c.Write(b); return nil }}
	if err := cat.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !bytes.HasPrefix(c.Bytes(), catalog_magic) {
		t.Fatal("empty catalog lacks the magic")
	}
	if _, err := ParseCatalogArchives(c.Bytes()); err == nil {
		t.Error("an empty catalog parsed as having archives")
	}
}
