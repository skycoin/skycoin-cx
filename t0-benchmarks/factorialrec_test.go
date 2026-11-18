package benchmarks

import "testing"

func BenchmarkFactorialRec(b *testing.B) {
	for i := 0; i < b.N; i++ {
		factorialRec(10)
	}
}
