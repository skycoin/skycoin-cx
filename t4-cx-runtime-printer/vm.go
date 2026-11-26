package printer

import (
	"fmt"
	"runtime/debug"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	"github.com/skycoin/skycoin-cx/t2-globals"
	"github.com/skycoin/skycoin-cx/t3-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
)

// PrinterVmStopMessage is what vm prints after the error string for stop.
func PrinterVmStopMessage(stop vm.Stop) string {
	return vm.StopMessages[stop]
}

// printerVmStopLine returns the file and line of the instruction machine
// stopped at, "" and 0 when it has none, as PreviousCXLineGet gives them.
func printerVmStopLine(machine *vm.Machine) (string, int) {
	image := machine.MachineImage
	line, found := ir.LineAt(image, machine.MachineFunction, machine.MachinePc)
	if !found {
		return "", 0
	}
	return image.ImageFileNames[line.LineFile], int(line.LineNumber)
}

// PrinterVmTestFailurePrint prints a failed test as assert in t4-opcodes
// does.
func PrinterVmTestFailurePrint(machine *vm.Machine) {
	fileName, lineNumber := printerVmStopLine(machine)
	message := machine.TestMessage()
	if len(message) > 0 {
		fmt.Printf("%s: %d: result was not equal to the expected value; %s\n", fileName, lineNumber, message)
	} else {
		fmt.Printf("%s: %d: result was not equal to the expected value\n", fileName, lineNumber)
	}
}

// PrinterVmErrorPrint prints the runtime error vm stopped with as the old
// loop prints it, the call stack included. PrinterStackPrint reads the old
// program state, so the vm memory and frames are copied into prgrm first.
func PrinterVmErrorPrint(prgrm *ast.CXProgram, machine *vm.Machine, stop vm.Stop) {
	image := machine.MachineImage
	fileName, lineNumber := printerVmStopLine(machine)
	code := int(vm.StopExitCodes[stop])
	message := vm.StopMessages[stop]
	fmt.Printf("%s, %s, %v", PrinterErrorHeader(fileName, lineNumber), ast.ErrorString(code), message)

	copy(prgrm.Memory, machine.MachineMemory[:ir.ImageDataEnd(image)])
	depth := machine.MachineDepth
	for i := uint32(0); i < depth; i++ {
		frame := &machine.MachineFrames[i]
		prgrm.CallStack[i].Operator = prgrm.CXProgramFunctionSlotGid(frame.FrameFunction)
		prgrm.CallStack[i].FramePointer = types.Pointer(frame.FramePointer)
	}
	prgrm.CallStack[depth].Operator = prgrm.CXProgramFunctionSlotGid(machine.MachineFunction)
	prgrm.CallStack[depth].FramePointer = types.Pointer(machine.MachineFramePointer)
	prgrm.CallCounter = types.Pointer(depth)
	prgrm.Stack.Pointer = types.Pointer(machine.MachineStackPointer)
	PrinterStackPrint(prgrm)

	if globals.DBG_GOLANG_STACK_TRACE {
		debug.PrintStack()
	}
}
