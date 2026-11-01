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
	log="build/check-$(printf '%s' "$name" | tr ' ' '-').log"
	if "$@" >"$log" 2>&1; then
		printf 'PASS  %s\n' "$name"
	else
		printf 'FAIL  %s (see %s)\n' "$name" "$log"
		sed -n '1,20p' "$log" | sed 's/^/      /'
		status=1
	fi
}

mkdir -p build

# gofmt prints offenders rather than failing, so its output decides.
if ! gofmt -l . >build/check-gofmt.log 2>&1; then
	printf 'FAIL  gofmt (see build/check-gofmt.log)\n'
	status=1
elif grep -v '^build/' build/check-gofmt.log >build/check-gofmt-files.log; then
	printf 'FAIL  gofmt (see build/check-gofmt-files.log)\n'
	sed 's/^/      /' build/check-gofmt-files.log
	status=1
else
	printf 'PASS  gofmt\n'
fi

run 'core pin' sh scripts/check-core-pin.sh
run 'go vet' go vet ./...
# The race detector is on by default: this app loads comments, browses and decodes
# images on goroutines while the interface reads the same state, which is exactly
# what it is for. NK_NO_RACE=1 turns it off where no C toolchain is available.
race="-race"
[ "${NK_NO_RACE:-}" = "1" ] && race=""
run "go test ($( [ -n "$race" ] && echo 'race detector' || echo 'no race detector' ))" 	go test -count=1 $race ./...
run 'docs' sh scripts/check-docs.sh
run 'partitions' sh scripts/check-partitions.sh
run 'capabilities' sh scripts/check-capabilities.sh

exit $status
