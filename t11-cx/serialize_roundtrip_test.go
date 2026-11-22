package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t5-cx-runtime-printer"
)

// serializeRoundTripProgramEnv holds the cx arguments a round trip child
// compiles, serializes, deserializes and runs, joined by spaces.
const serializeRoundTripProgramEnv = "CX_SERIALIZE_ROUND_TRIP_PROGRAM"

// serializeRoundTripFailed is the exit code of a child whose round trip
// failed, outside the CX exit codes.
const serializeRoundTripFailed = 125

// serializeRoundTripSkip matches source whose output differs from run to run,
// with the reason. A program matching none runs in both ways.
var serializeRoundTripSkip = []struct {
	pattern *regexp.Regexp
	reason  string
}{
	{regexp.MustCompile(`\bread\(`), "reads stdin"},
	{regexp.MustCompile(`\.rand\b`), "prints random values"},
	{regexp.MustCompile(`\bcipher\.GenerateKeyPair\b`), "prints random keys"},
	{regexp.MustCompile(`\btime\.Unix`), "prints the clock"},
	{regexp.MustCompile(`\bos\.Run\b`), "runs other cx programs"},
	{regexp.MustCompile(`\btcp\.Listen\b`), "listens on a fixed port that parallel runs share"},
	{regexp.MustCompile(`\bos\.Create\("[A-Za-z0-9_-]+\.tmp"\)`), "writes a fixed file that parallel runs share"},
}

// serializeRoundTripExecutors are the executor flags each program runs with.
// The default picks vm when the program lowers, old forces the old loop.
var serializeRoundTripExecutors = []string{"--executor=auto", "--executor=old"}

// TestSerializeRoundTrip runs every program of tests/*.cx and
// tests/examples/*.cx that compiles, and every t1-cxtest entry that passes
// flags or several files, once as cx runs it and once after a serialize and
// deserialize round trip between compiling and running, on both executors.
// Stdout and the exit code must be the same.
func TestSerializeRoundTrip(t *testing.T) {
	tests := filepath.Join("..", "tests")
	skipped := map[string]string{}
	var programs, run []string
	for _, program := range serializeRoundTripPrograms(t, tests) {
		programs = append(programs, program)
		reason := ""
		for _, arg := range strings.Fields(program) {
			if arg == "-ast" {
				reason = "prints the AST while it compiles"
			}
			if !strings.HasSuffix(arg, ".cx") {
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
		for _, executor := range serializeRoundTripExecutors {
			run = append(run, executor+" "+program)
		}
	}

	type outcome struct {
		direct, roundTrip lowerCompareRun
	}
	outcomes := make([]outcome, len(run))
	limit := make(chan struct{}, runtime.NumCPU())
	var wait sync.WaitGroup
	for i, program := range run {
		wait.Add(1)
		limit <- struct{}{}
		go func() {
			defer wait.Done()
			defer func() { <-limit }()
			outcomes[i].direct = lowerCompareProcess(t, []string{"CX_TEST_RUN_MAIN=1"}, tests, strings.Fields(program)...)
			if outcomes[i].direct.exitCode == types.CX_COMPILATION_ERROR {
				return
			}
			outcomes[i].roundTrip = lowerCompareProcess(t, []string{serializeRoundTripProgramEnv + "=" + program}, tests)
		}()
	}
	wait.Wait()

	var compileFailed, matched, mismatched int
	for i, program := range run {
		direct, roundTrip := outcomes[i].direct, outcomes[i].roundTrip
		switch {
		case direct.exitCode == types.CX_COMPILATION_ERROR:
			compileFailed++
		case direct.timedOut || roundTrip.timedOut:
			mismatched++
			t.Errorf("%s: timed out, direct %v, round trip %v", program, direct.timedOut, roundTrip.timedOut)
		case direct.exitCode != roundTrip.exitCode:
			mismatched++
			t.Errorf("%s: exit code %d direct, %d after the round trip, stdout %q", program, direct.exitCode, roundTrip.exitCode, lowerCompareClip(roundTrip.stdout))
		case !bytes.Equal(direct.stdout, roundTrip.stdout):
			mismatched++
			t.Errorf("%s: stdout differs, direct %q, round trip %q", program, lowerCompareClip(direct.stdout), lowerCompareClip(roundTrip.stdout))
		default:
			matched++
		}
	}
	names := make([]string, 0, len(skipped))
	for program, reason := range skipped {
		names = append(names, program+" ("+reason+")")
	}
	sort.Strings(names)
	t.Logf("%d programs, %d skipped, %d runs of the rest on %d executors: %d did not compile, %d match, %d mismatch",
		len(programs), len(skipped), len(run), len(serializeRoundTripExecutors), compileFailed, matched, mismatched)
	t.Logf("skipped: %s", strings.Join(names, ", "))
}

// serializeRoundTripPrograms lists each .cx file of tests/ and tests/examples/
// alone, then each t1-cxtest entry that passes flags or several files. A
// program is its cx arguments relative to tests/, joined by spaces.
func serializeRoundTripPrograms(t *testing.T, tests string) []string {
	var programs []string
	for _, pattern := range []string{"*.cx", filepath.Join("examples", "*.cx")} {
		matches, err := filepath.Glob(filepath.Join(tests, pattern))
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range matches {
			name, err := filepath.Rel(tests, match)
			if err != nil {
				t.Fatal(err)
			}
			programs = append(programs, filepath.ToSlash(name))
		}
	}
	sort.Strings(programs)

	harness, err := os.ReadFile(filepath.Join("..", "t1-cxtest", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, match := range lowerCompareHarnessRun.FindAllStringSubmatch(string(harness), -1) {
		args := strings.Fields(match[1])
		files := 0
		for _, arg := range args {
			if strings.HasSuffix(arg, ".cx") {
				files++
			}
		}
		if files > 0 && (len(args) > 1) {
			programs = append(programs, match[1])
		}
	}
	return programs
}

// serializeRoundTripChild runs args as Run does, except that the compiled
// program goes through SerializeCXProgram and Deserialize before it runs.
func serializeRoundTripChild(args []string) {
	options, cxArgs, sourceCode, fileNames, start := runStart(args)
	if !start || !parseProgram(options, fileNames, sourceCode) {
		os.Exit(types.CX_COMPILATION_ERROR)
	}
	serialized, err := ast.SerializeCXProgram(actions.AST, true, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(serializeRoundTripFailed)
	}
	prgrm, err := ast.Deserialize(serialized, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(serializeRoundTripFailed)
	}
	printer.PrinterCompileErrorsRegister(prgrm)
	actions.AST = prgrm
	runProgram(options, cxArgs, sourceCode)
	os.Exit(0)
}
