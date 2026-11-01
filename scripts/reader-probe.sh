#!/bin/sh
# Opens the reader on a real chapter and checks that pages actually decode.
#
#   NK_READ_SRC=/path/to/manhuagui.js NK_READ_QUERY=海贼王 sh scripts/reader-probe.sh
#
# This is the one check that needs a real site *and* a real decoder: everything
# else about images can be answered without one — a source can return byte counts
# that look right and still not be a picture. The tool exits non-zero when nothing
# decoded, so the failure is not something this script has to infer.
#
# A site that refuses the request is not a defect here: the tool's own message is
# passed through so the reason stays attributable.
set -eu

cd "$(dirname "$0")/.."
src=${NK_READ_SRC:-}
if [ -z "$src" ] || [ ! -f "$src" ]; then
	printf 'reader-probe: set NK_READ_SRC to a source script\n' >&2
	exit 2
fi
query=${NK_READ_QUERY:-海贼王}

out=build/probes
mkdir -p "$out"
log=$out/reader.log
shot=$out/reader.png

# NK_READ_SERVE=18731 serves one PNG on loopback, for a source whose chapter points
# at 127.0.0.1 — the same trick the iOS interface tests use. That makes the decode
# check runnable with no site at all.
serve=""
if [ -n "${NK_READ_SERVE:-}" ]; then
	serve="-serve ${NK_READ_SERVE}"
fi

rc=0
# shellcheck disable=SC2086 # serve is either empty or one flag and its value
go run ./cmd/readercheck -script "$src" -query "$query" $serve -out "$shot" >"$log" 2>&1 || rc=$?

if [ $rc -ne 0 ]; then
	# The last real log line, not the go run wrapper's "exit status 1":
	# the reason has to name the site's answer.
	last=$(grep -E '^[0-9]{4}/[0-9]{2}/[0-9]{2} ' "$log" | tail -1 || true)
	printf 'reader-probe: %s\n' "${last:-see $log}" >&2
	exit 1
fi

# The line names the page count and how many images decoded; without it the tool
# finished for a reason other than a rendered chapter.
line=$(grep -E 'page title=.* images=.* decoded=' "$log" | tail -1 || true)
if [ -z "$line" ]; then
	printf 'reader-probe: the tool reported no page state; see %s\n' "$log" >&2
	exit 1
fi

printf 'reader-probe: %s\n' "$line"
printf 'reader-probe: capture %s\n' "$shot"
