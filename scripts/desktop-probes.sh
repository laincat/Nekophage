#!/bin/sh
# Runs every probe that needs a real window, in one go, and prints a summary.
#
#   sh scripts/desktop-probes.sh
#   NK_OFFICIAL_SRC=/path/to/official-srcs sh scripts/desktop-probes.sh
#   NK_PROBE_SRC=/path/to/source.js sh scripts/desktop-probes.sh
#   NK_READ_SRC=/path/to/source.js sh scripts/desktop-probes.sh
#   NK_FAV_SRC=/path/to/source.js sh scripts/desktop-probes.sh
#
# These are the checks a headless CI cannot do: a WebView login, a challenge page,
# a source's own dialog, closing a window mid-wait, and the callbacks the official
# sources declare. They are separate from scripts/check.sh, which must stay runnable
# without a display.
#
# The official-source part needs NK_OFFICIAL_SRC and is skipped without it: those
# scripts are other people's work under other licences and cannot live here.
set -eu

cd "$(dirname "$0")/.."
out=build/probes
mkdir -p "$out"

failed=0
report() {
	if [ "$1" = 0 ]; then
		printf 'PASS  %s\n' "$2"
	else
		printf 'FAIL  %s  (see %s)\n' "$2" "$3"
		failed=1
	fi
}

# Each command's own status has to be captured where it runs: after `fi`, `$?`
# would be the status of the if-statement, not of the script inside it.
run() {
	label=$1
	log=$2
	shift 2
	rc=0
	"$@" >"$log" 2>&1 || rc=$?
	report "$rc" "$label" "$log"
}

run "login window: ok, timeout, failing callback, late cookie, close" \
	"$out/login.log" sh scripts/login-probe.sh
run "challenge window: cleared and dismissed" \
	"$out/challenge.log" sh scripts/challenge-probe.sh

run "the reader lays two pages side by side" 	"$out/layout-probe.log" sh scripts/layout-probe.sh

go build -o "$out/dialogprobe" ./cmd/dialogprobe
run "closing a source dialog releases its caller" \
	"$out/dialog.log" "$out/dialogprobe"

go build -o "$out/updateprobe" ./cmd/updateprobe
run "a closed window still takes updates and page reads" \
	"$out/update.log" "$out/updateprobe"

# The list a source keeps on its own side. It is asked for its contents rather
# than judged by a capture, for the same reason as the comment thread.
if [ -n "${NK_FAV_SRC:-}" ]; then
	run "the favourites the source keeps are listed" 		"$out/favorites-probe.log" sh scripts/favorites-probe.sh
else
	printf 'SKIP  source favourites (set NK_FAV_SRC to a source script)
'
fi

# Pages that actually decode, which is the one thing about images that cannot be
# answered without a decoder: a source can return plausible byte counts that are
# not a picture. The repository fixture and readercheck own serving make this a
# default check with no site and no fixture from another repository; NK_READ_SRC
# still points it at a real source when one is available.
run "the reader decodes a served page" \
	"$out/reader-probe.log" sh scripts/reader-probe.sh

# A page that does not load has to say so and come back when retried. The server
# fails the first two requests on purpose, so the notice is reachable without a
# network and the retry is what has to recover the picture.
rc=0
go run ./cmd/readercheck -script internal/reader/testdata/loopback_source.js \
	-query probe -serve 18731 -fail 2 -retry -out "$out/reader-retry.png" \
	>"$out/reader-retry.log" 2>&1 || rc=$?
report "$rc" "a failed page offers a retry that recovers" "$out/reader-retry.log"

# The page, not the probe, has to say where reading has got to: posting a report
# from the probe would only prove the endpoint answers.
rc=0
go run ./cmd/readercheck -script internal/reader/testdata/loopback_source.js \
	-query probe -serve 18731 -progress -out "$out/reader-progress.png" \
	>"$out/reader-progress.log" 2>&1 || rc=$?
report "$rc" "the page reports its own reading position" "$out/reader-progress.log"

# Scrolling is remembered per chapter: move down, reopen, look where it landed.
rc=0
go run ./cmd/readercheck -script internal/reader/testdata/loopback_source.js \
	-query probe -serve 18731 -scroll -out "$out/reader-scroll.png" \
	>"$out/reader-scroll.log" 2>&1 || rc=$?
report "$rc" "reopening a chapter returns to its place" "$out/reader-scroll.log"

# A thread the source answers, in a window, with no network: the comment screen is
# the one part of the source contract a capture tool can drive on its own.
if [ -n "${NK_PROBE_SRC:-}" ]; then
	run "a comment thread the source answers" 		"$out/comments.log" sh scripts/comment-probe.sh "$NK_PROBE_SRC"
	# The chip row follows the number of sections a source declares; both clients
	# have to agree on that, so it is a check rather than a screenshot.
	run "the browse chips follow the sections" 		"$out/browse-probe.log" sh scripts/browse-probe.sh "$NK_PROBE_SRC"
	# Reading writes the shelf and the shelf reads it back; the file says which comic,
	# chapter and page, so this is not a picture comparison.
	run "reading lands on the shelf" 		"$out/shelf-probe.log" sh scripts/shelf-probe.sh "$NK_PROBE_SRC"
	# The category contract has two halves, and a page that draws its choosers
	# without loading anything is the failure this one exists to name.
	run "the category page draws and loads" 		"$out/category-probe.log" sh scripts/category-probe.sh "$NK_PROBE_SRC"
else
	printf 'SKIP  comment thread (set NK_PROBE_SRC to a source script)
'
fi

if [ -n "${NK_OFFICIAL_SRC:-}" ]; then
	run "every callback the official sources declare" \
		"$out/callbacks.log" sh scripts/callback-probe.sh
	# Tags are the other thing a source decides for itself, and a namespace it does
	# not handle is a normal answer rather than a fault, so that script separates
	# site failures from source mappings instead of failing on both.
	run "what every source's own tags mean" \
		"$out/tags.log" sh scripts/tag-probe.sh
else
	printf 'SKIP  official source callbacks and tags (set NK_OFFICIAL_SRC)\n'
fi

printf 'logs: %s\n' "$out"
exit $failed
