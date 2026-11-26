package main

func compareAckermannFunction(m, n int32) int32 {
	if m == 0 {
		return n + 1
	}
	if n == 0 {
		return compareAckermannFunction(m-1, 1)
	}
	return compareAckermannFunction(m-1, compareAckermannFunction(m, n-1))
}

// compareAckermann runs Ackermann(3, 5) 350 times, 42,438 calls each.
func compareAckermann() (uint64, bool) {
	hash := uint64(compareFnvOffset)
	pass := true
	for round := 0; round < 350; round++ {
		result := compareAckermannFunction(3, 5)
		hash = compareFnv(hash, uint64(result))
		pass = pass && result == 253
	}
	return hash, pass
}
