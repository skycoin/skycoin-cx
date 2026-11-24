//go:build cxcorelibrarycipher

package main

import (
	"github.com/skycoin/skycoin-cx/t5-cipher"
)

// The cipher library is linked into this core, see coreLibraryRegisters.
func init() {
	coreLibraryRegisters = append(coreLibraryRegisters, cipher.RegisterPackage)
}
