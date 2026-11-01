#!/bin/sh
# Runs the contract checks that need real source scripts.
#
#   NK_OFFICIAL_SRC=/path/to/scripts sh scripts/live-contracts.sh
#
# These are not in `scripts/check.sh` because they need other people's source
# files, which cannot live in this repository. They are also not about the
# network: every one of them answers from the script itself, so a failure here is
# a client that reads a declaration wrongly rather than a site that changed.
#
#   1. the level this client reports satisfies every source's minAppVersion
#   2. a browser login hands over its cookies before the source's callback runs
#   3. logging out deletes what the source deletes, and no more
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

dir=${NK_OFFICIAL_SRC:-}
if [ -z "$dir" ] || [ ! -d "$dir" ]; then
	printf 'live-contracts: set NK_OFFICIAL_SRC to a directory of source scripts\n' >&2
	exit 2
fi

out=${NK_CONTRACT_LOG_DIR:-build/contracts}
mkdir -p "$out"

fail=0
run() {
	# run <name> <command...>
	name=$1
	shift
	if "$@" >"$out/$name.log" 2>&1; then
		printf 'PASS  %s  (%s)\n' "$name" "$out/$name.log"
	else
		printf 'FAIL  %s  (%s)\n' "$name" "$out/$name.log"
		grep -m1 -E 'FAIL:|requires|cookie' "$out/$name.log" | sed 's/^/      /' || true
		fail=1
	fi
}

NK_OFFICIAL_SRC=$dir run "source-api-level" go test ./internal/sourceapi -run TestLevelMeetsEveryOfficialSourceFloor -count=1
NK_OFFICIAL_SRC=$dir run "webview-login-handoff" go test ./internal/core -run TestWebviewLoginHandoff -count=1
NK_OFFICIAL_SRC=$dir run "logout-contract" go test ./internal/core -run TestLogout -count=1

if [ "$fail" -ne 0 ]; then
	printf 'live-contracts: a source contract is read wrongly\n'
	exit 1
fi
printf 'live-contracts: ok (source level, browser-login handoff, logout)\n'
