// Package serializeopcodes holds the old loop form of the serialize
// library, the opcodes of opcodetable.OpcodeGroupSerialize (decision D150).
// It encodes from the type table of the reflect library, which the vm and
// the native runtime read from the data segment.
package serializeopcodes

import (
	reflecttable "github.com/skycoin/skycoin-cx/t2-cx-library-reflecttable"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// LibraryState is the state of the serialize library, its one global, see
// D139 on #412. LibraryTypeTable is the type table of the reflect library,
// shared and not built again.
type LibraryState struct {
	LibraryTypeTable ast.ReflectTable
}

// Library is the state of the serialize library.
var Library LibraryState

// LibraryPrepare gives the library the type table the reflect library
// built for the run, before the old loop runs it.
func LibraryPrepare(table ast.ReflectTable) {
	Library.LibraryTypeTable = table
}

// valueBytes returns the bytes of value in place in program memory, so a
// decode reaches the variable.
func valueBytes(prgrm *ast.CXProgram, value *ast.CXValue) []byte {
	end := uint64(value.Offset) + uint64(value.Size)
	if end > uint64(len(prgrm.Memory)) {
		return nil
	}
	return prgrm.Memory[value.Offset:end]
}

func OpCxSerializeEncodedSizeInBytesGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.TypeEncodedSizeInBytesGet(Library.LibraryTypeTable.ReflectTableBytes, inputs[0].I64Get(prgrm)))
}

func OpCxSerializeValueEncode(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.ValueEncode(Library.LibraryTypeTable.ReflectTableBytes, prgrm.Memory, valueBytes(prgrm, &inputs[0]), inputs[1].I64Get(prgrm), valueBytes(prgrm, &inputs[2])))
}

func OpCxSerializeValueDecode(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, reflecttable.ValueDecode(Library.LibraryTypeTable.ReflectTableBytes, prgrm.Memory, valueBytes(prgrm, &inputs[0]), valueBytes(prgrm, &inputs[1]), inputs[2].I64Get(prgrm)))
}
