# Nimbus Backup — Windows-klient för Proxmox Backup Server

[🇬🇧 English](README.md) · [🇫🇷 Français](README.fr.md) · [🇮🇹 Italiano](README.it.md) · [🇩🇪 Deutsch](README.de.md) · [🇪🇸 Español](README.es.md) · [🇷🇺 Русский](README.ru.md) · [🇨🇳 中文](README.zh.md) · [🇯🇵 日本語](README.ja.md) · [🇬🇷 Ελληνικά](README.el.md) · [🇷🇴 Română](README.ro.md) · 🇸🇪 Svenska · [🇸🇦 العربية](README.ar.md) · [🇮🇷 فارسی](README.fa.md)

> 🤖 Denna översättning har tagits fram med AI utifrån den engelska README-filen; vid avvikelser gäller den [engelska versionen](README.md).
> *AI-generated translation of the English README; the [English version](README.md) is authoritative.*

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rdemsystems/NimbusBackupClient)](https://github.com/rdemsystems/NimbusBackupClient/releases)
[![Documentation](https://img.shields.io/badge/docs-nimbus.rdem--systems.com-orange)](https://nimbus.rdem-systems.com/en/?utm_source=github)

**Nimbus Backup är en säkerhetskopieringsklient med öppen källkod (GPL-3.0) för Windows, avsedd för Proxmox Backup Server (PBS).**
Ett modernt grafiskt gränssnitt för att säkerhetskopiera Windows-servrar och arbetsstationer till PBS — VSS-konsekventa ögonblicksbilder, schemalagda jobb, fil- och diskläge, bläddring i och återställning av ögonblicksbilder, stöd för flera PBS och en Windows-tjänst — samt kommandoradsverktyg för säkerhetskopiering av kataloger och hela maskiner. Letar du efter **offsite, oföränderlig** PBS-lagring utan att drifta den själv? Se den [hanterade tjänsten](#️-hanterad-pbs-offsite--oföränderlig) nedan.

Nimbus Backup är RDEM Systems build av [Proxmox Backup Client GO](https://github.com/tizbac/proxmoxbackupclient_go): det grafiska gränssnitt vi utvecklade har slagits ihop med upstream, och båda projekten delar nu samma kodbas (se [Förhållande till upstream](#-förhållande-till-upstream)).

📖 **Fullständig dokumentation, installationsguide och PBS-hosting:** [nimbus.rdem-systems.com](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github&utm_medium=readme&utm_campaign=nbc-readme-top)

## 📦 Nedladdning

👉 **[Ladda ner den senaste versionen](https://github.com/rdemsystems/NimbusBackupClient/releases)**

Varje version innehåller:
- `NimbusBackup.msi` — installationsprogram (GUI + Windows-tjänst), **rekommenderas för produktion**
- `NimbusBackup.exe` — fristående GUI
- `nimbus-backup-cli-<version>-windows.zip` / `-linux.tar.gz` / `-macos.tar.gz` — kommandoradsverktyg
- `SHA256SUMS.txt` — kontrollsummor

> ⚠️ **Säger Windows ”virus hittat” (t.ex. `Trojan:Win32/Sabsik.FL.A!ml`) eller visar en SmartScreen-varning?**
> Detta är ett känt **falskt positivt** resultat för Go/Wails-program — det är *inte* ett virus. Suffixet `!ml` betyder att det kommer från en maskininlärningsmodell som flaggar *sällsynta* körbara filer (och, till och med 0.4.0, osignerade).
> Läs [varför detta händer och hur du verifierar nedladdningen](https://nimbus.rdem-systems.com/en/antivirus-false-positive/?utm_source=github).

### 🔎 Verifiera en nedladdning

Varje version levereras med SHA-256-kontrollsummor och ett signerat **build-provenance-intyg** (kryptografiskt bevis på att binärfilen byggdes av detta repos CI, från en specifik commit):

```powershell
Get-FileHash .\NimbusBackup.msi -Algorithm SHA256   # compare against SHA256SUMS.txt
gh attestation verify .\NimbusBackup.msi --repo rdemsystems/NimbusBackupClient
```

**VirusTotal.** Versionsinformationen för varje version länkar till VirusTotal-rapporten för det bygget av installationsprogrammet (publiceras endast när genomsökningen är ren). Tidigare rapporter, 0 träffar:
[0.2.108](https://www.virustotal.com/gui/file/6e8fb7ce9af740d470e947addb8daba4331c0b88e8bfdec9e0697ea8f7f29e9e/detection) ·
[0.2.107](https://www.virustotal.com/gui/file/6fd6c6fa77e0305c129ef882a3745100aa6033187a6d52a4af94149ab6b666d2/detection) ·
[0.2.106](https://www.virustotal.com/gui/file/ad6e56700ed9df8e088906e38cee2e2882fc7045f4e39269de0e379a01784ad7/detection)

> 🔏 **Kodsignering:** sedan 0.4.1 är `NimbusBackup.exe`, dess tjänst och `NimbusBackup.msi` **Authenticode-signerade av RDEM SYSTEMS** (Azure Artifact Signing); *Egenskaper → Digitala signaturer* visar utgivaren. SmartScreen kan fortfarande varna för en ny version tills dess rykte har etablerats: kontrollera att utgivaren är RDEM SYSTEMS och välj sedan *Mer information → Kör ändå*. Kommandoradsverktygen är ännu inte signerade; build-provenance-intyget och kontrollsummorna ovan täcker alla filer.

### 🐧 Kör du Linux? Använd den officiella klienten

Nimbus Backups GUI finns endast för Windows (CLI-verktygen byggs även för Linux och macOS). För säkerhetskopiering på filnivå i Linux, använd Proxmox egen `proxmox-backup-client` — vi paketerar den för de distributioner som Proxmox inte täcker:

👉 **[rdemsystems/unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)** — signerade paketförråd för Debian/Ubuntu, Fedora/RHEL/Rocky/AlmaLinux, Arch och Alpine (amd64 & arm64). Proxmox officiella statiska binärfil, ompaketerad oförändrad — inte patchad, inte omkompilerad.

## ☁️ Hanterad PBS (offsite & oföränderlig)

Vill du inte drifta Proxmox Backup Server själv? Använd våra helt hanterade, **offsite och oföränderliga** PBS-datastores:
👉 **[Konfigurera din säkerhetskopiering & se priser](https://nimbus.rdem-systems.com/en/choose-backup/?utm_source=github)**

- ✅ Från 12 €/TB/månad
- ✅ 1 TB gratis provperiod
- ✅ Ett färdigt [offsite-mål för din PBS](https://nimbus.rdem-systems.com/en/offsite-proxmox-backup/?utm_source=github) — isolerade (air-gapped), oföränderliga datastores
- ✅ [NimbusBackup — Hanterad PBS-hosting i Frankrike](https://nimbus.rdem-systems.com/en/?utm_source=github)

## 📚 Dokumentation

- **Komplett guide till Proxmox Backup** — bästa praxis för driftsättning av PBS ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/complete-proxmox-backup-guide/?utm_source=github))
- **Säkerhetskopiera Windows med Proxmox Backup Server** — Windows-specifik driftsättningsguide ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github))
- **PBS vs Veeam** — jämförelse med Proxmox Backup Server ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/pbs-vs-veeam-proxmox-backup-comparison/?utm_source=github))
- I detta repo: [användarguide för flera PBS](MULTI_PBS_USER_GUIDE.md) · [exempel på obevakad driftsättning](examples/automation/) · [bare-metal-återställning med Clonezilla](PATCH-CLONEZILLA.md) · [ändringslogg](CHANGELOG.md)

## ✨ Funktioner

### GUI (rekommenderas)
- **🌍 Flerspråkigt** — gränssnitt på engelska, franska, italienska, tyska och polska
- Användarvänlig konfiguration med anslutningstest (API-token eller användarnamn/lösenord)
- Förlopp för säkerhetskopieringen i realtid med hastighet och beräknad tid kvar, kan avbrytas när som helst, en flik för pågående jobb
- VSS (Volume Shadow Copy) för konsekventa säkerhetskopior: en ögonblicksbild för en hel säkerhetskopia av flera mappar, en skuggkopia per volym
- Säkerhetskopiering av flera mappar, fil- och diskläge (hela maskinen); mappar kan köras parallellt (rekommenderat: CPU:er / 4)
- Disksäkerhetskopior återställs i Proxmox VE som en VM som motsvarar maskinen (CPU:er, RAM, firmware, nätverkskort med sina MAC-adresser, dedikerat VM-ID)
- Bläddring i ögonblicksbilder, filsökning (jokertecken) och återställning, inklusive NTFS-ACL:er
- Stöd för flera PBS-servrar med en PBS-server per jobb, fastlåsning av certifikatets fingeravtryck (TOFU)
- **🔒 Kryptering på klientsidan** (AES-256-GCM), nyckelfiler kompatibla med `proxmox-backup-client` och Proxmox VE
- Windows-tjänstläge + schemalagda säkerhetskopior, historik över säkerhetskopior med omkörning med ett klick
- Felsökningsloggning

### Kommandoradsverktyg
- `proxmoxbackup-directory` — katalogsäkerhetskopior (PXAR) med deduplicering, strömsäkerhetskopior (`-backupstream`, t.ex. en `mysqldump`-pipe), undantag (`-exclude "*.tmp"`, kan upprepas, eller `-exclude-from file`; `"exclude"` i JSON-konfigurationen), flera kataloger samtidigt (`-parallel N`, rekommenderat: CPU:er / 4), e-postaviseringar, JSON-konfigurationsfil
- `proxmoxbackup-machine` — fullständiga säkerhetskopior av en körande maskin som en startbar diskavbild (FIDX): VSS i Windows, inkrementell, parallell hashning
- `proxmoxbackup-nbd` — NBD-server för att montera en disksäkerhetskopia i Linux (återställning på filnivå, bare-metal-återställning från en [patchad Clonezilla live-ISO](PATCH-CLONEZILLA.md))

### 📸 Skärmbilder

![Server configuration](docs/screenshots/nimbus-gui-liste-servers.png)
*Hantering av flera PBS-servrar med statusindikatorer*

![Add server form](docs/screenshots/nimbus-gui-add-server-form.png)
*Enkel serverkonfiguration med anslutningstest*

![One-shot backup](docs/screenshots/nimbus-gui-one-shot-backup.png)
*Förlopp för säkerhetskopieringen i realtid med beräknad tid kvar och hastighet*

### Smarta systemundantag (filläge)
Vid säkerhetskopiering av en hel enhet (t.ex. `D:\`) undantar Nimbus Backup automatiskt:

**Systemmappar:** `System Volume Information` (VSS-lagring, kan vara över 100 GB), `$RECYCLE.BIN`, `Recovery`.
**Systemfiler:** `pagefile.sys`, `hiberfil.sys`, `swapfile.sys`.

**Varför det spelar roll:** en enhet kan rapportera 1,03 TB använt medan de verkliga filerna är ~141 GB. Utan undantag skulle säkerhetskopian innehålla VSS-ögonblicksbilder (bortslösat utrymme och tid); med dem motsvarar säkerhetskopians storlek de verkliga data.

**Rekommendation:** använd **filläge** (standard) med automatiska undantag för säkerhetskopior på filnivå; använd **diskläge** i ett separat jobb för bare-metal-återställning (inkluderar allt).

### Säkerhet & kvalitet
- Validering av indata och sanering av inloggningsuppgifter (hemligheter maskeras i loggar)
- Skydd mot path traversal
- Logik för nya försök med exponentiell backoff
- CI-kontroller vid varje bygge: tester, `golangci-lint`, `gosec`, `go mod tidy` och en end-to-end-svit mot en riktig PBS (återställningar med den officiella `proxmox-backup-client`, PBS verify)

### 🔒 Kryptering på klientsidan
Säkerhetskopior kan krypteras **på klienten** innan de lämnar maskinen, med
samma schema som den officiella `proxmox-backup-client` (AES-256-GCM, nyckelbaserade
chunk-digests). PBS-servern lagrar endast ogenomskinliga data och ser aldrig
nyckeln — användbart på en delad eller hanterad PBS.

Krypteringen har utvecklats i originalprojektet av Tiziano Bacocco ([tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go)); papperskopian av nyckeln, QR-kodsexporten och nyckelimporten är tillägg i Nimbus Backup.

- **GUI**: *PBS Configuration → Edit → Encryption key file* — skapa en nyckelfil eller välj
  en befintlig (från `proxmox-backup-client key create --kdf none` eller en PVE-
  storage); dess fingeravtryck visas. Spara sedan en kopia utanför maskinen:
  **Print (paper key)** sparar en utskrivbar sida med nyckeln och dess QR-kod (i
  formatet från `proxmox-backup-client key paperkey`), valfritt skyddad med lösenfras.
  **Import a key from text or a QR code** gör om en skannad QR-kod eller en pappersnyckel
  till en nyckelfil igen. GUI:t använder endast oskyddade nyckelfiler: import av en
  lösenfrasskyddad nyckel låser upp den och sparar den nya nyckelfilen **utan**
  lösenfras — förvara den filen lika säkert som själva nyckeln.
- **CLI**: `-keyfile path/to/key.json` (eller `"keyfile"` i JSON-konfigurationen); en
  lösenfrasskyddad nyckel tar `-keyfile-passphrase`, annars frågas efter den.
- **Bare-metal-återställning**: den patchade Clonezilla-ISO:n återställer även krypterade
  disksäkerhetskopior (nyckel från ett USB-minne, med dess lösenfras om sådan finns) — se
  [PATCH-CLONEZILLA.md](PATCH-CLONEZILLA.md).
- **Kompatibel med `proxmox-backup-client`, verifierad i CI vid varje bygge**
  mot en riktig PBS: en mapp krypterad av Nimbus Backup med en nyckel skapad av
  `proxmox-backup-client` återställs av den officiella klienten med samma nyckel
  (och nekas utan den), och en krypterad disksäkerhetskopia gjord av den officiella
  klienten läses tillbaka av Nimbus Backup. Proxmox VE använder samma nyckelfiler.

> ⚠️ **Utan nyckeln går krypterade säkerhetskopior inte att återställa.** Den första
> krypterade säkerhetskopian laddar upp allt på nytt (ingen deduplicering mot okrypterade
> säkerhetskopior). Säkerhetskopiornas ID, arkivnamn och storlekar förblir synliga för servern; filernas
> innehåll, filnamn och katalogen är krypterade.

## 🤖 Obevakad driftsättning (Ansible & IaC)

Nimbus Backup installeras och konfigureras helt via filer — ingen interaktiv installation.
Två vägar, båda täckta av färdiga exempel i
[`examples/automation/`](examples/automation/):

- **Kommandorad** — en JSON-fil per värd innehåller hela säkerhetskopieringen
  (`proxmoxbackup-directory.exe --config file.json`); schemalägg via Windows
  Schemaläggaren. Pålitliga slutkoder (`0` OK, `1` fatalt, `2` låst, `3` delvis)
  så att din orkestrerare upptäcker fel. Bäst för ren infrastructure-as-code.
- **GUI/tjänst (MSI)** — en **enda `config.json`** innehåller PBS-anslutningen,
  säkerhetskopieringsinställningarna **och** schemat (`scheduled_jobs`). Distribuera filen, starta om
  tjänsten `NimbusBackup`: jobben stäms av idempotent och `nextRun` beräknas
  åt dig. Ingen tidsstämpelsmatematik i din Jinja2-mall.

Tjänsten läser sin konfiguration från `C:\ProgramData\ProxmoxBackupClient\`
(sedan 0.4.0; se [Uppgradera](#️-uppgradera-från--030)).

📖 Fullständig genomgång: [Automate Windows backup to PBS with Ansible](https://nimbus.rdem-systems.com/en/blog/unattended-windows-backup-ansible/?utm_source=github).

## 🚀 Snabbstart

1. Ladda ner `NimbusBackup.msi` (eller den fristående `NimbusBackup.exe`) från versionssidan
2. Installera / kör den med administratörsbehörighet (krävs för VSS)
3. Konfigurera din PBS-anslutning och testa den
4. Välj kataloger att säkerhetskopiera
5. Starta säkerhetskopieringen — eller schemalägg den

### 🔑 PBS-användare och behörigheter

Klienten behöver endast rollen **`DatastoreBackup`** på mål-datastore — inget administratörskonto:

1. Skapa i PBS-gränssnittet en användare (t.ex. `nimbus@pbs`) och en API-token för den (t.ex. `nimbus@pbs!laptop01`).
2. Under **Datastore → Permissions** (eller **Configuration → Access Control → Permissions**), tilldela `DatastoreBackup` på `/datastore/<name>` — eller på `/datastore/<name>/<namespace>` om du säkerhetskopierar till ett namespace.
3. **Token med behörighetsseparation** (”Privilege Separation” ikryssat, standard): tilldela rollen till själva **token** (`nimbus@pbs!laptop01`), inte bara till användaren — de effektiva rättigheterna är snittet av båda. Detta är den vanligaste orsaken till ”permission denied”.

`DatastoreBackup` låter klienten skapa säkerhetskopior samt lista och återställa sina egna säkerhetskopiegrupper. Att radera eller rensa (prune) ögonblicksbilder kräver `DatastorePowerUser`.

## ⬆️ Uppgradera från ≤ 0.3.0

Att installera 0.4.0 eller senare ovanpå en befintlig installation uppgraderar den på plats (samma MSI-identitet, samma `NimbusBackup`-tjänst). Datamappen flyttas från `C:\ProgramData\NimbusBackup` till den delade `C:\ProgramData\ProxmoxBackupClient`: vid första start kopieras din konfiguration, dina schemalagda jobb, din historik och din API-token över en gång (befintliga filer skrivs aldrig över). Den gamla mappen behålls, märkt med `COPIED-TO-ProxmoxBackupClient.txt`, så att en nedgradering fortfarande fungerar. Ögonblicksbilder tagna av äldre versioner kan fortfarande återställas till sin ursprungliga plats.

## 📋 Krav

- Windows 10/11 eller Windows Server (64-bitars)
- Administratörsrättigheter (för VSS-ögonblicksbilder)
- Nätverksåtkomst till en Proxmox Backup Server

## 🔨 Bygga från källkod

### Förutsättningar
- Go 1.25 eller senare
- Node.js 20 eller senare
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0` (den version som CI använder)

### Bygge
```bash
cd gui
npm install --prefix frontend
wails build      # or: wails dev  (hot reload)
```

Eller bygg allt (CLI + GUI + tjänst) med Makefile: `make install-deps && make` (se `make help`). Windows-verktygskedja och korsbygge med Docker: [BUILD.md](BUILD.md).

Varumärket väljs utifrån den körbara filens namn: `NimbusBackup.exe` körs som Nimbus Backup, alla andra namn som den neutrala ”Proxmox Backup Client” ([`gui/brand.go`](gui/brand.go)).

## 🔗 Förhållande till upstream

Nimbus Backup började som en fork av [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go) (Proxmox Backup Client i Go, av Tiziano Bacocco, GPLv3), till vilken vi lade till Windows-GUI:t, tjänsten, schemaläggningen, stöd för flera PBS och återställning. I september 2026 slog upstream ihop det GUI:t och gjorde det varumärkesneutralt (”Proxmox Backup Client GUI”). I oktober 2026 (0.4.1) byggdes Nimbus Backup om på upstreams aktuella kod och tog över upstreams kryptering på klientsidan: **de två projekten delar samma kodbas.**

Nimbus Backup 0.4.1 bygger på originalprojektets `master`-gren, commit `3c1b989` (9 oktober 2026). Ingen release från originalprojektet innehåller ännu den här koden: den senaste, v1.1.3 (maj 2026), är äldre än integreringen av det grafiska gränssnittet.

Det detta repo lägger till ovanpå upstream är en dokumenterad patchserie ([`patches/`](patches/README.md)), varav det mesta har erbjudits upstream:

- **Funktioner**: paper key och nyckelimport (QR-kod), parallella säkerhetskopior av mappar, en VSS-ögonblicksbild per säkerhetskopia av flera mappar, undantag i CLI, en VM-konfiguration för Proxmox VE genererad från den verkliga maskinen.
- **Rättningar som ännu inte slagits ihop med upstream** (återställning av `proxmox-backup-client`:s komprimerade krypterade säkerhetskopior, den PBS-server som väljs för en säkerhetskopia via tjänsten, PBS:s orsaker till avvisning, återställning med servrar som använder användarnamn/lösenord, maximering av fönstret…) — skickas upstream allteftersom de slås ihop.
- **Nimbus Backup-identiteten** — `NimbusBackup.exe`/`.msi`, tjänsten `NimbusBackup` och MSI-uppgraderingskoden för befintliga installationer, så att de fortsätter att uppgraderas på plats.
- **Uppgraderingsväg** från Nimbus Backup ≤ 0.3.0 (migrering av datamappen, äldre metadata för ögonblicksbilder).
- **Releasepipeline** — build-provenance-intyg, kontrollsummor, VirusTotal-rapporter.

Vi slår regelbundet ihop upstream på nytt; upstreams egna versioner publiceras på [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go/releases).

## ⚠️ Ansvarsfriskrivning

Denna programvara tillhandahålls i befintligt skick. Även om vi strävar efter tillförlitlighet tar vi inget ansvar för dataförlust eller skada. Testa alltid dina säkerhetskopior och verifiera återställningen innan du förlitar dig på dem i produktion.

Detta projekt är **inte anslutet** till **Proxmox Server Solutions GmbH**. ”Proxmox” och relaterade namn tillhör respektive ägare och används här endast för att ange kompatibilitet.

## 📄 Licens

GPLv3 — se filen [LICENSE](LICENSE).

## Om RDEM Systems

NimbusBackupClient utvecklas och underhålls av [RDEM Systems](https://www.rdem-systems.com/en/?utm_source=github), en fransk infrastrukturleverantör specialiserad på hanterade tjänster för Proxmox VE/PBS och NTP/NTS-infrastruktur. Vi driver [16 publika NTS-servrar](https://ntp.rdem-systems.com/en/nts.php?utm_source=github) ([status i realtid](https://ntp.rdem-systems.com/en/status.php?utm_source=github); 11 av dem finns med i [communityns referenslista](https://github.com/jauderho/nts-servers)) och erbjuder [helt hanterad PBS-hosting](https://nimbus.rdem-systems.com/en/?utm_source=github) för användare som inte vill drifta själva. Vi underhåller också [unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client), signerade `proxmox-backup-client`-paket för Linuxdistributioner som Proxmox inte officiellt stöder.

---

**© 2024-2026 RDEM Systems and Proxmox Backup Client GO contributors.**
