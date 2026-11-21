package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	lower "github.com/skycoin/skycoin-cx/t4-cx-compiler-lower"
	"github.com/skycoin/skycoin-cx/t4-execute"
)

// lowerCompareProgramEnv names the program a child of TestLowerCompare
// checks. Each program runs in its own process because a compile error or a
// runtime error in the old loop ends the process.
const lowerCompareProgramEnv = "CX_LOWER_COMPARE_PROGRAM"

// lowerCompareMarker starts each result line a child prints.
const lowerCompareMarker = "LOWERCOMPARE "

// lowerCompareExcluded lists programs inside the subset whose results differ
// for a reason other than a lowering bug, with that reason.
var lowerCompareExcluded = map[string]string{}

// lowerCompareStage is how far a child got with its program.
type lowerCompareStage string

const (
	lowerCompareStageCompiled lowerCompareStage = "compiled"
	lowerCompareStageOutside  lowerCompareStage = "outside"
	lowerCompareStageLowered  lowerCompareStage = "lowered"
	lowerCompareStageMatch    lowerCompareStage = "match"
	lowerCompareStageMismatch lowerCompareStage = "mismatch"
)

// lowerCompareFailure is one function of a program that did not lower.
type lowerCompareFailure struct {
	Function string
	Reason   string
	Opcode   string
}

// lowerCompareResult is one result line of a child.
type lowerCompareResult struct {
	Stage         lowerCompareStage
	Failures      []lowerCompareFailure
	ReferenceCode int32
	Mismatch      string
}

// TestLowerCompare lowers every CX program in tests/ and, for each program
// whose functions all lower, compares the memory and stdout after the old
// loop with the reference interpreter, step 4 of #408.
func TestLowerCompare(t *testing.T) {
	if program := os.Getenv(lowerCompareProgramEnv); program != "" {
		lowerCompareChild(t, program)
		return
	}

	programs := lowerComparePrograms(t)

	type outcome struct {
		results  []lowerCompareResult
		exitCode int
		timedOut bool
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
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLowerCompare$")
			cmd.Env = append(os.Environ(), lowerCompareProgramEnv+"="+program)
			out, err := cmd.Output()
			if exitError, ok := err.(*exec.ExitError); ok {
				outcomes[i].exitCode = exitError.ExitCode()
			}
			outcomes[i].timedOut = ctx.Err() != nil
			scanner := bufio.NewScanner(bytes.NewReader(out))
			scanner.Buffer(make([]byte, 1<<20), 1<<20)
			for scanner.Scan() {
				line, found := strings.CutPrefix(scanner.Text(), lowerCompareMarker)
				if !found {
					continue
				}
				var result lowerCompareResult
				if err := json.Unmarshal([]byte(line), &result); err != nil {
					t.Errorf("%s: %v", program, err)
					continue
				}
				outcomes[i].results = append(outcomes[i].results, result)
			}
		}()
	}
	wait.Wait()

	var compileFailed, outside, inside, matched, mismatched, excluded int
	var matchedNames []string
	firstReasons := map[string]int{}
	opcodes := map[string]int{}
	for i, name := range programs {
		o := outcomes[i]
		if len(o.results) == 0 {
			compileFailed++
			continue
		}
		last := o.results[len(o.results)-1]
		if last.Stage == lowerCompareStageCompiled {
			t.Errorf("%s: the lowering or the reference crashed, exit code %d", name, o.exitCode)
			continue
		}
		if last.Stage == lowerCompareStageOutside {
			outside++
			reason := last.Failures[0].Reason
			if last.Failures[0].Opcode != "" {
				reason += " " + last.Failures[0].Opcode
			}
			firstReasons[reason]++
			for _, failure := range last.Failures {
				if failure.Opcode != "" {
					opcodes[failure.Opcode]++
				}
			}
			continue
		}
		inside++
		mismatch := ""
		switch {
		case last.Stage == lowerCompareStageMatch:
			matched++
			matchedNames = append(matchedNames, name)
			continue
		case last.Stage == lowerCompareStageMismatch:
			mismatch = last.Mismatch
		case o.timedOut:
			mismatch = "the old loop timed out"
		case o.exitCode != int(last.ReferenceCode):
			mismatch = fmt.Sprintf("the old loop exited with %d, the reference with %d", o.exitCode, last.ReferenceCode)
		default:
			matched++
			t.Logf("%s: both loops stopped with code %d", name, o.exitCode)
			continue
		}
		if reason, found := lowerCompareExcluded[name]; found {
			excluded++
			t.Logf("%s: excluded, %s: %s", name, reason, mismatch)
			continue
		}
		mismatched++
		t.Errorf("%s: %s", name, mismatch)
	}

	t.Logf("%d programs, %d did not compile, %d outside the subset, %d inside, %d match, %d mismatch, %d excluded",
		len(programs), compileFailed, outside, inside, matched, mismatched, excluded)
	t.Logf("matching programs: %s", strings.Join(matchedNames, ", "))
	t.Logf("first reason a program is outside: %s", lowerCompareTop(firstReasons, 12))
	t.Logf("opcodes outside the subset, by functions using them: %s", lowerCompareTop(opcodes, 20))
	for name, reason := range lowerCompareExcluded {
		if !slices.Contains(programs, name) {
			t.Errorf("excluded program %s is not in tests/ (%s)", name, reason)
		}
	}
}

