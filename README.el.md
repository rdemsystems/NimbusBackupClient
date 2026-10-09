# Nimbus Backup — Πρόγραμμα-πελάτης Windows για Proxmox Backup Server

[🇬🇧 English](README.md) · [🇫🇷 Français](README.fr.md) · [🇮🇹 Italiano](README.it.md) · [🇩🇪 Deutsch](README.de.md) · [🇪🇸 Español](README.es.md) · [🇷🇺 Русский](README.ru.md) · [🇨🇳 中文](README.zh.md) · [🇯🇵 日本語](README.ja.md) · 🇬🇷 Ελληνικά · [🇷🇴 Română](README.ro.md) · [🇸🇪 Svenska](README.sv.md) · [🇸🇦 العربية](README.ar.md) · [🇮🇷 فارسی](README.fa.md)

> 🤖 Αυτή η μετάφραση δημιουργήθηκε από τεχνητή νοημοσύνη με βάση το αγγλικό README· σε περίπτωση απόκλισης υπερισχύει η [αγγλική έκδοση](README.md).
> *AI-generated translation of the English README; the [English version](README.md) is authoritative.*

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rdemsystems/NimbusBackupClient)](https://github.com/rdemsystems/NimbusBackupClient/releases)
[![Documentation](https://img.shields.io/badge/docs-nimbus.rdem--systems.com-orange)](https://nimbus.rdem-systems.com/en/?utm_source=github)

**Το Nimbus Backup είναι ένα πρόγραμμα-πελάτης αντιγράφων ασφαλείας ανοιχτού κώδικα (GPL-3.0) για Windows, για το Proxmox Backup Server (PBS).**
Ένα σύγχρονο γραφικό περιβάλλον για τη λήψη αντιγράφων ασφαλείας διακομιστών και σταθμών εργασίας Windows στο PBS — συνεπή στιγμιότυπα VSS, προγραμματισμένες εργασίες, λειτουργίες αρχείων και δίσκου, περιήγηση και επαναφορά στιγμιοτύπων, υποστήριξη πολλαπλών PBS και υπηρεσία Windows — καθώς και εργαλεία γραμμής εντολών για αντίγραφα ασφαλείας φακέλων και ολόκληρου μηχανήματος. Αναζητάτε αποθηκευτικό χώρο PBS **εκτός έδρας και αμετάβλητο** χωρίς δική σας φιλοξενία; Δείτε την [διαχειριζόμενη υπηρεσία](#️-διαχειριζόμενο-pbs-εκτός-έδρας--αμετάβλητο) παρακάτω.

Το Nimbus Backup είναι η έκδοση της RDEM Systems του [Proxmox Backup Client GO](https://github.com/tizbac/proxmoxbackupclient_go): το γραφικό περιβάλλον που αναπτύξαμε έχει ενσωματωθεί στο upstream, και τα δύο έργα μοιράζονται πλέον την ίδια βάση κώδικα (δείτε [Σχέση με το upstream](#-σχέση-με-το-upstream)).

📖 **Πλήρης τεκμηρίωση, οδηγός εγκατάστασης και φιλοξενία PBS:** [nimbus.rdem-systems.com](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github&utm_medium=readme&utm_campaign=nbc-readme-top)

## 📦 Λήψη

👉 **[Λήψη της τελευταίας έκδοσης](https://github.com/rdemsystems/NimbusBackupClient/releases)**

Κάθε έκδοση περιλαμβάνει:
- `NimbusBackup.msi` — πρόγραμμα εγκατάστασης (GUI + υπηρεσία Windows), **συνιστάται για παραγωγή**
- `NimbusBackup.exe` — αυτόνομο GUI
- `nimbus-backup-cli-<version>-windows.zip` / `-linux.tar.gz` / `-macos.tar.gz` — εργαλεία γραμμής εντολών
- `SHA256SUMS.txt` — αθροίσματα ελέγχου

> ⚠️ **Τα Windows αναφέρουν «εντοπίστηκε ιός» (π.χ. `Trojan:Win32/Sabsik.FL.A!ml`) ή εμφανίζουν προειδοποίηση SmartScreen;**
> Πρόκειται για γνωστό **ψευδώς θετικό** αποτέλεσμα για εφαρμογές Go/Wails — *δεν* είναι ιός. Η κατάληξη `!ml` σημαίνει ότι προέρχεται από μοντέλο μηχανικής μάθησης που επισημαίνει *σπάνια* εκτελέσιμα αρχεία (και, έως την 0.4.0, μη υπογεγραμμένα).
> Διαβάστε [γιατί συμβαίνει αυτό και πώς να επαληθεύσετε τη λήψη](https://nimbus.rdem-systems.com/en/antivirus-false-positive/?utm_source=github).

### 🔎 Επαλήθευση οποιασδήποτε λήψης

Κάθε έκδοση συνοδεύεται από αθροίσματα ελέγχου SHA-256 και μια υπογεγραμμένη **βεβαίωση προέλευσης build** (κρυπτογραφική απόδειξη ότι το εκτελέσιμο παρήχθη από το CI αυτού του αποθετηρίου, από συγκεκριμένο commit):

```powershell
Get-FileHash .\NimbusBackup.msi -Algorithm SHA256   # compare against SHA256SUMS.txt
gh attestation verify .\NimbusBackup.msi --repo rdemsystems/NimbusBackupClient
```

**VirusTotal.** Οι σημειώσεις κάθε έκδοσης παραπέμπουν στην αναφορά VirusTotal του προγράμματος εγκατάστασης του αντίστοιχου build (δημοσιεύεται μόνο όταν η σάρωση είναι καθαρή). Προηγούμενες αναφορές, 0 εντοπισμοί:
[0.2.108](https://www.virustotal.com/gui/file/6e8fb7ce9af740d470e947addb8daba4331c0b88e8bfdec9e0697ea8f7f29e9e/detection) ·
[0.2.107](https://www.virustotal.com/gui/file/6fd6c6fa77e0305c129ef882a3745100aa6033187a6d52a4af94149ab6b666d2/detection) ·
[0.2.106](https://www.virustotal.com/gui/file/ad6e56700ed9df8e088906e38cee2e2882fc7045f4e39269de0e379a01784ad7/detection)

> 🔏 **Υπογραφή κώδικα:** από την 0.4.1, τα `NimbusBackup.exe`, η υπηρεσία του και το `NimbusBackup.msi` **φέρουν υπογραφή Authenticode από την RDEM SYSTEMS** (Azure Artifact Signing)· η καρτέλα *Ιδιότητες → Ψηφιακές υπογραφές* εμφανίζει τον εκδότη. Το SmartScreen ενδέχεται να εξακολουθεί να προειδοποιεί για μια νέα έκδοση έως ότου καθιερωθεί η φήμη της: ελέγξτε ότι ο εκδότης είναι η RDEM SYSTEMS και, στη συνέχεια, *Περισσότερες πληροφορίες → Εκτέλεση οπωσδήποτε*. Τα εργαλεία γραμμής εντολών δεν είναι ακόμη υπογεγραμμένα· η βεβαίωση προέλευσης build και τα αθροίσματα ελέγχου παραπάνω καλύπτουν κάθε αρχείο.

### 🐧 Σε Linux; Χρησιμοποιήστε τον επίσημο client

Το GUI του Nimbus Backup είναι μόνο για Windows (τα εργαλεία CLI μεταγλωττίζονται επίσης για Linux και macOS). Για αντίγραφα ασφαλείας σε επίπεδο αρχείων σε Linux, χρησιμοποιήστε τον `proxmox-backup-client` της ίδιας της Proxmox — τον πακετάρουμε για τις διανομές που η Proxmox δεν καλύπτει:

👉 **[rdemsystems/unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)** — υπογεγραμμένα αποθετήρια πακέτων για Debian/Ubuntu, Fedora/RHEL/Rocky/AlmaLinux, Arch και Alpine (amd64 & arm64). Το επίσημο στατικό εκτελέσιμο της Proxmox, επανασυσκευασμένο χωρίς αλλαγές — χωρίς patches, χωρίς επαναμεταγλώττιση.

## ☁️ Διαχειριζόμενο PBS (εκτός έδρας & αμετάβλητο)

Δεν θέλετε να φιλοξενείτε μόνοι σας το Proxmox Backup Server; Χρησιμοποιήστε τα πλήρως διαχειριζόμενα, **εκτός έδρας και αμετάβλητα** datastores PBS μας:
👉 **[Διαμορφώστε το αντίγραφο ασφαλείας σας & δείτε τις τιμές](https://nimbus.rdem-systems.com/en/choose-backup/?utm_source=github)**

- ✅ Από 12 €/TB/μήνα
- ✅ Δωρεάν δοκιμή 1 TB
- ✅ Έτοιμος [προορισμός εκτός έδρας για το PBS σας](https://nimbus.rdem-systems.com/en/offsite-proxmox-backup/?utm_source=github) — απομονωμένα (air-gapped), αμετάβλητα datastores
- ✅ [NimbusBackup — Διαχειριζόμενη φιλοξενία PBS στη Γαλλία](https://nimbus.rdem-systems.com/en/?utm_source=github)

## 📚 Τεκμηρίωση

- **Πλήρης οδηγός Proxmox Backup** — βέλτιστες πρακτικές ανάπτυξης PBS ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/complete-proxmox-backup-guide/?utm_source=github))
- **Αντίγραφα ασφαλείας Windows με το Proxmox Backup Server** — οδηγός ανάπτυξης ειδικά για Windows ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github))
- **PBS vs Veeam** — σύγκριση με το Proxmox Backup Server ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/pbs-vs-veeam-proxmox-backup-comparison/?utm_source=github))
- Σε αυτό το αποθετήριο: [οδηγός χρήσης πολλαπλών PBS](MULTI_PBS_USER_GUIDE.md) · [παραδείγματα αυτόματης ανάπτυξης](examples/automation/) · [επαναφορά bare-metal με Clonezilla](PATCH-CLONEZILLA.md) · [αρχείο αλλαγών](CHANGELOG.md)

## ✨ Χαρακτηριστικά

### GUI (συνιστάται)
- **🌍 Πολυγλωσσικό** — περιβάλλον στα αγγλικά, γαλλικά, ιταλικά, γερμανικά και πολωνικά
- Εύχρηστη διαμόρφωση με έλεγχο σύνδεσης (API token ή όνομα χρήστη/κωδικός)
- Πρόοδος αντιγράφου ασφαλείας σε πραγματικό χρόνο με ταχύτητα και εκτιμώμενο χρόνο, ακύρωση ανά πάσα στιγμή, καρτέλα εργασιών σε εξέλιξη
- VSS (Volume Shadow Copy) για συνεπή αντίγραφα ασφαλείας: ένα στιγμιότυπο για ολόκληρο αντίγραφο πολλαπλών φακέλων, ένα σκιώδες αντίγραφο ανά τόμο
- Αντίγραφα ασφαλείας πολλαπλών φακέλων, λειτουργίες αρχείων και δίσκου (ολόκληρο μηχάνημα)· οι φάκελοι μπορούν να εκτελούνται παράλληλα (συνιστάται: CPU / 4)
- Τα αντίγραφα δίσκου επαναφέρονται στο Proxmox VE ως VM που αντιστοιχεί στο μηχάνημα (CPU, RAM, firmware, κάρτες δικτύου με τις διευθύνσεις MAC τους, αποκλειστικό VM ID)
- Περιήγηση στιγμιοτύπων, αναζήτηση αρχείων (wildcards) και επαναφορά, συμπεριλαμβανομένων των NTFS ACL
- Υποστήριξη πολλαπλών διακομιστών PBS με διακομιστή PBS ανά εργασία, καρφίτσωμα αποτυπώματος πιστοποιητικού (TOFU)
- **🔒 Κρυπτογράφηση στην πλευρά του client** (AES-256-GCM), αρχεία κλειδιών συμβατά με `proxmox-backup-client` και Proxmox VE
- Λειτουργία υπηρεσίας Windows + προγραμματισμένα αντίγραφα ασφαλείας, ιστορικό αντιγράφων με επανεκτέλεση με ένα κλικ
- Καταγραφή αποσφαλμάτωσης για την αντιμετώπιση προβλημάτων

### Εργαλεία γραμμής εντολών
- `proxmoxbackup-directory` — αντίγραφα ασφαλείας φακέλων (PXAR) με αποδιπλασιασμό, αντίγραφα ροής (`-backupstream`, π.χ. ένα pipe από `mysqldump`), εξαιρέσεις (`-exclude "*.tmp"`, επαναλαμβανόμενο, ή `-exclude-from file`· `"exclude"` στη διαμόρφωση JSON), πολλοί φάκελοι ταυτόχρονα (`-parallel N`, συνιστάται: CPU / 4), ειδοποιήσεις μέσω e-mail, αρχείο διαμόρφωσης JSON
- `proxmoxbackup-machine` — πλήρη αντίγραφα ασφαλείας μηχανήματος εν λειτουργία ως εκκινήσιμη εικόνα δίσκου (FIDX): VSS σε Windows, αυξητικά, παράλληλος υπολογισμός hash
- `proxmoxbackup-nbd` — διακομιστής NBD για προσάρτηση αντιγράφου δίσκου σε Linux (επαναφορά σε επίπεδο αρχείων, επαναφορά bare-metal από [τροποποιημένο live ISO του Clonezilla](PATCH-CLONEZILLA.md))

### 📸 Στιγμιότυπα οθόνης

![Server configuration](docs/screenshots/nimbus-gui-liste-servers.png)
*Διαχείριση πολλαπλών διακομιστών PBS με ενδείξεις κατάστασης*

![Add server form](docs/screenshots/nimbus-gui-add-server-form.png)
*Εύκολη διαμόρφωση διακομιστή με έλεγχο σύνδεσης*

![One-shot backup](docs/screenshots/nimbus-gui-one-shot-backup.png)
*Πρόοδος αντιγράφου ασφαλείας σε πραγματικό χρόνο με εκτιμώμενο χρόνο και ταχύτητα*

### Έξυπνες εξαιρέσεις συστήματος (λειτουργία αρχείων)
Κατά τη λήψη αντιγράφου ασφαλείας ολόκληρου δίσκου (π.χ. `D:\`), το Nimbus Backup εξαιρεί αυτόματα:

**Φάκελοι συστήματος:** `System Volume Information` (αποθήκευση VSS, μπορεί να ξεπερνά τα 100 GB), `$RECYCLE.BIN`, `Recovery`.
**Αρχεία συστήματος:** `pagefile.sys`, `hiberfil.sys`, `swapfile.sys`.

**Γιατί έχει σημασία:** ένας δίσκος μπορεί να αναφέρει 1,03 TB χρησιμοποιημένα ενώ τα πραγματικά αρχεία είναι ~141 GB. Χωρίς εξαιρέσεις, το αντίγραφο θα περιλάμβανε στιγμιότυπα VSS (σπατάλη χώρου και χρόνου)· με αυτές, το μέγεθος του αντιγράφου αντιστοιχεί στα πραγματικά δεδομένα.

**Σύσταση:** χρησιμοποιήστε τη **λειτουργία αρχείων** (προεπιλογή) με αυτόματες εξαιρέσεις για αντίγραφα σε επίπεδο αρχείων· χρησιμοποιήστε τη **λειτουργία δίσκου** σε ξεχωριστή εργασία για επαναφορά bare-metal (περιλαμβάνει τα πάντα).

### Ασφάλεια & ποιότητα
- Επικύρωση εισόδου και απολύμανση διαπιστευτηρίων (τα μυστικά αποκρύπτονται από τα αρχεία καταγραφής)
- Αποτροπή path traversal
- Λογική επανάληψης με εκθετική καθυστέρηση (exponential backoff)
- Έλεγχοι CI σε κάθε build: tests, `golangci-lint`, `gosec`, `go mod tidy`, καθώς και σουίτα end-to-end έναντι πραγματικού PBS (επαναφορές με το επίσημο `proxmox-backup-client`, PBS verify)

### 🔒 Κρυπτογράφηση στην πλευρά του client
Τα αντίγραφα ασφαλείας μπορούν να κρυπτογραφούνται **στον client** πριν φύγουν από το μηχάνημα, με
το ίδιο σχήμα με τον επίσημο `proxmox-backup-client` (AES-256-GCM, digests chunk
με κλειδί). Ο διακομιστής PBS αποθηκεύει μόνο αδιαφανή δεδομένα και δεν βλέπει ποτέ
το κλειδί — χρήσιμο σε κοινόχρηστο ή διαχειριζόμενο PBS.

Η κρυπτογράφηση αναπτύχθηκε στο αρχικό έργο από τον Tiziano Bacocco ([tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go))· το χάρτινο κλειδί, η εξαγωγή σε κωδικό QR και η εισαγωγή κλειδιού είναι προσθήκες του Nimbus Backup.

- **GUI**: *PBS Configuration → Edit → Encryption key file* — δημιουργήστε αρχείο κλειδιού ή επιλέξτε
  ένα υπάρχον (από `proxmox-backup-client key create --kdf none` ή από storage
  του PVE)· εμφανίζεται το αποτύπωμά του. Στη συνέχεια κρατήστε αντίγραφο εκτός του μηχανήματος:
  το **Print (paper key)** αποθηκεύει μια εκτυπώσιμη σελίδα με το κλειδί και τον κωδικό QR του (τη
  μορφή του `proxmox-backup-client key paperkey`), προαιρετικά προστατευμένη με φράση πρόσβασης.
  Το **Import a key from text or a QR code** μετατρέπει έναν σαρωμένο κωδικό QR ή ένα χάρτινο κλειδί
  ξανά σε αρχείο κλειδιού. Το GUI χρησιμοποιεί μόνο απροστάτευτα αρχεία κλειδιών: η εισαγωγή ενός
  κλειδιού με φράση πρόσβασης το ξεκλειδώνει και αποθηκεύει το νέο αρχείο κλειδιού **χωρίς**
  φράση πρόσβασης — φυλάξτε αυτό το αρχείο τόσο προσεκτικά όσο και το ίδιο το κλειδί.
- **CLI**: `-keyfile path/to/key.json` (ή `"keyfile"` στη διαμόρφωση JSON)· ένα
  κλειδί με φράση πρόσβασης δέχεται `-keyfile-passphrase`, διαφορετικά τη ζητά.
- **Επαναφορά bare-metal**: το τροποποιημένο ISO του Clonezilla επαναφέρει και κρυπτογραφημένα αντίγραφα
  δίσκου (κλειδί από USB stick, με τη φράση πρόσβασής του αν υπάρχει) — δείτε
  [PATCH-CLONEZILLA.md](PATCH-CLONEZILLA.md).
- **Συμβατό με `proxmox-backup-client`, επαληθευμένο στο CI σε κάθε build**
  έναντι πραγματικού PBS: ένας φάκελος κρυπτογραφημένος από το Nimbus Backup με κλειδί που δημιουργήθηκε από
  τον `proxmox-backup-client` επαναφέρεται από τον επίσημο client με το ίδιο κλειδί
  (και απορρίπτεται χωρίς αυτό), και ένα κρυπτογραφημένο αντίγραφο δίσκου από τον επίσημο
  client διαβάζεται από το Nimbus Backup. Το Proxmox VE χρησιμοποιεί τα ίδια αρχεία κλειδιών.

> ⚠️ **Χωρίς το κλειδί, τα κρυπτογραφημένα αντίγραφα ασφαλείας δεν ανακτώνται.** Το πρώτο
> κρυπτογραφημένο αντίγραφο ανεβάζει τα πάντα ξανά (χωρίς αποδιπλασιασμό με τα μη κρυπτογραφημένα
> αντίγραφα). Τα αναγνωριστικά αντιγράφων, τα ονόματα αρχειοθηκών και τα μεγέθη παραμένουν ορατά στον διακομιστή· το
> περιεχόμενο των αρχείων, τα ονόματα αρχείων και ο κατάλογος κρυπτογραφούνται.

## 🤖 Αυτόματη ανάπτυξη (Ansible & IaC)

Το Nimbus Backup εγκαθίσταται και διαμορφώνεται εξ ολοκλήρου μέσω αρχείων — χωρίς διαδραστική ρύθμιση.
Δύο τρόποι, και οι δύο καλύπτονται από έτοιμα παραδείγματα στο
[`examples/automation/`](examples/automation/):

- **Γραμμή εντολών** — ένα αρχείο JSON ανά host περιέχει ολόκληρο το αντίγραφο ασφαλείας
  (`proxmoxbackup-directory.exe --config file.json`)· ο προγραμματισμός γίνεται μέσω του
  Task Scheduler των Windows. Αξιόπιστοι κωδικοί εξόδου (`0` OK, `1` μοιραίο σφάλμα, `2` κλειδωμένο, `3` μερικό)
  ώστε ο orchestrator σας να εντοπίζει αποτυχίες. Ιδανικό για καθαρό infrastructure-as-code.
- **GUI/υπηρεσία (MSI)** — ένα **μοναδικό `config.json`** περιέχει τη σύνδεση PBS,
  τις ρυθμίσεις αντιγράφου ασφαλείας **και** το χρονοδιάγραμμα (`scheduled_jobs`). Προωθήστε το αρχείο, επανεκκινήστε
  την υπηρεσία `NimbusBackup`: οι εργασίες συμφιλιώνονται με idempotent τρόπο και το `nextRun`
  υπολογίζεται για εσάς. Κανένας υπολογισμός χρονοσήμων στο πρότυπο Jinja2 σας.

Η υπηρεσία διαβάζει τη διαμόρφωσή της από το `C:\ProgramData\ProxmoxBackupClient\`
(από την έκδοση 0.4.0· δείτε [Αναβάθμιση](#️-αναβάθμιση-από--030)).

📖 Πλήρης οδηγός: [Automate Windows backup to PBS with Ansible](https://nimbus.rdem-systems.com/en/blog/unattended-windows-backup-ansible/?utm_source=github).

## 🚀 Γρήγορη εκκίνηση

1. Κατεβάστε το `NimbusBackup.msi` (ή το αυτόνομο `NimbusBackup.exe`) από τις εκδόσεις
2. Εγκαταστήστε το / εκτελέστε το με δικαιώματα διαχειριστή (απαιτείται για το VSS)
3. Διαμορφώστε τη σύνδεση PBS και ελέγξτε την
4. Επιλέξτε τους φακέλους για αντίγραφο ασφαλείας
5. Ξεκινήστε το αντίγραφο ασφαλείας — ή προγραμματίστε το

### 🔑 Χρήστης PBS και δικαιώματα

Ο client χρειάζεται μόνο τον ρόλο **`DatastoreBackup`** στο datastore προορισμού — όχι λογαριασμό διαχειριστή:

1. Στο περιβάλλον του PBS, δημιουργήστε έναν χρήστη (π.χ. `nimbus@pbs`) και ένα API token για αυτόν (π.χ. `nimbus@pbs!laptop01`).
2. Στο **Datastore → Permissions** (ή **Configuration → Access Control → Permissions**), παραχωρήστε `DatastoreBackup` στο `/datastore/<name>` — ή στο `/datastore/<name>/<namespace>` αν κάνετε αντίγραφα σε namespace.
3. **Token με διαχωρισμό δικαιωμάτων** (επιλεγμένο το «Privilege Separation», η προεπιλογή): παραχωρήστε τον ρόλο στο ίδιο το **token** (`nimbus@pbs!laptop01`), όχι μόνο στον χρήστη — τα πραγματικά δικαιώματα είναι η τομή των δύο. Αυτή είναι η πιο συχνή αιτία του «permission denied».

Το `DatastoreBackup` επιτρέπει στον client να δημιουργεί αντίγραφα ασφαλείας και να παραθέτει και να επαναφέρει τις δικές του ομάδες αντιγράφων. Η διαγραφή ή το κλάδεμα (prune) στιγμιοτύπων απαιτεί `DatastorePowerUser`.

## ⬆️ Αναβάθμιση από ≤ 0.3.0

Η εγκατάσταση της 0.4.0 ή νεότερης πάνω σε υπάρχουσα εγκατάσταση την αναβαθμίζει επί τόπου (ίδια ταυτότητα MSI, ίδια υπηρεσία `NimbusBackup`). Ο φάκελος δεδομένων μετακινείται από το `C:\ProgramData\NimbusBackup` στο κοινόχρηστο `C:\ProgramData\ProxmoxBackupClient`: στην πρώτη εκκίνηση, η διαμόρφωση, οι προγραμματισμένες εργασίες, το ιστορικό και το API token σας αντιγράφονται μία φορά (τα υπάρχοντα αρχεία δεν αντικαθίστανται ποτέ). Ο παλιός φάκελος διατηρείται, σημασμένος με το `COPIED-TO-ProxmoxBackupClient.txt`, ώστε να λειτουργεί και η υποβάθμιση. Τα στιγμιότυπα που λήφθηκαν από παλαιότερες εκδόσεις μπορούν ακόμη να επαναφερθούν στην αρχική τους θέση.

## 📋 Απαιτήσεις

- Windows 10/11 ή Windows Server (64-bit)
- Δικαιώματα διαχειριστή (για στιγμιότυπα VSS)
- Πρόσβαση δικτύου σε ένα Proxmox Backup Server

## 🔨 Μεταγλώττιση από τον πηγαίο κώδικα

### Προαπαιτούμενα
- Go 1.25 ή νεότερο
- Node.js 20 ή νεότερο
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0` (η έκδοση που χρησιμοποιεί το CI)

### Build
```bash
cd gui
npm install --prefix frontend
wails build      # or: wails dev  (hot reload)
```

Ή μεταγλωττίστε τα πάντα (CLI + GUI + υπηρεσία) με το Makefile: `make install-deps && make` (δείτε `make help`). Εργαλεία Windows και cross-build με Docker: [BUILD.md](BUILD.md).

Το brand επιλέγεται από το όνομα του εκτελέσιμου: το `NimbusBackup.exe` εκτελείται ως Nimbus Backup, οποιοδήποτε άλλο όνομα ως το ουδέτερο «Proxmox Backup Client» ([`gui/brand.go`](gui/brand.go)).

## 🔗 Σχέση με το upstream

Το Nimbus Backup ξεκίνησε ως fork του [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go) (Proxmox Backup Client σε Go, από τον Tiziano Bacocco, GPLv3), στο οποίο προσθέσαμε το GUI για Windows, την υπηρεσία, τον προγραμματισμό, τα πολλαπλά PBS και την επαναφορά. Τον Σεπτέμβριο του 2026, το upstream ενσωμάτωσε αυτό το GUI και το έκανε ουδέτερο ως προς το brand («Proxmox Backup Client GUI»). Τον Οκτώβριο του 2026 (0.4.1), το Nimbus Backup ανακατασκευάστηκε πάνω στον τρέχοντα κώδικα του upstream, υιοθετώντας την κρυπτογράφηση στην πλευρά του client του upstream: **τα δύο έργα μοιράζονται την ίδια βάση κώδικα.**

Το Nimbus Backup 0.4.1 βασίζεται στον κλάδο `master` του αρχικού έργου, στο commit `3c1b989` (9 Οκτωβρίου 2026). Καμία έκδοση του αρχικού έργου δεν περιέχει ακόμη αυτόν τον κώδικα: η τελευταία, η v1.1.3 (Μάιος 2026), προηγείται της ενσωμάτωσης του GUI.

Αυτό που προσθέτει αυτό το αποθετήριο πάνω από το upstream είναι μια τεκμηριωμένη σειρά patches ([`patches/`](patches/README.md)), το μεγαλύτερο μέρος της οποίας έχει προταθεί στο upstream:

- **Λειτουργίες**: paper key και εισαγωγή κλειδιού (κωδικός QR), παράλληλα αντίγραφα ασφαλείας φακέλων, ένα στιγμιότυπο VSS ανά αντίγραφο πολλαπλών φακέλων, εξαιρέσεις στο CLI, διαμόρφωση VM του Proxmox VE που παράγεται από το πραγματικό μηχάνημα.
- **Διορθώσεις που δεν έχουν ακόμη ενσωματωθεί στο upstream** (επαναφορά των συμπιεσμένων κρυπτογραφημένων αντιγράφων του `proxmox-backup-client`, ο διακομιστής PBS που επιλέγεται για ένα αντίγραφο της υπηρεσίας, οι λόγοι απόρριψης του PBS, επαναφορά με διακομιστές όνομα χρήστη/κωδικού, μεγιστοποίηση παραθύρου…) — αποστέλλονται στο upstream καθώς ενσωματώνονται.
- **Η ταυτότητα Nimbus Backup** — `NimbusBackup.exe`/`.msi`, η υπηρεσία `NimbusBackup` και ο κωδικός αναβάθμισης MSI των υπαρχουσών εγκαταστάσεων, ώστε να συνεχίζουν να αναβαθμίζονται επί τόπου.
- **Διαδρομή αναβάθμισης** από Nimbus Backup ≤ 0.3.0 (μετεγκατάσταση φακέλου δεδομένων, παλαιά μεταδεδομένα στιγμιοτύπων).
- **Pipeline εκδόσεων** — βεβαίωση προέλευσης build, αθροίσματα ελέγχου, αναφορές VirusTotal.

Ενσωματώνουμε τακτικά ξανά το upstream· οι εκδόσεις του ίδιου του upstream δημοσιεύονται στο [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go/releases).

## ⚠️ Αποποίηση ευθύνης

Αυτό το λογισμικό παρέχεται ως έχει. Αν και επιδιώκουμε την αξιοπιστία, δεν φέρουμε καμία ευθύνη για απώλεια δεδομένων ή ζημιά. Ελέγχετε πάντα τα αντίγραφα ασφαλείας σας και επαληθεύετε την επαναφορά πριν βασιστείτε σε αυτά σε παραγωγή.

Αυτό το έργο **δεν συνδέεται** με την **Proxmox Server Solutions GmbH**. Το «Proxmox» και τα σχετικά ονόματα ανήκουν στους αντίστοιχους κατόχους τους και χρησιμοποιούνται εδώ μόνο για τη δήλωση συμβατότητας.

## 📄 Άδεια χρήσης

GPLv3 — δείτε το αρχείο [LICENSE](LICENSE).

## Σχετικά με την RDEM Systems

Το NimbusBackupClient αναπτύσσεται και συντηρείται από την [RDEM Systems](https://www.rdem-systems.com/en/?utm_source=github), γαλλικό πάροχο υποδομών εξειδικευμένο σε διαχειριζόμενες υπηρεσίες Proxmox VE/PBS και υποδομές NTP/NTS. Λειτουργούμε [16 δημόσιους διακομιστές NTS](https://ntp.rdem-systems.com/en/nts.php?utm_source=github) ([κατάσταση σε πραγματικό χρόνο](https://ntp.rdem-systems.com/en/status.php?utm_source=github)· 11 από αυτούς περιλαμβάνονται στην [κοινοτική λίστα αναφοράς](https://github.com/jauderho/nts-servers)) και παρέχουμε [πλήρως διαχειριζόμενη φιλοξενία PBS](https://nimbus.rdem-systems.com/en/?utm_source=github) για χρήστες που δεν θέλουν δική τους φιλοξενία. Συντηρούμε επίσης το [unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client), υπογεγραμμένα πακέτα `proxmox-backup-client` για διανομές Linux που η Proxmox δεν υποστηρίζει επίσημα.

---

**© 2024-2026 RDEM Systems and Proxmox Backup Client GO contributors.**
