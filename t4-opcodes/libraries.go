package opcodes

import (
	"fmt"

	coreassert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
	"github.com/skycoin/skycoin-cx/t1-constants"
	// GCREACH: t3-ast owns the heap and the GC. This file registers library
	// functions and allocates nothing, see docs/GarbageCollector.md.
	"github.com/skycoin/skycoin-cx/t3-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
)

// libraryFunctionNumber returns the number of the live function name of
// library in constants.LibraryFunctionLifecycleTable.
func libraryFunctionNumber(library int, name string) int {
	row, found := constants.LibraryFunctionLiveByName(library, name)
	coreassert.AssertPreconditionTrue(found, fmt.Sprintf("%q is not a live function of library %d in constants.LibraryFunctionLifecycleTable", name, library))
	return row.Number
}

// RegisterLibraryFunction registers handler as the function name of system
// library. A call of name compiles to constants.OP_LIBRARY_CALL with the
// library and function numbers. Registering the same function again
// replaces it.
func RegisterLibraryFunction(prgrm *ast.CXProgram, library int, name string, handler ast.OpcodeHandler, inputs []*ast.CXArgument, outputs []*ast.CXArgument) {
	function := libraryFunctionNumber(library, name)
	handlers := &ast.LibraryFunctionHandlers[library]
	if function >= len(*handlers) {
		*handlers = append(*handlers, make([]ast.OpcodeHandler, function+1-len(*handlers))...)
	}
	(*handlers)[function] = handler

	ast.NativeNames.NativeNameIndexChanged()
	if inputs == nil {
		// OBJMODEL: state.CXArgumentMng (manager storage) or []CXArgumentGid, phase 4 #326.
		// Now a slice of objects, as MakeNativeFunction makes it.
		inputs = []*ast.CXArgument{}
	}
	if outputs == nil {
		// OBJMODEL: state.CXArgumentMng (manager storage) or []CXArgumentGid, phase 4 #326.
		// Now a slice of objects, as MakeNativeFunction makes it.
		outputs = []*ast.CXArgument{}
	}
	// OBJMODEL: state.CXNativeFunctionMng.CXNativeFunctionAlloc(), phase 4 #356.
	// Now a constructor call, as MakeNativeFunction makes it.
	native := MakeNativeFunction(prgrm, constants.OP_LIBRARY_CALL, inputs, outputs)
	native.LibraryNumber = library
	native.LibraryFunctionNumber = function
	natives := &ast.LibraryFunctionNatives[library]
	if function >= len(*natives) {
		*natives = append(*natives, make([]*ast.CXNativeFunction, function+1-len(*natives))...)
	}
	(*natives)[function] = native
}

// RegisterLibraryFunctionVm registers handler as the vm form of the function
// name of system library, which RegisterLibraryFunction registers.
func RegisterLibraryFunctionVm(library int, name string, handler vm.LibraryFunctionVm) {
	function := libraryFunctionNumber(library, name)
	handlers := &vm.LibraryFunctionVmHandlers[library]
	if function >= len(*handlers) {
		*handlers = append(*handlers, make([]vm.LibraryFunctionVm, function+1-len(*handlers))...)
	}
	(*handlers)[function] = handler
}
