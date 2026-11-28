package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/skycoin/skycoin-cx/t4-cx-compiler-astbuild"
	"github.com/skycoin/skycoin-cx/t4-cx-library-compileropcodes"
	"github.com/skycoin/skycoin-cx/t4-cx-library-netopcodes"
	"github.com/skycoin/skycoin-cx/t5-cx-library-fileopcodes"
	"github.com/skycoin/skycoin-cx/t7-cx-runtime-libraries"
)

func TestMain(m *testing.M) {
	compileropcodes.LibraryCompilerSet(compilerLibrary{})
	if os.Getenv("CX_TEST_RUN_MAIN") == "1" {
		os.Args = append([]string{"cx"}, os.Args[1:]...)
		main()
		os.Exit(0)
	}
	// The children of TestExecutionRecords run outside the test framework,
	// as cx runs.
	if args := os.Getenv(executionRecordsImageEnv); args != "" {
		executionRecordsImageChild(executionRecordsChildArguments(args))
	}
	if args := os.Getenv(executionRecordsProgramHashEnv); args != "" {
		executionRecordsProgramHashChild(executionRecordsChildArguments(args))
	}
	os.Exit(m.Run())
}

// cx with no arguments used to exit silently instead of starting the REPL.
// See #145.
func TestNoArgsStartsRepl(t *testing.T) {
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "CX_TEST_RUN_MAIN=1")
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cx: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Bye!") {
		t.Fatalf("cx with no arguments did not start the REPL, output %q", out)
	}
}

func initProgram() {
	astbuild.ConstantsInit()
	fileopcodes.ConstantsInit(astbuild.ConstI32Add)
	netopcodes.ConstantsInit(astbuild.ConstI32Add)
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Also state.CXProgramMng.CXProgramRelease(gid), phase 4 #371. Site S0064 S0065.
	astbuild.AST = libraries.ProgramMake()
	libraries.PackagesRegister(astbuild.AST)
}
