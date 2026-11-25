package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/skycoin/skycoin-cx/t0-cxhash"
	"github.com/skycoin/skycoin-cx/t0-textsections"
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-cx-runtime-printer"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
)

// executionRecordsFileName is the record file in tests/, see
// docs/ExecutionRecords.md.
const executionRecordsFileName = "execution-records.txt"

// executionRecordsOutputDirectory in tests/ holds the recorded stdout of
// every record, one file named by its output hash.
const executionRecordsOutputDirectory = "execution-records"

const executionRecordsOutputExtension = ".out"

// executionRecordsWriteEnv makes TestExecutionRecords write the record file
// from the old loop instead of checking it. make records sets it.
const executionRecordsWriteEnv = "CX_EXECUTION_RECORDS_WRITE"

// executionRecordsImageEnv holds the cx arguments, joined by
// executionRecordsArgumentSeparator, that a child compiles into the program
// image cx build appends, deserializes and runs.
const executionRecordsImageEnv = "CX_EXECUTION_RECORDS_IMAGE"

// executionRecordsProgramHashEnv holds the cx arguments of a program whose
// program hash a child prints.
const executionRecordsProgramHashEnv = "CX_EXECUTION_RECORDS_PROGRAM_HASH"

// executionRecordsArgumentSeparator joins arguments in an environment
// variable. No argument holds it.
const executionRecordsArgumentSeparator = "\x1f"

// executionRecordsChildFailed is the exit code of an image child whose
// serialize or deserialize failed, outside the CX exit codes.
const executionRecordsChildFailed = 125

// executionRecordsFormatVersion is the version of the record file format.
const executionRecordsFormatVersion = 2

// executionRecordHashName names the hash of every hash in the record file.
// It is the hash of CX, so decision D23 changes it in t0-cxhash for the
// cipher library and the records at once.
const executionRecordHashName = cxhash.CxHashName

// executionRecordHash is the one hash function of the execution records.
func executionRecordHash(data []byte) string {
	sum := cxhash.CxHashOfBytes(data)
	return hex.EncodeToString(sum[:])
}

// executionRecordForm is one way of running a program.
type executionRecordForm string

const (
	executionRecordFormOld      executionRecordForm = "old"
	executionRecordFormVm       executionRecordForm = "vm"
	executionRecordFormImageOld executionRecordForm = "image-old"
	executionRecordFormImageVm  executionRecordForm = "image-vm"
	executionRecordFormBuilt    executionRecordForm = "built"
	executionRecordFormNative   executionRecordForm = "native"
)

var executionRecordForms = []executionRecordForm{executionRecordFormOld, executionRecordFormVm, executionRecordFormImageOld, executionRecordFormImageVm, executionRecordFormBuilt, executionRecordFormNative}

// executionRecordCallStack starts the call stack vm prints after a runtime
// error. A native program prints the error line and no call stack, so its
// output is checked against the recorded output cut there.
var executionRecordCallStack = []byte("\n===Callstack===\n")

// executionRecord is one line of the records section.
type executionRecord struct {
	recordHash  string
	programHash string
	sourceHash  string
	inputHash   string
	outputHash  string
	exitCode    int
	name        string
}

// executionRecordInput is the input of one program, its os.Args, stdin and
// the replay record of the net library beside its first .cx file, if any.
type executionRecordInput struct {
	stdin      string
	arguments  []string
	replayPath string
	replay     []byte
}

// executionRecordExclusion is one line of the excluded section.
type executionRecordExclusion struct {
	reason string
	name   string
}

// executionRecordFile is the parsed record file.
type executionRecordFile struct {
	inputs   map[string]executionRecordInput
	records  []executionRecord
	excluded []executionRecordExclusion
}

// executionRecordReasons are the keys of the excluded section and their text.
var executionRecordReasons = map[string]string{
	"compile": "does not compile on its own: a compile error test, a file of a package, or a flag only test",
	"gui":     "opens a window (sys/gl, sys/glfw), needs the cxfx build and a display",
	"network": "uses the network and has no net replay record beside it, see docs/Networking.md",
	"random":  "prints random values",
	"clock":   "prints or depends on the clock",
	"process": "runs other cx programs",
	"shared":  "writes a fixed file that parallel runs share",
	"ast":     "prints the AST while it compiles, which a built program does not",
	"varies":  "two runs on the old loop print differently",
	"crash":   "the old loop crashes with a Go panic, which is a bug, not a result",
	"timeout": "does not finish in two minutes",
	"gostack": "prints a Go stack trace by design, which holds addresses",
	"path":    "prints the path of its source file, which has another separator on Windows",
}

