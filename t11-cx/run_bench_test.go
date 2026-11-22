package main

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t2-globals"
	"github.com/skycoin/skycoin-cx/t3-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-execute"
	"github.com/skycoin/skycoin-cx/t4-opcodes"
	cxinit "github.com/skycoin/skycoin-cx/t7-init"
	cxparsing "github.com/skycoin/skycoin-cx/t9-cxparsing"
)

// runBenchProgram is a CX program BenchmarkRun and TestRunAllocs execute.
// maxAllocs is its ceiling of allocations per run.
type runBenchProgram struct {
	name      string
	source    string
	maxAllocs float64
}

const runCountSource = `package main

func main() {
	var sum i32
	var i i32
	for i = 0; i < 100000; i++ {
		sum = sum + i
	}
}
`

const runFibSource = `package main

func fib(n i32) (out i32) {
	if n < 2 {
		out = n
	} else {
		out = fib(n-1) + fib(n-2)
	}
}

func main() {
	var r i32
	r = fib(18)
}
`

const runBubbleSource = `package main

func main() {
	var nums []i32
	var i i32
	for i = 0; i < 150; i++ {
		nums = append(nums, 150-i)
	}
	var j i32
	var t i32
	for i = 0; i < 149; i++ {
		for j = 0; j < 149-i; j++ {
			if nums[j] > nums[j+1] {
				t = nums[j]
				nums[j] = nums[j+1]
				nums[j+1] = t
			}
		}
	}
}
`

const runArraySource = `package main

func main() {
	var nums [150]i32
	var i i32
	for i = 0; i < 150; i++ {
		nums[i] = 150 - i
	}
	var j i32
	var t i32
	for i = 0; i < 149; i++ {
		for j = 0; j < 149-i; j++ {
			if nums[j] > nums[j+1] {
				t = nums[j]
				nums[j] = nums[j+1]
				nums[j+1] = t
			}
		}
	}
}
`

const runStructSource = `package main

type point struct {
	x i32
	y i32
}

func main() {
	var points [100]point
	var i i32
	for i = 0; i < 100; i++ {
		points[i].x = i
		points[i].y = 2 * i
	}
	var sum i32
	var p *point
	var round i32
	for round = 0; round < 100; round++ {
		for i = 0; i < 100; i++ {
			p = &points[i]
			sum = sum + p.x + p.y
		}
	}
}
`

// runBenchPrograms are count, fib, bubble sort, the fixed array sort and the
// struct walk plus the BenchmarkCompile programs. test-gc is left out because it panics in MarkAndCompact (#13).
// The ceilings are the counts measured when the test was added. The target
// is 0, see #407.
func runBenchPrograms(tb testing.TB) []runBenchProgram {
	programs := []runBenchProgram{
		{"count", runCountSource, 7},
		{"fib", runFibSource, 6},
		{"bubble", runBubbleSource, 7},
		{"array", runArraySource, 7},
		{"struct", runStructSource, 7},
	}
	ceilings := map[string]float64{
		"examples/hello-world.cx":                           9,
		"examples/bubble-sort.cx":                           21,
		"examples/fibonacci-recursive.cx":                   9,
		"test-array.cx":                                     13,
		"test-slices.cx":                                    8,
		"examples/testing-pointers-to-arrays-of-structs.cx": 6,
	}
	for _, programPath := range compileBenchmarkPrograms {
		if programPath == "test-gc.cx" {
			continue
		}
		source, err := os.ReadFile(filepath.Join("..", "tests", programPath))
		if err != nil {
			tb.Fatal(err)
		}
		programs = append(programs, runBenchProgram{filepath.Base(programPath), string(source), ceilings[programPath]})
	}
	return programs
}

// compileRunProgram compiles source into a fresh actions.AST, ready to run.
func compileRunProgram(tb testing.TB, p runBenchProgram) {
	compileRunSources(tb, []string{p.name}, []string{p.source})
}

