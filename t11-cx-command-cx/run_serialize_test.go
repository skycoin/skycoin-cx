package main

import (
	"runtime/debug"
	"testing"

	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-cx-compiler-astbuild"
	"github.com/skycoin/skycoin-cx/t4-cx-library-reflectopcodes"
	"github.com/skycoin/skycoin-cx/t4-cx-library-serializeopcodes"
	"github.com/skycoin/skycoin-cx/t4-cx-runtime-interpreter"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
)

// runSerializeSource encodes and decodes a struct with a nested struct, an
// array and a str field in a loop, and reads its encoded size.
const runSerializeSource = `package main

import "sys/reflect"
import "sys/serialize"

type Inner struct {
	flag  bool
	small i16
}

type Outer struct {
	id    i32
	inner Inner
	pair  [2]i64
	name  str
}

func main() {
	var value Outer
	var back Outer
	var buffer [64]ui8
	var t i64
	var total i32
	var i i32
	value.name = "outer"
	t = reflect.CxTypeOfValueGet(value)
	for i = 0; i < 100; i++ {
		value.id = i
		total = total + serialize.CxSerializeEncodedSizeInBytesGet(t)
		total = total + serialize.CxSerializeValueEncode(value, t, buffer)
		total = total + serialize.CxSerializeValueDecode(buffer, back, t)
	}
	printf("%d %d\n", total, back.id)
}
`

// runSerializeSliceSource encodes and decodes a slice in a loop, which only
// the old loop runs.
const runSerializeSliceSource = `package main

import "sys/reflect"
import "sys/serialize"

func main() {
	var numbers []i32
	var room []i32
	var buffer [64]ui8
	var t i64
	var total i32
	var i i32
	numbers = append(numbers, 1)
	numbers = append(numbers, 2)
	room = resize(room, 2)
	t = reflect.CxTypeOfValueGet(numbers)
	for i = 0; i < 100; i++ {
		total = total + serialize.CxSerializeValueEncode(numbers, t, buffer)
		total = total + serialize.CxSerializeValueDecode(buffer, room, t)
	}
	printf("%d\n", total)
}
`

// runSerializeCompile compiles source with the reflect and serialize
// libraries enabled.
func runSerializeCompile(tb testing.TB, source string) *ast.CXProgram {
	compileRunProgram(tb, runBenchProgram{name: "serialize", source: source})
	astbuild.AST.ProgramLibrariesEnabled[opcodetable.LibraryNumberReflect] = true
	astbuild.AST.ProgramLibrariesEnabled[opcodetable.LibraryNumberSerialize] = true
	core.LibrariesEnabledResolve(astbuild.AST)
	return astbuild.AST
}

// TestRunSerializeAllocs fails when a serialize function allocates Go
// memory on the old loop. The type table is built once before the run
// (decision D150).
func TestRunSerializeAllocs(t *testing.T) {
	for _, source := range []string{runSerializeSource, runSerializeSliceSource} {
		runSerializeAllocsCheck(t, source)
	}
}

func runSerializeAllocsCheck(t *testing.T, source string) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	discardStdout(t)
	const runs = 3
	var compiled []*ast.CXProgram
	for i := 0; i <= runs; i++ {
		prgrm := runSerializeCompile(t, source)
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
		if err := interpreter.RunCompiled(compiled[next], 0, nil); err != nil {
			t.Fatal(err)
		}
		next++
	})
	if allocs != 0 {
		t.Errorf("%.0f allocations per run of 100 iterations, want 0", allocs)
	}
}

// TestRunVmSerializeAllocs fails when the serialize functions do not lower
// or allocate on vm.
func TestRunVmSerializeAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	runSerializeCompile(t, runSerializeSource)
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
