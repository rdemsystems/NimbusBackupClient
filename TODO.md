# Nimbus Backup - TODO

## 🔴 P0 - CRITIQUE (En cours)

### Fix Bug Config Service (URGENT)
**Problème:** Service charge config au démarrage, ne recharge jamais → backup échoue si config change

- [x] ~~HTTP API existe~~ ✓ (`gui/api/server.go`)
- [x] ~~Service Windows fonctionne~~ ✓ (`gui/service.go`)
- [ ] **ReloadConfig avant backup** (PATCH EN COURS)
  - [x] Méthode `ReloadConfig()` ajoutée dans App
  - [x] `handleBackup()` appelle `ReloadConfig()` avant backup
  - [ ] Test: Sauvegarder config → Backup → Utilise nouvelle config
  - [ ] Rebuild + MSI + test sur Windows

**Temps estimé:** 1h (rebuild + test)

### Multi-PBS Architecture (FEATURE MORTELLE 🚀)
**Use case:** Multi-datastore (C:\ → bigdata, C:\Users → ssd) + GUI distante

- [ ] **Config Multi-PBS**
  ```json
  {
    "pbs_servers": {
      "pbs1": { "name": "Big Data", "baseurl": "...", "datastore": "bigdata" },
      "pbs2": { "name": "SSD Fast", "baseurl": "...", "datastore": "ssd" }
    },
    "backup_id_template": "{{hostname}}", // Ou custom
    "default_pbs": "pbs1"
  }
  ```
  - [ ] Structure config avec map de PBS
  - [ ] Validation: au moins 1 PBS configuré
  - [ ] Migration: config actuelle → pbs_servers["default"]

- [ ] **BackupRequest avec PBS ID**
  ```go
  type BackupRequest struct {
      PBSID       string   `json:"pbs_id"`        // "pbs1", "pbs2"
      BackupType  string   `json:"backup_type"`
      BackupDirs  []string `json:"backup_dirs"`
      BackupID    string   `json:"backup_id"`
      UseVSS      bool     `json:"use_vss"`
      ExcludeList []string `json:"exclude_list"`
  }
  ```
  - [ ] API: Accepter `pbs_id` dans requête
  - [ ] Service: Charger config du PBS correspondant
  - [ ] Fallback: Si `pbs_id` vide → utiliser `default_pbs`

- [ ] **Jobs gérés par Service** (pas GUI)
  - [ ] Jobs stockés dans service (`C:\ProgramData\Nimbus\jobs.json`)
  - [ ] Chaque job → 1 PBS spécifique
  - [ ] **Rechargement jobs** : Service relit `jobs.json` avant chaque exécution
  - [x] Config auto-reload avant backup ✓ (v0.1.81)
  - [ ] **API Reload** : `POST /api/reload/config` et `POST /api/reload/jobs`
  - [ ] GUI peut éditer, déclencher reload, service répond OK
  - [ ] API: `POST /jobs` (créer), `PUT /jobs/{id}` (modifier), `DELETE /jobs/{id}`
  - [ ] GUI distante peut provisionner le service via API

- [ ] **GUI Multi-PBS**
  - [ ] Dropdown "Serveur PBS" dans formulaire backup
  - [ ] Gestion PBS: Ajouter/Modifier/Supprimer serveurs
  - [ ] Test connexion par PBS
  - [ ] Indicateur: 🟢 Online / 🔴 Offline par PBS

**Temps estimé:** 1-2 semaines (grosse feature!)

---

## 🟠 P1 - IMPORTANT (Architecture Entreprise)

---

## 🟠 P1 - IMPORTANT (Prochaines semaines)

### Service Windows - Robustesse
- [ ] **VSS Cleanup au démarrage**
  ```go
  func cleanupOrphanedVSS() {
      exec.Command("vssadmin", "delete", "shadows", "/for=C:", "/all", "/quiet").Run()
  }
  ```
  - [ ] Appel dans `service.Start()`
  - [ ] Log les shadows supprimées

- [ ] **Working Directory fix**
  ```go
  exePath, _ := os.Executable()
  os.Chdir(filepath.Dir(exePath))
  ```
  - [ ] Force au démarrage du service
  - [ ] Test: config.json trouvé dans ProgramData

- [ ] **Logs accessibles**
  - [ ] Service log dans `C:\ProgramData\Nimbus\logs\service.log`
  - [ ] GUI: bouton "Voir logs du service" (lecture seule)
  - [ ] Rotation: max 10 MB par fichier

### MSI - Finitions

- [ ] **Installation Silencieuse (Silent Install)**
  - [ ] **Approche: Config JSON pré-configuré** (propre pour AD/GPO)
    ```powershell
    # Déploiement avec config centralisée
    msiexec /i NimbusBackup.msi /qn CONFIGFILE="\\ad-server\deploy\nimbus\config.json"
    ```
  - [ ] Property WiX: `CONFIGFILE` (chemin vers config.json)
  - [ ] CustomAction WiX:
    - Si `CONFIGFILE` fourni → copier vers `C:\ProgramData\Nimbus\config.json`
    - Valider JSON avant copie (éviter corruption)
    - Log erreur si fichier inaccessible
  - [ ] Template config.json à fournir:
    ```json
    {
      "pbs_url": "https://pbs.example.com:8007",
      "auth_id": "backup-user@pbs",
      "secret": "your-api-token-secret",
      "datastore": "backup",
      "namespace": "clients",
      "backup_id": "",  // Vide = utilise hostname
      "backup_dirs": ["C:\\Users", "C:\\Important"],
      "exclusions": ["*.tmp", "*.log"],
      "schedule": {
        "enabled": true,
        "time": "02:00",
        "days": ["monday", "wednesday", "friday"]
      },
      "vss_enabled": true
    }
    ```
  - [ ] Test: install silencieux → service démarre avec config OK
  - [ ] Doc: guide déploiement GPO/Intune avec config.json

