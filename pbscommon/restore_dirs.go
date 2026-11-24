package pbscommon

import "sort"

// FinishDirs applies the restored folders' recorded mode (and, with owner,
// their owner) once their contents are written, deepest first, so that a
// read-only folder does not stop its own restore. Every restore path calls
// it after extracting (the app's Restore, Undelete and Roll back).
//
// With owner, a root restore on Unix, it puts back the owner and the full
// mode, setuid, setgid and sticky included (ApplyDirMetadata). Otherwise it
// applies the permission bits only, and never opens a folder more than it
// already is (ApplyDirMode): a private 0700 folder comes back 0700, and a
// 0777 folder from a Windows backup stays at the default 0755.
//
// A folder that was already there and keeps its own time (an ancestor of
// the selection, or a folder the restore may not overwrite) is left as it
// is. Failures are ignored: the content is already on disk.
func FinishDirs(extracted []PXARExtractedFile, owner bool) {
	var dirs []PXARExtractedFile
	for _, f := range extracted {
		if f.IsDir && !f.Skipped && !(f.Existed && f.KeepOwnTime) {
			dirs = append(dirs, f)
		}
	}
	sort.SliceStable(dirs, func(i, j int) bool { return len(dirs[i].Path) > len(dirs[j].Path) })
	for _, d := range dirs {
		if owner {
			ApplyDirMetadata(d.Path, d.RawMode, d.UID, d.GID)
		} else {
			ApplyDirMode(d.Path, d.RawMode)
		}
	}
}
