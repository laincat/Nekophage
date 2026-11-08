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

# A negative probe has to fail for the right reason, not just fail. Without the
# marker the check would pass on a broken fixture, a missing source or a port
# collision, which are all failures too.
expect_rejection() {
	label=$1
	log=$2
	wanted=$3
	shift 3
	rc=0
	"$@" >"$log" 2>&1 || rc=$?
	if [ "$rc" -ne 0 ] && grep -q "$wanted" "$log"; then
		printf 'PASS  %s\n' "$label"
	else
		printf 'FAIL  %s  (rc=%s, see %s)\n' "$label" "$rc" "$log"
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
	# One retry, reported rather than hidden. Running a dozen probe processes
	# back to back on one desktop occasionally loses a window before the work
	# starts — the probe then says so and exits non-zero, which is correct but
	# says nothing about the product. A second attempt that passes is printed as
	# RETRY with both logs kept, so the flake stays visible and the suite does
	# not become noise. NK_NO_RETRY=1 turns it off when measuring flakiness.
	if [ "$rc" -ne 0 ] && [ "${NK_NO_RETRY:-0}" != "1" ]; then
		mv "$log" "$log.first"
		rc=0
		"$@" >"$log" 2>&1 || rc=$?
		if [ "$rc" -eq 0 ]; then
			printf 'RETRY %s (first attempt failed, see %s.first)
' "$label" "$log"
		fi
	fi
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
run "the favourites the source keeps are listed" \
	"$out/favorites-probe.log" sh scripts/favorites-probe.sh

# Pages that actually decode, which is the one thing about images that cannot be
# answered without a decoder: a source can return plausible byte counts that are
# not a picture. The repository fixture and readercheck own serving make this a
# default check with no site and no fixture from another repository; NK_READ_SRC
# still points it at a real source when one is available.
run "the reader decodes a served page" \
	"$out/reader-probe.log" sh scripts/reader-probe.sh

# A window that opened but whose reader page never loaded must be rejected.
# The fixture makes the reader's own chapter fetch fail after readercheck has
# picked the chapter, so no figure is ever created. Before the fix the ready
# assertion was cleared by `checkErr = nil`, and a later successful branch could
# turn this into a pass.
expect_rejection "an unloaded reader page is rejected, not reported as read" \
	"$out/reader-no-figures.log" "the reader page never reached a loaded state" \
	go run ./cmd/readercheck -script internal/reader/testdata/no_figures_source.js \
		-query probe -serve 18731 -out "$out/reader-no-figures.png"

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

# Video is the one path where a window can open and show nothing: the page has
# to keep decoding, the position has to advance on its own, and the player has
# to report progress. A local clip through the media proxy needs no site.
rc=0
go run ./cmd/videocheck -media internal/player/testdata/hls/stream.mp4 -range \
	-out "$out/video.png" >"$out/video.log" 2>&1 || rc=$?
report "$rc" "a local clip plays, reports progress and honours byte ranges" "$out/video.log"

# The default suite used to exercise only -range. These three prove the source's
# Referer and the login Cookie really reach the upstream through the media proxy:
# the positive case decodes and resumes, and each negative case must be rejected
# by the fixture with a 403 rather than playing. A fresh profile per run keeps an
# old shelf from hiding whether progress was written this time.
auth_work=$(mktemp -d "$out/video-auth.XXXXXX")
rc=0
go run ./cmd/uicheck -dir "$auth_work/positive" \
	-video-script internal/player/testdata/hls_probe.js \
	-video-media internal/player/testdata/hls -video-auth -video-hlsjs \
	-video clip -video-frame "$out/video-auth-frame.png" \
	>"$out/video-auth.log" 2>&1 || rc=$?
report "$rc" "HLS with Referer and Cookie decodes, ranges and resumes" "$out/video-auth.log"

expect_rejection "a missing Referer is refused with 403" \
	"$out/video-no-referer.log" "auth rejected" \
	go run ./cmd/uicheck -dir "$auth_work/no-referer" \
		-video-script internal/player/testdata/hls_no_referer_probe.js \
		-video-media internal/player/testdata/hls -video-auth -video-hlsjs \
		-video clip -video-frame "$out/video-no-referer.png"

expect_rejection "a missing Cookie is refused with 403" \
	"$out/video-no-cookie.log" "auth rejected" \
	go run ./cmd/uicheck -dir "$auth_work/no-cookie" \
		-video-script internal/player/testdata/hls_no_cookie_probe.js \
		-video-media internal/player/testdata/hls -video-auth -video-hlsjs \
		-video clip -video-frame "$out/video-no-cookie.png"

# Core rewrites EXT-X-MEDIA subtitle URIs through the same proxy. Checked with
# authentication on: a rewritten URI that dropped the Referer/Cookie would be
# refused with 403, the showing track would have no active cue, and this probe
# would fail. The subtitle text itself comes from an active cue, not a cached one.
rc=0
go run ./cmd/uicheck -dir "$auth_work/subtitles" \
	-video-script internal/player/testdata/hls_subs_probe.js \
	-video-media internal/player/testdata/hls-subs -video-auth -video-hlsjs \
	-video-subs-text '字幕渲染检查' -video clip \
	-video-frame "$out/video-subs-auth-frame.png" \
	>"$out/video-subs-auth.log" 2>&1 || rc=$?
report "$rc" "rewritten subtitle URIs keep Referer and Cookie under auth" "$out/video-subs-auth.log"

# The native MP4 decoder is a different path from hls.js: the browser fetches the
# file itself. Authed requests have to survive that path too, so this repeats the
# header check with the native decoder and reads back its decoded audio bytes.
rc=0
go run ./cmd/uicheck -dir "$auth_work/native" \
	-video-script internal/player/testdata/mp4_probe.js \
	-video-media internal/player/testdata/mp4-audio/clip.mp4 -video-auth -video-audio \
	-video clip -video-frame "$out/video-native-auth-frame.png" \
	>"$out/video-native-auth.log" 2>&1 || rc=$?
report "$rc" "the native MP4 decoder also carries Referer and Cookie" "$out/video-native-auth.log"

# A subtitle choice has to outlive the page: choosing English, closing and
# reopening used to fall back to the manifest default. The cue itself must be
# the one rendered after the reopen, not merely the name stored.
rc=0
go run ./cmd/subcheck -dir internal/player/testdata/hls-subs \
	-out "$out/subtitle.png" >"$out/subtitle.log" 2>&1 || rc=$?
report "$rc" "a subtitle choice survives reopening" "$out/subtitle.log"

# A saved choice can arrive before HLS.js finishes selecting its default.
# Restore English first, then choose off and reopen to check a real store update.
restore_work=$(mktemp -d "$out/subtitle-restore.XXXXXX")
cp internal/player/testdata/saved-english.json "$restore_work/shelf.json"
rc=0
go run ./cmd/subcheck -dir internal/player/testdata/hls-subs -work "$restore_work" \
	-baseline-text 'Subtitle rendering proof' -choose '关闭' \
	-out "$out/subtitle-restore.png" >"$out/subtitle-restore.log" 2>&1 || rc=$?
report "$rc" "saved English renders before switching off and reopening" "$out/subtitle-restore.log"
if [ "$rc" -eq 0 ]; then
	rc=0
	go run ./cmd/subcheck -dir internal/player/testdata/hls-subs -work "$restore_work" \
		-baseline-off -choose '关闭' \
		-out "$out/subtitle-off-process.png" >"$out/subtitle-off-process.log" 2>&1 || rc=$?
	report "$rc" "saved off survives a new process" "$out/subtitle-off-process.log"
fi

# Reading writes the shelf and the shelf reads it back; the file says which comic,
# chapter and page, so this is not a picture comparison.
run "reading lands on the shelf" \
	"$out/shelf-probe.log" sh scripts/shelf-probe.sh

# The native interface, not the reader page: a grid that found nothing and a
# grid that rendered capture to almost the same size, so the check reads how
# many results the window holds and what the first one is called.
rc=0
go run ./cmd/uicheck -script internal/reader/testdata/loopback_source.js \
	-query probe -serve 18731 -open -expect "Loopback Comic" -expect-detail "Loopback Comic" \
	-out "$out/search.png" >"$out/search.log" 2>&1 || rc=$?
report "$rc" "a search opens a comic whose detail lists its chapters" "$out/search.log"

# A thread the source answers, in a window, with no network: the comment screen
# is the one part of the source contract a capture tool can drive on its own.

# The chip row follows the number of sections a source declares; both clients
# have to agree on that, so it is a check rather than a screenshot. The fixture
# declares two, because one section is drawn without chips.
run "the browse chips follow the sections" \
	"$out/browse-probe.log" sh scripts/browse-probe.sh
run "a comment thread the source answers" \
	"$out/comments.log" sh scripts/comment-probe.sh

# The chip row follows the number of sections a source declares, and the
# category page has two halves that must agree; both need a source that
# declares them, which the repository fixture does not.
# The category contract has two halves: the parts drawn from `category` and the
# comics from `categoryComics`. Counting both is how a page that draws its
# choosers without loading anything becomes visible.
run "the category page draws and loads" \
	"$out/category-probe.log" sh scripts/category-probe.sh

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
