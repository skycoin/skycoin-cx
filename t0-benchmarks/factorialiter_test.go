package benchmarks

import "testing"

func BenchmarkFactorialIter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		factorialIter(10)
	}
}
