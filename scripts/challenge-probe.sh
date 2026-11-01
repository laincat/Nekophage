#!/bin/sh
# Exercises the challenge window against a local page.
#
# Like scripts/login-probe.sh this needs a display, so it is not part of
# scripts/check.sh. Each mode pins one behaviour a screenshot cannot show:
#
#   clear    a page that passes itself hands its clearance cookie to the source's
#            partition, together with the exact User-Agent the page used — sites
#            bind the two together, so a wrong one makes the next request fail
#   dismiss  a page that never passes ends with the wait being given up on and
#            stores nothing
#
# Usage: sh scripts/challenge-probe.sh [mode...]
set -eu

cd "$(dirname "$0")/.."
out=build/challenge-probe
mkdir -p "$out"

bin="$out/challengeprobe"
case "$(uname -s)" in
MINGW* | MSYS* | CYGWIN*) bin="$bin.exe" ;;
esac

go build -o "$bin" ./cmd/challengeprobe

modes=${*:-"clear dismiss"}
failed=0
for mode in $modes; do
	log="$out/$mode.log"
	if "$bin" -mode "$mode" >"$log" 2>&1; then
		echo "PASS  challenge-probe $mode  ($log)"
	else
		echo "FAIL  challenge-probe $mode  ($log)"
		tail -3 "$log" || true
		failed=1
	fi
done
exit $failed
