// Command t0-cx-tools-benchmarks runs one compare/ benchmark in Go, the
// reference the CX programs are timed against. Usage: t0-cx-tools-benchmarks
// name. It prints the FNV-1a hash of the results and the pass line, as the
// CX, Python and Bash programs do. Without a name it lists the names.
package main

import (
	"fmt"
	"os"
	"sort"
)

const (
	compareFnvOffset = 14695981039346656037
	compareFnvPrime  = 1099511628211
)

func compareFnv(hash uint64, value uint64) uint64 {
	return (hash ^ value) * compareFnvPrime
}

// comparePrograms maps each benchmark name to its Go version, which returns
// the hash and whether every result matched the math.
var comparePrograms = map[string]func() (uint64, bool){
	"ackermann":       compareAckermann,
	"array":           compareArray,
	"count":           compareCount,
	"digitalroot":     compareDigitalRoot,
	"factorialiter":   compareFactorialIter,
	"factorialrec":    compareFactorialRec,
	"fib":             compareFib,
	"memtest":         compareMemtest,
	"primes":          comparePrimes,
	"struct":          compareStruct,
	"structs":         compareStructs,
	"structserialize": compareStructSerialize,
}

func main() {
	if len(os.Args) == 1 {
		var names []string
		for name := range comparePrograms {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Println(name)
		}
		return
	}
	program, found := comparePrograms[os.Args[1]]
	if len(os.Args) != 2 || !found {
		fmt.Fprintln(os.Stderr, "usage: t0-cx-tools-benchmarks [name]")
		os.Exit(2)
	}
	hash, pass := program()
	fmt.Println(hash)
	if !pass {
		fmt.Println(os.Args[1], "FAIL")
		os.Exit(1)
	}
	fmt.Println(os.Args[1], "pass")
}
