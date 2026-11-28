package core

import (
	"fmt"
	"os"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t0-cx-runtime-output"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	// GCREACH: t3-cx-core-ast owns the heap and the GC roots. ExecutorRun runs vm, which has no
	// heap, or the old loop, which collects, see docs/GarbageCollector.md.
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	lower "github.com/skycoin/skycoin-cx/t4-cx-compiler-lower"
	"github.com/skycoin/skycoin-cx/t4-cx-library-reflectopcodes"
	"github.com/skycoin/skycoin-cx/t4-cx-runtime-interpreter"
	"github.com/skycoin/skycoin-cx/t4-cx-runtime-opcodes"
	"github.com/skycoin/skycoin-cx/t4-cx-runtime-printer"
)

// The --executor values. auto runs a program on vm when every function
// lowers and on the old loop otherwise, step 5 of #408.
const (
	ExecutorAuto = "auto"
	ExecutorOld  = "old"
	ExecutorVm   = "vm"
)

// ExecutorOutputSize is the output buffer of a vm run.
const ExecutorOutputSize = 64 << 10

// ExecutorLower lowers prgrm. It returns "" when the program can run on vm:
// every function lowered, there is a main function, and the frames of *init
// and main fit the stack, which the old loop checks only for main. Otherwise
// it returns why not.
func ExecutorLower(prgrm *ast.CXProgram) (image ir.Image, failures []lower.Failure, outside string) {
	image, failures = lower.Program(prgrm)
	switch {
	case len(failures) > 0:
		return image, failures, executorFailureDescribe(prgrm, failures[0])
	case image.ImageEntryMain == ir.ImageFuncNone:
		return image, failures, "no main function"
	}
	entries := uint64(image.ImageFuncs[image.ImageEntryMain].FuncFrameSize)
	if image.ImageEntryInit != ir.ImageFuncNone {
		entries += uint64(image.ImageFuncs[image.ImageEntryInit].FuncFrameSize)
	}
	if entries > uint64(image.ImageStackSize) {
		return image, failures, "the frames of *init and main do not fit the stack"
	}
	return image, failures, ""
}

// executorFailureDescribe names the reason and function of a failure.
func executorFailureDescribe(prgrm *ast.CXProgram, failure lower.Failure) string {
	description := lower.ReasonNames[failure.FailureReason]
	if failure.FailureReason == lower.ReasonOperationRefused {
		return description + ", " + opcodetable.LibraryOperationName(int(failure.FailureOpcode))
	}
	if failure.FailureReason == lower.ReasonOpcodeOutsideSubset {
		if row, found := opcodetable.OpcodeLifecycleLiveByNumber(int(failure.FailureOpcode)); found {
			description += " " + row.Name
		}
	}
	return description + " in " + prgrm.CXProgramNameText(prgrm.CxIrState.CXFunctionMng.CXFunctionGet(prgrm.CXProgramFunctionSlotGid(failure.FailureFunction)).Name)
}

// ExecutorVmRun runs image on vm as execute.RunCompiled and runProgram run
// prgrm, writing to os.Stdout. It returns the exit code: CXSuccess,
// CXAssert after a failed test, or the code of a runtime error, which it
// prints like printer.RuntimeErrorPrint.
func ExecutorVmRun(prgrm *ast.CXProgram, image *ir.Image) int32 {
	return ExecutorVmRunMachine(prgrm, vm.MachineNew(image, ExecutorOutputSize))
}

// ExecutorVmRunMachine runs the image of machine from a reset machine, see
// ExecutorVmRun.
func ExecutorVmRunMachine(prgrm *ast.CXProgram, machine *vm.Machine) int32 {
	image := machine.MachineImage
	machine.Reset()
	writer := executorWriter(image.ImageOutputMode)
	framePointer := uint32(0)
	if image.ImageEntryInit != ir.ImageFuncNone {
		if code := executorVmRunEntry(prgrm, machine, writer, image.ImageEntryInit, 0); code != valuetypes.CXSuccess {
			return code
		}
		framePointer = image.ImageFuncs[image.ImageEntryInit].FuncFrameSize
	}
	if code := executorVmRunEntry(prgrm, machine, writer, image.ImageEntryMain, framePointer); code != valuetypes.CXSuccess {
		return code
	}
	if machine.MachineTestFailures > 0 {
		return valuetypes.CXAssert
	}
	return valuetypes.CXSuccess
}

