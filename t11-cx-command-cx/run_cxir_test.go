package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-cx-compiler-astbuild"
	"github.com/skycoin/skycoin-cx/t4-cx-library-cxiropcodes"
	"github.com/skycoin/skycoin-cx/t4-cx-runtime-interpreter"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
)

// runCxirReadSource prints what it reads of its own image: the function
// count, the entries and every instruction and line of add.
const runCxirReadSource = `package main

import "sys/cxir"

func add(a i32, b i32) (c i32) {
	c = a + b
}

func main() {
	var image i64 = cxir.CxProgramImageCurrentGet()
	var count i32 = cxir.CxProgramImageFunctionCountGet(image)
	var name [3]ui8
	name[0] = 97UB
	name[1] = 100UB
	name[2] = 100UB
	var target i64 = cxir.CxStringPoolTextAppend(name, 3)
	var found i32 = -1
	var i i32
	for i = 0; i < count; i++ {
		if cxir.CxProgramFunctionNameGet(image, i) == target {
			found = i
		}
	}
	printf("functions %d main %d init %d add %d\n", count, cxir.CxProgramImageEntryMainFunctionGet(image), cxir.CxProgramImageEntryInitFunctionGet(image), found)
	var in cxir.CxInstruction
	var line cxir.CxSourceLine
	var status i32
	var pc i32
	for pc = 0; pc < cxir.CxProgramFunctionInstructionCountGet(image, found); pc++ {
		status = cxir.CxProgramFunctionInstructionRead(image, found, pc, in)
		status = cxir.CxSourceLineAtProgramCounterRead(image, found, pc, line) + status
		printf("%d %d %d %d %d %d %d %d %d %d line %d status %d\n", pc, in.Opcode, in.Mode, in.A, in.B, in.C, in.D, in.N, in.M, in.Aux, line.Number, status)
	}
	printf("sum %d\n", add(2, 3))
}
`

// TestRunCxirCurrentMatchesBuiltImage checks what a program reads of its
// running image on vm against the image cx build --target=image writes.
func TestRunCxirCurrentMatchesBuiltImage(t *testing.T) {
	initProgram()
	runCxirCompile(t, runCxirReadSource)
	path := filepath.Join(t.TempDir(), "read.cximage")
	if code := buildImage(buildOptions{buildOptionsOutput: path, buildOptionsDepthQuiet: true}); code != 0 {
		t.Fatalf("cx build --target=image exits %d", code)
	}
	file, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	built, problem := ir.ImageFileDecode(file)
	if problem != ir.ImageFileProblemNone {
		t.Fatal(ir.ImageFileProblemMessages[problem])
	}
	store := ir.ImageStoreNew(ir.ImageStoreRoomDefault)
	again := make([]byte, len(file))
	if size := store.ImageStoreFileEncode(nil, store.ImageStoreFileDecode(file), again); size != int32(len(file)) || string(again) != string(file) {
		t.Errorf("the built image decoded into a store encodes to %d bytes that differ from its %d", size, len(file))
	}
	add := -1
	for i, name := range built.ImageFuncNames {
		if name == "add" {
			add = i
		}
	}
	want := fmt.Sprintf("functions %d main %d init %d add %d\n", len(built.ImageFuncs), int32(built.ImageEntryMain), int32(built.ImageEntryInit), add)
	for pc, in := range built.ImageFuncs[add].FuncCode {
		line, _ := ir.LineAt(&built, uint32(add), uint32(pc))
		want += fmt.Sprintf("%d %d %d %d %d %d %d %d %d %d line %d status 0\n", pc, in.InstrOpcode, in.InstrMode, in.InstrA, in.InstrB, in.InstrC, in.InstrD, in.InstrN, in.InstrM, in.InstrAux, line.LineNumber)
	}
	want += "sum 5\n"

	runCxirCompile(t, runCxirReadSource)
	image, _, outside := core.ExecutorLower(astbuild.AST)
	if outside != "" {
		t.Fatalf("does not lower: %s", outside)
	}
	machine := vm.MachineNew(&image, core.ExecutorOutputSize)
	if stop, _ := machine.Run(image.ImageEntryMain, 0); stop != vm.StopReturn {
		t.Fatalf("stop %d", stop)
	}
	if got := string(machine.MachineOutput); got != want {
		t.Errorf("the program reads\n%s\nthe built image holds\n%s", got, want)
	}
	if !strings.Contains(want, "line ") || add < 0 {
		t.Errorf("found no add in the built image")
	}
}

