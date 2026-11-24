#!/bin/sh
# Checks the no JSON in core rule of skycoin-cx#509. Core is every package of
# the module, its tests and every build tag included, except vendor and the
# t0-cx-tools-* developer tools, which no CX program links.
found=$(find . -path ./vendor -prune -o -path ./.git -prune -o -path ./.claude -prune -o -path './t0-cx-tools-*' -prune -o -name '*.go' -print |
	xargs grep -l '"encoding/json"' | sed 's#^\./##' | sort)
for f in $found; do echo "encoding/json imported in core, use the CX text form of t0-textsections instead: $f"; done
[ -z "$found" ]
