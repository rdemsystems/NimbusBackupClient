# Mise à jour des packages Go 1.25

## ✅ Versions Go mises à jour
Tous les modules sont maintenant à `go 1.25`

## 📦 Packages à mettre à jour

### gui/go.mod - CRITIQUE (bloque CI)
**Packages principaux:**
- ✅ `github.com/wailsapp/wails/v2` v2.8.0 → v2.11.0 (mise à jour CI)
- ✅ `golang.org/x/crypto` v0.11.0 → latest (mise à jour CI)
- ✅ `golang.org/x/net` v0.12.0 → latest (mise à jour CI)
- ✅ `golang.org/x/sys` v0.13.0 → latest (mise à jour CI)
- ✅ `golang.org/x/text` v0.11.0 → latest (mise à jour CI)

**Autres dépendances:**
- `github.com/cornelk/hashmap` v1.0.8 → vérifier latest
- `github.com/go-ole/go-ole` v1.3.0 → OK (récent)
- `github.com/google/uuid` v1.3.0 → v1.6.0 disponible
- `github.com/labstack/echo/v4` v4.11.1 → v4.12+ disponible
- `golang.org/x/exp` v0.0.0-20230522175609 → très vieux

### pbscommon/go.mod
**Packages:**
- `github.com/klauspost/compress` v1.17.9 → OK (très récent)
- `golang.org/x/net` v0.23.0 → OK (2024)
- `golang.org/x/text` v0.14.0 → OK (2024)
- `github.com/dchest/siphash` v1.2.3 → OK

### snapshot/go.mod - À CORRIGER
**Packages:**
- ❌ `github.com/go-ole/go-ole` v1.2.6 → v1.3.0 disponible
- `github.com/st-matskevich/go-vss` v0.3.3 → vérifier latest
- ❌ `golang.org/x/sys` v0.1.0 → TRÈS VIEUX (2022)

### directorybackup/go.mod
**Packages récents:**
- `github.com/cornelk/hashmap` v1.0.8 → vérifier
- `github.com/go-ole/go-ole` v1.3.0 → OK
- `golang.org/x/sys` v0.30.0 → OK (très récent)
- `golang.org/x/text` v0.15.0 → OK (2024)
- `golang.org/x/exp` v0.0.0-20240531132922 → OK (2024)

**Packages getlantern:**
- Tous datent de 2019 mais stables, pas critique

### machinebackup/go.mod - À CORRIGER
**Packages:**
- ❌ `github.com/go-ole/go-ole` v1.2.6 → v1.3.0 disponible
- `github.com/google/uuid` v1.6.0 → OK
- `github.com/shirou/gopsutil` v3.21.11+incompatible → vérifier v3/v4 récents
- ❌ `golang.org/x/sys` v0.0.0-20201018230417 → EXTRÊMEMENT VIEUX (2020)

### service/go.mod
**Packages:**
- `github.com/kardianos/service` v1.2.2 → v1.2.4 disponible (requis Go 1.23+)
- `golang.org/x/sys` v0.30.0 → OK (très récent)

### nbd/go.mod
**Packages:**
- `github.com/pojntfx/go-nbd` v0.3.2 → vérifier latest
- `github.com/rivo/tview` v0.42.0 → vérifier latest
- `golang.org/x/sys` v0.29.0 → OK (récent)
- `golang.org/x/term` v0.28.0 → OK (récent)
- `golang.org/x/text` v0.21.0 → OK (récent)

### clientcommon/go.mod
**Packages:**
- `github.com/rodolfoag/gow32` v0.0.0-20230512144032 → pas de version tagged, vérifier commits

### pkg/retry/go.mod et pkg/security/go.mod
Pas de dépendances externes

## 🔧 Actions requises

### Automatique (CI GitHub)
- ✅ `golangci-lint` mis à jour vers latest
- ✅ Update automatique des packages critiques de gui/

### Manuel (après CI ou avec Go installé)
```bash
# Mettre à jour snapshot/go.mod
cd snapshot
GOWORK=off go get -u github.com/go-ole/go-ole golang.org/x/sys
GOWORK=off go mod tidy

# Mettre à jour machinebackup/go.mod
cd ../machinebackup
GOWORK=off go get -u github.com/go-ole/go-ole github.com/shirou/gopsutil golang.org/x/sys
GOWORK=off go mod tidy

# Mettre à jour service/go.mod
cd ../service
GOWORK=off go get -u github.com/kardianos/service
GOWORK=off go mod tidy

# Mettre à jour gui/go.mod (packages non critiques)
cd ../gui
GOWORK=off go get -u github.com/google/uuid github.com/labstack/echo/v4 golang.org/x/exp
GOWORK=off go mod tidy
```

## 📝 Notes

- Les packages `golang.org/x/*` sont critiques pour Go 1.25
- `wails/v2` v2.11.0 apporte compatibilité Go 1.25+
- Packages `getlantern/*` : vieux mais stables, pas prioritaires
- `golangci-lint` : maintenant installé en "latest" au lieu de version fixe

## ✅ Statut
- [x] Versions Go 1.25 dans tous les go.mod
- [x] CI GitHub : golangci-lint latest
- [x] CI GitHub : update automatique packages critiques gui/
- [ ] Updates manuels des autres modules (snapshot, machinebackup, service)
