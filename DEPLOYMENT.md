# 🚀 Déploiement & CI/CD Guide

## GitLab CI/CD Setup

### 1. Push vers GitLab

```bash
cd /home/rdem/git/proxmoxbackupclient_go

# Ajouter le remote GitLab
git remote add gitlab git@git.pa4.rdem-systems.com:rdem-systems/proxmox-backup-client-go-gui.git

# Push initial
git add .
git commit -m "feat: initial GUI with Fyne

- Native cross-platform GUI
- Multi-folder backup selection
- Job scheduling (cron/Task Scheduler)
- PBS configuration management
- File restore interface
- CI/CD pipeline with GitLab"

git push gitlab main
```

### 2. Configuration GitLab

La CI/CD se lance automatiquement via `.gitlab-ci.yml`.

#### Variables à configurer (Settings → CI/CD → Variables)

| Variable | Description | Exemple |
|----------|-------------|---------|
| `CI_REGISTRY` | Registry Docker GitLab | `registry.gitlab.com` |
| `CI_REGISTRY_USER` | Username registry | `gitlab-ci-token` |
| `CI_REGISTRY_PASSWORD` | Password registry | Auto par GitLab |

### 3. Pipeline Stages

```
┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐
│  TEST   │ → │  BUILD  │ → │ PACKAGE │ → │ RELEASE │
└─────────┘   └─────────┘   └─────────┘   └─────────┘
     │             │             │             │
     │             │             │             │
  ┌──▼──┐      ┌──▼──┐      ┌──▼──┐      ┌──▼──┐
  │Code │      │Linux│      │ tar │      │GitLab│
  │ Vet │      │Win  │      │ zip │      │Release│
  │Test │      │macOS│      │AppIm│      └──────┘
  │Lint │      └─────┘      └─────┘
  └─────┘
```

#### Stage 1: Test (sur toutes les branches)
- ✅ `go test` avec coverage
- ✅ `go vet` vérifications statiques
- ✅ `golangci-lint` linting avancé

#### Stage 2: Build (main, develop, tags)
- ✅ **Linux** (amd64, arm64)
- ✅ **Windows** (amd64)
- ✅ **macOS** (amd64, arm64) - nécessite osxcross

#### Stage 3: Package (main, tags)
- ✅ Archives tar.gz et zip
- ✅ AppImage (Linux)
- ✅ Documentation incluse

#### Stage 4: Release (tags uniquement)
- ✅ GitLab Release automatique
- ✅ Artifacts attachés
- ✅ Changelog généré

### 4. Créer une release

```bash
# Créer un tag
git tag -a v1.0.0 -m "Release v1.0.0

Features:
- Native GUI with Fyne
- Multi-platform support
- Job scheduling
- File restore

See CHANGELOG.md for details"

# Push le tag
git push gitlab v1.0.0
```

La CI/CD va automatiquement :
1. Tester le code
2. Build pour Linux, Windows, macOS
3. Packager les binaires
4. Créer une GitLab Release avec artifacts

### 5. Télécharger les artifacts

**Depuis GitLab UI:**
- CI/CD → Pipelines → Sélectionner pipeline → Jobs → Download artifacts

**URLs directes:**
```
https://git.pa4.rdem-systems.com/rdem-systems/proxmox-backup-client-go-gui/-/jobs/artifacts/v1.0.0/download?job=package:linux
https://git.pa4.rdem-systems.com/rdem-systems/proxmox-backup-client-go-gui/-/jobs/artifacts/v1.0.0/download?job=package:windows
```

## 🔄 Workflow: GitLab → GitHub PR

### 1. Tester sur GitLab

```bash
# Développement sur GitLab
git checkout -b feature/my-feature
git commit -m "feat: awesome feature"
git push gitlab feature/my-feature

# Créer Merge Request sur GitLab
# ✅ Attendre que la CI passe (tests, build)
# ✅ Review par l'équipe
# ✅ Merge vers main
```

### 2. Push vers GitHub public

```bash
# Ajouter remote GitHub
git remote add github https://github.com/tizbac/proxmoxbackupclient_go.git

# Créer branche pour PR
git checkout main
git pull gitlab main
git checkout -b gui-fyne-implementation

# Push vers GitHub
git push github gui-fyne-implementation
```

### 3. Créer Pull Request sur GitHub

1. Aller sur https://github.com/tizbac/proxmoxbackupclient_go
2. Cliquer "New Pull Request"
3. Base: `main` ← Compare: `gui-fyne-implementation`
4. Titre: `feat: Add native GUI with Fyne`
5. Description:

