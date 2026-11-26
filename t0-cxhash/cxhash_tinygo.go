//go:build tinygo

package cxhash

import "hash"

// CxHashNew returns a new incremental hash, the portable BLAKE3 of
// cxhash_portable.go, as TinyGo cannot link the assembly of
// lukechampine.com/blake3.
func CxHashNew() hash.Hash {
	return cxHashPortableNew()
}

// CxHashOfBytes returns the digest of input. It allocates nothing.
func CxHashOfBytes(input []byte) [CxHashSize]byte {
	return cxHashPortableOfBytes(input)
}
