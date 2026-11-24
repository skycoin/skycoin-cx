package main

import (
	"bytes"
	"context"
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
)

// astPrintRun is one -ast run of a program.
type astPrintRun struct {
	stdout   []byte
	stderr   []byte
	exitCode int
}

// TestAstPrintsEveryProgram runs cx -ast twice on every program that
// astPrintPrograms lists. A program that compiles prints its AST and
// one that does not reports its errors, and neither panics. Both runs print
// the same, so the order of packages does not change from run to run.
func TestAstPrintsEveryProgram(t *testing.T) {
	tests := filepath.Join("..", "tests")
	programs := astPrintPrograms(t, tests)
	runs := make([][2]astPrintRun, len(programs))
	limit := make(chan struct{}, runtime.NumCPU())
	var wait sync.WaitGroup
	for i, program := range programs {
		args := []string{"-ast"}
		for _, arg := range strings.Fields(program) {
			if arg != "-ast" {
				args = append(args, arg)
			}
		}
		for run := range runs[i] {
			wait.Add(1)
			limit <- struct{}{}
			go func() {
				defer wait.Done()
				defer func() { <-limit }()
				runs[i][run] = astPrintProcess(t, tests, args)
			}()
		}
	}
	wait.Wait()

	printed := 0
	for i, program := range programs {
		first, second := runs[i][0], runs[i][1]
		switch {
		case bytes.Contains(first.stderr, []byte("goroutine ")) || first.exitCode != types.CX_SUCCESS && first.exitCode != types.CX_COMPILATION_ERROR:
			t.Errorf("-ast %s: exit code %d, stderr %q", program, first.exitCode, lowerCompareClip(first.stderr))
		case first.exitCode != second.exitCode || !bytes.Equal(first.stdout, second.stdout):
			t.Errorf("-ast %s: two runs print differently, exit codes %d and %d", program, first.exitCode, second.exitCode)
		case first.exitCode == types.CX_SUCCESS:
			printed++
		}
	}
	t.Logf("%d programs, %d print their AST, the rest do not compile", len(programs), printed)
}

// astPrintProcess runs the test binary as cx with args in dir.
func astPrintProcess(t *testing.T, dir string, args []string) astPrintRun {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Error(err)
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env = append(os.Environ(), "CX_TEST_RUN_MAIN=1")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	run := astPrintRun{stdout: stdout.Bytes(), stderr: stderr.Bytes()}
	if exitError, ok := err.(*exec.ExitError); ok {
		run.exitCode = exitError.ExitCode()
	} else if err != nil {
		t.Error(err)
	}
	return run
}

// astPrintPrograms lists each .cx file of tests/ and tests/examples/
// alone, then each t1-cxtest entry that passes flags or several files. A
// program is its cx arguments relative to tests/, joined by spaces.
func astPrintPrograms(t *testing.T, tests string) []string {
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
