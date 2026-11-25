//go:build cxcorelibrarycipher

package main

import (
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-cipher"
)

// coreLibraryCipherLinked says that the cipher library is linked into this
// core.
const coreLibraryCipherLinked = true

// The cipher library is linked into this core, see coreLibraryRegisters.
func init() {
	coreLibraryRegisters = append(coreLibraryRegisters, cipher.RegisterPackage)
}

// coreOpcodeDispatchCipher runs an opcode of the cipher library.
func coreOpcodeDispatchCipher(prgrm *ast.CXProgram, code int, inputs []ast.CXValue, outputs []ast.CXValue) bool {
	return cipher.CipherOpcodeDispatch(prgrm, code, inputs, outputs)
}
