#!/usr/bin/env bash
. "${BASH_SOURCE%/*}/fnv.sh"
ARRAY_ROUNDS=${ARRAY_ROUNDS:-4}

h=$FNV_OFFSET
ok=1
nums=()
for (( round = 0; round < ARRAY_ROUNDS; round++ )); do
	for (( i = 0; i < 150; i++ )); do
		nums[i]=$(( 150 - i ))
	done
	swaps=0
	for (( i = 0; i < 149; i++ )); do
		for (( j = 0; j < 149 - i; j++ )); do
			if (( nums[j] > nums[j + 1] )); then
				t=${nums[j]}
				nums[j]=${nums[j + 1]}
				nums[j + 1]=$t
				(( swaps++ ))
			fi
		done
	done
	fnv "$swaps"
	(( ok = ok && swaps == 11175 ))
done
for (( i = 0; i < 150; i++ )); do
	fnv "${nums[i]}"
	(( ok = ok && nums[i] == i + 1 ))
done
report array "$ok"
