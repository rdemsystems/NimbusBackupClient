#!/bin/sh
# summary.sh <title> <report>... [--gaps <gap report>]: the matrix's summary
# lines as a table in the GitHub job summary (or on stdout outside Actions),
# then the known gaps, if a gap report is given.
title="$1"
shift
reports="" gaps=""
while [ $# -gt 0 ]; do
	case "$1" in
	--gaps) gaps="$2"; shift 2 ;;
	*) reports="$reports $1"; shift ;;
	esac
done
{
	echo "### $title"
	echo
	echo "| restored on | archive made on | restored by | | ok | ok by design | failed |"
	echo "|---|---|---|---|---:|---:|---:|"
	# shellcheck disable=SC2086
	cat $reports 2>/dev/null || echo "| no report | | | ❌ | | | |"
	echo
	if [ -n "$gaps" ] && [ -s "$gaps" ]; then
		echo "#### Known gaps (testing/restore-matrix/README.md)"
		echo
		echo "| | gap | state | what happens |"
		echo "|---|---|---|---|"
		cat "$gaps"
		echo
	fi
} >>"${GITHUB_STEP_SUMMARY:-/dev/stdout}"
