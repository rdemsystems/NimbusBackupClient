# Proxmox Backup Client — Status & notes

> Per-version changes are listed in the “Changes since…” section of each release (above) and in [CHANGELOG.md](CHANGELOG.md). This page describes the **stable** state of the product.

## 📦 Available builds

### NimbusBackup.msi (installer — recommended for production)
- ✅ **Windows service**: starts automatically at system boot
- ✅ **Persistent admin privileges**: the service runs as LocalSystem (VSS guaranteed)
- ✅ **Scheduled backups**: run automatically, even after a reboot
- ✅ **Clean uninstall**: full cleanup via Control Panel

### NimbusBackup.exe (standalone)
- ✅ **Manual and scheduled backups**: work as long as the app is running
- ❌ **No persistence across reboots**: no service → prefer the MSI in production
- 💡 **Use case**: one-off backups or testing

## ✅ Features

### Backup & restore
- One-shot (immediate) and scheduled (configurable time) backups
- **Split a first backup** of a large volume into smaller, resumable backups (opt-in)
- **VSS** (Volume Shadow Copy) for consistent backups: one snapshot for a whole multi-folder backup, one shadow copy per volume
- **Several folders in parallel** (no maximum; recommended: CPUs / 4)
- **Client-side encryption** compatible with `proxmox-backup-client` (developed upstream by Tiziano Bacocco), paper key with QR code
- File/folder exclusions + **automatic exclusion of Windows system folders** (System Volume Information, $RECYCLE.BIN, pagefile.sys…)
- Snapshot restore and browsing (**fast catalog reads**)
- Robust long-running backups (**30 s keep-alive**, validated on 11 h+ backups)
- **Multi-server PBS support**

### Interface & configuration
- Wails GUI (Go + React), **in English, French, Italian, German and Polish**
- Backup history and re-run of failed jobs
- Progress bar with statistics, minimize to tray
- PBS configuration with connection test, **certificate fingerprint pinning (TOFU)** and namespaces

## 📌 Known issues
- ℹ️ **Signed since 0.4.1**: the GUI, its service and the MSI are signed by RDEM SYSTEMS; SmartScreen may still warn until the reputation of a new release is established. The command-line tools are not signed yet.
- ℹ️ **Upgrading from <= 0.3.0**: the configuration folder is merged from `ProgramData\NimbusBackup` into `ProgramData\ProxmoxBackupClient` on first start (copied once, nothing overwritten, old folder kept).
- ⚠️ The **standalone .exe** does not persist across reboots → use the **MSI** in production.
- ⚠️ The exclusion format is not validated on input.
