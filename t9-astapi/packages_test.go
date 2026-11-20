package astapi_test

import (
	"testing"

	cxast "github.com/skycoin/skycoin-cx/t3-ast"
	cxinit "github.com/skycoin/skycoin-cx/t7-init" // thin: tests use only MakeProgram, RegisterPackages; tests only, moving these tests up lets astapi drop to t5
	"github.com/skycoin/skycoin-cx/t9-astapi"
)

func TestASTAPI_Packages(t *testing.T) {
	var cxprogram *cxast.CXProgram
	// The native packages are registered before main.
	var nativePkgs int
	t.Run("make program", func(t *testing.T) {
		cxprogram = cxinit.MakeProgram()
		cxinit.RegisterPackages(cxprogram)
		nativePkgs = len(cxprogram.Packages)
	})

	t.Run("add empty package", func(t *testing.T) {
		err := astapi.AddEmptyPackage(cxprogram, "main")
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}
	})

	t.Run("get packages name lists", func(t *testing.T) {
		list := astapi.GetPackagesNameList(cxprogram)
		wantLen := nativePkgs + 1
		if len(list) != wantLen {
			t.Errorf("got len(list)=%v, want %v", len(list), wantLen)
		}
	})

	t.Run("print program", func(t *testing.T) {
		cxprogram.PrintProgram()
	})
}
