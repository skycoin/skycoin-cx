package astapi_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"                 // thin: tests use only Code, I16, I32; no depth gain
	cxconstants "github.com/skycoin/skycoin-cx/t1-constants" // thin: tests use only OP_ADD, OP_DIV, OP_SUB; no depth gain
	cxast "github.com/skycoin/skycoin-cx/t3-ast"
	cxinit "github.com/skycoin/skycoin-cx/t7-init"                         // thin: tests use only MakeProgram, RegisterPackages; tests only, moving these tests up lets astapi drop to t5
	parsingcompletor "github.com/skycoin/skycoin-cx/t8-cxparsingcompletor" // thin: tests use only InitCXCore; tests only, could init with t7-init and t4-opcodes
	"github.com/skycoin/skycoin-cx/t9-astapi"
)

func TestASTAPI_Arguments(t *testing.T) {
	var cxprogram *cxast.CXProgram

	// Needed for AddNativeExpressionToFunction
	// because of dependency on cxast.OpNames
	parsingcompletor.InitCXCore()

	t.Run("make program", func(t *testing.T) {
		// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
		// Now a constructor call, site S1585.
		cxprogram = cxinit.MakeProgram()
		cxinit.RegisterPackages(cxprogram)
	})

	t.Run("add empty package", func(t *testing.T) {
		// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
		// Now an arena insert, site S1586.
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

	t.Run("add expression to function", func(t *testing.T) {
		err := astapi.AddNativeExpressionToFunction(cxprogram, "TestFunction", cxconstants.OP_ADD)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add first input to expression", func(t *testing.T) {
		buf := new(bytes.Buffer)
		var num int32 = 5
		binary.Write(buf, binary.LittleEndian, num)
		err := astapi.AddLiteralInputToExpression(cxprogram, "main", "TestFunction", buf.Bytes(), types.I16, 1)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add second input to expression", func(t *testing.T) {
		err := astapi.AddNativeInputToExpression(cxprogram, "main", "TestFunction", "y", types.I16, 1)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add output to expression", func(t *testing.T) {
		err := astapi.AddNativeOutputToExpression(cxprogram, "main", "TestFunction", "z", types.I16, 1)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("print program", func(t *testing.T) {
		cxprogram.PrintProgram()
	})

	t.Run("make input of an expression a pointer", func(t *testing.T) {
		err := astapi.MakeInputExpressionAPointer(cxprogram, "TestFunction", 1, 1)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("make output of an expression a pointer", func(t *testing.T) {
		err := astapi.MakeOutputExpressionAPointer(cxprogram, "TestFunction", 1, 0)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("print program", func(t *testing.T) {
		cxprogram.PrintProgram()
	})

	// t.Run("remove input from an expression", func(t *testing.T) {
	// err := astapi.RemoveInputFromExpression(cxprogram, "TestFunction", 0)
	// if err != nil {
	// 	t.Errorf("want no error, got %v", err)
	// }
	// })

	// t.Run("remove output from an expression", func(t *testing.T) {
	// 	err := astapi.RemoveOutputFromExpression(cxprogram, "TestFunction", 0)
	// 	if err != nil {
	// 		t.Errorf("want no error, got %v", err)
	// 	}
	// })

	t.Run("print program", func(t *testing.T) {
		cxprogram.PrintProgram()
	})

	t.Run("add global", func(t *testing.T) {
		_, arg := cxast.MakeGlobal(cxprogram, cxprogram.CXProgramNameIntern("testGlobal"), types.I16, cxast.SourceFileGidNil, -1)

		// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
		// Now an arena getter, site S1589.
		cxpackage, err := cxprogram.GetPackageFromArray(cxprogram.Packages[cxprogram.CxIrState.CxIrStateNameMain])
		if err != nil {
			panic(err)
		}

		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now an arena insert, site S1590.
		argIdx := cxprogram.AddCXArgInArray(arg)
		cxpackage.AddGlobal(cxprogram, argIdx)
	})

	t.Run("get accessible i16 args", func(t *testing.T) {
		args, err := astapi.GetAccessibleArgsForFunctionByType(cxprogram, "main", "TestFunction", types.I16)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}

		got := map[string]bool{}
		for _, arg := range args {
			t.Logf("Arg (name,type)=(%v,%v)\n", cxprogram.CXProgramNameText(arg.Name), types.Code(arg.Meta))
			got[cxprogram.CXProgramNameText(arg.Name)] = true
		}
		for _, want := range []string{"testGlobal", "y"} {
			if !got[want] {
				t.Errorf("want accessible i16 arg %s, got %v", want, got)
			}
		}
	})
}
