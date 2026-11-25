package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/skycoin/skycoin-cx/t2-globals"
	"github.com/skycoin/skycoin-cx/t4-actions"
	cxinit "github.com/skycoin/skycoin-cx/t7-init"
	cxparsing "github.com/skycoin/skycoin-cx/t9-cxparsing"
)

// compileBenchmarkPrograms are the CX programs BenchmarkCompile parses,
// chosen to cover small examples and the larger language tests.
var compileBenchmarkPrograms = []string{
	"examples/hello-world.cx",
	"examples/bubble-sort.cx",
	"examples/fibonacci-recursive.cx",
	"test-array.cx",
	"test-slices.cx",
	"test-gc.cx",
	"examples/testing-pointers-to-arrays-of-structs.cx",
}

// compileBenchmarkObjectCount is the number of IR objects in the arenas and
// managers of actions.AST, reported per compile so object model changes show
// up.
func compileBenchmarkObjectCount() int {
	program := actions.AST
	return len(program.CXPackages) + program.CxIrState.CXFunctionMng.CXFunctionLen() + len(program.CXStructs) +
		len(program.CXAtomicOps) + program.CxIrState.CXLineMng.CXLineLen() + program.CxIrState.CXArgumentMng.CXArgumentLen() +
		program.CxIrState.CXTypeSignatureMng.CXTypeSignatureLen() + program.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayLen() +
		program.CxIrState.CXTypeSignature_StructMng.CXTypeSignature_StructLen()
}

// BenchmarkCompile measures the compiler alone, parse passes one and two
// plus the *init function, with no run. Opcodes are registered once, as cx
// does, and the fresh program each compile starts from is not timed.
// It is the baseline for the object model migration, see docs/ObjectModel.md.
func BenchmarkCompile(b *testing.B) {
	for _, programPath := range compileBenchmarkPrograms {
		source, err := os.ReadFile(filepath.Join("..", "tests", programPath))
		if err != nil {
			b.Fatal(err)
		}
		b.Run(filepath.Base(programPath), func(b *testing.B) {
			initProgram()
			b.ReportAllocs()
			b.ResetTimer()
			objectCount := 0
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
				// Also state.CXProgramMng.CXProgramRelease(gid), phase 4 #371. Site S0054 S0055.
				actions.AST = cxinit.MakeProgram()
				cxinit.RegisterPackages(actions.AST)
				globals.FoundCompileErrors = false
				globals.HeapOffset = 0
				b.StartTimer()
				if err := cxparsing.ParseSourceStrings([]string{string(source)}, []string{programPath}); err != nil {
					b.Fatalf("%s: %v", programPath, err)
				}
				if err := cxparsing.AddInitFunction(actions.AST); err != nil {
					b.Fatalf("%s: %v", programPath, err)
				}
				b.StopTimer()
				objectCount = compileBenchmarkObjectCount()
				b.StartTimer()
			}
			b.ReportMetric(float64(objectCount), "irobjects/op")
		})
	}
}
