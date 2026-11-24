package pbscommon

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// An archive cut short anywhere after its first header, whether in the middle
// of an entry or exactly between two entries, is reported as incomplete and
// never listed or restored as if it were whole. (A 1.1.3 backup that met a
// link to a folder stopped part way like this; the restore used to say
// nothing when the cut fell between entries.)
func TestPXARTruncatedArchiveIsIncomplete(t *testing.T) {
	data := standardPXAR(t)
	if _, err := NewPXARReader(data).ListEntries(); err != nil {
		t.Fatalf("whole archive: %v", err)
	}
	for n := 16; n < len(data); n++ {
		_, err := NewPXARReader(data[:n]).ListEntries()
		if !errors.Is(err, ErrArchiveIncomplete) {
			t.Fatalf("archive cut at %d of %d bytes: err = %v, want ErrArchiveIncomplete", n, len(data), err)
		}
	}
}

// A restore of a cut-short archive fails with ErrArchiveIncomplete, after
// writing the files that come before the cut.
func TestPXARTruncatedArchiveRestoreFails(t *testing.T) {
	data := makePXAR(t,
		pxarFile("a.txt", []byte("first\n")),
		pxarFile("b.txt", []byte("second\n")),
	)
	// Cut inside the root folder's closing GOODBYE, after both files.
	dest := t.TempDir()
	_, err := NewPXARReader(data[:len(data)-1]).ExtractAll(dest)
	if !errors.Is(err, ErrArchiveIncomplete) {
		t.Fatalf("ExtractAll of a cut archive: err = %v, want ErrArchiveIncomplete", err)
	}
	for name, want := range map[string]string{"a.txt": "first\n", "b.txt": "second\n"} {
		if got, rerr := os.ReadFile(filepath.Join(dest, name)); rerr != nil || string(got) != want {
			t.Fatalf("%s before the cut: %q, %v", name, got, rerr)
		}
	}
}
