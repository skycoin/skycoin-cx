package main

func compareFactorialRecursive(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	return n * compareFactorialRecursive(n-1)
}

// compareFactorialRec takes n! for n 0 to 20 by recursion 50000 times.
func compareFactorialRec() (uint64, bool) {
	hash := uint64(compareFnvOffset)
	pass := true
	for round := 0; round < 50000; round++ {
		var factorial uint64
		for n := uint64(0); n <= 20; n++ {
			factorial = compareFactorialRecursive(n)
			hash = compareFnv(hash, factorial)
		}
		pass = pass && factorial == 2432902008176640000
	}
	return hash, pass
}
