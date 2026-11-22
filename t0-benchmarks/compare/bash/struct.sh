#!/usr/bin/env bash
. "${BASH_SOURCE%/*}/fnv.sh"
STRUCT_ROUNDS=${STRUCT_ROUNDS:-800}

# Bash has no structs, so a point is the pair x[i], y[i].
h=$FNV_OFFSET
ok=1
x=()
y=()
for (( i = 0; i < 100; i++ )); do
	x[i]=$i
	y[i]=$(( 2 * i ))
done
for (( round = 0; round < STRUCT_ROUNDS; round++ )); do
	sum=0
	for (( i = 0; i < 100; i++ )); do
		(( sum = sum + x[i] + y[i] ))
	done
	fnv "$sum"
	(( ok = ok && sum == 14850 ))
done
report struct "$ok"
