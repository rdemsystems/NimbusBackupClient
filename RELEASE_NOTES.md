# Nimbus Backup — Statut & notes

> Les changements **par version** figurent dans la section « Changes since… » de chaque release (ci-dessus) et dans [CHANGELOG.md](CHANGELOG.md). Cette page décrit l'état **pérenne** du produit.

## 📦 Versions disponibles

### NimbusBackup.msi (installateur — recommandé en production)
- ✅ **Service Windows** : démarre automatiquement au boot système
- ✅ **Privilèges admin permanents** : le service tourne en LocalSystem (VSS garanti)
- ✅ **Backups planifiés** : exécutés automatiquement, même après reboot
- ✅ **Désinstallation propre** : nettoyage complet via le Panneau de configuration

### NimbusBackup.exe (standalone)
- ✅ **Backups manuels et planifiés** : OK tant que l'application est lancée
- ❌ **Pas de persistance au reboot** : pas de service → préférez le MSI en production
- 💡 **Usage** : backups ponctuels ou tests

## ✅ Fonctionnalités

### Sauvegarde & restauration
- Backup one-shot (immédiat) et planifié (heure configurable)
- **Auto-split des gros backups** (>100 Go) en jobs équilibrés (~100 Go) avec retry par job
- **VSS** (Volume Shadow Copy) pour des backups cohérents
- Exclusions de fichiers/dossiers + **auto-exclusion des dossiers système Windows** (System Volume Information, $RECYCLE.BIN, pagefile.sys…)
- Restauration et navigation dans les snapshots (**lecture rapide du catalogue**)
- Backups longue durée robustes (**keep-alive 30 s**, validés sur des backups de 11 h+)
- **Support multi-serveurs PBS**

### Interface & configuration
- Interface graphique Wails (Go + React), **en français et en anglais**
- Historique des backups et relance des jobs échoués
- Barre de progression avec statistiques, minimize to tray
- Configuration PBS avec test de connexion, **épinglage d'empreinte de certificat (TOFU)** et namespaces

## 📌 Problèmes connus
- ⚠️ La version **.exe standalone** ne persiste pas au reboot → utilisez le **MSI** en production.
- ⚠️ Le format des exclusions n'est pas validé à la saisie.