// executorVmRunEntry runs one entry function, serving the resumable stops.
// A full output is handed to writer, and every other stop waits until the
// output is written, so an error or a failed test prints after it.
func executorVmRunEntry(prgrm *ast.CXProgram, machine *vm.Machine, writer *output.Writer, entry uint32, framePointer uint32) int32 {
	stop, _ := machine.Run(entry, framePointer)
	for {
		if stop == vm.StopOutputFull {
			writer.WriterHandOff(machine.MachineOutput)
		} else {
			writer.WriterFlush(machine.MachineOutput)
		}
		machine.MachineOutput = machine.MachineOutput[:0]
		switch {
		case stop == vm.StopReturn:
			return valuetypes.CXSuccess
		case stop == vm.StopTestFailed:
			printer.VmTestFailurePrint(machine)
		case !vm.Resumable(stop):
			printer.VmErrorPrint(prgrm, machine, stop)
			return vm.StopExitCodes[stop]
		}
		stop, _ = machine.Resume()
	}
}

// executorStdout writes to the os.Stdout of the moment, which a test may
// replace.
type executorStdout struct{}

// Write writes text to os.Stdout. A failed write is ignored, as fmt.Println
// in the old loop ignores it.
func (executorStdout) Write(text []byte) (int, error) {
	return os.Stdout.Write(text)
}

// VIOLATION: state outside State. Owner: t5-cx-runtime-core; moves to the run state in phase 1.
var executorWriters [output.ModeCount]*output.Writer

// executorWriter returns the writer of mode, made at the first run in mode,
// so a run allocates nothing (decision D142).
func executorWriter(mode output.Mode) *output.Writer {
	if executorWriters[mode] == nil {
		executorWriters[mode] = output.WriterNew(executorStdout{}, mode)
	}
	return executorWriters[mode]
}

// ExecutorMemoryLimitsClamp keeps the heap and stack sizes below
// constants.EscapeOffsetFlag, after the flags of cx have set them.
func ExecutorMemoryLimitsClamp() {
	opcodetable.MaxHeapSize = min(opcodetable.MaxHeapSize, opcodetable.EscapeOffsetFlag/2)
	opcodetable.HeapSizeInit = min(opcodetable.HeapSizeInit, opcodetable.MaxHeapSize)
	opcodetable.StackSize = min(opcodetable.StackSize, opcodetable.EscapeOffsetFlag/4)
}

// ExecutorRun runs the compiled or deserialized prgrm with executor, one of
// ExecutorAuto, ExecutorOld and ExecutorVm, and cxArgs as os.Args. It
// returns the exit code, after printing any runtime error.
func ExecutorRun(prgrm *ast.CXProgram, executor string, cxArgs []string) int32 {
	LibrariesEnabledResolve(prgrm)
	if code := OperationRefused(prgrm); code != 0 {
		fmt.Println(OperationRefusalLine(prgrm, code))
		return valuetypes.CXRuntimeLibraryDisabled
	}
	if LibrariesActive(prgrm)[opcodetable.LibraryNumberNet] {
		if err := executorNetRunStart(); err != nil {
			fmt.Fprintf(os.Stderr, "library %s: %s\n", opcodetable.LibraryNameOf(opcodetable.LibraryNumberNet), err)
			return valuetypes.CXInternalError
		}
	}
	prgrm.CxIrState.CxIrStateManagerGrowBoundary()
	prgrm.CxIrState.StringPool.StringPoolFreeze()

	if executor != ExecutorOld {
		image, _, outside := ExecutorLower(prgrm)
		switch {
		case outside == "":
			return ExecutorVmRun(prgrm, &image)
		case executor == ExecutorVm:
			fmt.Fprintf(os.Stderr, "cannot run on vm: %s\n", outside)
			return valuetypes.CXRuntimeNotImplemented
		}
	}

	reflectopcodes.LibraryPrepare(prgrm)
	interpreter.RunPrepare(prgrm)
	if err := interpreter.RunCompiled(prgrm, 0, cxArgs); err != nil {
		panic(err)
	}
	if prgrm.RuntimeError.RuntimeErrorRecordRaised {
		return int32(printer.RuntimeErrorPrint(prgrm))
	}
	if opcodes.AssertFailed() {
		return valuetypes.CXAssert
	}
	return valuetypes.CXSuccess
}
