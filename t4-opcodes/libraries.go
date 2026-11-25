package opcodes

import (
	// GCREACH: t3-ast owns the heap and the GC. This file links library
	// switches and allocates nothing, see docs/GarbageCollector.md.
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// LibraryLink links system library into this binary. The old loop runs its
// functions through dispatch, the switch t2-cx-core-opcodegen writes from
// constants.LibraryFunctionDefinitionTable (decision D96). A call of one of
// its functions compiles to constants.OP_LIBRARY_CALL. Only the process
// library is linked this way, the others are opcodes (decision D119).
func LibraryLink(library int, dispatch ast.LibraryDispatchFunction) {
	ast.LibraryDispatches[library] = dispatch
	ast.NativeNames.NativeNameIndexChanged()
}
