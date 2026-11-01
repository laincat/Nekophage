#!/bin/sh
# Runs every callback the official sources declare, without the interface.
#
# A callback is a button the source names itself, and pressing it runs code the
# source wrote: some only clear a cache, others reach a site and then show their
# own dialog. A capture tool cannot press those buttons, so this drives the same
# Core call directly and reports what each one answered.
#
#   NK_OFFICIAL_SRC=/path/to/official-srcs sh scripts/callback-probe.sh [script...]
#
# Which settings are callbacks comes from Core, not from reading the file: a text
# search picks up lookalikes and then reports a source as broken for a button it
# never declared.
set -eu

cd "$(dirname "$0")/.."
srcs=${NK_OFFICIAL_SRC:-}
if [ -z "$srcs" ] || [ ! -d "$srcs" ]; then
	printf 'callback-probe: set NK_OFFICIAL_SRC to the official source directory\n' >&2
	exit 2
fi

out=build/callback-probe
mkdir -p "$out"
bin="$out/cbprobe"
case "$(uname -s)" in
MINGW* | MSYS* | CYGWIN*) bin="$bin.exe" ;;
esac
go build -o "$bin" ./cmd/cbprobe

scripts=${*:-""}
if [ -z "$scripts" ]; then
	scripts=$(grep -l 'type: *"callback"' "$srcs"/*.js 2>/dev/null | sed 's|.*/||')
fi

failed=0
for file in $scripts; do
	case "$file" in
	*"$srcs"/*) path=$file ;;
	*) path="$srcs/$file" ;;
	esac
	name=$(basename "$file" .js)
	log="$out/$name.log"
	if timeout 300 "$bin" "$path" - --all 30 >"$log" 2>&1; then
		while IFS= read -r line; do
			case "$line" in
			# A source's own error is a result, not a broken probe: jm refuses a
			# check-in without a login, which is the source talking.
			ANSWER* | NONE* | ERROR*) printf '%s\n' "$line" ;;
			TIMEOUT*) printf '%s\n' "$line"; failed=1 ;;
			esac
		done <"$log"
	else
		printf 'FAIL    %s  (%s)\n' "$name" "$log"
		tail -2 "$log" || true
		failed=1
	fi
done

printf 'logs: %s\n' "$out"
exit $failed
