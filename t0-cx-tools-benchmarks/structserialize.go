package main

type compareNumbers struct {
	ints [16]int32
}

var compareSerializeBuffer [72]uint8

func compareSerializeEncode(at int32, value int32) {
	buffer := &compareSerializeBuffer
	buffer[at] = uint8(value & 255)
	buffer[at+1] = uint8((value >> 8) & 255)
	buffer[at+2] = uint8((value >> 16) & 255)
	buffer[at+3] = uint8((value >> 24) & 255)
}

func compareSerializeDecode(at int32) int32 {
	buffer := &compareSerializeBuffer
	return int32(buffer[at]) | int32(buffer[at+1])<<8 | int32(buffer[at+2])<<16 | int32(buffer[at+3])<<24
}

// compareStructSerialize encodes a point and a struct of 16 int32 into a 72
// byte little endian buffer and decodes them back, 76000 times. The encoding
// is written out by hand, as in the CX version.
func compareStructSerialize() (uint64, bool) {
	hash := uint64(compareFnvOffset)
	pass := true
	var point, decoded comparePoint
	var numbers, numbersDecoded compareNumbers
	for round := int32(0); round < 76000; round++ {
		point.horizontal = round
		point.vertical = 2 * round
		for k := int32(0); k < 16; k++ {
			numbers.ints[k] = 16*round + k
		}
		compareSerializeEncode(0, point.horizontal)
		compareSerializeEncode(4, point.vertical)
		for k := int32(0); k < 16; k++ {
			compareSerializeEncode(8+4*k, numbers.ints[k])
		}
		decoded.horizontal = compareSerializeDecode(0)
		decoded.vertical = compareSerializeDecode(4)
		for k := int32(0); k < 16; k++ {
			numbersDecoded.ints[k] = compareSerializeDecode(8 + 4*k)
		}
		sum := int64(decoded.horizontal) + int64(decoded.vertical)
		for k := 0; k < 16; k++ {
			sum += int64(numbersDecoded.ints[k])
		}
		hash = compareFnv(hash, uint64(sum))
		pass = pass && sum == 259*int64(round)+120
	}
	return hash, pass
}
