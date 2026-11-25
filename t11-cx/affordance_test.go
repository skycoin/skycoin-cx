package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
	"github.com/skycoin/skycoin-cx/t2-globals"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	cxparsing "github.com/skycoin/skycoin-cx/t9-cxparsing"
)

// affordanceSource declares a package with a struct, a global and two
// functions, and body as the body of main.
func affordanceSource(body string) string {
	return "package main\n\ntype Point struct {\n\tx i32\n\ty i32\n}\n\nvar g i32\n\nfunc area(p Point) (a i32) {\n\tvar w i32\n\ta = w\n}\n\nfunc main() {\n\tvar a aff\n" + body + "\n}\n"
}

// affordanceCompile compiles source and returns the compile errors as text.
func affordanceCompile(t *testing.T, source string) []string {
	t.Helper()
	initProgram()
	globals.FoundCompileErrors = false
	defer func() { globals.FoundCompileErrors = false }()
	cxparsing.ParseSourceStrings([]string{source}, []string{"aff.cx"})
	var errors []string
	for _, record := range actions.AST.CxIrState.CompileErrorList.CompileErrorListRecords {
		text := fmt.Sprintf("%d", record.CompileErrorRecordCode)
		if record.CompileErrorRecordCode == ast.CompileErrorCodeAffordanceInvalid || record.CompileErrorRecordCode == ast.CompileErrorCodeSyntax {
			text = actions.AST.CXProgramNameText(stringpool.StringGid(record.CompileErrorRecordOperands[0]))
		}
		errors = append(errors, fmt.Sprintf("%d:%d %s", record.CompileErrorRecordLine, record.CompileErrorRecordColumn, text))
	}
	return errors
}

// Each form of the literal parses to the value its clauses name. The
// literal is the last data of the program, after the global g.
func TestAffordanceLiteralParse(t *testing.T) {
	type expected struct {
		kind     ast.AffordanceKind
		item     string
		variable string
	}
	for _, test := range []struct {
		literal string
		want    expected
	}{
		{"#{}", expected{ast.AffordanceKindProgram, "", ""}},
		{"#{\n\t}", expected{ast.AffordanceKindProgram, "", ""}},
		{"#{mem(data)}", expected{ast.AffordanceKindMemory, "data", ""}},
		{"#{mem(stack)}", expected{ast.AffordanceKindMemory, "stack", ""}},
		{"#{mem(heap);}", expected{ast.AffordanceKindMemory, "heap", ""}},
		{"#{pkg(main)}", expected{ast.AffordanceKindPackage, "", ""}},
		{"#{pkg(main) fn(area)}", expected{ast.AffordanceKindFunction, "area", ""}},
		{"#{fn(area)}", expected{ast.AffordanceKindFunction, "area", ""}},
		{"#{pkg(main); strct(Point);}", expected{ast.AffordanceKindStruct, "Point", ""}},
		{"#{\n\t\tpkg(main)\n\t\tvar(g)\n\t}", expected{ast.AffordanceKindGlobal, "", "g"}},
		{"#{var(g)}", expected{ast.AffordanceKindGlobal, "", "g"}},
		{"#{pkg(main) fn(area) var(p)}", expected{ast.AffordanceKindLocal, "area", "p"}},
		{"#{fn(area) var(a)}", expected{ast.AffordanceKindLocal, "area", "a"}},
		{"#{fn(area) var(w)}", expected{ast.AffordanceKindLocal, "area", "w"}},
		{"#{fn(main) var(a)}", expected{ast.AffordanceKindLocal, "main", "a"}},
	} {
		t.Run(test.literal, func(t *testing.T) {
			if errors := affordanceCompile(t, affordanceSource("\ta = "+test.literal)); len(errors) > 0 {
				t.Fatalf("compile errors %q", errors)
			}
			prgrm := actions.AST
			value := ast.AffordanceValueRead(prgrm.Memory, prgrm.Data.StartsAt+prgrm.Data.Size-types.AFF_SIZE)
			pkg, err := prgrm.GetPackage(prgrm.CxIrState.CxIrStateNameMain)
			if err != nil {
				t.Fatal(err)
			}
			got := expected{kind: value.AffordanceValueKind}
			switch value.AffordanceValueKind {
			case ast.AffordanceKindMemory:
				got.item = ast.AffordanceMemoryNames[value.AffordanceValueItem]
			case ast.AffordanceKindFunction, ast.AffordanceKindLocal:
				got.item = prgrm.CXProgramNameText(prgrm.CXProgramFunctionSlotLive(int(value.AffordanceValueItem)).Name)
			case ast.AffordanceKindStruct:
				got.item = prgrm.CXProgramNameText(prgrm.CXStructs[value.AffordanceValueItem].Name)
			}
			if value.AffordanceValueVariable >= 0 {
				got.variable = prgrm.CXProgramNameText(actions.StripNameNumber(prgrm, prgrm.GetCXTypeSignatureFromArray(value.AffordanceValueVariable).Name))
			}
			if got != test.want {
				t.Fatalf("value %+v, want %+v", got, test.want)
			}
			if value.AffordanceValueKind >= ast.AffordanceKindPackage && value.AffordanceValuePackage != ast.CXPackageIndex(pkg.Index) {
				t.Fatalf("package %d, want main, %d", value.AffordanceValuePackage, pkg.Index)
			}
		})
	}
}

