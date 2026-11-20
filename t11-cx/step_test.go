package main

import (
	"io"
	"os"
	"testing"

	"github.com/skycoin/skycoin-cx/t4-actions"

	"github.com/skycoin/skycoin-cx/t4-execute"
	cxparsing "github.com/skycoin/skycoin-cx/t9-cxparsing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	done := make(chan []byte)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()
	defer func() { os.Stdout = stdout }()
	fn()
	w.Close()
	return string(<-done)
}

// RunCompiled with an op limit steps out of nested calls (#81), stops at
// the limit, and neither traces to stdout nor grows the program (#110).
func TestRunCompiledMaxOps(t *testing.T) {
	initProgram()
	src := "package main\nfunc f() {\n\tstr.print(\"in f\")\n}\nfunc main() {\n\tf()\n\tf()\n\tfor true {\n\t}\n}\n"
	if err := cxparsing.ParseSourceStrings([]string{src}, []string{"step.cx"}); err != nil {
		t.Fatal(err)
	}
	if err := cxparsing.AddInitFunction(actions.AST); err != nil {
		t.Fatal(err)
	}
	before := len(actions.AST.CXAtomicOps)
	out := captureStdout(t, func() {
		if err := execute.RunCompiled(actions.AST, 1000, nil); err != nil {
			t.Fatal(err)
		}
	})
	if out != "in f\nin f\n" {
		t.Errorf("stdout %q, want %q", out, "in f\nin f\n")
	}
	if after := len(actions.AST.CXAtomicOps); after != before {
		t.Errorf("CXAtomicOps grew from %d to %d", before, after)
	}
}
