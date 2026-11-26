// Command cx-core-image runs a CX program image on vm. It has no parser, no
// compiler and no AST, so it builds with TinyGo for microcontrollers, see
// docs/Microcontrollers.md. cx build --target=image writes the image.
package main

import (
	"os"
	"strconv"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
)

// coreImageOutputSize is the output buffer of the machine. It is small, as
// a microcontroller has little memory, and the run drains it when full.
const coreImageOutputSize = 1024

func main() {
	coreImageMarkerBegin()
	code := coreImageRun()
	coreImageMarkerExit(code)
	os.Exit(int(code))
}

// coreImageRun decodes the image, runs it and returns the exit code. The
// image comes from coreImageFileRead, embedded or named on the command line.
func coreImageRun() int32 {
	file, problem := coreImageFileRead()
	if problem != "" {
		coreImageWrite(os.Stderr, "cx-core-image: "+problem+"\n")
		return types.CX_COMPILATION_ERROR
	}
	image, fileProblem := ir.ImageFileDecode(file)
	if fileProblem != ir.ImageFileProblemNone {
		coreImageWrite(os.Stderr, "cx-core-image: "+ir.ImageFileProblemMessages[fileProblem]+"\n")
		return types.CX_COMPILATION_ERROR
	}
	for _, unlinked := range coreImageLibrariesUnlinked {
		if coreImageOpcodesEnabled(&image.ImageOperationsEnabled, unlinked.coreImageLibraryOpcodeFirst, unlinked.coreImageLibraryOpcodeLast) {
			coreImageWrite(os.Stderr, "cx-core-image: the image enables the "+constants.LibraryTable[unlinked.coreImageLibraryNumber-1].LibraryName+" library, which cx-core-image does not link\n")
			return types.CX_RUNTIME_LIBRARY_DISABLED
		}
	}
	return coreImageMachineRun(vm.MachineNew(&image, coreImageOutputSize))
}

// coreImageMachineRun runs the init entry and then main, as
// ExecutorVmRunMachine of t5-cx-runtime-core does.
func coreImageMachineRun(machine *vm.Machine) int32 {
	image := machine.MachineImage
	framePointer := uint32(0)
	if image.ImageEntryInit != ir.ImageFuncNone {
		if code := coreImageEntryRun(machine, image.ImageEntryInit, 0); code != types.CX_SUCCESS {
			return code
		}
		framePointer = image.ImageFuncs[image.ImageEntryInit].FuncFrameSize
	}
	if code := coreImageEntryRun(machine, image.ImageEntryMain, framePointer); code != types.CX_SUCCESS {
		return code
	}
	if machine.MachineTestFailures > 0 {
		return types.CX_ASSERT
	}
	return types.CX_SUCCESS
}

// coreImageEntryRun runs one entry, draining the output at every stop. An
// error prints the line a native program prints, with no call stack, as
// the image holds no variable values to print.
func coreImageEntryRun(machine *vm.Machine, entry uint32, framePointer uint32) int32 {
	stop, _ := machine.Run(entry, framePointer)
	for {
		coreImageOutputFlush(machine)
		switch {
		case stop == vm.StopReturn:
			return types.CX_SUCCESS
		case stop == vm.StopTestFailed:
			coreImageTestFailurePrint(machine)
		case !vm.Resumable(stop):
			coreImageErrorPrint(machine, stop)
			return vm.StopExitCodes[stop]
		}
		stop, _ = machine.Resume()
	}
}

func coreImageOutputFlush(machine *vm.Machine) {
	if len(machine.MachineOutput) == 0 {
		return
	}
	// A failed write is dropped, as cx drops a failed write to stdout.
	_, err := os.Stdout.Write(machine.MachineOutput)
	machine.MachineOutput = machine.MachineOutput[:0]
	if err != nil {
		return
	}
}

// coreImageStopLine returns the file and line of the instruction the
// machine stopped at, "" and 0 when it has none.
func coreImageStopLine(machine *vm.Machine) (string, string) {
	image := machine.MachineImage
	line, found := ir.LineAt(image, machine.MachineFunction, machine.MachinePc)
	if !found {
		return "", "0"
	}
	return image.ImageFileNames[line.LineFile], strconv.FormatUint(uint64(line.LineNumber), 10)
}

// coreImageTestFailurePrint prints a failed test as
// PrinterVmTestFailurePrint of t4-cx-runtime-printer does.
func coreImageTestFailurePrint(machine *vm.Machine) {
	fileName, lineNumber := coreImageStopLine(machine)
	text := fileName + ": " + lineNumber + ": result was not equal to the expected value"
	if message := machine.TestMessage(); len(message) > 0 {
		text += "; " + string(message)
	}
	coreImageWrite(os.Stdout, text+"\n")
}

// coreImageErrorPrint prints the error line of stop as a native program
// prints it, see docs/LlvmTarget.md.
func coreImageErrorPrint(machine *vm.Machine, stop vm.Stop) {
	fileName, lineNumber := coreImageStopLine(machine)
	errorString, found := constants.ErrorStrings[int(vm.StopExitCodes[stop])]
	if !found {
		errorString = constants.ErrorStrings[types.CX_RUNTIME_ERROR]
	}
	coreImageWrite(os.Stdout, "error: "+fileName+":"+lineNumber+", "+errorString+", "+vm.StopMessages[stop]+"\n")
}

// coreImageLibrary is a system library and the range of its opcodes.
type coreImageLibrary struct {
	coreImageLibraryNumber      int
	coreImageLibraryOpcodeFirst int
	coreImageLibraryOpcodeLast  int
}

// coreImageLibrariesUnlinked are the system libraries whose opcodes the vm
// of this command leaves out, as it builds with the cxcore tag and no
// cxcorelibrary tag. Cipher runs inside the vm and needs no tag. The ranges
// keep the opcode tables of t1-constants, 36 KB, out of a microcontroller.
var coreImageLibrariesUnlinked = [...]coreImageLibrary{
	{constants.LibraryNumberFile, constants.OpcodeFileGetWorkingDirectory, constants.OpcodeFileRemove},
	{constants.LibraryNumberNet, constants.OpcodeNetUpdate, constants.OpcodeNetUDPPeerResolve},
}

// coreImageOpcodesEnabled reports whether set holds an opcode from first to
// last.
func coreImageOpcodesEnabled(set *constants.OpcodeSet, first int, last int) bool {
	for code := first; code <= last; code++ {
		if set.OpcodeSetContains(code) {
			return true
		}
	}
	return false
}

// coreImageWrite writes text to file. A failed write is dropped, as cx
// drops a failed write to stdout.
func coreImageWrite(file *os.File, text string) {
	if _, err := file.WriteString(text); err != nil {
		return
	}
}