// compileRunSources compiles the files of one program into a fresh
// actions.AST, ready to run.
func compileRunSources(tb testing.TB, names []string, sources []string) {
	types.Allocator = ast.AllocateSeq
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Also state.CXProgramMng.CXProgramRelease(gid), phase 4 #371. Site S1780 S1781.
	actions.AST = cxinit.MakeProgram()
	cxinit.RegisterPackages(actions.AST)
	opcodes.RegisterOpcodes(actions.AST)
	globals.FoundCompileErrors = false
	globals.HeapOffset = 0
	if err := cxparsing.ParseSourceStrings(sources, names); err != nil {
		tb.Fatalf("%s: %v", names, err)
	}
	if err := cxparsing.AddInitFunction(actions.AST); err != nil {
		tb.Fatalf("%s: %v", names, err)
	}
	actions.AST.CxIrState.CxIrStateManagerGrowBoundary()
	actions.AST.CxIrState.StringPool.StringPoolFreeze()
}

// discardStdout points os.Stdout at the null device until the test ends.
func discardStdout(tb testing.TB) {
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		tb.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = null
	tb.Cleanup(func() {
		os.Stdout = stdout
		null.Close()
	})
}

// BenchmarkRun times execute.RunCompiled alone. Each iteration runs a fresh
// compile, and the compile is not timed.
func BenchmarkRun(b *testing.B) {
	initProgram()
	for _, p := range runBenchPrograms(b) {
		b.Run(p.name, func(b *testing.B) {
			discardStdout(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				compileRunProgram(b, p)
				b.StartTimer()
				if err := execute.RunCompiled(actions.AST, 0, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestRunAllocs fails when the allocations of execute.RunCompiled go above
// the recorded ceiling. Each run gets its own compile, made before
// AllocsPerRun starts, so it is a first run as cx makes it. The target is 0
// allocations per run (#407).
func TestRunAllocs(t *testing.T) {
	const runs = 3
	// A collection during a run adds allocations that vary with timing, for
	// example by emptying sync.Pool caches. #407 asks for no GC during a run.
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	for _, p := range runBenchPrograms(t) {
		t.Run(p.name, func(t *testing.T) {
			discardStdout(t)
			var compiled []*ast.CXProgram
			for i := 0; i <= runs; i++ {
				compileRunProgram(t, p)
				compiled = append(compiled, actions.AST)
			}
			next := 0
			allocs := testing.AllocsPerRun(runs, func() {
				if err := execute.RunCompiled(compiled[next], 0, nil); err != nil {
					t.Fatal(err)
				}
				next++
			})
			t.Logf("%s: %.0f allocations per run, ceiling %.0f", p.name, allocs, p.maxAllocs)
			if allocs > p.maxAllocs {
				t.Errorf("%s: %.0f allocations per run, ceiling %.0f (#407)", p.name, allocs, p.maxAllocs)
			}
		})
	}
}

// vmBenchMachine compiles and lowers p for vm. It returns nil when p does not
// lower.
func vmBenchMachine(tb testing.TB, p runBenchProgram) *vm.Machine {
	compileRunProgram(tb, p)
	image, _, outside := executorLower(actions.AST)
	if outside != "" {
		return nil
	}
	return vm.MachineNew(&image, executorOutputSize)
}

// BenchmarkRunVm times vm on the BenchmarkRun programs that lower, as cx
// runs them, output included. Each iteration resets the machine, which
// BenchmarkRun gets from a fresh compile.
func BenchmarkRunVm(b *testing.B) {
	initProgram()
	for _, p := range runBenchPrograms(b) {
		b.Run(p.name, func(b *testing.B) {
			machine := vmBenchMachine(b, p)
			if machine == nil {
				b.Skipf("%s does not lower", p.name)
			}
			discardStdout(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if code := executorVmRunMachine(actions.AST, machine); code != types.CX_SUCCESS {
					b.Fatalf("%s: code %d", p.name, code)
				}
			}
		})
	}
}

// TestRunVmAllocs fails when a run on vm allocates (#407, #408).
func TestRunVmAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	lowered := 0
	for _, p := range runBenchPrograms(t) {
		machine := vmBenchMachine(t, p)
		if machine == nil {
			continue
		}
		lowered++
		discardStdout(t)
		allocs := testing.AllocsPerRun(3, func() {
			if code := executorVmRunMachine(actions.AST, machine); code != types.CX_SUCCESS {
				t.Fatalf("%s: code %d", p.name, code)
			}
		})
		if allocs != 0 {
			t.Errorf("%s: %.0f allocations per run, want 0", p.name, allocs)
		}
	}
	if lowered < 2 {
		t.Errorf("%d of the run programs lower, count and fib at least should", lowered)
	}
}
