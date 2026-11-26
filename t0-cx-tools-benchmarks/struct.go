package main

type comparePoint struct {
	horizontal int32
	vertical   int32
}

// compareStruct walks a [100]comparePoint through a pointer 80000 times.
func compareStruct() (uint64, bool) {
	hash := uint64(compareFnvOffset)
	pass := true
	var points [100]comparePoint
	for i := int32(0); i < 100; i++ {
		points[i].horizontal = i
		points[i].vertical = 2 * i
	}
	for round := 0; round < 80000; round++ {
		sum := int32(0)
		for i := 0; i < 100; i++ {
			point := &points[i]
			sum = sum + point.horizontal + point.vertical
		}
		hash = compareFnv(hash, uint64(sum))
		pass = pass && sum == 14850
	}
	return hash, pass
}
