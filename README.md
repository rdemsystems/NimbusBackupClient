# Nimbus Backup — Windows client for Proxmox Backup Server

🇬🇧 English | [🇫🇷 Français](README.fr.md)

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rdemsystems/NimbusBackupClient)](https://github.com/rdemsystems/NimbusBackupClient/releases)
[![Documentation](https://img.shields.io/badge/docs-nimbus.rdem--systems.com-orange)](https://nimbus.rdem-systems.com/en/?utm_source=github)

**Nimbus Backup is an open-source (GPL-3.0) Windows backup client for Proxmox Backup Server (PBS).**
A modern GUI to back up Windows servers and workstations to PBS — VSS-consistent snapshots, scheduled jobs, file and disk modes, snapshot browsing and restore, multi-PBS support, and a Windows service. Looking for **offsite, immutable** PBS storage without self-hosting? See the [managed service](#️-managed-pbs-offsite--immutable) below.

> Keywords: proxmox backup client windows · PBS client · Windows VSS backup · offsite immutable backup · proxmox backup server GUI.

## 📦 Download

👉 **[Download the latest release](https://github.com/rdemsystems/NimbusBackupClient/releases)**

> ⚠️ **Windows says "virus detected" (e.g. `Trojan:Win32/Sabsik.FL.A!ml`) or shows a SmartScreen warning?**
> This is a known **false positive** for Go/Wails applications — it is *not* a virus. The `!ml` suffix means it comes from a machine-learning model that flags *unsigned, low-prevalence* executables.
> Read [why this happens and how to verify the download](https://nimbus.rdem-systems.com/en/antivirus-false-positive/?utm_source=github).

### 🔎 Verify any download

Every release ships SHA-256 checksums and a signed **build-provenance attestation** (cryptographic proof the binary was produced by this repo's CI, from a specific commit):

```powershell
Get-FileHash .\NimbusBackup.exe -Algorithm SHA256   # compare against SHA256SUMS.txt
gh attestation verify .\NimbusBackup.exe --repo rdemsystems/NimbusBackupClient
```

**VirusTotal — 0 detections.** Independent multi-engine reports for recent MSI installers:
[0.2.108](https://www.virustotal.com/gui/file/6e8fb7ce9af740d470e947addb8daba4331c0b88e8bfdec9e0697ea8f7f29e9e/detection) ·
[0.2.107](https://www.virustotal.com/gui/file/6fd6c6fa77e0305c129ef882a3745100aa6033187a6d52a4af94149ab6b666d2/detection) ·
[0.2.106](https://www.virustotal.com/gui/file/ad6e56700ed9df8e088906e38cee2e2882fc7045f4e39269de0e379a01784ad7/detection)

> ℹ️ **Code signing:** Windows binaries are **not yet Authenticode-signed** (an OSS certificate via [SignPath Foundation](https://signpath.org) is pending). Until then, provenance is established via the attestation and checksums above.

## ☁️ Managed PBS (offsite & immutable)

Don't want to self-host Proxmox Backup Server? Use our fully managed, **offsite immutable** PBS datastores:
👉 **[Configure your backup & see pricing](https://nimbus.rdem-systems.com/en/choose-backup/?utm_source=github)**

- ✅ From €12/TB/month
- ✅ 1 TB free trial
- ✅ [NimbusBackup — Managed PBS hosting in France](https://nimbus.rdem-systems.com/en/?utm_source=github)

## 📚 Documentation

- **Complete Proxmox Backup guide** — PBS deployment best practices ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/complete-proxmox-backup-guide/?utm_source=github))
- **Back up Windows with Proxmox Backup Server** — Windows-specific deployment guide ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github))
- **PBS vs Veeam** — Proxmox Backup Server comparison ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/pbs-vs-veeam-proxmox-backup-comparison/?utm_source=github))

## ✨ Features

### GUI interface (recommended)
- **🌍 Multi-language** — English & French interface
- User-friendly configuration with connection testing
- Real-time backup progress with speed and ETA
- VSS (Volume Shadow Copy) support for consistent backups
- Multi-folder backup, file and disk modes
- Snapshot browsing, file search (wildcards) and restore
- Multi-PBS server support, certificate fingerprint pinning (TOFU)
- Windows service mode + scheduled backups
- Debug logging for troubleshooting

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
- Input validation and credential sanitization
- Path-traversal prevention
- Retry logic with exponential backoff
- Comprehensive error handling and tests, 100% lint compliance

## 🚀 Quick start

1. Download `NimbusBackup.exe` (or the `.msi`) from releases
2. Run with administrator privileges (required for VSS)
3. Configure your PBS connection and test it
4. Select directories to back up
5. Start the backup

## 📋 Requirements

- Windows 10/11 (64-bit)
- Administrator rights (for VSS snapshots)
- Network access to a Proxmox Backup Server

## 🔨 Building from source

### Prerequisites
- Go 1.22 or later
- Node.js 20 or later
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Build
```bash
cd gui
npm install --prefix frontend
wails build      # or: wails dev  (hot reload)
```

## 📝 Source project

This project is a fork of [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go), enhanced with a modern GUI and additional features for Windows users.

**Original:** Proxmox Backup Client in Go · **Author:** Tiziano Bacocco (tizbac) · **License:** GPLv3

| Feature                 | tizbac/proxmoxbackupclient_go | NimbusBackupClient (this fork) |
|-------------------------|:-----------------------------:|:------------------------------:|
| CLI mode                | ✅                             | ✅                              |
| Wails GUI               | ❌                             | ✅                              |
| Multi-language (FR/EN)  | ❌                             | ✅                              |
| Real-time progress      | ❌                             | ✅                              |
| Smart system exclusions | ❌                             | ✅                              |
| Multi-PBS support       | ❌                             | ✅                              |
| CI/CD pipelines         | ❌                             | ✅                              |
| Comprehensive tests     | ❌                             | ✅                              |

## ⚠️ Disclaimer

This software is provided as-is. While we strive for reliability, we take no responsibility for any data loss or damage. Always test your backups and verify restoration before relying on them in production.

## 📄 License

GPLv3 — see the [LICENSE](LICENSE) file.

## About RDEM Systems

NimbusBackupClient is developed and maintained by [RDEM Systems](https://www.rdem-systems.com/), a French infrastructure provider specialized in Proxmox VE/PBS managed services and NTP/NTS infrastructure. We operate [11 public NTS servers](https://github.com/jauderho/nts-servers) listed in the community reference, and provide [fully managed PBS hosting](https://nimbus.rdem-systems.com/en/?utm_source=github) for users who don't want to self-host.

---

**© 2024-2026 RDEM Systems. All rights reserved.**
