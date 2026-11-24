# Backup and restore matrix

What a backup records and what a restore puts back, for every kind of file, owner and restoring user, on every Unix system Salvanza runs on, and across systems. The tests are in `pbscommon/pxar_perm_matrix_unix_test.go`; `.github/workflows/restore-matrix.yml` runs them nightly and on demand.

## What is tested

**Systems:** Linux, macOS, FreeBSD, OpenBSD, NetBSD and Solaris. Each one backs up its own fixture, and every archive is restored on every system: an archive made on FreeBSD is restored on Linux, one made on Linux on Solaris, and so on. On Linux, the official `pxar` tool also restores each archive, as the reference reader.

**What is backed up** (as root, each set three times, owned by root, by a system account, uid 1, and by a user, uid 1000):

| Kind | Modes |
|---|---|
| Files | 0755, 0644, 0444, 0400, 0700, 0600, 0640, setuid 4755, setgid 2755, both 6755, 4400, 4711 |
| Folders, each with a file in it | 0755, 0750, 0700, 0555, 0500, sticky 1777, setgid 2775 |
| Symbolic links, with their own owner and time | relative, dangling |
| A user's home, owned by uid 1000 | `~/.ssh` 0700 with its keys 0600, `~/.gnupg`, `~/private` 0700, `~/shared` 2775, `~/bin`, dot files, a link |

**Who restores:**

| User | uid | Case |
|---|---|---|
| root | 0 | a full restore |
| daemon | 1 | a system account |
| richard | 1000 | the user who owns the home, on the same computer |
| other user | 1512 | richard on another computer, where his uid is 1512: restoring his home from his laptop onto his desktop |

Each user also backs up and restores its own files.

**What is compared**, for every entry: the content (files), the target (links), the mode, the owner and the modification time (files and links; a folder's time is set by the app, `gui/restore_dirtimes.go`). The archive itself must record exactly what the file system had.

## Expected results

| | root | any other user |
|---|---|---|
| File permission bits | as recorded | as recorded |
| setuid, setgid, sticky | as recorded | dropped, as `tar` does |
| Folder permissions | as recorded | as recorded, but never more open than 0755 (the folder's mode at creation, under the usual umask): 0700 stays 0700, 0777 becomes 0755 |
| Owner and group | as recorded | the restoring user |
| Link target and own time | as recorded | as recorded |
| File time | as recorded, to the nanosecond where the file system keeps it | the same |

So richard restoring his home as uid 1512 gets every file owned by uid 1512, with `~/.ssh` still 0700 and his keys still 0600.

A result that differs from the recorded value for one of these reasons is counted as **ok by design** in the summary, with the reason in the log. Anything else fails the job.

## Running it

Any user, the current system only:

```sh
cd pbscommon && go test -run 'TestPermMatrix|TestFinishDirs|TestApplyDirMode' -v .
```

As root, the matrix also runs as uid 1, 1000 and 1512 (the test reruns itself with those credentials; no accounts are needed):

```sh
cd pbscommon && go test -c -o /tmp/pbscommon.test . && sudo /tmp/pbscommon.test -test.run TestPermMatrix -test.v
```

`run.sh` is what the workflow runs on each system. It writes the archive for the other systems and a summary that `summary.sh` turns into the job summary.

## Known gaps

What a backup or a restore does not keep yet. Each gap that can be shown in a unit test has a subtest in `pbscommon/pxar_gaps_unix_test.go` (`TestRestoreGaps`), which records whether the gap is still there without failing; the job summary lists them under the matrix. When a gap is closed, its subtest says so, and the case becomes a check in the matrix.

| | Gap | What happens | Shown by |
|---|---|---|---|
| GAP-1 | Restoring onto a computer where the user's uid differs | richard is uid 1000 on his laptop and 1512 on his desktop. A root restore of his home on the desktop gives it to uid 1000, the archive's: he cannot read his own `~/.ssh`, and uid 1000 may be someone else there. There is no option to restore as a chosen owner, or to map one uid to another. (Restoring as uid 1512 works: that user owns all it restores.) | `TestRestoreGaps` (root) |
| GAP-2 | Parent folders of a selective restore | Restoring one file creates the folders on the way to it as 0755, not with their recorded modes: a selective restore walks only the selection and never reads the ancestors' metadata. Restoring `~/.ssh/id_ed25519` alone leaves `~/.ssh` 0755. | `TestRestoreGaps` |
| GAP-3 | Hard links | Two names of one file are backed up as two files and restored as two files: the link is lost, and the data is stored twice. pxar has a hard link entry; the writer does not use it. | `TestRestoreGaps` |
| GAP-4 | FIFOs, sockets, device nodes | Skipped by the backup, because reading them would hang it. The official client records them as entries without data and restores them. | `TestRestoreGaps` |
| GAP-5 | The text-mode window's restore | `tui/restore.go` extracts in its own process and never puts back owners, setuid, setgid or sticky, folder permissions or folder times, even as root. The app's Restore, Undelete and Roll back do. | code (needs a PBS to test) |
| GAP-6 | A Windows backup restored on Unix | A Windows backup records every entry as 0777 and uid/gid 1000. A root restore on Unix gives every file to uid 1000, writable by everyone; a user's restore makes files 0777 (folders stay 0755). | code; a Windows producer in the matrix would show it |
| GAP-7 | A Unix backup restored on Windows | Symbolic links are not restored, and modes and owners are not applied. | code |
| GAP-8 | Extended attributes, ACLs, file capabilities | Not recorded on the BSDs, macOS and Solaris. On Linux, extended attributes and POSIX ACLs are in Salvanza's sidecar file (`proxmox-client-acls.json.gz.blob`), which the official client does not read. Planned for 2.2: write them into the pxar as the official client does, keep the sidecar for the rest (NFSv4 ACLs). | plan in #167 |
