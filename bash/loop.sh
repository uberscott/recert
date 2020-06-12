#!/bin/bash
set -e
MAKECMD=${1:-build}
for d in */; do
  [ -f "$d/Makefile" ] || continue
  echo "---entering $PWD/$d---"
  (
  cd "$d" || exit 1
  make "$MAKECMD"
  )
done
