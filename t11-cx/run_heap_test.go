package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-execute"
)

// runHeapProgram is a heap heavy test program and the heap sizes it runs
// with. initHeap 0 collects after nearly every step that allocates.
type runHeapProgram struct {
	path     string
	initHeap types.Pointer
	heapCap  types.Pointer
}

// runHeapPrograms allocate, collect and grow the heap: the GC tests with an
// initial heap of 0 bytes, test-gc.cx at the default heap, and a live heap
// grown near a heap cap of 512K that collects there.
func runHeapPrograms() []runHeapProgram {
	programs := []runHeapProgram{{"test-gc.cx", constants.INIT_HEAP_SIZE, constants.MAX_HEAP_SIZE}}
	for _, path := range append(gcBenchPrograms, "test-gc.cx", "test-gc-slice-resize.cx", "test-gc-array-of-slices.cx") {
		programs = append(programs, runHeapProgram{path, 0, constants.MAX_HEAP_SIZE})
	}
	return append(programs, runHeapProgram{"test-heap-cap-near.cx", constants.INIT_HEAP_SIZE, 512 << 10})
}

// runHeapCompile compiles the test program path with the heap sizes initHeap
// and heapCap into a fresh actions.AST, prepared to run.
func runHeapCompile(tb testing.TB, path string, initHeap types.Pointer, heapCap types.Pointer) {
	source, err := os.ReadFile(filepath.Join("..", "tests", path))
	if err != nil {
		tb.Fatal(err)
	}
	initSize, maxSize := constants.INIT_HEAP_SIZE, constants.MAX_HEAP_SIZE
	constants.INIT_HEAP_SIZE, constants.MAX_HEAP_SIZE = initHeap, heapCap
	defer func() { constants.INIT_HEAP_SIZE, constants.MAX_HEAP_SIZE = initSize, maxSize }()
	compileRunProgram(tb, runBenchProgram{name: path, source: string(source)})
	execute.RunPrepare(actions.AST)
}

// TestRunHeapAllocs fails when a heap heavy program makes a Go allocation or
// starts a Go collection while it runs (D42). The heap, the scratch of the
// collector and its root table are allocated by RunPrepare.
func TestRunHeapAllocs(t *testing.T) {
	initProgram()
	for _, p := range runHeapPrograms() {
		t.Run(fmt.Sprintf("%s/heap-initial-%d", p.path, p.initHeap), func(t *testing.T) {
			discardStdout(t)
			// The run shrinks the heap no lower than INIT_HEAP_SIZE, as cx does.
			initSize := constants.INIT_HEAP_SIZE
			defer func() { constants.INIT_HEAP_SIZE = initSize }()
			const runs = 2
			var compiled []*ast.CXProgram
			for i := 0; i <= runs; i++ {
				runHeapCompile(t, p.path, p.initHeap, p.heapCap)
				compiled = append(compiled, actions.AST)
			}
			constants.INIT_HEAP_SIZE = p.initHeap
			next := 0
			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			gcPercent := debug.SetGCPercent(-1)
			allocs := testing.AllocsPerRun(runs, func() {
				if err := execute.RunCompiled(compiled[next], 0, nil); err != nil {
					t.Fatal(err)
				}
				if compiled[next].RuntimeError.RuntimeErrorRecordRaised {
					t.Fatalf("runtime error %d", compiled[next].RuntimeError.RuntimeErrorRecordCode)
				}
				next++
			})
			debug.SetGCPercent(gcPercent)
			runtime.ReadMemStats(&after)
			if allocs != 0 {
				t.Errorf("%.0f allocations per run, want 0 (D42)", allocs)
			}
			if after.NumGC != before.NumGC {
				t.Errorf("%d Go GC cycles during the runs, want 0", after.NumGC-before.NumGC)
			}
		})
	}
}

// A slice past the heap cap stops the run with an out of memory error that
// names the request and the cap, and a serialized program keeps its cap.
func TestRunHeapCapExceeded(t *testing.T) {
	initProgram()
	discardStdout(t)
	const heapCap = 64 << 10
	runHeapCompile(t, "test-heap-cap-exceeded.cx", constants.INIT_HEAP_SIZE, heapCap)
	if err := execute.RunCompiled(actions.AST, 0, nil); err != nil {
		t.Fatal(err)
	}
	record := actions.AST.RuntimeError
	if !record.RuntimeErrorRecordRaised || record.RuntimeErrorRecordCode != types.CX_RUNTIME_HEAP_EXHAUSTED_ERROR {
		t.Fatalf("runtime error %v code %d, want CX_RUNTIME_HEAP_EXHAUSTED_ERROR", record.RuntimeErrorRecordRaised, record.RuntimeErrorRecordCode)
	}
	outOfMemory, ok := record.RuntimeErrorRecordPanicValue.(ast.RuntimeOutOfMemory)
	want := uint64(types.OBJECT_HEADER_SIZE+constants.SLICE_HEADER_SIZE) + 20000*4
	if !ok || outOfMemory.RuntimeOutOfMemoryCap != heapCap || outOfMemory.RuntimeOutOfMemoryRequest != want {
		t.Fatalf("panic value %#v, want a request of %d bytes and a cap of %d", record.RuntimeErrorRecordPanicValue, want, heapCap)
	}

	image, err := ast.SerializeCXProgram(actions.AST, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := ast.Deserialize(image, nil)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Heap.Max != heapCap {
		t.Fatalf("deserialized heap cap %d, want %d", loaded.Heap.Max, heapCap)
	}
}
