#!/bin/sh
# Runs every probe that needs a real window, in one go, and prints a summary.
#
#   sh scripts/desktop-probes.sh
#   NK_OFFICIAL_SRC=/path/to/official-srcs sh scripts/desktop-probes.sh
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

if [ -n "${NK_OFFICIAL_SRC:-}" ]; then
	run "every callback the official sources declare" \
		"$out/callbacks.log" sh scripts/callback-probe.sh
else
	printf 'SKIP  official source callbacks (set NK_OFFICIAL_SRC)\n'
fi

printf 'logs: %s\n' "$out"
exit $failed
