package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
)

func TestBuildLlvmLinkerChoose(t *testing.T) {
	cases := []struct {
		clang, tinygo        string
		linker, tinygoTarget string
		want, problem        bool
		wantLinker           string
	}{
		{clang: "sh", tinygo: "sh", wantLinker: buildLlvmLinkerClang},
		{clang: "cx-no-such-clang", tinygo: "sh", wantLinker: buildLlvmLinkerTinygo},
		{clang: "cx-no-such-clang", tinygo: "cx-no-such-tinygo", wantLinker: buildLlvmLinkerClang},
		{clang: "sh", tinygo: "sh", linker: buildLlvmLinkerTinygo, wantLinker: buildLlvmLinkerTinygo},
		{clang: "sh", tinygo: "sh", tinygoTarget: "pico", wantLinker: buildLlvmLinkerTinygo},
		{clang: "sh", tinygo: "sh", linker: buildLlvmLinkerClang, tinygoTarget: "pico", problem: true},
		{clang: "sh", tinygo: "sh", linker: "gcc", problem: true},
	}
	for _, c := range cases {
		t.Setenv(buildLlvmClangEnv, c.clang)
		t.Setenv(buildLlvmTinygoEnv, c.tinygo)
		linker, problem := buildLlvmLinkerChoose(buildOptions{buildOptionsLlvmLinker: c.linker, buildOptionsLlvmTinygoTarget: c.tinygoTarget})
		if (problem != "") != c.problem || !c.problem && linker != c.wantLinker {
			t.Errorf("%+v: linker %q problem %q", c, linker, problem)
		}
	}
}

// buildLlvmTinygoTestPrograms print every kind of number the runtime
// formats and stop with an error. TinyGo links a C library whose printf
// may format no float and no 64 bit integer.
var buildLlvmTinygoTestPrograms = []struct{ name, source string }{
	{"hello", "package main\n\nimport \"sys/string\"\n\nfunc main() {\n\tstring.PrintLine(\"hello from tinygo\")\n}\n"},
	{"numbers", `package main

func main() {
	var big i64 = -9223372036854775807L
	var huge ui64 = 18446744073709551615UL
	i64.print(big)
	ui64.print(huge)
	i32.print(-2147483647)
	var third f64 = 1.0D / 3.0D
	var tenth f32 = 0.1
	f64.print(third)
	f32.print(tenth)
	f64.print(1.5e300D)
	f64.print(0.00001D)
	printf("%f %f\n", tenth, third)
}
`},
	{"divide", "package main\n\nfunc main() {\n\tvar zero i32\n\ti32.print(7 / zero)\n}\n"},
}

// TestBuildLlvmTinygo links each program with TinyGo for the host and checks
// that it prints what cx prints and exits as cx exits. It needs tinygo.
func TestBuildLlvmTinygo(t *testing.T) {
	if buildLlvmTinygo() == "" {
		t.Skip("no tinygo on this machine")
	}
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, program := range buildLlvmTinygoTestPrograms {
		source := program.name + ".cx"
		if err := os.WriteFile(filepath.Join(dir, source), []byte(program.source), 0o644); err != nil {
			t.Fatal(err)
		}
		want := executionRecordProcess(t, executable, []string{"CX_TEST_RUN_MAIN=1"}, dir, "", "--executor=vm", source)
		output := filepath.Join(dir, program.name)
		build := executionRecordProcess(t, executable, []string{"CX_TEST_RUN_MAIN=1"}, dir, "", buildCommand, "--target="+buildTargetLlvm, "--llvm-linker="+buildLlvmLinkerTinygo, source, "-o", output)
		if build.exitCode != valuetypes.CXSuccess {
			t.Fatalf("%s: cx build exit code %d, stderr %s", program.name, build.exitCode, build.stderr)
		}
		run := executionRecordProcess(t, output, nil, dir, "")
		wantStdout := want.stdout
		if cut := bytes.Index(wantStdout, executionRecordCallStack); cut >= 0 {
			wantStdout = append(wantStdout[:cut:cut], '\n')
		}
		if run.exitCode != want.exitCode || string(run.stdout) != string(wantStdout) {
			t.Errorf("%s: exit code %d stdout %q, cx gives %d %q", program.name, run.exitCode, run.stdout, want.exitCode, wantStdout)
		}
	}
}

func TestBuildLlvmTinygoTripleClang(t *testing.T) {
	cases := []struct {
		version string
		triple  string
		want    string
	}{
		{"tinygo version 0.44.0 linux/amd64 (using go version go1.27.3 and LLVM version 23.1.3)", buildLlvmTinygoTripleWasi, buildLlvmTinygoTripleWasip1},
		{"tinygo version 0.40.0 linux/amd64 (using go version go1.25.0 and LLVM version 22.1.0)", buildLlvmTinygoTripleWasi, buildLlvmTinygoTripleWasip1},
		{"tinygo version 0.39.0 linux/amd64 (using go version go1.25.0 and LLVM version 20.1.1)", buildLlvmTinygoTripleWasi, buildLlvmTinygoTripleWasi},
		{"tinygo version 0.44.0 linux/amd64 (using go version go1.27.3 and LLVM version 23.1.3)", "thumbv6m-unknown-unknown-eabi", "thumbv6m-unknown-unknown-eabi"},
		{"no version here", buildLlvmTinygoTripleWasi, buildLlvmTinygoTripleWasi},
	}
	for _, c := range cases {
		if got := buildLlvmTinygoTripleClang(c.triple, buildLlvmTinygoLlvmMajorParse([]byte(c.version))); got != c.want {
			t.Errorf("%q %s: got %s, want %s", c.version, c.triple, got, c.want)
		}
	}
}
