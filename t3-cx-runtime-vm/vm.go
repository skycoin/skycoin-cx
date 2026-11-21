// Package vm runs an ir.Image, step 5 of skycoin/skycoin-cx#408. A Machine
// owns the memory, a fixed frame array and an output buffer, all sized
// before the run. A run allocates nothing and uses no string, interface or
// fmt. It stops with a Stop code and a pc, and the caller prints errors from
// the image's line and name tables.
package vm

import (
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// Stop says why a run returned.
type Stop uint32

const (
	// StopReturn is the entry function returning.
	StopReturn Stop = iota
	// StopOutputFull asks the caller to drain MachineOutput, then Resume.
	StopOutputFull
	// StopTestFailed is a failed test. MachineTestFailures counts it and
	// TestMessage returns its message. The caller prints it, then Resume.
	StopTestFailed
	// StopDivideByZero is an integer division or modulo by zero.
	StopDivideByZero
	// StopStackOverflow is a frame that does not fit in the stack, or a
	// call nested deeper than ir.ImageCallDepthMax.
	StopStackOverflow
	// StopOpcodeInvalid is an opcode the machine does not run.
	StopOpcodeInvalid
	// StopCount is one past the last Stop.
	StopCount
)

// stopSlow hands an instruction from the loop to the slow path. It never
// leaves the package.
const stopSlow = StopCount

// StopExitCodes is the process exit code of each Stop, the code the old loop
// exits with for the same failure. Read only.
var StopExitCodes = [StopCount]int32{
	StopReturn:        types.CX_SUCCESS,
	StopOutputFull:    types.CX_SUCCESS,
	StopTestFailed:    types.CX_SUCCESS,
	StopDivideByZero:  types.CX_RUNTIME_ERROR,
	StopStackOverflow: types.CX_RUNTIME_STACK_OVERFLOW_ERROR,
	StopOpcodeInvalid: types.CX_RUNTIME_NOT_IMPLEMENTED,
}

// Resumable reports whether the caller may Resume after stop.
func Resumable(stop Stop) bool {
	return stop == StopOutputFull || stop == StopTestFailed
}

// Frame is a caller saved by a call. FramePc is where it continues.
type Frame struct {
	FrameFunction uint32
	FramePc       uint32
	FramePointer  uint32
}

// MachineOutputMin is the smallest output buffer. Every print fits in it.
const MachineOutputMin = 64

// machinePrintMax is the most bytes one print appends, a float64 in the
// shortest form and a new line.
const machinePrintMax = 32

// Machine runs one image. MachineFrames[:MachineDepth] are the callers of
// MachineFunction. After a stop, MachineFunction, MachinePc,
// MachineFramePointer and MachineStackPointer say where it stopped.
type Machine struct {
	MachineImage        *ir.Image
	MachineMemory       []byte
	MachineOutput       []byte
	MachineTestFailures uint32
	MachineDepth        uint32
	MachineFunction     uint32
	MachinePc           uint32
	MachineFramePointer uint32
	MachineStackPointer uint32
	machineResumePc     uint32
	MachineFrames       [ir.ImageCallDepthMax]Frame
}

// MachineNew sizes the memory of a machine for image and gives it an output
// buffer of outputSize bytes, at least MachineOutputMin.
func MachineNew(image *ir.Image, outputSize int) *Machine {
	machine := &Machine{MachineImage: image}
	machine.MachineMemory = make([]byte, int(image.ImageDataStart)+len(image.ImageData))
	machine.MachineOutput = make([]byte, 0, max(outputSize, MachineOutputMin))
	machine.Reset()
	return machine
}

// Reset clears the stack, the output and the test count and reloads the data
// segment.
func (machine *Machine) Reset() {
	image := machine.MachineImage
	clear(machine.MachineMemory[:image.ImageDataStart])
	copy(machine.MachineMemory[image.ImageDataStart:], image.ImageData)
	machine.MachineOutput = machine.MachineOutput[:0]
	machine.MachineTestFailures = 0
	machine.MachineDepth = 0
}

// Run runs function entry with its frame at framePointer until it returns
// or the run stops. It returns the stop and the pc of the instruction that
// stopped.
func (machine *Machine) Run(entry uint32, framePointer uint32) (Stop, uint32) {
	image := machine.MachineImage
	machine.MachineDepth = 0
	machine.MachineFunction = entry
	machine.MachinePc = 0
	machine.MachineFramePointer = framePointer
	machine.MachineStackPointer = framePointer + image.ImageFuncs[entry].FuncFrameSize
	if machine.MachineStackPointer > image.ImageStackSize || machine.MachineStackPointer < framePointer {
		machine.MachineStackPointer = framePointer
		return StopStackOverflow, 0
	}
	machine.machineResumePc = 0
	return machine.Resume()
}

// Resume continues a run after a resumable stop.
func (machine *Machine) Resume() (Stop, uint32) {
	for {
		stop := machine.loop()
		machine.machineResumePc--
		machine.MachinePc = machine.machineResumePc
		if stop == stopSlow {
			stop = machine.slow()
		}
		if stop != stopSlow {
			return stop, machine.MachinePc
		}
	}
}

// TestMessage returns the message of the test that stopped the run, the
// bytes of a str in memory. It is empty for an empty or null str.
func (machine *Machine) TestMessage() []byte {
	in := &machine.MachineImage.ImageFuncs[machine.MachineFunction].FuncCode[machine.MachinePc]
	address := in.InstrC
	if in.InstrMode&ir.InstrModeFrameC != 0 {
		address += machine.MachineFramePointer
	}
	return memoryStr(machine.MachineMemory, address)
}

// memoryStr returns the bytes of the str at address, nil for a null str or
// one outside memory.
func memoryStr(mem []byte, address uint32) []byte {
	size := uint64(len(mem))
	if uint64(address)+uint64(types.POINTER_SIZE) > size {
		return nil
	}
	object := uint64(types.Read_ptr(mem, types.Pointer(address)))
	if object == 0 || object+uint64(types.OBJECT_HEADER_SIZE) > size {
		return nil
	}
	objectSize := uint64(types.Read_obj_size(mem, types.Pointer(object)))
	if objectSize < uint64(types.OBJECT_HEADER_SIZE) || object+objectSize > size {
		return nil
	}
	return mem[object+uint64(types.OBJECT_HEADER_SIZE) : object+objectSize]
}

// stopAt saves where the loop stopped. pc is one past the instruction, and
// Resume steps it back, out of the loop.
func (machine *Machine) stopAt(stop Stop, pc uint32, fp uint32, sp uint32) Stop {
	machine.machineResumePc = pc
	machine.MachineFramePointer = fp
	machine.MachineStackPointer = sp
	return stop
}
