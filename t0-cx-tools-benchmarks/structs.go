package main

const compareStructsSize = 100000

type compareStructsArrays struct {
	int32s [compareStructsSize]int32
	int64s [compareStructsSize]int64
}

var compareStructsBenchmark compareStructsArrays

// compareStructs fills and reads a struct of two 100000 element arrays 60
// times.
func compareStructs() (uint64, bool) {
	hash := uint64(compareFnvOffset)
	pass := true
	arrays := &compareStructsBenchmark
	for round := int32(0); round < 60; round++ {
		for c := int32(0); c < compareStructsSize; c++ {
			arrays.int32s[c] = c + round
			arrays.int64s[c] = int64(c) * 3
		}
		sum := int64(0)
		for c := int32(0); c < compareStructsSize; c++ {
			sum = sum + int64(arrays.int32s[c]) + arrays.int64s[c]
		}
		hash = compareFnv(hash, uint64(sum))
		pass = pass && sum == 19999800000+100000*int64(round)
	}
	return hash, pass
}
