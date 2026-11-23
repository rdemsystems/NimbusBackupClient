#!/bin/sh
# Entry point of the salvanza-nas image.
#
#   serve (default)  the Salvanza service (Backup Sets, schedules, restores)
#                    and the text window on http://<nas>:7681 behind a login
#   tui              the text window in this terminal (docker exec -it ... tui)
#   demo             the text window with invented data, no service needed
#   <anything else>  passed to salvanza-cli as is (e.g. "backup-folder -h")
#
# Settings for "serve":
#   UI_USER (default admin), UI_PASSWORD or UI_PASSWORD_FILE  the web login;
#     required: the text window can do anything the service can, as root.
#   UI_PORT (default 7681)
#   TZ  time zone of the schedules, e.g. Europe/Paris
set -eu

log() { printf '%s [salvanza-nas] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"; }

# A 256-colour terminal in the browser: the window uses the app's blue.
export TERM="${TERM:-xterm-256color}" COLORTERM="${COLORTERM:-truecolor}"

serve() {
	if [ -n "${UI_PASSWORD_FILE:-}" ]; then
		[ -r "$UI_PASSWORD_FILE" ] || { log "cannot read UI_PASSWORD_FILE=$UI_PASSWORD_FILE"; exit 1; }
		UI_PASSWORD="$(cat "$UI_PASSWORD_FILE")"
	fi
	[ -n "${UI_PASSWORD:-}" ] || {
		log "UI_PASSWORD (or UI_PASSWORD_FILE) must be set: the web window has full control of the backups"
		exit 1
	}
	user="${UI_USER:-admin}" port="${UI_PORT:-7681}"

	log "starting the Salvanza service ($(salvanza-cli backup-folder -version 2>&1 | awk '{print $3}'), TZ=${TZ:-UTC})"
	salvanza-svc &
	svc=$!

	# The service writes its API token when it starts; the window needs it.
	i=0
	while [ ! -s /var/lib/salvanza/api-token ] && [ $i -lt 30 ]; do
		kill -0 "$svc" 2>/dev/null || { log "the service stopped while starting"; exit 1; }
		sleep 1; i=$((i + 1))
	done

	log "web text window on port $port (user '$user')"
	# -W: the browser may type (ttyd is read-only otherwise). Each browser tab
	# gets its own text window; closing it leaves the backups running.
	ttyd -W -p "$port" -c "$user:$UI_PASSWORD" \
		-t titleFixed="Salvanza" -t fontSize=15 -t disableLeaveAlert=true \
		salvanza-cli tui &
	ui=$!

	# Stop both on docker stop, giving a running backup the service's own
	# shutdown; and stop the container if either one dies.
	trap 'log "stopping"; kill -TERM "$svc" "$ui" 2>/dev/null; wait "$svc" 2>/dev/null; exit 0' TERM INT
	while kill -0 "$svc" 2>/dev/null && kill -0 "$ui" 2>/dev/null; do
		sleep 5 & wait $!
	done
	log "$(kill -0 "$svc" 2>/dev/null && echo 'the web window' || echo 'the service') exited, stopping the container"
	kill -TERM "$svc" "$ui" 2>/dev/null || true
	wait 2>/dev/null || true
	exit 1
}

case "${1:-serve}" in
	serve) serve ;;
	tui) shift; exec salvanza-cli tui "$@" ;;
	demo) shift; exec salvanza-cli tui -demo "$@" ;;
	*) exec salvanza-cli "$@" ;;
esac
