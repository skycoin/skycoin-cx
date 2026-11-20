#!/bin/sh
# Checks the mechanical rules of docs/ObjectModel.md. The generated Gid and
# manager files are up to date, a Def file holds one struct and no function,
# and a Fon or Fof file names no manager, no CxIrState and no Alloc, Release
# or Get of a managed type.
rc=0
go run ./t0-cx-core-objmodelgen -dir t3-ast -check || rc=1
objs=$(sed -n 's/^[[:space:]]*{"\([A-Za-z_]*\)", 0x.*/\1/p' t0-cx-core-objmodelgen/main.go | paste -sd'|' -)
[ -n "$objs" ] || { echo "no object types read from t0-cx-core-objmodelgen/main.go"; exit 1; }
files=$(find . -path ./vendor -prune -o -path ./.git -prune -o -path ./.claude -prune -o -name '*.go' -print | sed 's#^\./##')
for f in $(echo "$files" | grep '_def\.go$'); do
	types=$(grep -c '^type ' "$f")
	funcs=$(grep -c '^func ' "$f")
	[ "$types" -eq 1 ] && [ "$funcs" -eq 0 ] || { echo "$f: a Def file holds one type and no function, has $types types and $funcs functions"; rc=1; }
done
for f in $(echo "$files" | grep -E '_fo[nf]\.go$'); do
	if grep -nE 'Mng|CxIrState|('"$objs"')(Alloc|Release|Get)([^A-Za-z0-9_]|$)' "$f"; then
		echo "$f: a Fon or Fof file names a manager, CxIrState or a manager call"
		rc=1
	fi
done
echo "$(echo "$files" | grep -c '_def\.go$') Def files, $(echo "$files" | grep -cE '_fo[nf]\.go$') Fon and Fof files, object model ok unless listed above"
exit $rc
