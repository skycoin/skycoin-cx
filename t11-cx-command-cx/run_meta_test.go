package main

import (
	"runtime/debug"
	"testing"

	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-cx-compiler-astbuild"
	"github.com/skycoin/skycoin-cx/t4-cx-library-metaopcodes"
	"github.com/skycoin/skycoin-cx/t4-cx-library-reflectopcodes"
	"github.com/skycoin/skycoin-cx/t4-cx-library-serializeopcodes"
	"github.com/skycoin/skycoin-cx/t4-cx-runtime-interpreter"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
)

// runMetaSource calls every meta function in a loop, a struct with a field
// and a method, an array, a slice and a pointer, a function signature and a
// package, and encodes a value of the new struct.
const runMetaSource = `package main

import "sys/meta"
import "sys/reflect"
import "sys/serialize"

func main() {
	var number i32
	var name [4]ui8
	var buffer [16]ui8
	var t i64
	var s i64
	var f i64
	var p i64
	var n i64
	var total i32
	var i i32
	name[0] = 97UB
	name[1] = 98UB
	t = reflect.CxTypeOfValueGet(number)
	for i = 0; i < 100; i++ {
		n = meta.CxStringPoolTextAppend(name, 2)
		f = meta.CxFunctionDefinitionBegin(n)
		total = total + meta.CxFunctionDefinitionInputAppend(f, n, t) + meta.CxFunctionDefinitionOutputAppend(f, 0L, meta.CxArrayTypeCreate(t, 2))
		total = total + meta.CxFunctionDefinitionFinish(f)
		s = meta.CxStructDefinitionBegin(n)
		total = total + meta.CxStructDefinitionFieldAppend(s, n, t) + meta.CxStructDefinitionMethodAppend(s, n, f)
		total = total + meta.CxStructDefinitionFinish(s)
		if meta.CxPointerTypeCreate(meta.CxSliceTypeCreate(s)) == 0L {
			total = total - 1
		}
		total = total + serialize.CxSerializeValueEncode(number, s, buffer)
		p = meta.CxPackageDefinitionBegin(n)
		total = total + meta.CxPackageDefinitionFinish(p)
	}
	printf("%d\n", total)
}
`

// runMetaCompile compiles source with the reflect, serialize and meta
// libraries enabled.
func runMetaCompile(tb testing.TB, source string) *ast.CXProgram {
	compileRunProgram(tb, runBenchProgram{name: "meta", source: source})
	astbuild.AST.ProgramLibrariesEnabled[opcodetable.LibraryNumberReflect] = true
	astbuild.AST.ProgramLibrariesEnabled[opcodetable.LibraryNumberSerialize] = true
	astbuild.AST.ProgramLibrariesEnabled[opcodetable.LibraryNumberMeta] = true
	core.LibrariesEnabledResolve(astbuild.AST)
	return astbuild.AST
}

// TestRunMetaAllocs fails when a meta function allocates Go memory on the
// old loop. The table and its room are built once before the run (D151).
func TestRunMetaAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	discardStdout(t)
	const runs = 3
	var compiled []*ast.CXProgram
	for i := 0; i <= runs; i++ {
		prgrm := runMetaCompile(t, runMetaSource)
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
		serializeopcodes.LibraryPrepare(tables[next])
		metaopcodes.LibraryPrepare(tables[next])
		if err := interpreter.RunCompiled(compiled[next], 0, nil); err != nil {
			t.Fatal(err)
		}
		next++
	})
	if allocs != 0 {
		t.Errorf("%.0f allocations per run of 100 iterations, want 0", allocs)
	}
}

// TestRunVmMetaAllocs fails when the meta functions do not lower or
// allocate on vm.
func TestRunVmMetaAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	runMetaCompile(t, runMetaSource)
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
