# Benchmarks

`compare/` holds twelve long benchmarks. Each has a CX version in
`compare/cx/` and a pure Go version in this directory, the command
`t0-cx-tools-benchmarks`, which runs one benchmark by name. The first six
also have Python 3 and Bash versions. Every version runs the same algorithm
at the same size, folds every result into a 64 bit FNV-1a hash, one 64 bit
word per step, prints it and prints `pass` when the results match the math.

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
- `ackermann`, `digitalroot`, `factorialiter`, `factorialrec`, `structs` and
  `structserialize` are about one second versions of the programs imported
  from github.com/skycoin/cx-benchmark at a511da9ec17c. `structs` fills and
  reads a struct of two 100000 element arrays. CX has no struct serializer,
  so `structserialize` encodes a point and a struct of 16 i32 into a little
  endian byte array and decodes them back by hand. These six have no Python
  version, since nothing runs the Python programs, and no Bash version.

Bash runs smaller sizes, the `bash` lines of `compare/expected.txt`.
`TestCompareExpected` derives every hash there from closed forms, math/big
and independent tests. `TestComparePrograms` runs the Go versions against
it. `make bench-compare` builds cx and a cx with `-tags cxopcount`, which
prints the operations each loop ran, and runs `compare/run.sh`. It needs
python3 and taskset, and clang for the native column of
`cx build --target=llvm`.

`../t0-cx-tools-benchmarkserialize` is the 2021 Go program that serializes
the same structs with the encoder of Skycoin.
