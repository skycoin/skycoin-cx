package repl_test

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	repl "github.com/skycoin/skycoin-cx/t10-cxrepl"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-opcodes"
	"github.com/skycoin/skycoin-cx/t5-cxos"
	cxinit "github.com/skycoin/skycoin-cx/t7-init"
)

const hello = "package main\nfunc main() {\n\tstr.print(\"hi\")\n}\n"

func TestMain(m *testing.M) {
	actions.InitConstants()
	cxos.InitConstants()
	actions.AST = cxinit.MakeProgram()
	cxinit.RegisterPackages(actions.AST)
	opcodes.RegisterOpcodes(actions.AST)
	os.Exit(m.Run())
}

func eval(t *testing.T, code string) string {
	t.Helper()
	stdout := os.Stdout
	start := time.Now()
	out := repl.Eval(code)
	if d := time.Since(start); d > 10*time.Second {
		t.Fatalf("Eval took %v", d)
	}
	if os.Stdout != stdout {
		os.Stdout = stdout
		t.Fatal("Eval left os.Stdout redirected")
	}
	return out
}

// The error returns used to leave os.Stdout pointing at the pipe. See #147.
func TestEvalErrorRestoresStdout(t *testing.T) {
	for _, code := range []string{
		"package foo\nfunc bar() {\n}\n",
		"package main\nfunc foo() {\n}\n",
	} {
		if out := eval(t, code); out == "" {
			t.Errorf("Eval(%q) returned no error", code)
		}
		if out := eval(t, hello); out != "hi\n" {
			t.Fatalf("Eval after %q: got %q, want %q", code, out, "hi\n")
		}
	}
}

// Output larger than the pipe buffer used to block Eval until it timed out.
// See #147.
func TestEvalLargeOutput(t *testing.T) {
	const line = "0123456789012345678901234567890123456789\n"
	out := eval(t, `package main
func main() {
	var i i32
	for i = 0; i < 5000; i++ {
		str.print("0123456789012345678901234567890123456789")
	}
}
`)
	if want := strings.Repeat(line, 5000); out != want {
		t.Fatalf("got %d bytes of output, want %d: %.80q", len(out), len(want), out)
	}
}

// Operators used to keep the argument index of the program that first used
// them, so the next program read the wrong argument. See #174.
func TestEvalNoStateBetweenPrograms(t *testing.T) {
	a := "package main\nfunc main() {\n\tvar i i32 = 1\n\ti++\n\tbool.print(i < 3)\n}\n"
	b := "package main\nfunc main() {\n\tvar f f32 = 1.5\n\tf32.print(f + 2.0)\n}\n"
	if out := eval(t, a); out != "true\n" {
		t.Fatalf("Eval(a): got %q, want %q", out, "true\n")
	}
	stderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	out := eval(t, b)
	os.Stderr = stderr
	w.Close()
	var errs strings.Builder
	io.Copy(&errs, r)
	if out != "3.5\n" || errs.Len() > 0 {
		t.Fatalf("Eval(b) after Eval(a): got %q, compile output %q", out, errs.String())
	}
}
