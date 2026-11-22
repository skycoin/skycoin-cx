#!/usr/bin/env bash
. "${BASH_SOURCE%/*}/fnv.sh"
MEM_N=${MEM_N:-16384}
MEM_WALK=${MEM_WALK:-1024}

h=$FNV_OFFSET
all=0xFFFFFFFF
n=$MEM_N
mem=()

bad=0
sum=0
for (( i = 0; i < n; i++ )); do mem[i]=$i; done
for (( i = 0; i < n; i++ )); do
	(( read = mem[i], read != i && bad++, sum += read ))
done
for (( i = 0; i < n; i++ )); do mem[i]=$(( i ^ all )); done
for (( i = 0; i < n; i++ )); do
	(( read = mem[i], read != (i ^ all) && bad++, sum += read ))
done
fnv "$bad"
fnv "$sum"
total=$bad

for (( bit = 0; bit < 32; bit++ )); do
	pattern=$(( 1 << bit ))
	bad=0
	sum=0
	for (( i = 0; i < MEM_WALK; i++ )); do mem[i]=$pattern; done
	for (( i = 0; i < MEM_WALK; i++ )); do
		(( read = mem[i], read != pattern && bad++, sum += read ))
	done
	fnv "$bad"
	fnv "$sum"
	(( total += bad ))
done

for pattern in 0x00000000 0xFFFFFFFF 0x55555555 0xAAAAAAAA; do
	pattern=$(( pattern ))
	inverse=$(( pattern ^ all ))
	bad=0
	sum=0
	for (( i = 0; i < n; i++ )); do mem[i]=$pattern; done
	for (( i = 0; i < n; i++ )); do
		(( read = mem[i], read != pattern && bad++, sum += read ))
		mem[i]=$inverse
	done
	for (( i = n - 1; i >= 0; i-- )); do
		(( read = mem[i], read != inverse && bad++, sum += read ))
		mem[i]=$pattern
	done
	for (( i = 0; i < n; i++ )); do
		(( read = mem[i], read != pattern && bad++, sum += read ))
	done
	fnv "$bad"
	fnv "$sum"
	(( total += bad ))
done
report memtest $(( total == 0 ))
