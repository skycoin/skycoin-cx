package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
// TestSerializeRoundTrip lists. A program that compiles prints its AST and
// one that does not reports its errors, and neither panics. Both runs print
// the same, so the order of packages does not change from run to run.
func TestAstPrintsEveryProgram(t *testing.T) {
	tests := filepath.Join("..", "tests")
	programs := serializeRoundTripPrograms(t, tests)
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
