#!/bin/sh
# Opens the source's category page and checks that it both draws and loads.
#
#   sh scripts/category-probe.sh /path/to/source.js
#
# The contract has two halves: `category` carries the parts a reader chooses from,
# `categoryComics` carries the option list and the loader. A client — or a source —
# that gets only the first half draws choosers that never fill the list, and that
# looks like an empty site rather than a missing function. Counting the parts and
# the comics side by side is what tells those apart.
#
# The view is asked for its contents, not photographed: an empty list and a list of
# one capture to nearly the same size.
set -eu

cd "$(dirname "$0")/.."
src=${1:-${NK_CAT_SRC:-}}
if [ -z "$src" ] || [ ! -f "$src" ]; then
	printf 'category-probe: pass a source script (or set NK_CAT_SRC)\n' >&2
	exit 2
fi

out=build/probes
mkdir -p "$out"
log=$out/category.log
shot=$out/category.png

go run ./cmd/uicheck -script "$src" -category -out "$shot" >"$log" 2>&1

line=$(grep -E 'category: parts=' "$log" | tail -1 || true)
if [ -z "$line" ]; then
	printf 'category-probe: the tool reported no category state; see %s\n' "$log" >&2
	exit 1
fi

num() { printf '%s' "$line" | sed -n "s/.*$1=\([0-9]*\).*/\1/p"; }
parts=$(num parts)
values=$(num values)
comics=$(num comics)
reason=$(printf '%s' "$line" | sed -n 's/.*err="\([^"]*\)".*/\1/p')

if [ -z "$parts" ] || [ "$parts" -eq 0 ]; then
	printf 'category-probe: the source offers no category parts (%s)\n' "${reason:-no reason given}" >&2
	exit 1
fi
if [ -z "$values" ] || [ "$values" -eq 0 ]; then
	printf 'category-probe: the first part lists no values\n' >&2
	exit 1
fi
if [ -z "$comics" ] || [ "$comics" -eq 0 ]; then
	printf 'category-probe: the page drew %s part(s) but loaded no comics (%s)\n' \
		"$parts" "${reason:-no reason given}" >&2
	exit 1
fi

printf 'category-probe: %s\n' "$line"
printf 'category-probe: capture %s\n' "$shot"
