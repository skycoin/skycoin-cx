package vm

import (
	"bytes"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// vmTestImage is a 64 byte stack and a data segment holding the i32
// constants 0, 1, 7 and 100 at 64, 68, 72 and 76, then a str "msg" at 80.
func vmTestImage(funcs ...ir.Func) *ir.Image {
	data := make([]byte, 16)
	for i, v := range []int32{0, 1, 7, 100} {
		wI32(data, uint32(4*i), v)
	}
	// A str literal points at itself, as a heap str object does.
	str := make([]byte, valuetypes.ObjectHeaderSize+3)
	valuetypes.PtrWrite(str, 0, valuetypes.Pointer(80))
	valuetypes.ObjectSizeWrite(str, 0, valuetypes.ObjectHeaderSize+3)
	copy(str[valuetypes.ObjectHeaderSize:], "msg")
	data = append(data, str...)
	names := make([]string, len(funcs))
	return &ir.Image{
		ImageFuncs:     funcs,
		ImageData:      data,
		ImageDataStart: 64,
		ImageStackSize: 64,
		ImageEntryInit: ir.ImageFuncNone,
		ImageEntryMain: 0,
		ImageFuncNames: names,
	}
}

const (
	vmTestZero    = 64
	vmTestOne     = 68
	vmTestSeven   = 72
	vmTestHundred = 76
	vmTestStr     = 80
)

func frameABC(op ir.Opcode, a, b, c uint32) ir.Instr {
	return ir.Instr{InstrOpcode: op, InstrMode: ir.InstrModeFrameC, InstrA: a, InstrB: b, InstrC: c}
}

// TestRunLoop sums 0 to 99 into frame offset 0 and prints it.
func TestRunLoop(t *testing.T) {
	main := ir.Func{FuncFrameSize: 9, FuncCode: []ir.Instr{
		{InstrOpcode: ir.OpcodeZero, InstrMode: ir.InstrModeFrameA, InstrA: 0, InstrN: 8},
		{InstrOpcode: ir.OpcodeI32Lt, InstrMode: ir.InstrModeFrameA | ir.InstrModeFrameC, InstrA: 4, InstrB: vmTestHundred, InstrC: 8},
		{InstrOpcode: ir.OpcodeJumpIf, InstrMode: ir.InstrModeFrameA, InstrA: 8, InstrM: 3, InstrN: 6},
		{InstrOpcode: ir.OpcodeI32Add, InstrMode: ir.InstrModeFrameA | ir.InstrModeFrameB | ir.InstrModeFrameC, InstrA: 0, InstrB: 4, InstrC: 0},
		{InstrOpcode: ir.OpcodeI32Add, InstrMode: ir.InstrModeFrameA | ir.InstrModeFrameC, InstrA: 4, InstrB: vmTestOne, InstrC: 4},
		{InstrOpcode: ir.OpcodeJump, InstrM: 1},
		{InstrOpcode: ir.OpcodeI32Print, InstrMode: ir.InstrModeFrameA, InstrA: 0},
		{InstrOpcode: ir.OpcodeReturn},
	}}
	machine := MachineNew(vmTestImage(main), 0)
	allocs := testing.AllocsPerRun(5, func() {
		machine.Reset()
		if stop, pc := machine.Run(0, 0); stop != StopReturn || pc != 7 {
			t.Fatalf("stop %d at pc %d, want a return at 7", stop, pc)
		}
	})
	if got := string(machine.MachineOutput); got != "4950\n" {
		t.Errorf("output %q, want 4950", got)
	}
	if allocs != 0 {
		t.Errorf("%.0f allocations per run, want 0", allocs)
	}
}

// TestRunCall calls a function that returns its input times seven.
func TestRunCall(t *testing.T) {
	main := ir.Func{FuncFrameSize: 4, FuncCode: []ir.Instr{
		{InstrOpcode: ir.OpcodeCallFrame, InstrN: 8},
		{InstrOpcode: ir.OpcodeCallArg, InstrA: vmTestHundred, InstrC: 0, InstrN: 4},
		{InstrOpcode: ir.OpcodeCall, InstrM: 1},
		{InstrOpcode: ir.OpcodeCallResult, InstrMode: ir.InstrModeFrameC, InstrA: 4, InstrC: 0, InstrN: 4},
		{InstrOpcode: ir.OpcodeI32Print, InstrMode: ir.InstrModeFrameA, InstrA: 0},
		{InstrOpcode: ir.OpcodeReturn},
	}}
	times := ir.Func{FuncFrameSize: 8, FuncCode: []ir.Instr{
		{InstrOpcode: ir.OpcodeI32Mul, InstrMode: ir.InstrModeFrameA | ir.InstrModeFrameC, InstrA: 0, InstrB: vmTestSeven, InstrC: 4},
		{InstrOpcode: ir.OpcodeReturn},
	}}
	machine := MachineNew(vmTestImage(main, times), 0)
	if stop, _ := machine.Run(0, 0); stop != StopReturn {
		t.Fatalf("stop %d", stop)
	}
	if got := string(machine.MachineOutput); got != "700\n" {
		t.Errorf("output %q, want 700", got)
	}
}

// TestRunDivideByZero stops at the division, inside the callee.
func TestRunDivideByZero(t *testing.T) {
	main := ir.Func{FuncFrameSize: 4, FuncCode: []ir.Instr{
		{InstrOpcode: ir.OpcodeCallFrame, InstrN: 4},
		{InstrOpcode: ir.OpcodeCall, InstrM: 1},
		{InstrOpcode: ir.OpcodeReturn},
	}}
	divide := ir.Func{FuncFrameSize: 4, FuncCode: []ir.Instr{
		{InstrOpcode: ir.OpcodeI32Add, InstrA: vmTestOne, InstrB: vmTestOne, InstrC: vmTestSeven},
		frameABC(ir.OpcodeI32Div, vmTestOne, vmTestZero, 0),
		{InstrOpcode: ir.OpcodeReturn},
	}}
	machine := MachineNew(vmTestImage(main, divide), 0)
	stop, pc := machine.Run(0, 0)
	if stop != StopDivideByZero || pc != 1 || machine.MachineFunction != 1 || machine.MachineDepth != 1 {
		t.Fatalf("stop %d at pc %d of %d, depth %d, want a division by zero at pc 1 of 1, depth 1", stop, pc, machine.MachineFunction, machine.MachineDepth)
	}
	if machine.MachineFrames[0].FrameFunction != 0 || machine.MachineFrames[0].FramePc != 2 {
		t.Errorf("caller frame %+v", machine.MachineFrames[0])
	}
	if StopExitCodes[stop] != valuetypes.CXRuntimeError {
		t.Errorf("exit code %d", StopExitCodes[stop])
	}
}

// TestRunStackOverflow recurses until the frames fill the stack, then until
// the call depth limit.
func TestRunStackOverflow(t *testing.T) {
	for _, frameSize := range []uint32{16, 0} {
		recur := ir.Func{FuncFrameSize: frameSize, FuncCode: []ir.Instr{
			{InstrOpcode: ir.OpcodeCallFrame, InstrN: frameSize},
			{InstrOpcode: ir.OpcodeCall, InstrM: 0},
			{InstrOpcode: ir.OpcodeReturn},
		}}
		machine := MachineNew(vmTestImage(recur), 0)
		stop, pc := machine.Run(0, 0)
		wantDepth := uint32(64/16 - 1)
		if frameSize == 0 {
			wantDepth = ir.ImageCallDepthMax - 1
		}
		if stop != StopStackOverflow || pc != 0 || machine.MachineDepth != wantDepth {
			t.Errorf("frame %d: stop %d at pc %d, depth %d, want a stack overflow at pc 0, depth %d", frameSize, stop, pc, machine.MachineDepth, wantDepth)
		}
	}
}

// TestRunResume fills a small output buffer and fails a test, and the run
// goes on after each.
func TestRunResume(t *testing.T) {
	code := []ir.Instr{}
	for range 100 {
		code = append(code, ir.Instr{InstrOpcode: ir.OpcodeI32Print, InstrA: vmTestHundred})
	}
	code = append(code,
		ir.Instr{InstrOpcode: ir.OpcodeTestEqual, InstrA: vmTestOne, InstrB: vmTestSeven, InstrC: vmTestStr, InstrN: 4, InstrM: 4},
		ir.Instr{InstrOpcode: ir.OpcodeTestEqual, InstrA: vmTestOne, InstrB: vmTestOne, InstrC: vmTestStr, InstrN: 4, InstrM: 4},
		ir.Instr{InstrOpcode: ir.OpcodeI32Print, InstrA: vmTestSeven},
		ir.Instr{InstrOpcode: ir.OpcodeReturn},
	)
	machine := MachineNew(vmTestImage(ir.Func{FuncCode: code}), MachineOutputMin)
	var output []byte
	var messages []string
	stop, _ := machine.Run(0, 0)
	for Resumable(stop) {
		output = append(output, machine.MachineOutput...)
		machine.MachineOutput = machine.MachineOutput[:0]
		if stop == StopTestFailed {
			messages = append(messages, string(machine.TestMessage()))
		}
		stop, _ = machine.Resume()
	}
	output = append(output, machine.MachineOutput...)
	if stop != StopReturn {
		t.Fatalf("stop %d", stop)
	}
	if want := bytes.Repeat([]byte("100\n"), 100); !bytes.Equal(output, append(want, "7\n"...)) {
		t.Errorf("output %q", output)
	}
	if len(messages) != 1 || messages[0] != "msg" || machine.MachineTestFailures != 1 {
		t.Errorf("messages %q, %d failures, want one msg", messages, machine.MachineTestFailures)
	}
}

// vmRunCollect runs main of machine to the end, draining the output into
// output at each resumable stop, and returns the stop and all output.
func vmRunCollect(machine *Machine, output []byte) (Stop, []byte) {
	stop, _ := machine.Run(0, 0)
	for Resumable(stop) {
		output = append(output, machine.MachineOutput...)
		machine.MachineOutput = machine.MachineOutput[:0]
		stop, _ = machine.Resume()
	}
	return stop, append(output, machine.MachineOutput...)
}

// TestRunAddress writes 7 to a[2] of an i32 array at frame offset 0 through
// an index, reads it back through a pointer to a[0] plus 8 and prints it.
func TestRunAddress(t *testing.T) {
	const scratch = 64 + 16 + uint32(valuetypes.ObjectHeaderSize) + 3
	frameA := ir.InstrModeFrameA
	main := ir.Func{FuncFrameSize: 24, FuncCode: []ir.Instr{
		// index at frame 16 is 2
		{InstrOpcode: ir.OpcodeI32Add, InstrMode: ir.InstrModeFrameC, InstrA: vmTestOne, InstrB: vmTestOne, InstrC: 16},
		{InstrOpcode: ir.OpcodeAddressIndexStatic, InstrMode: frameA | ir.InstrModeFrameB, InstrA: 0, InstrB: 16, InstrN: 4, InstrC: scratch, InstrAux: 4},
		{InstrOpcode: ir.OpcodeAddressStore, InstrA: vmTestSeven, InstrC: scratch, InstrD: 1, InstrN: 4},
		// pointer to a[0] at frame 20, then load *(p + 8)
		{InstrOpcode: ir.OpcodeAddressOf, InstrMode: frameA | ir.InstrModeFrameC, InstrA: 0, InstrC: 20},
		{InstrOpcode: ir.OpcodeAddressDerefStatic, InstrMode: frameA, InstrA: 20, InstrC: scratch},
		{InstrOpcode: ir.OpcodeAddressLoad, InstrA: scratch, InstrM: 8, InstrN: 4, InstrC: scratch + 4},
		{InstrOpcode: ir.OpcodeI32Print, InstrA: scratch + 4},
		{InstrOpcode: ir.OpcodeReturn},
	}}
	image := vmTestImage(main)
	image.ImageScratchSize = 8
	machine := MachineNew(image, 0)
	allocs := testing.AllocsPerRun(5, func() {
		machine.Reset()
		if stop, pc := machine.Run(0, 0); stop != StopReturn {
			t.Fatalf("stop %d at pc %d", stop, pc)
		}
	})
	if got := string(machine.MachineOutput); got != "7\n" {
		t.Errorf("output %q, want 7", got)
	}
	if allocs != 0 {
		t.Errorf("%.0f allocations per run, want 0", allocs)
	}
}

// TestRunAddressStops stops on a negative index, on an index at the length
// of its array and on an address past the data segment.
func TestRunAddressStops(t *testing.T) {
	const scratch = 64 + 16 + uint32(valuetypes.ObjectHeaderSize) + 3
	negative := ir.Func{FuncFrameSize: 4, FuncCode: []ir.Instr{
		{InstrOpcode: ir.OpcodeI32Sub, InstrMode: ir.InstrModeFrameC, InstrA: vmTestZero, InstrB: vmTestOne, InstrC: 0},
		{InstrOpcode: ir.OpcodeAddressIndexStatic, InstrMode: ir.InstrModeFrameB, InstrA: vmTestZero, InstrB: 0, InstrN: 4, InstrC: scratch, InstrAux: 4},
		{InstrOpcode: ir.OpcodeReturn},
	}}
	outside := ir.Func{FuncFrameSize: 4, FuncCode: []ir.Instr{
		{InstrOpcode: ir.OpcodeAddressIndexStatic, InstrA: vmTestZero, InstrB: vmTestHundred, InstrN: 4, InstrC: scratch, InstrAux: 1000},
		{InstrOpcode: ir.OpcodeAddressLoad, InstrA: scratch, InstrN: 4, InstrC: scratch},
		{InstrOpcode: ir.OpcodeReturn},
	}}
	pastEnd := ir.Func{FuncFrameSize: 4, FuncCode: []ir.Instr{
		{InstrOpcode: ir.OpcodeAddressIndexStatic, InstrA: vmTestZero, InstrB: vmTestHundred, InstrN: 4, InstrC: scratch, InstrAux: 100},
		{InstrOpcode: ir.OpcodeReturn},
	}}
	for _, c := range []struct {
		fn   ir.Func
		stop Stop
		pc   uint32
	}{{negative, StopIndexNegative, 1}, {outside, StopAddressInvalid, 1}, {pastEnd, StopIndexOutOfRange, 0}} {
		image := vmTestImage(c.fn)
		image.ImageScratchSize = 4
		stop, pc := MachineNew(image, 0).Run(0, 0)
		if stop != c.stop || pc != c.pc {
			t.Errorf("stop %d at pc %d, want %d at %d", stop, pc, c.stop, c.pc)
		}
	}
}

// vmTestStrImage is a 16 byte stack and a data segment holding a str
// literal with text, which points at itself.
func vmTestStrImage(text string, code []ir.Instr) *ir.Image {
	str := make([]byte, valuetypes.ObjectHeaderSize+valuetypes.Pointer(len(text)))
	valuetypes.PtrWrite(str, 0, valuetypes.Pointer(16))
	valuetypes.ObjectSizeWrite(str, 0, valuetypes.ObjectHeaderSize+valuetypes.Pointer(len(text)))
	copy(str[valuetypes.ObjectHeaderSize:], text)
	return &ir.Image{
		ImageFuncs:     []ir.Func{{FuncCode: code}},
		ImageData:      str,
		ImageDataStart: 16,
		ImageStackSize: 16,
		ImageEntryInit: ir.ImageFuncNone,
		ImageFuncNames: []string{"main"},
	}
}

// TestRunPrintStr prints a str longer than the output buffer in each mode,
// resuming after each full buffer, without allocating.
func TestRunPrintStr(t *testing.T) {
	text := string(bytes.Repeat([]byte(`a\%b\nc\`), 300))
	for _, c := range []struct {
		mode ir.PrintStrMode
		want string
	}{
		{ir.PrintStrModeRaw, text},
		{ir.PrintStrModeLine, text + "\n"},
	} {
		image := vmTestStrImage(text, []ir.Instr{
			{InstrOpcode: ir.OpcodePrintStr, InstrA: 16, InstrN: uint32(c.mode)},
			{InstrOpcode: ir.OpcodeReturn},
		})
		machine := MachineNew(image, 0)
		output := make([]byte, 0, 4*len(text))
		var stop Stop
		allocs := testing.AllocsPerRun(3, func() {
			machine.Reset()
			stop, output = vmRunCollect(machine, output[:0])
		})
		if stop != StopReturn || string(output) != c.want {
			t.Errorf("mode %d: stop %d, %d bytes, want %d", c.mode, stop, len(output), len(c.want))
		}
		if allocs != 0 {
			t.Errorf("mode %d: %.0f allocations per run", c.mode, allocs)
		}
	}
}

// TestRunPrintValueText prints constant text and values in the printf verbs.
func TestRunPrintValueText(t *testing.T) {
	image := vmTestImage(ir.Func{FuncCode: []ir.Instr{
		{InstrOpcode: ir.OpcodePrintText, InstrN: 0, InstrM: 2},
		{InstrOpcode: ir.OpcodePrintValue, InstrA: vmTestHundred, InstrM: uint32(valuetypes.I32), InstrN: 'd'},
		{InstrOpcode: ir.OpcodePrintText, InstrN: 2, InstrM: 1},
		{InstrOpcode: ir.OpcodePrintValue, InstrA: vmTestOne, InstrM: uint32(valuetypes.BOOL), InstrN: 'b'},
		{InstrOpcode: ir.OpcodePrintText, InstrN: 2, InstrM: 1},
		{InstrOpcode: ir.OpcodePrintValue, InstrA: vmTestZero, InstrM: uint32(valuetypes.F64), InstrN: 'f'},
		{InstrOpcode: ir.OpcodeReturn},
	}})
	image.ImageText = []byte("x= ")
	machine := MachineNew(image, 0)
	allocs := testing.AllocsPerRun(5, func() {
		machine.Reset()
		if stop, pc := machine.Run(0, 0); stop != StopReturn {
			t.Fatalf("stop %d at pc %d", stop, pc)
		}
	})
	if got := string(machine.MachineOutput); got != "x=100 true 0.0000000000000000" {
		t.Errorf("output %q", got)
	}
	if allocs != 0 {
		t.Errorf("%.0f allocations per run, want 0", allocs)
	}
}
