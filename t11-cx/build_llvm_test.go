package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	llvm "github.com/skycoin/skycoin-cx/t3-cx-compiler-llvm"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-actions"
	lower "github.com/skycoin/skycoin-cx/t4-cx-compiler-lower"
)

// TestLlvmOpcodeFacts checks that the generated table of the LLVM target
// describes every scalar image opcode.
func TestLlvmOpcodeFacts(t *testing.T) {
	inline, fallback := 0, 0
	for op := ir.OpcodeBoolNot; op < ir.OpcodeCount; op++ {
		fact := llvm.OpcodeFactOf(op)
		if fact.OpcodeFactName == "" {
			t.Errorf("image opcode %d has no row in the LLVM opcode table", op)
		}
		if fact.OpcodeFactTemplate != "" {
			inline++
		} else {
			fallback++
		}
	}
	for op := ir.OpcodeInvalid; op < ir.OpcodeBoolNot; op++ {
		if fact := llvm.OpcodeFactOf(op); fact.OpcodeFactName != "" {
			t.Errorf("image opcode %d is not scalar but has the row %q", op, fact.OpcodeFactName)
		}
	}
	t.Logf("scalar opcodes: %d inline by a template, %d call their runtime function", inline, fallback)
}

// llvmTestClang returns the C compiler the native tests use, CC or clang,
// and skips the test without one.
func llvmTestClang(t *testing.T) string {
	cc := os.Getenv("CC")
	if cc == "" {
		cc = "clang"
	}
	path, err := exec.LookPath(cc)
	if err != nil {
		t.Skipf("no %s to compile LLVM IR: %v", cc, err)
	}
	return path
}

// llvmTestValues are the operands each scalar opcode is run on.
func llvmTestValues(code types.Code) []uint64 {
	bits := uint64(8 * llvmTestSize(code))
	switch code {
	case types.BOOL:
		return []uint64{0, 1}
	case types.I8, types.I16, types.I32, types.I64:
		min := uint64(1) << (bits - 1)
		return []uint64{0, 1, ^uint64(0), 7, ^uint64(6), 3, min, min - 1, bits - 1, bits, bits + 1}
	case types.UI8, types.UI16, types.UI32, types.UI64:
		max := ^uint64(0) >> (64 - bits)
		return []uint64{0, 1, 2, 7, max, max - 1, bits - 1, bits}
	case types.F32:
		var list []uint64
		for _, v := range []float32{0, float32(math.Copysign(0, -1)), 1.5, -2.25, 3, 100.75, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), 1e30, 1e-7} {
			list = append(list, uint64(math.Float32bits(v)))
		}
		return list
	}
	var list []uint64
	for _, v := range []float64{0, math.Copysign(0, -1), 1.5, -2.25, 3, 100.75, math.NaN(), math.Inf(1), math.Inf(-1), 1e300, 1e-300} {
		list = append(list, math.Float64bits(v))
	}
	return list
}

func llvmTestSize(code types.Code) uint32 {
	switch code {
	case types.BOOL, types.I8, types.UI8:
		return 1
	case types.I16, types.UI16:
		return 2
	case types.I32, types.UI32, types.F32:
		return 4
	}
	return 8
}

// llvmTestSkip reports operands whose result Go leaves to the platform or
// that stop the run: a division by zero and a float out of the range of the
// integer it converts to.
func llvmTestSkip(fact llvm.OpcodeFact, a, b uint64) bool {
	code := fact.OpcodeFactType
	float := code == types.F32 || code == types.F64
	if !float && code != types.BOOL && fact.OpcodeFactInputs == 2 && (fact.OpcodeFactOperation == "div" || fact.OpcodeFactOperation == "mod") {
		return b&(^uint64(0)>>(64-8*llvmTestSize(code))) == 0
	}
	result := fact.OpcodeFactResult
	if !float || fact.OpcodeFactOperation != result.Name() || result == types.F32 || result == types.F64 {
		return false
	}
	v := math.Float64frombits(a)
	if code == types.F32 {
		v = float64(math.Float32frombits(uint32(a)))
	}
	signed := result == types.I8 || result == types.I16 || result == types.I32 || result == types.I64
	bits := 8 * llvmTestSize(result)
	low, high := 0.0, math.Ldexp(1, int(bits))
	if signed {
		low, high = -math.Ldexp(1, int(bits)-1), math.Ldexp(1, int(bits)-1)
	}
	return !(v > low-1 && v < high)
}

