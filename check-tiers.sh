#!/bin/sh
# Checks the layout rule in README.md. Every package is t<N>-<name>, imports
# only lower tiers (tests and every build tag set included), N is 1 + the
# highest tier it imports, and no .go file is deeper than <dir>/<file>.go.
M=$(go list -m)
E=$(mktemp)
for tags in "" cxfx ptr64 "cxfx,android" "cxfx,mobile" "os,mobile" "nocipher,nocxos,noregexp" cxcore "cxcore,cxcorelibrarynet"; do
	go list -e -tags "$tags" -json ./... | jq -r --arg m "$M" '.ImportPath as $p | ((.Imports//[])+(.TestImports//[])+(.XTestImports//[]))[] | select(startswith($m+"/")) | select(. != $p) | "\($p) \(.)"'
done | sed "s#$M/##g" | sort -u >"$E"
go list -e ./... | sed "s#^$M/##" | awk -v E="$E" '
	function tier(p) { if (p !~ /^t[0-9]+-/) return -1; return substr(p, 2, index(p, "-") - 2) + 0 }
	{ pk[$0] = 1; mx[$0] = -1; if (tier($0) < 0) { print "untiered package: " $0; bad = 1 } }
	END {
		while ((getline l < E) > 0) {
			split(l, a, " "); n++
			if (tier(a[1]) <= tier(a[2])) { print "import of an equal or higher tier: " l; bad = 1 }
			if (tier(a[2]) > mx[a[1]]) mx[a[1]] = tier(a[2])
		}
		for (p in pk) if (tier(p) != mx[p] + 1) { print "tier is not 1 + highest import: " p; bad = 1 }
		printf "%d packages, %d import edges, tiers ok unless listed above\n", length(pk), n
		exit bad
	}'
rc=$?
deep=$(find . -path ./vendor -prune -o -path ./.git -prune -o -path ./.claude -prune -o -name '*.go' -print | awk -F/ 'NF > 3')
[ -n "$deep" ] && { echo "go files deeper than <dir>/<file>.go:"; echo "$deep"; rc=1; }
rm -f "$E"
exit $rc
