#!/bin/sh
# Opens the browse view and checks that the chip row follows the rule.
#
#   sh scripts/browse-probe.sh /path/to/source.js
#
# The rule: a source that declares one section gets no chip row, because a single
# chip that is always selected says nothing; two or more get a chip per section.
# The iOS client already worked that way and the desktop was changed to match, so
# this is where the two stay in step — a capture shows a row or no row, but only
# this says how many chips the view believes it drew.
#
# The script also needs the current section to hold comics: a chip row over an
# empty list is a browse that did not load.
set -eu

cd "$(dirname "$0")/.."
src=${1:-${NK_BROWSE_SRC:-}}
if [ -z "$src" ] || [ ! -f "$src" ]; then
	printf 'browse-probe: pass a source script (or set NK_BROWSE_SRC)\n' >&2
	exit 2
fi

out=build/probes
mkdir -p "$out"
log=$out/browse.log
shot=$out/browse.png

go run ./cmd/uicheck -script "$src" -browse -out "$shot" >"$log" 2>&1

line=$(grep -E 'browse: sections=' "$log" | tail -1 || true)
if [ -z "$line" ]; then
	printf 'browse-probe: the tool reported no browse state; see %s\n' "$log" >&2
	exit 1
fi

num() { printf '%s' "$line" | sed -n "s/.*$1=\\([0-9]*\\).*/\\1/p"; }
sections=$(num sections)
chips=$(num chips)
comics=$(num comics)
reason=$(printf '%s' "$line" | sed -n 's/.*err="\([^"]*\)".*/\1/p')

if [ -z "$sections" ] || [ "$sections" -eq 0 ]; then
	printf 'browse-probe: the source offers no section (%s)\n' "${reason:-no reason given}" >&2
	exit 1
fi

want=$sections
[ "$sections" -le 1 ] && want=0
if [ "$chips" -ne "$want" ]; then
	printf 'browse-probe: %s section(s) should draw %s chip(s), the view drew %s\n' \
		"$sections" "$want" "$chips" >&2
	exit 1
fi

if [ -z "$comics" ] || [ "$comics" -eq 0 ]; then
	printf 'browse-probe: the current section holds no comics (%s)\n' "${reason:-no reason given}" >&2
	exit 1
fi

printf 'browse-probe: %s\n' "$line"
printf 'browse-probe: capture %s\n' "$shot"
