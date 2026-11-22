#!/usr/bin/env bash
. "${BASH_SOURCE%/*}/fnv.sh"
SIEVE_N=${SIEVE_N:-200000}
SIEVE_WANT=${SIEVE_WANT:-17984}
MR_N=${MR_N:-30000}
MR_WANT=${MR_WANT:-3245}
LL_ROUNDS=${LL_ROUNDS:-1}

# Functions return in a global, a subshell per call would dominate. Every
# value stays below 2^62, so signed bash arithmetic gives the unsigned result.
powmod() {
	local r=1 a=$(( $1 % $3 )) d=$2 n=$3
	while (( d > 0 )); do
		(( (d & 1) == 1 && (r = r * a % n) ))
		(( a = a * a % n, d >>= 1 ))
	done
	powmod_out=$r
}

witness() {
	local a=$1 d=$2 s=$3 n=$4 x r
	witness_out=0
	if (( a % n != 0 )); then
		powmod "$a" "$d" "$n"
		x=$powmod_out
		if (( x != 1 && x != n - 1 )); then
			witness_out=1
			for (( r = 1; r < s && witness_out; r++ )); do
				(( x = x * x % n ))
				(( x == n - 1 && (witness_out = 0) ))
			done
		fi
	fi
}

miller_rabin() {
	local n=$1 d s
	if (( n < 2 )); then
		mr_out=0
	elif (( n % 2 == 0 )); then
		mr_out=$(( n == 2 ))
	else
		(( d = n - 1, s = 0 ))
		while (( d % 2 == 0 )); do (( d /= 2, s++ )); done
		mr_out=0
		witness 2 "$d" "$s" "$n"; (( witness_out )) && return
		witness 7 "$d" "$s" "$n"; (( witness_out )) && return
		witness 61 "$d" "$s" "$n"; (( witness_out )) && return
		mr_out=1
	fi
}

mulmod() {
	local r=0 a=$(( $1 % $3 )) b=$2 m=$3
	while (( b > 0 )); do
		if (( (b & 1) == 1 )); then
			(( r += a, r >= m && (r -= m) ))
		fi
		(( a += a, a >= m && (a -= m), b >>= 1 ))
	done
	mulmod_out=$r
}

lucas_lehmer() {
	local p=$1 m s=4 i
	(( m = (1 << p) - 1 ))
	for (( i = 0; i < p - 2; i++ )); do
		mulmod "$s" "$s" "$m"
		(( s = (mulmod_out + m - 2) % m ))
	done
	ll_out=$(( s == 0 ))
}

h=$FNV_OFFSET
ok=1

sieve=()
for (( i = 2; i < SIEVE_N; i++ )); do sieve[i]=1; done
for (( i = 2; i * i < SIEVE_N; i++ )); do
	if (( sieve[i] == 1 )); then
		for (( j = i * i; j < SIEVE_N; j += i )); do sieve[j]=0; done
	fi
done
count=0
for (( i = 2; i < SIEVE_N; i++ )); do (( count += sieve[i] )); done
fnv "$count"
(( ok = ok && count == SIEVE_WANT ))

mr_count=0
for (( k = 0; k < MR_N; k++ )); do
	miller_rabin "$k"
	(( mr_count += mr_out ))
done
fnv "$mr_count"
(( ok = ok && mr_count == MR_WANT ))

exponents=(3 5 7 11 13 17 19 23 29 31 37 41 43 47 53 59 61)
for (( round = 0; round < LL_ROUNDS; round++ )); do
	mask=0
	for p in "${exponents[@]}"; do
		lucas_lehmer "$p"
		if (( ll_out )); then
			(( mask |= 1 << p ))
			fnv 1
		else
			fnv 0
		fi
	done
	(( ok = ok && mask == 2305843011361841320 ))
done
report primes "$ok"
