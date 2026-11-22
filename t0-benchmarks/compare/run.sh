#!/usr/bin/env bash
# Runs every benchmark in cx (old loop and vm), C, Python 3 and Bash, pinned
# to one core. It prints the median of RUNS runs, the same less the median
# start up time of an empty program in that language (net), the CX
# operations per net millisecond, and the net time relative to C.
# A run that prints another hash or no pass line is listed as wrong.
#
# CX is the cx binary and CXOPCOUNT one built with -tags cxopcount. Bash runs
# the smaller bash scale of expected.txt. Its net time is multiplied by the
# ratio of the vm operation counts of the two scales to estimate full scale.
set -eu
cd "$(dirname "$0")"
RUNS=${RUNS:-5}
CPU=${CPU:-2}
CX=${CX:-../../bin/cx}
CXOPCOUNT=${CXOPCOUNT:-../../bin/cx-opcount}
BENCHES=${BENCHES:-count fib array struct memtest primes}
LANGS=${LANGS:-cx-old cx-vm c python bash}
CC=${CC:-clang}
PYTHON=${PYTHON:-python3}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# vm operations at full scale over bash scale, measured with cx-opcount.
declare -A bash_factor=([count]=100.0 [fib]=322.0 [array]=99.7 [struct]=99.9 [memtest]=64.0 [primes]=10.9)

for bench in $BENCHES; do
	$CC -O2 -o "$work/$bench" "c/$bench.c"
done
printf 'int main(void) { return 0; }\n' >"$work/empty.c"
$CC -O2 -o "$work/empty" "$work/empty.c"
printf 'package main\n\nfunc main() {\n}\n' >"$work/empty.cx"

expected() { awk -v b="$1" -v s="$2" '$1 == b && $2 == s { print $3 }' expected.txt; }

command_of() {
	case $2 in
	cx-old) echo "$CX --executor=old cx/$1.cx" ;;
	cx-vm) echo "$CX --executor=vm cx/$1.cx" ;;
	c) echo "$work/$1" ;;
	python) echo "$PYTHON -I python/$1.py" ;;
	bash) echo "bash bash/$1.sh" ;;
	esac
}

empty_of() {
	case $1 in
	cx-old) echo "$CX --executor=old $work/empty.cx" ;;
	cx-vm) echo "$CX --executor=vm $work/empty.cx" ;;
	c) echo "$work/empty" ;;
	python) echo "$PYTHON -I -c pass" ;;
	bash) echo "bash -c :" ;;
	esac
}

median() { sort -n | awk '{ v[NR] = $1 } END { print v[int((NR + 1) / 2)] }'; }

# timed runs a command RUNS times and prints the median milliseconds. With a
# hash it prints "wrong" when a run prints another hash or no pass line.
timed() {
	local want=$1 times=() out start end
	shift
	for (( run = 0; run < RUNS; run++ )); do
		start=$EPOCHREALTIME
		out=$(taskset -c "$CPU" "$@" 2>/dev/null) || true
		end=$EPOCHREALTIME
		if [[ -n $want && ($(sed -n 1p <<<"$out") != "$want" || $(sed -n 2p <<<"$out") != *" pass") ]]; then
			echo wrong
			return
		fi
		times+=("$(awk -v s="$start" -v e="$end" 'BEGIN { printf "%.1f", (e - s) * 1000 }')")
	done
	printf '%s\n' "${times[@]}" | median
}

operations() {
	"$CXOPCOUNT" --executor="$2" "cx/$1.cx" 2>&1 >/dev/null | awk -v e="$2" '$1 == "cxopcount" { print (e == "old") ? $3 : $5 }'
}

declare -A startup
for lang in $LANGS c; do
	# shellcheck disable=SC2046
	startup[$lang]=$(timed "" $(empty_of "$lang"))
done
printf 'start up ms:'
for lang in "${!startup[@]}"; do printf ' %s %s' "$lang" "${startup[$lang]}"; done
printf '\n\n'

printf '| benchmark | language | scale | median ms | net ms | operations | ops/ms | x C |\n'
printf '|---|---|---|---:|---:|---:|---:|---:|\n'
for bench in $BENCHES; do
	full=$(expected "$bench" full)
	# shellcheck disable=SC2046
	c_ms=$(timed "$full" $(command_of "$bench" c))
	c_net=$(awk -v m="$c_ms" -v s="${startup[c]}" 'BEGIN { if (m == "wrong") print m; else printf "%.1f", m - s }')
	for lang in $LANGS; do
		scale=full want=$full ops= rate=
		if [[ $lang == bash ]]; then
			scale="bash x${bash_factor[$bench]}" want=$(expected "$bench" bash)
		fi
		if [[ $lang == c ]]; then
			ms=$c_ms
		else
			# shellcheck disable=SC2046
			ms=$(timed "$want" $(command_of "$bench" "$lang"))
		fi
		net=$(awk -v m="$ms" -v s="${startup[$lang]}" 'BEGIN { if (m == "wrong") print m; else printf "%.1f", m - s }')
		if [[ $lang == bash && $net != wrong ]]; then
			net=$(awk -v n="$net" -v f="${bash_factor[$bench]}" 'BEGIN { printf "%.1f", n * f }')
		fi
		if [[ $lang == cx-* && -x $CXOPCOUNT && $net != wrong ]]; then
			ops=$(operations "$bench" "${lang#cx-}")
			rate=$(awk -v o="$ops" -v n="$net" 'BEGIN { printf "%.0f", o / n }')
		fi
		ratio=$(awk -v n="$net" -v c="$c_net" 'BEGIN { if (n == "wrong" || c == "wrong" || c < 2) print "-"; else printf "%.0f", n / c }')
		printf '| %s | %s | %s | %s | %s | %s | %s | %s |\n' "$bench" "$lang" "$scale" "$ms" "$net" "$ops" "$rate" "$ratio"
	done
done
