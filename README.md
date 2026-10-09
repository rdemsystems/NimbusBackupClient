# Nimbus Backup — Windows client for Proxmox Backup Server

🇬🇧 English | [🇫🇷 Français](README.fr.md)

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rdemsystems/NimbusBackupClient)](https://github.com/rdemsystems/NimbusBackupClient/releases)
[![Documentation](https://img.shields.io/badge/docs-nimbus.rdem--systems.com-orange)](https://nimbus.rdem-systems.com/en/?utm_source=github)

**Nimbus Backup is an open-source (GPL-3.0) Windows backup client for Proxmox Backup Server (PBS).**
A modern GUI to back up Windows servers and workstations to PBS — VSS-consistent snapshots, scheduled jobs, file and disk modes, snapshot browsing and restore, multi-PBS support, and a Windows service — plus command-line tools for directory and full-machine backups. Looking for **offsite, immutable** PBS storage without self-hosting? See the [managed service](#️-managed-pbs-offsite--immutable) below.

Nimbus Backup is the RDEM Systems build of [Proxmox Backup Client GO](https://github.com/tizbac/proxmoxbackupclient_go): the GUI we developed has been merged upstream, and both projects now share the same code base (see [Relationship with upstream](#-relationship-with-upstream)).

📖 **Full documentation, installation guide and PBS hosting:** [nimbus.rdem-systems.com](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github&utm_medium=readme&utm_campaign=nbc-readme-top)

## 📦 Download

👉 **[Download the latest release](https://github.com/rdemsystems/NimbusBackupClient/releases)**

Each release ships:
- `NimbusBackup.msi` — installer (GUI + Windows service), **recommended for production**
- `NimbusBackup.exe` — standalone GUI
- `nimbus-backup-cli-<version>-windows.zip` / `-linux.tar.gz` / `-macos.tar.gz` — command-line tools
- `SHA256SUMS.txt` — checksums

> ⚠️ **Windows says "virus detected" (e.g. `Trojan:Win32/Sabsik.FL.A!ml`) or shows a SmartScreen warning?**
> This is a known **false positive** for Go/Wails applications — it is *not* a virus. The `!ml` suffix means it comes from a machine-learning model that flags *unsigned, low-prevalence* executables.
> Read [why this happens and how to verify the download](https://nimbus.rdem-systems.com/en/antivirus-false-positive/?utm_source=github).

### 🔎 Verify any download

Every release ships SHA-256 checksums and a signed **build-provenance attestation** (cryptographic proof the binary was produced by this repo's CI, from a specific commit):

```powershell
Get-FileHash .\NimbusBackup.msi -Algorithm SHA256   # compare against SHA256SUMS.txt
gh attestation verify .\NimbusBackup.msi --repo rdemsystems/NimbusBackupClient
```

**VirusTotal.** Each release's notes link the VirusTotal report of that build's installer (published only when the scan is clean). Earlier reports, 0 detections:
[0.2.108](https://www.virustotal.com/gui/file/6e8fb7ce9af740d470e947addb8daba4331c0b88e8bfdec9e0697ea8f7f29e9e/detection) ·
[0.2.107](https://www.virustotal.com/gui/file/6fd6c6fa77e0305c129ef882a3745100aa6033187a6d52a4af94149ab6b666d2/detection) ·
[0.2.106](https://www.virustotal.com/gui/file/ad6e56700ed9df8e088906e38cee2e2882fc7045f4e39269de0e379a01784ad7/detection)

> ℹ️ **Code signing:** Windows binaries are **not yet Authenticode-signed**, which is what triggers the SmartScreen / `!ml` warnings above. Our request for a free OSS certificate from the [SignPath Foundation](https://signpath.org) got no reply; signing through Azure Artifact Signing is being set up and is targeted for **0.4.1**. Until then, provenance is established via the build-provenance attestation and checksums above.

### 🐧 On Linux? Use the official client

The Nimbus Backup GUI is Windows-only (the CLI tools also build for Linux and macOS). For file-level backups on Linux, use Proxmox's own `proxmox-backup-client` — we package it for the distributions Proxmox doesn't cover:

👉 **[rdemsystems/unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)** — signed package repositories for Debian/Ubuntu, Fedora/RHEL/Rocky/AlmaLinux, Arch and Alpine (amd64 & arm64). Proxmox's official static binary, repackaged unchanged — not patched, not recompiled.

## ☁️ Managed PBS (offsite & immutable)

Don't want to self-host Proxmox Backup Server? Use our fully managed, **offsite immutable** PBS datastores:
👉 **[Configure your backup & see pricing](https://nimbus.rdem-systems.com/en/choose-backup/?utm_source=github)**

- ✅ From €12/TB/month
- ✅ 1 TB free trial
- ✅ A ready-made [offsite target for your PBS](https://nimbus.rdem-systems.com/en/offsite-proxmox-backup/?utm_source=github) — air-gapped, immutable datastores
- ✅ [NimbusBackup — Managed PBS hosting in France](https://nimbus.rdem-systems.com/en/?utm_source=github)

## 📚 Documentation

- **Complete Proxmox Backup guide** — PBS deployment best practices ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/complete-proxmox-backup-guide/?utm_source=github))
- **Back up Windows with Proxmox Backup Server** — Windows-specific deployment guide ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github))
- **PBS vs Veeam** — Proxmox Backup Server comparison ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/pbs-vs-veeam-proxmox-backup-comparison/?utm_source=github))
- In this repo: [multi-PBS user guide](MULTI_PBS_USER_GUIDE.md) · [unattended deployment examples](examples/automation/) · [bare-metal restore with Clonezilla](PATCH-CLONEZILLA.md) · [changelog](CHANGELOG.md)

## ✨ Features

### GUI (recommended)
- **🌍 Multi-language** — English, French, Italian, German and Polish interface
- User-friendly configuration with connection testing (API token or username/password)
- Real-time backup progress with speed and ETA, cancel at any time
- VSS (Volume Shadow Copy) support for consistent backups
- Multi-folder backup, file and disk (full machine) modes
- Snapshot browsing, file search (wildcards) and restore
- Multi-PBS server support, certificate fingerprint pinning (TOFU)
- **🔒 Client-side encryption** (AES-256-GCM), key files compatible with `proxmox-backup-client` and Proxmox VE
- Windows service mode + scheduled backups, backup history with one-click rerun
- Debug logging for troubleshooting

### Command-line tools
- `proxmoxbackup-directory` — directory (PXAR) backups with deduplication, stream backups (`-backupstream`, e.g. a `mysqldump` pipe), e-mail notifications, JSON config file
- `proxmoxbackup-machine` — full live machine backups as a bootable disk image (FIDX): VSS on Windows, incremental, parallel hashing
- `proxmoxbackup-nbd` — NBD server to mount a disk backup on Linux (file-level restore, bare-metal restore from a [patched Clonezilla live ISO](PATCH-CLONEZILLA.md))

### 📸 Screenshots

![Server configuration](docs/screenshots/nimbus-gui-liste-servers.png)
*Multi-PBS server management with status indicators*

![Add server form](docs/screenshots/nimbus-gui-add-server-form.png)
*Easy server configuration with connection testing*

![One-shot backup](docs/screenshots/nimbus-gui-one-shot-backup.png)
*Real-time backup progress with ETA and speed*

### Smart system exclusions (file mode)
When backing up an entire drive (e.g. `D:\`), Nimbus Backup automatically excludes:

**System folders:** `System Volume Information` (VSS storage, can be 100+ GB), `$RECYCLE.BIN`, `Recovery`.
**System files:** `pagefile.sys`, `hiberfil.sys`, `swapfile.sys`.

**Why it matters:** a drive may report 1.03 TB used while the real files are ~141 GB. Without exclusions the backup would include VSS snapshots (wasted space and time); with them the backup size matches the real data.

**Recommendation:** use **file mode** (default) with auto-exclusions for file-level backups; use **disk mode** in a separate job for bare-metal restore (includes everything).

### Security & quality
- Input validation and credential sanitization (secrets redacted from logs)
- Path-traversal prevention
- Retry logic with exponential backoff
- CI gates on every build: tests, `golangci-lint`, `gosec`, `go mod tidy`

### 🔒 Client-side encryption
Backups can be encrypted **on the client** before they leave the machine, with
the same scheme as the official `proxmox-backup-client` (AES-256-GCM, keyed chunk
digests, signed manifest). The PBS server only stores opaque data and never sees
the key — useful on a shared or managed PBS.

- **GUI**: *Servers → Edit → Encryption key* — create a key file or pick an
  existing one (from `proxmox-backup-client key create --kdf none` or a PVE
  storage); its fingerprint is shown. Then keep a copy off the machine:
  **Print (paper key)** saves a printable page with the key and its QR code (the
  format of `proxmox-backup-client key paperkey`), optionally passphrase-protected.
  **Import a key from text or a QR code** turns a scanned QR code or a paper key
  back into a key file.
- **CLI**: `-keyfile path/to/key.json` (or `"keyfile"` in the JSON config); a
  passphrase-protected key takes `-keyfile-passphrase`, or prompts for it.
- **Bare-metal restore**: the patched Clonezilla ISO restores encrypted disk
  backups too (key from a USB stick, with its passphrase if any) — see
  [PATCH-CLONEZILLA.md](PATCH-CLONEZILLA.md).
- **Interoperable**: an encrypted backup restores with `proxmox-backup-client`
  or Proxmox VE using the same key file, and vice versa.

> ⚠️ **Without the key, encrypted backups are unrecoverable.** The first
> encrypted backup uploads everything again (no deduplication with unencrypted
> backups). Backup IDs, archive names and sizes stay visible to the server; file
> contents, file names and the catalog are encrypted.

## 🤖 Unattended deployment (Ansible & IaC)

Nimbus Backup installs and configures entirely from files — no interactive setup.
Two paths, both covered by ready-to-use examples in
[`examples/automation/`](examples/automation/):

- **Command line** — one JSON file per host carries the whole backup
  (`proxmoxbackup-directory.exe --config file.json`); schedule via the Windows
  Task Scheduler. Reliable exit codes (`0` OK, `1` fatal, `2` locked, `3` partial)
  so your orchestrator detects failures. Best for pure infrastructure-as-code.
- **GUI/service (MSI)** — a **single `config.json`** carries the PBS connection,
  backup settings **and** the schedule (`scheduled_jobs`). Push the file, restart
  the `NimbusBackup` service: jobs are reconciled idempotently and `nextRun` is
  computed for you. No timestamp math in your Jinja2 template.

The service reads its configuration from `C:\ProgramData\ProxmoxBackupClient\`
(since 0.4.0; see [Upgrading](#️-upgrading-from--030)).

📖 Full walkthrough: [Automate Windows backup to PBS with Ansible](https://nimbus.rdem-systems.com/en/blog/unattended-windows-backup-ansible/?utm_source=github).

## 🚀 Quick start

1. Download `NimbusBackup.msi` (or the standalone `NimbusBackup.exe`) from the releases
2. Install it / run it with administrator privileges (required for VSS)
3. Configure your PBS connection and test it
4. Select directories to back up
5. Start the backup — or schedule it

### 🔑 PBS user and permissions

The client only needs the **`DatastoreBackup`** role on the target datastore — no admin account:

1. In the PBS UI, create a user (e.g. `nimbus@pbs`) and an API token for it (e.g. `nimbus@pbs!laptop01`).
2. In **Datastore → Permissions** (or **Configuration → Access Control → Permissions**), grant `DatastoreBackup` on `/datastore/<name>` — or on `/datastore/<name>/<namespace>` if you back up into a namespace.
3. **Privilege-separated token** ("Privilege Separation" checked, the default): grant the role to the **token** itself (`nimbus@pbs!laptop01`), not only to the user — the effective rights are the intersection of both. This is the most common cause of "permission denied".

`DatastoreBackup` lets the client create backups and list and restore its own backup groups. Deleting or pruning snapshots needs `DatastorePowerUser`.

## ⬆️ Upgrading from ≤ 0.3.0

Installing 0.4.0 or later over an existing install upgrades it in place (same MSI identity, same `NimbusBackup` service). The data folder moves from `C:\ProgramData\NimbusBackup` to the shared `C:\ProgramData\ProxmoxBackupClient`: on first start, your configuration, scheduled jobs, history and API token are copied over once (existing files are never overwritten). The old folder is kept, marked with `COPIED-TO-ProxmoxBackupClient.txt`, so a downgrade still works. Snapshots taken by older versions can still be restored to their original location.

## 📋 Requirements

- Windows 10/11 or Windows Server (64-bit)
- Administrator rights (for VSS snapshots)
- Network access to a Proxmox Backup Server

## 🔨 Building from source

### Prerequisites
- Go 1.25 or later
- Node.js 20 or later
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0` (the version CI uses)

### Build
```bash
cd gui
npm install --prefix frontend
wails build      # or: wails dev  (hot reload)
```

Or build everything (CLI + GUI + service) with the Makefile: `make install-deps && make` (see `make help`). Windows toolchain and Docker cross-build: [BUILD.md](BUILD.md).

The brand is picked from the executable name: `NimbusBackup.exe` runs as Nimbus Backup, any other name as the neutral "Proxmox Backup Client" ([`gui/brand.go`](gui/brand.go)).

## 🔗 Relationship with upstream

Nimbus Backup started as a fork of [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go) (Proxmox Backup Client in Go, by Tiziano Bacocco, GPLv3), to which we added the Windows GUI, the service, scheduling, multi-PBS and restore. In September 2026, upstream merged that GUI back and made it brand-neutral ("Proxmox Backup Client GUI"). Since 0.4.0, Nimbus Backup is built from the same code base: **the two projects are now functionally almost identical.**

What this repository adds on top of upstream is a small, documented patch series ([`patches/`](patches/README.md)):

- **Fixes not yet merged upstream** (service build, restore with username/password servers, exit codes, log redaction, machine backup reliability…) — sent upstream as they are merged.
- **The Nimbus Backup identity** — `NimbusBackup.exe`/`.msi`, the `NimbusBackup` service, and the MSI upgrade code of existing installs, so they keep upgrading in place.
- **Upgrade path** from Nimbus Backup ≤ 0.3.0 (data-folder migration, legacy snapshot metadata).
- **Release pipeline** — build-provenance attestation, checksums, VirusTotal reports.

We re-merge upstream regularly; upstream's own releases are published at [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go/releases).

## ⚠️ Disclaimer

This software is provided as-is. While we strive for reliability, we take no responsibility for any data loss or damage. Always test your backups and verify restoration before relying on them in production.

This project is **not affiliated** with **Proxmox Server Solutions GmbH**. "Proxmox" and related names are the property of their respective owners and are used here only to state compatibility.

## 📄 License

GPLv3 — see the [LICENSE](LICENSE) file.

## About RDEM Systems

NimbusBackupClient is developed and maintained by [RDEM Systems](https://www.rdem-systems.com/en/?utm_source=github), a French infrastructure provider specialized in Proxmox VE/PBS managed services and NTP/NTS infrastructure. We operate [16 public NTS servers](https://ntp.rdem-systems.com/en/nts.php?utm_source=github) ([live status](https://ntp.rdem-systems.com/en/status.php?utm_source=github); 11 of them are listed in the [community reference](https://github.com/jauderho/nts-servers)), and provide [fully managed PBS hosting](https://nimbus.rdem-systems.com/en/?utm_source=github) for users who don't want to self-host. We also maintain [unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client), signed `proxmox-backup-client` packages for Linux distributions Proxmox doesn't officially support.

---

**© 2024-2026 RDEM Systems and Proxmox Backup Client GO contributors.**
