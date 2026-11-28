package vm

import (
	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	reflecttable "github.com/skycoin/skycoin-cx/t2-cx-library-reflecttable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// metaOpcodeRun runs the meta opcode op on the block at offset, decision
// D151. table is the address of the type table in the data segment, which
// meta writes in place. The inputs are i64 handles from offset 0 on, and the
// output follows them.
func metaOpcodeRun(op ir.Opcode, memory []byte, offset uint32, table uint32) {
	var types []byte
	if table < uint32(len(memory)) {
		types = memory[table:]
	}
	at := func(delta uint32) valuetypes.Pointer { return valuetypes.Pointer(offset + delta) }
	first := valuetypes.I64Read(memory, at(0))
	second := valuetypes.I64Read(memory, at(8))
	third := valuetypes.I64Read(memory, at(16))
	switch op {
	case ir.OpcodeMetaCxStructDefinitionBegin:
		valuetypes.I64Write(memory, at(8), reflecttable.StructDefinitionBegin(types, first))
	case ir.OpcodeMetaCxStructDefinitionFieldAppend:
		valuetypes.I32Write(memory, at(24), reflecttable.StructDefinitionFieldAppend(types, first, second, third))
	case ir.OpcodeMetaCxStructDefinitionMethodAppend:
		valuetypes.I32Write(memory, at(24), reflecttable.StructDefinitionMethodAppend(types, first, second, third))
	case ir.OpcodeMetaCxStructDefinitionFinish:
		valuetypes.I32Write(memory, at(8), reflecttable.StructDefinitionFinish(types, first))
	case ir.OpcodeMetaCxArrayTypeCreate:
		valuetypes.I64Write(memory, at(12), reflecttable.ArrayTypeCreate(types, first, valuetypes.I32Read(memory, at(8))))
	case ir.OpcodeMetaCxSliceTypeCreate:
		valuetypes.I64Write(memory, at(8), reflecttable.SliceTypeCreate(types, first))
	case ir.OpcodeMetaCxPointerTypeCreate:
		valuetypes.I64Write(memory, at(8), reflecttable.PointerTypeCreate(types, first))
	case ir.OpcodeMetaCxFunctionDefinitionBegin:
		valuetypes.I64Write(memory, at(8), reflecttable.FunctionDefinitionBegin(types, first))
	case ir.OpcodeMetaCxFunctionDefinitionInputAppend:
		valuetypes.I32Write(memory, at(24), reflecttable.FunctionDefinitionInputAppend(types, first, second, third))
	case ir.OpcodeMetaCxFunctionDefinitionOutputAppend:
		valuetypes.I32Write(memory, at(24), reflecttable.FunctionDefinitionOutputAppend(types, first, second, third))
	case ir.OpcodeMetaCxFunctionDefinitionFinish:
		valuetypes.I32Write(memory, at(8), reflecttable.FunctionDefinitionFinish(types, first))
	case ir.OpcodeMetaCxPackageDefinitionBegin:
		valuetypes.I64Write(memory, at(8), reflecttable.PackageDefinitionBegin(types, first))
	case ir.OpcodeMetaCxPackageDefinitionFinish:
		valuetypes.I32Write(memory, at(8), reflecttable.PackageDefinitionFinish(types, first))
	case ir.OpcodeMetaCxPackageDefinitionNameGet:
		valuetypes.I64Write(memory, at(8), reflecttable.PackageDefinitionNameGet(types, first))
	case ir.OpcodeMetaCxStringPoolTextAppend:
		const value = 2 * machinePointerSize
		buffer := reflectBlockBytes(memory, offset)
		length := valuetypes.I32Read(memory, at(value))
		name := int64(0)
		if length >= 0 && int64(length) <= int64(len(buffer)) {
			name = reflecttable.StringPoolTextAppend(types, buffer[:length])
		}
		valuetypes.I64Write(memory, at(value+4), name)
	}
}
