#!/usr/bin/env bash
# Builds and tests the applications that use nui against this working copy
# of nui, and against the nui version each of them pins, to tell the
# breakages the local changes make from those that were there before.
#
#   scripts/check-clients.sh             # the modules next to nui (../*)
#   scripts/check-clients.sh ../altfm    # the given ones
#   QUICK=1 scripts/check-clients.sh     # build and vet only, no tests
#
# A client is a Go module whose go.mod requires github.com/ipoluianov/nui.
# The local nui is put in with a temporary go.work, so the clients'
# go.mod files are not touched. Exit status 1 if a client works with its
# pinned nui but not with the local one.

set -u

nui=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
# The logs of the failures with the local nui
logs=${TMPDIR:-/tmp}/nui-clients
rm -rf "$logs"

if [ $# -gt 0 ]; then
	dirs=()
	for d in "$@"; do
		dirs+=("$(cd "$d" && pwd)")
	done
else
	dirs=()
	for d in "$nui"/../*/; do
		d=$(cd "$d" && pwd)
		[ "$d" = "$nui" ] && continue
		[ -f "$d/go.mod" ] || continue
		grep -qE '^[[:space:]]*(require[[:space:]]+)?github.com/ipoluianov/nui[[:space:]]' "$d/go.mod" || continue
		dirs+=("$d")
	done
fi

goversion=$(awk '$1 == "go" { print $2; exit }' "$nui/go.mod")

# check runs the steps in the module with the given GOWORK; prints the
# first failing step or "ok", the details go to the log
check() {
	local dir=$1 gowork=$2 log=$3
	(
		cd "$dir" || exit 1
		export GOWORK=$gowork
		echo "== build" && go build ./... || { echo "FAIL build"; exit 1; }
		echo "== vet" && go vet ./... || { echo "FAIL vet"; exit 1; }
		if [ -z "${QUICK:-}" ]; then
			echo "== test" && go test -count=1 -timeout 300s ./... || { echo "FAIL test"; exit 1; }
		fi
		echo "OK"
	) >"$log" 2>&1
	tail -n 1 "$log"
}

printf '%-20s %-14s %-14s\n' client pinned local
regressions=0
for dir in "${dirs[@]}"; do
	name=$(basename "$dir")
	cat >"$work/$name.work" <<EOF
go $goversion

use (
	$dir
	$nui
)
EOF
	pinned=$(check "$dir" off "$work/$name.pinned.log")
	local_=$(check "$dir" "$work/$name.work" "$work/$name.local.log")
	printf '%-20s %-14s %-14s\n' "$name" "$pinned" "$local_"
	if [ "$local_" != OK ]; then
		mkdir -p "$logs"
		cp "$work/$name.local.log" "$logs/$name.log"
		if [ "$pinned" = OK ]; then
			regressions=$((regressions + 1))
		fi
	fi
done

if [ -d "$logs" ]; then
	echo
	echo "Logs of the failures with the local nui: $logs"
fi
if [ $regressions -gt 0 ]; then
	echo "$regressions client(s) broken by the local nui"
	exit 1
fi
