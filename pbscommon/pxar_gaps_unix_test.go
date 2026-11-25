//go:build unix

package pbscommon

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

// Known gaps of backup and restore (testing/restore-matrix/README.md, "Known
// gaps"). Each subtest shows whether one gap is still there. It passes
// either way: it is a record, not a check. When a gap is closed, its
// subtest says so; the case then becomes a check in the matrix and leaves
// this list.

const envGapReport = "SALVANZA_GAP_REPORT"

func reportGap(t *testing.T, id, title string, present bool, detail string) {
	t.Helper()
	state := "🟠 present"
	if !present {
		state = "🟢 closed: make it a check in the matrix"
	}
	line := fmt.Sprintf("| %s | %s | %s | %s |", id, title, state, detail)
	t.Log("GAP| " + line)
	if path := os.Getenv(envGapReport); path != "" && !isPermChild() {
		if f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			_, _ = fmt.Fprintln(f, line)
			_ = f.Close()
		}
	}
}

func TestRestoreGaps(t *testing.T) {
	// GAP-1: richard is uid 1000 on a laptop and 1512 on a desktop. A
	// root restore of their home puts back uid 1000, the archive's: on the
	// desktop they cannot read their own ~/.ssh, and uid 1000 may be someone
	// else there. There is no way to restore as another owner or to map one
	// uid to another. (A restore run as uid 1512 is fine: it owns all it
	// restores; the matrix checks that.)
	t.Run("GAP-1 owner mapping", func(t *testing.T) {
		if os.Geteuid() != 0 {
			t.Skip("needs root")
		}
		src := t.TempDir()
		ssh := filepath.Join(src, "home", ".ssh")
		if err := os.MkdirAll(ssh, 0o700); err != nil {
			t.Fatal(err)
		}
		key := filepath.Join(ssh, "id_ed25519")
		if err := os.WriteFile(key, []byte("key"), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, p := range []string{key, ssh, filepath.Join(src, "home")} {
			if err := os.Lchown(p, 1000, 1000); err != nil {
				t.Fatal(err)
			}
		}
		pxar, _, _ := writePosixArchive(t, src)
		dest := t.TempDir()
		pr := NewPXARReader(pxar)
		pr.SetRestoreOwnership(true)
		res, err := pr.ExtractAll(dest)
		if err != nil {
			t.Fatal(err)
		}
		FinishDirs(res, true)
		fi, err := os.Lstat(filepath.Join(dest, "home", ".ssh", "id_ed25519"))
		if err != nil {
			t.Fatal(err)
		}
		uid := fi.Sys().(*syscall.Stat_t).Uid
		reportGap(t, "GAP-1", "Restore onto a computer where the user's uid differs", uid == 1000,
			fmt.Sprintf("a root restore gives richard's home to uid %d (the archive's), not to their uid on this computer (1512); no option to choose the owner or map uids", uid))
	})

	// GAP-3: two names of one file (a hard link) are backed up as two
	// files and restored as two files: the link is lost and the data is
	// stored twice. pxar has a hard link entry; the writer does not use it.
	t.Run("GAP-3 hard links", func(t *testing.T) {
		src := t.TempDir()
		a, b := filepath.Join(src, "a"), filepath.Join(src, "b")
		if err := os.WriteFile(a, []byte("same file"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(a, b); err != nil {
			t.Fatal(err)
		}
		pxar, _, _ := writePosixArchive(t, src)
		dest := t.TempDir()
		if _, err := NewPXARReader(pxar).ExtractAll(dest); err != nil {
			t.Fatal(err)
		}
		fa, errA := os.Stat(filepath.Join(dest, "a"))
		fb, errB := os.Stat(filepath.Join(dest, "b"))
		if errA != nil || errB != nil {
			t.Fatalf("restore: %v %v", errA, errB)
		}
		reportGap(t, "GAP-3", "Hard links", !os.SameFile(fa, fb),
			"two names of one file come back as two separate files (the data is also stored twice)")
	})

	// GAP-4: FIFOs and sockets (and, as root, device nodes) are skipped:
	// reading them would hang the backup, but the official client records
	// them as entries without data and restores them.
	t.Run("GAP-4 special files", func(t *testing.T) {
		src := t.TempDir()
		if err := unix.Mkfifo(filepath.Join(src, "fifo"), 0o644); err != nil {
			t.Fatal(err)
		}
		// A socket path must be short (104 bytes on the BSDs and macOS).
		sockDir, err := os.MkdirTemp("/tmp", "gap-")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(sockDir) }()
		l, err := net.Listen("unix", filepath.Join(sockDir, "s"))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = l.Close() }()
		if err := os.Rename(filepath.Join(sockDir, "s"), filepath.Join(src, "socket")); err != nil {
			t.Logf("socket not moved into the tree (%v): FIFO only", err)
		}
		pxar, _, _ := writePosixArchive(t, src)
		entries, err := NewPXARReader(pxar).ListEntries()
		if err != nil {
			t.Fatal(err)
		}
		kept := map[string]bool{}
		for _, e := range entries {
			kept[e.Path] = true
		}
		reportGap(t, "GAP-4", "FIFOs, sockets and device nodes", !kept["fifo"],
			fmt.Sprintf("not in the archive (FIFO kept: %v, socket kept: %v); the official client keeps them as entries without data", kept["fifo"], kept["socket"]))
	})
}
