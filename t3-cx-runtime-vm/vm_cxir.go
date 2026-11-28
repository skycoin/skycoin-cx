package vm

import (
	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// ImageRunMemoryMax is the most memory an image sys/cxir runs may use, its
// stack, data segment and scratch. The machine that runs it is this size,
// made before the run (decision D153).
const ImageRunMemoryMax = 2 << 20

// ImageRunner runs an image of sys/cxir in a fresh machine and gives back
// its exit code (decision D153). The image passes ir.ImageCheck, the loader
// check of an image file, and may only use the library operations the
// running program enables, ImageRunnerOperations. The machine it runs on
// has no runner, so an image it runs cannot run another.
type ImageRunner struct {
	ImageRunnerStore      *ir.ImageStore
	ImageRunnerOperations opcodetable.OpcodeSet
	imageRunnerMachine    Machine
	imageRunnerMemory     []byte
	imageRunnerLabels     []uint64
	imageRunnerGid        int64
	imageRunnerActive     bool
	imageRunnerStarted    bool
	imageRunnerMain       bool
}

// ImageRunnerNew returns a runner of the images of store for a program that
// enables operations, with its machine and output buffer made.
func ImageRunnerNew(store *ir.ImageStore, operations *opcodetable.OpcodeSet) *ImageRunner {
	runner := &ImageRunner{ImageRunnerStore: store, ImageRunnerOperations: *operations}
	runner.imageRunnerMemory = make([]byte, ImageRunMemoryMax+ir.ImageCheckTailSize)
	runner.imageRunnerLabels = make([]uint64, store.ImageStoreRoom.ImageStoreRoomInstructions/64+1)
	runner.imageRunnerMachine.MachineOutput = make([]byte, 0, MachineOutputMin)
	return runner
}

// imageRunnerStart checks the image of imageGid and loads it into the
// machine, false when it is refused and must not run.
func (runner *ImageRunner) imageRunnerStart(current *ir.Image, imageGid int64) bool {
	image := runner.ImageRunnerStore.ImageStoreRunBegin(current, imageGid)
	if image == nil {
		return false
	}
	size := uint64(ir.ImageDataEnd(image)) + uint64(image.ImageScratchSize)
	if !ir.ImageOperationsWithin(image, &runner.ImageRunnerOperations) || ir.ImageCheckLabelWords(image) > len(runner.imageRunnerLabels) ||
		ir.ImageCheck(image, runner.imageRunnerLabels) != ir.ImageCheckProblemNone || size > ImageRunMemoryMax {
		runner.ImageRunnerStore.ImageStoreRunEnd(imageGid)
		return false
	}
	machine := &runner.imageRunnerMachine
	machine.MachineImage = image
	machine.machineDataEnd = ir.ImageDataEnd(image)
	machine.MachineMemory = runner.imageRunnerMemory[:size+ir.ImageCheckTailSize]
	clear(machine.MachineMemory)
	machine.MachineImageStore = runner.ImageRunnerStore
	machine.MachineImageRunner = nil
	output := machine.MachineOutput
	machine.Reset()
	machine.MachineOutput = output
	runner.imageRunnerGid = imageGid
	runner.imageRunnerActive = true
	runner.imageRunnerStarted = false
	runner.imageRunnerMain = image.ImageEntryInit == ir.ImageFuncNone
	return true
}

// imageRunnerResume runs the loaded image until it ends or stops with a
// full output or a flush, which the caller serves before it resumes. It
// returns the stop, whether the image ended, and its exit code when it did.
// A failed test counts and goes on, and an error ends the image with the
// exit code of its stop, printing nothing.
func (runner *ImageRunner) imageRunnerResume() (Stop, bool, int32) {
	machine := &runner.imageRunnerMachine
	image := machine.MachineImage
	var stop Stop
	switch {
	case runner.imageRunnerStarted:
		stop, _ = machine.Resume()
	case runner.imageRunnerMain:
		stop, _ = machine.Run(image.ImageEntryMain, runner.imageRunnerMainFramePointer())
	default:
		stop, _ = machine.Run(image.ImageEntryInit, 0)
	}
	runner.imageRunnerStarted = true
	for {
		switch {
		case stop == StopOutputFull || stop == StopOutputFlush:
			return stop, false, 0
		case stop == StopTestFailed:
			stop, _ = machine.Resume()
			continue
		case stop == StopReturn && !runner.imageRunnerMain:
			runner.imageRunnerMain = true
			stop, _ = machine.Run(image.ImageEntryMain, runner.imageRunnerMainFramePointer())
			continue
		}
		code := StopExitCodes[stop]
		if stop == StopReturn && machine.MachineTestFailures > 0 {
			code = valuetypes.CXAssert
		}
		runner.imageRunnerActive = false
		runner.ImageRunnerStore.ImageStoreRunEnd(runner.imageRunnerGid)
		return stop, true, code
	}
}

// imageRunnerMainFramePointer is where the frame of main starts, after the
// frame of *init when the image has one.
func (runner *ImageRunner) imageRunnerMainFramePointer() uint32 {
	image := runner.imageRunnerMachine.MachineImage
	if image.ImageEntryInit == ir.ImageFuncNone {
		return 0
	}
	return image.ImageFuncs[image.ImageEntryInit].FuncFrameSize
}

// ImageRunnerRun runs the image of imageGid to its end for a caller that is
// not a machine, the old loop, passing its output to write at every stop.
// It returns the exit code, -1 when the image is refused. current is the
// image of the caller, nil when it has none.
func (runner *ImageRunner) ImageRunnerRun(current *ir.Image, imageGid int64, write func([]byte)) int32 {
	if runner.imageRunnerActive || !runner.imageRunnerStart(current, imageGid) {
		return -1
	}
	machine := &runner.imageRunnerMachine
	for {
		_, done, code := runner.imageRunnerResume()
		write(machine.MachineOutput)
		machine.MachineOutput = machine.MachineOutput[:0]
		if done {
			return code
		}
	}
}

// cxirRun runs OpcodeCxirCxProgramImageRun on the block at sp. The image
// prints into the output of machine, so a full output or a flush stops
// machine at this instruction, which goes on with the image when the caller
// resumes.
func (machine *Machine) cxirRun() Stop {
	sp := machine.MachineStackPointer
	runner := machine.MachineImageRunner
	if runner == nil {
		valuetypes.I32Write(machine.MachineMemory, valuetypes.Pointer(sp+8), -1)
		return stopSlow
	}
	if !runner.imageRunnerActive && !runner.imageRunnerStart(machine.MachineImage, valuetypes.I64Read(machine.MachineMemory, valuetypes.Pointer(sp))) {
		valuetypes.I32Write(machine.MachineMemory, valuetypes.Pointer(sp+8), -1)
		return stopSlow
	}
	child := &runner.imageRunnerMachine
	child.MachineOutput = machine.MachineOutput
	stop, done, code := runner.imageRunnerResume()
	machine.MachineOutput = child.MachineOutput
	if !done {
		machine.machineResumePc = machine.MachinePc
		return stop
	}
	valuetypes.I32Write(machine.MachineMemory, valuetypes.Pointer(sp+8), code)
	return stopSlow
}

// cxirBlock reads the inputs of a cxir opcode from its block in order and
// writes its output after them.
type cxirBlock struct {
	memory []byte
	at     uint32
}

func (block *cxirBlock) i64() int64 {
	value := valuetypes.I64Read(block.memory, valuetypes.Pointer(block.at))
	block.at += 8
	return value
}

func (block *cxirBlock) i32() int32 {
	value := valuetypes.I32Read(block.memory, valuetypes.Pointer(block.at))
	block.at += 4
	return value
}

// bytes returns the value or buffer whose address and size the lowering
// wrote, empty when it lies outside memory.
func (block *cxirBlock) bytes() []byte {
	value := reflectBlockBytes(block.memory, block.at)
	block.at += 2 * machinePointerSize
	return value
}

func (block *cxirBlock) i64Write(value int64) {
	valuetypes.I64Write(block.memory, valuetypes.Pointer(block.at), value)
}

func (block *cxirBlock) i32Write(value int32) {
	valuetypes.I32Write(block.memory, valuetypes.Pointer(block.at), value)
}

// cxirOpcodeRun runs the cxir opcode op on the block at offset, decision
// D153, with the images of store. current is the image the machine runs.
func cxirOpcodeRun(op ir.Opcode, memory []byte, offset uint32, store *ir.ImageStore, current *ir.Image) {
	block := cxirBlock{memory: memory, at: offset}
	if store == nil {
		return
	}
	switch op {
	case ir.OpcodeCxirCxProgramImageCurrentGet:
		block.i64Write(store.ImageStoreImageCurrentGet(current))
	case ir.OpcodeCxirCxProgramImageCreate:
		block.i64Write(store.ImageStoreImageCreate())
	case ir.OpcodeCxirCxProgramImageFunctionCountGet:
		block.i32Write(store.ImageStoreFunctionCountGet(current, block.i64()))
	case ir.OpcodeCxirCxProgramImageEntryMainFunctionGet:
		block.i32Write(store.ImageStoreEntryMainFunctionGet(current, block.i64()))
	case ir.OpcodeCxirCxProgramImageEntryInitFunctionGet:
		block.i32Write(store.ImageStoreEntryInitFunctionGet(current, block.i64()))
	case ir.OpcodeCxirCxProgramImageStackSizeInBytesGet:
		block.i32Write(store.ImageStoreStackSizeInBytesGet(current, block.i64()))
	case ir.OpcodeCxirCxProgramImageDataStartGet:
		block.i32Write(store.ImageStoreSegmentStartGet(current, block.i64()))
	case ir.OpcodeCxirCxProgramImageDataRead:
		image, offset, buffer := block.i64(), block.i32(), block.bytes()
		block.i32Write(store.ImageStoreSegmentRead(current, image, offset, buffer))
	case ir.OpcodeCxirCxProgramImageDataWrite:
		image, offset, buffer := block.i64(), block.i32(), block.bytes()
		block.i32Write(store.ImageStoreSegmentWrite(image, offset, buffer))
	case ir.OpcodeCxirCxProgramImageFileEncode:
		image, buffer := block.i64(), block.bytes()
		block.i32Write(store.ImageStoreFileEncode(current, image, buffer))
	case ir.OpcodeCxirCxProgramImageFileDecode:
		buffer, length := block.bytes(), block.i32()
		imageGid := int64(-1)
		if length >= 0 && int(length) <= len(buffer) {
			imageGid = store.ImageStoreFileDecode(buffer[:length])
		}
		block.i64Write(imageGid)
	case ir.OpcodeCxirCxProgramFunctionNameGet:
		image, function := block.i64(), block.i32()
		block.i64Write(store.ImageStoreFunctionNameGet(current, image, function))
	case ir.OpcodeCxirCxProgramFunctionFrameSizeInBytesGet:
		image, function := block.i64(), block.i32()
		block.i32Write(store.ImageStoreFunctionFrameSizeInBytesGet(current, image, function))
	case ir.OpcodeCxirCxProgramFunctionInputCountGet:
		image, function := block.i64(), block.i32()
		block.i32Write(store.ImageStoreFunctionInputCountGet(current, image, function))
	case ir.OpcodeCxirCxProgramFunctionInputSlotRead:
		image, function, ordinal, slot := block.i64(), block.i32(), block.i32(), block.bytes()
		block.i32Write(store.ImageStoreFunctionInputSlotRead(current, image, function, ordinal, slot))
	case ir.OpcodeCxirCxProgramFunctionOutputCountGet:
		image, function := block.i64(), block.i32()
		block.i32Write(store.ImageStoreFunctionOutputCountGet(current, image, function))
	case ir.OpcodeCxirCxProgramFunctionOutputSlotRead:
		image, function, ordinal, slot := block.i64(), block.i32(), block.i32(), block.bytes()
		block.i32Write(store.ImageStoreFunctionOutputSlotRead(current, image, function, ordinal, slot))
	case ir.OpcodeCxirCxProgramFunctionLocalCountGet:
		image, function := block.i64(), block.i32()
		block.i32Write(store.ImageStoreFunctionLocalCountGet(current, image, function))
	case ir.OpcodeCxirCxProgramFunctionLocalRead:
		image, function, ordinal, local := block.i64(), block.i32(), block.i32(), block.bytes()
		block.i32Write(store.ImageStoreFunctionLocalRead(current, image, function, ordinal, local))
	case ir.OpcodeCxirCxProgramFunctionAppend:
		image, name, frameSize := block.i64(), block.i64(), block.i32()
		block.i32Write(store.ImageStoreFunctionAppend(image, name, frameSize))
	case ir.OpcodeCxirCxProgramFunctionInstructionCountGet:
		image, function := block.i64(), block.i32()
		block.i32Write(store.ImageStoreFunctionInstructionCountGet(current, image, function))
	case ir.OpcodeCxirCxProgramFunctionInstructionRead:
		image, function, pc, instruction := block.i64(), block.i32(), block.i32(), block.bytes()
		block.i32Write(store.ImageStoreFunctionInstructionRead(current, image, function, pc, instruction))
	case ir.OpcodeCxirCxProgramFunctionInstructionWrite:
		image, function, pc, instruction := block.i64(), block.i32(), block.i32(), block.bytes()
		block.i32Write(store.ImageStoreFunctionInstructionWrite(image, function, pc, instruction))
	case ir.OpcodeCxirCxProgramFunctionInstructionAppend:
		image, function, instruction := block.i64(), block.i32(), block.bytes()
		block.i32Write(store.ImageStoreFunctionInstructionAppend(image, function, instruction))
	case ir.OpcodeCxirCxOpcodeNameGet:
		block.i64Write(store.ImageStoreOpcodeNameGet(block.i32()))
	case ir.OpcodeCxirCxOpcodeByNameGet:
		block.i32Write(store.ImageStoreOpcodeByNameGet(block.i64()))
	case ir.OpcodeCxirCxOpcodeLibraryRequiredGet:
		required := ir.ImageStoreOpcodeLibraryRequiredGet(block.i32())
		wBool(memory, block.at, required)
	case ir.OpcodeCxirCxSourceLineAtProgramCounterRead:
		image, function, pc, line := block.i64(), block.i32(), block.i32(), block.bytes()
		block.i32Write(store.ImageStoreSourceLineAtProgramCounterRead(current, image, function, pc, line))
	case ir.OpcodeCxirCxSourceFileNameGet:
		image, sourceFile := block.i64(), block.i32()
		block.i64Write(store.ImageStoreSourceFileNameGet(current, image, sourceFile))
	case ir.OpcodeCxirCxProgramImageEntryMainFunctionSet:
		image, function := block.i64(), block.i32()
		block.i32Write(store.ImageStoreEntryMainFunctionSet(image, function))
	case ir.OpcodeCxirCxProgramLocalNameGet:
		image, localName := block.i64(), block.i32()
		block.i64Write(store.ImageStoreLocalNameGet(current, image, localName))
	case ir.OpcodeCxirCxStringPoolTextAppend:
		buffer, length := block.bytes(), block.i32()
		name := int64(0)
		if length >= 0 && int(length) <= len(buffer) {
			name = store.ImageStoreStringPoolTextAppend(buffer[:length])
		}
		block.i64Write(name)
	case ir.OpcodeCxirCxStringPoolTextRead:
		name, buffer := block.i64(), block.bytes()
		block.i32Write(store.ImageStoreStringPoolTextRead(name, buffer))
	}
}