// llvmTestImage builds an image whose main calls one function per scalar
// opcode. Each runs its opcode on every pair of operands and prints the
// result, or prints the operand for a print opcode. With readOnly set the
// operands are literals of ImageReadOnly, which the target folds.
func llvmTestImage(readOnly bool) *ir.Image {
	const stack = 64
	image := &ir.Image{ImageStackSize: stack, ImageDataStart: stack, ImageEntryInit: ir.ImageFuncNone}
	data := make([]byte, 16)
	result := uint32(stack)
	offsets := map[types.Code][]uint32{}
	for code := types.BOOL; code <= types.F64; code++ {
		values := llvmTestValues(code)
		for _, v := range values {
			offsets[code] = append(offsets[code], stack+uint32(len(data)))
			data = binary.LittleEndian.AppendUint64(data, v)[:len(data)+int(llvmTestSize(code))]
		}
	}
	image.ImageData = data
	if readOnly {
		image.ImageReadOnly = []ir.Span{{SpanStart: stack + 16, SpanEnd: stack + uint32(len(data))}}
	}
	var text bytes.Buffer
	textOf := func(s string) ir.Instr {
		in := ir.Instr{InstrOpcode: ir.OpcodePrintText, InstrN: uint32(text.Len()), InstrM: uint32(len(s))}
		text.WriteString(s)
		return in
	}
	var mainCode []ir.Instr
	for op := ir.OpcodeBoolNot; op < ir.OpcodeCount; op++ {
		fact := llvm.OpcodeFactOf(op)
		var code []ir.Instr
		as := llvmTestValues(fact.OpcodeFactType)
		bs := []uint64{0}
		if fact.OpcodeFactInputs == 2 {
			bs = as
		}
		for i, a := range as {
			for j, b := range bs {
				if llvmTestSkip(fact, a, b) {
					continue
				}
				code = append(code, textOf(fmt.Sprintf("%s %d %d: ", fact.OpcodeFactName, i, j)))
				in := ir.Instr{InstrOpcode: op, InstrA: offsets[fact.OpcodeFactType][i], InstrB: offsets[fact.OpcodeFactType][j], InstrC: result}
				code = append(code, in)
				if fact.OpcodeFactResult != types.UNDEFINED {
					code = append(code, ir.Instr{InstrOpcode: ir.OpcodePrintValue, InstrA: result, InstrM: uint32(fact.OpcodeFactResult), InstrN: 'v'})
					code = append(code, textOf("\n"))
				}
			}
		}
		code = append(code, ir.Instr{InstrOpcode: ir.OpcodeReturn})
		id := uint32(len(image.ImageFuncs))
		image.ImageFuncs = append(image.ImageFuncs, ir.Func{FuncCode: code})
		image.ImageFuncNames = append(image.ImageFuncNames, strings.ReplaceAll(fact.OpcodeFactName, ".", "_"))
		mainCode = append(mainCode, ir.Instr{InstrOpcode: ir.OpcodeCallFrame}, ir.Instr{InstrOpcode: ir.OpcodeCall, InstrM: id})
	}
	mainCode = append(mainCode, ir.Instr{InstrOpcode: ir.OpcodeReturn})
	image.ImageEntryMain = uint32(len(image.ImageFuncs))
	image.ImageFuncs = append(image.ImageFuncs, ir.Func{FuncCode: mainCode})
	image.ImageFuncNames = append(image.ImageFuncNames, "main")
	image.ImageText = text.Bytes()
	return image
}

// llvmTestVm runs image on vm and returns its output.
func llvmTestVm(t *testing.T, image *ir.Image) []byte {
	machine := vm.MachineNew(image, 1<<20)
	stop, _ := machine.Run(image.ImageEntryMain, 0)
	if stop != vm.StopReturn {
		t.Fatalf("vm stopped with %d", stop)
	}
	return machine.MachineOutput
}

// llvmTestNative compiles image with clang and returns what it prints.
func llvmTestNative(t *testing.T, cc string, image *ir.Image, options llvm.Options) []byte {
	module, _, err := llvm.Emit(image, options)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	ll, runtime, exe := filepath.Join(dir, "test.ll"), filepath.Join(dir, llvm.RuntimeFileName), filepath.Join(dir, "test.exe")
	if err := os.WriteFile(ll, module, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtime, llvm.RuntimeC, 0o644); err != nil {
		t.Fatal(err)
	}
	args := []string{"-O2", "-w", "-o", exe, ll, runtime}
	if goruntime.GOOS != "windows" {
		args = append(args, "-lm")
	}
	if out, err := exec.Command(cc, args...).CombinedOutput(); err != nil {
		t.Fatalf("%s: %v\n%s", cc, err, out)
	}
	out, err := exec.Command(exe).Output()
	if err != nil {
		t.Fatalf("native run: %v", err)
	}
	return out
}

