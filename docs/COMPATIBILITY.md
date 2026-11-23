# Supported systems

What Salvanza 2.0.0 runs on and what it has been tested with. "Tested" means we
run it there ourselves or in CI; "expected to work" means it should, but we have
not tested it.

## The app and the background service

| System | Download | Status |
|---|---|---|
| Windows 10 and 11, Windows Server 2022 and 2025 (x64) | MSI | Tested |
| Windows Server 2016 and 2019 (x64) | MSI | Expected to work |
| Debian, Ubuntu, Linux Mint (amd64, arm64) | .deb | Tested |
| Fedora (x86_64, aarch64) | .rpm | Tested |
| Arch Linux (x86_64) | .pkg.tar.zst | Tested |
| Windows 7, 8 and 8.1, Server 2008 and 2012, XP and 2003 | | Not supported: Go, which the client is built with, no longer runs there, and the app needs WebView2 |
| Windows on ARM | | No build |

On Linux the app needs WebKitGTK, and polkit to connect to the service. The
background service is a systemd unit: without systemd (Slackware, Alpine...) the
app still runs its Backup Sets itself while it is open.

## The command line (salvanza-cli)

A single file with nothing to install alongside it.

| System | Folder backups | Machine backups | Snapshot |
|---|---|---|---|
| Windows (x64) | Tested | Tested | VSS |
| Linux (amd64, arm64) | Tested | Tested | elastio-snap or dattobd (not btrfs) |
| FreeBSD, OpenBSD, NetBSD, Solaris (x86_64) | Tested in CI | No | None |
| macOS (Intel, Apple Silicon) | Expected to work | No | None |
| 32-bit systems | No build | | |

What a folder backup keeps of each file besides its contents:

| | Symbolic links | Permissions and owner | Other |
|---|---|---|---|
| Linux | Stored as links | Real permissions, owner, time to the nanosecond | Extended attributes and POSIX ACLs |
| Windows | Skipped, as are junctions | NTFS permissions (owner, group, access list) | Attributes, creation time, alternate data streams, extended attributes |
| BSD, Solaris, macOS | Skipped | Fixed: 0777, owner 1000, whole seconds | None |

## Proxmox Backup Server

| PBS | Status |
|---|---|
| 4.0, 4.1, 4.2 | Tested |
| 3.4 | Passes our test suite, but PBS 3 is no longer supported by Proxmox |

## The official Proxmox client

Backups made by Salvanza restore byte for byte with proxmox-backup-client
(tested with 3.4.9 and 4.2.14), and backups made by it restore with Salvanza.
What Salvanza records beside the archive (Windows permissions and attributes,
Linux extended attributes and ACLs) is only applied by Salvanza; see the
[command line manual](manual/CLI.md).

## The restore ISO

The bare-metal restore ISO (built on Clonezilla Live) restores Windows and Linux
machine backups on UEFI and BIOS computers, onto a disk at least as large as the
original.

## Older Windows

For Windows XP to Server 2012, back the folders up from outside: share them over
SMB and run the Linux command line on a small VM, for example in Proxmox VE. XP,
2000 and 2003 need SMB1, so keep that on an isolated network. There are no
shadow copies that way, so run it while the programs are idle.