- [ ] **Code Signing**
  - [ ] Signer le binaire `.exe`
  - [ ] Signer le `.msi`
  - [ ] Certificat: à obtenir (DigiCert/Sectigo ~300€/an)

- [ ] **Désinstallation propre**
  - [ ] Script CustomAction: stop service avant uninstall
  - [ ] Nettoyer `C:\ProgramData\Nimbus` (option: garder config)

### Multi-jobs - Stabilisation
- [ ] **Queue management**
  - [ ] Pas de 2 jobs VSS simultanés
  - [ ] File d'attente FIFO
  - [ ] UI: afficher "En attente..." si queue pleine

- [ ] **Test de charge**
  - [ ] Lancer 5 jobs en même temps
  - [ ] Vérifier pas de corruption d'index PBS
  - [ ] RAM usage < 500 MB

---

## 🟢 P2 - NICE TO HAVE (Backlog)

### Windows - Compatibilité avancée
- [ ] **LongPath Support**
  - [ ] Ajouter manifeste: `<longPathAware>true</longPathAware>`
  - [ ] Test: backup d'un chemin >260 caractères

- [ ] **Gestion des locks**
  - [ ] Détecter fichier ouvert sans VSS
  - [ ] Erreur propre: "Fichier X verrouillé, activer VSS?"

### API Remote - Provisioning Distant (Phase 2)
**Use case:** MSP gère 100+ clients Nimbus depuis interface centrale

- [ ] **API Remote activable**
  ```json
  {
    "api": {
      "remote_enabled": false,  // Désactivé par défaut (sécurité)
      "bind_address": "0.0.0.0:18765",  // Si activé
      "auth_token": "generated-at-install",
      "tls_cert": "/path/to/cert.pem",  // Optionnel
      "allowed_ips": ["192.168.1.0/24"]  // Whitelist
    }
  }
  ```
  - [ ] Flag service: `--remote-api` pour activer
  - [ ] Auth: Bearer token (généré install, 32 chars)
  - [ ] TLS: Certificat auto-signé ou fourni
  - [ ] Rate limiting: max 10 req/s par IP
  - [ ] Whitelist IPs configurables

- [ ] **Endpoints Provisioning**
  - `GET /api/v1/info` - Info système (hostname, version, mode)
  - `GET /api/v1/pbs` - Liste serveurs PBS configurés
  - `POST /api/v1/pbs` - Ajouter serveur PBS
  - `PUT /api/v1/pbs/{id}` - Modifier serveur PBS
  - `DELETE /api/v1/pbs/{id}` - Supprimer serveur PBS
  - `POST /api/v1/pbs/{id}/test` - Test connexion
  - `GET /api/v1/jobs` - Liste jobs
  - `POST /api/v1/jobs` - Créer job
  - `PUT /api/v1/jobs/{id}` - Modifier job
  - `DELETE /api/v1/jobs/{id}` - Supprimer job
  - `POST /api/v1/backup` - Lancer backup manuel

- [ ] **GUI Centrale MSP** (Futur produit séparé)
  - Dashboard: grille avec tous les clients
  - Actions groupées: "Backup tout le parc"
  - Alertes: machine pas vue depuis 24h
  - Statistiques globales

**Temps estimé:** 2-3 semaines

### Multi-Serveurs PBS
**→ DÉPLACÉ EN P0** (voir "Multi-PBS Architecture" ci-dessus)

### Chiffrement (Phase 3)
- [ ] **Key Management**
  - [ ] Génération clé asymétrique
  - [ ] Stockage: Windows Credential Manager (DPAPI)
  - [ ] Export: bouton "Sauvegarder clé de récupération"

- [ ] **GUI**
  - [ ] Checkbox "Activer chiffrement"
  - [ ] Warning: "Sans la clé, restauration impossible!"

### Restauration locale (Phase 4)
- [ ] **Navigateur de snapshots**
  - [ ] Liste des snapshots depuis PBS
  - [ ] Parcourir le catalog (lazy loading)
  - [ ] Sélection fichiers/dossiers

- [ ] **Download & Restore**
  - [ ] Bouton "Restaurer vers..."
  - [ ] Gestion conflits (Écraser/Renommer)
  - [ ] Extraction via service (droits admin)

### Mode Entreprise (Phase 5)
**→ DÉPLACÉ EN P1** (voir "API Remote - Provisioning Distant")

---

## 🗑️ DROP (Ignoré pour l'instant)

- ❌ UUID machine (hostname suffit)
- ❌ Heartbeat vers API RDEM (overkill)
- ❌ go-msi (WiX fonctionne)
- ❌ Mount FUSE/WinFSP (restauration web suffit)

---

## 📅 Roadmap suggérée

**v0.2.x - Communication GUI-Service** (2-3 semaines)
- Named Pipes fonctionnel
- VSS via service uniquement
- GUI = télécommande

**v0.3.x - Robustesse Service** (2-3 semaines)
- VSS cleanup
- Logs propres
- MSI signed

**v0.4.x - Chiffrement** (3-4 semaines)
- Key management
- Encryption at rest

**v0.5.x - Restauration locale** (4-6 semaines)
- Browse snapshots
- Selective restore

**v1.0.0 - Production Ready** (3 mois total)
- Tout ce qui précède
- Doc complète
- Support officiel

---

**Dernière mise à jour:** 2026-03-19
**Mainteneur:** RDEM Systems
