// Command cx-core-image runs a CX program image on vm. It has no parser, no
// compiler and no AST, so it builds with TinyGo for microcontrollers, see
// docs/Microcontrollers.md. cx build --target=image writes the image.
package main

import (
	"os"
	"strconv"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t0-cx-library-terminal"
	"github.com/skycoin/skycoin-cx/t0-cx-runtime-output"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
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
	file, args, problem := coreImageFileRead()
	if problem != "" {
		coreImageWrite(os.Stderr, "cx-core-image: "+problem+"\n")
		return valuetypes.CXCompilationError
	}
	image, fileProblem := ir.ImageFileDecode(file)
	if fileProblem != ir.ImageFileProblemNone {
		coreImageWrite(os.Stderr, "cx-core-image: "+ir.ImageFileProblemMessages[fileProblem]+"\n")
		return valuetypes.CXCompilationError
	}
	checkProblem := ir.ImageCheck(&image, make([]uint64, ir.ImageCheckLabelWords(&image)))
	if checkProblem == ir.ImageCheckProblemNone {
		checkProblem = ir.ImageCheckNames(&image)
	}
	if checkProblem != ir.ImageCheckProblemNone {
		coreImageWrite(os.Stderr, "cx-core-image: "+ir.ImageCheckProblemMessages[checkProblem]+"\n")
		return valuetypes.CXCompilationError
	}
	terminal.ArgsSet(args)
	for _, unlinked := range coreImageLibrariesUnlinked {
		if coreImageOpcodesEnabled(&image.ImageOperationsEnabled, unlinked.coreImageLibraryOpcodeFirst, unlinked.coreImageLibraryOpcodeLast) {
			coreImageWrite(os.Stderr, "cx-core-image: the image enables the "+opcodetable.LibraryTable[unlinked.coreImageLibraryNumber-1].LibraryName+" library, which cx-core-image does not link\n")
			return valuetypes.CXRuntimeLibraryDisabled
		}
	}
	return coreImageMachineRun(vm.MachineNew(&image, coreImageOutputSize), output.WriterNew(os.Stdout, image.ImageOutputMode))
}

// coreImageMachineRun runs the init entry and then main, as
// ExecutorVmRunMachine of t5-cx-runtime-core does.
func coreImageMachineRun(machine *vm.Machine, writer *output.Writer) int32 {
	image := machine.MachineImage
	framePointer := uint32(0)
	if image.ImageEntryInit != ir.ImageFuncNone {
		if code := coreImageEntryRun(machine, writer, image.ImageEntryInit, 0); code != valuetypes.CXSuccess {
			return code
		}
		framePointer = image.ImageFuncs[image.ImageEntryInit].FuncFrameSize
	}
	if code := coreImageEntryRun(machine, writer, image.ImageEntryMain, framePointer); code != valuetypes.CXSuccess {
		return code
	}
	if machine.MachineTestFailures > 0 {
		return valuetypes.CXAssert
	}
	return valuetypes.CXSuccess
}

// coreImageEntryRun runs one entry, passing the output to writer at every
// stop, and waiting until it is written at every stop but a full output.
// An error prints the line a native program prints, with no call stack,
// as the image holds no variable values to print.
func coreImageEntryRun(machine *vm.Machine, writer *output.Writer, entry uint32, framePointer uint32) int32 {
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
			coreImageTestFailurePrint(machine)
		case !vm.Resumable(stop):
			coreImageErrorPrint(machine, stop)
			return vm.StopExitCodes[stop]
		}
		stop, _ = machine.Resume()
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
// VmTestFailurePrint of t4-cx-runtime-printer does.
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
	errorString, found := opcodetable.ErrorStrings[int(vm.StopExitCodes[stop])]
	if !found {
		errorString = opcodetable.ErrorStrings[valuetypes.CXRuntimeError]
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
// keep the opcode tables of t1-cx-core-opcodetable, 36 KB, out of a microcontroller.
var coreImageLibrariesUnlinked = [...]coreImageLibrary{
	{opcodetable.LibraryNumberFile, opcodetable.OpcodeFileGetWorkingDirectory, opcodetable.OpcodeFileRemove},
	{opcodetable.LibraryNumberNet, opcodetable.OpcodeNetUpdate, opcodetable.OpcodeNetUDPPeerResolve},
}

// coreImageOpcodesEnabled reports whether set holds an opcode from first to
// last.
func coreImageOpcodesEnabled(set *opcodetable.OpcodeSet, first int, last int) bool {
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
