#!/bin/sh
# Runs every probe that needs a real window, in one go, and prints a summary.
#
#   sh scripts/desktop-probes.sh
#   NK_OFFICIAL_SRC=/path/to/official-srcs sh scripts/desktop-probes.sh
#   NK_PROBE_SRC=/path/to/source.js sh scripts/desktop-probes.sh
#   NK_READ_SRC=/path/to/source.js sh scripts/desktop-probes.sh
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

go build -o "$out/dialogprobe" ./cmd/dialogprobe
run "closing a source dialog releases its caller" \
	"$out/dialog.log" "$out/dialogprobe"

go build -o "$out/updateprobe" ./cmd/updateprobe
run "a closed window still takes updates and page reads" \
	"$out/update.log" "$out/updateprobe"

# Pages that actually decode, which is the one thing about images that cannot be
# answered without a decoder: a source can return plausible byte counts that are
# not a picture.
if [ -n "${NK_READ_SRC:-}" ]; then
	run "the reader decodes a real chapter" 		"$out/reader-probe.log" sh scripts/reader-probe.sh
else
	printf 'SKIP  reader decode (set NK_READ_SRC to a source script)
'
fi

# A thread the source answers, in a window, with no network: the comment screen is
# the one part of the source contract a capture tool can drive on its own.
if [ -n "${NK_PROBE_SRC:-}" ]; then
	run "a comment thread the source answers" 		"$out/comments.log" sh scripts/comment-probe.sh "$NK_PROBE_SRC"
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