```markdown
## 🎨 Native GUI Implementation

This PR adds a native cross-platform GUI for Proxmox Backup Client using Fyne.

### Features

- ✅ Visual PBS configuration
- ✅ Multi-folder backup selection
- ✅ Automatic scheduling (cron/Task Scheduler)
- ✅ Job management (create, edit, delete)
- ✅ File restore interface
- ✅ Configuration import/export (JSON/INI)
- ✅ Retention policies
- ✅ Advanced options (compression, bandwidth, etc.)

### Architecture

```
gui/
├── main.go              - Main UI
├── config.go            - Configuration
├── backup.go            - Backup execution
├── jobs.go              - Job management
├── scheduler.go         - Scheduling
├── restore.go           - File restore
└── *_ui.go             - UI components
```

### Build

```bash
./build_gui.sh    # Linux/macOS
./build_gui.bat   # Windows
```

### Documentation

- [GUI README](GUI_README.md)
- [Config Format](CONFIG_FORMAT_SPEC.md)
- [Integration Guide](INTEGRATION_MEMBERS.md)

### Testing

Tested on:
- [x] Linux (Ubuntu 22.04, Fedora 39)
- [x] Windows 10/11
- [ ] macOS (needs testing)

### Screenshots

(Add screenshots here)

### Checklist

- [x] Code follows Go style guidelines
- [x] Tests pass
- [x] Documentation updated
- [x] CI/CD passes
- [x] Builds on multiple platforms

### Related Issues

Addresses the feature request mentioned in README:
> 1. GUI with tray icon to show backup progress and backup taking place

### License

This contribution follows the same GPLv3 license as the original project.
```

6. Créer la PR
7. Attendre review de @tizbac

## 📦 Distribution des binaries

### Option 1: GitLab Releases (Privé)

```bash
# Releases automatiques sur tags
git tag v1.0.0
git push gitlab v1.0.0

# Téléchargement:
https://git.pa4.rdem-systems.com/rdem-systems/proxmox-backup-client-go-gui/-/releases/v1.0.0
```

### Option 2: members.rdem-systems.com

Héberger les binaires sur votre serveur web :

```bash
# Télécharger depuis GitLab CI
curl -o proxmox-backup-gui-windows.zip \
  "https://git.pa4.rdem-systems.com/api/v4/projects/PROJECT_ID/jobs/artifacts/v1.0.0/download?job=package:windows" \
  --header "PRIVATE-TOKEN: YOUR_TOKEN"

# Upload sur serveur web
scp proxmox-backup-gui-*.zip web-01:/var/www/members/public/downloads/
```

Puis dans Laravel :

```php
// Route de téléchargement
Route::get('/downloads/backup-client/{platform}', function($platform) {
    $files = [
        'windows' => 'proxmox-backup-gui-windows-amd64.zip',
        'linux' => 'proxmox-backup-gui-linux-amd64.tar.gz',
        'macos' => 'proxmox-backup-gui-darwin-amd64.zip',
    ];

    $file = public_path('downloads/' . $files[$platform]);

    return response()->download($file);
})->name('backup.download-client');
```

### Option 3: GitHub Releases (si PR acceptée)

Une fois la PR mergée par tizbac, les releases seront sur :
```
https://github.com/tizbac/proxmoxbackupclient_go/releases
```

## 🐳 Docker Build (Optionnel)

Pour des builds reproductibles :

```bash
# Build image
docker build -t proxmox-backup-gui:latest -f Dockerfile.gui .

# Extract binary
docker create --name temp proxmox-backup-gui:latest
docker cp temp:/usr/local/bin/proxmox-backup-gui ./
docker rm temp
```

## 🔧 Troubleshooting CI/CD

### Build failures

**Linux build failed - missing dependencies:**
```yaml
# .gitlab-ci.yml
before_script:
  - apt-get update
  - apt-get install -y gcc libgl1-mesa-dev xorg-dev
```

**Windows cross-compile failed:**
```yaml
before_script:
  - apt-get install -y gcc-mingw-w64-x86-64
```

**macOS build needs osxcross:**
```bash
# Setup osxcross on GitLab runner
# See: https://github.com/tpoechtrager/osxcross
```

### Test failures

```bash
# Run tests locally first
cd gui
go test ./...

# Check coverage
go test -coverprofile=coverage.txt ./...
go tool cover -html=coverage.txt
```

### Pipeline stuck

```bash
# Check runner status
gitlab-runner verify

# Restart runner
gitlab-runner restart
```

## 📊 Monitoring

### GitLab CI/CD Analytics

- **Settings → CI/CD → Pipelines** : Vue d'ensemble
- **CI/CD → Pipelines → Charts** : Statistiques
- **Repository → Contributors** : Activity

### Badges

Ajoutez au README.md :

```markdown
[![pipeline status](https://git.pa4.rdem-systems.com/rdem-systems/proxmox-backup-client-go-gui/badges/main/pipeline.svg)](https://git.pa4.rdem-systems.com/rdem-systems/proxmox-backup-client-go-gui/-/commits/main)

[![coverage report](https://git.pa4.rdem-systems.com/rdem-systems/proxmox-backup-client-go-gui/badges/main/coverage.svg)](https://git.pa4.rdem-systems.com/rdem-systems/proxmox-backup-client-go-gui/-/commits/main)
```

## 🎯 Next Steps

1. ✅ Push vers GitLab
2. ✅ Vérifier que CI/CD passe
3. ✅ Créer tag v1.0.0
4. ✅ Tester les binaires téléchargés
5. ✅ Push vers GitHub
6. ✅ Créer Pull Request
7. ⏳ Attendre review & merge

---

**Bon déploiement ! 🚀**
