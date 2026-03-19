# Nimbus Backup - Release Notes

## ✅ Works (Fonctionnalités opérationnelles)

### Backup & Restore
- ✅ Backup one-shot (exécution immédiate)
- ✅ Backup planifié (quotidien avec heure configurable)
- ✅ Édition des jobs planifiés
- ✅ VSS (Volume Shadow Copy) support
- ✅ Exclusion de fichiers/dossiers
- ✅ Restauration de snapshots
- ✅ Liste des snapshots disponibles

### Interface & UX
- ✅ Interface graphique Wails (Go + React)
- ✅ Historique des backups (6 derniers affichés)
- ✅ Relance des backups échoués
- ✅ Mode "Run at startup" pour jobs planifiés
- ✅ Barre de progression avec statistiques
- ✅ Minimize to tray
- ✅ Exit from tray (force quit after 2s)

### Configuration
- ✅ Configuration Proxmox Backup Server
- ✅ Test de connexion
- ✅ Fingerprint de certificat
- ✅ Namespace support

## 🚧 In Progress (En cours de développement)

- 🔄 **Systray icon visible** (v0.1.42 - fix avec go:embed du vrai .ico, en test)
- 🔄 **Auto-start avec privilèges admin** (v0.1.42 - Task Scheduler HIGHEST, en test)
- 🔄 **Fix double lancement au boot** (v0.1.42 - cleanup + délai, en test)

## 📋 TODO (À faire)

### Priorité haute
- 🌍 **Traduction EN** (interface actuellement en français uniquement)
- 📦 **Installateur MSI** (plus professionnel que .exe standalone)
- 🔧 **Amélioration gestion erreurs VSS** (messages plus clairs)
- 🔔 **Notifications Windows** (succès/échec des backups planifiés)

### Priorité moyenne
- 📊 **Statistiques détaillées** (taille sauvegardée, durée, vitesse moyenne)
- 🗓️ **Planification hebdomadaire/mensuelle** (actuellement quotidien uniquement)
- 🔐 **Stockage sécurisé des credentials** (actuellement en clair dans config.json)
- 📧 **Alertes email** (en cas d'échec de backup)
- 🌐 **Support multi-serveurs PBS** (basculement automatique)

### Priorité basse
- 🎨 **Thèmes interface** (dark mode)
- 📝 **Logs détaillés exportables**
- 🔄 **Auto-update intégré** (vérification de nouvelle version)
- 💾 **Import/export configuration**
- 🖥️ **Support Linux/macOS** (actuellement Windows uniquement)

## 📌 Known Issues (Problèmes connus)

- ⚠️ Icône systray peut être transparente (fix v0.1.42 en test)
- ⚠️ Double lancement au boot (fix v0.1.42 en test)
- ⚠️ Pas de validation du format des exclusions
- ⚠️ Interface uniquement en français

## 📜 Changelog récent

### v0.1.42 (2026-03-19)
- **FIX**: Icône systray embarquée depuis vrai .ico (go:embed)
- **FIX**: Auto-start via Task Scheduler avec privilèges HIGHEST
- **FIX**: Nettoyage ancienne entrée registre (migration)
- **FIX**: Délai 5s dans HandleStartupRun pour éviter double exécution

### v0.1.41 (2026-03-19)
- **FEAT**: Édition des jobs planifiés (bouton Éditer/Annuler)
- **FEAT**: Fonction UpdateScheduledJob backend
- **FEAT**: Limitation historique à 6 derniers backups
- **FIX**: Quit systray avec force exit (2s timeout)

---

**Version actuelle:** 0.1.42
**Dernière mise à jour:** 2026-03-19
