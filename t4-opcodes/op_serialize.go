package opcodes

import (
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	// GCREACH: t3-ast owns the heap and the GC roots. This file allocates a byte slice,
	// which adds to the GC pressure, see docs/GarbageCollector.md.
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// OpSerialize returns what its aff value selects as a []ui8: the whole
// running program with its memory, a memory segment, a variable, or a
// package, function or struct, see "Affordances" in docs/IR.md. A variable
// of a function that is not running is the runtime error
// CX_RUNTIME_INVALID_ARGUMENT.
// GCREACH: allocates the output byte slice, which adds to the GC pressure,
// see docs/GarbageCollector.md.
// VIOLATION VMALLOC-023: runs the serializer, compiler code, #407.
func OpSerialize(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	byts, ok := ast.SerializeAffordance(prgrm, ast.AffordanceValueRead(prgrm.Memory, inputs[0].Offset))
	if !ok {
		panic(types.CX_RUNTIME_INVALID_ARGUMENT)
	}
	count := types.Cast_int_to_ptr(len(byts))
	if count == 0 {
		outputs[0].Set_ptr(prgrm, 0)
		return
	}
	if count > ast.SliceElementLimit(prgrm, 1) {
		ast.HeapOutOfMemory(prgrm, uint64(types.OBJECT_HEADER_SIZE+constants.SLICE_HEADER_SIZE)+uint64(count))
	}
	slice := ast.SliceResizeEx(prgrm, 0, count, 1)
	copy(ast.GetSliceData(prgrm, slice, 1), byts)
	outputs[0].Set_ptr(prgrm, slice)
}

// OpDeserialize checks that a []ui8 holds a program this cx can load. Bytes
// that do not, such as those of a selection, are a runtime error. The
// program itself is not kept.
// VIOLATION VMALLOC-023: runs the serializer, compiler code, #407.
func OpDeserialize(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	byts := inputs[0].GetSlice_ui8(prgrm)
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, site S1309.
	if _, err := ast.Deserialize(byts, nil); err != nil {
		panic(types.CX_RUNTIME_INVALID_ARGUMENT)
	}
}
