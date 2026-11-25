//go:build cxcorelibraryprocess

package main

import (
	cxos "github.com/skycoin/skycoin-cx/t5-cxos"
)

// The process library is linked into this core, see coreLibraryRegisters.
func init() {
	coreLibraryRegisters = append(coreLibraryRegisters, cxos.LibraryProcessRegister)
}
