package ast

import (
	"github.com/pierrec/lz4"
)

// CompressBytesLZ4 replaces data with an LZ4 block. Input that does not
// shrink is stored as literals, which is still a valid block.
func CompressBytesLZ4(data *[]byte) {
	if len(*data) == 0 {
		return // lz4.UncompressBlock maps empty input to empty output
	}
	buf := make([]byte, lz4.CompressBlockBound(len(*data)))
	ht := make([]int, 64<<10) // buffer for the compression table

	n, err := lz4.CompressBlock(*data, buf, ht)
	if err != nil {
		panic(err)
	}

	*data = buf[:n] // compressed data
}

// lz4MaxRatio bounds the output of an LZ4 block. A match length byte of 255
// is the densest encoding, see https://github.com/lz4/lz4/blob/dev/doc/lz4_Block_format.md.
const lz4MaxRatio = 255

// UncompressBytesLZ4 replaces data with its decompressed contents. The
// original length is not stored, so the buffer grows until the block fits.
func UncompressBytesLZ4(data *[]byte) {
	src := *data
	maxSize := lz4MaxRatio*len(src) + 64
	size := 4*len(src) + 64
	for {
		if size > maxSize {
			size = maxSize
		}
		out := make([]byte, size)
		n, err := lz4.UncompressBlock(src, out)
		if err == nil {
			*data = out[:n] // uncompressed data
			return
		}
		if size == maxSize {
			panic(err)
		}
		size *= 4
	}
}
