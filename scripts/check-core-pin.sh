#!/bin/sh
# Check the actual module selected by go.work/replace, not just its label.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
pin=$(tr -d ' \r\n' < scripts/core-version)
core=$(go list -m -f '{{.Dir}}' github.com/laincat/veneracat-core)
head=$(git -C "$core" rev-parse HEAD)
if [ "$head" != "$pin" ]; then
    printf 'check-core-pin: linked Core is %s, expected %s\n' "$head" "$pin" >&2
    printf 'check-core-pin: run sh scripts/fetch-core.sh\n' >&2
    exit 1
fi
if [ -n "$(git -C "$core" status --porcelain)" ]; then
    printf 'check-core-pin: linked Core has uncommitted changes: %s\n' "$core" >&2
    exit 1
fi
stamp=$(sed -n 's/.*SetRelease("\([0-9a-f]*\)").*/\1/p' corestamp.go)
if [ "$stamp" != "$head" ]; then
    printf 'check-core-pin: generated stamp differs from linked Core; run sh scripts/fetch-core.sh\n' >&2
    exit 1
fi
printf 'check-core-pin: ok (%s)\n' "$head"
