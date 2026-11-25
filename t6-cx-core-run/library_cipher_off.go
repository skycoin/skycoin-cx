//go:build !cxcorelibrarycipher

package main

import (
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// coreLibraryCipherLinked says that the cipher library is linked into this
// core.
const coreLibraryCipherLinked = false

// coreOpcodeDispatchCipher reports false, this core leaves out the cipher
// library.
func coreOpcodeDispatchCipher(prgrm *ast.CXProgram, code int, inputs []ast.CXValue, outputs []ast.CXValue) bool {
	return false
}
