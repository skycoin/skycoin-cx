package main

import (
	"encoding/binary"
	"runtime/debug"
	"testing"

	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-execute"
)

const runCallbackSource = `package main

func onEvent(x i32, y i32) {
	var sum i32
	var i i32
	for i = 0; i < 100; i++ {
		sum = sum + x*i - y
	}
	printf("%d %v\n", sum, x)
}

func main() {
	printf("start\n")
}
`

// runCallbackFunction returns the function of prgrm named name.
func runCallbackFunction(t testing.TB, prgrm *ast.CXProgram, name string) *ast.CXFunction {
	for i := 0; i < prgrm.CXProgramFunctionSlotCount(); i++ {
		if fn := prgrm.CXProgramFunctionSlotLive(ast.CXFunctionIndex(i)); fn != nil && prgrm.CXProgramNameText(fn.Name) == name {
			return fn
		}
	}
	t.Fatalf("no function %s", name)
	return nil
}

// runCallbackPrepare compiles runCallbackSource and leaves main as the call
// in progress at its first expression, as when an opcode of t6-cxfx calls
// back.
func runCallbackPrepare(t testing.TB) (*ast.CXProgram, *ast.CXFunction, [][]byte) {
	compileRunProgram(t, runBenchProgram{name: "callback", source: runCallbackSource})
	prgrm := actions.AST
	execute.RunPrepare(prgrm)
	mainFn := runCallbackFunction(t, prgrm, "main")
	prgrm.CallStack[0] = execute.MakeCall(prgrm, mainFn)
	prgrm.CallCounter = 0
	prgrm.Stack.Pointer = mainFn.Size
	inputs := [][]byte{make([]byte, 4), make([]byte, 4)}
	binary.LittleEndian.PutUint32(inputs[0], 3)
	binary.LittleEndian.PutUint32(inputs[1], 1)
	return prgrm, runCallbackFunction(t, prgrm, "onEvent"), inputs
}

// TestRunCallbackAllocs fails when a Callback, the nested loop t6-cxfx runs
// for an event, allocates (#407, VMALLOC-021).
func TestRunCallbackAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	discardStdout(t)
	prgrm, fn, inputs := runCallbackPrepare(t)
	stackPointer := prgrm.Stack.Pointer
	allocs := testing.AllocsPerRun(100, func() {
		execute.Callback(prgrm, fn, inputs)
	})
	if allocs != 0 {
		t.Errorf("%.0f allocations per callback, want 0", allocs)
	}
	if prgrm.CallCounter != 0 || prgrm.Stack.Pointer != stackPointer || prgrm.ProgramCallbackDepth != 0 {
		t.Errorf("after the callbacks: call %d, stack pointer %d, depth %d, want 0, %d, 0", prgrm.CallCounter, prgrm.Stack.Pointer, prgrm.ProgramCallbackDepth, stackPointer)
	}
}

// BenchmarkRunCallback times one Callback of runCallbackSource.
func BenchmarkRunCallback(b *testing.B) {
	initProgram()
	discardStdout(b)
	prgrm, fn, inputs := runCallbackPrepare(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		execute.Callback(prgrm, fn, inputs)
	}
}
