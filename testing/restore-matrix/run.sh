#!/bin/sh
# The backup and restore matrix on one system (see README.md). POSIX sh: the
# BSDs and Solaris do not all have bash. Run as root from the repository
# root, with the test binaries restore-matrix.yml builds in
# dist/restore-matrix/.
#
#   run.sh produce <system>  this system's own matrix (backed up and restored
#                            here, by root and by the other users), and its
#                            archive for the others in restore-matrix-out/archives
#   run.sh restore <system>  every archive in restore-matrix-in (one per
#                            system) restored here, by root and the other users
#
# The summary lines go to restore-matrix-out/report-<step>-<system>.md, the
# known gaps (TestRestoreGaps) to gaps-<step>-<system>.md.
set -eu

step="${1:?produce or restore}" sys="${2:?system (GOOS)}"
[ "$(id -u)" = 0 ] || { echo "run.sh: run as root" >&2; exit 1; }

bin="./dist/restore-matrix/pbscommon-test-$sys"
out="$PWD/restore-matrix-out"
mkdir -p "$out"
chmod +x "$bin"
SALVANZA_PERM_REPORT="$out/report-$step-$sys.md"
SALVANZA_GAP_REPORT="$out/gaps-$step-$sys.md"
export SALVANZA_PERM_REPORT SALVANZA_GAP_REPORT
: >"$SALVANZA_PERM_REPORT"
: >"$SALVANZA_GAP_REPORT"
echo "[restore-matrix] $(uname -a)"

case "$step" in
produce)
	SALVANZA_PERM_PRODUCE="$out/archives"
	export SALVANZA_PERM_PRODUCE
	tests='^(TestPermMatrix|TestPermProduce|TestRestoreGaps|TestFinishDirsSelectedFile|TestApplyDirMode|TestPosixArchiveRoundTrip|TestSelectiveSymlinkRestoreViaBST|TestApplyDirMetadata|TestEntryMetaPosixShape)$'
	;;
restore)
	SALVANZA_PERM_ARCHIVES="$PWD/restore-matrix-in"
	export SALVANZA_PERM_ARCHIVES
	tests='^TestPermRestore$'
	;;
*)
	echo "run.sh: unknown step $step" >&2
	exit 2
	;;
esac

"$bin" -test.v -test.count=1 -test.run "$tests"