// executionRecordSourcePatterns exclude programs by their source.
var executionRecordSourcePatterns = []struct {
	pattern *regexp.Regexp
	reason  string
}{
	{regexp.MustCompile(`"sys/(gl|glfw|gltext|al|cxfx)"`), "gui"},
	{regexp.MustCompile(`\bnet\.(TCP|UDP)(Listen|Dial)\b|"http"|\bhttp\.`), "network"},
	{regexp.MustCompile(`\.rand\b`), "random"},
	{regexp.MustCompile(`\btime\.(Unix|Now)`), "clock"},
	{regexp.MustCompile(`\bos\.Run\b`), "process"},
	{regexp.MustCompile(`\bos\.(Create|OpenReadWrite)\("[A-Za-z0-9_-]+\.tmp"\)`), "shared"},
}

// executionRecordExcludedByHand overrides the reason make records finds.
var executionRecordExcludedByHand = map[string]string{
	"--debug-go-stack test-debug-go-stack.cx": "gostack",
	"examples/errors.cx":                      "path",
	// Run without --enable=cipher, they stop with a runtime error that
	// names the file.
	"deprecated/test-cipher.cx": "path",
	"examples/cipher.cx":        "path",
}

// executionRecordFormOutcome is the run of one form, or why it did not
// apply. A run that applies with a note failed before the program ran.
type executionRecordFormOutcome struct {
	run        executionRecordRun
	applicable bool
	note       string
}

// executionRecordRun is one run of a process.
type executionRecordRun struct {
	stdout   []byte
	stderr   []byte
	exitCode int
	timedOut bool
}

// executionRecordInputHash hashes the input: the decimal argument count and
// a zero byte, each argument and a zero byte, then stdin, and for a program
// with a replay record a zero byte, its decimal length, a zero byte and the
// record (decision D53).
func executionRecordInputHash(input executionRecordInput) string {
	var data bytes.Buffer
	data.WriteString(strconv.Itoa(len(input.arguments)))
	data.WriteByte(0)
	for _, argument := range input.arguments {
		data.WriteString(argument)
		data.WriteByte(0)
	}
	data.WriteString(input.stdin)
	if input.replayPath != "" {
		data.WriteByte(0)
		data.WriteString(strconv.Itoa(len(input.replay)))
		data.WriteByte(0)
		data.Write(input.replay)
	}
	return executionRecordHash(data.Bytes())
}

// executionRecordRecordHash hashes the program, input and output hashes as
// 32 bytes each, then the exit code as a little endian int32.
func executionRecordRecordHash(record executionRecord) string {
	var data []byte
	for _, text := range []string{record.programHash, record.inputHash, record.outputHash} {
		digest, err := hex.DecodeString(text)
		if err != nil {
			return ""
		}
		data = append(data, digest...)
	}
	data = binary.LittleEndian.AppendUint32(data, uint32(int32(record.exitCode)))
	return executionRecordHash(data)
}

// executionRecordSourceHash hashes the .cx files a program names, in order,
// one after the other.
func executionRecordSourceHash(tests string, name string) (string, error) {
	var data []byte
	for _, arg := range strings.Fields(name) {
		if !strings.HasSuffix(arg, ".cx") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(tests, arg))
		if err != nil {
			return "", err
		}
		data = append(data, source...)
	}
	return executionRecordHash(data), nil
}

