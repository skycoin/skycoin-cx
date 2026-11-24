package cxhash

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

// cxHashVector is a published test vector of a hash.
type cxHashVector struct {
	vectorInput  string
	vectorDigest string
}

// cxHashVectors holds the published vectors of each candidate of D23 by
// CxHashName. A new hash adds its own and the test picks them by name.
var cxHashVectors = map[string][]cxHashVector{
	// FIPS 180-2 appendix B and the NIST examples, https://csrc.nist.gov/projects/cryptographic-standards-and-guidelines/example-values
	"SHA-256": {
		{"", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"abc", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{"abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq", "248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1"},
		{"abcdefghbcdefghicdefghijdefghijkefghijklfghijklmghijklmnhijklmnoijklmnopjklmnopqklmnopqrlmnopqrsmnopqrstnopqrstu", "cf5b16a778af8380036ce59e7b0492370b249b11e8f07a51afac45037afee9d1"},
	},
}

// cxHashVectorsOf returns the vectors of the current hash, failing when it
// has none.
func cxHashVectorsOf(t *testing.T) []cxHashVector {
	vectors := cxHashVectors[CxHashName]
	if len(vectors) == 0 {
		t.Fatalf("no published vectors for %s, add them to cxHashVectors", CxHashName)
	}
	return vectors
}

func TestCxHashVectors(t *testing.T) {
	for _, vector := range cxHashVectorsOf(t) {
		want, err := hex.DecodeString(vector.vectorDigest)
		if err != nil {
			t.Fatal(err)
		}
		digest := CxHashOfBytes([]byte(vector.vectorInput))
		if !bytes.Equal(digest[:], want) {
			t.Errorf("%s(%q) = %x, want %s", CxHashName, vector.vectorInput, digest, vector.vectorDigest)
		}
		// The incremental hash fed one byte at a time gives the same digest.
		state := CxHashNew()
		for i := range len(vector.vectorInput) {
			state.Write([]byte{vector.vectorInput[i]})
		}
		if sum := state.Sum(nil); !bytes.Equal(sum, want) {
			t.Errorf("incremental %s(%q) = %x, want %s", CxHashName, vector.vectorInput, sum, vector.vectorDigest)
		}
		if state.Size() != CxHashSize {
			t.Errorf("digest size %d, want %d", state.Size(), CxHashSize)
		}
	}
}

// The million a vector of FIPS 180-2 appendix B.3.
func TestCxHashMillionA(t *testing.T) {
	if CxHashName != "SHA-256" {
		t.Skipf("vector of SHA-256, the hash is %s", CxHashName)
	}
	digest := CxHashOfBytes(bytes.Repeat([]byte{'a'}, 1000000))
	if got := hex.EncodeToString(digest[:]); got != "cdc76e5c9914fb9281a1c7e284d73e67f1809a48a497200e046d39ccc7112cd0" {
		t.Errorf("million a: %s", got)
	}
}

func TestCxHashOfBytesAllocs(t *testing.T) {
	data := make([]byte, 1024)
	if allocs := testing.AllocsPerRun(100, func() { CxHashOfBytes(data) }); allocs != 0 {
		t.Errorf("%.0f allocations per hash, want 0", allocs)
	}
}

func BenchmarkCxHashOfBytes(b *testing.B) {
	for _, size := range []int{64, 1024} {
		data := make([]byte, size)
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(size))
			for b.Loop() {
				CxHashOfBytes(data)
			}
		})
	}
}
