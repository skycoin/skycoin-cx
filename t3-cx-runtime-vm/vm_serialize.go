package vm

import (
	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	reflecttable "github.com/skycoin/skycoin-cx/t2-cx-library-reflecttable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// serializeOpcodeRun runs the serialize opcode op on the block at offset,
// decision D150. table is the address of the type table of the reflect
// library in the data segment. A value or buffer is the address and size
// the lowering wrote.
func serializeOpcodeRun(op ir.Opcode, memory []byte, offset uint32, table uint32) {
	var types []byte
	if table < uint32(len(memory)) {
		types = memory[table:]
	}
	at := func(delta uint32) valuetypes.Pointer { return valuetypes.Pointer(offset + delta) }
	const value = 2 * machinePointerSize
	switch op {
	case ir.OpcodeSerializeCxSerializeEncodedSizeInBytesGet:
		valuetypes.I32Write(memory, at(8), reflecttable.TypeEncodedSizeInBytesGet(types, valuetypes.I64Read(memory, at(0))))
	case ir.OpcodeSerializeCxSerializeValueEncode:
		valuetypes.I32Write(memory, at(2*value+8), reflecttable.ValueEncode(types, memory, reflectBlockBytes(memory, offset), valuetypes.I64Read(memory, at(value)), reflectBlockBytes(memory, offset+value+8)))
	case ir.OpcodeSerializeCxSerializeValueDecode:
		valuetypes.I32Write(memory, at(2*value+8), reflecttable.ValueDecode(types, memory, reflectBlockBytes(memory, offset), reflectBlockBytes(memory, offset+value), valuetypes.I64Read(memory, at(2*value))))
	}
}
