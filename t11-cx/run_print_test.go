package main

import (
	"runtime/debug"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t3-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-execute"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
)

const runPrintAtomicSource = `package main

func main() {
	var b bool
	b = true
	var i8v i8
	i8v = -8B
	var i16v i16
	i16v = -16H
	var i64v i64
	i64v = -64L
	var u8 ui8
	u8 = 8UB
	var u16 ui16
	u16 = 16UH
	var u32 ui32
	u32 = 32U
	var u64 ui64
	u64 = 64UL
	var f32v f32
	f32v = 1.5
	var f64v f64
	f64v = 0.1D
	var s str
	s = "text"
	var i i32
	for i = 0; i < 100; i++ {
		printf("%v %v %v %v %v %v %v %v %v %v %v %v\n", b, i8v, i16v, i, i64v, u8, u16, u32, u64, f32v, f64v, s)
	}
}
`

const runPrintCompositeSource = `package main

type point struct {
	x i32
	y f64
}

type shape struct {
	name str
	corner point
	sides [3]i32
	tags []str
	next *point
	ok bool
}

func main() {
	var p point
	p.x = 3
	p.y = 2.5D
	var sh shape
	sh.name = "tri"
	sh.corner = p
	sh.sides[0] = 1
	sh.sides[2] = 3
	sh.tags = append(sh.tags, "a")
	sh.tags = append(sh.tags, "b")
	sh.ok = true
	var a [3]i32
	a[1] = 7
	var m [2][2]i32
	m[1][0] = 5
	var sl []f64
	sl = append(sl, 1.25D)
	var np *point
	var ip *i32
	var out str
	var i i32
	for i = 0; i < 100; i++ {
		printf("%v|%v|%v|%v|%v|%v|%v|%v\n", p, sh, a, m, sl, sh.corner, sh.sides, np)
		out = sprintf("%v %v", sh, i)
		printf("%d\n", i, out, p.y, p, a, m, sl, np, ip, sh.ok)
	}
	np = &p
	ip = &p.x
	sh.next = np
	for i = 0; i < 100; i++ {
		printf("%v %v %v %+v\n", np, ip, &p, sh)
	}
}
`

const runPrintDebugSource = `package main

type point struct {
	x i32
	y f64
}

func twice(x i32, p point, s []f64) (y i32) {
	var a [2]i32
	a[1] = x
	y = x * 2
	var t str
	t = sprintf("%d", y)
	debug()
}

func main() {
	var p point
	p.x = 4
	var i i32
	var r i32
	var s []f64
	s = append(s, 1.5D)
	for i = 0; i < 50; i++ {
		r = twice(i, p, s)
	}
}
`

// runPrintPrograms print values with printf, sprintf and the call stack
// dump of the debug opcode.
var runPrintPrograms = []runBenchProgram{
	{name: "atomic", source: runPrintAtomicSource},
	{name: "composite", source: runPrintCompositeSource},
	{name: "debug", source: runPrintDebugSource},
}

// runPrintAllocs returns the allocations of one old loop run of p, each run
// on its own compile as in TestRunAllocs.
func runPrintAllocs(t *testing.T, p runBenchProgram) float64 {
	const runs = 3
	var compiled []*ast.CXProgram
	for i := 0; i <= runs; i++ {
		compileRunProgram(t, p)
		execute.RunPrepare(actions.AST)
		compiled = append(compiled, actions.AST)
	}
	next := 0
	return testing.AllocsPerRun(runs, func() {
		if err := execute.RunCompiled(compiled[next], 0, nil); err != nil {
			t.Fatal(err)
		}
		next++
	})
}

// TestRunPrintAllocs fails when printf or sprintf of %v and of extra
// arguments, or the call stack dump of debug, allocates on the old loop
// (#407, VMALLOC-015).
func TestRunPrintAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	discardStdout(t)
	for _, p := range runPrintPrograms {
		if allocs := runPrintAllocs(t, p); allocs != 0 {
			t.Errorf("%s: %.0f allocations per run, want 0", p.name, allocs)
		}
	}
}

// TestRunVmPrintAllocs fails when printf of atomic %v values does not lower
// or allocates on vm (#407).
func TestRunVmPrintAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	discardStdout(t)
	compileRunProgram(t, runBenchProgram{name: "atomic", source: runPrintAtomicSource})
	image, _, outside := core.ExecutorLower(actions.AST)
	if outside != "" {
		t.Fatalf("does not lower: %s", outside)
	}
	machine := vm.MachineNew(&image, core.ExecutorOutputSize)
	allocs := testing.AllocsPerRun(3, func() {
		if code := core.ExecutorVmRunMachine(actions.AST, machine); code != types.CX_SUCCESS {
			t.Fatalf("code %d", code)
		}
	})
	if allocs != 0 {
		t.Errorf("%.0f allocations per run, want 0", allocs)
	}
}

// BenchmarkRunPrint times the old loop runs of TestRunPrintAllocs.
func BenchmarkRunPrint(b *testing.B) {
	initProgram()
	for _, p := range runPrintPrograms {
		b.Run(p.name, func(b *testing.B) {
			discardStdout(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				compileRunProgram(b, p)
				execute.RunPrepare(actions.AST)
				b.StartTimer()
				if err := execute.RunCompiled(actions.AST, 0, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
