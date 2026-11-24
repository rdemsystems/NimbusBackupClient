#!/bin/sh
# install.sh - installs the Salvanza background service and command line from
# this archive, on a Linux computer with no Salvanza package for it
# (Slackware, Alpine, Gentoo, Void...). Run as root:
#
#   sh install.sh            install (or update) and start the service
#   sh install.sh remove     stop the service, turn it off and remove it
#
# Puts salvanza-svc in /usr/libexec/salvanza and salvanza-cli in /usr/bin,
# then sets the service up for this computer's service manager: systemd, SysV
# init (/etc/init.d), Slackware (/etc/rc.d/rc.salvanza, started from
# rc.local) or OpenRC. The settings, Backup Sets and keys live in
# /var/lib/salvanza and are kept when the service is removed.
set -eu

NAME=salvanza
HERE=$(cd "$(dirname "$0")" && pwd)
EXE=/usr/libexec/$NAME/salvanza-svc
CLI=/usr/bin/salvanza-cli
PIDFILE=/run/$NAME.pid
[ -d /run ] || PIDFILE=/var/run/$NAME.pid

die() { echo "install.sh: $*" >&2; exit 1; }
[ "$(id -u)" = 0 ] || die "run this as root"

# The service manager, as the app tells it (gui/service_init_linux.go).
if [ -d /run/systemd/system ]; then
    INIT=systemd
elif [ -d /run/openrc ]; then
    INIT=openrc
elif [ -f /etc/slackware-version ] && [ -d /etc/rc.d ]; then
    INIT=slackware
elif [ -d /etc/init.d ]; then
    INIT=sysv
else
    INIT=none
fi

tool() {
    for d in $(echo "$PATH" | tr ':' ' ') /usr/sbin /sbin /usr/bin /bin; do
        [ -x "$d/$1" ] && { echo "$d/$1"; return 0; }
    done
    return 1
}

MARKER="# $NAME: Salvanza for Proxmox Backup Server background service (added by the app)"

slack_add() { # file action
    f=$1
    [ -f "$f" ] || printf '#!/bin/sh\n' > "$f"
    if ! grep -qxF "$MARKER" "$f"; then
        printf '\n%s\nif [ -x /etc/rc.d/rc.%s ]; then\n  /etc/rc.d/rc.%s %s\nfi\n' "$MARKER" "$NAME" "$NAME" "$2" >> "$f"
    fi
    chmod 0755 "$f"
}

slack_remove() { # file
    f=$1
    [ -f "$f" ] || return 0
    awk -v m="$MARKER" '
        $0 == m { skip = 4 }
        skip > 0 { skip--; next }
        { lines[++n] = $0 }
        END {
            # the blank line before the block goes with it
            for (i = 1; i <= n; i++) {
                if (lines[i] == "" && i == n) continue
                print lines[i]
            }
        }' "$f" > "$f.new" && cat "$f.new" > "$f" && rm -f "$f.new"
}

service_stop() {
    case "$INIT" in
        systemd) systemctl stop "$NAME.service" 2>/dev/null || true ;;
        openrc) rc-service "$NAME" stop 2>/dev/null || true ;;
        slackware) [ -x "/etc/rc.d/rc.$NAME" ] && "/etc/rc.d/rc.$NAME" stop || true ;;
        sysv) [ -x "/etc/init.d/$NAME" ] && "/etc/init.d/$NAME" stop || true ;;
    esac
}

remove() {
    service_stop
    case "$INIT" in
        systemd)
            systemctl disable "$NAME.service" 2>/dev/null || true
            rm -f "/etc/systemd/system/$NAME.service"
            systemctl daemon-reload || true
            ;;
        openrc)
            rc-update del "$NAME" default 2>/dev/null || true
            rm -f "/etc/init.d/$NAME"
            ;;
        slackware)
            slack_remove /etc/rc.d/rc.local
            slack_remove /etc/rc.d/rc.local_shutdown
            rm -f "/etc/rc.d/rc.$NAME"
            ;;
        sysv)
            if t=$(tool update-rc.d); then "$t" -f "$NAME" remove >/dev/null || true
            elif t=$(tool chkconfig); then "$t" --del "$NAME" || true
            else rm -f /etc/rc[0-6].d/[SK][0-9][0-9]"$NAME"
            fi
            rm -f "/etc/init.d/$NAME"
            ;;
    esac
    rm -f "$EXE" "$CLI" "$PIDFILE"
    rmdir "/usr/libexec/$NAME" 2>/dev/null || true
    echo "The service is removed. Its settings, Backup Sets and keys are kept in /var/lib/$NAME."
}

install_files() {
    [ -f "$HERE/salvanza-svc" ] || die "salvanza-svc is not beside install.sh"
    mkdir -p "/usr/libexec/$NAME"
    service_stop
    cp "$HERE/salvanza-svc" "$EXE.new" && chmod 0755 "$EXE.new" && mv -f "$EXE.new" "$EXE"
    cp "$HERE/salvanza-cli" "$CLI.new" && chmod 0755 "$CLI.new" && mv -f "$CLI.new" "$CLI"
}

install_service() {
    case "$INIT" in
        systemd)
            cp "$HERE/init/salvanza.service" "/etc/systemd/system/$NAME.service"
            systemctl daemon-reload
            systemctl enable --now "$NAME.service"
            ;;
        openrc)
            cp "$HERE/init/salvanza.openrc" "/etc/init.d/$NAME"
            chmod 0755 "/etc/init.d/$NAME"
            rc-update add "$NAME" default >/dev/null
            rc-service "$NAME" start
            ;;
        slackware)
            cp "$HERE/init/salvanza.sysv" "/etc/rc.d/rc.$NAME"
            chmod 0755 "/etc/rc.d/rc.$NAME"
            slack_add /etc/rc.d/rc.local start
            slack_add /etc/rc.d/rc.local_shutdown stop
            "/etc/rc.d/rc.$NAME" start
            ;;
        sysv)
            cp "$HERE/init/salvanza.sysv" "/etc/init.d/$NAME"
            chmod 0755 "/etc/init.d/$NAME"
            if t=$(tool update-rc.d); then "$t" "$NAME" defaults >/dev/null
            elif t=$(tool chkconfig); then "$t" --add "$NAME"
            else
                for l in 2 3 4 5; do [ -d "/etc/rc$l.d" ] && ln -sf "../init.d/$NAME" "/etc/rc$l.d/S80$NAME"; done
                for l in 0 1 6; do [ -d "/etc/rc$l.d" ] && ln -sf "../init.d/$NAME" "/etc/rc$l.d/K20$NAME"; done
            fi
            "/etc/init.d/$NAME" start
            ;;
        none)
            echo "This computer has no service manager this script knows (systemd, SysV init, Slackware's rc.d or OpenRC)."
            echo "salvanza-cli is installed; start $EXE yourself to run the service."
            return 0
            ;;
    esac
    echo "The service runs ($INIT). Use salvanza-cli tui as root to set up servers and Backup Sets."
}

case "${1:-install}" in
    install) install_files; install_service ;;
    remove) remove ;;
    *) die "usage: sh install.sh [install|remove]" ;;
esac
