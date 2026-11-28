// Package reflectopcodes holds the old loop form of the reflect library,
// the opcodes of opcodetable.OpcodeGroupReflect (decision D150). The vm and
// the native runtime read the same table from the data segment.
package reflectopcodes

import (
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	reflecttable "github.com/skycoin/skycoin-cx/t2-cx-library-reflecttable"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// LibraryState is the state of the reflect library, its one global, see
// D139 on #412. LibraryTypeTable is the type table of the program, built
// once by LibraryPrepare before the run.
type LibraryState struct {
	LibraryTypeTable ast.ReflectTable
}

// Library is the state of the reflect library.
var Library LibraryState

// LibraryPrepare builds the type table of prgrm when it calls the reflect
// library, before the old loop runs it, so no opcode builds anything.
func LibraryPrepare(prgrm *ast.CXProgram) {
	Library.LibraryTypeTable = ast.ReflectTable{}
	if prgrm.ProgramLibrariesUsed[opcodetable.LibraryNumberReflect] {
		Library.LibraryTypeTable = ast.ReflectTableBuild(prgrm)
	}
}

// valueBytes returns the bytes of value in place in program memory, so a
// write reaches the variable.
func valueBytes(prgrm *ast.CXProgram, value *ast.CXValue) []byte {
	end := uint64(value.Offset) + uint64(value.Size)
	if end > uint64(len(prgrm.Memory)) {
		return nil
	}
	return prgrm.Memory[value.Offset:end]
}

func table() []byte {
	return Library.LibraryTypeTable.ReflectTableBytes
}

// OpCxTypeOfValueGet writes the type the compiler found for the value.
func OpCxTypeOfValueGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	index, found := Library.LibraryTypeTable.ReflectTableTypes[inputs[0].TypeSignature.Index]
	if !found {
		outputs[0].I64Set(prgrm, 0)
		return
	}
	outputs[0].I64Set(prgrm, reflecttable.TypeOfIndex(table(), index))
}

func OpCxStructFieldValueRead(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.StructFieldValueRead(table(), valueBytes(prgrm, &inputs[0]), inputs[1].I64Get(prgrm), inputs[2].I32Get(prgrm), valueBytes(prgrm, &inputs[3])))
}

func OpCxStructFieldValueWrite(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.StructFieldValueWrite(table(), valueBytes(prgrm, &inputs[0]), inputs[1].I64Get(prgrm), inputs[2].I32Get(prgrm), valueBytes(prgrm, &inputs[3])))
}

func OpCxStringPoolTextRead(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.NameTextRead(table(), inputs[0].I64Get(prgrm), valueBytes(prgrm, &inputs[1])))
}

func OpCxTypeKindGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.TypeKindGet(table(), inputs[0].I64Get(prgrm)))
}

func OpCxTypeNameGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.TypeNameGet(table(), inputs[0].I64Get(prgrm)))
}

func OpCxTypeSizeInBytesGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.TypeSizeInBytesGet(table(), inputs[0].I64Get(prgrm)))
}

func OpCxTypeAlignmentInBytesGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.TypeAlignmentInBytesGet(table(), inputs[0].I64Get(prgrm)))
}

func OpCxStructFieldCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.StructFieldCountGet(table(), inputs[0].I64Get(prgrm)))
}

func OpCxStructMethodCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.StructMethodCountGet(table(), inputs[0].I64Get(prgrm)))
}

func OpCxArrayLengthGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.ArrayLengthGet(table(), inputs[0].I64Get(prgrm)))
}

func OpCxArrayElementTypeGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.ArrayElementTypeGet(table(), inputs[0].I64Get(prgrm)))
}

func OpCxSliceElementTypeGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.SliceElementTypeGet(table(), inputs[0].I64Get(prgrm)))
}

func OpCxPointerElementTypeGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.PointerElementTypeGet(table(), inputs[0].I64Get(prgrm)))
}

func OpCxFunctionNameGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.FunctionNameGet(table(), inputs[0].I64Get(prgrm)))
}

func OpCxFunctionInputCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.FunctionInputCountGet(table(), inputs[0].I64Get(prgrm)))
}

func OpCxFunctionOutputCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.FunctionOutputCountGet(table(), inputs[0].I64Get(prgrm)))
}

func OpCxStructFieldNameGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.StructFieldNameGet(table(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxStructFieldTypeGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.StructFieldTypeGet(table(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxStructFieldOffsetInBytesGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.StructFieldOffsetInBytesGet(table(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxStructMethodNameGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.StructMethodNameGet(table(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxStructMethodFunctionGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.StructMethodFunctionGet(table(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxFunctionInputTypeGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.FunctionInputTypeGet(table(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxFunctionOutputTypeGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.FunctionOutputTypeGet(table(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxStructFieldOrdinalByNameGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.StructFieldOrdinalByNameGet(table(), inputs[0].I64Get(prgrm), inputs[1].I64Get(prgrm)))
}
