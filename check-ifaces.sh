#!/bin/sh
# Checks the interface rule in README.md. No interface embeds another
# interface, except the lines in iface-allowlist.txt, which must only shrink.
export LC_ALL=C
A=iface-allowlist.txt
F=$(mktemp)
go run ./t0-ifacecheck >"$F" || { rm -f "$F"; exit 1; }
rc=0
comm -23 "$F" "$A" | while read -r l; do echo "interface embeds an interface, list its methods instead: $l"; done
comm -13 "$F" "$A" | while read -r l; do echo "fixed, remove from $A: $l"; done
[ -n "$(comm -3 "$F" "$A")" ] && rc=1
echo "$(wc -l <"$F") allowlisted interface embeddings"
rm -f "$F"
exit $rc
