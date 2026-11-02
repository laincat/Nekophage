#!/bin/sh
# Opens a two-page chapter in a real window and measures the double-page layout.
#
#   sh scripts/layout-probe.sh
#
# The pages come from 127.0.0.1, filled by readercheck itself, and the source is
# the repository fixture `internal/reader/testdata/loopback_source.js`. Before
# that fixture existed this check needed a live site with a multi-page chapter,
# so in practice it never ran on a machine without one.
set -eu

cd "$(dirname "$0")/.."
out=build/probes
mkdir -p "$out"
log=$out/layout.log
shot=$out/layout.png
rm -f "$shot"

rc=0
go run ./cmd/readercheck \
	-script internal/reader/testdata/loopback_source.js \
	-query probe -serve 18731 -layout -out "$shot" >"$log" 2>&1 || rc=$?

if [ "$rc" -ne 0 ]; then
	last=$(grep -E '^[0-9]{4}/[0-9]{2}/[0-9]{2} ' "$log" | tail -1 || true)
	printf 'layout-probe: %s\n' "${last:-see $log}" >&2
	exit 1
fi
line=$(grep -E 'layout: ' "$log" | tail -1 || true)
printf 'layout-probe: %s\n' "$line"
printf 'layout-probe: capture %s\n' "$shot"
