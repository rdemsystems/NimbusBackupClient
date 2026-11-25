package pbscommon

import (
	"os"
	"sort"
	"time"
)

// ApplyDirTimes sets each restored folder's recorded modification time once
// its contents are written (writing into a folder changes its time), deepest
// first. A folder that keeps its own time (KeepOwnTime) is left as it is.
// For restores that have no progress to show (the text-mode window); the
// app's own pass (gui/restore_dirtimes.go) also reports progress and logs.
func ApplyDirTimes(extracted []PXARExtractedFile) (applied, failed int) {
	var dirs []PXARExtractedFile
	for _, f := range extracted {
		if f.IsDir && !f.Skipped && !f.KeepOwnTime && (f.ModTimeSet || f.ModTime != 0) {
			dirs = append(dirs, f)
		}
	}
	sort.SliceStable(dirs, func(i, j int) bool { return len(dirs[i].Path) > len(dirs[j].Path) })
	for _, d := range dirs {
		t := time.Unix(d.ModTime, int64(d.ModNanos))
		if err := os.Chtimes(OSPath(d.Path), t, t); err != nil {
			failed++
		} else {
			applied++
		}
	}
	return applied, failed
}

// FinishDirs applies the restored folders' recorded mode (and, with owner,
// their owner) once their contents are written, deepest first, so that a
// read-only folder does not stop its own restore. Every restore path calls
// it after extracting (the app's Restore, Undelete and Roll back).
//
// With owner, a root restore on Unix, it puts back the owner and the full
// mode, setuid, setgid and sticky included (ApplyDirMetadata). Otherwise it
// applies the permission bits only, and never opens a folder more than it
// already is (ApplyDirMode): a private 0700 folder comes back 0700, and a
// 0777 folder from a Windows backup stays at the default 0755. A folder the
// extraction had to open to write into (PXARExtractedFile.Unlocked, such as
// a 0555 folder put back by an earlier restore) is measured against its
// mode before the restore, so it is 0555 again.
//
// A folder that was already there and keeps its own time (an ancestor of
// the selection, or a folder the restore may not overwrite) is left as it
// is, and so is one from an archive that records no real permissions
// (FixedPerms). Failures are ignored: the content is already on disk.
func FinishDirs(extracted []PXARExtractedFile, owner bool) {
	var dirs []PXARExtractedFile
	for _, f := range extracted {
		if f.IsDir && !f.Skipped && !(f.Existed && f.KeepOwnTime) && !f.FixedPerms {
			dirs = append(dirs, f)
		}
	}
	sort.SliceStable(dirs, func(i, j int) bool { return len(dirs[i].Path) > len(dirs[j].Path) })
	for _, d := range dirs {
		if owner {
			ApplyDirMetadata(d.Path, d.RawMode, d.UID, d.GID)
		} else {
			applyDirMode(d.Path, d.RawMode, d.PrevMode, d.Unlocked)
		}
	}
}

// RelockDirs gives the folders an extraction opened to write into
// (PXARExtractedFile.Unlocked) their mode before the restore back, deepest
// first. A restore that does not call FinishDirs, or stops before it, calls
// it instead, so that no folder stays more open than it was.
func RelockDirs(extracted []PXARExtractedFile) {
	var dirs []PXARExtractedFile
	for _, f := range extracted {
		if f.Unlocked {
			dirs = append(dirs, f)
		}
	}
	sort.SliceStable(dirs, func(i, j int) bool { return len(dirs[i].Path) > len(dirs[j].Path) })
	for _, d := range dirs {
		relockDir(OSPath(d.Path), d.PrevMode)
	}
}
