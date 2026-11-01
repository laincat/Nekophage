#!/bin/sh
# Opens the source's own favourites view and checks that it lists something.
#
#   NK_FAV_SRC=/path/to/source.js sh scripts/favorites-probe.sh
#   sh scripts/favorites-probe.sh /path/to/source.js
#
# Source favourites live in a top-level `favorites` block, not inside `comic`, so
# getting that wrong produces an empty page rather than an error — which is why
# this asks the view for its contents instead of looking at a capture. An empty
# listing and a full one capture to nearly the same size; that mistake has already
# been made once on the comment thread.
#
# A source that needs a login, or fails on its own side, reports an error string;
# that string is printed so the failure stays attributable instead of reading as a
# client defect.
set -eu

cd "$(dirname "$0")/.."
src=${1:-${NK_FAV_SRC:-}}
if [ -z "$src" ] || [ ! -f "$src" ]; then
	printf 'favorites-probe: pass a source script (or set NK_FAV_SRC)\n' >&2
	exit 2
fi

out=build/probes
mkdir -p "$out"
log=$out/favorites.log
shot=$out/favorites.png

go run ./cmd/uicheck -script "$src" -favorites -out "$shot" >"$log" 2>&1

line=$(grep -E 'favorites: count=' "$log" | tail -1 || true)
if [ -z "$line" ]; then
	printf 'favorites-probe: the tool reported no favourites state; see %s\n' "$log" >&2
	exit 1
fi

count=$(printf '%s' "$line" | sed -n 's/.*count=\([0-9]*\).*/\1/p')
first=$(printf '%s' "$line" | sed -n 's/.*first="\([^"]*\)".*/\1/p')
reason=$(printf '%s' "$line" | sed -n 's/.*err="\([^"]*\)".*/\1/p')

if [ -z "$count" ] || [ "$count" -eq 0 ] || [ -z "$first" ]; then
	printf 'favorites-probe: the view lists nothing (%s)\n' "${reason:-no reason given}" >&2
	exit 1
fi

printf 'favorites-probe: %s\n' "$line"
printf 'favorites-probe: capture %s\n' "$shot"
