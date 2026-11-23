#!/bin/sh
# VM side of the cross-platform e2e (e2e-matrix.yml, job "vm"). POSIX sh only:
# FreeBSD, OpenBSD, NetBSD and Solaris do not all have bash.
#
# Runs the cross-built dist/salvanza-cli-<os> against the PBS container on the
# runner, reached through the VM's default gateway (the runner side of the
# NAT). PBS_HOST overrides the address.
set -eu

: "${E2E_OS:?}" "${WORK:?}"
. "$WORK/remote.env"

if [ -z "${PBS_HOST:-}" ]; then
	PBS_HOST=$(netstat -rn | awk '$1 == "default" || $1 == "0.0.0.0" { print $2; exit }')
fi
[ -n "$PBS_HOST" ] || { echo "cannot find the default gateway" >&2; exit 1; }

CLI="./dist/salvanza-cli-$E2E_OS"
chmod +x "$CLI"
echo "[vm] $(uname -a)"
echo "[vm] PBS at https://$PBS_HOST:$PBS_PORT"

backup() {
	"$CLI" backup-folder \
		-baseurl "https://$PBS_HOST:$PBS_PORT" \
		-certfingerprint "$PBS_FINGERPRINT" \
		-authid "$PBS_AUTHID" \
		-secret "$PBS_TOKEN_SECRET" \
		-datastore "$PBS_DATASTORE" \
		-backupdir "$WORK/src" \
		-novss \
		"$@"
}

backup -backup-id "vm-$E2E_OS"
backup -backup-id "vm-$E2E_OS-enc" -keyfile "$WORK/key.pem"
echo "[vm] both backups done"
