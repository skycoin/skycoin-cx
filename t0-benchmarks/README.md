# Benchmarks

Imported from github.com/skycoin/cx-benchmark at a511da9ec17c.

`cx/` holds the CX programs. The Go files here hold the same algorithms in Go
for comparison. Run `make` here, or `make cx` and `make go`.

`bm-structs.cx` and `bm-struct-serialization.cx` use pre 2022 syntax and fail
to parse with the current compiler, the same as with skycoin/cx develop.
`../t0-benchmarks-structser` is the Go side of the struct serialization
benchmark.

## compare

`compare/` holds six long benchmarks, each in CX, C, Python 3 and Bash with
the same algorithm and sizes. Each program folds every result into a 64 bit
FNV-1a hash, one 64 bit word per step, prints it and prints `pass` when the
results match the math.

- `primes` counts the primes below 2,000,000 with a sieve (148933), the
  primes below 300,000 with deterministic Miller-Rabin, bases 2, 7 and 61
  (25997), and runs the Lucas-Lehmer test 20 times for the odd prime
  exponents 3 to 61. The square mod 2^p - 1 is done by doubling, so every
  value stays below 2^62 and fits a ui64. 61 is the largest Mersenne prime
  exponent that does.
- `memtest` runs address in address, walking ones and moving inversions over
  a 4 MiB ui32 array and folds the mismatches (0) and the sum of the words
  read.
- `count`, `fib`, `array` and `struct` are about one second versions of the
  BenchmarkRunVm programs.

Bash runs smaller sizes, the `bash` lines of `compare/expected.txt`.
`TestCompareExpected` derives every hash there from closed forms and math/big's
prime test. `make bench-compare` builds cx and a cx with `-tags cxopcount`,
which prints the operations each loop ran, and runs `compare/run.sh`. It
needs clang, python3 and taskset.
