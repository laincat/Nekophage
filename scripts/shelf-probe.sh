#!/bin/sh
# Opens a chapter, then checks the shelf file for the visit.
#
#   sh scripts/shelf-probe.sh /path/to/source.js [query]
#
# This is the reading half of the shelf: the reader writes progress and the shelf
# reads it back, which a unit test on the store cannot show. It needs a chapter
# with real pages, and the repository's own fixture points at loopback, so this
# serves that page itself — no site, no account, no network.
#
# The check reads `shelf.json` rather than the screen: a capture of an empty shelf
# and a shelf with one row are nearly the same size, and the file says exactly
# which comic, which chapter and which page.
set -eu

cd "$(dirname "$0")/.."
src=${1:-}
query=${2:-probe}
if [ -z "$src" ] || [ ! -f "$src" ]; then
	printf 'shelf-probe: pass a source script whose chapter has pages\n' >&2
	exit 2
fi

out=build/probes
mkdir -p "$out"
profile=$out/shelf-profile
rm -rf "$profile"
mkdir -p "$profile"
port=18731

log=$out/shelf.log
shot=$out/shelf.png

# One run opens the chapter; the profile keeps the shelf between runs.
go run ./cmd/uicheck -dir "$profile" -serve "$port" -script "$src" -query "$query" \
	-open -chapter 0 -out "$out/shelf-reader.png" >"$log" 2>&1

shelf=$profile/shelf.json
if [ ! -f "$shelf" ]; then
	printf 'shelf-probe: reading wrote no shelf.json; see %s\n' "$log" >&2
	exit 1
fi

lesson=$(python3 - "$shelf" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    data = json.load(f)
rows = data.get("history") or []
if not rows:
    print("the shelf records nothing")
    raise SystemExit(1)
row = rows[0]
missing = [k for k in ("title", "chapter", "page") if not row.get(k)]
if missing:
    # No page means the chapter served no images, which is this probe's input being
    # wrong rather than the client: say which, so the two are not confused.
    if missing == ["page"]:
        print("the shelf row has no page: the chapter served no images, "
              "and this probe needs a chapter with pages")
    else:
        print("the shelf row is missing %s" % ", ".join(missing))
    raise SystemExit(1)
print("shelf row: %s / %s / page %s" % (row["title"], row["chapter"], row["page"]))
PY
) || { printf 'shelf-probe: %s\n' "$lesson" >&2; exit 1; }

# A capture for a person to look at; the verdict above did not depend on it.
go run ./cmd/uicheck -dir "$profile" -serve "$port" -script "$src" -query "$query" \
	-shelf -out "$shot" >"$out/shelf-view.log" 2>&1 || true

printf 'shelf-probe: %s\n' "$lesson"
printf 'shelf-probe: capture %s\n' "$shot"
