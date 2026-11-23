package main

import (
	"fmt"
	"slices"
	"testing"

	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	cxparsing "github.com/skycoin/skycoin-cx/t9-cxparsing"
)

// A global that takes its type from a value, var inferred = f(), has the
// global of var declared T, where f returns T.
func TestGlobalInferredMatchesDeclared(t *testing.T) {
	for _, typ := range []string{"i32", "f64", "bool", "str", "*i32", "[3]i32", "[2][3]f32", "[]i32", "[][]i32", "*[2]i32", "Point", "*Point", "[]Point", "[2]Point", "[]str"} {
		t.Run(typ, func(t *testing.T) {
			initProgram()
			src := fmt.Sprintf("package main\ntype Point struct {\n\tx i32\n\ty f64\n}\nfunc value() (r %s) {\n\treturn r\n}\nvar inferred = value()\nvar declared %s\nfunc main() {\n}\n", typ, typ)
			if err := cxparsing.ParseSourceStrings([]string{src}, []string{"infer.cx"}); err != nil {
				t.Fatal(err)
			}
			pkg, err := actions.AST.GetPackage(actions.AST.CxIrState.CxIrStateNameMain)
			if err != nil {
				t.Fatal(err)
			}
			inferred, err := pkg.GetGlobal(actions.AST, actions.AST.CXProgramNameIntern("inferred"))
			if err != nil {
				t.Fatal(err)
			}
			declared, err := pkg.GetGlobal(actions.AST, actions.AST.CXProgramNameIntern("declared"))
			if err != nil {
				t.Fatal(err)
			}
			if got, want := globalTypeDescription(actions.AST, inferred), globalTypeDescription(actions.AST, declared); got != want {
				t.Errorf("inferred global\n%s\nwant the declared global\n%s", got, want)
			}
		})
	}
}

// globalTypeDescription prints what the type of the global glbl is made of.
func globalTypeDescription(prgrm *ast.CXProgram, glbl *ast.CXTypeSignature) string {
	text := fmt.Sprintf("signature type %d pass by %d deref %v type %s size %d", glbl.Type, glbl.PassBy, glbl.IsDeref, ast.GetFormattedType(prgrm, glbl), glbl.GetSize(prgrm, false))
	switch glbl.Type {
	case ast.TYPE_ATOMIC, ast.TYPE_POINTER_ATOMIC:
		text += fmt.Sprintf(" meta %d", glbl.Meta)
	case ast.TYPE_ARRAY_ATOMIC, ast.TYPE_POINTER_ARRAY_ATOMIC, ast.TYPE_SLICE_ATOMIC, ast.TYPE_POINTER_SLICE_ATOMIC:
		array := prgrm.GetCXTypeSignatureArrayFromArray(glbl.Meta)
		text += fmt.Sprintf(" element %d lengths %v indexes %d", array.Meta, array.Lengths, len(array.Indexes))
	case ast.TYPE_CXARGUMENT_DEPRECATE:
		arg := prgrm.GetCXArg(ast.CXArgumentIndex(glbl.Meta))
		structName := ""
		if arg.StructType != nil {
			structName = prgrm.CXProgramNameText(arg.StructType.Name)
		}
		text += fmt.Sprintf(" argument type %d target %d struct %q size %d lengths %v specifiers %v pass by %d", arg.Type, arg.PointerTargetType, structName, arg.Size, arg.Lengths, slices.Clone(arg.DeclarationSpecifiers), arg.PassBy)
	}
	return text
}
