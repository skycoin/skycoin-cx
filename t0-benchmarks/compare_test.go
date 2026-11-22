package benchmarks

import (
	"bufio"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
)

const (
	compareFnvOffset = 14695981039346656037
	compareFnvPrime  = 1099511628211
)

func compareFnv(hash uint64, value uint64) uint64 {
	return (hash ^ value) * compareFnvPrime
}

// compareScale holds the sizes of one scale of the compare/ programs.
type compareScale struct {
	countN       uint64
	fibN         int
	arrayRounds  int
	structRounds int
	memN         uint64
	memWalk      uint64
	sieveN       int64
	millerN      int64
	lucasRounds  int
}

var compareScales = map[string]compareScale{
	"full": {30000000, 34, 400, 80000, 1048576, 65536, 2000000, 300000, 20},
	"bash": {300000, 22, 4, 800, 16384, 1024, 200000, 30000, 1},
}

// comparePrimeCount counts the primes below n. Below 2^20 it uses math/big's
// Baillie-PSW test, which is exact below 2^64, and above that a sieve, so
// the Miller-Rabin counts are checked against another test.
func comparePrimeCount(n int64) uint64 {
	var count uint64
	if n < 1<<20 {
		for k := int64(2); k < n; k++ {
			if big.NewInt(k).ProbablyPrime(0) {
				count++
			}
		}
		return count
	}
	composite := make([]bool, n)
	for k := int64(2); k < n; k++ {
		if composite[k] {
			continue
		}
		count++
		for multiple := k * k; multiple < n; multiple += k {
			composite[multiple] = true
		}
	}
	return count
}

// compareExpected derives the hash each program must print from the math.
func compareExpected(name string, scale compareScale) uint64 {
	hash := uint64(compareFnvOffset)
	switch name {
	case "count":
		hash = compareFnv(hash, scale.countN*(scale.countN-1)/2)
	case "fib":
		previous, current := uint64(0), uint64(1)
		for i := 0; i < scale.fibN; i++ {
			previous, current = current, previous+current
		}
		hash = compareFnv(hash, previous)
	case "array":
		// A reversed array of 150 has 150*149/2 inversions, one swap each.
		for round := 0; round < scale.arrayRounds; round++ {
			hash = compareFnv(hash, 150*149/2)
		}
		for value := uint64(1); value <= 150; value++ {
			hash = compareFnv(hash, value)
		}
	case "struct":
		for round := 0; round < scale.structRounds; round++ {
			hash = compareFnv(hash, 3*99*100/2)
		}
	case "memtest":
		const all = 0xFFFFFFFF
		// i and all-i read back once each.
		hash = compareFnv(compareFnv(hash, 0), scale.memN*all)
		for bit := 0; bit < 32; bit++ {
			hash = compareFnv(compareFnv(hash, 0), scale.memWalk<<bit)
		}
		// pattern, its inverse, then pattern again.
		for _, pattern := range []uint64{0, all, 0x55555555, 0xAAAAAAAA} {
			hash = compareFnv(compareFnv(hash, 0), scale.memN*(pattern+all))
		}
	case "primes":
		hash = compareFnv(hash, comparePrimeCount(scale.sieveN))
		hash = compareFnv(hash, comparePrimeCount(scale.millerN))
		// The Mersenne prime exponents up to 61, OEIS A000043.
		mersenne := map[uint64]bool{3: true, 5: true, 7: true, 13: true, 17: true, 19: true, 31: true, 61: true}
		for round := 0; round < scale.lucasRounds; round++ {
			for _, exponent := range []uint64{3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61} {
				flag := uint64(0)
				if mersenne[exponent] {
					flag = 1
				}
				hash = compareFnv(hash, flag)
			}
		}
	}
	return hash
}

// TestCompareExpected checks compare/expected.txt, the hashes the
// compare/ programs print, against the math.
func TestCompareExpected(t *testing.T) {
	file, err := os.Open("compare/expected.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	lines := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		lines++
		scale, found := compareScales[fields[1]]
		if len(fields) != 3 || !found {
			t.Fatalf("bad line %q", scanner.Text())
		}
		listed, err := strconv.ParseUint(fields[2], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		if want := compareExpected(fields[0], scale); listed != want {
			t.Errorf("%s %s: expected.txt has %d, the math gives %d", fields[0], fields[1], listed, want)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if lines != 12 {
		t.Errorf("%d hashes in expected.txt, want 12", lines)
	}
}
