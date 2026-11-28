package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t0-cx-runtime-output"
	reflecttable "github.com/skycoin/skycoin-cx/t2-cx-library-reflecttable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	"github.com/skycoin/skycoin-cx/t4-cx-library-compileropcodes"
)

// runCompilerImageMax is the largest image and source test-compiler-compile.cx
// holds in its buffers.
const runCompilerImageMax = 65536

// runCompilerFnv is the FNV-1a hash test-compiler-compile.cx prints.
func runCompilerFnv(data []byte) uint32 {
	hash := uint32(2166136261)
	for _, b := range data {
		hash = (hash ^ uint32(b)) * 16777619
	}
	return hash
}

// TestRunCompilerImagesMatchBuild compiles every single file program of
// tests/ that cx build --target=image accepts with sys/compiler from CX,
// test-compiler-compile.cx, and checks that its image is the file cx build
// writes, byte for byte, and that it runs with the output of cx (D152).
func TestRunCompilerImagesMatchBuild(t *testing.T) {
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	tests, err := filepath.Abs(filepath.Join("..", "tests"))
	if err != nil {
		t.Fatal(err)
	}
	programs, err := filepath.Glob(filepath.Join(tests, "*.cx"))
	if err != nil {
		t.Fatal(err)
	}
	examples, err := filepath.Glob(filepath.Join(tests, "examples", "*.cx"))
	if err != nil {
		t.Fatal(err)
	}
	programs = append(programs, examples...)
	dir := t.TempDir()
	env := []string{"CX_TEST_RUN_MAIN=1"}
	var compared, ran, skipped int
	var lock sync.Mutex
	limit := make(chan struct{}, runtime.NumCPU())
	var wait sync.WaitGroup
	for i, path := range programs {
		name, err := filepath.Rel(tests, path)
		if err != nil {
			t.Fatal(err)
		}
		name = filepath.ToSlash(name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(filepath.Base(name), "test-compiler-") || bytes.Contains(source, []byte("sys/os")) || len(source) >= runCompilerImageMax {
			continue
		}
		wait.Add(1)
		limit <- struct{}{}
		go func() {
			defer wait.Done()
			defer func() { <-limit }()
			output := filepath.Join(dir, fmt.Sprintf("program-%d.cximage", i))
			build, _ := buildTestProcess(t, executable, env, tests, buildCommand, "--target="+buildTargetImage, name, "-o", output)
			image, err := os.ReadFile(output)
			if build.exitCode != valuetypes.CXSuccess || err != nil || len(image) >= runCompilerImageMax {
				lock.Lock()
				skipped++
				lock.Unlock()
				return
			}
			compile := executionRecordProcess(t, executable, env, tests, string(source), "--enable=compiler,cxir", "test-compiler-compile.cx", "++"+name)
			want := fmt.Sprintf("file 0, image true, errors 0\nimage %d bytes, hash %d\n", len(image), runCompilerFnv(image))
			if !bytes.HasPrefix(compile.stdout, []byte(want)) {
				t.Errorf("%s: sys/compiler printed\n%s\nwant it to start with\n%s", name, executionRecordClip(compile.stdout), want)
				return
			}
			run := executionRecordProcess(t, executable, env, tests, "", name)
			lock.Lock()
			compared++
			lock.Unlock()
			if run.timedOut || run.exitCode != valuetypes.CXSuccess {
				return
			}
			got := bytes.TrimPrefix(compile.stdout, []byte(want))
			if wantRun := string(run.stdout) + "run 0\n"; string(got) != wantRun {
				line, gotLine, wantLine := executionRecordFirstDifference(got, []byte(wantRun))
				t.Errorf("%s: the image sys/compiler made prints otherwise than cx at line %d:\n got %q\nwant %q", name, line, gotLine, wantLine)
				return
			}
			lock.Lock()
			ran++
			lock.Unlock()
		}()
	}
	wait.Wait()
	t.Logf("%d images equal to cx build --target=image, %d of them ran with the output of cx, %d programs have no image", compared, ran, skipped)
	if compared < 100 {
		t.Errorf("only %d programs compared, want at least 100", compared)
	}
}

// executionRecordClip cuts long output for an error message.
func executionRecordClip(out []byte) string {
	if len(out) > 2000 {
		return string(out[:2000]) + "..."
	}
	return string(out)
}

// runCompilerErrorsSource has two type errors, at lines 5 and 6.
const runCompilerErrorsSource = `package main

func main() {
	var x i32 = 1
	var y bool = x
	z = 2
	printf("%v\n", y)
}
`

// runCompilerPrepare makes the library state of a run with the default
// libraries and the Go compiler of cx, and its store.
func runCompilerPrepare(t *testing.T, table []byte) *ir.ImageStore {
	_, operations, problem := librariesEnabledParse("")
	if problem != "" {
		t.Fatal(problem)
	}
	store := ir.ImageStoreNew(ir.ImageStoreRoomDefault)
	compileropcodes.LibraryCompilerSet(compilerLibrary{})
	compileropcodes.LibraryPrepare(&operations, output.ModeBatch, table, store)
	return store
}

// TestRunCompilerErrors checks the count, text and line of the errors of a
// program with type errors and of one with a syntax error.
func TestRunCompilerErrors(t *testing.T) {
	initProgram()
	store := runCompilerPrepare(t, nil)
	state := &compileropcodes.Library
	name := store.ImageStoreStringPoolTextAppend([]byte("errors.cx"))
	state.SourceFileAppend(store, name, []byte(runCompilerErrorsSource), int32(len(runCompilerErrorsSource)))
	ast := state.SourceParse(store)
	if ast <= 0 || state.ProgramTypeCheck(ast) != 2 || state.ErrorCountGet() != 2 {
		t.Fatalf("parse gives %#x with %d errors, %d kept, want 2", ast, state.ProgramTypeCheck(ast), state.ErrorCountGet())
	}
	want := []struct {
		line int32
		text string
	}{
		{5, "error: errors.cx:5:15 cannot assign value of type 'i32' to identifier 'y' of type 'bool'"},
		{6, "error: errors.cx:6:2 identifier 'z' does not exist"},
	}
	buffer := make([]byte, 256)
	for i, w := range want {
		size := state.ErrorMessageRead(int32(i), buffer)
		if got := string(buffer[:max(size, 0)]); got != w.text || state.ErrorSourceLineGet(int32(i)) != w.line {
			t.Errorf("error %d at line %d: %q, want line %d: %q", i, state.ErrorSourceLineGet(int32(i)), got, w.line, w.text)
		}
	}
	if state.ProgramLower(store, ast) != -1 {
		t.Errorf("a program with errors lowered")
	}
	broken := "package main\nfunc main( {\n"
	state.SourceFileAppend(store, name, []byte(broken), int32(len(broken)))
	if got := state.SourceParse(store); got != -1 || state.ErrorCountGet() != 1 || state.ErrorSourceLineGet(0) != 2 {
		t.Errorf("a syntax error parses to %#x with %d errors at line %d", got, state.ErrorCountGet(), state.ErrorSourceLineGet(0))
	}
	size := state.ErrorMessageRead(0, buffer)
	if got := string(buffer[:max(size, 0)]); got != "error: errors.cx:2:12 syntax error: unexpected {, expected ( or ) or name" {
		t.Errorf("syntax error text %q", got)
	}
}

// runCompilerAstSource is a program with a struct, a method, two functions
// and two globals.
const runCompilerAstSource = `package main

type Point struct {
	X i32
	Y i64
}

var Origin Point
var count i32

func (p Point) Sum() (s i32) {
	s = p.X
}

func add(a i32, b i32) (c i32) {
	c = a + b
}

func main() {
	global count
	count = add(1, 2)
}
`

// TestRunVmCompilerAllocs runs every opcode that reads an AST or an error
// through the vm form, on blocks laid out as the lowering lays them out,
// and fails when one allocates after the first read added the types.
func TestRunVmCompilerAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	var builder reflecttable.Builder
	builder.BuilderMetaRoom = true
	table := builder.BuilderEncode()
	store := runCompilerPrepare(t, nil)
	state := &compileropcodes.Library
	state.SourceFileAppend(store, store.ImageStoreStringPoolTextAppend([]byte("ast.cx")), []byte(runCompilerAstSource), int32(len(runCompilerAstSource)))
	ast := state.SourceParse(store)
	if ast <= 0 {
		t.Fatal("the program does not parse")
	}
	memory := make([]byte, 64<<10)
	const tableAt, blockAt, bufferAt = 4096, 0, 2048
	copy(memory[tableAt:], table)
	run := func(op ir.Opcode, inputs ...int64) int64 {
		at := uint32(blockAt)
		for _, input := range inputs {
			switch {
			case input == -2:
				valuetypes.PtrWrite(memory, valuetypes.Pointer(at), valuetypes.Pointer(bufferAt))
				valuetypes.PtrWrite(memory, valuetypes.Pointer(at+uint32(valuetypes.PointerSize)), 256)
				at += 2 * uint32(valuetypes.PointerSize)
			case input > 1<<32:
				valuetypes.I64Write(memory, valuetypes.Pointer(at), input)
				at += 8
			default:
				valuetypes.I32Write(memory, valuetypes.Pointer(at), int32(input))
				at += 4
			}
		}
		compileropcodes.VmOpcodeRun(op, memory, blockAt, tableAt, store)
		return valuetypes.I64Read(memory, valuetypes.Pointer(at))
	}
	reads := func() {
		run(ir.OpcodeCompilerCxCompilerProgramTypeCheck, ast)
		run(ir.OpcodeCompilerCxAstPackageCountGet, ast)
		run(ir.OpcodeCompilerCxAstPackageNameGet, ast, 0)
		run(ir.OpcodeCompilerCxAstPackageFunctionCountGet, ast, 0)
		run(ir.OpcodeCompilerCxAstPackageFunctionGet, ast, 0, 1)
		run(ir.OpcodeCompilerCxAstPackageStructCountGet, ast, 0)
		run(ir.OpcodeCompilerCxAstPackageStructGet, ast, 0, 0)
		run(ir.OpcodeCompilerCxAstPackageGlobalCountGet, ast, 0)
		run(ir.OpcodeCompilerCxAstPackageGlobalNameGet, ast, 0, 1)
		run(ir.OpcodeCompilerCxCompilerErrorCountGet)
		run(ir.OpcodeCompilerCxCompilerErrorMessageRead, 0, -2)
		run(ir.OpcodeCompilerCxCompilerErrorSourceLineGet, 0)
	}
	strct := run(ir.OpcodeCompilerCxAstPackageStructGet, ast, 0, 0)
	if fields := reflecttable.StructFieldCountGet(memory[tableAt:], strct); strct == 0 || fields != 2 {
		t.Fatalf("struct handle %#x with %d fields in the table of the machine", strct, fields)
	}
	if allocs := testing.AllocsPerRun(10, reads); allocs != 0 {
		t.Errorf("%.0f allocations per round of reads, want 0", allocs)
	}
	if count := int32(run(ir.OpcodeCompilerCxAstPackageGlobalCountGet, ast, 0)); count != 2 {
		t.Errorf("globals %d, want 2", count)
	}
}

