//go:build unix

package pbscommon

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// The backup and restore matrix (testing/restore-matrix/README.md).
//
//   - what is backed up: files and folders of every mode (plain, read-only,
//     private, setuid, setgid, sticky), symbolic links, and a user's home
//     (~/.ssh and the like), owned by root, a system account (uid 1) and a
//     user (uid 1000, "richard") when root makes the backup;
//   - who restores: root, uid 1, uid 1000 (the same user) and uid 1512
//     (richard on another computer, where their uid is not the same);
//   - where: on the system that made the backup, and on every other one
//     (an archive made on FreeBSD restored on Linux, and so on), with our
//     reader and, on Linux, with the official pxar tool.
//
// Each restored entry is compared with what the file system recorded at
// backup time: content, link target, mode, owner and modification time. The
// expectations, by who restores:
//
//   - root: everything comes back as recorded.
//   - anyone else: the permission bits come back, setuid, setgid and sticky
//     dropped (as tar does); a folder is never made more open than at its
//     creation (0755 under the usual umask); everything belongs to the user
//     who restores, whatever the archive recorded.
//
// Without the environment of testing/restore-matrix/run.sh, TestPermMatrix
// runs the matrix of the current system as the current user and, as root,
// as the other users too; the other tests skip.

const (
	envPermChild     = "SALVANZA_PERM_CHILD"     // set in a rerun as another user
	envPermArchives  = "SALVANZA_PERM_ARCHIVES"  // folder of <system>.pxar + .json to restore
	envPermProduce   = "SALVANZA_PERM_PRODUCE"   // folder to write this system's archive to
	envPermExtracted = "SALVANZA_PERM_EXTRACTED" // a tree another tool restored as root
	envPermManifest  = "SALVANZA_PERM_MANIFEST"  // the manifest of that tree's archive
	envPermTool      = "SALVANZA_PERM_TOOL"      // the name of that tool, for the report
	envPermReport    = "SALVANZA_PERM_REPORT"    // file the summary lines are added to
	permReportTag    = "REPORT| "
)

type permUser struct {
	name string
	uid  int
}

// Who restores, besides root.
var permRestorers = []permUser{{"daemon", 1}, {"richard", 1000}, {"other user", 1512}}

// Who owns the entries of an archive made by root. The last one owns the home.
var permOwners = []permUser{{"root", 0}, {"daemon", 1}, {"richard", 1000}}

var permFileModes = []uint32{
	0o755, 0o644, 0o444, 0o400, 0o700, 0o600, 0o640,
	0o4755, 0o2755, 0o6755, 0o4400, 0o4711,
}

var permDirModes = []uint32{0o755, 0o750, 0o700, 0o555, 0o500, 0o1777, 0o2775}

// A user's home, as restored from one computer to another. Kind: d folder,
// f file, l link (target in the last field).
var permHome = []struct {
	path, kind string
	mode       uint32
	target     string
}{
	{"home", "d", 0o750, ""},
	{"home/.ssh", "d", 0o700, ""},
	{"home/.ssh/id_ed25519", "f", 0o600, ""},
	{"home/.ssh/id_ed25519.pub", "f", 0o644, ""},
	{"home/.ssh/authorized_keys", "f", 0o644, ""},
	{"home/.ssh/config", "f", 0o600, ""},
	{"home/.gnupg", "d", 0o700, ""},
	{"home/.gnupg/pubring.kbx", "f", 0o600, ""},
	{"home/.bashrc", "f", 0o644, ""},
	{"home/.profile", "l", 0, ".bashrc"},
	{"home/bin", "d", 0o755, ""},
	{"home/bin/tool", "f", 0o755, ""},
	{"home/private", "d", 0o700, ""},
	{"home/private/notes.txt", "f", 0o600, ""},
	{"home/shared", "d", 0o2775, ""},
	{"home/shared/team.txt", "f", 0o664, ""},
}

type permEntry struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"` // file, dir, link
	Mode    uint32 `json:"mode"` // st_mode & 07777
	UID     uint32 `json:"uid"`
	GID     uint32 `json:"gid"`
	MtimeNs int64  `json:"mtime_ns"`
	Target  string `json:"target,omitempty"`
	Content string `json:"content,omitempty"`
}

