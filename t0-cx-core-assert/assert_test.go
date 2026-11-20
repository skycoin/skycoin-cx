package assert

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

const assertTestCaseEnv = "CX_ASSERT_TEST_CASE"

// assertTestCaseLine prints the line after its caller, where the assert is.
func assertTestCaseLine() {
	_, _, line, _ := runtime.Caller(1)
	fmt.Printf("line %d\n", line+1)
}

func assertTestCaseRun(name string) {
	switch name {
	case "precondition":
		assertTestCaseLine()
		AssertPreconditionTrue(1 > 2, "count must be positive")
	case "invariant":
		assertTestCaseLine()
		AssertInvariantTrue(false, "stack pointer below frame")
	case "postcondition":
		assertTestCaseLine()
		AssertPostconditionTrue(false, "result not sorted")
	case "unreachable":
		assertTestCaseLine()
		AssertUnreachable("unknown type code 99")
	}
}

// TestAssertCrashChild is the subprocess body. It does nothing in a normal run.
func TestAssertCrashChild(t *testing.T) {
	name := os.Getenv(assertTestCaseEnv)
	if name == "" {
		t.Skip("subprocess only")
	}
	defer fmt.Fprintln(os.Stderr, "deferred function ran")
	assertTestCaseRun(name)
	fmt.Fprintln(os.Stderr, "returned after a failed assert")
}

func TestAssertCrash(t *testing.T) {
	for name, kind := range map[string]string{
		"precondition":  "precondition violated",
		"invariant":     "invariant violated",
		"postcondition": "postcondition violated",
		"unreachable":   "unreachable violated",
	} {
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestAssertCrashChild$")
			cmd.Env = append(os.Environ(), assertTestCaseEnv+"="+name)
			var stdout, stderr strings.Builder
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != AssertExitCode {
				t.Fatalf("exit = %v, want status %d\nstderr:\n%s", err, AssertExitCode, stderr.String())
			}
			var line int
			if _, err := fmt.Sscanf(stdout.String(), "line %d", &line); err != nil {
				t.Fatalf("child printed no line: %q", stdout.String())
			}
			out := stderr.String()
			for _, want := range []string{
				"assert failed: " + kind,
				fmt.Sprintf("assert_test.go:%d: ", line),
				"goroutine ",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("stderr lacks %q:\n%s", want, out)
				}
			}
			for _, bad := range []string{"deferred function ran", "returned after a failed assert"} {
				if strings.Contains(out, bad) {
					t.Errorf("stderr has %q, the assert did not end the process at once", bad)
				}
			}
		})
	}
}

func TestAssertTrueReturns(t *testing.T) {
	AssertPreconditionTrue(true, "unused")
	AssertInvariantTrue(true, "unused")
	AssertPostconditionTrue(true, "unused")
}
