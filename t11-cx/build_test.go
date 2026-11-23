package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/skycoin/skycoin-cx/t0-types"
	core "github.com/skycoin/skycoin-cx/t6-cx-runtime-core"
)

// buildTestRunFlags are the cx flags, without dashes, that only change how
// cx runs a program. A built program runs with their defaults, so programs
// that pass one are not compared.
var buildTestRunFlags = map[string]bool{"executor": true, "debug-go-stack": true, "ast": true, "heap-max": true, "hm": true, "min-heap-free": true, "max-heap-free": true}

// buildTestCore builds the core binary into dir, with ptr64 when this cx
// uses 8 byte pointers.
func buildTestCore(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, buildCoreName)
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	args := []string{"build", "-o", path}
	if types.POINTER_SIZE == 8 {
		args = append(args, "-tags", "ptr64")
	}
	cmd := exec.Command("go", append(args, "../t7-cx-core-run")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build of the core: %v\n%s", err, out)
	}
	return path
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
	corePath := buildTestCore(t, dir)
	coreInfo, err := os.Stat(corePath)
	if err != nil {
		t.Fatal(err)
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
			buildArgs = append(buildArgs, "-o", output, "--core", corePath)
			build, stderr := buildTestProcess(t, executable, []string{"CX_TEST_RUN_MAIN=1"}, tests, buildArgs...)
			outcomes[i].buildCode, outcomes[i].buildStderr = build.exitCode, stderr
			if build.exitCode != types.CX_SUCCESS {
				return
			}
			defer os.Remove(output)
			if info, err := os.Stat(output); err == nil {
				outcomes[i].payload = info.Size() - coreInfo.Size() - core.CoreTrailerSize
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
		case result.buildCode != types.CX_SUCCESS && (bytes.Contains(result.buildStderr, []byte("opcodes it does not include")) || bytes.Contains(result.buildStderr, []byte("links no system library"))):
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
	t.Logf("core %d bytes; %d programs: %d did not compile, %d refused for opcodes outside the core, %d built and match, %d mismatch, %d skipped before building",
		coreInfo.Size(), len(programs)+len(skipped)-refused, compileFailed, refused, matched, mismatched, len(skipped)-refused)
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
	if build.exitCode != types.CX_COMPILATION_ERROR || !bytes.Contains(stderr, []byte("libraries: file")) {
		t.Errorf("file program: exit code %d, stderr %q", build.exitCode, stderr)
	}
}
