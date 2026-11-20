package init

import (
	"github.com/skycoin/skycoin-cx/t1-constants" // thin: only CALLSTACK_SIZE, NULL_HEAP_ADDRESS_OFFSET, STACK_SIZE; no depth gain
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t5-cipher" // thin: only RegisterPackage; registration list, keep
	"github.com/skycoin/skycoin-cx/t5-cxos"   // thin: only RegisterPackage; registration list, keep
	"github.com/skycoin/skycoin-cx/t5-tcp"    // thin: only RegisterPackage; registration list, keep
	"github.com/skycoin/skycoin-cx/t6-cxfx"   // thin: only RegisterPackage; registration list, keep

	// "github.com/skycoin/cx/cx/packages/http"

	"github.com/skycoin/skycoin-cx/t5-regexp" // thin: only RegisterPackage; registration list, keep
)

// RegisterPackages adds the native packages to prgrm and their functions to
// the global opcode tables.
func RegisterPackages(prgrm *ast.CXProgram) {
	cipher.RegisterPackage(prgrm)
	cxfx.RegisterPackage(prgrm)
	cxos.RegisterPackage(prgrm)
	// http.RegisterPackage()
	regexp.RegisterPackage(prgrm)
	tcp.RegisterPackage(prgrm)
}

// MakeProgram returns a new empty program. Call RegisterPackages on it to add
// the native packages.
// VIOLATION: created outside manager; phase 1.
func MakeProgram() *ast.CXProgram {
	minHeapSize := ast.MinHeapSize()
	newPrgrm := &ast.CXProgram{
		Packages:       make(map[string]ast.CXPackageIndex, 0),
		CXFunctions:    make([]ast.CXFunction, 0),
		CurrentPackage: -1,
		CallStack:      make([]ast.CXCall, constants.CALLSTACK_SIZE),
		Memory:         make([]byte, constants.STACK_SIZE+minHeapSize),
		Stack: ast.StackSegmentStruct{
			Size: constants.STACK_SIZE,
		},
		Data: ast.DataSegmentStruct{
			StartsAt: constants.STACK_SIZE,
		},
		Heap: ast.HeapSegmentStruct{
			Size:    minHeapSize,
			Pointer: constants.NULL_HEAP_ADDRESS_OFFSET, // We can start adding objects to the heap after the NULL (nil) bytes.
		},
		CXArgs: make([]ast.CXArgument, 0),
	}

	return newPrgrm
}
