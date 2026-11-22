#!/usr/bin/env bash
. "${BASH_SOURCE%/*}/fnv.sh"
FIB_N=${FIB_N:-22}
FIB_WANT=${FIB_WANT:-17711}

# fib returns its result in fib_out, a subshell per call would dominate.
fib() {
	if (( $1 < 2 )); then
		fib_out=$1
	else
		fib $(( $1 - 1 ))
		local left=$fib_out
		fib $(( $1 - 2 ))
		fib_out=$(( left + fib_out ))
	fi
}

fib "$FIB_N"
h=$FNV_OFFSET
fnv "$fib_out"
report fib $(( fib_out == FIB_WANT ))
