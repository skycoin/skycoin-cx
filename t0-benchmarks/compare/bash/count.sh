#!/usr/bin/env bash
. "${BASH_SOURCE%/*}/fnv.sh"
COUNT_N=${COUNT_N:-300000}

sum=0
for (( i = 0; i < COUNT_N; i++ )); do
	(( sum += i ))
done
h=$FNV_OFFSET
fnv "$sum"
report count $(( sum == COUNT_N * (COUNT_N - 1) / 2 ))
