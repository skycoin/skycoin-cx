// Package cxhash is the one hash function of CX, used by the cipher system
// library and the execution records. It is BLAKE3 with a 256 bit output,
// decision D23 of skycoin-cx#510. Changing it changes CxHashName here and
// CxHashNew and CxHashOfBytes in cxhash_go.go and cxhash_tinygo.go only.
package cxhash

// CxHashName names the hash, as the execution records file names it.
const CxHashName = "BLAKE3-256"

// CxHashSize is the size of a digest in bytes.
const CxHashSize = 32
