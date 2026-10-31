#!/bin/sh
# Cookie partitions are named in one place.
#
#   sh scripts/check-partitions.sh
#
# A backup restore once wrote cookies under the bare source key while the transport
# reads "comic:<key>", so the cookies were stored where nothing would ever send
# them — and the tests passed, because they were written from the same wrong
# assumption. The name now lives in core.ComicPartition; this refuses a second one.
set -eu

cd "$(dirname "$0")/.."

allowed='internal/core/core.go'

hits=$(grep -rn '"comic:"' --include='*.go' internal main.go 2>/dev/null | grep -v "^$allowed:" | grep -v '_test.go:' || true)
if [ -n "$hits" ]; then
	printf 'check-partitions: the partition prefix is written outside core.ComicPartition:\n%s\n' "$hits" >&2
	exit 1
fi

# The helper has to be the one that spells it, or this check protects nothing.
if ! grep -q 'return "comic:" + key' "$allowed"; then
	printf 'check-partitions: core.ComicPartition no longer builds "comic:<key>"\n' >&2
	exit 1
fi

printf 'check-partitions: ok\n'