// A literal with an unknown, repeated or misplaced clause, or a name the
// clause does not find, is a compile error at that clause, and an aff value
// and other types do not mix.
func TestAffordanceLiteralErrors(t *testing.T) {
	for _, test := range []struct {
		body string
		want string
	}{
		{"\ta = #{expr(x)}", "17:8 unknown clause 'expr', expected mem, pkg, fn, var or strct"},
		{"\ta = #{filter(isI32)}", "17:8 unknown clause 'filter', expected mem, pkg, fn, var or strct"},
		{"\ta = #{pkg(main) pkg(main)}", "17:18 clause 'pkg' given twice"},
		{"\ta = #{mem(code)}", "17:8 unknown memory 'code', expected data, stack or heap"},
		{"\ta = #{mem(data) pkg(main)}", "17:18 mem cannot be combined with pkg"},
		{"\ta = #{fn(area) strct(Point)}", "17:17 strct cannot be combined with fn"},
		{"\ta = #{strct(Point) var(g)}", "17:8 strct cannot be combined with var"},
		{"\ta = #{pkg(os)}", "17:8 package 'os' is not imported"},
		{"\ta = #{fn(missing)}", "17:8 function 'missing' is not in package 'main'"},
		{"\ta = #{strct(Missing)}", "17:8 struct 'Missing' is not in package 'main'"},
		{"\ta = #{var(missing)}", "17:8 global 'missing' is not in package 'main'"},
		{"\ta = #{fn(area) var(missing)}", "17:17 variable 'missing' is not in function 'area'"},
		{"\tif true {\n\t\tvar v i32\n\t\tv = 1\n\t}\n\tif true {\n\t\tvar v i32\n\t\tv = 2\n\t}\n\ta = #{fn(main) var(v)}", "variable 'v' is declared in more than one block of function 'main'"},
		{"\ta = #{pkg(main), fn(area)}", "17:17 unexpected comma"},
		{"\ta = #{pkg(\"main\")}", "17:12 unexpected literal, expected name"},
		{"\ta = 5", ""},
		{"\tvar i i32\n\ti = #{}", ""},
		{"\tvar b []ui8\n\tb = serialize(5)", ""},
		{"\tvar b []ui8\n\tb = serialize(#{}, #{})", ""},
		{"\tvar c aff\n\tc = a + a", ""},
	} {
		t.Run(test.body, func(t *testing.T) {
			errors := affordanceCompile(t, affordanceSource(test.body))
			if len(errors) == 0 {
				t.Fatalf("no compile error, want %q", test.want)
			}
			if !strings.Contains(strings.Join(errors, "\n"), test.want) {
				t.Fatalf("compile errors %q, want %q", errors, test.want)
			}
		})
	}
}

// A package, function or struct serializes after a header with the magic
// CXSELECT and the kind, and Deserialize refuses it, since it loads whole
// programs only.
func TestAffordanceSerializeDeclarations(t *testing.T) {
	for _, test := range []struct {
		literal string
		kind    ast.AffordanceKind
	}{
		{"#{pkg(main)}", ast.AffordanceKindPackage},
		{"#{fn(area)}", ast.AffordanceKindFunction},
		{"#{strct(Point)}", ast.AffordanceKindStruct},
	} {
		t.Run(test.literal, func(t *testing.T) {
			if errors := affordanceCompile(t, affordanceSource("\ta = "+test.literal)); len(errors) > 0 {
				t.Fatalf("compile errors %q", errors)
			}
			prgrm := actions.AST
			value := ast.AffordanceValueRead(prgrm.Memory, prgrm.Data.StartsAt+prgrm.Data.Size-types.AFF_SIZE)
			b, ok := ast.SerializeAffordance(prgrm, value)
			if !ok {
				t.Fatal("SerializeAffordance failed")
			}
			if len(b) <= 25 || string(b[:8]) != "CXSELECT" || ast.AffordanceKind(b[24]) != test.kind {
				t.Fatalf("bytes start %q, kind %d, want CXSELECT and kind %d", b[:min(len(b), 8)], b[min(len(b)-1, 24)], test.kind)
			}
			if _, err := ast.Deserialize(b, nil); err == nil {
				t.Fatal("Deserialize loaded a selection")
			}
			value.AffordanceValueItem = 1 << 20
			if test.kind != ast.AffordanceKindPackage {
				if _, ok := ast.SerializeAffordance(prgrm, value); ok {
					t.Fatal("SerializeAffordance took an index outside the program")
				}
			}
		})
	}
}
