package main

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
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

// TestBuildPrograms builds every program of TestSerializeRoundTrip that the
// core supports with cx build and runs the built file. Its stdout and exit
// code must be those of cx running the source. It logs the core size and the
// sizes of the appended programs.
func TestBuildPrograms(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs every test program")
	}
	tests, err := filepath.Abs(filepath.Join("..", "tests"))
	if err != nil {
		t.Fatal(err)
	}
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cores := buildTestCores(t, dir)
	var coreSizes []string
	for _, name := range slices.Sorted(maps.Keys(cores)) {
		info, err := os.Stat(cores[name])
		if err != nil {
			t.Fatal(err)
		}
		coreSizes = append(coreSizes, fmt.Sprintf("%s %d", name, info.Size()))
	}

	skipped := map[string]string{}
	var programs []string
	for _, program := range serializeRoundTripPrograms(t, tests) {
		reason := ""
		for _, arg := range strings.Fields(program) {
			if name, _, _ := strings.Cut(strings.TrimLeft(arg, "-"), "="); strings.HasPrefix(arg, "-") && buildTestRunFlags[name] {
				reason = "passes the run time flag -" + name
			}
			if reason != "" || !strings.HasSuffix(arg, ".cx") {
				continue
			}
			source, err := os.ReadFile(filepath.Join(tests, arg))
			if err != nil {
				t.Fatal(err)
			}
			for _, skip := range serializeRoundTripSkip {
				if reason == "" && skip.pattern.Match(source) {
					reason = skip.reason
				}
			}
		}
		if reason != "" {
			skipped[program] = reason
			continue
		}
		programs = append(programs, program)
	}

	type outcome struct {
		direct, built lowerCompareRun
		buildCode     int
		buildStderr   []byte
		payload       int64
	}
	outcomes := make([]outcome, len(programs))
	limit := make(chan struct{}, runtime.NumCPU())
	var wait sync.WaitGroup
	for i, program := range programs {
		wait.Add(1)
		limit <- struct{}{}
		go func() {
			defer wait.Done()
			defer func() { <-limit }()
			args := strings.Fields(program)
			outcomes[i].direct = lowerCompareProcess(t, []string{"CX_TEST_RUN_MAIN=1"}, tests, args...)
			if outcomes[i].direct.exitCode == types.CX_COMPILATION_ERROR {
				return
			}
			output := filepath.Join(dir, fmt.Sprintf("program-%d", i))
			if runtime.GOOS == "windows" {
				output += ".exe"
			}
			buildArgs := append([]string{buildCommand}, args...)
			buildArgs = append(buildArgs, "-o", output, "--core", dir)
			build, stderr := buildTestProcess(t, executable, []string{"CX_TEST_RUN_MAIN=1"}, tests, buildArgs...)
			outcomes[i].buildCode, outcomes[i].buildStderr = build.exitCode, stderr
			if build.exitCode != types.CX_SUCCESS {
				return
			}
			defer os.Remove(output)
			if program, fileError := core.CoreTrailerRead(output); fileError == nil {
				outcomes[i].payload = int64(len(program))
			}
			outcomes[i].built, _ = buildTestProcess(t, output, nil, tests)
		}()
	}
	wait.Wait()

	var compileFailed, refused, matched, mismatched int
	var payloads []int64
	for i, program := range programs {
		result := outcomes[i]
		switch {
		case result.direct.exitCode == types.CX_COMPILATION_ERROR:
			compileFailed++
		case result.buildCode != types.CX_SUCCESS && bytes.Contains(result.buildStderr, []byte("opcodes it does not include")):
			refused++
			skipped[program] = "needs " + strings.TrimSpace(string(result.buildStderr[bytes.LastIndex(result.buildStderr, []byte(": "))+2:]))
		case result.buildCode != types.CX_SUCCESS:
			mismatched++
			t.Errorf("%s: cx build exit code %d, stderr %q", program, result.buildCode, lowerCompareClip(result.buildStderr))
		case result.direct.timedOut || result.built.timedOut:
			mismatched++
			t.Errorf("%s: timed out, direct %v, built %v", program, result.direct.timedOut, result.built.timedOut)
		case result.direct.exitCode != result.built.exitCode:
			mismatched++
			t.Errorf("%s: exit code %d direct, %d built, stdout %q", program, result.direct.exitCode, result.built.exitCode, lowerCompareClip(result.built.stdout))
		case !bytes.Equal(result.direct.stdout, result.built.stdout):
			mismatched++
			t.Errorf("%s: stdout differs, direct %q, built %q", program, lowerCompareClip(result.direct.stdout), lowerCompareClip(result.built.stdout))
		default:
			matched++
			payloads = append(payloads, result.payload)
		}
	}
	names := make([]string, 0, len(skipped))
	for program, reason := range skipped {
		names = append(names, program+" ("+reason+")")
	}
	sort.Strings(names)
	sort.Slice(payloads, func(i, j int) bool { return payloads[i] < payloads[j] })
	t.Logf("cores in bytes: %s", strings.Join(coreSizes, ", "))
	t.Logf("%d programs: %d did not compile, %d refused for opcodes outside the core, %d built and match, %d mismatch, %d skipped before building",
		len(programs)+len(skipped)-refused, compileFailed, refused, matched, mismatched, len(skipped)-refused)
	if len(payloads) > 0 {
		t.Logf("appended program bytes: min %d, median %d, max %d", payloads[0], payloads[len(payloads)/2], payloads[len(payloads)-1])
	}
	t.Logf("skipped: %s", strings.Join(names, ", "))
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
