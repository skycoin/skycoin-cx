package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	core "github.com/skycoin/skycoin-cx/t6-cx-runtime-core"
)

// buildTestRunFlags are the cx flags, without dashes, that only change how
// cx runs a program. A built program runs with their defaults, so programs
// that pass one are not compared.
var buildTestRunFlags = map[string]bool{"executor": true, "debug-go-stack": true, "ast": true, "heap-max": true, "hm": true, "min-heap-free": true, "max-heap-free": true}

// buildTestCore builds the core that links libraries into dir, by its
// buildCoreFileName, with ptr64 when this cx uses 8 byte pointers.
func buildTestCore(t *testing.T, dir string, libraries ...string) string {
	t.Helper()
	path := filepath.Join(dir, buildCoreFileName(libraries))
	tags := []string{}
	if types.POINTER_SIZE == 8 {
		tags = append(tags, "ptr64")
	}
	for _, library := range libraries {
		tags = append(tags, "cxcorelibrary"+library)
	}
	cmd := exec.Command("go", "build", "-o", path, "-tags", strings.Join(tags, ","), "../t7-cx-core-run")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build of the core %v: %v\n%s", libraries, err, out)
	}
	return path
}

// buildTestCores builds every core into dir, one per set of system
// libraries, and returns their paths by buildCoreFileName.
func buildTestCores(t *testing.T, dir string) map[string]string {
	t.Helper()
	cores := map[string]string{}
	for set := 0; set < 1<<len(constants.LibraryTable); set++ {
		var libraries []string
		for i, library := range constants.LibraryTable {
			if set>>i&1 != 0 {
				libraries = append(libraries, library.LibraryName)
			}
		}
		cores[buildCoreFileName(libraries)] = buildTestCore(t, dir, libraries...)
	}
	return cores
}

// buildTestProcess runs executable in dir and returns its stdout, stderr and
// exit code.
func buildTestProcess(t *testing.T, executable string, env []string, dir string, args ...string) (lowerCompareRun, []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	run := lowerCompareRun{stdout: out, timedOut: ctx.Err() != nil}
	if exitError, ok := err.(*exec.ExitError); ok {
		run.exitCode = exitError.ExitCode()
	} else if err != nil {
		t.Error(err)
	}
	return run, stderr.Bytes()
}

// A core binary with no program appended, and a built program whose
// trailer is damaged, fail with a message instead of running.
func TestBuildCoreRefuses(t *testing.T) {
	dir := t.TempDir()
	corePath := buildTestCore(t, dir)
	run, stderr := buildTestProcess(t, corePath, nil, dir)
	if run.exitCode != types.CX_COMPILATION_ERROR || !bytes.Contains(stderr, []byte("no CX program is appended")) {
		t.Errorf("core alone: exit code %d, stderr %q", run.exitCode, stderr)
	}

	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	tests, err := filepath.Abs(filepath.Join("..", "tests"))
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "hello")
	if runtime.GOOS == "windows" {
		output += ".exe"
	}
	build, stderr := buildTestProcess(t, executable, []string{"CX_TEST_RUN_MAIN=1"}, tests, buildCommand, "examples/hello-world.cx", "-o", output, "--core", corePath)
	if build.exitCode != types.CX_SUCCESS {
		t.Fatalf("cx build: exit code %d, stderr %q", build.exitCode, stderr)
	}
	built, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	built[len(built)-core.CoreTrailerSize-1] ^= 0xff
	if err := os.WriteFile(output, built, 0o755); err != nil {
		t.Fatal(err)
	}
	run, stderr = buildTestProcess(t, output, nil, dir)
	if run.exitCode != types.CX_COMPILATION_ERROR || !bytes.Contains(stderr, []byte("checksum")) {
		t.Errorf("damaged program: exit code %d, stderr %q", run.exitCode, stderr)
	}

	build, stderr = buildTestProcess(t, executable, []string{"CX_TEST_RUN_MAIN=1"}, tests, buildCommand, "--enable=file", "test-os-read-slice-grow.cx", "-o", output, "--core", corePath)
	if build.exitCode != types.CX_SUCCESS {
		t.Fatalf("file program on a given core: exit code %d, stderr %q", build.exitCode, stderr)
	}
	run, stderr = buildTestProcess(t, output, nil, dir)
	if run.exitCode != types.CX_COMPILATION_ERROR || !bytes.Contains(stderr, []byte("does not link: file")) {
		t.Errorf("file program on a core without file: exit code %d, stderr %q", run.exitCode, stderr)
	}
}

// cx build links a system library only when the program calls it and
// --enable enables it, and a call into a library left out stops the built
// program as it stops cx.
func TestBuildLibraries(t *testing.T) {
	dir := t.TempDir()
	buildTestCore(t, dir)
	fileNet := buildTestCore(t, dir, "file", "net")
	fileNetBinary, err := os.ReadFile(fileNet)
	if err != nil {
		t.Fatal(err)
	}
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	tests, err := filepath.Abs(filepath.Join("..", "tests"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		enable   string
		linked   bool
		exitCode int
		want     string
	}{
		{"--enable=file,net", true, types.CX_SUCCESS, "7\n"},
		{"--enable=file,net,process", true, types.CX_SUCCESS, "7\n"},
		{"--enable=process", false, types.CX_RUNTIME_LIBRARY_DISABLED, "library file is not enabled"},
		{"--enable=", false, types.CX_RUNTIME_LIBRARY_DISABLED, "library file is not enabled"},
	}
	for i, c := range cases {
		output := filepath.Join(dir, fmt.Sprintf("program-%d", i))
		if runtime.GOOS == "windows" {
			output += ".exe"
		}
		build, stderr := buildTestProcess(t, executable, []string{"CX_TEST_RUN_MAIN=1"}, tests, buildCommand, c.enable, "test-library-call.cx", "-o", output, "--core", dir)
		if build.exitCode != types.CX_SUCCESS {
			t.Errorf("%s: cx build exit code %d, stderr %q", c.enable, build.exitCode, stderr)
			continue
		}
		built, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		if linked := bytes.HasPrefix(built, fileNetBinary); linked != c.linked {
			t.Errorf("%s: built on the file and net core %v, want %v", c.enable, linked, c.linked)
		}
		run, _ := buildTestProcess(t, output, nil, dir)
		if run.exitCode != c.exitCode || !bytes.Contains(run.stdout, []byte(c.want)) {
			t.Errorf("%s: exit code %d, stdout %q, want %d and %q", c.enable, run.exitCode, run.stdout, c.exitCode, c.want)
		}
	}
}
