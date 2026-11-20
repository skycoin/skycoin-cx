#!/bin/sh
# Checks the no recovery rule in docs/ErrorHandling.md. No recover( call
# outside vendor, except the path:count lines in recover-allowlist.txt, which
# must only shrink.
export LC_ALL=C
A=recover-allowlist.txt
F=$(mktemp)
find . -path ./vendor -prune -o -path ./.git -prune -o -path ./.claude -prune -o -name '*.go' -print |
	xargs grep -c -E '(^|[^A-Za-z0-9_.])recover\(' | grep -v ':0$' | sed 's#^\./##' | sort >"$F"
rc=0
comm -23 "$F" "$A" | while read -r l; do echo "recover( call, crash with an assert or return an error struct instead: $l"; done
comm -13 "$F" "$A" | while read -r l; do echo "changed or fixed, update $A: $l"; done
[ -n "$(comm -3 "$F" "$A")" ] && rc=1
echo "$(wc -l <"$F") allowlisted files with recover("
rm -f "$F"
exit $rc
