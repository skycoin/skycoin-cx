package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	"github.com/skycoin/skycoin-cx/t3-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-actions"
	lower "github.com/skycoin/skycoin-cx/t4-cx-compiler-lower"
	"github.com/skycoin/skycoin-cx/t4-execute"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
)

// lowerCompareProgramEnv names the program a child of TestLowerCompare
// checks. Each program runs in its own process because a compile error or a
// runtime error in the old loop ends the process.
const lowerCompareProgramEnv = "CX_LOWER_COMPARE_PROGRAM"

// lowerCompareMarker starts each result line a child prints.
const lowerCompareMarker = "LOWERCOMPARE "

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
	Stage    lowerCompareStage
	Failures []lowerCompareFailure
	VmCode   int32
	Mismatch string
}

// lowerCompareRun is one run of cx on a program.
type lowerCompareRun struct {
	stdout   []byte
	exitCode int
	timedOut bool
}

// TestLowerCompare lowers every CX program in tests/ and, for each program
// whose functions all lower, compares the memory after the old loop and
// after vm, steps 4 and 5 of #408. TestExecutionRecords compares their
// stdout and exit codes, runtime errors and failed tests included.
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
			child := lowerCompareProcess(nil, []string{lowerCompareProgramEnv + "=" + program}, "", "-test.run=^TestLowerCompare$")
			outcomes[i].exitCode = child.exitCode
			outcomes[i].timedOut = child.timedOut
			scanner := bufio.NewScanner(bytes.NewReader(child.stdout))
			scanner.Buffer(make([]byte, 1<<20), 1<<20)
			for scanner.Scan() {
				line, found := strings.CutPrefix(scanner.Text(), lowerCompareMarker)
				if !found {
					continue
				}
				result, err := lowerCompareParse(line)
				if err != nil {
					t.Errorf("%s: %v", program, err)
					continue
				}
				outcomes[i].results = append(outcomes[i].results, result)
			}
		}()
	}
	wait.Wait()

	var compileFailed, outside, inside, matched, mismatched, stopped int
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
			t.Errorf("%s: the lowering or vm crashed, exit code %d", name, o.exitCode)
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
		case last.Stage == lowerCompareStageMismatch:
			mismatch = last.Mismatch
		case last.Stage == lowerCompareStageLowered && last.VmCode == types.CX_SUCCESS && o.timedOut:
			mismatch = "the child timed out"
		}
		if mismatch != "" {
			mismatched++
			t.Errorf("%s: %s", name, mismatch)
			continue
		}
		matched++
		matchedNames = append(matchedNames, name)
		if last.VmCode != types.CX_SUCCESS {
			stopped++
			t.Logf("%s: vm exits with code %d", name, last.VmCode)
		}
	}

	t.Logf("%d programs, %d did not compile, %d outside the subset, %d inside, %d match (%d of them exit with an error code on vm), %d mismatch",
		len(programs), compileFailed, outside, inside, matched, stopped, mismatched)
	t.Logf("matching programs: %s", strings.Join(matchedNames, ", "))
	t.Logf("first reason a program is outside: %s", lowerCompareTop(firstReasons, 12))
	t.Logf("opcodes outside the subset, by functions using them: %s", lowerCompareTop(opcodes, 20))
}

