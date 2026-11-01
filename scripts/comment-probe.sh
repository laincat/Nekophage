#!/bin/sh
# Opens a comic's comment thread in a real window and captures it.
#
# The desktop client has driven comment threads against real sources; what this
# adds is a run that needs no network and no login, so the screen can be checked
# on any machine. Point it at a source script that answers `loadComments` — the
# iOS repository's UITests/Fixtures/callback_probe.js is one.
#
#   sh scripts/comment-probe.sh /path/to/source.js
#
# A thread that opens on an empty page still draws its entry, so the capture has
# to show more than a frame: the file size is the cheapest signal that something
# was drawn, and the log line names where the image went.
set -eu

cd "$(dirname "$0")/.."
src=${1:-}
if [ -z "$src" ] || [ ! -f "$src" ]; then
	printf 'comment-probe: pass a source script that answers loadComments\n' >&2
	exit 2
fi

out=build/probes
mkdir -p "$out"
shot=$out/comments.png
rm -f "$shot"

go run ./cmd/uicheck -script "$src" -comments -comments-out "$shot" >"$out/comments-uicheck.log" 2>&1

if [ ! -f "$shot" ]; then
	printf 'comment-probe: no capture was written; see %s\n' "$out/comments-uicheck.log" >&2
	exit 1
fi

# A blank frame compresses to almost nothing; a thread with an author, a body and
# its buttons does not. The threshold is deliberately low: this separates "drew
# something" from "drew nothing", not one thread from another.
bytes=$(wc -c <"$shot" | tr -d ' ')
if [ "$bytes" -lt 4000 ]; then
	printf 'comment-probe: the capture is %s bytes, which is a blank frame\n' "$bytes" >&2
	exit 1
fi

printf 'comment-probe: thread captured to %s (%s bytes)\n' "$shot" "$bytes"
