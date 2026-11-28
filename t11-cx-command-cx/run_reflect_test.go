package main

import (
	"runtime/debug"
	"testing"

	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-cx-compiler-astbuild"
	"github.com/skycoin/skycoin-cx/t4-cx-library-reflectopcodes"
	"github.com/skycoin/skycoin-cx/t4-cx-runtime-interpreter"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
)

// runReflectSource calls every kind of reflect function in a loop, the
// reads of the table, a field copy each way and a name read.
const runReflectSource = `package main

import "sys/reflect"

type Pair struct {
	left  i32
	right [2]i64
}

func (p Pair) Sum() (total i64) {
	total = 0L
}

func main() {
	var pair Pair
	var buffer [16]ui8
	var t i64
	var total i32
	var i i32
	for i = 0; i < 100; i++ {
		t = reflect.CxTypeOfValueGet(pair)
		total = total + reflect.CxTypeKindGet(t) + reflect.CxStructFieldCountGet(t) + reflect.CxStructFieldOffsetInBytesGet(t, 1)
		total = total + reflect.CxArrayLengthGet(reflect.CxStructFieldTypeGet(t, 1))
		total = total + reflect.CxFunctionInputCountGet(reflect.CxStructMethodFunctionGet(t, 0))
		total = total + reflect.CxStructFieldOrdinalByNameGet(t, reflect.CxStructFieldNameGet(t, 1))
		total = total + reflect.CxStructFieldValueRead(pair, t, 0, buffer)
		total = total + reflect.CxStructFieldValueWrite(pair, t, 0, buffer)
		total = total + reflect.CxStringPoolTextRead(reflect.CxTypeNameGet(t), buffer)
	}
	printf("%d\n", total)
}
`

// runReflectCompile compiles runReflectSource with the reflect library
// enabled.
func runReflectCompile(tb testing.TB) *ast.CXProgram {
	compileRunProgram(tb, runBenchProgram{name: "reflect", source: runReflectSource})
	astbuild.AST.ProgramLibrariesEnabled[opcodetable.LibraryNumberReflect] = true
	core.LibrariesEnabledResolve(astbuild.AST)
	return astbuild.AST
}

// TestRunReflectAllocs fails when a reflect function allocates Go memory on
// the old loop. The table is built once before the run (decision D150).
func TestRunReflectAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	discardStdout(t)
	const runs = 3
	var compiled []*ast.CXProgram
	for i := 0; i <= runs; i++ {
		prgrm := runReflectCompile(t)
		interpreter.RunPrepare(prgrm)
		compiled = append(compiled, prgrm)
	}
	tables := make([]ast.ReflectTable, len(compiled))
	for i, prgrm := range compiled {
		tables[i] = ast.ReflectTableBuild(prgrm)
	}
	next := 0
	allocs := testing.AllocsPerRun(runs, func() {
		reflectopcodes.Library.LibraryTypeTable = tables[next]
		if err := interpreter.RunCompiled(compiled[next], 0, nil); err != nil {
			t.Fatal(err)
		}
		next++
	})
	if allocs != 0 {
		t.Errorf("%.0f allocations per run of 100 iterations, want 0", allocs)
	}
}

// TestRunVmReflectAllocs fails when the reflect functions do not lower or
// allocate on vm.
func TestRunVmReflectAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	runReflectCompile(t)
	image, _, outside := core.ExecutorLower(astbuild.AST)
	if outside != "" {
		t.Fatalf("does not lower: %s", outside)
	}
	machine := vm.MachineNew(&image, core.ExecutorOutputSize)
	allocs := testing.AllocsPerRun(3, func() {
		machine.Reset()
		if stop, _ := machine.Run(image.ImageEntryMain, 0); stop != vm.StopReturn {
			t.Fatalf("stop %d", stop)
		}
	})
	if allocs != 0 {
		t.Errorf("%.0f allocations per run, want 0", allocs)
	}
}