type permManifest struct {
	System  string      `json:"system"`  // runtime.GOOS of the backup
	Release string      `json:"release"` // uname -sr
	MadeBy  int         `json:"made_by"` // uid that made the backup
	Entries []permEntry `json:"entries"`
}

type permTally struct{ ok, design, fail int }

func isPermChild() bool { return os.Getenv(envPermChild) != "" }

// TestPermMatrix backs up a fixture of the current system as the current user
// and restores it as the same user. As root it then reruns the matrix as
// the other users: each backs up and restores its own files, and restores
// root's archive, whose files belong to other users.
func TestPermMatrix(t *testing.T) {
	src := t.TempDir()
	unlockOnCleanup(t, src)
	m := makePermFixture(t, src)
	pxar := writePermArchive(t, src)
	permReport(t, verifyPermRestore(t, pxar, m), m, "own backup")

	if os.Geteuid() != 0 || isPermChild() {
		return
	}
	shared := permSharedDir(t)
	archives := filepath.Join(shared, "archives")
	if err := savePermArchive(archives, pxar, m); err != nil {
		t.Fatal(err)
	}
	for _, u := range permRestorers {
		u := u
		t.Run(u.name, func(t *testing.T) {
			runPermAs(t, shared, u, "^(TestPermMatrix|TestPermRestore)$", envPermArchives+"="+archives)
		})
	}
}

// TestPermProduce writes this system's archive and manifest, for the other
// systems to restore. Run as root, so that the archive has several owners.
func TestPermProduce(t *testing.T) {
	out := os.Getenv(envPermProduce)
	if out == "" {
		t.Skip(envPermProduce + " not set")
	}
	if os.Geteuid() != 0 {
		t.Fatal("run as root: the archive must hold files of several owners")
	}
	src := t.TempDir()
	unlockOnCleanup(t, src)
	m := makePermFixture(t, src)
	if err := savePermArchive(out, writePermArchive(t, src), m); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s.pxar and %s.json (%d entries) to %s", m.System, m.System, len(m.Entries), out)
}

