package ast

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t1-constants"
)

// TestLibraryCallSandbox checks the D16 sandbox of the old loop. A library
// is disabled by default (D17), and an enabled call runs its function through
// the switch of its library. The disabled calls, which panic, are the
// test-library-disabled.cx cases of t1-cxtest.
func TestLibraryCallSandbox(t *testing.T) {
	if constants.LibraryPolicyEnabledWhenImported {
		t.Fatalf("libraries are enabled by default, decision D17 keeps them disabled")
	}
	prgrm := &CXProgram{}
	for library, enabled := range prgrm.ProgramLibrariesEnabled {
		if enabled {
			t.Errorf("library %d is enabled in a new program", library)
		}
	}
	saved := LibraryDispatches[constants.LibraryNumberFile]
	defer func() { LibraryDispatches[constants.LibraryNumberFile] = saved }()
	var ran []int
	LibraryDispatches[constants.LibraryNumberFile] = func(prgrm *CXProgram, function int, inputs []CXValue, outputs []CXValue) bool {
		ran = append(ran, function)
		return true
	}
	prgrm.ProgramLibrariesEnabled[constants.LibraryNumberFile] = true
	libraryCall(prgrm, &CXFunction{AtomicOPCode: constants.OP_LIBRARY_CALL, LibraryNumber: constants.LibraryNumberFile, LibraryFunctionNumber: 3}, nil, nil)
	if len(ran) != 1 || ran[0] != 3 {
		t.Errorf("an enabled call ran functions %v, want [3]", ran)
	}
}