// lowerCompareHarnessRun matches a t1-cxtest entry, its files and flags.
var lowerCompareHarnessRun = regexp.MustCompile(`t\.(?:Run|RunEx|RunWant)\("([^"]+)"`)

// lowerComparePrograms lists each .cx file of tests/ alone, then each entry
// of the t1-cxtest harness that compiles several files together and passes
// no flags. A program is its files relative to tests/, joined by spaces.
func lowerComparePrograms(t *testing.T) []string {
	tests := filepath.Join("..", "tests")
	var programs []string
	err := filepath.WalkDir(tests, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".cx") {
			name, err := filepath.Rel(tests, path)
			if err != nil {
				return err
			}
			programs = append(programs, filepath.ToSlash(name))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(programs)

	harness, err := os.ReadFile(filepath.Join("..", "t1-cxtest", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, match := range lowerCompareHarnessRun.FindAllStringSubmatch(string(harness), -1) {
		files := strings.Fields(match[1])
		plain := len(files) > 1
		for _, file := range files {
			plain = plain && strings.HasSuffix(file, ".cx")
		}
		if plain {
			programs = append(programs, strings.Join(files, " "))
		}
	}
	return programs
}

// lowerCompareTop formats the n largest counts.
func lowerCompareTop(counts map[string]int, n int) string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		return keys[i] < keys[j]
	})
	if len(keys) > n {
		keys = keys[:n]
	}
	parts := make([]string, len(keys))
	for i, key := range keys {
		parts[i] = fmt.Sprintf("%s %d", key, counts[key])
	}
	return strings.Join(parts, ", ")
}

// lowerCompareEmit prints a result line for the parent.
func lowerCompareEmit(t *testing.T, result lowerCompareResult) {
	line, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(lowerCompareMarker + string(line))
}

