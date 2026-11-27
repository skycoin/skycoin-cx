package main

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t2-cx-core-environment"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-cx-compiler-astbuild"
	"github.com/skycoin/skycoin-cx/t4-cx-runtime-interpreter"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
	"github.com/skycoin/skycoin-cx/t7-cx-runtime-libraries"
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
// struct walk plus the BenchmarkCompile programs. A run allocates nothing
// (#407), test-gc.cx and its collections included (D42).
func runBenchPrograms(tb testing.TB) []runBenchProgram {
	programs := []runBenchProgram{
		{"count", runCountSource, 0},
		{"fib", runFibSource, 0},
		{"bubble", runBubbleSource, 0},
		{"array", runArraySource, 0},
		{"struct", runStructSource, 0},
	}
	for _, programPath := range compileBenchmarkPrograms {
		source, err := os.ReadFile(filepath.Join("..", "tests", programPath))
		if err != nil {
			tb.Fatal(err)
		}
		programs = append(programs, runBenchProgram{filepath.Base(programPath), string(source), 0})
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
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Also state.CXProgramMng.CXProgramRelease(gid), phase 4 #371. Site S1780 S1781.
	astbuild.AST = libraries.ProgramMake()
	libraries.PackagesRegister(astbuild.AST)
	environment.FoundCompileErrors = false
	environment.HeapOffset = 0
	if err := compileSourceStrings(names, sources); err != nil {
		tb.Fatalf("%s: %v", names, err)
	}
	astbuild.AST.CxIrState.CxIrStateManagerGrowBoundary()
	astbuild.AST.CxIrState.StringPool.StringPoolFreeze()
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
				ast.ProgramMemoryRelease(astbuild.AST)
				compileRunProgram(b, p)
				interpreter.RunPrepare(astbuild.AST)
				b.StartTimer()
				if err := interpreter.RunCompiled(astbuild.AST, 0, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestRunAllocs fails when the allocations of execute.RunCompiled go above
// the ceiling, 0 (#407). Each run gets its own compile and
// RunPrepare, made before AllocsPerRun starts, so it is a first run as cx
// makes it.
func TestRunAllocs(t *testing.T) {
	const runs = 3
	// A collection during a run adds allocations that vary with timing, for
	// example by emptying sync.Pool caches. TestRunNoGC checks there is none.
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	for _, p := range append(runBenchPrograms(t), runComparePrograms(t)...) {
		t.Run(p.name, func(t *testing.T) {
			// With GC off, the programs of earlier subtests fill a 32-bit address space.
			runtime.GC()
			discardStdout(t)
			var compiled []*ast.CXProgram
			for i := 0; i <= runs; i++ {
				compileRunProgram(t, p)
				interpreter.RunPrepare(astbuild.AST)
				compiled = append(compiled, astbuild.AST)
			}
			next := 0
			allocs := testing.AllocsPerRun(runs, func() {
				if err := interpreter.RunCompiled(compiled[next], 0, nil); err != nil {
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
	image, _, outside := core.ExecutorLower(astbuild.AST)
	if outside != "" {
		return nil
	}
	return vm.MachineNew(&image, core.ExecutorOutputSize)
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
				if code := core.ExecutorVmRunMachine(astbuild.AST, machine); code != valuetypes.CXSuccess {
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
			if code := core.ExecutorVmRunMachine(astbuild.AST, machine); code != valuetypes.CXSuccess {
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

const runLibraryCallSource = `package main

import "sys/os"
import "sys/net"

func main() {
	var i i32
	var closed bool
	for i = 0; i < 1000; i++ {
		closed = os.Close(-1)
		closed = net.Close(-1)
	}
}
`

// TestRunVmLibraryCallAllocs fails when the file and net opcodes on vm
// allocate or do not lower, or when the loader does not refuse them for a
// run that does not enable their libraries (#407, #492, D119).
func TestRunVmLibraryCallAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	for _, enabled := range []bool{true, false} {
		compileRunProgram(t, runBenchProgram{name: "library", source: runLibraryCallSource})
		astbuild.AST.ProgramLibrariesEnabled[opcodetable.LibraryNumberFile] = enabled
		astbuild.AST.ProgramLibrariesEnabled[opcodetable.LibraryNumberNet] = enabled
		core.LibrariesEnabledResolve(astbuild.AST)
		if refused := core.OperationRefused(astbuild.AST) != 0; refused == enabled {
			t.Fatalf("enabled %v: refused %v", enabled, refused)
		}
		image, _, outside := core.ExecutorLower(astbuild.AST)
		if !enabled {
			if !strings.Contains(outside, "operation not enabled") {
				t.Fatalf("not enabled: lower gives %q, want the refusal", outside)
			}
			continue
		}
		if outside != "" {
			t.Fatalf("does not lower: %s", outside)
		}
		machine := vm.MachineNew(&image, core.ExecutorOutputSize)
		want := vm.StopReturn
		allocs := testing.AllocsPerRun(3, func() {
			machine.Reset()
			if stop, _ := machine.Run(image.ImageEntryMain, 0); stop != want {
				t.Fatalf("enabled %v: stop %d, want %d", enabled, stop, want)
			}
		})
		if allocs != 0 {
			t.Errorf("enabled %v: %.0f allocations per run, want 0", enabled, allocs)
		}
	}
}

// runCompareScale shrinks each t0-cx-tools-benchmarks/compare program to a size that
// runs in milliseconds on the old loop. The code paths, prints included, stay
// those of the full size, only the pass line prints FAIL.
var runCompareScale = map[string][][2]string{
	"count":           {{"i < 30000000", "i < 30000"}},
	"fib":             {{"r = fib(34)", "r = fib(12)"}},
	"array":           {{"round < 400", "round < 2"}},
	"struct":          {{"round < 80000", "round < 20"}},
	"memtest":         {{"n = 1048576", "n = 4096"}, {"walk = 65536", "walk = 256"}},
	"primes":          {{"n = 2000000", "n = 2000"}, {"k < 300000UL", "k < 300UL"}, {"round < 20", "round < 1"}},
	"ackermann":       {{"round < 350", "round < 2"}},
	"digitalroot":     {{"n < 1200000", "n < 1200"}},
	"factorialiter":   {{"round < 120000", "round < 2"}},
	"factorialrec":    {{"round < 50000", "round < 2"}},
	"structs":         {{"round < 60", "round < 1"}, {"size = 100000", "size = 100"}},
	"structserialize": {{"round < 76000", "round < 20"}},
}

// runComparePrograms are the t0-cx-tools-benchmarks/compare programs at the scale of
// runCompareScale. They allocate nothing during a run (#407).
func runComparePrograms(tb testing.TB) []runBenchProgram {
	var programs []runBenchProgram
	for _, name := range []string{"count", "fib", "array", "struct", "memtest", "primes", "ackermann", "digitalroot", "factorialiter", "factorialrec", "structs", "structserialize"} {
		source, err := os.ReadFile(filepath.Join("..", "t0-cx-tools-benchmarks", "compare", "cx", name+".cx"))
		if err != nil {
			tb.Fatal(err)
		}
		text := string(source)
		for _, replace := range runCompareScale[name] {
			if strings.Count(text, replace[0]) != 1 {
				tb.Fatalf("compare/%s.cx: %q is not there once, update runCompareScale", name, replace[0])
			}
			text = strings.Replace(text, replace[0], replace[1], 1)
		}
		programs = append(programs, runBenchProgram{"compare-" + name, text, 0})
	}
	return programs
}

// TestRunNoGC fails when a Go garbage collection runs while execute.RunCompiled
// runs a program that allocates nothing (#407). It collects first, so a cycle
// could only come from the run's own allocations.
func TestRunNoGC(t *testing.T) {
	initProgram()
	for _, p := range append(runBenchPrograms(t), runComparePrograms(t)...) {
		if p.maxAllocs != 0 {
			continue
		}
		t.Run(p.name, func(t *testing.T) {
			discardStdout(t)
			compileRunProgram(t, p)
			interpreter.RunPrepare(astbuild.AST)
			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			if err := interpreter.RunCompiled(astbuild.AST, 0, nil); err != nil {
				t.Fatal(err)
			}
			runtime.ReadMemStats(&after)
			if after.NumGC != before.NumGC {
				t.Errorf("%s: %d Go GC cycles during the run, want 0", p.name, after.NumGC-before.NumGC)
			}
		})
	}
}

// BenchmarkLoad compiles, prepares and runs a program end to end, as an
// embedding program loads one after another, and reports each stage.
func BenchmarkLoad(b *testing.B) {
	initProgram()
	for _, programPath := range []string{"examples/hello-world.cx", "test-array.cx", "test-slices.cx"} {
		source, err := os.ReadFile(filepath.Join("..", "tests", programPath))
		if err != nil {
			b.Fatal(err)
		}
		name := filepath.Base(programPath)
		b.Run(name, func(b *testing.B) {
			discardStdout(b)
			b.ReportAllocs()
			var compile, prepare, run time.Duration
			for i := 0; i < b.N; i++ {
				ast.ProgramMemoryRelease(astbuild.AST)
				compileStart := time.Now()
				compileRunSources(b, []string{name}, []string{string(source)})
				prepareStart := time.Now()
				interpreter.RunPrepare(astbuild.AST)
				runStart := time.Now()
				if err := interpreter.RunCompiled(astbuild.AST, 0, nil); err != nil {
					b.Fatal(err)
				}
				runEnd := time.Now()
				compile += prepareStart.Sub(compileStart)
				prepare += runStart.Sub(prepareStart)
				run += runEnd.Sub(runStart)
			}
			b.ReportMetric(float64(compile.Nanoseconds())/float64(b.N), "compile-ns/op")
			b.ReportMetric(float64(prepare.Nanoseconds())/float64(b.N), "prepare-ns/op")
			b.ReportMetric(float64(run.Nanoseconds())/float64(b.N), "run-ns/op")
		})
	}
}
