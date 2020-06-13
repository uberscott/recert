#!/bin/bash


shellcheck()
{
  docker run --rm -v "$PWD":/work --entrypoint shellcheck tool_shellcheck "$@"
}


set -e

cd "$(dirname "$0")"

# Search ./* removes hidden folders from find results.
# Upside: easy way to not search /.git
# Downside: other hidden folders in folder from which this is executed
# are skipped too.
find ./* -type f | while read -r SCRIPT; do
  file "$SCRIPT" | grep -q 'shell script text' || continue
  echo "=== $SCRIPT ==="
  shellcheck -s bash -x "$SCRIPT"
done
