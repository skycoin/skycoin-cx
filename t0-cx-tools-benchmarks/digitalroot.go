package main

func compareDigitSum(n int32) int32 {
	sum := int32(0)
	for ; n > 0; n /= 10 {
		sum += n % 10
	}
	return sum
}

// compareDigitalRoot takes the digital root and additive persistence, base
// 10, of every n below 1,200,000. Each n folds root + 10 * persistence.
func compareDigitalRoot() (uint64, bool) {
	hash := uint64(compareFnvOffset)
	pass := true
	for n := int32(1); n < 1200000; n++ {
		root, persistence := n, int32(0)
		for root >= 10 {
			root = compareDigitSum(root)
			persistence++
		}
		hash = compareFnv(hash, uint64(root+10*persistence))
		pass = pass && root == 1+(n-1)%9
	}
	return hash, pass
}
