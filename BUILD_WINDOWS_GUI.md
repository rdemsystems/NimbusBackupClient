# Building Windows GUI - Solutions

## ❌ Problème
Cross-compilation Windows depuis Linux avec MinGW ne gère pas correctement les dépendances OpenGL de Fyne, résultant en l'erreur :
```
APIUnavailable: WGL: The driver does not appear to support OpenGL
```

## ✅ Solutions (par ordre de recommandation)

### Solution 1️⃣ : GitHub Actions (GRATUIT, automatique)

**Avantages** :
- ✅ Runners Windows natifs gratuits
- ✅ Build automatique à chaque push/tag
- ✅ Binaires publiés dans les Releases GitHub
- ✅ Aucune infrastructure Windows nécessaire

**Configuration** :
Le workflow `.github/workflows/build-gui-windows.yml` est déjà configuré.

**Utilisation** :
```bash
# 1. Push vers GitHub
git push origin master

# 2. Créer un tag pour une release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 3. Le binaire sera disponible dans :
#    - GitHub Actions Artifacts (toujours)
#    - GitHub Releases (si tag)
```

---

### Solution 2️⃣ : fyne-cross (GitLab CI/CD avec Docker)

**Avantages** :
- ✅ Cross-compilation correcte avec toutes les dépendances OpenGL
- ✅ Reste sur GitLab CI/CD
- ✅ Utilise Docker (pas besoin de runner Windows)

**Configuration** :
Déjà intégré dans `.gitlab-ci.yml` avec `lucor/fyne-cross:latest`.

**Test local** :
```bash
cd gui
docker run --rm -v $(pwd):/app -w /app lucor/fyne-cross:latest windows -arch=amd64
# Binaire dans : fyne-cross/bin/windows-amd64/
```

---

### Solution 3️⃣ : Build local Windows (tests rapides)

**Avantages** :
- ✅ Build natif = 100% compatible
- ✅ Parfait pour tests rapides
- ✅ Pas de dépendance CI/CD

**Prérequis Windows** :
1. Go 1.22+ : https://go.dev/dl/
2. GCC (MinGW-w64) : https://www.msys2.org/

**Build** :
```cmd
# Sur Windows
.\build_gui.bat

# Ou manuellement
cd gui
go build -ldflags="-s -w -H windowsgui" -o ..\proxmox-backup-gui.exe .
```

---

## 🎯 Recommandation

**Pour vous** : Utilisez **GitHub Actions** (Solution 1)

**Pourquoi ?** :
- Pas de coût (runners gratuits)
- Build automatique propre
- Compatible avec votre GitLab (mirroring possible)
- Releases automatiques

**Setup rapide** :
```bash
# 1. Push ce repo sur GitHub (si pas déjà fait)
git remote add github https://github.com/YOUR_USERNAME/proxmox-backup-gui.git
git push github master

# 2. C'est tout ! Le workflow se lance automatiquement
```

---

## 📝 Notes

- **Ne pas utiliser** : Cross-compilation MinGW basique (build actuel GitLab ligne 120)
- **Certificat de signature** : Optionnel, pas nécessaire pour fonctionner
- **Test** : Le binaire généré par GitHub Actions ou fyne-cross fonctionnera correctement
