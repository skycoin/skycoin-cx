//go:build cxcorelibraryfile

package main

import (
	"github.com/skycoin/skycoin-cx/t3-ast"
	cxos "github.com/skycoin/skycoin-cx/t5-cxos"
)

// coreLibraryFileLinked says that the file library is linked into this core.
const coreLibraryFileLinked = true

// coreOpcodeDispatchFile runs an opcode of the file library.
func coreOpcodeDispatchFile(prgrm *ast.CXProgram, code int, inputs []ast.CXValue, outputs []ast.CXValue) bool {
	return cxos.FileOpcodeDispatch(prgrm, code, inputs, outputs)
}
