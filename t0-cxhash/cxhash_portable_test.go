//go:build !tinygo

package cxhash

import (
	"testing"

	"lukechampine.com/blake3"
)

// TestCxHashPortable checks the portable BLAKE3 of TinyGo builds against
// lukechampine.com/blake3, one shot and written in pieces, over lengths
// that cross the block, chunk and tree boundaries.
func TestCxHashPortable(t *testing.T) {
	data := make([]byte, 9*1024+77)
	for i := range data {
		data[i] = byte(i*7 + i/251)
	}
	for _, size := range []int{0, 1, 63, 64, 65, 1023, 1024, 1025, 2048, 2049, 3072, 4096, 5000, 8192, len(data)} {
		want := blake3.Sum256(data[:size])
		if got := cxHashPortableOfBytes(data[:size]); got != want {
			t.Errorf("length %d: one shot %x, want %x", size, got, want)
		}
		for _, piece := range []int{1, 7, 64, 100, 1024} {
			hasher := cxHashPortableNew()
			for start := 0; start < size; start += piece {
				if _, err := hasher.Write(data[start:min(start+piece, size)]); err != nil {
					t.Fatal(err)
				}
			}
			if got := hasher.Sum(nil); string(got) != string(want[:]) {
				t.Errorf("length %d in pieces of %d: %x, want %x", size, piece, got, want)
			}
		}
	}
	big := make([]byte, 1<<20+3)
	for i := range big {
		big[i] = byte(i ^ i>>9)
	}
	if got, want := cxHashPortableOfBytes(big), blake3.Sum256(big); got != want {
		t.Errorf("length %d: %x, want %x", len(big), got, want)
	}
}
