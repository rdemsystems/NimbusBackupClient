# Nimbus Backup — client Windows per Proxmox Backup Server

[🇬🇧 English](README.md) · [🇫🇷 Français](README.fr.md) · 🇮🇹 Italiano · [🇩🇪 Deutsch](README.de.md) · [🇪🇸 Español](README.es.md) · [🇷🇺 Русский](README.ru.md) · [🇨🇳 中文](README.zh.md) · [🇯🇵 日本語](README.ja.md) · [🇬🇷 Ελληνικά](README.el.md) · [🇷🇴 Română](README.ro.md) · [🇸🇪 Svenska](README.sv.md) · [🇸🇦 العربية](README.ar.md) · [🇮🇷 فارسی](README.fa.md)

> 🤖 Questa traduzione è stata generata da un'IA a partire dal README inglese. In caso di discrepanze, fa fede la [versione inglese](README.md).
> *AI-generated translation of the English README; the [English version](README.md) is authoritative.*

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rdemsystems/NimbusBackupClient)](https://github.com/rdemsystems/NimbusBackupClient/releases)
[![Documentation](https://img.shields.io/badge/docs-nimbus.rdem--systems.com-orange)](https://nimbus.rdem-systems.com/en/?utm_source=github)

**Nimbus Backup è un client di backup per Windows open source (GPL-3.0) per Proxmox Backup Server (PBS).**
Un'interfaccia grafica moderna per eseguire il backup di server e workstation Windows su PBS — snapshot coerenti tramite VSS, job pianificati, modalità file e disco, navigazione e ripristino degli snapshot, supporto multi-PBS e un servizio Windows — oltre a strumenti da riga di comando per il backup di directory e di macchine intere. Cercate uno storage PBS **offsite e immutabile** senza doverlo ospitare voi stessi? Consultate il [servizio gestito](#️-pbs-gestito-offsite-e-immutabile) più sotto.

Nimbus Backup è la build di RDEM Systems di [Proxmox Backup Client GO](https://github.com/tizbac/proxmoxbackupclient_go): l'interfaccia grafica che abbiamo sviluppato è stata integrata upstream e i due progetti condividono ora la stessa base di codice (vedere [Rapporto con il progetto upstream](#-rapporto-con-il-progetto-upstream)).

📖 **Documentazione completa, guida all'installazione e hosting PBS:** [nimbus.rdem-systems.com](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github&utm_medium=readme&utm_campaign=nbc-readme-top)

## 📦 Download

👉 **[Scarica l'ultima release](https://github.com/rdemsystems/NimbusBackupClient/releases)**

Ogni release include:
- `NimbusBackup.msi` — programma di installazione (GUI + servizio Windows), **consigliato in produzione**
- `NimbusBackup.exe` — GUI standalone
- `nimbus-backup-cli-<version>-windows.zip` / `-linux.tar.gz` / `-macos.tar.gz` — strumenti da riga di comando
- `SHA256SUMS.txt` — checksum

> ⚠️ **Windows segnala "virus rilevato" (ad es. `Trojan:Win32/Sabsik.FL.A!ml`) o mostra un avviso SmartScreen?**
> Si tratta di un **falso positivo** noto per le applicazioni Go/Wails — *non* è un virus. Il suffisso `!ml` indica che proviene da un modello di machine learning che segnala gli eseguibili *non firmati e poco diffusi*.
> Leggete [perché succede e come verificare il download](https://nimbus.rdem-systems.com/en/antivirus-false-positive/?utm_source=github).

### 🔎 Verificare un download

Ogni release include i checksum SHA-256 e un'**attestazione di provenienza della build** firmata (prova crittografica che il binario è stato prodotto dalla CI di questo repository, a partire da un commit specifico):

```powershell
Get-FileHash .\NimbusBackup.msi -Algorithm SHA256   # compare against SHA256SUMS.txt
gh attestation verify .\NimbusBackup.msi --repo rdemsystems/NimbusBackupClient
```

**VirusTotal.** Le note di ogni release rimandano al report VirusTotal del programma di installazione di quella build (pubblicato solo se la scansione è pulita). Report precedenti, 0 rilevamenti:
[0.2.108](https://www.virustotal.com/gui/file/6e8fb7ce9af740d470e947addb8daba4331c0b88e8bfdec9e0697ea8f7f29e9e/detection) ·
[0.2.107](https://www.virustotal.com/gui/file/6fd6c6fa77e0305c129ef882a3745100aa6033187a6d52a4af94149ab6b666d2/detection) ·
[0.2.106](https://www.virustotal.com/gui/file/ad6e56700ed9df8e088906e38cee2e2882fc7045f4e39269de0e379a01784ad7/detection)

> ℹ️ **Firma del codice:** i binari Windows **non sono ancora firmati Authenticode**, ed è questo che provoca gli avvisi SmartScreen / `!ml` descritti sopra. La nostra richiesta di un certificato OSS gratuito alla [SignPath Foundation](https://signpath.org) non ha ricevuto risposta; la firma tramite Azure Artifact Signing è in fase di configurazione ed è prevista per la **0.4.1**. Fino ad allora, la provenienza è garantita dall'attestazione di provenienza della build e dai checksum indicati sopra.

### 🐧 Su Linux? Usate il client ufficiale

La GUI di Nimbus Backup è disponibile solo per Windows (gli strumenti CLI si compilano anche per Linux e macOS). Per i backup a livello di file su Linux, usate il `proxmox-backup-client` di Proxmox — lo pacchettizziamo per le distribuzioni non coperte da Proxmox:

👉 **[rdemsystems/unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)** — repository di pacchetti firmati per Debian/Ubuntu, Fedora/RHEL/Rocky/AlmaLinux, Arch e Alpine (amd64 e arm64). Il binario statico ufficiale di Proxmox, ripacchettizzato senza modifiche — né patchato né ricompilato.

## ☁️ PBS gestito (offsite e immutabile)

Non volete ospitare voi stessi un Proxmox Backup Server? Usate i nostri datastore PBS completamente gestiti, **offsite e immutabili**:
👉 **[Configurate il vostro backup e consultate i prezzi](https://nimbus.rdem-systems.com/en/choose-backup/?utm_source=github)**

- ✅ Da 12 €/TB/mese
- ✅ Prova gratuita di 1 TB
- ✅ Una [destinazione offsite per il vostro PBS](https://nimbus.rdem-systems.com/en/offsite-proxmox-backup/?utm_source=github) pronta all'uso — datastore isolati (air-gapped) e immutabili
- ✅ [NimbusBackup — Hosting PBS gestito in Francia](https://nimbus.rdem-systems.com/en/?utm_source=github)

## 📚 Documentazione

- **Guida completa a Proxmox Backup** — buone pratiche per il deployment di PBS ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/complete-proxmox-backup-guide/?utm_source=github))
- **Backup di Windows con Proxmox Backup Server** — guida al deployment specifica per Windows ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github))
- **PBS vs Veeam** — confronto con Proxmox Backup Server ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/pbs-vs-veeam-proxmox-backup-comparison/?utm_source=github))
- In questo repository: [guida utente multi-PBS](MULTI_PBS_USER_GUIDE.md) · [esempi di deployment automatizzato](examples/automation/) · [ripristino bare-metal con Clonezilla](PATCH-CLONEZILLA.md) · [changelog](CHANGELOG.md)

## ✨ Funzionalità

### GUI (consigliata)
- **🌍 Multilingue** — interfaccia in inglese, francese, italiano, tedesco e polacco
- Configurazione semplice con test della connessione (token API o nome utente/password)
- Avanzamento del backup in tempo reale con velocità e tempo stimato, annullabile in qualsiasi momento
- Supporto VSS (Volume Shadow Copy) per backup coerenti
- Backup di più cartelle, modalità file e disco (macchina intera); le cartelle possono essere elaborate in parallelo (consigliato: CPU / 4)
- Navigazione degli snapshot, ricerca di file (caratteri jolly) e ripristino
- Supporto di più server PBS, pinning dell'impronta del certificato (TOFU)
- **🔒 Cifratura lato client** (AES-256-GCM), file di chiave compatibili con `proxmox-backup-client` e Proxmox VE
- Modalità servizio Windows + backup pianificati, cronologia dei backup con riesecuzione in un clic
- Log di debug per la risoluzione dei problemi

### Strumenti da riga di comando
- `proxmoxbackup-directory` — backup di directory (PXAR) con deduplicazione, backup di flussi (`-backupstream`, ad es. una pipe `mysqldump`), esclusioni (`-exclude "*.tmp"`, ripetibile, oppure `-exclude-from file`; `"exclude"` nella configurazione JSON), più directory contemporaneamente (`-parallel N`, consigliato: CPU / 4), notifiche e-mail, file di configurazione JSON
- `proxmoxbackup-machine` — backup completi a caldo della macchina come immagine disco avviabile (FIDX): VSS su Windows, incrementale, hashing parallelo
- `proxmoxbackup-nbd` — server NBD per montare un backup disco su Linux (ripristino a livello di file, ripristino bare-metal da una [ISO live di Clonezilla patchata](PATCH-CLONEZILLA.md))

### 📸 Screenshot

![Server configuration](docs/screenshots/nimbus-gui-liste-servers.png)
*Gestione di più server PBS con indicatori di stato*

![Add server form](docs/screenshots/nimbus-gui-add-server-form.png)
*Configurazione semplice del server con test della connessione*

![One-shot backup](docs/screenshots/nimbus-gui-one-shot-backup.png)
*Avanzamento del backup in tempo reale con tempo stimato e velocità*

### Esclusioni di sistema intelligenti (modalità file)
Quando si esegue il backup di un'intera unità (ad es. `D:\`), Nimbus Backup esclude automaticamente:

**Cartelle di sistema:** `System Volume Information` (archivio VSS, può superare i 100 GB), `$RECYCLE.BIN`, `Recovery`.
**File di sistema:** `pagefile.sys`, `hiberfil.sys`, `swapfile.sys`.

**Perché è importante:** un'unità può indicare 1,03 TB occupati mentre i file reali sono ~141 GB. Senza esclusioni il backup includerebbe gli snapshot VSS (spreco di spazio e di tempo); con le esclusioni la dimensione del backup corrisponde ai dati reali.

**Raccomandazione:** usate la **modalità file** (predefinita) con le esclusioni automatiche per i backup a livello di file; usate la **modalità disco** in un job separato per il ripristino bare-metal (include tutto).

### Sicurezza e qualità
- Validazione degli input e sanificazione delle credenziali (segreti oscurati nei log)
- Prevenzione del path traversal
- Logica di retry con backoff esponenziale
- Controlli CI a ogni build: test, `golangci-lint`, `gosec`, `go mod tidy`

### 🔒 Cifratura lato client
I backup possono essere cifrati **sul client** prima di lasciare la macchina, con
lo stesso schema del `proxmox-backup-client` ufficiale (AES-256-GCM, digest dei chunk
con chiave). Il server PBS memorizza solo dati opachi e non vede mai
la chiave — utile su un PBS condiviso o gestito.

- **GUI**: *Server → Modifica → Chiave di cifratura* — create un file di chiave o selezionatene uno
  esistente (da `proxmox-backup-client key create --kdf none` o da uno storage
  PVE); ne viene mostrata l'impronta. Conservatene poi una copia fuori dalla macchina:
  **Stampa (chiave cartacea)** salva una pagina stampabile con la chiave e il relativo codice QR (il
  formato di `proxmox-backup-client key paperkey`), facoltativamente protetta da passphrase.
  **Importa una chiave da testo o da codice QR** trasforma un codice QR scansionato o una chiave cartacea
  di nuovo in un file di chiave. La GUI usa solo file di chiave non protetti: l'importazione di una
  chiave protetta da passphrase la sblocca e salva il nuovo file di chiave **senza**
  passphrase — custodite quel file con la stessa cura della chiave stessa.
- **CLI**: `-keyfile path/to/key.json` (oppure `"keyfile"` nella configurazione JSON); una
  chiave protetta da passphrase richiede `-keyfile-passphrase`, altrimenti la passphrase viene chiesta interattivamente.
- **Ripristino bare-metal**: la ISO di Clonezilla patchata ripristina anche i backup disco
  cifrati (chiave da una chiavetta USB, con la relativa passphrase se presente) — vedere
  [PATCH-CLONEZILLA.md](PATCH-CLONEZILLA.md).
- **Compatibile con `proxmox-backup-client`, verificato in CI a ogni build**
  su un PBS reale: una cartella cifrata da Nimbus Backup con una chiave creata da
  `proxmox-backup-client` viene ripristinata dal client ufficiale con la stessa chiave
  (e rifiutata senza di essa), e un backup disco cifrato creato dal client
  ufficiale viene riletto da Nimbus Backup. Proxmox VE usa gli stessi file di chiave.

> ⚠️ **Senza la chiave, i backup cifrati sono irrecuperabili.** Il primo
> backup cifrato ricarica tutti i dati (nessuna deduplicazione con i backup
> non cifrati). Gli ID dei backup, i nomi degli archivi e le dimensioni restano visibili al server; il
> contenuto dei file, i nomi dei file e il catalogo sono cifrati.

## 🤖 Deployment automatizzato (Ansible e IaC)

Nimbus Backup si installa e si configura interamente tramite file — nessuna configurazione interattiva.
Due approcci, entrambi coperti da esempi pronti all'uso in
[`examples/automation/`](examples/automation/):

- **Riga di comando** — un file JSON per host contiene l'intero backup
  (`proxmoxbackup-directory.exe --config file.json`); pianificazione tramite l'Utilità di pianificazione
  di Windows. Codici di uscita affidabili (`0` OK, `1` errore fatale, `2` bloccato, `3` parziale)
  affinché il vostro orchestratore rilevi gli errori. Ideale per un approccio puramente infrastructure-as-code.
- **GUI/servizio (MSI)** — un **unico `config.json`** contiene la connessione PBS,
  le impostazioni di backup **e** la pianificazione (`scheduled_jobs`). Distribuite il file, riavviate
  il servizio `NimbusBackup`: i job vengono riconciliati in modo idempotente e `nextRun` viene
  calcolato automaticamente. Nessun calcolo di timestamp nel vostro template Jinja2.

Il servizio legge la propria configurazione da `C:\ProgramData\ProxmoxBackupClient\`
(dalla 0.4.0; vedere [Aggiornamento](#️-aggiornamento-da--030)).

📖 Guida completa: [Automate Windows backup to PBS with Ansible](https://nimbus.rdem-systems.com/en/blog/unattended-windows-backup-ansible/?utm_source=github).

## 🚀 Avvio rapido

1. Scaricate `NimbusBackup.msi` (o il `NimbusBackup.exe` standalone) dalle release
2. Installatelo / eseguitelo con privilegi di amministratore (necessari per VSS)
3. Configurate la connessione al PBS e testatela
4. Selezionate le directory di cui eseguire il backup
5. Avviate il backup — oppure pianificatelo

### 🔑 Utente PBS e permessi

Il client necessita solo del ruolo **`DatastoreBackup`** sul datastore di destinazione — nessun account amministratore:

1. Nell'interfaccia di PBS, create un utente (ad es. `nimbus@pbs`) e un token API associato (ad es. `nimbus@pbs!laptop01`).
2. In **Datastore → Permissions** (oppure **Configuration → Access Control → Permissions**), assegnate `DatastoreBackup` su `/datastore/<name>` — oppure su `/datastore/<name>/<namespace>` se eseguite il backup in un namespace.
3. **Token con separazione dei privilegi** ("Privilege Separation" selezionato, impostazione predefinita): assegnate il ruolo al **token** stesso (`nimbus@pbs!laptop01`), non solo all'utente — i diritti effettivi sono l'intersezione dei due. È la causa più comune di "permission denied".

`DatastoreBackup` consente al client di creare backup e di elencare e ripristinare i propri gruppi di backup. L'eliminazione o il pruning degli snapshot richiede `DatastorePowerUser`.

## ⬆️ Aggiornamento da ≤ 0.3.0

L'installazione della 0.4.0 o successive su un'installazione esistente la aggiorna sul posto (stessa identità MSI, stesso servizio `NimbusBackup`). La cartella dei dati passa da `C:\ProgramData\NimbusBackup` alla cartella condivisa `C:\ProgramData\ProxmoxBackupClient`: al primo avvio, configurazione, job pianificati, cronologia e token API vengono copiati una sola volta (i file esistenti non vengono mai sovrascritti). La vecchia cartella viene conservata, contrassegnata con `COPIED-TO-ProxmoxBackupClient.txt`, così un downgrade resta possibile. Gli snapshot creati dalle versioni precedenti possono ancora essere ripristinati nella loro posizione originale.

## 📋 Requisiti

- Windows 10/11 o Windows Server (64 bit)
- Diritti di amministratore (per gli snapshot VSS)
- Accesso di rete a un Proxmox Backup Server

## 🔨 Compilazione dai sorgenti

### Prerequisiti
- Go 1.25 o successivo
- Node.js 20 o successivo
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0` (la versione usata dalla CI)

### Compilazione
```bash
cd gui
npm install --prefix frontend
wails build      # or: wails dev  (hot reload)
```

Oppure compilate tutto (CLI + GUI + servizio) con il Makefile: `make install-deps && make` (vedere `make help`). Toolchain Windows e cross-compilazione con Docker: [BUILD.md](BUILD.md).

Il marchio viene scelto in base al nome dell'eseguibile: `NimbusBackup.exe` si avvia come Nimbus Backup, qualsiasi altro nome come il neutro "Proxmox Backup Client" ([`gui/brand.go`](gui/brand.go)).

## 🔗 Rapporto con il progetto upstream

Nimbus Backup è nato come fork di [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go) (Proxmox Backup Client in Go, di Tiziano Bacocco, GPLv3), a cui abbiamo aggiunto la GUI Windows, il servizio, la pianificazione, il multi-PBS e il ripristino. Nel settembre 2026, il progetto upstream ha integrato quella GUI rendendola neutra rispetto al marchio ("Proxmox Backup Client GUI"). Dalla 0.4.0, Nimbus Backup è compilato a partire dalla stessa base di codice: **i due progetti sono ora quasi identici dal punto di vista funzionale.**

Ciò che questo repository aggiunge rispetto all'upstream è una piccola serie di patch documentata ([`patches/`](patches/README.md)):

- **Correzioni non ancora integrate upstream** (build del servizio, ripristino con server a nome utente/password, codici di uscita, oscuramento dei log, affidabilità del backup della macchina…) — inviate upstream man mano che vengono integrate.
- **L'identità Nimbus Backup** — `NimbusBackup.exe`/`.msi`, il servizio `NimbusBackup` e l'upgrade code MSI delle installazioni esistenti, affinché continuino ad aggiornarsi sul posto.
- **Percorso di aggiornamento** da Nimbus Backup ≤ 0.3.0 (migrazione della cartella dei dati, metadati degli snapshot legacy).
- **Pipeline di release** — attestazione di provenienza della build, checksum, report VirusTotal.

Reintegriamo regolarmente l'upstream; le release proprie dell'upstream sono pubblicate su [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go/releases).

## ⚠️ Esclusione di responsabilità

Questo software è fornito così com'è. Pur impegnandoci per garantirne l'affidabilità, non ci assumiamo alcuna responsabilità per eventuali perdite o danni ai dati. Testate sempre i vostri backup e verificate il ripristino prima di farvi affidamento in produzione.

Questo progetto **non è affiliato** a **Proxmox Server Solutions GmbH**. "Proxmox" e i nomi correlati sono di proprietà dei rispettivi titolari e sono qui utilizzati solo per indicare la compatibilità.

## 📄 Licenza

GPLv3 — vedere il file [LICENSE](LICENSE).

## Informazioni su RDEM Systems

NimbusBackupClient è sviluppato e mantenuto da [RDEM Systems](https://www.rdem-systems.com/en/?utm_source=github), un fornitore di infrastrutture francese specializzato in servizi gestiti Proxmox VE/PBS e in infrastrutture NTP/NTS. Gestiamo [16 server NTS pubblici](https://ntp.rdem-systems.com/en/nts.php?utm_source=github) ([stato in tempo reale](https://ntp.rdem-systems.com/en/status.php?utm_source=github); 11 di essi figurano nel [riferimento della comunità](https://github.com/jauderho/nts-servers)) e offriamo un [hosting PBS completamente gestito](https://nimbus.rdem-systems.com/en/?utm_source=github) per chi non vuole ospitarlo in proprio. Manteniamo inoltre [unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client), pacchetti `proxmox-backup-client` firmati per le distribuzioni Linux che Proxmox non supporta ufficialmente.

---

**© 2024-2026 RDEM Systems and Proxmox Backup Client GO contributors.**
