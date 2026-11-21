package vm

import (
	"bytes"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// vmTestImage is a 64 byte stack and a data segment holding the i32
// constants 0, 1, 7 and 100 at 64, 68, 72 and 76, then a str "msg" at 80.
func vmTestImage(funcs ...ir.Func) *ir.Image {
	data := make([]byte, 16)
	for i, v := range []int32{0, 1, 7, 100} {
		wI32(data, uint32(4*i), v)
	}
	// A str literal points at itself, see AllocWrite_obj_data.
	str := make([]byte, types.OBJECT_HEADER_SIZE+3)
	types.Write_ptr(str, 0, types.Pointer(80))
	types.Write_obj_size(str, 0, types.OBJECT_HEADER_SIZE+3)
	copy(str[types.OBJECT_HEADER_SIZE:], "msg")
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
	if StopExitCodes[stop] != types.CX_RUNTIME_ERROR {
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
	for range 10 {
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
	if want := bytes.Repeat([]byte("100\n"), 10); !bytes.Equal(output, append(want, "7\n"...)) {
		t.Errorf("output %q", output)
	}
	if len(messages) != 1 || messages[0] != "msg" || machine.MachineTestFailures != 1 {
		t.Errorf("messages %q, %d failures, want one msg", messages, machine.MachineTestFailures)
	}
}