// executionRecordPrograms lists each .cx file under tests/ alone, then each
// t1-cxtest entry that passes flags or several files, without its executor
// flag, which the forms choose. A program is its cx arguments relative to
// tests/, joined by spaces.
func executionRecordPrograms(t *testing.T, tests string) []string {
	seen := map[string]bool{}
	err := filepath.WalkDir(tests, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".cx") {
			name, err := filepath.Rel(tests, path)
			if err != nil {
				return err
			}
			seen[filepath.ToSlash(name)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	harness, err := os.ReadFile(filepath.Join("..", "t1-cxtest", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, match := range lowerCompareHarnessRun.FindAllStringSubmatch(string(harness), -1) {
		var args []string
		files := 0
		for _, arg := range strings.Fields(match[1]) {
			if strings.HasPrefix(arg, "--executor=") {
				continue
			}
			if strings.HasSuffix(arg, ".cx") {
				files++
			}
			args = append(args, strings.TrimPrefix(arg, "./"))
		}
		if files > 0 && len(args) > 1 {
			seen[strings.Join(args, " ")] = true
		}
	}
	return slices.Sorted(maps.Keys(seen))
}

// executionRecordProcess runs executable in dir with stdin and returns its
// stdout, stderr and exit code.
func executionRecordProcess(t *testing.T, executable string, env []string, dir string, stdin string, args ...string) executionRecordRun {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	run := executionRecordRun{stdout: stdout.Bytes(), stderr: stderr.Bytes(), timedOut: ctx.Err() != nil}
	if exitError, ok := err.(*exec.ExitError); ok {
		run.exitCode = exitError.ExitCode()
	} else if err != nil {
		t.Error(err)
	}
	return run
}

// executionRecordHarness runs the test binary as cx.
type executionRecordHarness struct {
	t          *testing.T
	executable string
	tests      string
}

func executionRecordHarnessNew(t *testing.T) executionRecordHarness {
	tests, err := filepath.Abs(filepath.Join("..", "tests"))
	if err != nil {
		t.Fatal(err)
	}
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	return executionRecordHarness{t: t, executable: executable, tests: tests}
}

// cxArguments returns the cx arguments of name with the program arguments
// of input after them.
func (harness executionRecordHarness) cxArguments(name string, input executionRecordInput) []string {
	var args []string
	if input.replayPath != "" {
		args = append(args, "--net-replay="+input.replayPath)
	}
	return append(append(args, strings.Fields(name)...), input.arguments...)
}

// executionRecordReplaySuffix replaces .cx in the name of the replay record
// of the net library beside a program, see docs/Networking.md.
const executionRecordReplaySuffix = ".netreplay"

// inputOf returns the input of name, with the replay record beside its first
// .cx file when there is one.
func (harness executionRecordHarness) inputOf(inputs map[string]executionRecordInput, name string) executionRecordInput {
	input := inputs[name]
	for _, arg := range strings.Fields(name) {
		if !strings.HasSuffix(arg, ".cx") {
			continue
		}
		replayPath := strings.TrimSuffix(arg, ".cx") + executionRecordReplaySuffix
		if replay, err := os.ReadFile(filepath.Join(harness.tests, replayPath)); err == nil {
			input.replayPath, input.replay = replayPath, replay
		}
		break
	}
	return input
}

// programHash compiles name in a child and returns the hash of the image cx
// build would append, or the child's exit code when it does not compile.
func (harness executionRecordHarness) programHash(name string) (string, executionRecordRun) {
	run := executionRecordProcess(harness.t, harness.executable, []string{executionRecordsProgramHashEnv + "=" + strings.Join(strings.Fields(name), executionRecordsArgumentSeparator)}, harness.tests, "")
	if run.exitCode != types.CX_SUCCESS {
		return "", run
	}
	lines := strings.Fields(string(run.stdout))
	if len(lines) == 0 {
		return "", run
	}
	return lines[len(lines)-1], run
}

// cx runs cx with the executor on name.
func (harness executionRecordHarness) cx(executor string, name string, input executionRecordInput) executionRecordRun {
	args := append([]string{"--executor=" + executor}, harness.cxArguments(name, input)...)
	return executionRecordProcess(harness.t, harness.executable, []string{"CX_TEST_RUN_MAIN=1"}, harness.tests, input.stdin, args...)
}

// image runs name with the executor after a round trip through the image
// cx build appends.
func (harness executionRecordHarness) image(executor string, name string, input executionRecordInput) executionRecordRun {
	args := append([]string{"--executor=" + executor}, harness.cxArguments(name, input)...)
	return executionRecordProcess(harness.t, harness.executable, []string{executionRecordsImageEnv + "=" + strings.Join(args, executionRecordsArgumentSeparator)}, harness.tests, input.stdin)
}

// executionRecordVmOutside reports whether run is vm refusing a program that
// does not lower.
func executionRecordVmOutside(run executionRecordRun) bool {
	return run.exitCode == types.CX_RUNTIME_NOT_IMPLEMENTED && bytes.Contains(run.stderr, []byte("cannot run on vm:"))
}

// executionRecordGoPanic reports whether run ended in a Go panic.
func executionRecordGoPanic(run executionRecordRun) bool {
	return bytes.Contains(run.stderr, []byte("goroutine ")) && bytes.Contains(run.stderr, []byte("panic"))
}

// executionRecordRunFlag returns the first cx flag of name that only changes
// how cx runs a program, or "". A built program runs with their defaults.
func executionRecordRunFlag(name string) string {
	for _, arg := range strings.Fields(name) {
		flag, _, _ := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if strings.HasPrefix(arg, "-") && buildTestRunFlags[flag] {
			return arg
		}
	}
	return ""
}

// executionRecordFirstDifference returns the first line, counted from 1,
// where got and want differ, and both lines.
func executionRecordFirstDifference(got, want []byte) (int, string, string) {
	gotLines := strings.SplitAfter(string(got), "\n")
	wantLines := strings.SplitAfter(string(want), "\n")
	for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
		gotLine, wantLine := "(end of output)", "(end of output)"
		if i < len(gotLines) {
			gotLine = gotLines[i]
		}
		if i < len(wantLines) {
			wantLine = wantLines[i]
		}
		if gotLine != wantLine {
			return i + 1, gotLine, wantLine
		}
	}
	return 0, "", ""
}

// executionRecordFileRead parses the record file, in the text form of
// t0-textsections.
func executionRecordFileRead(path string) (executionRecordFile, error) {
	file := executionRecordFile{inputs: map[string]executionRecordInput{}}
	content, err := os.ReadFile(path)
	if err != nil {
		return file, err
	}
	sections, err := textsections.TextSectionsRead(content)
	if err != nil {
		return file, fmt.Errorf("%s: %w", path, err)
	}
	for _, section := range sections {
		for _, entry := range section.TextSectionEntries {
			line := entry.TextSectionEntryText
			fail := func(problem string) error {
				return fmt.Errorf("%s:%d: %s: %q", path, entry.TextSectionEntryLine, problem, line)
			}
			fields := strings.Fields(line)
			switch section.TextSectionName {
			case "format":
				if len(fields) != 2 || fields[0] != strconv.Itoa(executionRecordsFormatVersion) || fields[1] != executionRecordHashName {
					return file, fail(fmt.Sprintf("want format %d and hash %s", executionRecordsFormatVersion, executionRecordHashName))
				}
			case "inputs":
				var input executionRecordInput
				rest := line
				quoted := func() (string, bool) {
					prefix, err := strconv.QuotedPrefix(rest)
					if err != nil {
						return "", false
					}
					rest = strings.TrimLeft(rest[len(prefix):], " ")
					value, err := strconv.Unquote(prefix)
					return value, err == nil
				}
				stdin, ok := quoted()
				if !ok {
					return file, fail("stdin is not a quoted string")
				}
				input.stdin = stdin
				countText, after, _ := strings.Cut(rest, " ")
				count, err := strconv.Atoi(countText)
				if err != nil {
					return file, fail("no argument count")
				}
				rest = after
				for range count {
					argument, ok := quoted()
					if !ok {
						return file, fail("an argument is not a quoted string")
					}
					input.arguments = append(input.arguments, argument)
				}
				if rest == "" {
					return file, fail("no program name")
				}
				file.inputs[rest] = input
			case "records":
				if len(fields) < 7 {
					return file, fail("want 7 columns")
				}
				exitCode, err := strconv.Atoi(fields[5])
				if err != nil {
					return file, fail("the exit code is not a number")
				}
				file.records = append(file.records, executionRecord{
					recordHash: fields[0], programHash: fields[1], sourceHash: fields[2], inputHash: fields[3],
					outputHash: fields[4], exitCode: exitCode, name: strings.Join(fields[6:], " "),
				})
			case "excluded":
				if len(fields) < 2 {
					return file, fail("want a reason and a name")
				}
				file.excluded = append(file.excluded, executionRecordExclusion{reason: fields[0], name: strings.Join(fields[1:], " ")})
			case "reasons":
			default:
				return file, fail("a line outside a known section")
			}
		}
	}
	return file, nil
}

// executionRecordFileWrite writes the record file.
func executionRecordFileWrite(path string, file executionRecordFile) error {
	section := func(name string, columns string, lines []string) textsections.TextSection {
		entries := make([]textsections.TextSectionEntry, len(lines))
		for index, line := range lines {
			entries[index].TextSectionEntryText = line
		}
		return textsections.TextSection{TextSectionName: name, TextSectionColumns: strings.Fields(columns), TextSectionEntries: entries}
	}
	var inputs, records, reasons, excluded []string
	for _, name := range slices.Sorted(maps.Keys(file.inputs)) {
		input := file.inputs[name]
		line := strconv.Quote(input.stdin) + " " + strconv.Itoa(len(input.arguments))
		for _, argument := range input.arguments {
			line += " " + strconv.Quote(argument)
		}
		inputs = append(inputs, line+" "+name)
	}
	for _, record := range file.records {
		records = append(records, fmt.Sprintf("%s %s %s %s %s %d %s", record.recordHash, record.programHash, record.sourceHash, record.inputHash, record.outputHash, record.exitCode, record.name))
	}
	for _, key := range slices.Sorted(maps.Keys(executionRecordReasons)) {
		reasons = append(reasons, key+" "+executionRecordReasons[key])
	}
	for _, exclusion := range file.excluded {
		excluded = append(excluded, exclusion.reason+" "+exclusion.name)
	}
	content := textsections.TextSectionsWrite([]string{
		"Execution records of the CX programs in tests/, see docs/ExecutionRecords.md.",
		"make records writes this file from the old loop and keeps the inputs",
		"section, which is edited by hand. Review the diff before committing it.",
	}, []textsections.TextSection{
		section("format", "version hash", []string{fmt.Sprintf("%d %s", executionRecordsFormatVersion, executionRecordHashName)}),
		section("inputs", "stdin argument-count arguments name", inputs),
		section("records", "record-hash program-hash source-hash input-hash output-hash exit-code name", records),
		section("reasons", "key text", reasons),
		section("excluded", "reason name", excluded),
	})
	return os.WriteFile(path, content, 0o644)
}

// TestExecutionRecords checks every execution record of
// tests/execution-records.txt: each program runs on its input in every form
// and must print the recorded stdout and exit with the recorded code. With
// CX_EXECUTION_RECORDS_WRITE=1, as make records runs it, it writes the file
// from the old loop instead. See docs/ExecutionRecords.md.
func TestExecutionRecords(t *testing.T) {
	harness := executionRecordHarnessNew(t)
	path := filepath.Join(harness.tests, executionRecordsFileName)
	if os.Getenv(executionRecordsWriteEnv) == "1" {
		executionRecordsWrite(t, harness, path)
		return
	}
	if testing.Short() {
		t.Skip("runs every recorded program in every form")
	}
	start := time.Now()
	file, err := executionRecordFileRead(path)
	if err != nil {
		t.Fatal(err)
	}

	listed := map[string]bool{}
	for _, record := range file.records {
		listed[record.name] = true
	}
	for _, exclusion := range file.excluded {
		if _, found := executionRecordReasons[exclusion.reason]; !found {
			t.Errorf("excluded %s: unknown reason %s", exclusion.name, exclusion.reason)
		}
		listed[exclusion.name] = true
	}
	programs := executionRecordPrograms(t, harness.tests)
	for _, program := range programs {
		if !listed[program] {
			t.Errorf("%s is neither recorded nor excluded, run make records", program)
		}
		delete(listed, program)
	}
	for name := range listed {
		t.Errorf("%s is listed in %s but is no program of tests/, run make records", name, executionRecordsFileName)
	}

	dir := t.TempDir()
	cores := buildTestCores(t, dir)
	clang := buildLlvmClang()

	type outcome struct {
		programHash string
		forms       map[executionRecordForm]executionRecordFormOutcome
		payload     int
	}
	outcomes := make([]outcome, len(file.records))
	limit := make(chan struct{}, runtime.NumCPU())
	var wait sync.WaitGroup
	for i, record := range file.records {
		wait.Add(1)
		limit <- struct{}{}
		go func() {
			defer wait.Done()
			defer func() { <-limit }()
			input := harness.inputOf(file.inputs, record.name)
			forms := map[executionRecordForm]executionRecordFormOutcome{}
			outcomes[i].programHash, _ = harness.programHash(record.name)
			forms[executionRecordFormOld] = executionRecordFormOutcome{run: harness.cx("old", record.name, input), applicable: true}
			vmRun := harness.cx("vm", record.name, input)
			if executionRecordVmOutside(vmRun) {
				forms[executionRecordFormVm] = executionRecordFormOutcome{note: "does not lower"}
				forms[executionRecordFormImageVm] = executionRecordFormOutcome{note: "does not lower"}
			} else {
				forms[executionRecordFormVm] = executionRecordFormOutcome{run: vmRun, applicable: true}
				forms[executionRecordFormImageVm] = executionRecordFormOutcome{run: harness.image("vm", record.name, input), applicable: true}
			}
			forms[executionRecordFormImageOld] = executionRecordFormOutcome{run: harness.image("old", record.name, input), applicable: true}
			switch {
			case executionRecordVmOutside(vmRun):
				forms[executionRecordFormNative] = executionRecordFormOutcome{note: "does not lower"}
			case executionRecordRunFlag(record.name) != "":
				forms[executionRecordFormNative] = executionRecordFormOutcome{note: "passes the run time flag " + executionRecordRunFlag(record.name)}
			case clang == "":
				forms[executionRecordFormNative] = executionRecordFormOutcome{note: "no clang on this machine, set " + buildLlvmClangEnv + " or install clang"}
			default:
				forms[executionRecordFormNative] = harness.native(dir, i, record.name, input)
			}

			if flag := executionRecordRunFlag(record.name); flag != "" {
				forms[executionRecordFormBuilt] = executionRecordFormOutcome{note: "passes the run time flag " + flag}
			} else {
				output := filepath.Join(dir, fmt.Sprintf("program-%d", i))
				if runtime.GOOS == "windows" {
					output += ".exe"
				}
				buildArgs := append([]string{buildCommand}, strings.Fields(record.name)...)
				buildArgs = append(buildArgs, "-o", output, "--core", dir)
				build := executionRecordProcess(t, harness.executable, []string{"CX_TEST_RUN_MAIN=1"}, harness.tests, "", buildArgs...)
				switch {
				case build.exitCode != types.CX_SUCCESS && bytes.Contains(build.stderr, []byte("opcodes it does not include")):
					forms[executionRecordFormBuilt] = executionRecordFormOutcome{note: "uses opcodes outside the core"}
				case build.exitCode != types.CX_SUCCESS:
					forms[executionRecordFormBuilt] = executionRecordFormOutcome{run: build, applicable: true, note: "cx build failed"}
				default:
					if program, fileError := core.CoreTrailerRead(output); fileError == nil {
						outcomes[i].payload = len(program)
					}
					var env []string
					if input.replayPath != "" {
						env = append(env, "CX_NET_REPLAY="+input.replayPath)
					}
					forms[executionRecordFormBuilt] = executionRecordFormOutcome{run: executionRecordProcess(t, output, env, harness.tests, input.stdin, input.arguments...), applicable: true}
					os.Remove(output)
				}
			}
			outcomes[i].forms = forms
		}()
	}
	wait.Wait()

	checked := map[executionRecordForm]int{}
	notApplicable := map[string]int{}
	var mismatches, staleProgram, staleSource int
	var payloads []int
	for i, record := range file.records {
		result := outcomes[i]
		input := harness.inputOf(file.inputs, record.name)
		if inputHash := executionRecordInputHash(input); inputHash != record.inputHash {
			t.Errorf("record %s: input hash %s, the inputs section gives %s, run make records", record.name, record.inputHash, inputHash)
		}
		if recordHash := executionRecordRecordHash(record); recordHash != record.recordHash {
			t.Errorf("record %s: record hash %s, its columns give %s", record.name, record.recordHash, recordHash)
		}
		if result.programHash != record.programHash {
			staleProgram++
		}
		if sourceHash, err := executionRecordSourceHash(harness.tests, record.name); err != nil || sourceHash != record.sourceHash {
			staleSource++
		}
		recorded, err := os.ReadFile(filepath.Join(harness.tests, executionRecordsOutputDirectory, record.outputHash+executionRecordsOutputExtension))
		if err != nil {
			t.Errorf("record %s: no recorded stdout: %v, run make records", record.name, err)
		} else if executionRecordHash(recorded) != record.outputHash {
			t.Errorf("record %s: the recorded stdout does not hash to its name %s", record.name, record.outputHash)
		}
		matches := func(form executionRecordForm, run executionRecordRun) bool {
			if run.timedOut || run.exitCode != record.exitCode {
				return false
			}
			if cut := bytes.Index(recorded, executionRecordCallStack); form == executionRecordFormNative && cut >= 0 {
				return bytes.Equal(run.stdout, append(recorded[:cut:cut], '\n'))
			}
			return executionRecordHash(run.stdout) == record.outputHash
		}
		for _, form := range executionRecordForms {
			outcome := result.forms[form]
			if !outcome.applicable {
				notApplicable[string(form)+" "+outcome.note]++
				continue
			}
			checked[form]++
			if matches(form, outcome.run) {
				continue
			}
			mismatches++
			run := outcome.run
			var problem string
			switch {
			case outcome.note != "":
				problem = fmt.Sprintf("%s, exit code %d, stderr %q", outcome.note, run.exitCode, lowerCompareClip(run.stderr))
			case run.timedOut:
				problem = "timed out"
			case run.exitCode != record.exitCode:
				problem = fmt.Sprintf("exit code %d, recorded %d", run.exitCode, record.exitCode)
			default:
				problem = "stdout differs from the record"
			}
			if line, got, want := executionRecordFirstDifference(run.stdout, recorded); line != 0 {
				problem += fmt.Sprintf(", first differing line %d: %q, recorded %q", line, got, want)
			}
			t.Errorf("record %s, form %s: %s", record.name, form, problem)
		}
		if result.payload > 0 {
			payloads = append(payloads, result.payload)
		}
	}

	var coreSizes []string
	for _, name := range slices.Sorted(maps.Keys(cores)) {
		if info, err := os.Stat(cores[name]); err == nil {
			coreSizes = append(coreSizes, fmt.Sprintf("%s %d", name, info.Size()))
		}
	}
	var checkedText, skippedText []string
	for _, form := range executionRecordForms {
		checkedText = append(checkedText, fmt.Sprintf("%s %d", form, checked[form]))
	}
	for _, key := range slices.Sorted(maps.Keys(notApplicable)) {
		skippedText = append(skippedText, fmt.Sprintf("%s %d", key, notApplicable[key]))
	}
	sort.Ints(payloads)
	t.Logf("%d records, %d excluded, %d mismatches in %s", len(file.records), len(file.excluded), mismatches, time.Since(start).Round(time.Second))
	t.Logf("runs checked by form: %s", strings.Join(checkedText, ", "))
	t.Logf("forms not run: %s", strings.Join(skippedText, ", "))
	if staleProgram > 0 || staleSource > 0 {
		t.Logf("%d records have another program hash and %d another source hash than the compiler or the source gives now, make records refreshes them", staleProgram, staleSource)
	}
	t.Logf("cores in bytes: %s", strings.Join(coreSizes, ", "))
	if len(payloads) > 0 {
		t.Logf("appended program bytes: min %d, median %d, max %d", payloads[0], payloads[len(payloads)/2], payloads[len(payloads)-1])
	}
}

// executionRecordsWrite runs every program of tests/ twice on the old loop
// and writes the record file, keeping the inputs section of the old one.
func executionRecordsWrite(t *testing.T, harness executionRecordHarness, path string) {
	file := executionRecordFile{inputs: map[string]executionRecordInput{}}
	if old, err := executionRecordFileRead(path); err == nil {
		file.inputs = old.inputs
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	programs := executionRecordPrograms(t, harness.tests)
	type outcome struct {
		record executionRecord
		stdout []byte
		reason string
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
			outcomes[i].reason = executionRecordReason(t, harness, program, harness.inputOf(file.inputs, program), &outcomes[i].record, &outcomes[i].stdout)
		}()
	}
	wait.Wait()
	var stdouts [][]byte
	for i := range programs {
		if reason, found := executionRecordExcludedByHand[programs[i]]; found {
			outcomes[i].reason = reason
		}
		if outcomes[i].reason != "" {
			file.excluded = append(file.excluded, executionRecordExclusion{reason: outcomes[i].reason, name: programs[i]})
			continue
		}
		file.records = append(file.records, outcomes[i].record)
		stdouts = append(stdouts, outcomes[i].stdout)
	}
	for name := range file.inputs {
		if !slices.Contains(programs, name) {
			t.Errorf("the inputs section names %s, which is no program of tests/", name)
		}
	}
	if t.Failed() {
		return
	}
	if err := executionRecordFileWrite(path, file); err != nil {
		t.Fatal(err)
	}
	outputs := filepath.Join(harness.tests, executionRecordsOutputDirectory)
	if err := os.RemoveAll(outputs); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outputs, 0o755); err != nil {
		t.Fatal(err)
	}
	for i, record := range file.records {
		if err := os.WriteFile(filepath.Join(outputs, record.outputHash+executionRecordsOutputExtension), stdouts[i], 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("wrote %d records and %d exclusions to %s", len(file.records), len(file.excluded), path)
}

// executionRecordReason fills record and stdout for program from two runs
// on the old loop, or returns the reason it is excluded.
func executionRecordReason(t *testing.T, harness executionRecordHarness, program string, input executionRecordInput, record *executionRecord, stdout *[]byte) string {
	for _, arg := range strings.Fields(program) {
		if arg == "-ast" {
			return "ast"
		}
		if !strings.HasSuffix(arg, ".cx") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(harness.tests, arg))
		if err != nil {
			t.Fatal(err)
		}
		for _, exclusion := range executionRecordSourcePatterns {
			// A network program with a replay record runs without the network.
			if exclusion.reason == "network" && input.replayPath != "" {
				continue
			}
			if exclusion.pattern.Match(source) {
				return exclusion.reason
			}
		}
	}
	programHash, compile := harness.programHash(program)
	if programHash == "" {
		if compile.timedOut {
			return "timeout"
		}
		return "compile"
	}
	first := harness.cx("old", program, input)
	second := harness.cx("old", program, input)
	switch {
	case first.timedOut || second.timedOut:
		return "timeout"
	case executionRecordGoPanic(first) || executionRecordGoPanic(second):
		return "crash"
	case first.exitCode != second.exitCode || !bytes.Equal(first.stdout, second.stdout):
		return "varies"
	}
	sourceHash, err := executionRecordSourceHash(harness.tests, program)
	if err != nil {
		t.Fatal(err)
	}
	*record = executionRecord{
		programHash: programHash,
		sourceHash:  sourceHash,
		inputHash:   executionRecordInputHash(input),
		outputHash:  executionRecordHash(first.stdout),
		exitCode:    first.exitCode,
		name:        program,
	}
	record.recordHash = executionRecordRecordHash(*record)
	*stdout = first.stdout
	return ""
}

// executionRecordsProgramHashChild prints the program hash of the program
// args name and exits, with CX_COMPILATION_ERROR when it does not compile.
func executionRecordsProgramHashChild(args []string) {
	options, _, sourceCode, fileNames, start := runStart(args)
	if !start || !parseProgram(options, fileNames, sourceCode) {
		os.Exit(types.CX_COMPILATION_ERROR)
	}
	image, err := buildProgramImage()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(executionRecordsChildFailed)
	}
	fmt.Println(executionRecordHash(image))
	os.Exit(types.CX_SUCCESS)
}

// executionRecordsImageChild runs args as cx does, except that the compiled
// program goes through the image cx build appends, as a built program
// loads it, before it runs.
func executionRecordsImageChild(args []string) {
	options, cxArgs, sourceCode, fileNames, start := runStart(args)
	if !start || !parseProgram(options, fileNames, sourceCode) {
		os.Exit(types.CX_COMPILATION_ERROR)
	}
	image, err := buildProgramImage()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(executionRecordsChildFailed)
	}
	prgrm, err := ast.Deserialize(image, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(executionRecordsChildFailed)
	}
	printer.PrinterCompileErrorsRegister(prgrm)
	actions.AST = prgrm
	runProgram(options, cxArgs, sourceCode)
	os.Exit(types.CX_SUCCESS)
}

// executionRecordsChildArguments splits an argument environment variable.
func executionRecordsChildArguments(value string) []string {
	return strings.Split(value, executionRecordsArgumentSeparator)
}

// native builds name with cx build --target=llvm into dir and runs the
// executable on input. A program that calls a system library has no native
// form.
func (harness executionRecordHarness) native(dir string, index int, name string, input executionRecordInput) executionRecordFormOutcome {
	output := filepath.Join(dir, fmt.Sprintf("native-%d", index))
	if runtime.GOOS == "windows" {
		output += ".exe"
	}
	buildArgs := append([]string{buildCommand, "--target=" + buildTargetLlvm}, strings.Fields(name)...)
	buildArgs = append(buildArgs, "-o", output)
	build := executionRecordProcess(harness.t, harness.executable, []string{"CX_TEST_RUN_MAIN=1"}, harness.tests, "", buildArgs...)
	switch {
	case build.exitCode != types.CX_SUCCESS:
		return executionRecordFormOutcome{run: build, applicable: true, note: "cx build --target=llvm failed"}
	}
	defer os.Remove(output)
	defer os.Remove(strings.TrimSuffix(output, ".exe") + buildLlvmExtension)
	var env []string
	if input.replayPath != "" {
		env = append(env, "CX_NET_REPLAY="+input.replayPath)
	}
	return executionRecordFormOutcome{run: executionRecordProcess(harness.t, output, env, harness.tests, input.stdin, input.arguments...), applicable: true}
}
