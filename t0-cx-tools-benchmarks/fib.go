package main

// compareFibRecursive is fib(n) by the two call recursion.
func compareFibRecursive(n int32) int32 {
	if n < 2 {
		return n
	}
	return compareFibRecursive(n-1) + compareFibRecursive(n-2)
}

// compareFib runs the recursive fib(34), 18,454,929 calls.
func compareFib() (uint64, bool) {
	result := compareFibRecursive(34)
	return compareFnv(compareFnvOffset, uint64(result)), result == 5702887
}
