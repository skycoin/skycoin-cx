//go:build cxcorelibrarynet

package main

import (
	"github.com/skycoin/skycoin-cx/t5-cxnet"
)

// The net library is linked into this core, see coreLibraryRegisters.
func init() {
	coreLibraryRegisters = append(coreLibraryRegisters, cxnet.RegisterPackage)
}
