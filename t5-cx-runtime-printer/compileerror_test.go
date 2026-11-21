package printer

import (
	"fmt"
	"testing"

	"github.com/skycoin/skycoin-cx/t3-ast"
)

// TestPrinterCompileErrorAppend checks each code against the text the
// compiler printed before compile errors were records.
func TestPrinterCompileErrorAppend(t *testing.T) {
	state := &ast.CxIrState{}
	state.CxIrStateInit()
	prgrm := &ast.CXProgram{CxIrState: state}
	file := prgrm.CXProgramSourceFileIntern("a.cx")
	text := prgrm.CXProgramCompileErrorText
	header := "error: a.cx:7"

	cases := []struct {
		code     ast.CompileErrorCode
		operands ast.CompileErrorOperands
		want     string
	}{
		{ast.CompileErrorCodeMessage, ast.CompileErrorOperands{text("m n")}, header + " m n\n"},
		{ast.CompileErrorCodeAssignOutputlessOperator, ast.CompileErrorOperands{}, header + " trying to use an outputless operator in an assignment\n"},
		{ast.CompileErrorCodeAssignSymbolType, ast.CompileErrorOperands{text("i32"), text("x"), text("str")}, header + " " + fmt.Sprintf("trying to assign argument of type '%s' to symbol '%s' of type '%s'", "i32", "x", "str") + "\n"},
		{ast.CompileErrorCodeAssignValueType, ast.CompileErrorOperands{text("i32"), text("x"), text("str")}, header + " " + fmt.Sprintf("cannot assign value of type '%s' to identifier '%s' of type '%s'", "i32", "x", "str") + "\n"},
		{ast.CompileErrorCodeBuiltinPackageNotImported, ast.CompileErrorOperands{text("os"), text("os")}, header + " " + fmt.Sprintf("built-in package %s is used without importing it, add import \"%s%s\"", "os", "sys/", "os") + "\n"},
		{ast.CompileErrorCodeFunctionInputType, ast.CompileErrorOperands{text("f"), text("i32"), text("str")}, header + " " + fmt.Sprintf("function '%s' expected input argument of type '%s'; '%s' was provided", "f", "i32", "str") + "\n"},
		{ast.CompileErrorCodeFunctionMissing, ast.CompileErrorOperands{text("i32.foo")}, header + " function '" + "i32.foo" + "' does not exist\n"},
		{ast.CompileErrorCodeFunctionReceiverType, ast.CompileErrorOperands{text("f"), text("i32"), text("str")}, header + " " + fmt.Sprintf("function '%s' expected receiving variable of type '%s'; '%s' was provided", "f", "i32", "str") + "\n"},
		{ast.CompileErrorCodeFunctionRedeclaration, ast.CompileErrorOperands{}, header + " function redeclaration\n"},
		{ast.CompileErrorCodeFunctionReturnCount, ast.CompileErrorOperands{text("f"), 0, 1}, header + " " + fmt.Sprintf("function '%s' expects to return %d argument%s, but %d output argument%s %s provided", "f", 0, "", 1, "", "was") + "\n"},
		{ast.CompileErrorCodeFunctionReturnCount, ast.CompileErrorOperands{text("f"), 1, 2}, header + " " + fmt.Sprintf("function '%s' expects to return %d argument%s, but %d output argument%s %s provided", "f", 1, "", 2, "s", "were") + "\n"},
		{ast.CompileErrorCodeFunctionReturnCount, ast.CompileErrorOperands{text("f"), 2, 0}, header + " " + fmt.Sprintf("function '%s' expects to return %d argument%s, but %d output argument%s %s provided", "f", 2, "s", 0, "s", "were") + "\n"},
		{ast.CompileErrorCodeIdentifierMissing, ast.CompileErrorOperands{text("x")}, header + " " + fmt.Sprintf("identifier '%s' does not exist", "x") + "\n"},
		{ast.CompileErrorCodeIndexType, ast.CompileErrorOperands{text("str")}, header + " " + fmt.Sprintf("wrong index type; expected either 'i32' or 'i64', got '%s'", "str") + "\n"},
		{ast.CompileErrorCodeInputTypesNotEqual, ast.CompileErrorOperands{text("i32.add"), text("i32"), text("f32")}, header + " " + fmt.Sprintf("first and second input arguments' types are not equal in '%s' call ('%s' != '%s')", "i32.add", "i32", "f32") + "\n"},
		{ast.CompileErrorCodeInvalidExpression, ast.CompileErrorOperands{}, header + " invalid expression\n"},
		{ast.CompileErrorCodeInvalidIndexing, ast.CompileErrorOperands{}, header + " invalid indexing\n"},
		{ast.CompileErrorCodeInvalidIndirection, ast.CompileErrorOperands{}, header + " invalid indirection\n"},
		{ast.CompileErrorCodeLoopConditionNoValue, ast.CompileErrorOperands{}, header + " loop condition has no value\n"},
		{ast.CompileErrorCodeNoFields, ast.CompileErrorOperands{text("x")}, header + " " + fmt.Sprintf("'%s' has no fields", "x") + "\n"},
		{ast.CompileErrorCodeNoPackageDefined, ast.CompileErrorOperands{}, header + " No package defined\n"},
		{ast.CompileErrorCodeOpcodeRenamed, ast.CompileErrorOperands{text("str.concat"), text("string.Concatenation"), 2611200002}, header + " " + fmt.Sprintf("%s was renamed to %s in language version %d", "str.concat", "string.Concatenation", 2611200002) + "\n"},
		{ast.CompileErrorCodeOpcodeRenamedImport, ast.CompileErrorOperands{text("str.concat"), text("string.Concatenation"), 2611200002, text("string")}, header + " " + fmt.Sprintf("%s was renamed to %s in language version %d", "str.concat", "string.Concatenation", 2611200002) + fmt.Sprintf(", import \"%s%s\"", "sys/", "string") + "\n"},
		{ast.CompileErrorCodeOperatorArgumentTypes, ast.CompileErrorOperands{text("i32.add")}, header + " " + fmt.Sprintf("invalid argument types for '%s' operator", "i32.add") + "\n"},
		{ast.CompileErrorCodeOperatorInputCount, ast.CompileErrorOperands{text("op"), 2, 1}, header + " " + fmt.Sprintf("operator '%s' expects %d input/s, but %d input argument\n were provided", "op", 2, 1) + "\n"},
		{ast.CompileErrorCodeOperatorOutputCount, ast.CompileErrorOperands{text("op"), 1, 2}, header + " " + fmt.Sprintf("operator '%s' expects to return %d output/s, but %d receiving argument/s were provided", "op", 1, 2) + "\n"},
		{ast.CompileErrorCodePackageIdentifierMissing, ast.CompileErrorOperands{text("os"), text("Foo")}, header + " " + fmt.Sprintf("identifier '%s.%s' does not exist", "os", "Foo") + "\n"},
		{ast.CompileErrorCodePointerOrderingOperator, ast.CompileErrorOperands{text("<")}, header + " " + fmt.Sprintf("pointer %s pointer is not allowed, pointers support only == and !=", "<") + "\n"},
		{ast.CompileErrorCodePrimitiveFieldAccess, ast.CompileErrorOperands{text("x"), text("i32")}, header + " " + fmt.Sprintf("illegal method call or field access on identifier '%s' of primitive type '%s'", "x", "i32") + "\n"},
		{ast.CompileErrorCodeRedeclared, ast.CompileErrorOperands{text("x")}, header + " " + fmt.Sprintf("'%s' redeclared", "x") + "\n"},
		{ast.CompileErrorCodeStrArithmeticOperator, ast.CompileErrorOperands{text("%")}, header + " " + fmt.Sprintf("operator %s is not allowed on str, strings have no arithmetic operators", "%") + "\n"},
		{ast.CompileErrorCodeStrConcatenation, ast.CompileErrorOperands{text("+")}, header + " str + str is not allowed, use string.Concatenation\n"},
		{ast.CompileErrorCodeStrEqualityOperator, ast.CompileErrorOperands{text("==")}, header + " " + fmt.Sprintf("str %s str is not allowed, use string.EqualityByByteValue or string.EqualityByHeapPointer", "==") + "\n"},
		{ast.CompileErrorCodeStrOrderingOperator, ast.CompileErrorOperands{text("<")}, header + " " + fmt.Sprintf("str %s str is not allowed, use string.OrderingByByteValue", "<") + "\n"},
		{ast.CompileErrorCodeStructFieldDuplicate, ast.CompileErrorOperands{}, fmt.Sprintf("%s : duplicate field", header)},
		{ast.CompileErrorCodeStructFieldMultiplyDefined, ast.CompileErrorOperands{text("x")}, header + " Multiply defined struct field: x\n"},
		{ast.CompileErrorCodeStructLiteralFieldType, ast.CompileErrorOperands{text("x"), text("S"), text("i32"), text("str")}, header + " " + fmt.Sprintf("field '%s' in struct literal of type '%s' expected argument of type '%s'; '%s' was provided", "x", "S", "i32", "str") + "\n"},
		{ast.CompileErrorCodeMethodMissing, ast.CompileErrorOperands{text("T"), text("Nope")}, header + " type 'T' has no method 'Nope'\n"},
		{ast.CompileErrorCodeDeclarationTooLarge, ast.CompileErrorOperands{text("a")}, header + " 'a' is too large\n"},
		{ast.CompileErrorCodeStructRecursive, ast.CompileErrorOperands{text("T")}, header + " invalid recursive type 'T'\n"},
	}

	covered := map[ast.CompileErrorCode]bool{}
	for _, tc := range cases {
		covered[tc.code] = true
		record := ast.CompileErrorRecord{CompileErrorRecordCode: tc.code, CompileErrorRecordFile: file, CompileErrorRecordLine: 7, CompileErrorRecordOperands: tc.operands}
		if got := string(PrinterCompileErrorAppend(nil, prgrm, &record)); got != tc.want {
			t.Errorf("code %d: got %q, want %q", tc.code, got, tc.want)
		}
	}
	for code := ast.CompileErrorCodeNil + 1; code < ast.CompileErrorCodeEnd; code++ {
		if !covered[code] {
			t.Errorf("code %d has no case", code)
		}
	}
}
