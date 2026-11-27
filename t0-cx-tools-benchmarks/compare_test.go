package main

import (
	"bufio"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// compareScale holds the sizes of one scale of the compare/ programs. The
// bash scale leaves the sizes of the programs that have no Bash version 0.
type compareScale struct {
	countN                uint64
	fibN                  int
	arrayRounds           int
	structRounds          int
	memN                  uint64
	memWalk               uint64
	sieveN                int64
	millerN               int64
	lucasRounds           int
	ackermannRounds       int
	digitalRootN          int
	factorialIterRounds   int
	factorialRecRounds    int
	structsRounds         int
	structSerializeRounds int
}

var compareScales = map[string]compareScale{
	"full": {30000000, 34, 400, 80000, 1048576, 65536, 2000000, 300000, 20, 350, 1200000, 120000, 50000, 60, 76000},
	"bash": {300000, 22, 4, 800, 16384, 1024, 200000, 30000, 1, 0, 0, 0, 0, 0, 0},
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

// comparePersistence counts the digit sums that take n to one digit, summing
// the digits of its decimal text rather than dividing.
func comparePersistence(n int) uint64 {
	var steps uint64
	for text := strconv.Itoa(n); len(text) > 1; steps++ {
		sum := 0
		for _, digit := range text {
			sum += int(digit - '0')
		}
		text = strconv.Itoa(sum)
	}
	return steps
}

// compareFactorialsFold folds n! for n 0 to 20, from math/big, rounds times.
func compareFactorialsFold(hash uint64, rounds int) uint64 {
	var factorials [21]uint64
	for n := int64(0); n <= 20; n++ {
		factorials[n] = new(big.Int).MulRange(1, n).Uint64()
	}
	for round := 0; round < rounds; round++ {
		for _, factorial := range factorials {
			hash = compareFnv(hash, factorial)
		}
	}
	return hash
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
	case "ackermann":
		// Ackermann(3, n) is 2^(n+3) - 3.
		for round := 0; round < scale.ackermannRounds; round++ {
			hash = compareFnv(hash, 1<<8-3)
		}
	case "digitalroot":
		// The digital root of n > 0 is 1 + (n - 1) mod 9.
		for n := 1; n < scale.digitalRootN; n++ {
			hash = compareFnv(hash, uint64(1+(n-1)%9)+10*comparePersistence(n))
		}
	case "factorialiter":
		hash = compareFactorialsFold(hash, scale.factorialIterRounds)
	case "factorialrec":
		hash = compareFactorialsFold(hash, scale.factorialRecRounds)
	case "structs":
		// The sum over c below 100000 of c + round + 3c.
		for round := uint64(0); round < uint64(scale.structsRounds); round++ {
			hash = compareFnv(hash, 2*100000*99999+100000*round)
		}
	case "structserialize":
		// round + 2 round + the sum over k below 16 of 16 round + k.
		for round := uint64(0); round < uint64(scale.structSerializeRounds); round++ {
			hash = compareFnv(hash, 259*round+120)
		}
	}
	return hash
}

// compareExpectedRead returns the hashes of compare/expected.txt by
// benchmark and scale.
func compareExpectedRead(t testing.TB) map[[2]string]uint64 {
	file, err := os.Open("compare/expected.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	hashes := map[[2]string]uint64{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		if _, found := compareScales[fields[1]]; len(fields) != 3 || !found {
			t.Fatalf("bad line %q", scanner.Text())
		}
		listed, err := strconv.ParseUint(fields[2], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		hashes[[2]string{fields[0], fields[1]}] = listed
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return hashes
}

// TestCompareExpected checks compare/expected.txt, the hashes the
// compare/ programs print, against the math.
func TestCompareExpected(t *testing.T) {
	hashes := compareExpectedRead(t)
	for key, listed := range hashes {
		if want := compareExpected(key[0], compareScales[key[1]]); listed != want {
			t.Errorf("%s %s: expected.txt has %d, the math gives %d", key[0], key[1], listed, want)
		}
	}
	if len(hashes) != 18 {
		t.Errorf("%d hashes in expected.txt, want 18", len(hashes))
	}
}

// TestComparePrograms runs the Go version of every compare/ benchmark and
// checks its hash against the full line of compare/expected.txt.
func TestComparePrograms(t *testing.T) {
	if testing.Short() {
		t.Skip("runs every benchmark at full scale")
	}
	hashes := compareExpectedRead(t)
	for name, program := range comparePrograms {
		want, found := hashes[[2]string{name, "full"}]
		if !found {
			t.Errorf("%s: no full line in expected.txt", name)
			continue
		}
		if hash, pass := program(); hash != want || !pass {
			t.Errorf("%s: hash %d pass %v, want %d and pass", name, hash, pass, want)
		}
	}
}

// TestCompareAckermannFunction checks small values against the closed forms
// (#151).
func TestCompareAckermannFunction(t *testing.T) {
	for _, value := range []struct{ m, n, want int32 }{
		{0, 0, 1}, {1, 2, 4}, {2, 3, 9}, {3, 1, 13}, {3, 3, 61},
	} {
		if got := compareAckermannFunction(value.m, value.n); got != value.want {
			t.Errorf("ackermann(%d, %d) = %d, want %d", value.m, value.n, got, value.want)
		}
	}
}

// BenchmarkCompare times the Go version of every compare/ benchmark.
func BenchmarkCompare(b *testing.B) {
	for name, program := range comparePrograms {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				program()
			}
		})
	}
}

// compareLuaFind returns a Lua 5.4 or later interpreter on PATH, or "" when
// there is none. The Lua versions need its 64 bit integers.
func compareLuaFind() string {
	for _, name := range []string{"lua5.5", "lua5.4", "lua"} {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		version, err := exec.Command(path, "-e", "io.write(_VERSION)").Output()
		if err != nil {
			continue
		}
		var major, minor int
		if _, err := fmt.Sscanf(string(version), "Lua %d.%d", &major, &minor); err == nil && (major > 5 || major == 5 && minor >= 4) {
			return path
		}
	}
	return ""
}

// TestCompareLua runs every compare/lua program and checks its output
// against the full line of compare/expected.txt. It skips without Lua 5.4.
func TestCompareLua(t *testing.T) {
	if testing.Short() {
		t.Skip("runs every benchmark at full scale")
	}
	lua := compareLuaFind()
	if lua == "" {
		t.Skip("no Lua 5.4 or later on PATH")
	}
	hashes := compareExpectedRead(t)
	for name := range comparePrograms {
		output, err := exec.Command(lua, filepath.Join("compare", "lua", name+".lua")).Output()
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		want := fmt.Sprintf("%d\n%s pass\n", hashes[[2]string{name, "full"}], name)
		if got := strings.ReplaceAll(string(output), "\r\n", "\n"); got != want {
			t.Errorf("%s: printed %q, want %q", name, got, want)
		}
	}
}
