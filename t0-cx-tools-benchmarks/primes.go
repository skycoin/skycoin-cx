package main

const compareSieveN = 2000000

var compareSieve [compareSieveN]int32

func comparePowMod(base, exponent, modulus uint64) uint64 {
	result, power := uint64(1), base%modulus
	for ; exponent > 0; exponent >>= 1 {
		if exponent&1 == 1 {
			result = result * power % modulus
		}
		power = power * power % modulus
	}
	return result
}

// compareWitness reports whether base a proves n composite, n - 1 = d 2^s.
func compareWitness(a, d, s, n uint64) bool {
	composite := false
	if a%n != 0 {
		x := comparePowMod(a, d, n)
		if x != 1 && x != n-1 {
			composite = true
			for r := uint64(1); r < s && composite; r++ {
				x = x * x % n
				if x == n-1 {
					composite = false
				}
			}
		}
	}
	return composite
}

// compareMillerRabin is deterministic below 4,759,123,141 with bases 2, 7
// and 61.
func compareMillerRabin(n uint64) bool {
	if n < 2 {
		return false
	}
	if n%2 == 0 {
		return n == 2
	}
	d, s := n-1, uint64(0)
	for d%2 == 0 {
		d /= 2
		s++
	}
	return !compareWitness(2, d, s, n) && !compareWitness(7, d, s, n) && !compareWitness(61, d, s, n)
}

// compareMulMod multiplies by doubling, so every value stays below 2^62.
func compareMulMod(x, y, modulus uint64) uint64 {
	result, addend := uint64(0), x%modulus
	for ; y > 0; y >>= 1 {
		if y&1 == 1 {
			result += addend
			if result >= modulus {
				result -= modulus
			}
		}
		addend += addend
		if addend >= modulus {
			addend -= modulus
		}
	}
	return result
}

func compareLucasLehmer(exponent uint64) bool {
	modulus, s := uint64(1)<<exponent-1, uint64(4)
	for i := uint64(0); i < exponent-2; i++ {
		s = compareMulMod(s, s, modulus)
		s = (s + modulus - 2) % modulus
	}
	return s == 0
}

// comparePrimes counts the primes below 2,000,000 with a sieve and below
// 300,000 with Miller-Rabin, and runs Lucas-Lehmer 20 times for the odd prime
// exponents 3 to 61.
func comparePrimes() (uint64, bool) {
	hash := uint64(compareFnvOffset)
	pass := true
	sieve := &compareSieve
	for i := int32(2); i < compareSieveN; i++ {
		sieve[i] = 1
	}
	for i := int32(2); i*i < compareSieveN; i++ {
		if sieve[i] == 1 {
			for j := i * i; j < compareSieveN; j += i {
				sieve[j] = 0
			}
		}
	}
	count := int32(0)
	for i := int32(2); i < compareSieveN; i++ {
		count += sieve[i]
	}
	hash = compareFnv(hash, uint64(count))
	pass = pass && count == 148933

	millerCount := int32(0)
	for k := uint64(0); k < 300000; k++ {
		if compareMillerRabin(k) {
			millerCount++
		}
	}
	hash = compareFnv(hash, uint64(millerCount))
	pass = pass && millerCount == 25997

	exponents := [17]uint64{3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61}
	for round := 0; round < 20; round++ {
		mask := uint64(0)
		for _, exponent := range exponents {
			if compareLucasLehmer(exponent) {
				mask |= 1 << exponent
				hash = compareFnv(hash, 1)
			} else {
				hash = compareFnv(hash, 0)
			}
		}
		pass = pass && mask == 2305843011361841320
	}
	return hash, pass
}