// TestBuildRefusesCompiler checks that cx build makes no executable, image
// or native executable of a program that calls sys/compiler.
func TestBuildRefusesCompiler(t *testing.T) {
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	tests, err := filepath.Abs(filepath.Join("..", "tests"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	env := []string{"CX_TEST_RUN_MAIN=1"}
	for target, refusal := range map[string]string{
		"core":           buildCoreCompilerRefusal,
		buildTargetImage: "the program uses the compiler library, which cx-core-image does not link",
		buildTargetLlvm:  buildLlvmCompilerRefusal,
	} {
		args := []string{buildCommand, "--enable=compiler,cxir", "test-compiler-compile.cx", "-o", filepath.Join(dir, target)}
		if target != "core" {
			args = append([]string{buildCommand, "--target=" + target}, args[1:]...)
		}
		build, stderr := buildTestProcess(t, executable, env, tests, args...)
		if build.exitCode == valuetypes.CXSuccess || !bytes.Contains(stderr, []byte(refusal)) {
			t.Errorf("cx build for %s exits %d with %q, want the refusal %q", target, build.exitCode, stderr, refusal)
		}
	}
}

// TestRunCompilerImagesInOneRun compiles several programs one after the
// other in one process, as a compiler loop in CX does, and checks each
// image against the file a fresh cx build --target=image writes. The
// compiler state of the run and of each compile stay apart (D152).
func TestRunCompilerImagesInOneRun(t *testing.T) {
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	tests, err := filepath.Abs(filepath.Join("..", "tests"))
	if err != nil {
		t.Fatal(err)
	}
	initProgram()
	store := runCompilerPrepare(t, nil)
	state := &compileropcodes.Library
	encoded := make([]byte, 1<<20)
	dir := t.TempDir()
	for i, name := range []string{"examples/testing-packages.cx", "examples/hello-world.cx", "issue-218.cx", "test-var-grouped.cx", "examples/fibonacci-recursive.cx", "examples/testing-packages.cx"} {
		output := filepath.Join(dir, fmt.Sprintf("program-%d.cximage", i))
		if build, stderr := buildTestProcess(t, executable, []string{"CX_TEST_RUN_MAIN=1"}, tests, buildCommand, "--target="+buildTargetImage, name, "-o", output); build.exitCode != valuetypes.CXSuccess {
			t.Fatalf("%s: cx build exits %d: %s", name, build.exitCode, stderr)
		}
		want, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		source, err := os.ReadFile(filepath.Join(tests, name))
		if err != nil {
			t.Fatal(err)
		}
		state.SourceFileAppend(store, store.ImageStoreStringPoolTextAppend([]byte(name)), source, int32(len(source)))
		image := state.ProgramCompile(store)
		size := store.ImageStoreFileEncode(nil, image, encoded)
		if size < 0 || !bytes.Equal(encoded[:size], want) {
			t.Errorf("%s: compile %d of the run gives %d bytes, cx build %d", name, i, size, len(want))
		}
		if store.ImageStoreImageRelease(image) != 0 {
			t.Errorf("%s: the image does not release", name)
		}
	}
}
