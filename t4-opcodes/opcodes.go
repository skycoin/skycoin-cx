package opcodes

import (
	// GCREACH: t3-ast owns the heap and the GC roots. This file builds native
	// functions and allocates nothing, see docs/GarbageCollector.md.
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// In returns a slice of arguments from an argument list.
func In(params ...*ast.CXArgument) []*ast.CXArgument {
	return params
}

// Out returns a slice of arguments from an argument list.
func Out(params ...*ast.CXArgument) []*ast.CXArgument {
	return params
}

// MakeNativeFunction returns the native function of a library function.
// OBJMODEL: state.CXNativeFunctionMng.CXNativeFunctionAlloc(), phase 4 #356.
// Now a constructor, site S1347.
func MakeNativeFunction(prgrm *ast.CXProgram, opCode int, inputs []*ast.CXArgument, outputs []*ast.CXArgument) *ast.CXNativeFunction {
	// OBJMODEL: state.CXNativeFunctionMng.CXNativeFunctionAlloc(), phase 4 #356.
	// Now a heap literal, site S1348.
	fn := &ast.CXNativeFunction{
		AtomicOPCode: opCode,
		// OBJMODEL: state.CXArgumentMng (manager storage) or []CXArgumentGid, phase 4 #326.
		// Now a slice of objects, site S1349.
		Inputs: []*ast.CXArgument{},
		// OBJMODEL: state.CXArgumentMng (manager storage) or []CXArgumentGid, phase 4 #326.
		// Now a slice of objects, site S1350.
		Outputs: []*ast.CXArgument{},
	}

	// A native has no frame, so its parameters keep Offset 0. Writing it into the
	// shared ConstCxArg templates left every native the last registration's offset.
	fn.Inputs = append(fn.Inputs, inputs...)
	fn.Outputs = append(fn.Outputs, outputs...)

	return fn
}
