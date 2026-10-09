# Fork patches on top of upstream (tizbac/proxmoxbackupclient_go)

This fork (Nimbus Backup, RDEM Systems) and upstream are **sibling repos**:
each side works independently and the fork is periodically rebuilt on
upstream. Since October 2026 the fork is **upstream's `master` plus the patch
series below** (no merge commit): to follow upstream, re-apply the series on
its new `master` (see "Rebuilding on a newer upstream").

Each patch is a regular commit on the fork branch; the `.patch` files in this
folder are the same commits exported with `git format-patch`, kept so the
series can be re-applied (or sent upstream) even if history is rewritten.

- Base: tizbac/master `3c1b989` (2026-10-09)
- Series: `3c1b989..<tip of the fork branch>`

## The series

**Upstream candidates** fix bugs or add features useful to every brand: send
them to tizbac as PRs and drop them here once merged. **Fork-only** patches
carry the Nimbus identity, docs and CI.

| # | Patch | Kind | Why |
|---|---|---|---|
| 0001 | fork: Nimbus Backup docs, changelog and automation examples | fork-only | Fork README (EN/FR), CHANGELOG, TODO, `examples/automation`; upstream-branded translations removed. |
| 0002 | feat(deploy): single-file unattended config with schedule provisioning | upstream candidate | 0.3.0 feature: `scheduled_jobs` in `config.json` upserted into the scheduler store at service start (Windows and systemd), validated like jobs saved from the GUI. |
| 0003 | gui: migrate data from the legacy `ProgramData\NimbusBackup` folder | upstream candidate | Nimbus <= 0.3.0 kept its data there. Service: copied once into `ProgramData\ProxmoxBackupClient` (no-clobber, markers). Standalone GUI: also a source for the home-dir migration. |
| 0004 | gui: keep reading the legacy `.nimbus_backup_meta.json` sidecar | upstream candidate | Snapshots taken by Nimbus <= 0.3.0 keep "restore to original location". |
| 0005 | gui: pass the PBS ticket to the restore readers | upstream candidate | Username/password servers could list snapshots but not restore or browse them. |
| 0006 | gui, installer: keep the service name the MSI registers | **fork invariant** | The SCM name is the brand's ExeName (`NimbusBackup`, the name every Nimbus install has), not `<ExeName>SVC`: `ProductBody.wxi` registers `$(var.ExeName)` and `serviceIdentityForExeBase` strips the `SVC` suffix (tests updated). |
| 0007 | gui: do not report a cancelled multi-folder backup as a success | upstream candidate | Cancelling between folders announced "completed". |
| 0008 | pbscommon: redact the CSRF token in logged request headers | upstream candidate | Header key canonicalization made the redaction miss. |
| 0009 | installer: delete the real data directory on "delete configuration" | upstream candidate | Uninstall removed the wrong folder (credentials left behind). |
| 0010 | gui: per-language buy-storage link and label for brands | upstream candidate | Localized landing pages per brand. |
| 0011 | installer: WiX icon ids WiX 3 accepts | upstream candidate | Shortcut icon Id needs the `.ico` extension; Programs & Features icon through `ARPPRODUCTICON`. |
| 0012 | gui: never split a machine backup into one snapshot per disk | upstream candidate | A "vm" snapshot's backup-id is the numeric VM ID; a per-disk split broke it. |
| 0013 | fork: real Nimbus Backup brand identity | fork-only | `NimbusBackup.wxs` keeps UpgradeCode `12345678-…`, RDEM Systems, old HKCU key. |
| 0014 | fork: build and release the Nimbus Backup brand in CI | fork-only | Fork `build-and-release.yml`. |
| 0015 | ci: fork build workflow; run the PBS end-to-end suite on fork branches | fork-only | Fork CI (master pushes, gating module tests, VM metadata tests) and upstream's real-PBS e2e suite on fork branches. |
| 0016 | fix(pbs): show PBS's reason when it refuses the backup session | upstream candidate | Every upgrade refusal read "authentication failed" (e.g. a 400 "backup owner check failed"); upstream's namespace-not-found error is kept. |
| 0017 | fork: version 0.4.0 base and release notes | fork-only | Fork version number in `gui/wails.json`. |
| 0018 | feat(machine): generate the PVE VM config from the real machine | upstream candidate | CPUs, RAM, firmware, OS type, NICs with MACs, SMBIOS, boot disk; keeps upstream's `#qmdump#map` lines, `cache=writeback`, OVMF for a GPT boot disk. |
| 0019 | fix(gui): dedicated Proxmox VM ID for "vm" machine backups | upstream candidate | "VM" mode reused the hostname-filled Backup ID field. |
| 0020 | feat(crypto): printable paper key with QR code | upstream candidate | `proxmox-backup-client key paperkey` equivalent (`rsc.io/qr`), `ProtectKeyConfig`, paper-key text accepted by `ParseKeyConfig` / `LoadKeyConfig`. |
| 0021 | feat(gui): paper key, key QR code and key import from text | upstream candidate | Print / show the QR code next to the key field; rebuild a key file from a scanned QR code or a paper key (protected keys unlocked). |
| 0022 | docs: encryption, paper key and upstream re-merge in README and CHANGELOG | fork-only | |
| 0023 | fix: review findings on the upstream rebuild | upstream candidate | `EncryptionKeyField` awaited the Wails Promise (upstream bug: fingerprint never shown); OVMF follows the boot disk; a changed provisioned schedule recomputes nextRun; a retired legacy folder is never a migration source. |
| 0024 | feat(crypto): compressed encrypted chunks (ENCR_COMPR), both ways | upstream candidate | Upstream rejected ENCR_COMPR blobs, which `proxmox-backup-client` writes by default (official encrypted backups could not be restored), and uploaded encrypted chunks uncompressed. Golden vector from libzstd + OpenSSL. |
| 0025 | ci: upstream's real-PBS e2e suite gates the release, both interop directions | fork-only (test 9 + readback helper: upstream candidate) | `e2e.yml` called by `build-and-release.yml`, release needs it; test 9 reads back an official encrypted block backup with `machinebackup/readback`. |
| 0026 | ci: run upstream's make lint over every module (informational) | fork-only | |
| 0027 | test: framing test uploads uncompressed; e2e test 9 path works with a local client | upstream candidate | |
| 0028 | ci: sign the Windows GUI, service and MSI with Azure Artifact Signing | fork-only | Account `github-nimbus`; active once `AZURE_SIGNING_PROFILE` is set. |
| 0029 | fix(ci): gofmt-align ScheduledJob, gosec G703 annotations, failures as annotations | fork-only (gofmt + G703: upstream candidate) | Upstream code failed gofmt and gosec; e2e failures surface as job annotations. |
| 0030 | fix(ci): gosec G703 on upstream file operations; e2e log tail as sub-4KB annotations | fork-only (G703: upstream candidate) | |
| 0031 | fix: real file owners in pxar on Unix; upstream lint findings | upstream candidate | pxar hardcoded uid/gid 1000: restoring as another user failed ("failed to set ownership"), restoring as root gave files to uid 1000. `serviceIdentity` unused outside the service build; ST1020 comment. |
| 0032 | ci: run Build GUI in the 'signing' environment | fork-only | One Azure federated credential (`repo:rdemsystems/NimbusBackupClient:environment:signing`) for every branch and tag. |
| 0033 | docs: PBS user, token and DatastoreBackup permissions | fork-only | |
| 0034 | ci: lint-all lists every finding, per module, as annotations | fork-only |  |
| 0035 | ci: lint-all strips colours and hides raw output from the Go problem matcher | fork-only |  |
| 0036 | pbscommon: check (discard) Close and Write errors flagged by errcheck | upstream candidate |  |
| 0037 | fix(gui): maximise fills the screen (issue #1) | upstream candidate | MaxWidth/MaxHeight 1680x1008 capped the maximised window; startup window shrunk to fit small screens instead. |
| 0038 | ci: lint-all reads golangci-lint JSON reports; pbscommon errcheck fixes | fork-only (pbscommon part: upstream candidate) | The upgrade request write error is now returned. |
| 0039 | fix(service): honour the selected PBS server and compression (issue #2) | upstream candidate | The /backup route swapped PBS id and compression; the service ignored the selected PBS. |
| 0040 | feat(cli): exclusions in proxmoxbackup-directory (issue #4) | upstream candidate | `-exclude`, `-exclude-from`, `"exclude"`/`"exclude-from"`. |
| 0041 | fix: the 67 golangci-lint findings outside gui | upstream candidate | Includes the service backup goroutines' context leak (lostcancel). |
| 0042 | feat(cli): -parallel N backs up several directories at once (issue #6) | upstream candidate | e2e test 10 (parallel + exclusions). |
| 0043 | feat(gui): back up several folders in parallel (issue #6) | upstream candidate | Config `parallel`, one lock and one VSS set per batch; Linux one snapshot per device. |
| 0044 | feat(vss): one snapshot for a whole multi-folder backup, one shadow copy per volume | upstream candidate | Sequential runs too; per-folder fallback when the set cannot be taken. |
| 0045 | docs, gui: encryption wording for the release notes | fork-only (import warning: upstream candidate) |  |
| 0046 | docs: README in upstream's 11 other languages (AI translations) | fork-only |  |
| 0047 | docs: credit the client-side encryption to upstream in every README | fork-only |  |
| 0048 | docs: 0.4.1 is signed; issue #9 in full; README features and upstream section | fork-only | Also removes CHANGELOG duplicates a sed in 0039 had inserted under every released "Fixed" section. |
| 0049 | docs: carry the README update into the 11 AI translations | fork-only |  |

Dropped when rebuilding on `3c1b989` because upstream fixed them: the old 0003
(service build, `/backup/machine`), 0006 (CLI exit code), the build-break parts
of 0011, the reader/uploader handoff and early VM ID check of 0012, 0013
(lint), and the fork's own client-side encryption (replaced by upstream's).

## Rebuilding on a newer upstream

```bash
git fetch https://github.com/tizbac/proxmoxbackupclient_go.git master
git switch -c merge-upstream-YYYY-MM FETCH_HEAD
git am -3 patches/*.patch   # resolve, or `git am --skip` a patch upstream made obsolete
```

Pushing a `merge-upstream-*` branch runs the full build and tests and the e2e
suite (no release). Hot spots:

- `README.md`, `README.fr.md`, `CHANGELOG.md`: keep ours; delete any new
  upstream `README.<lang>.md`.
- `gui/wails.json`: upstream identity, **our** version number.
- `.github/workflows/build-and-release.yml`: ours. `release.yaml`: the tag
  trigger stays disabled.
- `installer/wix/NimbusBackup.wxs` and the Nimbus entry of `gui/brand.go`: ours.

Invariants to re-check after every rebuild (an upgrade breaks if one changes):

- MSI UpgradeCode of the Nimbus build = `12345678-1234-1234-1234-123456789012`.
- The GUI exe is installed as `NimbusBackup.exe` (brand key `nimbusbackup`).
- The Windows service is registered as `NimbusBackup` (patch 0006).
- The data directory still migrates from `ProgramData\NimbusBackup` (0003).
- CI gates pass: tests, `golangci-lint`, `gosec`, `go mod tidy`, the Windows
  and `-tags service` builds of the GUI, the MSI build, the e2e suite.

Regenerate the `.patch` files after changing the series:

```bash
git rm -q patches/*.patch
git format-patch -o patches/ <upstream base>..HEAD -- . ':!patches'
```

## Known gaps not patched (yet)

- **PBS passwords stored in clear text** in `config.json` (username/password
  login). The service hardens the file to SYSTEM-only; DPAPI would be better.
- **The GUI cannot use a passphrase-protected key file** (no console to
  prompt on): import it with "Import a key from text or a QR code", which
  writes an unprotected copy.
- **Worker error handling in `machinebackuplib.uploadWorker`**: the fork's
  fail-once dispatcher (no shared error variable, no dispatcher stuck after a
  worker error) was not carried over onto upstream's rewritten worker. To redo
  as an upstream PR.
- **Ticket expiry during long multi-folder backups** (PBS tickets live ~2 h).
- Upstream's own `Product.wxs` uses the same UpgradeCode as historical Nimbus
  MSIs, so installing upstream's MSI replaces a Nimbus install.
