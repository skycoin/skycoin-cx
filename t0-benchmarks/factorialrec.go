package benchmarks

func factorialRec(in int) int {
	if in == 0 {
		return 1
	} else {
		return in * factorialRec(in-1)
	}
}
