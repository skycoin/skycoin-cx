package main

import (
	"bytes"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

const testSource = `package x

import (
	"fmt"
	"strings"
)

func plain(s string, b []byte, m map[string]int) {
	_ = s + "x"
	_ = s == "y"
	_ = string(b)
	_ = m[s]
	fmt.Println(1)
	_ = strings.ToUpper(s)
	_ = len(b)
	panic("constant")
	_ = append(b, "constant"...)
}

// VIOLATION STRHOT-001: P, test.
func coded(s string) string {
	return s + "!"
}

func line(b []byte) {
	// VIOLATION STRHOT-002: E, test.
	_ = string(b)
	_ = b[0]
}

// STRHOT REPORTER: test.
func reported(n int) {
	_ = fmt.Sprint(n)
	panic("constant")
}

// STRHOT REPORTER: test.
func reportedText(s string) {
	_ = s + "x"
}

func reportedCaller(n int) {
	reported(n)
}
`

func testFunctions(t *testing.T) map[string]*function {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", testSource, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	config := types.Config{Importer: importer.Default()}
	if _, err := config.Check("x", fset, []*ast.File{f}, info); err != nil {
		t.Fatal(err)
	}
	functions := map[string]*function{}
	scanFile(fset, info, f, "x.go", functions)
	return functions
}

func TestScanFile(t *testing.T) {
	functions := testFunctions(t)
	var got []string
	for _, s := range functions["x.plain"].sites {
		got = append(got, s.category)
	}
	want := "concat compare conv maplookup fmt strings"
	if strings.Join(got, " ") != want {
		t.Errorf("plain sites %q, want %q", strings.Join(got, " "), want)
	}
	if s := functions["x.coded"].sites; len(s) != 1 || s[0].code != "STRHOT-001" || s[0].commentClass != "P" {
		t.Errorf("coded sites %+v, want one STRHOT-001 P", s)
	}
	if s := functions["x.line"].sites; len(s) != 1 || s[0].code != "STRHOT-002" || s[0].commentClass != "E" {
		t.Errorf("line sites %+v, want one STRHOT-002 E", s)
	}
}

func TestReport(t *testing.T) {
	functions := testFunctions(t)
	for _, fn := range functions {
		fn.opcodeGoFunction = true
	}
	codes := map[string]codeEntry{"STRHOT-001": {class: "P"}, "STRHOT-002": {class: "O"}, "STRHOT-003": {class: "E"}}
	var out bytes.Buffer
	if rc := report(&out, functions, codes, reportCheck); rc != 1 {
		t.Errorf("report returned %d, want 1", rc)
	}
	for _, want := range []string{
		"x.go:9: plain: concat on the run path without a STRHOT code",
		`x.go:27: STRHOT-002 says class "E", strhot-codes.txt O`,
		"STRHOT-003 has no site on the run path",
		"x.reportedText: a STRHOT reporter takes a string parameter",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("report output lacks %q:\n%s", want, out.String())
		}
	}
}

func TestBuildTagsCxfx(t *testing.T) {
	for tags, want := range map[string]bool{"": false, "cxfx": true, "ptr64 cxfx": true, "ptr64,cxfx": true, "cxfxextra": false} {
		if got := buildTagsCxfx(tags); got != want {
			t.Errorf("buildTagsCxfx(%q) = %v, want %v", tags, got, want)
		}
	}
	codes := map[string]codeEntry{"STRHOT-001": {class: "O", cxfx: true}}
	codesCxfxOpen(codes)
	if codes["STRHOT-001"].cxfx {
		t.Error("a cxfx code stayed cxfx in the cxfx build")
	}
}

func TestReporterNotWalked(t *testing.T) {
	functions := testFunctions(t)
	functions["x.reportedCaller"].opcodeGoFunction = true
	for _, fn := range reachable(functions) {
		if fn.name == "x.reported" {
			t.Error("the walk entered the reporter x.reported")
		}
	}
	if problem := functions["x.reported"].reporterProblem; problem != "" {
		t.Errorf("x.reported breaks the reporter rule: %s", problem)
	}
}
