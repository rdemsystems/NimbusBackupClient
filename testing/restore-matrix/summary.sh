#!/bin/sh
# summary.sh <title> <report>...: the matrix's summary lines as a table in
# the GitHub job summary (or on stdout outside Actions).
title="$1"
shift
{
	echo "### $title"
	echo
	echo "| restored on | archive made on | restored by | | ok | ok by design | failed |"
	echo "|---|---|---|---|---:|---:|---:|"
	cat "$@" 2>/dev/null || echo "| no report | | | ❌ | | | |"
	echo
} >>"${GITHUB_STEP_SUMMARY:-/dev/stdout}"
