package ast

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
)

const objmodelTestCaseEnv = "CX_OBJMODEL_TEST_CASE"

// objmodelTestCaseName is the assert case a subprocess runs, empty in a
// normal run.
func objmodelTestCaseName() string {
	return os.Getenv(objmodelTestCaseEnv)
}

// objmodelTestAssertSubprocess runs childTest with caseName in a subprocess,
// since an assert exits, and checks that it crashed with want on stderr.
func objmodelTestAssertSubprocess(t *testing.T, childTest string, caseName string, want string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^"+childTest+"$")
	cmd.Env = append(os.Environ(), objmodelTestCaseEnv+"="+caseName)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != assert.AssertExitCode {
		t.Fatalf("exit = %v, want status %d\nstderr:\n%s", err, assert.AssertExitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "assert failed: ") || !strings.Contains(stderr.String(), want) {
		t.Errorf("stderr lacks an assert with %q:\n%s", want, stderr.String())
	}
}

func TestCxIrState(t *testing.T) {
	var state CxIrState
	state.CxIrStateInit()
	if state.CXProgramMng.CXProgramCap() != 1 || state.CXProgramMng.CXProgramMngGrowable {
		t.Fatal("the program manager must have a fixed Cap of 1")
	}
	if state.CXCallMng.CXCallCap() == 0 || state.CXCallMng.CXCallMngGrowable {
		t.Fatal("the call manager must have a fixed Cap")
	}
	lens := state.CxIrStateManagerLens()
	if len(lens) != 14 {
		t.Fatalf("%d managers, want 14", len(lens))
	}
	seen := map[string]bool{}
	for _, managerLen := range lens {
		if seen[managerLen.Name] || managerLen.Len != 0 {
			t.Fatalf("manager %s repeated or not empty", managerLen.Name)
		}
		seen[managerLen.Name] = true
	}

	program := state.CXProgramMng.CXProgramAlloc()
	state.CXProgramMng.CXProgramGet(program).Version = 1
	argument := state.CXArgumentMng.CXArgumentAlloc()
	state.CxIrStateTeardown1()
	if state.CXProgramMng.CXProgramLen() != 0 {
		t.Fatal("Teardown1 left the program live")
	}
	// Teardown1 does not walk the owners yet (#300), so release by hand.
	state.CXArgumentMng.CXArgumentRelease(argument)
	state.CxIrStateTeardown2()
	for _, managerLen := range state.CxIrStateManagerLens() {
		if managerLen.Len != 0 {
			t.Fatalf("manager %s has Len %d after teardown", managerLen.Name, managerLen.Len)
		}
	}
	if state.CXProgramMng.CXProgramCap() != 0 || state.CXCallMng.CXCallCap() != 0 {
		t.Fatal("Teardown2 left a manager array")
	}
}

func TestCxIrStateAssertChild(t *testing.T) {
	if objmodelTestCaseName() != "teardown2 with a live object" {
		t.Skip("subprocess only")
	}
	var state CxIrState
	state.CxIrStateInit()
	state.CXLineMng.CXLineAlloc()
	state.CxIrStateTeardown1()
	state.CxIrStateTeardown2()
	t.Fatal("returned after a failed assert")
}

func TestCxIrStateAssert(t *testing.T) {
	objmodelTestAssertSubprocess(t, "TestCxIrStateAssertChild", "teardown2 with a live object", "CXLineMngTeardown with 1 live objects")
}
