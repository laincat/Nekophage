#!/bin/sh
# Runs the live end-to-end test against every source script in a directory.
#
#   NK_LIVE_DIR=/path/to/scripts NK_LIVE_QUERY=test scripts/live-sources.sh
#
# The test itself is TestLiveSource in internal/library. It needs the real
# network, and it needs the source scripts, which cannot live in this repository
# (they are other people's work under other licences).
#
# One source failing is a result, not a crash: the summary says which ones passed
# so a site that changed can be told apart from a client that broke.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

dir=${NK_LIVE_DIR:-}
query=${NK_LIVE_QUERY:-test}
# A site that answers once and fails the next time is the common case, and one run
# cannot tell that apart from a working site. NK_LIVE_REPEAT=2 or 3 makes it visible.
repeat=${NK_LIVE_REPEAT:-1}
if [ -z "$dir" ] || [ ! -d "$dir" ]; then
	printf 'live-sources: set NK_LIVE_DIR to a directory of source scripts\n' >&2
	exit 2
fi

out=${NK_LIVE_LOG_DIR:-build/live}
mkdir -p "$out"

passed=''
failed=''
flaky=''
count=0
for script in "$dir"/*.js; do
	[ -f "$script" ] || continue
	name=$(basename "$script" .js)
	count=$((count + 1))
	ok=0
	bad=0
	round=1
	while [ "$round" -le "$repeat" ]; do
		# Anchored: -run is a prefix match, so an unanchored pattern also ran
		# TestLiveSourceSettingsTakeEffect, whose subject is one source's own
		# quality setting. A source whose setting is broken for its own reasons
		# then failed the patrol, which measures the reading path.
		if NK_LIVE_JS="$script" NK_LIVE_QUERY="$query" \
			go test -run '^TestLiveSource$' -count=1 ./internal/library >"$out/$name.log" 2>&1; then
			ok=$((ok + 1))
		else
			bad=$((bad + 1))
		fi
		round=$((round + 1))
	done
	printf '%-24s ' "$name"
	if [ "$bad" -eq 0 ]; then
		printf 'PASS\n'
		passed="$passed $name"
	elif [ "$ok" -eq 0 ]; then
		printf 'FAIL  (see %s)\n' "$out/$name.log"
		failed="$failed $name"
	else
		printf 'FLAKY %d/%d passed  (see %s)\n' "$ok" "$repeat" "$out/$name.log"
		flaky="$flaky $name"
	fi
done

if [ "$count" -eq 0 ]; then
	printf 'live-sources: no .js files in %s\n' "$dir" >&2
	exit 2
fi

printf 'live-sources: %d source(s); passed:%s\n' "$count" "${passed:- none}"
if [ -n "$flaky" ]; then
	printf 'live-sources: flaky (passed some runs, failed others):%s\n' "$flaky"
fi
if [ -n "$failed" ]; then
	printf 'live-sources: failed:%s\n' "$failed"
	exit 1
fi
if [ -n "$flaky" ]; then
	exit 1
fi
