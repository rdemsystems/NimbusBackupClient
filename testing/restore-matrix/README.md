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
