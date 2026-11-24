package main

import (
	"os"
	"runtime/debug"
	"testing"

	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-execute"
)

const runParseFloatSource = `package main

import "sys/string"

func main() {
	var a f32
	var b f64
	var i i32
	for i = 0; i < 100; i++ {
		a = string.ParseFloatF32("333")
		a = string.ParseFloatF32("1.5")
		a = string.ParseFloatF32("-2.5e-3")
		a = string.ParseFloatF32("3.4028235e38")
		a = string.ParseFloatF32("0x1.8p-3")
		a = string.ParseFloatF32("1_000.25")
		a = string.ParseFloatF32("inf")
		b = string.ParseFloatF64("0.1")
		b = string.ParseFloatF64("1.7976931348623157e308")
		b = string.ParseFloatF64("4.9406564584124654e-324")
		b = string.ParseFloatF64("123456789012345678901234567890")
		b = string.ParseFloatF64("2.2250738585072011e-308")
		b = string.ParseFloatF64("NaN")
	}
	a = string.ParseFloatF32("0x1.8p-3")
	b = string.ParseFloatF64("2.2250738585072011e-308")
	printf("%v %v\n", a, b)
}
`

// TestRunParseFloatAllocs fails when string.ParseFloatF32 or
// string.ParseFloatF64 allocates on the old loop (#412, D92).
func TestRunParseFloatAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	p := runBenchProgram{name: "parsefloat", source: runParseFloatSource}
	if allocs := runPrintAllocs(t, p); allocs != 0 {
		t.Errorf("%.0f allocations per run, want 0", allocs)
	}
}

// BenchmarkRunParseFloat times the old loop run of TestRunParseFloatAllocs,
// 1300 float parses.
func BenchmarkRunParseFloat(b *testing.B) {
	initProgram()
	discardStdout(b)
	p := runBenchProgram{name: "parsefloat", source: runParseFloatSource}
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
}

// TestRunParseFloatOutput fails when the program of TestRunParseFloatAllocs
// stops early or prints other values.
func TestRunParseFloatOutput(t *testing.T) {
	initProgram()
	out, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = out
	defer func() { os.Stdout = stdout }()
	compileRunProgram(t, runBenchProgram{name: "parsefloat", source: runParseFloatSource})
	execute.RunPrepare(actions.AST)
	if err := execute.RunCompiled(actions.AST, 0, nil); err != nil {
		t.Fatal(err)
	}
	os.Stdout = stdout
	text, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatal(err)
	}
	if want := "0.1875 2.225073858507201e-308\n"; string(text) != want {
		t.Errorf("output %q, want %q", text, want)
	}
}
