//go:build !nocxos

package cxos

import (
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-opcodes" // thin: only LibraryLink; registration API, keep
)

// RegisterPackage links the process library. The file functions are the
// opcodes of constants.OpcodeGroupFile, which FileOpcodeDispatch runs.
func RegisterPackage(prgrm *ast.CXProgram) {
	LibraryProcessRegister(prgrm)
}

// LibraryProcessRegister links the process library, os.Run.
func LibraryProcessRegister(prgrm *ast.CXProgram) {
	opcodes.LibraryLink(constants.LibraryNumberProcess, LibraryProcessDispatch)
}