// lowerCompareChild compiles, lowers and compares one program.
func lowerCompareChild(t *testing.T, program string) {
	names := strings.Fields(program)
	sources := make([]string, len(names))
	for i, name := range names {
		source, err := os.ReadFile(filepath.Join("..", "tests", name))
		if err != nil {
			t.Fatal(err)
		}
		sources[i] = string(source)
		names[i] = filepath.Base(name)
	}
	initProgram()
	compileRunSources(t, names, sources)
	lowerCompareEmit(t, lowerCompareResult{Stage: lowerCompareStageCompiled})

	image, failures := lower.Program(actions.AST)
	if len(failures) > 0 {
		result := lowerCompareResult{Stage: lowerCompareStageOutside}
		for _, failure := range failures {
			entry := lowerCompareFailure{
				Function: actions.AST.CXFunctions[failure.FailureFunction].Name,
				Reason:   lower.ReasonNames[failure.FailureReason],
			}
			if failure.FailureReason == lower.ReasonOpcodeOutsideSubset {
				row, _ := constants.OpcodeLifecycleLiveByNumber(int(failure.FailureOpcode))
				entry.Opcode = row.Name
			}
			result.Failures = append(result.Failures, entry)
		}
		lowerCompareEmit(t, result)
		return
	}

	machine := referenceMachineNew(&image)
	referenceReset(machine)
	// A wrong jump can loop for ever, so the reference gets a time limit.
	done := make(chan int32, 1)
	go func() { done <- referenceRun(machine) }()
	var referenceCode int32
	select {
	case referenceCode = <-done:
	case <-time.After(time.Minute):
		lowerCompareEmit(t, lowerCompareResult{Stage: lowerCompareStageMismatch, Mismatch: "the reference did not finish in a minute"})
		os.Exit(1)
	}
	lowerCompareEmit(t, lowerCompareResult{Stage: lowerCompareStageLowered, ReferenceCode: referenceCode})

	oldOutput := lowerCompareCaptureStdout(t, func() {
		if err := execute.RunCompiled(actions.AST, 0, nil); err != nil {
			t.Fatal(err)
		}
	})

	result := lowerCompareResult{Stage: lowerCompareStageMatch, ReferenceCode: referenceCode}
	result.Mismatch = lowerCompareDiff(actions.AST, machine, oldOutput)
	if result.Mismatch != "" {
		result.Stage = lowerCompareStageMismatch
	}
	lowerCompareEmit(t, result)
}

// lowerCompareDiff describes the first difference between the old loop and
// the reference machine, or returns "".
func lowerCompareDiff(prgrm *ast.CXProgram, machine *referenceMachine, oldOutput []byte) string {
	if machine.stopCode != types.CX_SUCCESS {
		return fmt.Sprintf("the reference stopped with code %d in %s at pc %d", machine.stopCode,
			machine.image.ImageFuncNames[machine.stopFunction], machine.stopPc)
	}
	// A failed test prints its source line and message, which the reference
	// only counts. Those lines are compared by number.
	var oldLines [][]byte
	oldFailures := uint32(0)
	for _, line := range bytes.SplitAfter(oldOutput, []byte("\n")) {
		if bytes.Contains(line, []byte(": result was not equal to the expected value")) {
			oldFailures++
		} else {
			oldLines = append(oldLines, line)
		}
	}
	if oldFailures != machine.testFailures {
		return fmt.Sprintf("%d tests failed in the old loop, %d in the reference", oldFailures, machine.testFailures)
	}
	oldOutput = bytes.Join(oldLines, nil)
	oldMemory := prgrm.Memory
	if len(oldMemory) < len(machine.memory) {
		return fmt.Sprintf("the old memory is %d bytes, the reference %d", len(oldMemory), len(machine.memory))
	}
	for i := range machine.memory {
		if oldMemory[i] != machine.memory[i] {
			segment := "stack"
			if uint32(i) >= machine.image.ImageDataStart {
				segment = "data"
			}
			return fmt.Sprintf("memory differs at %d in the %s, old %d, reference %d", i, segment, oldMemory[i], machine.memory[i])
		}
	}
	if !bytes.Equal(oldOutput, machine.output) {
		return fmt.Sprintf("stdout differs, old %q, reference %q", oldOutput, machine.output)
	}
	return ""
}

// lowerCompareCaptureStdout returns what run writes to os.Stdout.
func lowerCompareCaptureStdout(t *testing.T, run func()) []byte {
	file, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	stdout := os.Stdout
	os.Stdout = file
	run()
	os.Stdout = stdout
	out, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	return out
}
