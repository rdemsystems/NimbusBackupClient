# Fork patches on top of upstream (tizbac/proxmoxbackupclient_go)

This fork (Nimbus Backup, RDEM Systems) and upstream are **sibling repos**:
each side works independently and we re-merge upstream from time to time.
Upstream re-merged our GUI in September 2026 and moved it to a
brand-neutral identity ("Proxmox Backup Client", exe-name based branding).
The patches below are what the fork needs on top of upstream so that
**existing Nimbus Backup installs (<= 0.3.0) upgrade in place** and the
fork keeps shipping the Nimbus brand.

Each patch is a regular commit on the fork branch; the `.patch` files in this
folder are the same commits exported with `git format-patch`, kept so the
series can be re-applied (or sent upstream) even if history is rewritten.

- Base merge: `c90b4b9` — *Merge tizbac/master (94e8f40) into the Nimbus fork*
- Series: `c90b4b9..<tip of the fork branch>`

## The series

**Upstream candidates** fix bugs that affect every brand, upstream's own
included. Send them to tizbac as PRs; once merged upstream, drop them here.
**Fork-only** patches carry the Nimbus identity and stay in the fork.

| # | Patch | Kind | Why an upgrade needs it |
|---|---|---|---|
| 0001 | gui: migrate data from the legacy `ProgramData\NimbusBackup` folder | upstream candidate | The data dir became `ProgramData\ProxmoxBackupClient` with no migration: upgraded installs lost their config and scheduled jobs. Copies config/jobs/history/token once, never overwrites (no-clobber), retries if incomplete, keeps the old folder and leaves a `COPIED-TO-ProxmoxBackupClient.txt` marker in it (copy never re-run afterwards). |
| 0002 | gui: keep reading the legacy `.nimbus_backup_meta.json` sidecar | upstream candidate | The sidecar was renamed; snapshots taken by Nimbus <= 0.3.0 lost "restore to original location". |
| 0003 | gui: fix the `-tags service` build and machine backups via the service | upstream candidate | Service build did not compile (removed `BackupDirs` field, missing `StartMachineBackup`); `/backup/machine` was never routed (404); scheduled machine jobs walked `\\.\PhysicalDriveN` as a directory. |
| 0004 | gui: pass the PBS ticket to the restore readers | upstream candidate | Username/password servers could list snapshots but not browse/search/restore them. |
| 0005 | gui, installer: keep the service name the MSI registers | upstream candidate | Service registered as `<ExeName>` (e.g. `NimbusBackup`, the name every Nimbus install has) instead of `<ExeName>SVC`; Go derives the same name, so `-service start/stop/uninstall` works. |
| 0006 | machinebackup: exit non-zero when the backup fails | upstream candidate | Fork audit V-1: the CLI exited 0 on failure, so schedulers/RMM reported success. |
| 0007 | gui: do not report a cancelled multi-folder backup as a success | upstream candidate | Cancelling between folders announced "completed". |
| 0008 | pbscommon: redact the CSRF token in logged request headers | upstream candidate | Header key canonicalization made the redaction miss; the token was logged in clear. |
| 0009 | installer: delete the real data directory on "delete configuration" | upstream candidate | Uninstall removed `ProgramData\$(ExeName)` while the data lives in `ProgramData\ProxmoxBackupClient` (credentials left behind). |
| 0010 | gui: per-language buy-storage link and label for brands | upstream candidate | Lets a brand send FR/EN users to localized pages (the fork's pre-merge GUI did). |
| 0011 | gui, installer: fix upstream build breaks (service build, MSI) | upstream candidate | `PhysicalDiskInfo` only existed in the `!service` build (service build failed); the per-brand WiX files could not produce an MSI (`ProductBody.wxi` lacked the `<Include>` root WiX 3 requires, invalid `Product/@Icon`, shortcut icon Id without `.ico`). |
| 0012 | machinebackuplib: fix the reader/uploader handoff and vm backup-id checks | upstream candidate | Raw file/device backups hung forever even on success; a reader error could commit a partial index; multi-disk machine backups were split into non-numeric IDs; the numeric VM ID was only checked after uploading every disk. |
| 0013 | gui, snapshot: fix the CI lint findings and tidy snapshot/go.mod | upstream candidate | golangci-lint (staticcheck "all") failed on upstream code (ST1005, 3× QF1001), which blocks the CI build job; `snapshot/go.mod` was not tidy. |
| 0014 | fork: real Nimbus Backup brand identity | **fork-only** | Replaces the `nimbus.example` placeholders; `NimbusBackup.wxs` keeps UpgradeCode `12345678-…` (the code every Nimbus MSI shipped with), RDEM Systems as manufacturer, old HKCU key. |
| 0015 | fork: build and release the Nimbus Backup brand in CI | **fork-only** | Fork copy of `build-and-release.yml`: builds `NimbusBackup.exe`/`NimbusBackupSVC.exe`/`NimbusBackup.msi` from `NimbusBackup.wxs`, Nimbus release notes, Wails CLI 2.13.0; disables upstream's goreleaser tag trigger. |

## Re-merging upstream later

```bash
git fetch https://github.com/tizbac/proxmoxbackupclient_go.git master
git checkout -b merge-upstream-YYYY-MM <fork branch>
git merge FETCH_HEAD
```

Conflict hot spots and how to resolve them:

- `README.md`, `README.fr.md` — keep ours (fork identity).
- `gui/wails.json` — keep upstream's neutral identity; keep **our** version
  number (CI stamps the Nimbus identity at build time, patch 0015).
- `.github/workflows/build-and-release.yml` — keep ours, then port any
  genuinely new upstream step (upstream's copy is our file with names swapped).
  Pushing a `merge-upstream-*` branch runs the full build + tests (no release).
- `.github/workflows/release.yaml` — keep the tag trigger disabled.
- `installer/wix/NimbusBackup.wxs`, Nimbus entry of `gui/brand.go` — keep ours.
- Any upstream patch listed above as merged upstream: take upstream's version.

Invariants to re-check after every merge (an upgrade breaks if one changes):

- MSI UpgradeCode of the Nimbus build = `12345678-1234-1234-1234-123456789012`.
- The GUI exe is installed as `NimbusBackup.exe` (brand key `nimbusbackup`).
- The data directory still migrates from `ProgramData\NimbusBackup`
  (`gui/legacy_datadir.go`), and the uninstall cleanup removes the real one.
- `-tags service` still compiles (upstream CI does not build it).
- The Windows service is still registered as `NimbusBackup`.
- CI gates pass on the merged tree: `go mod tidy -diff` clean in every module,
  `golangci-lint run` (gui, Linux) and `gosec -severity high -confidence high`
  (gui) report 0 issues, `GOOS=windows go build` and `go build -tags service`
  of gui succeed.

Regenerate the `.patch` files after changing the series:

```bash
rm patches/*.patch
git format-patch -o patches/ c90b4b9..HEAD -- . ':!patches'
```

## Known gaps not patched (yet)

- **PBS passwords stored in clear text** in `config.json` (username/password
  login, new upstream). `0600` does nothing on Windows; the folder's ACLs decide
  who can read it. Proposal: DPAPI (machine scope) encryption, upstream.
- **Data folder ACLs**: `ProgramData\ProxmoxBackupClient` (like the old
  `NimbusBackup` folder) only inherits the default ProgramData ACLs, which let
  local users read files created there — including `config.json` and
  `api-token`. Same exposure as before the merge; fix by setting an explicit
  DACL (SYSTEM + Administrators) on the folder, in the MSI or at creation.
- **Machine backups need a numeric backup ID** (the PBS VM ID). The GUI still
  defaults to the hostname; patch 0012 now fails fast with a clear message
  instead of after uploading every disk, but the UI should ask for a number.
- **Ticket expiry during long multi-folder backups**: a ticket is minted once
  per operation and PBS tickets live ~2 h; a folder started after that fails.
  An already-open backup session is not affected.
- WebView2 profile moved to `%APPDATA%\ProxmoxBackupClient`: the UI language
  choice is re-detected from the OS once after the upgrade (cosmetic).
- The GUI strings (i18n) say "Proxmox Backup Client" in a few places
  (e.g. `appTitle` when the brand is the default); the Nimbus build shows the
  brand title instead, but some texts remain neutral.
- Upstream's own `Product.wxs` (Proxmox Backup Client brand) uses the same
  UpgradeCode as historical Nimbus MSIs, so installing upstream's MSI replaces
  a Nimbus install. Worth a fresh code upstream.
