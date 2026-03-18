# Changelog

All notable changes to Nimbus Backup (GUI) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- System tray icon and background service
- Automatic scheduling (daily, weekly, custom cron)
- Windows service installation
- Notification system (Windows toast)
- Machine backup (full disk with PhysicalDrive - requires code signing)

## [0.1.0] - 2026-03-18

### Added
- Initial Wails v2 GUI with React frontend
- PBS server configuration interface
- Directory backup mode with multi-folder support (one per line)
- Real-time backup progress with accurate percentage
- Background directory size calculation for precise ETA
- Professional progress display (speed, elapsed time, ETA)
- Granular progress updates (every 10 MB)
- VSS (Volume Shadow Copy) support with admin privilege detection
- Snapshot listing and restore functionality
- PBS connection test with real authentication
- Automatic hostname detection for backup-id
- Debug logging to %APPDATA%\NimbusBackup\debug.log
- Crash reporting system
- RDEM Systems branding with custom icon

### Technical
- Inline backup implementation (no external binaries)
- PXAR archive format support
- Chunk deduplication with SHA256
- Dynamic index creation (DIDX)
- HTTP/2 protocol for PBS communication
- Cross-platform build support (Windows primary)

### Known Issues
- Machine backup disabled due to Windows Defender false positive (PhysicalDrive syscalls)
- Requires code signing certificate for full disk backup feature

---

## Version Numbering

- **Major.Minor.Patch** (Semantic Versioning)
- Major: Breaking changes
- Minor: New features, backwards compatible
- Patch: Bug fixes, small improvements

## Links

- [Original CLI Project](https://github.com/tizbac/proxmoxbackupclient_go)
- [RDEM Systems](https://rdem-systems.com)
- [Backup Portal](https://nimbus.rdem-systems.com)