// lowerCompareProcess runs the test binary with args and env added, in dir
// when it is not "", and returns its stdout and exit code.
func lowerCompareProcess(t *testing.T, env []string, dir string, args ...string) lowerCompareRun {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	executable, err := filepath.Abs(os.Args[0])
	if err != nil && t != nil {
		t.Error(err)
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = dir
	out, err := cmd.Output()
	run := lowerCompareRun{stdout: out, timedOut: ctx.Err() != nil}
	if exitError, ok := err.(*exec.ExitError); ok {
		run.exitCode = exitError.ExitCode()
	} else if err != nil && t != nil {
		t.Error(err)
	}
	return run
}

// lowerCompareClip shortens stdout for a message.
func lowerCompareClip(out []byte) []byte {
	if len(out) > 400 {
		return out[:400]
	}
	return out
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

// lowerCompareEmit prints a result line for the parent: the stage, the vm
// exit code, the quoted mismatch, then the function, reason and opcode of
// each failure, quoted.
func lowerCompareEmit(t *testing.T, result lowerCompareResult) {
	fields := []string{string(result.Stage), strconv.Itoa(int(result.VmCode)), strconv.Quote(result.Mismatch)}
	for _, failure := range result.Failures {
		fields = append(fields, strconv.Quote(failure.Function), strconv.Quote(failure.Reason), strconv.Quote(failure.Opcode))
	}
	fmt.Println(lowerCompareMarker + strings.Join(fields, " "))
}

// lowerCompareParse reads a result line lowerCompareEmit printed.
func lowerCompareParse(line string) (lowerCompareResult, error) {
	var result lowerCompareResult
	stage, rest, _ := strings.Cut(line, " ")
	codeText, rest, _ := strings.Cut(rest, " ")
	code, err := strconv.ParseInt(codeText, 10, 32)
	if err != nil {
		return result, fmt.Errorf("vm exit code: %w", err)
	}
	result.Stage, result.VmCode = lowerCompareStage(stage), int32(code)
	var quoted []string
	for rest != "" {
		prefix, err := strconv.QuotedPrefix(rest)
		if err != nil {
			return result, fmt.Errorf("not a quoted string: %q", rest)
		}
		value, _ := strconv.Unquote(prefix)
		quoted = append(quoted, value)
		rest = strings.TrimPrefix(rest[len(prefix):], " ")
	}
	if len(quoted)%3 != 1 {
		return result, fmt.Errorf("want a mismatch and three fields a failure, got %d fields", len(quoted))
	}
	result.Mismatch = quoted[0]
	for index := 1; index < len(quoted); index += 3 {
		result.Failures = append(result.Failures, lowerCompareFailure{Function: quoted[index], Reason: quoted[index+1], Opcode: quoted[index+2]})
	}
	return result, nil
}

// lowerCompareChild compiles and lowers one program, runs it on vm and then
// on the old loop, and compares the memory after both.
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

	image, failures, outside := core.ExecutorLower(actions.AST)
	if outside != "" {
		result := lowerCompareResult{Stage: lowerCompareStageOutside}
		for _, failure := range failures {
			entry := lowerCompareFailure{
				Function: actions.AST.CXProgramNameText(actions.AST.CXFunctions[failure.FailureFunction].Name),
				Reason:   lower.ReasonNames[failure.FailureReason],
			}
			if failure.FailureReason == lower.ReasonOpcodeOutsideSubset {
				row, _ := constants.OpcodeLifecycleLiveByNumber(int(failure.FailureOpcode))
				entry.Opcode = row.Name
			}
			result.Failures = append(result.Failures, entry)
		}
		if len(failures) == 0 {
			result.Failures = []lowerCompareFailure{{Reason: outside}}
		}
		lowerCompareEmit(t, result)
		return
	}

	// The loader refuses a program that uses a library operation its run
	// does not enable, before either loop runs (D119).
	core.CoreLibrariesEnabledResolve(actions.AST)
	if core.CoreOperationRefused(actions.AST) != 0 {
		lowerCompareEmit(t, lowerCompareResult{Stage: lowerCompareStageLowered, VmCode: types.CX_RUNTIME_LIBRARY_DISABLED})
		return
	}

	machine := vm.MachineNew(&image, core.ExecutorOutputSize)
	// A wrong jump can loop for ever, so vm gets a time limit.
	done := make(chan int32, 1)
	var vmCode int32
	lowerCompareCaptureStdout(t, func() {
		go func() { done <- core.ExecutorVmRunMachine(actions.AST, machine) }()
		select {
		case vmCode = <-done:
		case <-time.After(time.Minute):
			vmCode = -1
		}
	})
	if vmCode == -1 {
		lowerCompareEmit(t, lowerCompareResult{Stage: lowerCompareStageMismatch, Mismatch: "vm did not finish in a minute"})
		os.Exit(1)
	}
	lowerCompareEmit(t, lowerCompareResult{Stage: lowerCompareStageLowered, VmCode: vmCode})
	if vmCode != types.CX_SUCCESS && vmCode != types.CX_ASSERT {
		// The error print copied the vm state into the program, and the
		// runs of cx compare the output.
		return
	}

	lowerCompareCaptureStdout(t, func() {
		if err := execute.RunCompiled(actions.AST, 0, nil); err != nil {
			t.Fatal(err)
		}
	})

	result := lowerCompareResult{Stage: lowerCompareStageMatch, VmCode: vmCode}
	result.Mismatch = lowerCompareDiff(actions.AST, machine)
	if result.Mismatch != "" {
		result.Stage = lowerCompareStageMismatch
	}
	lowerCompareEmit(t, result)
}

// lowerCompareDiff describes the first difference between the memory after
// the old loop and after vm, or returns "". The vm scratch after the data
// segment has no counterpart in the old loop and is not compared.
func lowerCompareDiff(prgrm *ast.CXProgram, machine *vm.Machine) string {
	oldMemory := prgrm.Memory
	memory := machine.MachineMemory[:ir.ImageDataEnd(machine.MachineImage)]
	if len(oldMemory) < len(memory) {
		return fmt.Sprintf("the old memory is %d bytes, vm %d", len(oldMemory), len(memory))
	}
	for i := range memory {
		if oldMemory[i] != memory[i] {
			segment := "stack"
			if uint32(i) >= machine.MachineImage.ImageDataStart {
				segment = "data"
			}
			return fmt.Sprintf("memory differs at %d in the %s, old %d, vm %d", i, segment, oldMemory[i], memory[i])
		}
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
