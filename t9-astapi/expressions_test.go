package astapi_test

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"                 // thin: tests use only Code, I16, I32; no depth gain
	cxconstants "github.com/skycoin/skycoin-cx/t1-constants" // thin: tests use only OP_ADD, OP_DIV, OP_SUB; no depth gain
	cxast "github.com/skycoin/skycoin-cx/t3-ast"
	cxinit "github.com/skycoin/skycoin-cx/t7-init"                         // thin: tests use only MakeProgram, RegisterPackages; tests only, moving these tests up lets astapi drop to t5
	parsingcompletor "github.com/skycoin/skycoin-cx/t8-cxparsingcompletor" // thin: tests use only InitCXCore; tests only, could init with t7-init and t4-opcodes
	"github.com/skycoin/skycoin-cx/t9-astapi"
)

func TestASTAPI_Expressions(t *testing.T) {
	var cxprogram *cxast.CXProgram

	// Needed for AddNativeExpressionToFunction
	// because of dependency on cxast.OpNames
	parsingcompletor.InitCXCore()

	t.Run("make program", func(t *testing.T) {
		// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
		// Now a constructor call, site S1598.
		cxprogram = cxinit.MakeProgram()
		cxinit.RegisterPackages(cxprogram)
	})

	t.Run("add empty package", func(t *testing.T) {
		// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
		// Now an arena insert, site S1599.
		err := astapi.AddEmptyPackage(cxprogram, "main")
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add empty function to package", func(t *testing.T) {
		err := astapi.AddEmptyFunctionToPackage(cxprogram, "main", "TestFunction")
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add input to function", func(t *testing.T) {
		err := astapi.AddNativeInputToFunction(cxprogram, "main", "TestFunction", "inputOne", types.I32)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add output to function", func(t *testing.T) {
		err := astapi.AddNativeOutputToFunction(cxprogram, "main", "TestFunction", "outputOne", types.I16)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add first expression to function", func(t *testing.T) {
		err := astapi.AddNativeExpressionToFunction(cxprogram, "TestFunction", cxconstants.OP_ADD)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add second expression to function", func(t *testing.T) {
		err := astapi.AddNativeExpressionToFunction(cxprogram, "TestFunction", cxconstants.OP_SUB)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add an expression to function on a line number", func(t *testing.T) {
		err := astapi.AddNativeExpressionToFunctionByLineNumber(cxprogram, "TestFunction", cxconstants.OP_DIV, 1)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("print program", func(t *testing.T) {
		cxprogram.PrintProgram()
	})

	t.Run("remove expression from a function", func(t *testing.T) {
		// OBJMODEL: state.CXExpressionMng.CXExpressionRelease(gid), phase 4 #346.
		// Now a release, site S1601.
		err := astapi.RemoveExpressionFromFunction(cxprogram, "TestFunction", 2)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("print program", func(t *testing.T) {
		cxprogram.PrintProgram()
	})
}
