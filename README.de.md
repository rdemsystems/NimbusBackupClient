# Nimbus Backup — Windows-Client für Proxmox Backup Server

[🇬🇧 English](README.md) · [🇫🇷 Français](README.fr.md) · [🇮🇹 Italiano](README.it.md) · 🇩🇪 Deutsch · [🇪🇸 Español](README.es.md) · [🇷🇺 Русский](README.ru.md) · [🇨🇳 中文](README.zh.md) · [🇯🇵 日本語](README.ja.md) · [🇬🇷 Ελληνικά](README.el.md) · [🇷🇴 Română](README.ro.md) · [🇸🇪 Svenska](README.sv.md) · [🇸🇦 العربية](README.ar.md) · [🇮🇷 فارسی](README.fa.md)

> 🤖 Diese Übersetzung wurde von einer KI aus der englischen README erstellt. Bei Abweichungen gilt die [englische Version](README.md).
> *AI-generated translation of the English README; the [English version](README.md) is authoritative.*

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rdemsystems/NimbusBackupClient)](https://github.com/rdemsystems/NimbusBackupClient/releases)
[![Documentation](https://img.shields.io/badge/docs-nimbus.rdem--systems.com-orange)](https://nimbus.rdem-systems.com/en/?utm_source=github)

**Nimbus Backup ist ein quelloffener (GPL-3.0) Windows-Backup-Client für Proxmox Backup Server (PBS).**
Eine moderne grafische Oberfläche, um Windows-Server und -Arbeitsplätze auf PBS zu sichern — VSS-konsistente Snapshots, geplante Jobs, Datei- und Datenträgermodus, Durchsuchen und Wiederherstellen von Snapshots, Multi-PBS-Unterstützung und ein Windows-Dienst — dazu Kommandozeilen-Tools für Verzeichnis- und Komplettsicherungen ganzer Maschinen. Sie suchen **externen, unveränderlichen** PBS-Speicher, ohne ihn selbst zu betreiben? Siehe den [verwalteten Dienst](#️-verwaltetes-pbs-offsite--unveränderlich) weiter unten.

Nimbus Backup ist der Build von RDEM Systems von [Proxmox Backup Client GO](https://github.com/tizbac/proxmoxbackupclient_go): Die von uns entwickelte GUI wurde upstream übernommen, und beide Projekte teilen sich nun dieselbe Codebasis (siehe [Beziehung zum Upstream-Projekt](#-beziehung-zum-upstream-projekt)).

📖 **Vollständige Dokumentation, Installationsanleitung und PBS-Hosting:** [nimbus.rdem-systems.com](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github&utm_medium=readme&utm_campaign=nbc-readme-top)

## 📦 Download

👉 **[Neueste Version herunterladen](https://github.com/rdemsystems/NimbusBackupClient/releases)**

Jede Version enthält:
- `NimbusBackup.msi` — Installationsprogramm (GUI + Windows-Dienst), **für den Produktivbetrieb empfohlen**
- `NimbusBackup.exe` — eigenständige GUI
- `nimbus-backup-cli-<version>-windows.zip` / `-linux.tar.gz` / `-macos.tar.gz` — Kommandozeilen-Tools
- `SHA256SUMS.txt` — Prüfsummen

> ⚠️ **Windows meldet „Virus erkannt“ (z. B. `Trojan:Win32/Sabsik.FL.A!ml`) oder zeigt eine SmartScreen-Warnung?**
> Dies ist ein bekannter **Fehlalarm** bei Go/Wails-Anwendungen — es ist *kein* Virus. Das Suffix `!ml` bedeutet, dass die Meldung von einem Machine-Learning-Modell stammt, das *wenig verbreitete* ausführbare Dateien markiert (und, bis 0.4.0, unsignierte).
> Lesen Sie, [warum das passiert und wie Sie den Download überprüfen](https://nimbus.rdem-systems.com/en/antivirus-false-positive/?utm_source=github).

### 🔎 Jeden Download überprüfen

Jede Version enthält SHA-256-Prüfsummen und eine signierte **Build-Provenienz-Attestierung** (kryptografischer Nachweis, dass die Binärdatei von der CI dieses Repositorys aus einem bestimmten Commit erzeugt wurde):

```powershell
Get-FileHash .\NimbusBackup.msi -Algorithm SHA256   # compare against SHA256SUMS.txt
gh attestation verify .\NimbusBackup.msi --repo rdemsystems/NimbusBackupClient
```

**VirusTotal.** Die Release Notes jeder Version verlinken den VirusTotal-Bericht des Installationsprogramms dieses Builds (nur veröffentlicht, wenn der Scan sauber ist). Frühere Berichte, 0 Erkennungen:
[0.2.108](https://www.virustotal.com/gui/file/6e8fb7ce9af740d470e947addb8daba4331c0b88e8bfdec9e0697ea8f7f29e9e/detection) ·
[0.2.107](https://www.virustotal.com/gui/file/6fd6c6fa77e0305c129ef882a3745100aa6033187a6d52a4af94149ab6b666d2/detection) ·
[0.2.106](https://www.virustotal.com/gui/file/ad6e56700ed9df8e088906e38cee2e2882fc7045f4e39269de0e379a01784ad7/detection)

> 🔏 **Codesignierung:** Seit 0.4.1 sind `NimbusBackup.exe`, sein Dienst und `NimbusBackup.msi` **von RDEM SYSTEMS Authenticode-signiert** (Azure Artifact Signing); *Eigenschaften → Digitale Signaturen* zeigt den Herausgeber. SmartScreen kann bei einem neuen Release weiterhin warnen, bis dessen Reputation aufgebaut ist: Prüfen Sie, dass der Herausgeber RDEM SYSTEMS ist, dann *Weitere Informationen → Trotzdem ausführen*. Die Kommandozeilenwerkzeuge sind noch nicht signiert; die oben genannte Build-Provenienz-Attestierung und die Prüfsummen decken jede Datei ab.

### 🐧 Unter Linux? Verwenden Sie den offiziellen Client

Die Nimbus-Backup-GUI ist nur für Windows verfügbar (die CLI-Tools lassen sich auch für Linux und macOS bauen). Für Sicherungen auf Dateiebene unter Linux verwenden Sie Proxmox' eigenen `proxmox-backup-client` — wir paketieren ihn für die Distributionen, die Proxmox nicht abdeckt:

👉 **[rdemsystems/unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)** — signierte Paket-Repositorys für Debian/Ubuntu, Fedora/RHEL/Rocky/AlmaLinux, Arch und Alpine (amd64 & arm64). Die offizielle statische Binärdatei von Proxmox, unverändert neu paketiert — weder gepatcht noch neu kompiliert.

## ☁️ Verwaltetes PBS (offsite & unveränderlich)

Sie möchten Proxmox Backup Server nicht selbst betreiben? Nutzen Sie unsere vollständig verwalteten, **externen und unveränderlichen** PBS-Datastores:
👉 **[Sicherung konfigurieren & Preise ansehen](https://nimbus.rdem-systems.com/en/choose-backup/?utm_source=github)**

- ✅ Ab 12 €/TB/Monat
- ✅ 1 TB kostenlos testen
- ✅ Ein sofort einsatzbereites [externes Ziel für Ihren PBS](https://nimbus.rdem-systems.com/en/offsite-proxmox-backup/?utm_source=github) — physisch getrennte (air-gapped), unveränderliche Datastores
- ✅ [NimbusBackup — Verwaltetes PBS-Hosting in Frankreich](https://nimbus.rdem-systems.com/en/?utm_source=github)

## 📚 Dokumentation

- **Vollständiger Proxmox-Backup-Leitfaden** — Best Practices für den PBS-Einsatz ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/complete-proxmox-backup-guide/?utm_source=github))
- **Windows mit Proxmox Backup Server sichern** — Windows-spezifische Einsatzanleitung ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github))
- **PBS vs. Veeam** — Vergleich mit Proxmox Backup Server ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/pbs-vs-veeam-proxmox-backup-comparison/?utm_source=github))
- In diesem Repository: [Multi-PBS-Benutzerhandbuch](MULTI_PBS_USER_GUIDE.md) · [Beispiele für die unbeaufsichtigte Bereitstellung](examples/automation/) · [Bare-Metal-Wiederherstellung mit Clonezilla](PATCH-CLONEZILLA.md) · [Changelog](CHANGELOG.md)

## ✨ Funktionen

### GUI (empfohlen)
- **🌍 Mehrsprachig** — Oberfläche auf Englisch, Französisch, Italienisch, Deutsch und Polnisch
- Benutzerfreundliche Konfiguration mit Verbindungstest (API-Token oder Benutzername/Passwort)
- Sicherungsfortschritt in Echtzeit mit Geschwindigkeit und Restzeit, jederzeit abbrechbar, ein Reiter „Laufende Jobs“
- VSS (Volume Shadow Copy) für konsistente Sicherungen: ein Snapshot für eine ganze Sicherung mehrerer Ordner, eine Schattenkopie pro Volume
- Sicherung mehrerer Ordner, Datei- und Datenträgermodus (ganze Maschine); Ordner können parallel gesichert werden (empfohlen: CPUs / 4)
- Datenträgersicherungen lassen sich in Proxmox VE als VM wiederherstellen, die der Maschine entspricht (CPUs, RAM, Firmware, Netzwerkkarten mit ihren MACs, eigene VM-ID)
- Durchsuchen von Snapshots, Dateisuche (Platzhalter) und Wiederherstellung, einschließlich NTFS-ACLs
- Unterstützung mehrerer PBS-Server mit einem PBS-Server pro Job, Pinning des Zertifikat-Fingerabdrucks (TOFU)
- **🔒 Clientseitige Verschlüsselung** (AES-256-GCM), Schlüsseldateien kompatibel mit `proxmox-backup-client` und Proxmox VE
- Windows-Dienstmodus + geplante Sicherungen, Sicherungsverlauf mit erneuter Ausführung per Klick
- Debug-Protokollierung zur Fehlersuche

### Kommandozeilen-Tools
- `proxmoxbackup-directory` — Verzeichnissicherungen (PXAR) mit Deduplizierung, Stream-Sicherungen (`-backupstream`, z. B. eine `mysqldump`-Pipe), Ausschlüsse (`-exclude "*.tmp"`, wiederholbar, oder `-exclude-from file`; `"exclude"` in der JSON-Konfiguration), mehrere Verzeichnisse gleichzeitig (`-parallel N`, empfohlen: CPUs / 4), E-Mail-Benachrichtigungen, JSON-Konfigurationsdatei
- `proxmoxbackup-machine` — vollständige Live-Sicherungen der Maschine als bootfähiges Datenträgerabbild (FIDX): VSS unter Windows, inkrementell, paralleles Hashing
- `proxmoxbackup-nbd` — NBD-Server zum Einhängen einer Datenträgersicherung unter Linux (Wiederherstellung auf Dateiebene, Bare-Metal-Wiederherstellung von einer [gepatchten Clonezilla-Live-ISO](PATCH-CLONEZILLA.md))

### 📸 Screenshots

![Server configuration](docs/screenshots/nimbus-gui-liste-servers.png)
*Verwaltung mehrerer PBS-Server mit Statusanzeigen*

![Add server form](docs/screenshots/nimbus-gui-add-server-form.png)
*Einfache Serverkonfiguration mit Verbindungstest*

![One-shot backup](docs/screenshots/nimbus-gui-one-shot-backup.png)
*Sicherungsfortschritt in Echtzeit mit Restzeit und Geschwindigkeit*

### Intelligente Systemausschlüsse (Dateimodus)
Beim Sichern eines ganzen Laufwerks (z. B. `D:\`) schließt Nimbus Backup automatisch aus:

**Systemordner:** `System Volume Information` (VSS-Speicher, kann über 100 GB groß sein), `$RECYCLE.BIN`, `Recovery`.
**Systemdateien:** `pagefile.sys`, `hiberfil.sys`, `swapfile.sys`.

**Warum das wichtig ist:** Ein Laufwerk kann 1,03 TB belegten Speicher melden, während die tatsächlichen Dateien nur ~141 GB umfassen. Ohne Ausschlüsse würde die Sicherung die VSS-Snapshots enthalten (verschwendeter Platz und verschwendete Zeit); mit ihnen entspricht die Sicherungsgröße den tatsächlichen Daten.

**Empfehlung:** Verwenden Sie den **Dateimodus** (Standard) mit automatischen Ausschlüssen für Sicherungen auf Dateiebene; verwenden Sie den **Datenträgermodus** in einem separaten Job für die Bare-Metal-Wiederherstellung (schließt alles ein).

### Sicherheit & Qualität
- Eingabevalidierung und Bereinigung von Anmeldedaten (Geheimnisse werden in Protokollen geschwärzt)
- Schutz vor Path Traversal
- Wiederholungslogik mit exponentiellem Backoff
- CI-Prüfungen bei jedem Build: Tests, `golangci-lint`, `gosec`, `go mod tidy` sowie eine End-to-End-Testsuite gegen einen echten PBS (Wiederherstellungen mit dem offiziellen `proxmox-backup-client`, PBS-Verify)

### 🔒 Clientseitige Verschlüsselung
Sicherungen können **auf dem Client** verschlüsselt werden, bevor sie die Maschine verlassen, mit
demselben Verfahren wie der offizielle `proxmox-backup-client` (AES-256-GCM, schlüsselabhängige
Chunk-Digests). Der PBS-Server speichert nur undurchsichtige Daten und bekommt
den Schlüssel nie zu sehen — nützlich auf einem gemeinsam genutzten oder verwalteten PBS.

Die Verschlüsselung wurde im Upstream-Projekt von Tiziano Bacocco entwickelt ([tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go)); der Papierschlüssel, der QR-Code-Export und der Schlüsselimport sind Ergänzungen von Nimbus Backup.

- **GUI**: *Server → Bearbeiten → Verschlüsselungsschlüssel* — erstellen Sie eine Schlüsseldatei oder wählen Sie eine
  vorhandene aus (aus `proxmox-backup-client key create --kdf none` oder einem PVE-
  Storage); ihr Fingerabdruck wird angezeigt. Bewahren Sie anschließend eine Kopie außerhalb der Maschine auf:
  **Drucken (Papierschlüssel)** speichert eine druckbare Seite mit dem Schlüssel und seinem QR-Code (das
  Format von `proxmox-backup-client key paperkey`), optional mit Passphrase geschützt.
  **Schlüssel aus Text oder QR-Code importieren** wandelt einen gescannten QR-Code oder einen Papierschlüssel
  wieder in eine Schlüsseldatei um. Die GUI verwendet nur ungeschützte Schlüsseldateien: Beim Importieren eines
  passphrasegeschützten Schlüssels wird dieser entsperrt und die neue Schlüsseldatei **ohne**
  Passphrase gespeichert — bewahren Sie diese Datei ebenso sicher auf wie den Schlüssel selbst.
- **CLI**: `-keyfile path/to/key.json` (oder `"keyfile"` in der JSON-Konfiguration); ein
  passphrasegeschützter Schlüssel erfordert `-keyfile-passphrase`, andernfalls wird die Passphrase abgefragt.
- **Bare-Metal-Wiederherstellung**: Die gepatchte Clonezilla-ISO stellt auch verschlüsselte
  Datenträgersicherungen wieder her (Schlüssel von einem USB-Stick, gegebenenfalls mit Passphrase) — siehe
  [PATCH-CLONEZILLA.md](PATCH-CLONEZILLA.md).
- **Kompatibel mit `proxmox-backup-client`, bei jedem Build in der CI geprüft**
  gegen einen echten PBS: Ein von Nimbus Backup mit einem von
  `proxmox-backup-client` erstellten Schlüssel verschlüsselter Ordner wird vom offiziellen Client mit demselben Schlüssel wiederhergestellt
  (und ohne ihn verweigert), und eine vom offiziellen Client erstellte verschlüsselte
  Datenträgersicherung wird von Nimbus Backup gelesen. Proxmox VE verwendet dieselben Schlüsseldateien.

> ⚠️ **Ohne den Schlüssel sind verschlüsselte Sicherungen nicht wiederherstellbar.** Die erste
> verschlüsselte Sicherung lädt alles erneut hoch (keine Deduplizierung mit unverschlüsselten
> Sicherungen). Backup-IDs, Archivnamen und Größen bleiben für den Server sichtbar; Datei-
> inhalte, Dateinamen und der Katalog sind verschlüsselt.

## 🤖 Unbeaufsichtigte Bereitstellung (Ansible & IaC)

Nimbus Backup lässt sich vollständig über Dateien installieren und konfigurieren — ohne interaktive Einrichtung.
Zwei Wege, beide mit einsatzbereiten Beispielen in
[`examples/automation/`](examples/automation/):

- **Kommandozeile** — eine JSON-Datei pro Host enthält die gesamte Sicherung
  (`proxmoxbackup-directory.exe --config file.json`); Planung über die Windows-
  Aufgabenplanung. Zuverlässige Exit-Codes (`0` OK, `1` schwerwiegender Fehler, `2` gesperrt, `3` teilweise),
  damit Ihr Orchestrator Fehler erkennt. Ideal für reines Infrastructure-as-Code.
- **GUI/Dienst (MSI)** — eine **einzige `config.json`** enthält die PBS-Verbindung,
  die Sicherungseinstellungen **und** den Zeitplan (`scheduled_jobs`). Datei verteilen,
  den Dienst `NimbusBackup` neu starten: Die Jobs werden idempotent abgeglichen und `nextRun` wird
  automatisch berechnet. Keine Zeitstempel-Berechnungen in Ihrem Jinja2-Template.

Der Dienst liest seine Konfiguration aus `C:\ProgramData\ProxmoxBackupClient\`
(seit 0.4.0; siehe [Upgrade](#️-upgrade-von--030)).

📖 Vollständige Anleitung: [Automate Windows backup to PBS with Ansible](https://nimbus.rdem-systems.com/en/blog/unattended-windows-backup-ansible/?utm_source=github).

## 🚀 Schnellstart

1. Laden Sie `NimbusBackup.msi` (oder die eigenständige `NimbusBackup.exe`) aus den Releases herunter
2. Installieren / starten Sie es mit Administratorrechten (für VSS erforderlich)
3. Konfigurieren Sie Ihre PBS-Verbindung und testen Sie sie
4. Wählen Sie die zu sichernden Verzeichnisse aus
5. Starten Sie die Sicherung — oder planen Sie sie

### 🔑 PBS-Benutzer und Berechtigungen

Der Client benötigt nur die Rolle **`DatastoreBackup`** auf dem Ziel-Datastore — kein Administratorkonto:

1. Erstellen Sie in der PBS-Oberfläche einen Benutzer (z. B. `nimbus@pbs`) und dafür ein API-Token (z. B. `nimbus@pbs!laptop01`).
2. Vergeben Sie unter **Datastore → Permissions** (oder **Configuration → Access Control → Permissions**) `DatastoreBackup` auf `/datastore/<name>` — oder auf `/datastore/<name>/<namespace>`, wenn Sie in einen Namespace sichern.
3. **Token mit Rechtetrennung** („Privilege Separation“ aktiviert, Standard): Vergeben Sie die Rolle an das **Token** selbst (`nimbus@pbs!laptop01`), nicht nur an den Benutzer — die effektiven Rechte sind die Schnittmenge aus beiden. Dies ist die häufigste Ursache für „permission denied“.

`DatastoreBackup` erlaubt dem Client, Sicherungen zu erstellen sowie seine eigenen Backup-Gruppen aufzulisten und wiederherzustellen. Zum Löschen oder Ausdünnen (Prune) von Snapshots ist `DatastorePowerUser` erforderlich.

## ⬆️ Upgrade von ≤ 0.3.0

Die Installation von 0.4.0 oder neuer über eine bestehende Installation aktualisiert diese direkt (gleiche MSI-Identität, gleicher Dienst `NimbusBackup`). Der Datenordner wechselt von `C:\ProgramData\NimbusBackup` zum gemeinsam genutzten `C:\ProgramData\ProxmoxBackupClient`: Beim ersten Start werden Ihre Konfiguration, geplanten Jobs, der Verlauf und das API-Token einmalig kopiert (vorhandene Dateien werden nie überschrieben). Der alte Ordner bleibt erhalten und wird mit `COPIED-TO-ProxmoxBackupClient.txt` markiert, sodass ein Downgrade weiterhin möglich ist. Mit älteren Versionen erstellte Snapshots können weiterhin an ihrem ursprünglichen Ort wiederhergestellt werden.

## 📋 Voraussetzungen

- Windows 10/11 oder Windows Server (64 Bit)
- Administratorrechte (für VSS-Snapshots)
- Netzwerkzugriff auf einen Proxmox Backup Server

## 🔨 Aus dem Quellcode bauen

### Voraussetzungen
- Go 1.25 oder neuer
- Node.js 20 oder neuer
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0` (die von der CI verwendete Version)

### Build
```bash
cd gui
npm install --prefix frontend
wails build      # or: wails dev  (hot reload)
```

Oder bauen Sie alles (CLI + GUI + Dienst) mit dem Makefile: `make install-deps && make` (siehe `make help`). Windows-Toolchain und Docker-Cross-Build: [BUILD.md](BUILD.md).

Die Marke wird anhand des Namens der ausführbaren Datei gewählt: `NimbusBackup.exe` läuft als Nimbus Backup, jeder andere Name als neutraler „Proxmox Backup Client“ ([`gui/brand.go`](gui/brand.go)).

## 🔗 Beziehung zum Upstream-Projekt

Nimbus Backup begann als Fork von [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go) (Proxmox Backup Client in Go, von Tiziano Bacocco, GPLv3), dem wir die Windows-GUI, den Dienst, die Zeitplanung, Multi-PBS und die Wiederherstellung hinzugefügt haben. Im September 2026 hat das Upstream-Projekt diese GUI übernommen und markenneutral gestaltet („Proxmox Backup Client GUI“). Im Oktober 2026 (0.4.1) wurde Nimbus Backup auf dem aktuellen Code des Upstream-Projekts neu aufgebaut und hat dessen clientseitige Verschlüsselung übernommen: **Die beiden Projekte teilen dieselbe Codebasis.**

Was dieses Repository gegenüber dem Upstream-Projekt hinzufügt, ist eine dokumentierte Patch-Serie ([`patches/`](patches/README.md)), die größtenteils upstream angeboten wird:

- **Funktionen**: Papierschlüssel und Schlüsselimport (QR-Code), parallele Ordnersicherungen, ein VSS-Snapshot pro Sicherung mehrerer Ordner, Ausschlüsse in der Kommandozeile, eine aus der realen Maschine erzeugte Proxmox-VE-VM-Konfiguration.
- **Upstream noch nicht übernommene Korrekturen** (Wiederherstellung komprimierter, verschlüsselter Sicherungen von `proxmox-backup-client`, der für eine Dienstsicherung gewählte PBS-Server, Ablehnungsgründe von PBS, Wiederherstellung bei Servern mit Benutzername/Passwort, Maximieren des Fensters…) — werden upstream eingereicht und dort nach und nach übernommen.
- **Die Nimbus-Backup-Identität** — `NimbusBackup.exe`/`.msi`, der Dienst `NimbusBackup` und der MSI-Upgrade-Code bestehender Installationen, damit diese weiterhin direkt aktualisiert werden.
- **Upgrade-Pfad** von Nimbus Backup ≤ 0.3.0 (Migration des Datenordners, Legacy-Snapshot-Metadaten).
- **Release-Pipeline** — Build-Provenienz-Attestierung, Prüfsummen, VirusTotal-Berichte.

Wir führen das Upstream-Projekt regelmäßig erneut zusammen; die eigenen Releases des Upstream-Projekts werden unter [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go/releases) veröffentlicht.

## ⚠️ Haftungsausschluss

Diese Software wird ohne Gewähr bereitgestellt. Obwohl wir uns um Zuverlässigkeit bemühen, übernehmen wir keine Verantwortung für Datenverluste oder Schäden. Testen Sie Ihre Sicherungen stets und überprüfen Sie die Wiederherstellung, bevor Sie sich im Produktivbetrieb darauf verlassen.

Dieses Projekt ist **nicht verbunden** mit der **Proxmox Server Solutions GmbH**. „Proxmox“ und zugehörige Namen sind Eigentum ihrer jeweiligen Inhaber und werden hier nur zur Angabe der Kompatibilität verwendet.

## 📄 Lizenz

GPLv3 — siehe die Datei [LICENSE](LICENSE).

## Über RDEM Systems

NimbusBackupClient wird von [RDEM Systems](https://www.rdem-systems.com/en/?utm_source=github) entwickelt und gepflegt, einem französischen Infrastrukturanbieter, der auf verwaltete Proxmox-VE/PBS-Dienste und NTP/NTS-Infrastruktur spezialisiert ist. Wir betreiben [16 öffentliche NTS-Server](https://ntp.rdem-systems.com/en/nts.php?utm_source=github) ([Live-Status](https://ntp.rdem-systems.com/en/status.php?utm_source=github); 11 davon sind in der [Community-Referenz](https://github.com/jauderho/nts-servers) gelistet) und bieten [vollständig verwaltetes PBS-Hosting](https://nimbus.rdem-systems.com/en/?utm_source=github) für alle, die nicht selbst hosten möchten. Außerdem pflegen wir [unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client), signierte `proxmox-backup-client`-Pakete für Linux-Distributionen, die Proxmox nicht offiziell unterstützt.

---

**© 2024-2026 RDEM Systems and Proxmox Backup Client GO contributors.**
