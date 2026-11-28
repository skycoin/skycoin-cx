// Package metaopcodes holds the old loop form of the meta library, the
// opcodes of opcodetable.OpcodeGroupMeta (decision D151). They add types,
// function signatures and packages to the type table of the reflect
// library in place, in the room it keeps, so reflect and serialize read
// them. The vm and the native runtime write the same table in the data
// segment.
package metaopcodes

import (
	reflecttable "github.com/skycoin/skycoin-cx/t2-cx-library-reflecttable"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// LibraryState is the state of the meta library, its one global, see D139
// on #412. LibraryTypeTable is the type table of the reflect library, which
// meta writes in place.
type LibraryState struct {
	LibraryTypeTable ast.ReflectTable
}

// Library is the state of the meta library.
var Library LibraryState

// LibraryPrepare gives the library the type table the reflect library
// built for the run, before the old loop runs it.
func LibraryPrepare(table ast.ReflectTable) {
	Library.LibraryTypeTable = table
}

func table() []byte {
	return Library.LibraryTypeTable.ReflectTableBytes
}

func OpCxStructDefinitionBegin(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.StructDefinitionBegin(table(), inputs[0].I64Get(prgrm)))
}

func OpCxStructDefinitionFieldAppend(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.StructDefinitionFieldAppend(table(), inputs[0].I64Get(prgrm), inputs[1].I64Get(prgrm), inputs[2].I64Get(prgrm)))
}

func OpCxStructDefinitionMethodAppend(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.StructDefinitionMethodAppend(table(), inputs[0].I64Get(prgrm), inputs[1].I64Get(prgrm), inputs[2].I64Get(prgrm)))
}

func OpCxStructDefinitionFinish(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.StructDefinitionFinish(table(), inputs[0].I64Get(prgrm)))
}

func OpCxArrayTypeCreate(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.ArrayTypeCreate(table(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxSliceTypeCreate(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.SliceTypeCreate(table(), inputs[0].I64Get(prgrm)))
}

func OpCxPointerTypeCreate(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.PointerTypeCreate(table(), inputs[0].I64Get(prgrm)))
}

func OpCxFunctionDefinitionBegin(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.FunctionDefinitionBegin(table(), inputs[0].I64Get(prgrm)))
}

func OpCxFunctionDefinitionInputAppend(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.FunctionDefinitionInputAppend(table(), inputs[0].I64Get(prgrm), inputs[1].I64Get(prgrm), inputs[2].I64Get(prgrm)))
}

func OpCxFunctionDefinitionOutputAppend(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.FunctionDefinitionOutputAppend(table(), inputs[0].I64Get(prgrm), inputs[1].I64Get(prgrm), inputs[2].I64Get(prgrm)))
}

func OpCxFunctionDefinitionFinish(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.FunctionDefinitionFinish(table(), inputs[0].I64Get(prgrm)))
}

func OpCxPackageDefinitionBegin(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.PackageDefinitionBegin(table(), inputs[0].I64Get(prgrm)))
}

func OpCxPackageDefinitionFinish(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.PackageDefinitionFinish(table(), inputs[0].I64Get(prgrm)))
}

func OpCxPackageDefinitionNameGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, reflecttable.PackageDefinitionNameGet(table(), inputs[0].I64Get(prgrm)))
}

// OpCxStringPoolTextAppend adds the first length bytes of the buffer as a
// member name, 0 when length is outside the buffer.
func OpCxStringPoolTextAppend(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	length := inputs[1].I32Get(prgrm)
	end := uint64(inputs[0].Offset) + uint64(length)
	if length < 0 || uint32(length) > uint32(inputs[0].Size) || end > uint64(len(prgrm.Memory)) {
		outputs[0].I64Set(prgrm, 0)
		return
	}
	outputs[0].I64Set(prgrm, reflecttable.StringPoolTextAppend(table(), prgrm.Memory[inputs[0].Offset:end]))
}
