#!/bin/sh
# Runs every source's browse pages against the real network.
#
#   NK_LIVE_DIR=/path/to/scripts scripts/live-explore.sh
#
# The sweep itself is cmd/explorecheck. It exists because the reading path —
# search, details, chapter, images — never touched `explore`, so a whole tab of
# every source had no live evidence: ehentai's three browses were unusable for as
# long as nobody looked.
#
# One source failing is a result, not a crash: the summary separates a site that
# refused from a browse the client could not read. Sources that need a server the
# user runs are listed separately, and NK_LIVE_SERVER_SOURCES=1 includes them.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

dir=${NK_LIVE_DIR:-}
if [ -z "$dir" ] || [ ! -d "$dir" ]; then
	printf 'live-explore: set NK_LIVE_DIR to a directory of source scripts\n' >&2
	exit 2
fi

out=${NK_LIVE_LOG_DIR:-build/live-explore}
mkdir -p "$out"

# A browse that failed once usually fails the same way twice, but a timeout
# reads exactly like a site refusing; NK_LIVE_REPEAT=2 makes the difference
# visible.
repeat=${NK_LIVE_REPEAT:-1}
status=0
run=1
while [ "$run" -le "$repeat" ]; do
	log="$out/sweep-$run.log"
	if go run ./cmd/explorecheck -dir "$dir" -timeout "${NK_LIVE_TIMEOUT:-60s}" >"$log" 2>&1; then
		summary=ok
	else
		summary=failed
		status=1
	fi
	printf 'live-explore %d/%d: %s (see %s)\n' "$run" "$repeat" "$summary" "$log"
	grep -E '^(explorecheck|failed|needs a server|no browses)' "$log" || true
	run=$((run + 1))
done
exit "$status"