// TestPermRestore restores every archive of a folder (from this system or
// others) as the current user and, as root, as the other users too.
func TestPermRestore(t *testing.T) {
	dir := os.Getenv(envPermArchives)
	if dir == "" {
		t.Skip(envPermArchives + " not set")
	}
	manifests, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(manifests) == 0 {
		t.Fatalf("no archive in %s (%v)", dir, err)
	}
	sort.Strings(manifests)
	type archive struct {
		m    permManifest
		pxar []byte
	}
	var all []archive
	for _, mf := range manifests {
		m, pxar := loadPermArchive(t, mf, true)
		all = append(all, archive{m, pxar})
		t.Run("from "+m.System, func(t *testing.T) {
			permReport(t, verifyPermRestore(t, pxar, m), m, "")
		})
	}
	if os.Geteuid() != 0 || isPermChild() {
		return
	}
	// The other users read a copy in a folder they can reach.
	shared := permSharedDir(t)
	archives := filepath.Join(shared, "archives")
	for _, a := range all {
		if err := savePermArchive(archives, a.pxar, a.m); err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range permRestorers {
		u := u
		t.Run(u.name, func(t *testing.T) {
			runPermAs(t, shared, u, "^TestPermRestore$", envPermArchives+"="+archives)
		})
	}
}

// TestPermExtracted checks a tree another tool (the official pxar) restored
// as root from one of the archives, against that archive's manifest.
func TestPermExtracted(t *testing.T) {
	dir, mf := os.Getenv(envPermExtracted), os.Getenv(envPermManifest)
	if dir == "" || mf == "" {
		t.Skip(envPermExtracted + " and " + envPermManifest + " not set")
	}
	m, _ := loadPermArchive(t, mf, false)
	tool := os.Getenv(envPermTool)
	if tool == "" {
		tool = "another tool"
	}
	permReport(t, comparePermTree(t, dir, m, 0), m, tool)
}

// makePermFixture builds the tree to back up in src and returns what the file
// system recorded for each entry.
func makePermFixture(t *testing.T, src string) permManifest {
	t.Helper()
	owners := []permUser{{"self", -1}}
	if os.Geteuid() == 0 {
		owners = permOwners
	}
	stamp := int64(1_700_000_000)
	next := func() time.Time { stamp += 61; return time.Unix(stamp, 123_456_789) }
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	chown := func(p string, uid int) {
		if uid >= 0 {
			must(os.Lchown(p, uid, uid))
		}
	}
	// chmod after chown, which clears setuid and setgid. A user may not be
	// allowed setgid (on the BSDs a file takes its folder's group): the
	// fixture then keeps what the file system allowed, and the manifest
	// records that.
	chmod := func(p string, mode uint32) {
		if os.Chmod(p, posixFileMode(mode)) != nil {
			_ = os.Chmod(p, posixFileMode(mode&^0o2000))
		}
	}
	file := func(p string, mode uint32, uid int) {
		must(os.WriteFile(p, []byte(filepath.Base(p)+"\n"), 0o600))
		chown(p, uid)
		chmod(p, mode)
		tm := next()
		must(os.Chtimes(p, tm, tm))
	}
	link := func(p, target string, uid int) {
		must(os.Symlink(target, p))
		chown(p, uid)
		ts := unix.NsecToTimespec(next().UnixNano())
		must(unix.UtimesNanoAt(unix.AT_FDCWD, p, []unix.Timespec{ts, ts}, unix.AT_SYMLINK_NOFOLLOW))
	}
	// Folders get their mode last, deepest first, once their contents exist.
	type pendingDir struct {
		path string
		mode uint32
	}
	var dirs []pendingDir
	dir := func(p string, mode uint32, uid int) {
		must(os.Mkdir(p, 0o700))
		chown(p, uid)
		dirs = append(dirs, pendingDir{p, mode})
	}

	for _, o := range owners {
		base := filepath.Join(src, "o-"+o.name)
		dir(base, 0o755, o.uid)
		for _, m := range permFileModes {
			file(filepath.Join(base, fmt.Sprintf("f-%04o", m)), m, o.uid)
		}
		for _, m := range permDirModes {
			d := filepath.Join(base, fmt.Sprintf("d-%04o", m))
			dir(d, m, o.uid)
			file(filepath.Join(d, "inner.txt"), 0o644, o.uid)
		}
		link(filepath.Join(base, "link-rel"), "f-0644", o.uid)
		link(filepath.Join(base, "link-dangling"), "does-not-exist", o.uid)
	}
	homeOwner := owners[len(owners)-1].uid
	for _, h := range permHome {
		p := filepath.Join(src, filepath.FromSlash(h.path))
		switch h.kind {
		case "d":
			dir(p, h.mode, homeOwner)
		case "f":
			file(p, h.mode, homeOwner)
		case "l":
			link(p, h.target, homeOwner)
		}
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		chmod(dirs[i].path, dirs[i].mode)
		tm := next()
		must(os.Chtimes(dirs[i].path, tm, tm))
	}

	m := permManifest{System: runtime.GOOS, Release: permRelease(), MadeBy: os.Geteuid()}
	must(filepath.Walk(src, func(p string, fi os.FileInfo, err error) error {
		if err != nil || p == src {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		st := fi.Sys().(*syscall.Stat_t)
		e := permEntry{
			Path: filepath.ToSlash(rel), Mode: uint32(st.Mode) & 0o7777,
			UID: st.Uid, GID: st.Gid, MtimeNs: fi.ModTime().UnixNano(),
		}
		switch {
		case fi.Mode()&os.ModeSymlink != 0:
			e.Kind = "link"
			e.Target, err = os.Readlink(p)
		case fi.IsDir():
			e.Kind = "dir"
		default:
			e.Kind = "file"
			var b []byte
			b, err = os.ReadFile(p)
			e.Content = string(b)
		}
		m.Entries = append(m.Entries, e)
		return err
	}))
	for _, e := range m.Entries {
		if want := permWanted(e.Path); want != 0 && e.Mode != want {
			t.Logf("%s: the file system kept %04o of %04o", e.Path, e.Mode, want)
		}
	}
	return m
}

// permWanted is the mode the fixture asked for, read from an f-/d- name.
func permWanted(path string) uint32 {
	var m uint32
	b := filepath.Base(path)
	if _, err := fmt.Sscanf(b, "f-%o", &m); err == nil {
		return m
	}
	if _, err := fmt.Sscanf(b, "d-%o", &m); err == nil {
		return m
	}
	return 0
}

func permRelease() string {
	var u unix.Utsname
	if unix.Uname(&u) != nil {
		return runtime.GOOS
	}
	return unix.ByteSliceToString(u.Sysname[:]) + " " + unix.ByteSliceToString(u.Release[:])
}

func writePermArchive(t *testing.T, src string) []byte {
	t.Helper()
	pxar, _, _ := writePosixArchive(t, src)
	return pxar
}

// verifyPermRestore checks the archive against the manifest, restores it as
// the app does (owners and full modes only for a root restore, folders
// after their contents), and compares the result.
func verifyPermRestore(t *testing.T, pxar []byte, m permManifest) permTally {
	t.Helper()
	var tally permTally
	entries, err := NewPXARReader(pxar).ListEntries()
	if err != nil {
		t.Fatal(err)
	}
	archived := map[string]PXARTreeEntry{}
	for _, e := range entries {
		archived[e.Path] = e
	}
	// The archive records exactly what the file system had.
	for _, e := range m.Entries {
		a, ok := archived[e.Path]
		var bad string
		switch {
		case !ok:
			bad = "not in the archive"
		case e.Kind == "link" && (!a.IsSymlink || a.LinkTarget != e.Target):
			bad = fmt.Sprintf("archived as link=%v to %q", a.IsSymlink, a.LinkTarget)
		case e.Kind != "link" && a.Mode&0o7777 != e.Mode:
			bad = fmt.Sprintf("archived mode %04o", a.Mode&0o7777)
		case a.UID != e.UID || a.GID != e.GID:
			bad = fmt.Sprintf("archived owner %d:%d", a.UID, a.GID)
		case e.Kind != "dir" && a.ModTime*1e9+int64(a.ModNanos) != e.MtimeNs:
			bad = fmt.Sprintf("archived time %d.%09d", a.ModTime, a.ModNanos)
		}
		if bad != "" {
			t.Errorf("%s: %s, recorded %04o %d:%d %d", e.Path, bad, e.Mode, e.UID, e.GID, e.MtimeNs)
			tally.fail++
		}
	}

	root := os.Geteuid() == 0
	dest := t.TempDir()
	unlockOnCleanup(t, dest)
	// Restore into an empty folder, then again over that restore with
	// overwrite, as a second restore or a Roll back does: the folders the
	// first one made read-only (0555, 0500) must not stop the second.
	var r permTally
	firstLen := 0
	for _, pass := range []struct {
		name      string
		overwrite bool
	}{{"into an empty folder", false}, {"over the first restore", true}} {
		pr := NewPXARReader(pxar)
		pr.SetRestoreOwnership(root)
		res, err := pr.ExtractFiltered(dest, nil, pass.overwrite)
		if err != nil {
			t.Fatalf("restore %s: %v", pass.name, err)
		}
		for _, f := range res {
			if f.Skipped {
				t.Errorf("restore %s: %s skipped: %s", pass.name, f.Path, f.SkipReason)
				tally.fail++
			}
		}
		FinishDirs(res, root)
		t.Logf("restore %s:", pass.name)
		c := comparePermTree(t, dest, m, os.Geteuid())
		if !pass.overwrite {
			r = c
			firstLen = len(res)
		} else {
			if len(res) != firstLen {
				t.Errorf("restore %s: %d entries, the first restore had %d", pass.name, len(res), firstLen)
				r.fail++
			}
			// Count the entries once; the second pass adds only failures.
			r.fail += c.fail
		}
	}

	// One file selected, into an empty folder: the folders on the way to it
	// are restored with their recorded modes and owners, not just created
	// (it was GAP-2: ~/.ssh came back 0755).
	const sel = "home/.ssh/id_ed25519"
	selDest := t.TempDir()
	unlockOnCleanup(t, selDest)
	pr := NewPXARReader(pxar)
	pr.SetRestoreOwnership(root)
	res, err := pr.ExtractFiltered(selDest, []string{sel}, false)
	if err != nil {
		t.Fatalf("restore of %s alone: %v", sel, err)
	}
	for _, f := range res {
		if f.Skipped {
			t.Errorf("restore of %s alone: %s skipped: %s", sel, f.Path, f.SkipReason)
			r.fail++
		}
	}
	FinishDirs(res, root)
	sub := m
	sub.Entries = nil
	for _, e := range m.Entries {
		if e.Path == sel || strings.HasPrefix(sel, e.Path+"/") {
			sub.Entries = append(sub.Entries, e)
		}
	}
	if len(sub.Entries) != 3 {
		t.Errorf("restore of %s alone: %d entries in the manifest, want 3 (home, home/.ssh, the file)", sel, len(sub.Entries))
		r.fail++
	}
	t.Logf("restore of %s alone:", sel)
	r.fail += comparePermTree(t, selDest, sub, os.Geteuid()).fail
	return permTally{tally.ok + r.ok, tally.design + r.design, tally.fail + r.fail}
}

// comparePermTree compares a restored tree with the manifest, for a restore
// run by uid, and logs one line per entry.
func comparePermTree(t *testing.T, dest string, m permManifest, uid int) permTally {
	t.Helper()
	var tally permTally
	root := uid == 0
	umask := unix.Umask(0)
	unix.Umask(umask)
	created := uint32(0o755) &^ uint32(umask) // the mode a restore creates a folder with

	t.Logf("restore as uid %d of the archive made on %s (%s) by uid %d", uid, m.System, m.Release, m.MadeBy)
	t.Logf("%-32s %-4s  %-16s  %-16s  %s", "entry", "kind", "recorded", "restored", "verdict")
	for _, e := range m.Entries {
		p := filepath.Join(dest, filepath.FromSlash(e.Path))
		fi, err := os.Lstat(p)
		if err != nil {
			t.Errorf("%s: %v", e.Path, err)
			tally.fail++
			continue
		}
		st := fi.Sys().(*syscall.Stat_t)
		mode, ru, rg := uint32(st.Mode)&0o7777, st.Uid, st.Gid
		var problems, design []string

		switch e.Kind {
		case "link":
			if fi.Mode()&os.ModeSymlink == 0 {
				problems = append(problems, "not a link")
			} else if tgt, _ := os.Readlink(p); tgt != e.Target {
				problems = append(problems, fmt.Sprintf("target %q", tgt))
			}
		case "dir":
			if !fi.IsDir() {
				problems = append(problems, "not a folder")
			}
		default:
			if b, err := os.ReadFile(p); err != nil || string(b) != e.Content {
				problems = append(problems, "content differs")
			}
		}

		// Mode (a link has none of its own to restore).
		if e.Kind != "link" {
			want := e.Mode
			if !root {
				want = e.Mode & 0o777
				if e.Kind == "dir" {
					want &= created
				}
			}
			if mode != want {
				problems = append(problems, fmt.Sprintf("mode %04o, want %04o", mode, want))
			} else if want != e.Mode {
				design = append(design, "special bits or group/other write dropped (not root)")
			}
		}

		// Owner: root puts back the recorded one; a user owns what it restores.
		if root {
			if ru != e.UID || rg != e.GID {
				problems = append(problems, fmt.Sprintf("owner %d:%d", ru, rg))
			}
		} else if ru != uint32(uid) {
			problems = append(problems, fmt.Sprintf("owner %d, want %d", ru, uid))
		} else if e.UID != uint32(uid) {
			design = append(design, fmt.Sprintf("owned by the restoring user, archive says %d", e.UID))
		}

		// Time, for files and links (folders: gui/restore_dirtimes.go).
		if e.Kind != "dir" {
			got := fi.ModTime().UnixNano()
			switch d := e.MtimeNs - got; {
			case d == 0:
			case d > 0 && d < 1e9 && got%1000 == 0:
				design = append(design, "time rounded by the file system")
			default:
				problems = append(problems, fmt.Sprintf("time %s, want %s",
					time.Unix(0, got).UTC().Format(time.RFC3339Nano), time.Unix(0, e.MtimeNs).UTC().Format(time.RFC3339Nano)))
			}
		}

		verdict := "ok"
		switch {
		case len(problems) > 0:
			verdict = "FAIL " + strings.Join(problems, "; ")
			t.Errorf("%s: %s", e.Path, strings.Join(problems, "; "))
			tally.fail++
		case len(design) > 0:
			verdict = "ok, " + strings.Join(design, "; ")
			tally.design++
		default:
			tally.ok++
		}
		t.Logf("%-32s %-4s  %04o %5d:%-5d  %04o %5d:%-5d  %s", e.Path, e.Kind, e.Mode, e.UID, e.GID, mode, ru, rg, verdict)
	}
	return tally
}

// permReport logs one summary line and, outside a rerun, adds it to the
// report file (a rerun's lines are added by its parent, from its output).
func permReport(t *testing.T, r permTally, m permManifest, how string) {
	t.Helper()
	who := fmt.Sprintf("uid %d", os.Geteuid())
	for _, u := range append([]permUser{{"root", 0}}, permRestorers...) {
		if u.uid == os.Geteuid() {
			who = fmt.Sprintf("%s (%d)", u.name, u.uid)
		}
	}
	if how != "" {
		who += ", " + how
	}
	status := "✅"
	if r.fail > 0 {
		status = "❌"
	}
	line := fmt.Sprintf("| %s | %s | %s | %s | %d | %d | %d |", runtime.GOOS, m.System, who, status, r.ok, r.design, r.fail)
	t.Log(permReportTag + line)
	if !isPermChild() {
		addPermReport(line)
	}
}

func addPermReport(line string) {
	path := os.Getenv(envPermReport)
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintln(f, line)
}

func savePermArchive(dir string, pxar []byte, m permManifest) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		return err
	}
	j, err := json.MarshalIndent(m, "", " ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, m.System+".pxar"), pxar, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, m.System+".json"), j, 0o644)
}

func loadPermArchive(t *testing.T, manifest string, withArchive bool) (permManifest, []byte) {
	t.Helper()
	var m permManifest
	j, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(j, &m); err != nil {
		t.Fatal(err)
	}
	if !withArchive {
		return m, nil
	}
	pxar, err := os.ReadFile(strings.TrimSuffix(manifest, ".json") + ".pxar")
	if err != nil {
		t.Fatal(err)
	}
	return m, pxar
}

// permSharedDir is a folder every user can reach, under /tmp: the folders
// above t.TempDir() may be root-only (macOS puts root's in /var/folders).
func permSharedDir(t *testing.T) string {
	t.Helper()
	base, err := os.MkdirTemp("/tmp", "salvanza-perm-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	if err := os.Chmod(base, 0o755); err != nil {
		t.Fatal(err)
	}
	return base
}

// runPermAs runs this test binary again as u (gid = uid, no supplementary
// groups; no account needed), from a copy in shared, and adds its summary
// lines to the report.
func runPermAs(t *testing.T, shared string, u permUser, run string, env ...string) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(shared, "pbscommon.test")
	if _, err := os.Stat(bin); err != nil {
		if err := copyExecutable(self, bin); err != nil {
			t.Fatal(err)
		}
	}
	tmp, err := os.MkdirTemp(shared, "tmp-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(tmp, 0o1777); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "-test.run", run, "-test.v", "-test.count=1")
	cmd.Dir = tmp
	cmd.Env = append([]string{envPermChild + "=1", "TMPDIR=" + tmp, "HOME=" + tmp, "PATH=/usr/bin:/bin"}, env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{
		Uid: uint32(u.uid), Gid: uint32(u.uid), Groups: []uint32{},
	}}
	out, err := cmd.CombinedOutput()
	for _, l := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		t.Log(l)
		if i := strings.Index(l, permReportTag); i >= 0 {
			addPermReport(l[i+len(permReportTag):])
		}
	}
	if err != nil {
		t.Fatalf("as %s (uid %d): %v", u.name, u.uid, err)
	}
}

func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, 0o755)
}

// unlockOnCleanup makes every folder under dir writable again before the
// test's clean-up removes it: read-only folders would stop a user's clean-up.
func unlockOnCleanup(t *testing.T, dir string) {
	t.Helper()
	t.Cleanup(func() {
		_ = filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
			if err == nil && fi.IsDir() {
				_ = os.Chmod(p, 0o755)
			}
			return nil
		})
	})
}
