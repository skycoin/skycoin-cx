# Sourced by the benchmarks. Bash arithmetic is signed 64 bit and wraps, so
# the FNV-1a product is the unsigned one and printf %u prints it as such.
FNV_OFFSET=-3750763034362895579
FNV_PRIME=1099511628211

fnv() { h=$(( (h ^ $1) * FNV_PRIME )); }

report() {
	printf '%u\n' "$h"
	if (( $2 )); then echo "$1 pass"; else echo "$1 FAIL"; exit 1; fi
}
