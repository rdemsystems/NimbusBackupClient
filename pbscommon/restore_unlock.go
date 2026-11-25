package pbscommon

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// dirUnlocks records the folders a restore that does not run as root had to
// open before writing into them: a folder the user owns but may not write
// into or enter, such as a 0555 folder put back by an earlier restore (Go's
// module cache is read-only throughout). Without it a second restore, or a
// Roll back, could not create its temporary files there.
type dirUnlocks struct {
	mu sync.Mutex
	// prev holds each opened folder's mode before the restore.
	prev map[string]os.FileMode
	// seen holds every folder already looked at, opened or not.
	seen map[string]bool
}

// writableDir creates dir and its missing parents, as os.MkdirAll does, after
// making sure the restore may write into it, or into the folder that receives
// the first missing one (unlockDir).
func (pr *PXARReader) writableDir(dir string) error {
	if dirUnlockApplies() && !pr.dirSeen(dir) {
		if _, err := os.Lstat(dir); err == nil {
			pr.unlockDir(dir)
		} else {
			for p := dir; filepath.Dir(p) != p; {
				p = filepath.Dir(p)
				if _, err := os.Lstat(p); err == nil {
					pr.unlockDir(p)
					break
				}
			}
		}
	}
	return os.MkdirAll(dir, 0755)
}

// dirSeen reports whether unlockDir has already looked at dir, so that a
// folder holding many files costs one look, not one per file.
func (pr *PXARReader) dirSeen(dir string) bool {
	pr.unlocks.mu.Lock()
	defer pr.unlocks.mu.Unlock()
	return pr.unlocks.seen[filepath.Clean(dir)]
}

// unlockDir opens dir for the restore once (openDirForRestore) and remembers
// its earlier mode.
func (pr *PXARReader) unlockDir(dir string) {
	dir = filepath.Clean(dir)
	u := &pr.unlocks
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.seen[dir] {
		return
	}
	if u.seen == nil {
		u.seen = map[string]bool{}
	}
	u.seen[dir] = true
	if prev, ok := openDirForRestore(dir); ok {
		if u.prev == nil {
			u.prev = map[string]os.FileMode{}
		}
		u.prev[dir] = prev
	}
}

// settleUnlocks runs when an extraction ends. After a successful one, a
// folder it opened that the archive restores keeps its earlier mode in
// PrevMode, for FinishDirs to apply the recorded mode from (or RelockDirs to
// put it back). Every other one (a folder on the way to the selection, or
// one that was already there and keeps its own time), and every one after a
// failed extraction, gets its earlier mode back now, deepest first.
//
// Known limit: a folder its owner may not even enter (no u+x, which nothing
// makes by itself) that is on the way to a folder handed to FinishDirs is
// closed before FinishDirs reaches that folder, which then stays opened.
func (pr *PXARReader) settleUnlocks(res []PXARExtractedFile, err error) {
	u := &pr.unlocks
	u.mu.Lock()
	prev := u.prev
	u.prev, u.seen = nil, nil
	u.mu.Unlock()
	if len(prev) == 0 {
		return
	}
	if err == nil {
		for i := range res {
			f := &res[i]
			p := filepath.Clean(OSPath(f.Path))
			m, ok := prev[p]
			if !ok || !f.IsDir || f.Skipped || (f.Existed && f.KeepOwnTime) {
				continue
			}
			f.Unlocked, f.PrevMode = true, m
			delete(prev, p)
		}
	}
	paths := make([]string, 0, len(prev))
	for p := range prev {
		paths = append(paths, p)
	}
	sort.Slice(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })
	for _, p := range paths {
		relockDir(p, prev[p])
	}
}

// relockDir gives an opened folder its earlier mode back, if it is still a
// folder: a symbolic link put in its place meanwhile is not followed.
func relockDir(path string, prev os.FileMode) {
	if fi, err := os.Lstat(path); err == nil && fi.IsDir() {
		_ = os.Chmod(path, prev)
	}
}
