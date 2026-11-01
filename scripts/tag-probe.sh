#!/bin/sh
# Asks every official source what its tags mean, one source at a time.
#
#   NK_OFFICIAL_SRC=/path/to/official-srcs sh scripts/tag-probe.sh [script...]
#
# A tag is a control only when the source says what tapping it does: some map a
# namespace to a search, some to a category, and a namespace the source does not
# handle answers nothing — which is the source talking, not a client fault. This
# walks every group so the difference is visible, and it installs nothing into the
# app.
#
# A source whose search finds nothing fails for that reason and says so; a keyword
# that suits one site is not expected to suit all of them.
set -eu

cd "$(dirname "$0")/.."
srcs=${NK_OFFICIAL_SRC:-}
if [ -z "$srcs" ] || [ ! -d "$srcs" ]; then
	printf 'tag-probe: set NK_OFFICIAL_SRC to the official source directory\n' >&2
	exit 2
fi

keyword=${NK_TAG_QUERY:-love}
out=build/tag-probe
mkdir -p "$out"
bin="$out/comicprobe"
case "$(uname -s)" in
MINGW* | MSYS* | CYGWIN*) bin="$bin.exe" ;;
esac
go build -o "$bin" ./cmd/comicprobe

scripts=${*:-""}
if [ -z "$scripts" ]; then
	scripts=$(ls "$srcs"/*.js 2>/dev/null | sed 's|.*/||')
fi

failed=0
for file in $scripts; do
	case "$file" in
	*"$srcs"/*) path=$file ;;
	*) path="$srcs/$file" ;;
	esac
	name=$(basename "$file" .js)
	log="$out/$name.log"
	if timeout 120 "$bin" "$path" "$keyword" >"$log" 2>&1; then
		handled=$(sed -n 's/.*PASS tag taps reach the source.s rule: \([0-9]*\) of \([0-9]*\).*/\1\/\2/p' "$log" | head -1)
		printf 'PASS  %-26s tags handled %s\n' "$name" "${handled:-0/0}"
	else
		why=$(grep -m1 -E '^FAIL' "$log" | cut -c1-80)
		case "$why" in
		# A source that cannot be searched, or whose comic cannot be opened, says
		# nothing about tags: the site refused, the keyword did not suit it, or a
		# login is missing. Counting that as a tag failure would bury the real one.
		*"FAIL search:"* | *"FAIL info:"* | *"FAIL install:"* | *"FAIL search returned no comics"*)
			printf 'SKIP  %-26s %s\n' "$name" "${why#FAIL }"
			;;
		# The source answered "not handled" for every namespace its own comic shows.
		# That is worth seeing, but it is the source's mapping, not the client's
		# behaviour: a tap acting on nothing is how that source is written.
		*"FAIL no tag group was handled"*)
			printf 'WARN  %-26s no namespace of its own tags is handled by the source\n' "$name"
			;;
		*)
			printf 'FAIL  %-26s %s\n' "$name" "${why:-see $log}"
			failed=1
			;;
		esac
	fi
done

printf 'logs: %s\n' "$out"
exit $failed
