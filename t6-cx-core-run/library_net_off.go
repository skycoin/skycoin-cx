//go:build !cxcorelibrarynet

package main

import (
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// coreLibraryNetLinked says that the net library is linked into this core.
const coreLibraryNetLinked = false

// coreOpcodeDispatchNet reports false, this core leaves out the net library.
func coreOpcodeDispatchNet(prgrm *ast.CXProgram, code int, inputs []ast.CXValue, outputs []ast.CXValue) bool {
	return false
}
