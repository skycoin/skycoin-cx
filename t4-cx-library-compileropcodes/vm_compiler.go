package compileropcodes

import (
	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// vmBlock reads the inputs of an opcode from its block at sp in order and
// writes its output after them, the layout the lowering packs.
type vmBlock struct {
	memory []byte
	at     uint32
}

func (block *vmBlock) i64() int64 {
	value := valuetypes.I64Read(block.memory, valuetypes.Pointer(block.at))
	block.at += 8
	return value
}

func (block *vmBlock) i32() int32 {
	value := valuetypes.I32Read(block.memory, valuetypes.Pointer(block.at))
	block.at += 4
	return value
}

// bytes returns the buffer whose address and size the lowering wrote, nil
// when it lies outside memory.
func (block *vmBlock) bytes() []byte {
	address := uint64(valuetypes.PtrRead(block.memory, valuetypes.Pointer(block.at)))
	size := uint64(valuetypes.PtrRead(block.memory, valuetypes.Pointer(block.at+uint32(valuetypes.PointerSize))))
	block.at += 2 * uint32(valuetypes.PointerSize)
	if address+size > uint64(len(block.memory)) {
		return nil
	}
	return block.memory[address : address+size]
}

func (block *vmBlock) i64Write(value int64) {
	valuetypes.I64Write(block.memory, valuetypes.Pointer(block.at), value)
}

func (block *vmBlock) i32Write(value int32) {
	valuetypes.I32Write(block.memory, valuetypes.Pointer(block.at), value)
}

// VmOpcodeRun runs the compiler opcode op on the block at offset, the
// vm.CompilerOpcodeRunner cx gives a machine (decision D152). table is the
// address of the type table of reflect in memory.
func VmOpcodeRun(op ir.Opcode, memory []byte, offset uint32, table uint32, store *ir.ImageStore) {
	block := vmBlock{memory: memory, at: offset}
	var types []byte
	if table < uint32(len(memory)) {
		types = memory[table:]
	}
	state := &Library
	switch op {
	case ir.OpcodeCompilerCxCompilerSourceFileAppend:
		name, text, length := block.i64(), block.bytes(), block.i32()
		block.i32Write(state.SourceFileAppend(store, name, text, length))
	case ir.OpcodeCompilerCxCompilerSourceParse:
		block.i64Write(state.SourceParse(store))
	case ir.OpcodeCompilerCxCompilerProgramTypeCheck:
		block.i32Write(state.ProgramTypeCheck(block.i64()))
	case ir.OpcodeCompilerCxCompilerProgramLower:
		block.i64Write(state.ProgramLower(store, block.i64()))
	case ir.OpcodeCompilerCxCompilerProgramCompile:
		block.i64Write(state.ProgramCompile(store))
	case ir.OpcodeCompilerCxCompilerErrorCountGet:
		block.i32Write(state.ErrorCountGet())
	case ir.OpcodeCompilerCxCompilerErrorMessageRead:
		ordinal, buffer := block.i32(), block.bytes()
		block.i32Write(state.ErrorMessageRead(ordinal, buffer))
	case ir.OpcodeCompilerCxCompilerErrorSourceLineGet:
		block.i32Write(state.ErrorSourceLineGet(block.i32()))
	case ir.OpcodeCompilerCxAstPackageCountGet:
		block.i32Write(state.AstPackageCountGet(block.i64()))
	case ir.OpcodeCompilerCxAstPackageNameGet:
		astGid, pkg := block.i64(), block.i32()
		block.i64Write(state.AstPackageNameGet(astGid, pkg))
	case ir.OpcodeCompilerCxAstPackageFunctionCountGet:
		astGid, pkg := block.i64(), block.i32()
		block.i32Write(state.AstPackageFunctionCountGet(astGid, pkg))
	case ir.OpcodeCompilerCxAstPackageFunctionGet:
		astGid, pkg, ordinal := block.i64(), block.i32(), block.i32()
		block.i64Write(state.AstPackageFunctionGet(types, astGid, pkg, ordinal))
	case ir.OpcodeCompilerCxAstPackageStructCountGet:
		astGid, pkg := block.i64(), block.i32()
		block.i32Write(state.AstPackageStructCountGet(astGid, pkg))
	case ir.OpcodeCompilerCxAstPackageStructGet:
		astGid, pkg, ordinal := block.i64(), block.i32(), block.i32()
		block.i64Write(state.AstPackageStructGet(types, astGid, pkg, ordinal))
	case ir.OpcodeCompilerCxAstPackageGlobalCountGet:
		astGid, pkg := block.i64(), block.i32()
		block.i32Write(state.AstPackageGlobalCountGet(astGid, pkg))
	case ir.OpcodeCompilerCxAstPackageGlobalNameGet:
		astGid, pkg, ordinal := block.i64(), block.i32(), block.i32()
		block.i64Write(state.AstPackageGlobalNameGet(astGid, pkg, ordinal))
	case ir.OpcodeCompilerCxCompilerAstRelease:
		block.i32Write(state.AstRelease(block.i64()))
	}
}
