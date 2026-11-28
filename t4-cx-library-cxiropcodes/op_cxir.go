// Package cxiropcodes holds the old loop form of the cxir library, the
// opcodes of opcodetable.OpcodeGroupCxir (decision D153). They read the
// image of the running program, which the old loop lowers once before the
// run, and build, encode, decode and run images in an ir.ImageStore whose
// room is fixed before the run. The vm runs the same store itself.
package cxiropcodes

import (
	"os"

	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
)

// LibraryState is the state of the cxir library, its one global, see D139
// on #412. LibraryImageCurrent is the image of the running program, nil
// when it does not lower.
type LibraryState struct {
	LibraryImageStore   *ir.ImageStore
	LibraryImageRunner  *vm.ImageRunner
	LibraryImageCurrent *ir.Image
}

// Library is the state of the cxir library.
var Library LibraryState

// LibraryPrepare makes the store and runner of a program that enables
// operations, before the old loop runs it. current is its image, nil when
// it has none.
func LibraryPrepare(current *ir.Image, operations *opcodetable.OpcodeSet) {
	store := ir.ImageStoreNew(ir.ImageStoreRoomDefault)
	if current != nil {
		store.ImageStoreCurrentSet(current)
	}
	Library = LibraryState{LibraryImageStore: store, LibraryImageRunner: vm.ImageRunnerNew(store, operations), LibraryImageCurrent: current}
}

// valueBytes returns the memory of a value or buffer input, nil when it
// lies outside memory.
func valueBytes(prgrm *ast.CXProgram, value *ast.CXValue) []byte {
	end := uint64(value.Offset) + uint64(value.Size)
	if end > uint64(len(prgrm.Memory)) {
		return nil
	}
	return prgrm.Memory[value.Offset:end]
}

// outputWrite writes the output of an image the old loop runs. A failed
// write is ignored, as the prints of the old loop ignore it.
func outputWrite(text []byte) {
	if _, err := os.Stdout.Write(text); err != nil {
		return
	}
}

func store() *ir.ImageStore {
	return Library.LibraryImageStore
}

func current() *ir.Image {
	return Library.LibraryImageCurrent
}

func OpCxProgramImageCurrentGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, store().ImageStoreImageCurrentGet(current()))
}

func OpCxProgramImageCreate(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, store().ImageStoreImageCreate())
}

func OpCxProgramImageFunctionCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFunctionCountGet(current(), inputs[0].I64Get(prgrm)))
}

func OpCxProgramImageEntryMainFunctionGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreEntryMainFunctionGet(current(), inputs[0].I64Get(prgrm)))
}

func OpCxProgramImageEntryInitFunctionGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreEntryInitFunctionGet(current(), inputs[0].I64Get(prgrm)))
}

func OpCxProgramImageStackSizeInBytesGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreStackSizeInBytesGet(current(), inputs[0].I64Get(prgrm)))
}

func OpCxProgramImageDataStartGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreSegmentStartGet(current(), inputs[0].I64Get(prgrm)))
}

func OpCxProgramImageDataRead(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreSegmentRead(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm), valueBytes(prgrm, &inputs[2])))
}

func OpCxProgramImageDataWrite(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreSegmentWrite(inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm), valueBytes(prgrm, &inputs[2])))
}

func OpCxProgramImageFileEncode(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFileEncode(current(), inputs[0].I64Get(prgrm), valueBytes(prgrm, &inputs[1])))
}

// OpCxProgramImageFileDecode decodes the first length bytes of the buffer,
// -1 when length is outside it.
func OpCxProgramImageFileDecode(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	buffer, length := valueBytes(prgrm, &inputs[0]), inputs[1].I32Get(prgrm)
	if length < 0 || int(length) > len(buffer) {
		outputs[0].I64Set(prgrm, -1)
		return
	}
	outputs[0].I64Set(prgrm, store().ImageStoreFileDecode(buffer[:length]))
}

// OpCxProgramImageRun runs the image to its end, its prints after the
// prints of the program so far.
func OpCxProgramImageRun(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, Library.LibraryImageRunner.ImageRunnerRun(current(), inputs[0].I64Get(prgrm), outputWrite))
}

func OpCxProgramFunctionNameGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, store().ImageStoreFunctionNameGet(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxProgramFunctionFrameSizeInBytesGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFunctionFrameSizeInBytesGet(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxProgramFunctionInputCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFunctionInputCountGet(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxProgramFunctionInputSlotRead(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFunctionInputSlotRead(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm), inputs[2].I32Get(prgrm), valueBytes(prgrm, &inputs[3])))
}

func OpCxProgramFunctionOutputCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFunctionOutputCountGet(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxProgramFunctionOutputSlotRead(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFunctionOutputSlotRead(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm), inputs[2].I32Get(prgrm), valueBytes(prgrm, &inputs[3])))
}

func OpCxProgramFunctionLocalCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFunctionLocalCountGet(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxProgramFunctionLocalRead(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFunctionLocalRead(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm), inputs[2].I32Get(prgrm), valueBytes(prgrm, &inputs[3])))
}

func OpCxProgramFunctionAppend(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFunctionAppend(inputs[0].I64Get(prgrm), inputs[1].I64Get(prgrm), inputs[2].I32Get(prgrm)))
}

func OpCxProgramFunctionInstructionCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFunctionInstructionCountGet(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxProgramFunctionInstructionRead(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFunctionInstructionRead(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm), inputs[2].I32Get(prgrm), valueBytes(prgrm, &inputs[3])))
}

func OpCxProgramFunctionInstructionWrite(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFunctionInstructionWrite(inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm), inputs[2].I32Get(prgrm), valueBytes(prgrm, &inputs[3])))
}

func OpCxProgramFunctionInstructionAppend(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreFunctionInstructionAppend(inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm), valueBytes(prgrm, &inputs[2])))
}

func OpCxOpcodeNameGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, store().ImageStoreOpcodeNameGet(inputs[0].I32Get(prgrm)))
}

func OpCxOpcodeByNameGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreOpcodeByNameGet(inputs[0].I64Get(prgrm)))
}

func OpCxOpcodeLibraryRequiredGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].BoolSet(prgrm, ir.ImageStoreOpcodeLibraryRequiredGet(inputs[0].I32Get(prgrm)))
}

func OpCxSourceLineAtProgramCounterRead(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreSourceLineAtProgramCounterRead(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm), inputs[2].I32Get(prgrm), valueBytes(prgrm, &inputs[3])))
}

func OpCxSourceFileNameGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, store().ImageStoreSourceFileNameGet(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxProgramImageEntryMainFunctionSet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreEntryMainFunctionSet(inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxProgramLocalNameGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, store().ImageStoreLocalNameGet(current(), inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

// OpCxStringPoolTextAppend names the first length bytes of the buffer, 0
// when length is outside it.
func OpCxStringPoolTextAppend(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	buffer, length := valueBytes(prgrm, &inputs[0]), inputs[1].I32Get(prgrm)
	if length < 0 || int(length) > len(buffer) {
		outputs[0].I64Set(prgrm, 0)
		return
	}
	outputs[0].I64Set(prgrm, store().ImageStoreStringPoolTextAppend(buffer[:length]))
}

func OpCxStringPoolTextRead(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreStringPoolTextRead(inputs[0].I64Get(prgrm), valueBytes(prgrm, &inputs[1])))
}

// OpCxProgramImageRelease frees a writable image for the next create,
// decode or compile, 0 or -1 (D152).
func OpCxProgramImageRelease(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, store().ImageStoreImageRelease(inputs[0].I64Get(prgrm)))
}
