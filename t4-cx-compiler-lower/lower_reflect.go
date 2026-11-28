package lower

import (
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	// GCREACH: t3-cx-core-ast owns the heap. The type table is built from the
	// program before the run and allocates nothing at run time, see docs/GarbageCollector.md.
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// reflectTableAppend appends the type table of the reflect library to the
// data segment, before any function lowers, so the scratch starts after it.
// No instruction writes it, so it is read only (decision D150), unless the
// program calls the meta library, which adds records to it (decision D151),
// or the compiler library, which adds the types of a parsed program (D152).
func (s *state) reflectTableAppend() {
	table := ast.ReflectTableBuild(s.stateProgram)
	s.stateReflectTable = s.stateDataEnd
	s.stateReflectTypes = table.ReflectTableTypes
	s.stateImage.ImageData = append(s.stateImage.ImageData, table.ReflectTableBytes...)
	s.stateDataEnd += uint32(len(table.ReflectTableBytes))
	if used := &s.stateProgram.ProgramLibrariesUsed; !used[opcodetable.LibraryNumberMeta] && !used[opcodetable.LibraryNumberCompiler] {
		s.stateReadOnly = append(s.stateReadOnly, ir.Span{SpanStart: s.stateReflectTable, SpanEnd: s.stateDataEnd})
	}
}

// reflectCall gives a reflect, serialize or meta opcode the address of the
// type table, and reflect.CxTypeOfValueGet the type number of its input,
// which the compiler knows. Any other library opcode is returned as it is.
func (s *state) reflectCall(call ir.Instr, inputs []ast.CXTypeSignatureIndex) (ir.Instr, ReasonCode) {
	if !ir.OpcodeTypeTableUses(call.InstrOpcode) {
		return call, ReasonNone
	}
	call.InstrAux = s.stateReflectTable
	if call.InstrOpcode == ir.OpcodeReflectCxTypeOfValueGet {
		if len(inputs) != 1 {
			return call, ReasonOperandCount
		}
		index, found := s.stateReflectTypes[inputs[0]]
		if !found {
			return call, ReasonOperandNotScalar
		}
		call.InstrN = uint32(index)
	}
	return call, ReasonNone
}
