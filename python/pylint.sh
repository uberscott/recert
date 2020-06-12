#!/bin/bash
MAKECMD=${1:-build}
for d in */; do
	[ -f "$d/Makefile" ] || continue
	(
	cd "$d" || exit 1
	make "$MAKECMD"
	)
done
