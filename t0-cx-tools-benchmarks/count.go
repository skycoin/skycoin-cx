package main

const compareCountN = 30000000

// compareCount sums 0 to 30,000,000 - 1.
func compareCount() (uint64, bool) {
	var sum int64
	for i := int32(0); i < compareCountN; i++ {
		sum += int64(i)
	}
	want := int64(compareCountN) * (compareCountN - 1) / 2
	return compareFnv(compareFnvOffset, uint64(sum)), sum == want
}
