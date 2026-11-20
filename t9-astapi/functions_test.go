package astapi_test

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types" // thin: tests use only Code, I16, I32; no depth gain
	cxast "github.com/skycoin/skycoin-cx/t3-ast"
	cxinit "github.com/skycoin/skycoin-cx/t7-init" // thin: tests use only MakeProgram, RegisterPackages; tests only, moving these tests up lets astapi drop to t5
	"github.com/skycoin/skycoin-cx/t9-astapi"
)

func TestASTAPI_Functions(t *testing.T) {
	var cxprogram *cxast.CXProgram
	t.Run("make program", func(t *testing.T) {
		// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
		// Now a constructor call, site S1606.
		cxprogram = cxinit.MakeProgram()
		cxinit.RegisterPackages(cxprogram)
	})

	t.Run("add empty package", func(t *testing.T) {
		// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
		// Now an arena insert, site S1607.
		err := astapi.AddEmptyPackage(cxprogram, "main")
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add empty function to package", func(t *testing.T) {
		// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
		// Now an arena insert, site S1608.
		err := astapi.AddEmptyFunctionToPackage(cxprogram, "main", "TestFunction")
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add first input to function", func(t *testing.T) {
		err := astapi.AddNativeInputToFunction(cxprogram, "main", "TestFunction", "inputOne", types.I32)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add second input to function", func(t *testing.T) {
		err := astapi.AddNativeInputToFunction(cxprogram, "main", "TestFunction", "inputTwo", types.I32)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add first output to function", func(t *testing.T) {
		err := astapi.AddNativeOutputToFunction(cxprogram, "main", "TestFunction", "outputOne", types.I16)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("add second output to function", func(t *testing.T) {
		err := astapi.AddNativeOutputToFunction(cxprogram, "main", "TestFunction", "outputTwo", types.I16)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("print program", func(t *testing.T) {
		cxprogram.PrintProgram()
	})

	// t.Run("remove first input of function", func(t *testing.T) {
	// 	err := astapi.RemoveFunctionInput(cxprogram, "TestFunction", "inputOne")
	// 	if err != nil {
	// 		t.Errorf("want no error, got %v", err)
	// 	}
	// })

	// t.Run("remove first output of function", func(t *testing.T) {
	// 	err := astapi.RemoveFunctionOutput(cxprogram, "TestFunction", "outputOne")
	// 	if err != nil {
	// 		t.Errorf("want no error, got %v", err)
	// 	}
	// })

	t.Run("print program", func(t *testing.T) {
		cxprogram.PrintProgram()
	})

	t.Run("remove function from package", func(t *testing.T) {
		// OBJMODEL: state.CXFunctionMng.CXFunctionRelease(gid), phase 4 #351.
		// Now a release, site S1609.
		err := astapi.RemoveFunctionFromPackage(cxprogram, "main", "TestFunction")
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("print program", func(t *testing.T) {
		cxprogram.PrintProgram()
	})
}
