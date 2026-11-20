package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-opcodes"
	"github.com/skycoin/skycoin-cx/t5-cxos"
	cxinit "github.com/skycoin/skycoin-cx/t7-init"
)

func TestMain(m *testing.M) {
	if os.Getenv("CX_TEST_RUN_MAIN") == "1" {
		os.Args = append([]string{"cx"}, os.Args[1:]...)
		main()
		os.Exit(0)
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
	actions.InitConstants()
	cxos.InitConstants()
	actions.AST = cxinit.MakeProgram()
	cxinit.RegisterPackages(actions.AST)
	opcodes.RegisterOpcodes(actions.AST)
}
