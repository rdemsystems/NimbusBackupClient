# Nimbus Backup — client Windows pentru Proxmox Backup Server

[🇬🇧 English](README.md) · [🇫🇷 Français](README.fr.md) · [🇮🇹 Italiano](README.it.md) · [🇩🇪 Deutsch](README.de.md) · [🇪🇸 Español](README.es.md) · [🇷🇺 Русский](README.ru.md) · [🇨🇳 中文](README.zh.md) · [🇯🇵 日本語](README.ja.md) · [🇬🇷 Ελληνικά](README.el.md) · 🇷🇴 Română · [🇸🇪 Svenska](README.sv.md) · [🇸🇦 العربية](README.ar.md) · [🇮🇷 فارسی](README.fa.md)

> 🤖 Această traducere a fost realizată de inteligența artificială pornind de la README-ul în engleză; în caz de neconcordanță, prevalează [versiunea în engleză](README.md).
> *AI-generated translation of the English README; the [English version](README.md) is authoritative.*

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rdemsystems/NimbusBackupClient)](https://github.com/rdemsystems/NimbusBackupClient/releases)
[![Documentation](https://img.shields.io/badge/docs-nimbus.rdem--systems.com-orange)](https://nimbus.rdem-systems.com/en/?utm_source=github)

**Nimbus Backup este un client de backup open-source (GPL-3.0) pentru Windows, destinat Proxmox Backup Server (PBS).**
O interfață grafică modernă pentru backup-ul serverelor și stațiilor de lucru Windows în PBS — snapshot-uri consistente VSS, sarcini programate, moduri fișier și disc, navigare și restaurare a snapshot-urilor, suport multi-PBS și un serviciu Windows — plus instrumente în linie de comandă pentru backup-ul directoarelor și al mașinilor complete. Căutați stocare PBS **offsite și imuabilă** fără să o găzduiți singuri? Vedeți [serviciul gestionat](#️-pbs-gestionat-offsite--imuabil) mai jos.

Nimbus Backup este build-ul RDEM Systems al [Proxmox Backup Client GO](https://github.com/tizbac/proxmoxbackupclient_go): interfața grafică pe care am dezvoltat-o a fost integrată în upstream, iar cele două proiecte împart acum aceeași bază de cod (vedeți [Relația cu upstream](#-relația-cu-upstream)).

📖 **Documentație completă, ghid de instalare și găzduire PBS:** [nimbus.rdem-systems.com](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github&utm_medium=readme&utm_campaign=nbc-readme-top)

## 📦 Descărcare

👉 **[Descărcați cea mai recentă versiune](https://github.com/rdemsystems/NimbusBackupClient/releases)**

Fiecare versiune include:
- `NimbusBackup.msi` — program de instalare (GUI + serviciu Windows), **recomandat pentru producție**
- `NimbusBackup.exe` — GUI de sine stătător
- `nimbus-backup-cli-<version>-windows.zip` / `-linux.tar.gz` / `-macos.tar.gz` — instrumente în linie de comandă
- `SHA256SUMS.txt` — sume de control

> ⚠️ **Windows raportează „virus detectat” (de ex. `Trojan:Win32/Sabsik.FL.A!ml`) sau afișează un avertisment SmartScreen?**
> Este un **fals pozitiv** cunoscut pentru aplicațiile Go/Wails — *nu* este un virus. Sufixul `!ml` înseamnă că provine de la un model de învățare automată care semnalează executabilele *nesemnate și puțin răspândite*.
> Citiți [de ce se întâmplă acest lucru și cum să verificați descărcarea](https://nimbus.rdem-systems.com/en/antivirus-false-positive/?utm_source=github).

### 🔎 Verificarea oricărei descărcări

Fiecare versiune include sume de control SHA-256 și o **atestare de proveniență a build-ului** semnată (dovadă criptografică a faptului că binarul a fost produs de CI-ul acestui depozit, dintr-un commit anume):

```powershell
Get-FileHash .\NimbusBackup.msi -Algorithm SHA256   # compare against SHA256SUMS.txt
gh attestation verify .\NimbusBackup.msi --repo rdemsystems/NimbusBackupClient
```

**VirusTotal.** Notele fiecărei versiuni conțin linkul către raportul VirusTotal al programului de instalare din acel build (publicat doar când scanarea este curată). Rapoarte anterioare, 0 detecții:
[0.2.108](https://www.virustotal.com/gui/file/6e8fb7ce9af740d470e947addb8daba4331c0b88e8bfdec9e0697ea8f7f29e9e/detection) ·
[0.2.107](https://www.virustotal.com/gui/file/6fd6c6fa77e0305c129ef882a3745100aa6033187a6d52a4af94149ab6b666d2/detection) ·
[0.2.106](https://www.virustotal.com/gui/file/ad6e56700ed9df8e088906e38cee2e2882fc7045f4e39269de0e379a01784ad7/detection)

> ℹ️ **Semnarea codului:** binarele Windows **nu sunt încă semnate Authenticode**, ceea ce declanșează avertismentele SmartScreen / `!ml` de mai sus. Cererea noastră pentru un certificat OSS gratuit de la [SignPath Foundation](https://signpath.org) nu a primit răspuns; semnarea prin Azure Artifact Signing este în curs de configurare și este vizată pentru **0.4.1**. Până atunci, proveniența este stabilită prin atestarea de proveniență a build-ului și sumele de control de mai sus.

### 🐧 Pe Linux? Folosiți clientul oficial

GUI-ul Nimbus Backup este doar pentru Windows (instrumentele CLI se compilează și pentru Linux și macOS). Pentru backup la nivel de fișier pe Linux, folosiți `proxmox-backup-client` al Proxmox — îl împachetăm pentru distribuțiile pe care Proxmox nu le acoperă:

👉 **[rdemsystems/unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)** — depozite de pachete semnate pentru Debian/Ubuntu, Fedora/RHEL/Rocky/AlmaLinux, Arch și Alpine (amd64 & arm64). Binarul static oficial Proxmox, reîmpachetat neschimbat — fără patch-uri, fără recompilare.

## ☁️ PBS gestionat (offsite & imuabil)

Nu doriți să găzduiți singuri Proxmox Backup Server? Folosiți datastore-urile noastre PBS complet gestionate, **offsite și imuabile**:
👉 **[Configurați backup-ul & vedeți prețurile](https://nimbus.rdem-systems.com/en/choose-backup/?utm_source=github)**

- ✅ De la 12 €/TB/lună
- ✅ Perioadă de probă gratuită de 1 TB
- ✅ O [destinație offsite pentru PBS-ul dumneavoastră](https://nimbus.rdem-systems.com/en/offsite-proxmox-backup/?utm_source=github) gata de folosit — datastore-uri izolate (air-gapped) și imuabile
- ✅ [NimbusBackup — Găzduire PBS gestionată în Franța](https://nimbus.rdem-systems.com/en/?utm_source=github)

## 📚 Documentație

- **Ghid complet Proxmox Backup** — bune practici de implementare PBS ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/complete-proxmox-backup-guide/?utm_source=github))
- **Backup Windows cu Proxmox Backup Server** — ghid de implementare specific Windows ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github))
- **PBS vs Veeam** — comparație cu Proxmox Backup Server ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/pbs-vs-veeam-proxmox-backup-comparison/?utm_source=github))
- În acest depozit: [ghid de utilizare multi-PBS](MULTI_PBS_USER_GUIDE.md) · [exemple de implementare automată](examples/automation/) · [restaurare bare-metal cu Clonezilla](PATCH-CLONEZILLA.md) · [jurnal de modificări](CHANGELOG.md)

## ✨ Funcționalități

### GUI (recomandat)
- **🌍 Multilingv** — interfață în engleză, franceză, italiană, germană și poloneză
- Configurare ușoară, cu testarea conexiunii (token API sau nume de utilizator/parolă)
- Progresul backup-ului în timp real, cu viteză și timp estimat, anulare oricând
- Suport VSS (Volume Shadow Copy) pentru backup-uri consistente
- Backup pentru mai multe directoare, moduri fișier și disc (mașină completă); directoarele pot rula în paralel (recomandat: CPU-uri / 4)
- Navigare în snapshot-uri, căutare de fișiere (wildcard-uri) și restaurare
- Suport pentru mai multe servere PBS, fixarea amprentei certificatului (TOFU)
- **🔒 Criptare pe partea clientului** (AES-256-GCM), fișiere cheie compatibile cu `proxmox-backup-client` și Proxmox VE
- Mod serviciu Windows + backup-uri programate, istoric al backup-urilor cu reluare dintr-un clic
- Jurnalizare de depanare pentru diagnosticarea problemelor

### Instrumente în linie de comandă
- `proxmoxbackup-directory` — backup de directoare (PXAR) cu deduplicare, backup din flux (`-backupstream`, de ex. un pipe `mysqldump`), excluderi (`-exclude "*.tmp"`, repetabil, sau `-exclude-from file`; `"exclude"` în configurația JSON), mai multe directoare simultan (`-parallel N`, recomandat: CPU-uri / 4), notificări prin e-mail, fișier de configurare JSON
- `proxmoxbackup-machine` — backup complet al mașinii în funcțiune, ca imagine de disc bootabilă (FIDX): VSS pe Windows, incremental, hashing paralel
- `proxmoxbackup-nbd` — server NBD pentru montarea unui backup de disc pe Linux (restaurare la nivel de fișier, restaurare bare-metal dintr-un [ISO live Clonezilla modificat](PATCH-CLONEZILLA.md))

### 📸 Capturi de ecran

![Server configuration](docs/screenshots/nimbus-gui-liste-servers.png)
*Gestionarea mai multor servere PBS, cu indicatori de stare*

![Add server form](docs/screenshots/nimbus-gui-add-server-form.png)
*Configurare simplă a serverului, cu testarea conexiunii*

![One-shot backup](docs/screenshots/nimbus-gui-one-shot-backup.png)
*Progresul backup-ului în timp real, cu timp estimat și viteză*

### Excluderi inteligente de sistem (modul fișier)
La backup-ul unei unități întregi (de ex. `D:\`), Nimbus Backup exclude automat:

**Directoare de sistem:** `System Volume Information` (stocare VSS, poate depăși 100 GB), `$RECYCLE.BIN`, `Recovery`.
**Fișiere de sistem:** `pagefile.sys`, `hiberfil.sys`, `swapfile.sys`.

**De ce contează:** o unitate poate raporta 1,03 TB utilizați, în timp ce fișierele reale au ~141 GB. Fără excluderi, backup-ul ar include snapshot-urile VSS (spațiu și timp irosite); cu ele, dimensiunea backup-ului corespunde datelor reale.

**Recomandare:** folosiți **modul fișier** (implicit) cu excluderi automate pentru backup la nivel de fișier; folosiți **modul disc** într-o sarcină separată pentru restaurare bare-metal (include totul).

### Securitate & calitate
- Validarea datelor de intrare și igienizarea credențialelor (secretele sunt mascate în jurnale)
- Prevenirea path traversal
- Logică de reîncercare cu backoff exponențial
- Verificări CI la fiecare build: teste, `golangci-lint`, `gosec`, `go mod tidy`

### 🔒 Criptare pe partea clientului
Backup-urile pot fi criptate **pe client** înainte de a părăsi mașina, cu
aceeași schemă ca `proxmox-backup-client` oficial (AES-256-GCM, digest-uri de chunk
cu cheie). Serverul PBS stochează doar date opace și nu vede niciodată
cheia — util pe un PBS partajat sau gestionat.

- **GUI**: *Servers → Edit → Encryption key* — creați un fișier cheie sau alegeți
  unul existent (din `proxmox-backup-client key create --kdf none` sau dintr-un storage
  PVE); amprenta sa este afișată. Apoi păstrați o copie în afara mașinii:
  **Print (paper key)** salvează o pagină imprimabilă cu cheia și codul ei QR (în
  formatul `proxmox-backup-client key paperkey`), opțional protejată prin frază de acces.
  **Import a key from text or a QR code** transformă un cod QR scanat sau o cheie pe hârtie
  înapoi într-un fișier cheie. GUI-ul folosește doar fișiere cheie neprotejate: importul unei
  chei protejate prin frază de acces o deblochează și salvează noul fișier cheie **fără**
  frază de acces — păstrați acel fișier la fel de bine ca pe cheia însăși.
- **CLI**: `-keyfile path/to/key.json` (sau `"keyfile"` în configurația JSON); o
  cheie protejată prin frază de acces primește `-keyfile-passphrase`, sau o cere interactiv.
- **Restaurare bare-metal**: ISO-ul Clonezilla modificat restaurează și backup-uri de disc
  criptate (cheie de pe un stick USB, cu fraza ei de acces, dacă există) — vedeți
  [PATCH-CLONEZILLA.md](PATCH-CLONEZILLA.md).
- **Compatibil cu `proxmox-backup-client`, verificat în CI la fiecare build**
  față de un PBS real: un director criptat de Nimbus Backup cu o cheie creată de
  `proxmox-backup-client` este restaurat de clientul oficial cu aceeași cheie
  (și refuzat fără ea), iar un backup de disc criptat realizat de clientul
  oficial este citit de Nimbus Backup. Proxmox VE folosește aceleași fișiere cheie.

> ⚠️ **Fără cheie, backup-urile criptate sunt irecuperabile.** Primul
> backup criptat încarcă totul din nou (fără deduplicare cu backup-urile
> necriptate). ID-urile backup-urilor, numele arhivelor și dimensiunile rămân vizibile serverului; conținutul
> fișierelor, numele fișierelor și catalogul sunt criptate.

## 🤖 Implementare automată (Ansible & IaC)

Nimbus Backup se instalează și se configurează integral din fișiere — fără configurare interactivă.
Două căi, ambele acoperite de exemple gata de folosit în
[`examples/automation/`](examples/automation/):

- **Linie de comandă** — un fișier JSON per host conține întregul backup
  (`proxmoxbackup-directory.exe --config file.json`); programarea se face prin
  Task Scheduler din Windows. Coduri de ieșire fiabile (`0` OK, `1` eroare fatală, `2` blocat, `3` parțial)
  pentru ca orchestratorul să detecteze eșecurile. Ideal pentru infrastructure-as-code pur.
- **GUI/serviciu (MSI)** — un **singur `config.json`** conține conexiunea PBS,
  setările de backup **și** programarea (`scheduled_jobs`). Distribuiți fișierul, reporniți
  serviciul `NimbusBackup`: sarcinile sunt reconciliate idempotent, iar `nextRun` este
  calculat automat. Niciun calcul de marcaje temporale în șablonul Jinja2.

Serviciul își citește configurația din `C:\ProgramData\ProxmoxBackupClient\`
(începând cu 0.4.0; vedeți [Actualizare](#️-actualizare-de-la--030)).

📖 Ghid complet: [Automate Windows backup to PBS with Ansible](https://nimbus.rdem-systems.com/en/blog/unattended-windows-backup-ansible/?utm_source=github).

## 🚀 Pornire rapidă

1. Descărcați `NimbusBackup.msi` (sau `NimbusBackup.exe` de sine stătător) din versiunile publicate
2. Instalați-l / rulați-l cu drepturi de administrator (necesare pentru VSS)
3. Configurați conexiunea PBS și testați-o
4. Selectați directoarele de salvat
5. Porniți backup-ul — sau programați-l

### 🔑 Utilizator PBS și permisiuni

Clientul are nevoie doar de rolul **`DatastoreBackup`** pe datastore-ul țintă — fără cont de administrator:

1. În interfața PBS, creați un utilizator (de ex. `nimbus@pbs`) și un token API pentru acesta (de ex. `nimbus@pbs!laptop01`).
2. În **Datastore → Permissions** (sau **Configuration → Access Control → Permissions**), acordați `DatastoreBackup` pe `/datastore/<name>` — sau pe `/datastore/<name>/<namespace>` dacă faceți backup într-un namespace.
3. **Token cu separarea privilegiilor** („Privilege Separation” bifat, implicit): acordați rolul **token-ului** însuși (`nimbus@pbs!laptop01`), nu doar utilizatorului — drepturile efective sunt intersecția celor două. Aceasta este cea mai frecventă cauză a erorii „permission denied”.

`DatastoreBackup` permite clientului să creeze backup-uri și să listeze și să restaureze propriile grupuri de backup. Ștergerea sau curățarea (prune) snapshot-urilor necesită `DatastorePowerUser`.

## ⬆️ Actualizare de la ≤ 0.3.0

Instalarea versiunii 0.4.0 sau mai noi peste o instalare existentă o actualizează pe loc (aceeași identitate MSI, același serviciu `NimbusBackup`). Directorul de date se mută din `C:\ProgramData\NimbusBackup` în directorul partajat `C:\ProgramData\ProxmoxBackupClient`: la prima pornire, configurația, sarcinile programate, istoricul și token-ul API sunt copiate o singură dată (fișierele existente nu sunt niciodată suprascrise). Vechiul director este păstrat, marcat cu `COPIED-TO-ProxmoxBackupClient.txt`, astfel încât un downgrade funcționează în continuare. Snapshot-urile realizate de versiunile mai vechi pot fi în continuare restaurate în locația lor originală.

## 📋 Cerințe

- Windows 10/11 sau Windows Server (64-bit)
- Drepturi de administrator (pentru snapshot-urile VSS)
- Acces de rețea la un Proxmox Backup Server

## 🔨 Compilare din sursă

### Cerințe preliminare
- Go 1.25 sau mai nou
- Node.js 20 sau mai nou
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0` (versiunea folosită de CI)

### Build
```bash
cd gui
npm install --prefix frontend
wails build      # or: wails dev  (hot reload)
```

Sau compilați totul (CLI + GUI + serviciu) cu Makefile-ul: `make install-deps && make` (vedeți `make help`). Toolchain Windows și cross-build cu Docker: [BUILD.md](BUILD.md).

Brandul este ales după numele executabilului: `NimbusBackup.exe` rulează ca Nimbus Backup, orice alt nume ca „Proxmox Backup Client” neutru ([`gui/brand.go`](gui/brand.go)).

## 🔗 Relația cu upstream

Nimbus Backup a pornit ca fork al [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go) (Proxmox Backup Client în Go, de Tiziano Bacocco, GPLv3), la care am adăugat GUI-ul Windows, serviciul, programarea, suportul multi-PBS și restaurarea. În septembrie 2026, upstream a integrat acest GUI și l-a făcut neutru din punct de vedere al brandului („Proxmox Backup Client GUI”). Începând cu 0.4.0, Nimbus Backup este construit din aceeași bază de cod: **cele două proiecte sunt acum aproape identice din punct de vedere funcțional.**

Ceea ce adaugă acest depozit peste upstream este o serie mică și documentată de patch-uri ([`patches/`](patches/README.md)):

- **Corecturi încă neintegrate în upstream** (build-ul serviciului, restaurare cu servere cu nume de utilizator/parolă, coduri de ieșire, mascarea datelor în jurnale, fiabilitatea backup-ului de mașină…) — trimise upstream pe măsură ce sunt integrate.
- **Identitatea Nimbus Backup** — `NimbusBackup.exe`/`.msi`, serviciul `NimbusBackup` și codul de actualizare MSI al instalărilor existente, astfel încât acestea să se actualizeze în continuare pe loc.
- **Calea de actualizare** de la Nimbus Backup ≤ 0.3.0 (migrarea directorului de date, metadate vechi ale snapshot-urilor).
- **Pipeline-ul de lansare** — atestare de proveniență a build-ului, sume de control, rapoarte VirusTotal.

Reintegrăm upstream în mod regulat; versiunile proprii ale upstream sunt publicate la [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go/releases).

## ⚠️ Declinarea responsabilității

Acest software este furnizat ca atare. Deși urmărim fiabilitatea, nu ne asumăm nicio responsabilitate pentru pierderi de date sau daune. Testați întotdeauna backup-urile și verificați restaurarea înainte de a vă baza pe ele în producție.

Acest proiect **nu este afiliat** cu **Proxmox Server Solutions GmbH**. „Proxmox” și denumirile asociate sunt proprietatea deținătorilor respectivi și sunt folosite aici doar pentru a indica compatibilitatea.

## 📄 Licență

GPLv3 — vedeți fișierul [LICENSE](LICENSE).

## Despre RDEM Systems

NimbusBackupClient este dezvoltat și întreținut de [RDEM Systems](https://www.rdem-systems.com/en/?utm_source=github), un furnizor francez de infrastructură specializat în servicii gestionate Proxmox VE/PBS și infrastructură NTP/NTS. Operăm [16 servere NTS publice](https://ntp.rdem-systems.com/en/nts.php?utm_source=github) ([stare în timp real](https://ntp.rdem-systems.com/en/status.php?utm_source=github); 11 dintre ele sunt listate în [lista de referință a comunității](https://github.com/jauderho/nts-servers)) și oferim [găzduire PBS complet gestionată](https://nimbus.rdem-systems.com/en/?utm_source=github) pentru utilizatorii care nu doresc să o găzduiască singuri. Întreținem de asemenea [unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client), pachete `proxmox-backup-client` semnate pentru distribuțiile Linux pe care Proxmox nu le suportă oficial.

---

**© 2024-2026 RDEM Systems and Proxmox Backup Client GO contributors.**
