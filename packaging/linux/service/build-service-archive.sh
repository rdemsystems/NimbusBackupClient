#!/bin/sh
# build-service-archive.sh - builds salvanza-service-<version>-linux-<arch>.tar.gz:
# the background service and the command line for a Linux computer that has
# no Salvanza package (Slackware, Alpine, Gentoo...), with install.sh and the
# init scripts for systemd, SysV init, Slackware and OpenRC. Both programs are
# static (CGO_ENABLED=0), so the archive runs with glibc and musl alike.
#
# Usage: packaging/linux/service/build-service-archive.sh [out-dir] [goarch]
# (default out-dir <repo>/dist, goarch amd64; PBS_VERSION overrides the
# version read from gui/wails.json).
set -eu

HERE=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$HERE/../../.." && pwd)
OUT=${1:-$ROOT/dist}
ARCH=${2:-amd64}
VERSION=${PBS_VERSION:-$(sed -n 's/.*"productVersion"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$ROOT/gui/wails.json" | head -n1)}
[ -n "$VERSION" ] || { echo "error: no version" >&2; exit 1; }

NAME=salvanza
TITLE="Salvanza for Proxmox Backup Server"
EXE=/usr/libexec/$NAME/salvanza-svc
DIR=$NAME-service-$VERSION-linux-$ARCH
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT INT TERM
mkdir -p "$WORK/$DIR/init" "$OUT"

echo "==> building salvanza-svc and salvanza-cli ($ARCH)"
(cd "$ROOT/gui" && CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build -trimpath -tags service \
    -ldflags "-s -w -X main.appVersion=$VERSION" -o "$WORK/$DIR/salvanza-svc" .)
(cd "$ROOT" && CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build -trimpath \
    -ldflags "-s -w -X main.version=$VERSION" -o "$WORK/$DIR/salvanza-cli" ./cli)

subst() { sed -e "s|@NAME@|$NAME|g" -e "s|@EXE@|$EXE|g" -e "s|@TITLE@|$TITLE|g" "$1"; }
subst "$ROOT/gui/initscripts/service.sysv" > "$WORK/$DIR/init/salvanza.sysv"
subst "$ROOT/gui/initscripts/service.openrc" > "$WORK/$DIR/init/salvanza.openrc"
sed "s|^ExecStart=.*|ExecStart=$EXE|" "$ROOT/packaging/linux/salvanza.service" > "$WORK/$DIR/init/salvanza.service"
cp "$HERE/install.sh" "$WORK/$DIR/install.sh"
cp "$HERE/README.txt" "$WORK/$DIR/README.txt"
chmod 0755 "$WORK/$DIR/salvanza-svc" "$WORK/$DIR/salvanza-cli" "$WORK/$DIR/install.sh" \
    "$WORK/$DIR/init/salvanza.sysv" "$WORK/$DIR/init/salvanza.openrc"
chmod 0644 "$WORK/$DIR/init/salvanza.service" "$WORK/$DIR/README.txt"

tar -C "$WORK" --owner=0 --group=0 -czf "$OUT/$DIR.tar.gz" "$DIR"
echo "==> done: $OUT/$DIR.tar.gz"
