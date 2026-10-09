# Nimbus Backup — Client Windows pour Proxmox Backup Server

[🇬🇧 English](README.md) · 🇫🇷 Français · [🇮🇹 Italiano](README.it.md) · [🇩🇪 Deutsch](README.de.md) · [🇪🇸 Español](README.es.md) · [🇷🇺 Русский](README.ru.md) · [🇨🇳 中文](README.zh.md) · [🇯🇵 日本語](README.ja.md) · [🇬🇷 Ελληνικά](README.el.md) · [🇷🇴 Română](README.ro.md) · [🇸🇪 Svenska](README.sv.md) · [🇸🇦 العربية](README.ar.md) · [🇮🇷 فارسی](README.fa.md)

[![Licence](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rdemsystems/NimbusBackupClient)](https://github.com/rdemsystems/NimbusBackupClient/releases)
[![Documentation](https://img.shields.io/badge/docs-nimbus.rdem--systems.com-orange)](https://nimbus.rdem-systems.com/?utm_source=github)

**Nimbus Backup est un client de sauvegarde Windows open-source (GPL-3.0) pour Proxmox Backup Server (PBS).**
Une interface graphique moderne pour sauvegarder serveurs et postes Windows vers PBS — snapshots cohérents via VSS, tâches planifiées, modes fichier et disque, navigation/restauration de snapshots, support multi-PBS et mode service Windows — ainsi que des outils en ligne de commande pour la sauvegarde de dossiers et de machines complètes. Besoin d'un stockage PBS **déporté et immuable** sans auto-héberger ? Voir le [service infogéré](#️-pbs-infogéré-déporté--immuable) ci-dessous.

Nimbus Backup est la version RDEM Systems de [Proxmox Backup Client GO](https://github.com/tizbac/proxmoxbackupclient_go) : l'interface graphique que nous avons développée a été intégrée au projet d'origine, et les deux projets partagent désormais le même code (voir [Relation avec le projet d'origine](#-relation-avec-le-projet-dorigine)).

📖 **Documentation complète, guide d'installation et hébergement PBS :** [nimbus.rdem-systems.com](https://nimbus.rdem-systems.com/blog/sauvegarder-windows-proxmox-backup-server/?utm_source=github&utm_medium=readme&utm_campaign=nbc-readme-top)

## 📦 Téléchargement

👉 **[Télécharger la dernière version](https://github.com/rdemsystems/NimbusBackupClient/releases)**

Chaque release contient :
- `NimbusBackup.msi` — installeur (GUI + service Windows), **recommandé en production**
- `NimbusBackup.exe` — GUI autonome
- `nimbus-backup-cli-<version>-windows.zip` / `-linux.tar.gz` / `-macos.tar.gz` — outils en ligne de commande
- `SHA256SUMS.txt` — empreintes

> ⚠️ **Windows affiche « virus détecté » (ex. `Trojan:Win32/Sabsik.FL.A!ml`) ou un avertissement SmartScreen ?**
> C'est un **faux positif** connu pour les applications Go/Wails — ce n'est *pas* un virus. Le suffixe `!ml` indique une détection par un modèle de machine learning qui signale les exécutables *non signés et peu répandus*.
> Lisez [pourquoi cela arrive et comment vérifier le téléchargement](https://nimbus.rdem-systems.com/faux-positif-antivirus/?utm_source=github).

### 🔎 Vérifier n'importe quel téléchargement

Chaque release fournit des empreintes SHA-256 et une **attestation de provenance signée** (preuve cryptographique que le binaire a été produit par la CI de ce dépôt, à partir d'un commit précis) :

```powershell
Get-FileHash .\NimbusBackup.msi -Algorithm SHA256   # comparer avec SHA256SUMS.txt
gh attestation verify .\NimbusBackup.msi --repo rdemsystems/NimbusBackupClient
```

**VirusTotal.** Les notes de chaque release donnent le lien vers le rapport VirusTotal de l'installeur de cette version (publié uniquement si l'analyse est propre). Rapports précédents, 0 détection :
[0.2.108](https://www.virustotal.com/gui/file/6e8fb7ce9af740d470e947addb8daba4331c0b88e8bfdec9e0697ea8f7f29e9e/detection) ·
[0.2.107](https://www.virustotal.com/gui/file/6fd6c6fa77e0305c129ef882a3745100aa6033187a6d52a4af94149ab6b666d2/detection) ·
[0.2.106](https://www.virustotal.com/gui/file/ad6e56700ed9df8e088906e38cee2e2882fc7045f4e39269de0e379a01784ad7/detection)

> ℹ️ **Signature de code :** les binaires Windows ne sont **pas encore signés Authenticode**, ce qui déclenche les alertes SmartScreen / `!ml` ci-dessus. Notre demande de certificat OSS gratuit auprès de la [SignPath Foundation](https://signpath.org) est restée sans réponse ; la signature via Azure Artifact Signing est en cours de mise en place, visée pour la **0.4.1**. En attendant, la provenance est établie via l'attestation de build et les empreintes ci-dessus.

### 🐧 Sous Linux ? Utilisez le client officiel

L'interface graphique Nimbus Backup est réservée à Windows (les outils en ligne de commande se compilent aussi pour Linux et macOS). Pour des sauvegardes de fichiers sous Linux, utilisez le `proxmox-backup-client` de Proxmox — nous l'empaquetons pour les distributions que Proxmox ne couvre pas :

👉 **[rdemsystems/unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)** — dépôts de paquets signés pour Debian/Ubuntu, Fedora/RHEL/Rocky/AlmaLinux, Arch et Alpine (amd64 & arm64). Le binaire statique officiel de Proxmox, réempaqueté tel quel — ni modifié, ni recompilé.

## ☁️ PBS infogéré (déporté & immuable)

Vous ne voulez pas auto-héberger Proxmox Backup Server ? Utilisez nos datastores PBS entièrement infogérés, **déportés et immuables** :
👉 **[Configurez votre sauvegarde & voir les tarifs](https://nimbus.rdem-systems.com/choisir-mon-backup/?utm_source=github)**

- ✅ À partir de 12 €/To/mois
- ✅ 1 To d'essai gratuit
- ✅ Une [cible de sauvegarde externalisée pour votre PBS](https://nimbus.rdem-systems.com/sauvegarde-proxmox-externalisee/?utm_source=github) — datastores immuables et isolés
- ✅ [NimbusBackup — hébergement PBS infogéré en France](https://nimbus.rdem-systems.com/?utm_source=github)

## 📚 Documentation

- **Guide complet de sauvegarde Proxmox** — bonnes pratiques de déploiement PBS ([🇫🇷 FR](https://nimbus.rdem-systems.com/blog/guide-complet-backup-proxmox/?utm_source=github))
- **Sauvegarder Windows avec Proxmox Backup Server** — guide de déploiement spécifique Windows ([🇫🇷 FR](https://nimbus.rdem-systems.com/blog/sauvegarder-windows-proxmox-backup-server/?utm_source=github))
- **PBS vs Veeam** — comparatif backup Proxmox ([🇫🇷 FR](https://nimbus.rdem-systems.com/blog/pbs-vs-veeam-comparatif-backup-proxmox/?utm_source=github))
- Dans ce dépôt (en anglais) : [guide multi-PBS](MULTI_PBS_USER_GUIDE.md) · [exemples de déploiement sans surveillance](examples/automation/) · [restauration bare-metal avec Clonezilla](PATCH-CLONEZILLA.md) · [changelog](CHANGELOG.md)

## ✨ Fonctionnalités

### Interface graphique (recommandée)
- **🌍 Multilingue** — interface en français, anglais, italien, allemand et polonais
- Configuration conviviale avec test de connexion (jeton API ou identifiant/mot de passe)
- Progression de sauvegarde en temps réel avec débit et temps restant, annulation à tout moment
- Support VSS (Volume Shadow Copy) pour des sauvegardes cohérentes
- Sauvegarde multi-dossiers, modes fichier et disque (machine complète) ; dossiers sauvegardables en parallèle (recommandé : nombre de CPU / 4)
- Navigation dans les snapshots, recherche de fichiers (jokers) et restauration
- Support multi-serveurs PBS, épinglage d'empreinte de certificat (TOFU)
- **🔒 Chiffrement côté client** (AES-256-GCM), fichiers de clé compatibles avec `proxmox-backup-client` et Proxmox VE
- Mode service Windows + sauvegardes planifiées, historique avec relance en un clic
- Journalisation de débogage pour le diagnostic

### Outils en ligne de commande
- `proxmoxbackup-directory` — sauvegarde de dossiers (PXAR) avec déduplication, sauvegarde de flux (`-backupstream`, ex. un `mysqldump` en pipe), exclusions (`-exclude "*.tmp"`, répétable, ou `-exclude-from fichier` ; `"exclude"` dans la config JSON), plusieurs dossiers à la fois (`-parallel N`, recommandé : nombre de CPU / 4), notifications par e-mail, fichier de configuration JSON
- `proxmoxbackup-machine` — sauvegarde à chaud d'une machine complète en image disque amorçable (FIDX) : VSS sous Windows, incrémentale, hachage parallélisé
- `proxmoxbackup-nbd` — serveur NBD pour monter une sauvegarde disque sous Linux (restauration de fichiers, restauration bare-metal depuis une [ISO Clonezilla live patchée](PATCH-CLONEZILLA.md))

### 📸 Captures d'écran

![Configuration des serveurs](docs/screenshots/nimbus-gui-liste-servers.png)
*Gestion multi-serveurs PBS avec indicateurs d'état*

![Formulaire d'ajout de serveur](docs/screenshots/nimbus-gui-add-server-form.png)
*Configuration de serveur simple avec test de connexion*

![Sauvegarde immédiate](docs/screenshots/nimbus-gui-one-shot-backup.png)
*Progression de sauvegarde en temps réel avec ETA et débit*

### Exclusions système intelligentes (mode fichier)
Lors de la sauvegarde d'un disque entier (ex. `D:\`), Nimbus Backup exclut automatiquement :

**Dossiers système :** `System Volume Information` (stockage VSS, peut atteindre 100+ Go), `$RECYCLE.BIN`, `Recovery`.
**Fichiers système :** `pagefile.sys`, `hiberfil.sys`, `swapfile.sys`.

**Pourquoi c'est important :** un disque peut afficher 1,03 To utilisés alors que les fichiers réels font ~141 Go. Sans exclusions, la sauvegarde inclurait les snapshots VSS (espace et temps gaspillés) ; avec elles, la taille correspond aux données réelles.

**Recommandation :** utilisez le **mode fichier** (par défaut) avec auto-exclusions pour les sauvegardes au niveau fichier ; utilisez le **mode disque** dans une tâche séparée pour la restauration bare-metal (inclut tout).

### Sécurité & qualité
- Validation des entrées et nettoyage des identifiants (secrets masqués dans les journaux)
- Prévention des traversées de chemin (path traversal)
- Logique de réessai avec backoff exponentiel
- Contrôles CI à chaque build : tests, `golangci-lint`, `gosec`, `go mod tidy`

### 🔒 Chiffrement côté client
Les sauvegardes peuvent être chiffrées **sur le poste** avant de le quitter, avec
le même schéma que le client officiel `proxmox-backup-client` (AES-256-GCM,
empreintes de blocs à clé). Le serveur PBS ne stocke que des
données opaques et ne voit jamais la clé — idéal sur un PBS mutualisé ou infogéré.

- **Interface** : *Serveurs → Modifier → Clé de chiffrement* — créez un fichier
  de clé ou choisissez-en un existant (créé par
  `proxmox-backup-client key create --kdf none` ou issu d'un stockage PVE) ; son
  empreinte s'affiche. Gardez-en ensuite une copie hors du poste :
  **Imprimer (copie papier)** enregistre une page imprimable avec la clé et son QR
  code (le format de `proxmox-backup-client key paperkey`), éventuellement
  protégée par une phrase secrète. **Importer une clé depuis un texte ou un QR
  code** refait un fichier de clé à partir d'un QR code scanné ou d'une copie papier.
  L'interface n'utilise que des fichiers de clé non protégés : importer une clé
  protégée par une phrase secrète la déverrouille et enregistre le nouveau fichier
  **sans** phrase secrète — conservez ce fichier aussi soigneusement que la clé.
- **Ligne de commande** : `-keyfile chemin/vers/cle.json` (ou `"keyfile"` dans le
  fichier JSON) ; une clé protégée prend `-keyfile-passphrase`, sinon la phrase
  secrète est demandée.
- **Restauration bare-metal** : l'ISO Clonezilla modifiée restaure aussi les
  sauvegardes de disque chiffrées (clé sur clé USB, avec sa phrase secrète le cas
  échéant) — voir [PATCH-CLONEZILLA.md](PATCH-CLONEZILLA.md).
- **Compatible avec `proxmox-backup-client`, vérifié par la CI à chaque build**
  sur un vrai PBS : un dossier chiffré par Nimbus Backup avec une clé créée par
  `proxmox-backup-client` est restauré par le client officiel avec la même clé
  (et refusé sans elle), et une sauvegarde de disque chiffrée par le client
  officiel est relue par Nimbus Backup. Proxmox VE utilise les mêmes fichiers de clé.

> ⚠️ **Sans la clé, les sauvegardes chiffrées sont irrécupérables.** La première
> sauvegarde chiffrée renvoie toutes les données (pas de déduplication avec les
> sauvegardes non chiffrées). Les identifiants de sauvegarde, noms d'archives et
> tailles restent visibles du serveur ; le contenu et les noms des fichiers ainsi
> que le catalogue sont chiffrés.

## 🤖 Déploiement sans surveillance (Ansible & IaC)

Nimbus Backup s'installe et se configure entièrement par fichiers — aucune étape
interactive. Deux voies, toutes deux couvertes par des exemples prêts à l'emploi
dans [`examples/automation/`](examples/automation/) :

- **Ligne de commande** — un fichier JSON par hôte porte toute la sauvegarde
  (`proxmoxbackup-directory.exe --config fichier.json`) ; planification via le
  Planificateur de tâches Windows. Codes de retour fiables (`0` OK, `1` fatal,
  `2` verrou, `3` partiel) pour détecter les échecs. Idéal pour l'IaC pure.
- **Service/GUI (MSI)** — un **unique `config.json`** porte la connexion PBS, les
  réglages de sauvegarde **et** la planification (`scheduled_jobs`). Poussez le
  fichier, redémarrez le service `NimbusBackup` : les jobs sont réconciliés de
  façon idempotente et le `nextRun` est calculé pour vous. Aucun calcul de
  timestamp dans votre template Jinja2.

Le service lit sa configuration dans `C:\ProgramData\ProxmoxBackupClient\`
(depuis la 0.4.0 ; voir [Mise à jour](#️-mise-à-jour-depuis--030)).

📖 Tutoriel complet : [Déployer une sauvegarde Windows vers PBS avec Ansible](https://nimbus.rdem-systems.com/blog/deploiement-automatise-sauvegarde-windows-ansible/?utm_source=github).

## 🚀 Démarrage rapide

1. Téléchargez `NimbusBackup.msi` (ou le `NimbusBackup.exe` autonome) depuis les releases
2. Installez-le / lancez-le avec les droits administrateur (requis pour VSS)
3. Configurez votre connexion PBS et testez-la
4. Sélectionnez les dossiers à sauvegarder
5. Lancez la sauvegarde — ou planifiez-la

### 🔑 Utilisateur PBS et droits

Le client n'a besoin que du rôle **`DatastoreBackup`** sur le datastore cible — pas d'un compte administrateur :

1. Dans l'interface PBS, créez un utilisateur (ex. `nimbus@pbs`) et un jeton d'API pour lui (ex. `nimbus@pbs!laptop01`).
2. Dans **Datastore → Permissions** (ou **Configuration → Contrôle d'accès → Permissions**), attribuez `DatastoreBackup` sur `/datastore/<nom>` — ou sur `/datastore/<nom>/<namespace>` si vous sauvegardez dans un namespace.
3. **Jeton à privilèges séparés** (« Privilege Separation » cochée, le défaut) : attribuez le rôle au **jeton** lui-même (`nimbus@pbs!laptop01`), pas seulement à l'utilisateur — les droits effectifs sont l'intersection des deux. C'est la cause la plus fréquente de « permission denied ».

`DatastoreBackup` permet de créer des sauvegardes, de lister et de restaurer ses propres groupes de sauvegarde. Supprimer ou purger des snapshots demande `DatastorePowerUser`.

## ⬆️ Mise à jour depuis ≤ 0.3.0

Installer la 0.4.0 ou une version ultérieure par-dessus une installation existante la met à jour sur place (même identité MSI, même service `NimbusBackup`). Le dossier de données passe de `C:\ProgramData\NimbusBackup` au dossier partagé `C:\ProgramData\ProxmoxBackupClient` : au premier démarrage, la configuration, les tâches planifiées, l'historique et le jeton API y sont copiés une seule fois (aucun fichier existant n'est écrasé). L'ancien dossier est conservé et marqué par `COPIED-TO-ProxmoxBackupClient.txt`, ce qui permet encore un retour arrière. Les snapshots pris par les anciennes versions restent restaurables à leur emplacement d'origine.

## 📋 Prérequis

- Windows 10/11 ou Windows Server (64 bits)
- Droits administrateur (pour les snapshots VSS)
- Accès réseau à un serveur Proxmox Backup Server

## 🔨 Compilation depuis les sources

### Prérequis
- Go 1.25 ou ultérieur
- Node.js 20 ou ultérieur
- Wails CLI : `go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0` (la version utilisée par la CI)

### Build
```bash
cd gui
npm install --prefix frontend
wails build      # ou : wails dev  (rechargement à chaud)
```

Ou tout compiler (CLI + GUI + service) avec le Makefile : `make install-deps && make` (voir `make help`). Chaîne d'outils Windows et compilation croisée via Docker : [BUILD.md](BUILD.md).

La marque est déterminée par le nom de l'exécutable : `NimbusBackup.exe` s'affiche comme Nimbus Backup, tout autre nom comme « Proxmox Backup Client » neutre ([`gui/brand.go`](gui/brand.go)).

## 🔗 Relation avec le projet d'origine

Nimbus Backup est né comme un fork de [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go) (Proxmox Backup Client en Go, par Tiziano Bacocco, GPLv3), auquel nous avons ajouté l'interface graphique Windows, le service, la planification, le multi-PBS et la restauration. En septembre 2026, le projet d'origine a intégré cette interface et l'a rendue neutre (« Proxmox Backup Client GUI »). Depuis la 0.4.0, Nimbus Backup est compilé à partir du même code : **les deux projets sont désormais quasiment identiques fonctionnellement.**

Ce dépôt ajoute par-dessus une courte série de patchs documentée ([`patches/`](patches/README.md)) :

- **Des correctifs pas encore intégrés en amont** (build du service, restauration avec les serveurs identifiant/mot de passe, codes de retour, masquage des secrets dans les journaux, fiabilité de la sauvegarde machine…) — proposés au projet d'origine au fil de l'eau.
- **L'identité Nimbus Backup** — `NimbusBackup.exe`/`.msi`, le service `NimbusBackup` et le code de mise à niveau MSI des installations existantes, pour qu'elles continuent de se mettre à jour sur place.
- **Le chemin de mise à jour** depuis Nimbus Backup ≤ 0.3.0 (migration du dossier de données, métadonnées des anciens snapshots).
- **La chaîne de release** — attestation de provenance, empreintes, rapports VirusTotal.

Nous refusionnons régulièrement le projet d'origine ; ses propres releases sont publiées sur [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go/releases).

## ⚠️ Avertissement

Ce logiciel est fourni « tel quel ». Bien que nous visions la fiabilité, nous déclinons toute responsabilité en cas de perte ou de dommage de données. Testez toujours vos sauvegardes et vérifiez la restauration avant de vous y fier en production.

Ce projet n'est **pas affilié** à **Proxmox Server Solutions GmbH**. « Proxmox » et les noms associés sont la propriété de leurs détenteurs respectifs et ne sont utilisés ici que pour indiquer la compatibilité.

## 📄 Licence

GPLv3 — voir le fichier [LICENSE](LICENSE).

## À propos de RDEM Systems

NimbusBackupClient est développé et maintenu par [RDEM Systems](https://www.rdem-systems.com/?utm_source=github), un fournisseur d'infrastructure français spécialisé dans l'infogérance Proxmox VE/PBS et l'infrastructure NTP/NTS. Nous exploitons [16 serveurs NTS publics](https://ntp.rdem-systems.com/nts.php?utm_source=github) ([statut en direct](https://ntp.rdem-systems.com/status.php?utm_source=github) ; 11 d'entre eux sont référencés dans la [référence communautaire](https://github.com/jauderho/nts-servers)), et proposons un [hébergement PBS entièrement infogéré](https://nimbus.rdem-systems.com/?utm_source=github) pour ceux qui ne veulent pas auto-héberger. Nous maintenons également [unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client), des paquets `proxmox-backup-client` signés pour les distributions Linux non supportées officiellement par Proxmox.

---

**© 2024-2026 RDEM Systems et les contributeurs de Proxmox Backup Client GO.**
