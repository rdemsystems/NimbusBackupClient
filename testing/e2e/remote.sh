#!/usr/bin/env bash
# Host side of the cross-platform e2e (e2e-matrix.yml, job "vm"): the backup
# itself runs in a VM (vm-backup.sh), everything else runs here, on Linux,
# with the same helpers as run.sh.
#
#   remote.sh up       start + configure PBS, make the fixture and the key,
#                      write $WORK/remote.env for the VM
#   remote.sh verify   restore the VM's snapshots with the official client
#                      and compare them with the fixture; PBS verify
#   remote.sh down     remove the PBS container
#
# E2E_OS names the system the backup ran on (backup IDs vm-<os>, vm-<os>-enc).
set -euo pipefail
# run.sh for its fixtures and checks (it sources lib.sh); sourced, it does
# not run the suite.
# shellcheck source=run.sh
. "$(dirname "$0")/run.sh"

E2E_OS="${E2E_OS:?E2E_OS must name the VM system}"
mkdir -p "$WORK"

up() {
	command -v docker >/dev/null 2>&1 || die "docker is required"
	make_directory_fixture
	detect_pbc
	start_pbs
	configure_pbs
	make_keyfile
	# Plain KEY=value lines, read by a POSIX sh in the VM. The token belongs
	# to a container that only lives for this job.
	cat >"$WORK/remote.env" <<-EOF
		PBS_PORT='$PBS_PORT'
		PBS_FINGERPRINT='$PBS_FINGERPRINT'
		PBS_AUTHID='$PBS_AUTHID'
		PBS_TOKEN_SECRET='$PBS_TOKEN_SECRET'
		PBS_DATASTORE='$PBS_DATASTORE'
	EOF
	ok "PBS ready for the $E2E_OS VM; settings in $WORK/remote.env"
}

# assert_tree_off_linux <src> <restored> <label>: what a backup made off
# Linux keeps. pxar_meta_other.go (posixFidelity = false) skips symbolic links
# there, as on Windows, so files and folders must match and no link may come
# back. If links start being stored on these systems, this fails on purpose:
# switch the check to assert_tree_identical then.
assert_tree_off_linux() {
	local a="$1" b="$2" label="$3"
	diff -u <(tree_manifest "$a") <(tree_manifest "$b") >&2 \
		|| die "$label: restored files differ from the source"
	diff -u <(tree_dirs "$a") <(tree_dirs "$b") >&2 \
		|| die "$label: restored directory tree differs from the source"
	local links
	links="$(tree_links "$b")"
	[ -z "$links" ] || die "$label: symbolic links came back, which off Linux used to be skipped: $links"
	ok "$label: $(tree_manifest "$a" | wc -l) file(s) byte-identical, tree layout identical, $(tree_links "$a" | wc -l) symlink(s) skipped as documented off Linux"
}

verify() {
	# A new process: take back what `up` found (configure_pbs's exports are gone).
	# shellcheck disable=SC1091
	. "$WORK/remote.env"
	export PBS_FINGERPRINT PBS_TOKEN_SECRET
	export PBS_PASSWORD="$PBS_TOKEN_SECRET"
	export PBS_REPOSITORY="${PBS_AUTHID}@127.0.0.1:${PBS_PORT}:${PBS_DATASTORE}"
	detect_pbc   # the image is cached from `up`, so this is quick

	section "Snapshots made on $E2E_OS"
	pbc_restore "host/vm-$E2E_OS" backup.pxar.didx "restore-$E2E_OS"
	assert_tree_off_linux "$WORK/src" "$WORK/restore-$E2E_OS" "$E2E_OS directory backup"
	assert_backup_meta "$WORK/restore-$E2E_OS" "$WORK/src" "vm-$E2E_OS"

	pbc_restore "host/vm-$E2E_OS-enc" backup.pxar.didx "restore-$E2E_OS-enc" --keyfile key.pem
	assert_tree_off_linux "$WORK/src" "$WORK/restore-$E2E_OS-enc" "$E2E_OS encrypted directory backup"

	test_server_side_verify || die "PBS verify failed"
}

case "${1:-}" in
	up) up ;;
	verify) verify ;;
	down) remove_pbs ;;
	*) die "usage: $0 up|verify|down" ;;
esac
