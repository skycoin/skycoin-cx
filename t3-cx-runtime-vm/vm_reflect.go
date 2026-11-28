package vm

import (
	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	reflecttable "github.com/skycoin/skycoin-cx/t2-cx-library-reflecttable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// reflectOpcodeRun runs the reflect opcode op on the block at offset,
// decision D150. table is the address of the type table in the data
// segment and typeIndex the type number of OpcodeReflectCxTypeOfValueGet. A
// value or buffer is the address and size the lowering wrote.
func reflectOpcodeRun(op ir.Opcode, memory []byte, offset uint32, table uint32, typeIndex uint32) {
	var types []byte
	if table < uint32(len(memory)) {
		types = memory[table:]
	}
	at := func(delta uint32) valuetypes.Pointer { return valuetypes.Pointer(offset + delta) }
	inputGid := valuetypes.I64Read(memory, at(0))
	ordinal := valuetypes.I32Read(memory, at(8))
	const value = 2 * machinePointerSize
	switch op {
	case ir.OpcodeReflectCxTypeOfValueGet:
		valuetypes.I64Write(memory, at(value), reflecttable.TypeOfIndex(types, int32(typeIndex)))
	case ir.OpcodeReflectCxTypeKindGet:
		valuetypes.I32Write(memory, at(8), reflecttable.TypeKindGet(types, inputGid))
	case ir.OpcodeReflectCxTypeNameGet:
		valuetypes.I64Write(memory, at(8), reflecttable.TypeNameGet(types, inputGid))
	case ir.OpcodeReflectCxTypeSizeInBytesGet:
		valuetypes.I32Write(memory, at(8), reflecttable.TypeSizeInBytesGet(types, inputGid))
	case ir.OpcodeReflectCxTypeAlignmentInBytesGet:
		valuetypes.I32Write(memory, at(8), reflecttable.TypeAlignmentInBytesGet(types, inputGid))
	case ir.OpcodeReflectCxStructFieldCountGet:
		valuetypes.I32Write(memory, at(8), reflecttable.StructFieldCountGet(types, inputGid))
	case ir.OpcodeReflectCxStructFieldNameGet:
		valuetypes.I64Write(memory, at(12), reflecttable.StructFieldNameGet(types, inputGid, ordinal))
	case ir.OpcodeReflectCxStructFieldTypeGet:
		valuetypes.I64Write(memory, at(12), reflecttable.StructFieldTypeGet(types, inputGid, ordinal))
	case ir.OpcodeReflectCxStructFieldOffsetInBytesGet:
		valuetypes.I32Write(memory, at(12), reflecttable.StructFieldOffsetInBytesGet(types, inputGid, ordinal))
	case ir.OpcodeReflectCxStructFieldOrdinalByNameGet:
		valuetypes.I32Write(memory, at(16), reflecttable.StructFieldOrdinalByNameGet(types, inputGid, valuetypes.I64Read(memory, at(8))))
	case ir.OpcodeReflectCxStructFieldValueRead, ir.OpcodeReflectCxStructFieldValueWrite:
		structValue := reflectBlockBytes(memory, offset)
		structType := valuetypes.I64Read(memory, at(value))
		fieldOrdinal := valuetypes.I32Read(memory, at(value+8))
		buffer := reflectBlockBytes(memory, offset+value+12)
		if op == ir.OpcodeReflectCxStructFieldValueRead {
			valuetypes.I32Write(memory, at(2*value+12), reflecttable.StructFieldValueRead(types, structValue, structType, fieldOrdinal, buffer))
		} else {
			valuetypes.I32Write(memory, at(2*value+12), reflecttable.StructFieldValueWrite(types, structValue, structType, fieldOrdinal, buffer))
		}
	case ir.OpcodeReflectCxStructMethodCountGet:
		valuetypes.I32Write(memory, at(8), reflecttable.StructMethodCountGet(types, inputGid))
	case ir.OpcodeReflectCxStructMethodNameGet:
		valuetypes.I64Write(memory, at(12), reflecttable.StructMethodNameGet(types, inputGid, ordinal))
	case ir.OpcodeReflectCxStructMethodFunctionGet:
		valuetypes.I64Write(memory, at(12), reflecttable.StructMethodFunctionGet(types, inputGid, ordinal))
	case ir.OpcodeReflectCxArrayLengthGet:
		valuetypes.I32Write(memory, at(8), reflecttable.ArrayLengthGet(types, inputGid))
	case ir.OpcodeReflectCxArrayElementTypeGet:
		valuetypes.I64Write(memory, at(8), reflecttable.ArrayElementTypeGet(types, inputGid))
	case ir.OpcodeReflectCxSliceElementTypeGet:
		valuetypes.I64Write(memory, at(8), reflecttable.SliceElementTypeGet(types, inputGid))
	case ir.OpcodeReflectCxPointerElementTypeGet:
		valuetypes.I64Write(memory, at(8), reflecttable.PointerElementTypeGet(types, inputGid))
	case ir.OpcodeReflectCxFunctionNameGet:
		valuetypes.I64Write(memory, at(8), reflecttable.FunctionNameGet(types, inputGid))
	case ir.OpcodeReflectCxFunctionInputCountGet:
		valuetypes.I32Write(memory, at(8), reflecttable.FunctionInputCountGet(types, inputGid))
	case ir.OpcodeReflectCxFunctionInputTypeGet:
		valuetypes.I64Write(memory, at(12), reflecttable.FunctionInputTypeGet(types, inputGid, ordinal))
	case ir.OpcodeReflectCxFunctionOutputCountGet:
		valuetypes.I32Write(memory, at(8), reflecttable.FunctionOutputCountGet(types, inputGid))
	case ir.OpcodeReflectCxFunctionOutputTypeGet:
		valuetypes.I64Write(memory, at(12), reflecttable.FunctionOutputTypeGet(types, inputGid, ordinal))
	case ir.OpcodeReflectCxStringPoolTextRead:
		valuetypes.I32Write(memory, at(8+value), reflecttable.NameTextRead(types, inputGid, reflectBlockBytes(memory, offset+8)))
	}
}

// reflectBlockBytes returns the memory whose address and size are at
// offset, empty when it lies outside memory.
func reflectBlockBytes(memory []byte, offset uint32) []byte {
	address := uint64(valuetypes.PtrRead(memory, valuetypes.Pointer(offset)))
	size := uint64(valuetypes.PtrRead(memory, valuetypes.Pointer(offset+machinePointerSize)))
	if address+size > uint64(len(memory)) {
		return nil
	}
	return memory[address : address+size]
}
