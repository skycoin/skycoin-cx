package main

func compareFactorialLoop(n uint64) uint64 {
	result := uint64(1)
	for k := uint64(2); k <= n; k++ {
		result *= k
	}
	return result
}

// compareFactorialIter takes n! for n 0 to 20 with a loop 120000 times. 20!
// is the largest that fits a uint64.
func compareFactorialIter() (uint64, bool) {
	hash := uint64(compareFnvOffset)
	pass := true
	for round := 0; round < 120000; round++ {
		var factorial uint64
		for n := uint64(0); n <= 20; n++ {
			factorial = compareFactorialLoop(n)
			hash = compareFnv(hash, factorial)
		}
		pass = pass && factorial == 2432902008176640000
	}
	return hash, pass
}
