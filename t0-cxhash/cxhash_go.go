//go:build !tinygo

package cxhash

import (
	"hash"

	"lukechampine.com/blake3"
)

// CxHashNew returns a new incremental hash.
func CxHashNew() hash.Hash {
	return blake3.New(CxHashSize, nil)
}

// CxHashOfBytes returns the digest of input. It allocates nothing.
func CxHashOfBytes(input []byte) [CxHashSize]byte {
	return blake3.Sum256(input)
}