// runCxirSource calls every cxir function in a loop: it reads the running
// image and an image it builds, writes and appends instructions, encodes,
// decodes and runs the image, which prints a line each time.
const runCxirSource = `package main

import "sys/cxir"

func main() {
	var name [4]ui8
	name[0] = 109UB
	name[1] = 97UB
	name[2] = 105UB
	name[3] = 110UB
	var file [1024]ui8
	var data [4]ui8
	data[0] = 42UB
	var in cxir.CxInstruction
	var slot cxir.CxFrameSlot
	var local cxir.CxFrameLocal
	var line cxir.CxSourceLine
	var total i32
	var i i32
	var mainName i64 = cxir.CxStringPoolTextAppend(name, 4)
	var current i64 = cxir.CxProgramImageCurrentGet()
	var image i64 = cxir.CxProgramImageCreate()
	var function i32 = cxir.CxProgramFunctionAppend(image, mainName, 0)
	var spare i32 = cxir.CxProgramFunctionAppend(image, mainName, 0)
	total = cxir.CxProgramImageEntryMainFunctionSet(image, function) + cxir.CxProgramImageDataWrite(image, 0, data)
	in.Opcode = i32.ui16(cxir.CxOpcodeByNameGet(cxir.CxStringPoolTextAppend([8]ui8{73UB, 51UB, 50UB, 80UB, 114UB, 105UB, 110UB, 116UB}, 8)))
	in.A = i32.ui32(cxir.CxProgramImageDataStartGet(image))
	total = total + cxir.CxProgramFunctionInstructionAppend(image, function, in)
	in.Opcode = 1UH
	in.A = 0U
	total = total + cxir.CxProgramFunctionInstructionAppend(image, function, in)
	for i = 0; i < 100; i++ {
		total = total + cxir.CxProgramImageFunctionCountGet(current) + cxir.CxProgramImageEntryMainFunctionGet(current) + cxir.CxProgramImageEntryInitFunctionGet(current)
		total = total + cxir.CxProgramImageStackSizeInBytesGet(image) + cxir.CxProgramImageDataStartGet(current) + cxir.CxProgramImageDataRead(image, 0, data)
		total = total + cxir.CxProgramImageDataWrite(image, 0, data)
		total = total + cxir.CxProgramFunctionFrameSizeInBytesGet(current, 0) + cxir.CxProgramFunctionInputCountGet(current, 0) + cxir.CxProgramFunctionOutputCountGet(current, 0) + cxir.CxProgramFunctionLocalCountGet(current, 0)
		total = total + cxir.CxProgramFunctionInputSlotRead(current, 0, 0, slot) + cxir.CxProgramFunctionOutputSlotRead(current, 0, 0, slot) + cxir.CxProgramFunctionLocalRead(current, 0, 0, local)
		total = total + cxir.CxProgramFunctionInstructionCountGet(current, 0) + cxir.CxProgramFunctionInstructionRead(image, function, 1, in)
		total = total + cxir.CxProgramFunctionInstructionWrite(image, function, 1, in) + cxir.CxProgramFunctionInstructionAppend(image, spare, in)
		total = total + cxir.CxProgramFunctionAppend(image, mainName, 8) + cxir.CxSourceLineAtProgramCounterRead(current, 0, 0, line)
		if cxir.CxProgramFunctionNameGet(current, 0) == cxir.CxSourceFileNameGet(current, 0) || cxir.CxProgramLocalNameGet(current, 0) == 0L || cxir.CxOpcodeLibraryRequiredGet(1) {
			total = total + 1
		}
		total = total + cxir.CxStringPoolTextRead(mainName, name) + cxir.CxOpcodeByNameGet(mainName)
		total = total + cxir.CxProgramImageFileEncode(image, file)
		if cxir.CxProgramImageFileDecode(file, 320) == 0L {
			total = total + 1
		}
		total = total + cxir.CxProgramImageRun(image)
	}
	printf("%d\n", total)
}
`

// runCxirCompile compiles source with the cxir library enabled.
func runCxirCompile(tb testing.TB, source string) *ast.CXProgram {
	compileRunProgram(tb, runBenchProgram{name: "cxir", source: source})
	astbuild.AST.ProgramLibrariesEnabled[opcodetable.LibraryNumberCxir] = true
	core.LibrariesEnabledResolve(astbuild.AST)
	return astbuild.AST
}

// TestRunCxirAllocs fails when a cxir function, a run of an image among
// them, allocates Go memory on the old loop. The store, its room and the
// machine of a run are made before the run (D153).
func TestRunCxirAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	discardStdout(t)
	const runs = 3
	var compiled []*ast.CXProgram
	var states []cxiropcodes.LibraryState
	for i := 0; i <= runs; i++ {
		prgrm := runCxirCompile(t, runCxirSource)
		image, _, outside := core.ExecutorLower(prgrm)
		if outside != "" {
			t.Fatalf("does not lower: %s", outside)
		}
		interpreter.RunPrepare(prgrm)
		compiled = append(compiled, prgrm)
		cxiropcodes.LibraryPrepare(&image, &prgrm.ProgramOperationsEnabled)
		states = append(states, cxiropcodes.Library)
	}
	next := 0
	allocs := testing.AllocsPerRun(runs, func() {
		cxiropcodes.Library = states[next]
		if err := interpreter.RunCompiled(compiled[next], 0, nil); err != nil {
			t.Fatal(err)
		}
		next++
	})
	if allocs != 0 {
		t.Errorf("%.0f allocations per run of 100 iterations, want 0", allocs)
	}
}

// TestRunVmCxirAllocs fails when the cxir functions do not lower or
// allocate on vm, the run of an image included.
func TestRunVmCxirAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	runCxirCompile(t, runCxirSource)
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
	if lines := strings.Count(string(machine.MachineOutput), "42\n"); lines != 100 {
		t.Errorf("the image printed %d times in the last run, want 100", lines)
	}
}
