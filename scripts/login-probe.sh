#!/bin/sh
# Exercises the browser-login window end to end against a local page.
#
# The window, the WebView and the source's own callbacks all run for real, so this
# needs a display and is not part of scripts/check.sh. Each mode pins one
# behaviour that a screenshot cannot show:
#
#   ok            the cookie reaches the source's partition and onLoginSuccess
#                 sees it through Network.getCookies
#   timeout       a page the source never accepts ends the wait with a deadline
#                 and stores nothing, including cookies the page already set
#   failcallback  a throwing onLoginSuccess surfaces the source's own reason
#                 instead of waiting out the deadline
#
# Usage: sh scripts/login-probe.sh [mode...]
set -eu

cd "$(dirname "$0")/.."
out=build/login-probe
mkdir -p "$out"

bin="$out/loginprobe"
case "$(uname -s)" in
MINGW* | MSYS* | CYGWIN*) bin="$bin.exe" ;;
esac

go build -o "$bin" ./cmd/loginprobe

modes=${*:-"ok timeout failcallback"}
failed=0
for mode in $modes; do
	log="$out/$mode.log"
	if "$bin" -mode "$mode" >"$log" 2>&1; then
		echo "PASS  login-probe $mode  ($log)"
	else
		echo "FAIL  login-probe $mode  ($log)"
		tail -3 "$log" || true
		failed=1
	fi
done
exit $failed
