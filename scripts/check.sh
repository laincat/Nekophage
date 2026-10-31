#!/bin/sh
# The whole local gate, in the order that fails fastest.
#
#   sh scripts/check.sh
#
# Every step is run and reported; the exit status is non-zero if any failed, so a
# single command answers "is this tree clean?" without a piped validator hiding a
# failure behind tail's status.
set -eu

cd "$(dirname "$0")/.."

status=0
run() {
	name=$1
	shift
	if "$@" >"build/check.log" 2>&1; then
		printf 'PASS  %s\n' "$name"
	else
		printf 'FAIL  %s (see build/check.log)\n' "$name"
		sed -n '1,20p' build/check.log | sed 's/^/      /'
		status=1
	fi
}

mkdir -p build

# gofmt prints offenders rather than failing, so its output decides.
if [ -n "$(gofmt -l . 2>/dev/null | grep -v '^build/' || true)" ]; then
	printf 'FAIL  gofmt\n'
	gofmt -l . | grep -v '^build/' | sed 's/^/      /'
	status=1
else
	printf 'PASS  gofmt\n'
fi

run 'go vet' go vet ./...
run 'go test' go test -count=1 ./...
run 'docs' sh scripts/check-docs.sh
run 'partitions' sh scripts/check-partitions.sh

exit $status
