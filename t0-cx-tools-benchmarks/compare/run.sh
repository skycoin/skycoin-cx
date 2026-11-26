#!/usr/bin/env bash
# Runs every benchmark in cx (old loop, vm and native through the LLVM
# target), Go, Python 3 and Bash, pinned
# to one core. It prints the median of RUNS runs, the same less the median
# start up time of an empty program in that language (net), the CX
# operations per net millisecond, and the net time relative to Go.
# A run that prints another hash or no pass line is listed as wrong. A
# benchmark with no Python or Bash version skips that language.
#
# CX is the cx binary and CXOPCOUNT one built with -tags cxopcount. The Go
# versions are the t0-cx-tools-benchmarks command. Bash runs the smaller bash
# scale of expected.txt. Its net time is multiplied by the ratio of the vm
# operation counts of the two scales to estimate full scale.
set -eu
cd "$(dirname "$0")"
RUNS=${RUNS:-5}
CPU=${CPU:-2}
CX=${CX:-../../bin/cx}
CXOPCOUNT=${CXOPCOUNT:-../../bin/cx-opcount}
BENCHES=${BENCHES:-count fib array struct memtest primes ackermann digitalroot factorialiter factorialrec structs structserialize}
LANGS=${LANGS:-cx-old cx-vm cx-native go python bash}
PYTHON=${PYTHON:-python3}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# vm operations at full scale over bash scale, measured with cx-opcount.
declare -A bash_factor=([count]=100.0 [fib]=322.0 [array]=99.7 [struct]=99.9 [memtest]=64.0 [primes]=10.9)

go build -o "$work/benchmarks" ..
printf 'package main\n\nfunc main() {\n}\n' >"$work/empty.cx"
if [[ " $LANGS " == *" cx-native "* ]]; then
	for bench in $BENCHES; do
		$CX build --target=llvm "cx/$bench.cx" -o "$work/$bench-native"
	done
	$CX build --target=llvm "$work/empty.cx" -o "$work/empty-native"
fi

expected() { awk -v b="$1" -v s="$2" '$1 == b && $2 == s { print $3 }' expected.txt; }

# present tells whether a benchmark has a version in a language.
present() {
	case $2 in
	python) [[ -f python/$1.py ]] ;;
	bash) [[ -f bash/$1.sh ]] ;;
	*) true ;;
	esac
}

command_of() {
	case $2 in
	cx-old) echo "$CX --executor=old cx/$1.cx" ;;
	cx-vm) echo "$CX --executor=vm cx/$1.cx" ;;
	cx-native) echo "$work/$1-native" ;;
	go) echo "$work/benchmarks $1" ;;
	python) echo "$PYTHON -I python/$1.py" ;;
	bash) echo "bash bash/$1.sh" ;;
	esac
}

# The Go command lists the benchmark names when given none.
empty_of() {
	case $1 in
	cx-old) echo "$CX --executor=old $work/empty.cx" ;;
	cx-vm) echo "$CX --executor=vm $work/empty.cx" ;;
	cx-native) echo "$work/empty-native" ;;
	go) echo "$work/benchmarks" ;;
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
for lang in $LANGS go; do
	# shellcheck disable=SC2046
	startup[$lang]=$(timed "" $(empty_of "$lang"))
done
printf 'start up ms:'
for lang in "${!startup[@]}"; do printf ' %s %s' "$lang" "${startup[$lang]}"; done
printf '\n\n'

printf '| benchmark | language | scale | median ms | net ms | operations | ops/ms | x Go |\n'
printf '|---|---|---|---:|---:|---:|---:|---:|\n'
for bench in $BENCHES; do
	full=$(expected "$bench" full)
	# shellcheck disable=SC2046
	go_ms=$(timed "$full" $(command_of "$bench" go))
	go_net=$(awk -v m="$go_ms" -v s="${startup[go]}" 'BEGIN { if (m == "wrong") print m; else printf "%.1f", m - s }')
	for lang in $LANGS; do
		present "$bench" "$lang" || continue
		scale=full want=$full ops='' rate=''
		if [[ $lang == bash ]]; then
			scale="bash x${bash_factor[$bench]}" want=$(expected "$bench" bash)
		fi
		if [[ $lang == go ]]; then
			ms=$go_ms
		else
			# shellcheck disable=SC2046
			ms=$(timed "$want" $(command_of "$bench" "$lang"))
		fi
		net=$(awk -v m="$ms" -v s="${startup[$lang]}" 'BEGIN { if (m == "wrong") print m; else printf "%.1f", m - s }')
		if [[ $lang == bash && $net != wrong ]]; then
			net=$(awk -v n="$net" -v f="${bash_factor[$bench]}" 'BEGIN { printf "%.1f", n * f }')
		fi
		if [[ $lang == cx-* && -x $CXOPCOUNT && $net != wrong ]]; then
			executor=${lang#cx-}
			# The native program runs the image vm runs.
			[[ $executor == native ]] && executor=vm
			ops=$(operations "$bench" "$executor")
			rate=$(awk -v o="$ops" -v n="$net" 'BEGIN { if (n < 1) print "-"; else printf "%.0f", o / n }')
		fi
		ratio=$(awk -v n="$net" -v g="$go_net" 'BEGIN { if (n == "wrong" || g == "wrong" || g < 2) print "-"; else printf "%.1f", n / g }')
		printf '| %s | %s | %s | %s | %s | %s | %s | %s |\n' "$bench" "$lang" "$scale" "$ms" "$net" "$ops" "$rate" "$ratio"
	done
done
