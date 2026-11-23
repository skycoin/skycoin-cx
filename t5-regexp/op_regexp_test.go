//go:build !noregexp

package regexp

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	stringpool "github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
	"github.com/skycoin/skycoin-cx/t3-ast"
)

func newTestProgram() *ast.CXProgram {
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a heap literal, site S1418.
	prgrm := &ast.CXProgram{
		Packages:       make(map[stringpool.StringGid]ast.CXPackageIndex),
		CurrentPackage: -1,
		// OBJMODEL: state.CXCallMng (manager storage) or []CXCallGid, phase 4 #366.
		// Now a slice of objects, site S1419.
		CallStack: make([]ast.CXCall, constants.CALLSTACK_SIZE),
		Memory:    make([]byte, constants.STACK_SIZE+ast.MinHeapSize()),
		Stack:     ast.StackSegmentStruct{Size: constants.STACK_SIZE},
		Data:      ast.DataSegmentStruct{StartsAt: constants.STACK_SIZE},
		Heap:      ast.HeapSegmentStruct{StartsAt: constants.STACK_SIZE, Size: ast.MinHeapSize(), Pointer: constants.NULL_HEAP_ADDRESS_OFFSET},
		CxIrState: &ast.CxIrState{},
	}
	prgrm.CxIrState.CxIrStateInit()
	RegisterPackage(prgrm)
	return prgrm
}

func strValue(prgrm *ast.CXProgram, offset types.Pointer, s string) ast.CXValue {
	ast.AllocateStrAt(prgrm, offset, s)
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now a heap literal, site S1420.
	return ast.CXValue{Offset: offset, TypeSignature: &ast.CXTypeSignature{Offset: offset}}
}

// Compile and Find added a CXArgument to the program on every call. See #120.
func TestRegexpCallsDoNotGrowProgram(t *testing.T) {
	prgrm := newTestProgram()
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now a heap literal, site S1421.
	reg := ast.CXValue{Offset: 100, TypeSignature: &ast.CXTypeSignature{Offset: 100}}
	in := strValue(prgrm, 200, "abbbc")
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now a heap literal, site S1422.
	out := ast.CXValue{Offset: 300, TypeSignature: &ast.CXTypeSignature{Offset: 300}}
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now a heap literal, site S1423.
	errOut := ast.CXValue{Offset: 400, TypeSignature: &ast.CXTypeSignature{Offset: 400}}

	before := prgrm.CxIrState.CXArgumentMng.CXArgumentLen()
	for i := 0; i < 100; i++ {
		opRegexpCompile(prgrm, []ast.CXValue{strValue(prgrm, 0, "b+")}, []ast.CXValue{reg, errOut})
		opRegexpFind(prgrm, []ast.CXValue{reg, in}, []ast.CXValue{out})
	}
	if got := out.Get_str(prgrm); got != "bbb" {
		t.Fatalf("Find returned %q, want %q", got, "bbb")
	}
	if after := prgrm.CxIrState.CXArgumentMng.CXArgumentLen(); after != before {
		t.Errorf("CXArgs grew from %d to %d", before, after)
	}
}

// A failed Compile stored nil, and Find on it was a Go nil dereference. See #120.
func TestRegexpFindAfterFailedCompile(t *testing.T) {
	prgrm := newTestProgram()
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now a heap literal, site S1424.
	reg := ast.CXValue{Offset: 500, TypeSignature: &ast.CXTypeSignature{Offset: 500}}
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now a heap literal, site S1425.
	errOut := ast.CXValue{Offset: 600, TypeSignature: &ast.CXTypeSignature{Offset: 600}}
	opRegexpCompile(prgrm, []ast.CXValue{strValue(prgrm, 0, "a(")}, []ast.CXValue{reg, errOut})
	if errOut.Get_str(prgrm) == "" {
		t.Fatal("Compile(\"a(\") returned no error")
	}
	if _, ok := regexps["a("]; ok {
		t.Error("failed Compile stored a Regexp")
	}

	defer func() {
		if r := recover(); r != types.CX_RUNTIME_ERROR {
			t.Errorf("Find on a failed Regexp panicked with %v, want CX_RUNTIME_ERROR", r)
		}
	}()
	in := strValue(prgrm, 200, "abc")
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now a heap literal, site S1426.
	out := ast.CXValue{Offset: 300, TypeSignature: &ast.CXTypeSignature{Offset: 300}}
	opRegexpFind(prgrm, []ast.CXValue{reg, in}, []ast.CXValue{out})
}
