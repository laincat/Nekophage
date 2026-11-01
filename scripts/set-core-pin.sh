#!/bin/sh
# Points the client at a Core revision, everywhere that records it.
#
#   scripts/set-core-pin.sh <40 hex chars>
#
# A pin lives in more than one place, and a release that updates some of them
# leaves the client claiming one Core and linking another. That has already
# happened once: the published Core history is a single amended commit, so an old
# pin stops existing and a half-updated tree is worse than an outdated one.
#
# What this does NOT do: fetch or rebuild Core. Run scripts/fetch-core.sh after it,
# which regenerates the stamp and proves the linked module is the pinned revision.
set -eu

cd "$(dirname "$0")/.."

pin=${1:-}
case "$pin" in
[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]) ;;
*)
	printf 'set-core-pin: need a full 40-character revision, got %s\n' "${pin:-nothing}" >&2
	exit 2
	;;
esac

old=$(tr -d ' \r\n' <scripts/core-version)
printf '%s\n' "$pin" >scripts/core-version

# The stamp is generated from this file, so it is only rewritten here when a
# previous fetch left it pointing elsewhere.
if [ -f corestamp.go ] && ! grep -q "SetRelease(\"$pin\")" corestamp.go; then
	printf 'set-core-pin: corestamp.go still says %s; run sh scripts/fetch-core.sh\n' \
		"$(sed -n 's/.*SetRelease("\([0-9a-f]*\)").*/\1/p' corestamp.go)"
fi

printf 'set-core-pin: %s -> %s\n' "${old:-none}" "$pin"
printf 'set-core-pin: next, run sh scripts/fetch-core.sh\n'
