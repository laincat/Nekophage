#!/bin/sh
# Check the actual module selected by go.work/replace, not just its label.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
core=$(go list -m -f '{{.Dir}}' github.com/laincat/veneracat-core)
head=$(git -C "$core" rev-parse HEAD)
if [ -n "$(git -C "$core" status --porcelain)" ]; then
    printf 'check-core: linked Core has uncommitted changes: %s\n' "$core" >&2
    exit 1
fi
stamp=$(sed -n 's/.*SetRelease("\([0-9a-f]*\)").*/\1/p' corestamp.go)
if [ "$stamp" != "$head" ]; then
    printf 'check-core: generated stamp differs from linked Core; run sh scripts/fetch-core.sh\n' >&2
    exit 1
fi
printf 'check-core: ok (%s)\n' "$head"
