package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
)

// buildOutputTestExecutable is name as cx build names an executable.
func buildOutputTestExecutable(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// Without -o, cx build names the output as go build does, after the first
// source file or the one source directory, and refuses sources that give no
// single name.
func TestBuildOutputDefault(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	for _, path := range []string{"fib.cx", "util.cx", "prog", "project/main.cx", "taken.cx", "taken/main.cx", "work/main.cx"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		sources []string
		want    string
	}{
		{[]string{"fib.cx"}, "fib"},
		{[]string{"fib.cx", "util.cx"}, "fib"},
		{[]string{"--enable=file", "++name", "fib.cx"}, "fib"},
		{[]string{"-heap-max", "util.cx", "fib.cx"}, "fib"},
		{[]string{"project"}, "project"},
		{[]string{"project/"}, "project"},
		{[]string{"."}, "work"},
	} {
		// A directory is built from work, so that its output is not itself.
		sources := test.sources
		if len(sources) == 1 && !strings.HasSuffix(sources[0], ".cx") {
			t.Chdir(filepath.Join(dir, "work"))
			if sources[0] != "." {
				sources = []string{filepath.Join("..", sources[0])}
			}
		}
		got, problem := buildOutputDefault(sources)
		if want := buildOutputTestExecutable(test.want); got != want || problem != "" {
			t.Errorf("sources %q: output %q, problem %q, want %q", sources, got, problem, want)
		}
		t.Chdir(dir)
	}
	for _, sources := range [][]string{{"prog"}, {"project"}, {"project", "fib.cx"}, {"project", "taken"}, {"taken.cx"}} {
		// On Windows the output gets .exe, so it is not the directory.
		if runtime.GOOS == "windows" && len(sources) == 1 && (sources[0] == "taken.cx" || sources[0] == "project") {
			continue
		}
		if got, problem := buildOutputDefault(sources); problem == "" {
			t.Errorf("sources %q: output %q, want a problem", sources, got)
		}
	}
}

// cx build without -o writes hello for hello.cx, greeting for the directory
// ../greeting, and with --target=llvm main.ll and main for ../greeting/main.cx.
func TestBuildOutputDefaultBuilds(t *testing.T) {
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join("..", "tests", "examples", "hello-world.cx"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	corePath := buildTestCore(t, dir)
	work := filepath.Join(dir, "work")
	for _, path := range []string{work, filepath.Join(dir, "greeting")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{filepath.Join("work", "hello.cx"), filepath.Join("greeting", "main.cx")} {
		if err := os.WriteFile(filepath.Join(dir, path), source, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	env := []string{"CX_TEST_RUN_MAIN=1"}
	builds := []struct {
		args   []string
		output string
	}{
		{[]string{"hello.cx", "--core", corePath}, "hello"},
		{[]string{filepath.Join("..", "greeting"), "--core", corePath}, "greeting"},
	}
	if buildLlvmClang() != "" {
		builds = append(builds, struct {
			args   []string
			output string
		}{[]string{"--target=llvm", filepath.Join("..", "greeting", "main.cx")}, "main"})
	} else {
		t.Log("no clang, the llvm build without -o is not run")
	}
	for _, build := range builds {
		output := filepath.Join(work, buildOutputTestExecutable(build.output))
		run, stderr := buildTestProcess(t, executable, env, work, append([]string{buildCommand}, build.args...)...)
		if run.exitCode != types.CX_SUCCESS {
			t.Fatalf("cx build %s: exit code %d, stderr %q", strings.Join(build.args, " "), run.exitCode, stderr)
		}
		run, stderr = buildTestProcess(t, output, nil, work)
		if run.exitCode != types.CX_SUCCESS || !strings.Contains(string(run.stdout), "Hello World!") {
			t.Errorf("%s from cx build %s: exit code %d, stdout %q, stderr %q", output, strings.Join(build.args, " "), run.exitCode, run.stdout, stderr)
		}
		if build.args[0] == "--target=llvm" {
			if _, err := os.Stat(filepath.Join(work, "main.ll")); err != nil {
				t.Errorf("cx build --target=llvm ../greeting/main.cx wrote no main.ll: %v", err)
			}
		}
	}
}
