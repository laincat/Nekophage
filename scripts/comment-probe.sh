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
# A thread that opens on an empty page still draws its entry, so the check cannot
# be "a window appeared": an empty thread and a real one capture to almost the same
# number of bytes. The window is asked what it holds instead.
set -eu

cd "$(dirname "$0")/.."
src=${1:-internal/reader/testdata/loopback_source.js}
if [ -z "$src" ] || [ ! -f "$src" ]; then
	printf 'comment-probe: no source script at %s\n' "$src" >&2
	exit 2
fi

out=build/probes
mkdir -p "$out"
shot=$out/comments.png
rm -f "$shot"

go run ./cmd/uicheck -script "$src" -comments -comments-out "$shot" >"$out/comments-uicheck.log" 2>&1

line=$(grep -E 'comments: count=' "$out/comments-uicheck.log" | tail -1 || true)
if [ -z "$line" ]; then
	printf 'comment-probe: the tool reported no comment state; see %s
' "$out/comments-uicheck.log" >&2
	exit 1
fi

if ! printf '%s' "$line" | grep -q 'count=0 '; then
	: # a thread was read
else
	reason=$(printf '%s' "$line" | sed -n 's/.*err="\(.*\)"$/\1/p')
	printf 'comment-probe: the window holds no comments (%s)
' "${reason:-no reason given}" >&2
	exit 1
fi

if printf '%s' "$line" | grep -q 'author="" body=""'; then
	printf 'comment-probe: a comment arrived with no author and no body: %s
' "$line" >&2
	exit 1
fi

if [ ! -f "$shot" ]; then
	printf 'comment-probe: no capture was written; see %s
' "$out/comments-uicheck.log" >&2
	exit 1
fi

bytes=$(wc -c <"$shot" | tr -d ' ')
printf 'comment-probe: %s
' "$line"
printf 'comment-probe: thread captured to %s (%s bytes)
' "$shot" "$bytes"
