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
| 0050 | ci: verify the Authenticode signatures of every shipped Windows file | fork-only | Valid + CN=RDEM SYSTEMS + timestamp on the exe, service, MSI and the exes the MSI installs; the build fails otherwise. |
| 0051 | docs(fa): Windows menu names in Persian, English in parentheses | fork-only |  |
| 0052 | ci: install the signed MSI on Windows, fresh and as an upgrade from 0.4.0 | fork-only | Release notes from the CHANGELOG section. |
| 0053 | docs: state the upstream commit 0.4.1 is built on, in every README | fork-only |  |
| 0054 | docs: the GUI path to the encryption key and the Running tab use the real labels | fork-only |  |
| 0055 | fix(vss): a busy VSS is waited for, never wiped; cleanup spares shadows in use | upstream candidate | Inherited: `vssadmin delete shadows /all` + VSS restart on "busy"; startup cleanup deleting another live run's shadow. |
| 0056 | fix(vss): take the in-use lease before the symlink; ci: bounded, diagnosable install test | upstream candidate (VSS part) |  |
| 0057 | ci: quote msiexec arguments in the install test | fork-only |  |
| 0058 | release: v0.4.1 | fork-only |  |
| 0059 | test(e2e): wait until PBS is initialised, not just installed | upstream candidate | `datastore create` panicked on a missing authkey.key when the API answered before the keys existed. |
| 0060 | ci: block the release when Windows Defender flags the build | fork-only | |
| 0061 | fork: restore the Nimbus Backup icon and logo | fork-only | The rebuild brought back upstream's regenerated `gui/build/windows/icon.ico` (an "f"); About showed the placeholder `nimbus.svg`. |
| 0062 | gui: report backend and status messages in the selected language (issue #10) | upstream candidate | Go user-facing messages in English (were French); App.jsx literals through `t()`. |
| 0063 | machinebackup: read VSS snapshot volumes to the device end | upstream candidate | `FSCTL_ALLOW_EXTENDED_DASD_IO`: the NTFS backup boot sector was zero-padded. From CJInvent d496ada. |
| 0064 | gui: send stdlib log output to the rotating log | upstream candidate | The service's `log.Printf` went to the discarded stderr. From CJInvent 729abbd. |
| 0065 | gui: WebView2 user data in Local AppData for new installs | upstream candidate | Roaming failed for domain accounts without a roaming profile; an existing folder is kept (localStorage). From CJInvent 756d388. |
| 0066 | gui: show the VSS-needs-admin warning in standalone mode | upstream candidate | Compared with `'Standalone'`, backend sends `"Standalone Mode"`. Found by CJInvent's GUI audit. |
| 0067 | gui: do not leave ghost tray icons behind | upstream candidate | Quit waits for systray's `onExit` (after `NIM_DELETE`); a crash gets 300 ms. From CJInvent 9a7097e. |
| 0068 | docs: CHANGELOG for the next release (issue #10, icon, CJInvent fixes) | fork-only | `Fixes #10`. |
| 0069 | snapshot: delete our VSS shadow copies after every backup, through the VSS API | upstream (tizbac/proxmoxbackupclient_go@630ed7b) | Taken ahead of the next rebuild; conflict with 0055/0056 resolved by keeping the in-use check. Drop at the next rebuild. |
| 0070 | machinebackup: skip hashing blank (all-zero) chunks | upstream (tizbac/proxmoxbackupclient_go@eae0068) | Drop at the next rebuild. |
| 0071 | fix(sched): honour machine backup kind (host/vm) | upstream (tizbac/proxmoxbackupclient_go@0839562) | Fork: VM ID check on the dedicated VM ID field (0019). Drop at the next rebuild, keep the fork's VM ID field. |
| 0072 | fix(service): Running Jobs, duplicate GUI instances, locale-proof ACLs | upstream (tizbac/proxmoxbackupclient_go@44969cf) | Drop at the next rebuild. |
| 0073 | gui: commit wails-generated SetJobReporter bindings | upstream (tizbac/proxmoxbackupclient_go@07859b0) | Drop at the next rebuild. |
| 0074 | docs: CHANGELOG for the upstream fixes taken into the next release | fork-only | |
| 0075 | docs: official builds are published and signed by RDEM Systems only | fork-only | Notice in every README (EN, FR and the 11 AI translations). |
| 0076 | release: v0.4.2 | fork-only | |
| 0077 | fix(crypto): encrypted GUI backups use the keyed digest; Service Mode reads get the key (issue #13) | upstream candidate (tizbac/proxmoxbackupclient_go PR #115) | `ChunkState` hashed with a bare sha256; the restore ticket carried no key file; token-only error message. |
| 0078 | fix(restore): restore directory modification times (issue #12) | fork-only | Reverted by 0080, superseded by 0083. |
| 0079 | fix(pbs): send the BaseURL host in the upgrade request's Host header | upstream candidate (tizbac/proxmoxbackupclient_go PR #116) | `host:443` defeated Host-routing reverse proxies (upstream issue #89). |
| 0080 | Revert 0078 | fork-only | Drop together with 0078 at the next rebuild. |
| 0081 | installer: SERVICEAUTOSTART flag + document silent-install switches | upstream (tizbac/proxmoxbackupclient_go@af67fc9) | Fork: READMEs kept; the action and DEPLOYMENT.md use the SCM name `$(var.ExeName)` (0006). Keep that part at the next rebuild. |
| 0082 | CLIs: report errors on stderr, never in a message box | upstream (tizbac/proxmoxbackupclient_go@922cba8) | Drop at the next rebuild. |
| 0083 | Restore: put back every file time, also 1970 and earlier, and folder times | upstream (tizbac/proxmoxbackupclient_go@fa06835) | Drop at the next rebuild. |
| 0084 | directorybackup: -no-mtime leaves real times out of the archive | upstream (tizbac/proxmoxbackupclient_go@d7fa828) | Fork: threaded through the exclusion and `-parallel` paths (0040, 0042). |
| 0085 | machinebackup: close the PBS session when a backup fails or is stopped | upstream (tizbac/proxmoxbackupclient_go@9148294) | Drop at the next rebuild. |
| 0086 | pbscommon: give ListSnapshots 120 s instead of 10 s | upstream (tizbac/proxmoxbackupclient_go@29e18e8) | Drop at the next rebuild. |
| 0087 | pxar: leave out "Recovery" only at the root of a volume | upstream (tizbac/proxmoxbackupclient_go@9781273) | Fork: the CLI already set `ExcludeRoot` (0040). |
| 0088 | BMR wizard: remove the backup's snapshot store from restored partitions | upstream (tizbac/proxmoxbackupclient_go@78a0bab) | Drop at the next rebuild. |
| 0089 | machinebackup: send the notification e-mail | upstream (tizbac/proxmoxbackupclient_go@024b55b) | Fork: machinebackuplib var block kept (`errUploadAborted` lives in windows.go). |
| 0090 | Back up and restore Windows creation times | upstream (tizbac/proxmoxbackupclient_go@8503e03) | Drop at the next rebuild. |
| 0091 | Back up and restore alternate data streams and NTFS extended attributes | upstream (tizbac/proxmoxbackupclient_go@d33477b) | Fork: `dist/index.js` kept (rebuilt by wails). |
| 0092 | clonezilla: also search /usr/lib/syslinux/bios for isohdpfx.bin | upstream (tizbac/proxmoxbackupclient_go@8cfdab2) | Drop at the next rebuild. |
| 0093 | clonezilla: read volume id with xorriso, not isoinfo | upstream (tizbac/proxmoxbackupclient_go@cee965c) | Drop at the next rebuild. |
| 0094 | BMR wizard: never offer the boot stick as the restore target | upstream PR (tizbac/proxmoxbackupclient_go #113, mjb-is) | Drop once #113 is merged. |
| 0095 | BMR wizard: refuse a target disk smaller than the backed up disk | upstream PR (tizbac/proxmoxbackupclient_go #113, mjb-is) | Drop once #113 is merged. |
| 0096 | Read unencrypted snapshots with an encryption key loaded | mjb-is (mjb-is/proxmoxbackupclient_go@6d1b303) | Replaces 0077's `legacyEncryptedDigest` with `ChunkMatchesDigest`. Upstream candidate. |
| 0097 | ci: deps/** branches bump Go and x/net, tidy and test every module | fork-only | `deps-bump.yml`: bump with the target toolchain, patch as artifact. |
| 0098 | fix: CI findings on the imported upstream commits and the issue #13 test | upstream candidate (lint and test parts) | Windows-only stream helpers moved out of the linux build; fork wording of the VM ID error in the mail test. |
| 0099 | ci(deps-bump): build through the workspace; vet and tests informational | fork-only |  |
| 0100 | test(e2e): encrypted backup by the GUI/service writer, read back by our readers and the official client | upstream candidate | e2e test 11, `gui/e2e_live_test.go`. |
| 0101 | ci: short PBS hostname in the e2e container; go work edit outside GOWORK=off | upstream candidate (e2e part) | PBS's certificate CN (64 chars max) took the runner's long FQDN. |
| 0102 | test(api): wait for the token file to catch up with the rotated token | upstream candidate | Race in `TestTokenRotatorRotatesOnInterval`. |
| 0103 | ci: actions on Node.js 24 | fork-only (upstream candidate for its workflows) |  |
| 0104 | build: Go 1.26.9 and golang.org/x/net v0.61.0 | upstream candidate | Idea from CJInvent 2cea7a2 / a3fcbed. |
| 0105 | build: remove the unused Dockerfile.gui | upstream candidate | Could not build (copied `gui/` alone). |
| 0106 | pbscommon: keep x/net's http2.Transport for the PBS session, said why | upstream candidate | SA1019 nolint; the move to `http.ClientConn` is its own change. |
| 0107 | gui: the Restore notice lists NTFS permissions, ADS and EAs as supported | upstream candidate |  |
| 0108 | release: v0.4.3 | fork-only |  |
| 0109 | gui: add the two French running-jobs strings | upstream candidate | The French Running tab showed the raw key `noRunningJobs`. |
| 0110 | gui: check that every UI language carries every key | upstream candidate | `scripts/check-translations.mjs` (`npm test`) and a Translations workflow; also fixes `editServer` defined twice and `t()` on empty strings. |
| 0111 | gui: keep a scheduled job's compression on edit, show the saved parallel count | upstream candidate | Editing a job reset its compression; the parallel field always showed 1. |
| 0112 | stats: log the GitHub stars daily and chart them in the README | fork-only | Workflow writing to the `stats` branch. |
| 0113 | stats: chart and log our own stars only | fork-only |  |
| 0114 | gui: stop a directory backup in the middle of a folder on cancel | upstream candidate | Sent as tizbac/proxmoxbackupclient_go PR #117 (pxar hook + GUI). |
| 0115 | gui: run "at machine startup" jobs once per boot, in the service | upstream candidate | The GUI ran them at every launch, past the service's duplicate guard (issue #12). |
| 0116 | gui: build runStartupJobsAtBoot only where the service is built | upstream candidate | Folds into 0115. |
| 0117 | Add Brazilian Portuguese GUI translation | upstream candidate | By Tácio Andrade, PR #16. |
| 0118 | ci: a release no longer waits for the macOS CLI build | fork-only | GitHub's macOS runners can be unavailable for hours. |
| 0119 | docs: credit Tácio Andrade for the Brazilian Portuguese translation | fork-only | README Contributors section. |
| 0120 | release: v0.4.4 | fork-only |  |

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
