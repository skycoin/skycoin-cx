//go:build !cxcorelibraryfile

package main

import (
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// coreLibraryFileLinked says that the file library is linked into this core.
const coreLibraryFileLinked = false

// coreOpcodeDispatchFile reports false, this core leaves out the file
// library.
func coreOpcodeDispatchFile(prgrm *ast.CXProgram, code int, inputs []ast.CXValue, outputs []ast.CXValue) bool {
	return false
}
