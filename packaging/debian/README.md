# Debian packaging (pbsgo)

Builds `pbsgo_<version>_<arch>.deb` with `dpkg-deb`. The version is taken
from `gui/wails.json` (`productVersion`).

## Build host requirements

- Go toolchain
- `dpkg-dev` (dpkg-deb, dpkg-shlibdeps)
- `libwebkit2gtk-4.1-dev` (cgo headers to link the GUI)
- built GUI frontend in `gui/frontend/dist`
  (`cd gui/frontend && npm install && npm run build`)

## Build

```sh
packaging/debian/build-deb.sh [output-dir]     # default output dir: dist/
# or
make deb
```

## Package contents

| Path | Description |
| ---- | ----------- |
| `/usr/bin/pbsgo` | directory/stream backup CLI |
| `/usr/bin/pbsgo-machine` | whole-machine (raw disk) backup CLI |
| `/usr/bin/pbsgo-nbd` | fidx-to-NBD restore tool (run as root) |
| `/usr/bin/pbsgo-gui` | GUI launcher (current user) |
| `/usr/bin/pbsgo-gui-root` | GUI launcher via sudo/pkexec (needed for machine backup) |
| `/usr/lib/pbsgo/pbsgo-gui` | GUI application binary (frontend embedded) |
| `/usr/share/applications/pbsgo-gui.desktop` | desktop entry |
| `/usr/share/icons/hicolor/256x256/apps/pbsgo.png` | icon |

Runtime dependencies of the GUI are computed with `dpkg-shlibdeps`; when
that is not possible the script falls back to
`libwebkit2gtk-4.1-0, libgtk-3-0t64 | libgtk-3-0`.
