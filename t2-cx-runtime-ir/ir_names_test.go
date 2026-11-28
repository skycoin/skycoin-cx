package ir

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestOpcodeNameTable fails when OpcodeNameTable is not the Opcode constants
// of ir.go in order.
func TestOpcodeNameTable(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "ir.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.CONST {
			continue
		}
		for _, spec := range general.Specs {
			for _, name := range spec.(*ast.ValueSpec).Names {
				if name.Name == "OpcodeInvalid" {
					names = names[:0]
				}
				names = append(names, name.Name)
			}
		}
		if len(names) > 0 && names[0] == "OpcodeInvalid" {
			break
		}
	}
	if len(names) != int(OpcodeCount)+1 {
		t.Fatalf("ir.go has %d opcode constants, OpcodeCount is %d", len(names)-1, OpcodeCount)
	}
	for op, name := range OpcodeNameTable {
		if want := strings.TrimPrefix(names[op], "Opcode"); name != want {
			t.Errorf("OpcodeNameTable[%d] is %q, ir.go names it %q", op, name, want)
		}
	}
}

// TestImageFileMagicBytes keeps the bytes the run path writes equal to
// ImageFileMagic.
func TestImageFileMagicBytes(t *testing.T) {
	if string(imageFileMagicBytes[:]) != ImageFileMagic {
		t.Errorf("imageFileMagicBytes is %q, ImageFileMagic %q", imageFileMagicBytes[:], ImageFileMagic)
	}
}