// TestLlvmScalarOpcodes runs every scalar opcode on edge operands on vm and
// as native code, with the operands in memory and as folded literals, by its
// template and by its runtime function, and checks that all print the same.
// It needs clang.
func TestLlvmScalarOpcodes(t *testing.T) {
	cc := llvmTestClang(t)
	for _, variant := range []struct{ readOnly, fallbackOnly bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
		t.Run(fmt.Sprintf("readonly=%v,fallback=%v", variant.readOnly, variant.fallbackOnly), func(t *testing.T) {
			t.Parallel()
			image := llvmTestImage(variant.readOnly)
			want := llvmTestVm(t, image)
			options := buildLlvmOptions()
			options.FallbackOnly = variant.fallbackOnly
			got := llvmTestNative(t, cc, image, options)
			if !bytes.Equal(got, want) {
				line, gotLine, wantLine := executionRecordFirstDifference(got, want)
				t.Errorf("native differs from vm at line %d: %q, vm %q", line, gotLine, wantLine)
			}
		})
	}
}

// TestLlvmDebug builds a program with debug information and checks that it
// names the variables from ImageLocals and runs as vm does. It needs clang.
func TestLlvmDebug(t *testing.T) {
	cc := llvmTestClang(t)
	initProgram()
	compileRunSources(t, []string{"debug.cx"}, []string{`package main

func add(a i32, b i32) (c i32) {
	var twice i32
	twice = a * 2
	c = twice + b
}

func main() {
	var total i32
	var ratio f64
	total = add(20, 2)
	ratio = 1.5D
	i32.print(total)
	f64.print(ratio)
}
`})
	image, failures := lower.Program(actions.AST)
	if len(failures) > 0 {
		t.Fatalf("does not lower: %+v", failures)
	}
	options := buildLlvmOptions()
	options.Debug = true
	module, _, err := llvm.Emit(&image, options)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`!DILocalVariable(name: "total"`, `!DILocalVariable(name: "ratio"`, `!DISubprogram(name: "add"`, `!DIBasicType(name: "f64", size: 64, encoding: DW_ATE_float)`} {
		if !bytes.Contains(module, []byte(want)) {
			t.Errorf("module has no %s", want)
		}
	}
	if got, want := llvmTestNative(t, cc, &image, options), llvmTestVm(t, &image); !bytes.Equal(got, want) {
		t.Errorf("native prints %q, vm %q", got, want)
	}
}

// TestLlvmCompareBenchmarks builds the t0-benchmarks/compare programs with
// cx build --target=llvm and checks that each prints the hash of
// expected.txt and its pass line. It needs clang.
func TestLlvmCompareBenchmarks(t *testing.T) {
	llvmTestClang(t)
	if testing.Short() {
		t.Skip("builds the compare benchmarks")
	}
	dir, err := filepath.Abs(filepath.Join("..", "t0-benchmarks", "compare"))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(dir, "expected.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{}
	for _, line := range strings.Split(string(expected), "\n") {
		if fields := strings.Fields(line); len(fields) == 3 && fields[1] == "full" {
			want[fields[0]] = fields[2]
		}
	}
	harness := executionRecordHarnessNew(t)
	for _, bench := range []string{"count", "fib", "array", "struct", "memtest", "primes"} {
		output := filepath.Join(t.TempDir(), bench)
		if goruntime.GOOS == "windows" {
			output += ".exe"
		}
		build := executionRecordProcess(t, harness.executable, []string{"CX_TEST_RUN_MAIN=1"}, dir, "", buildCommand, "--target="+buildTargetLlvm, filepath.Join("cx", bench+".cx"), "-o", output)
		if build.exitCode != 0 {
			t.Fatalf("%s: cx build exit code %d: %s", bench, build.exitCode, build.stderr)
		}
		run := executionRecordProcess(t, output, nil, dir, "")
		lines := strings.Split(string(run.stdout), "\n")
		if run.exitCode != 0 || len(lines) < 2 || lines[0] != want[bench] || lines[1] != bench+" pass" {
			t.Errorf("%s: exit code %d, output %q, want hash %s and a pass line", bench, run.exitCode, run.stdout, want[bench])
		}
	}
}
