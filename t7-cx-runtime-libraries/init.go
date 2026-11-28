package libraries

import (
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable" // thin: only CallStackSize, NullHeapAddressOffset, StackSize; no depth gain
	stringpool "github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
	"github.com/skycoin/skycoin-cx/t4-cx-library-cipher"         // thin: only PackageRegister, CipherOpcodeDispatch; registration list and opcode switch, keep
	"github.com/skycoin/skycoin-cx/t4-cx-library-netopcodes"     // thin: only PackageRegister, NetOpcodeDispatch; registration list and opcode switch, keep
	"github.com/skycoin/skycoin-cx/t4-cx-library-reflectopcodes" // thin: only PackageRegister, ReflectOpcodeDispatch; registration list and opcode switch, keep
	"github.com/skycoin/skycoin-cx/t4-cx-runtime-printer"        // thin: only CompileErrorsRegister; registration list, keep
	"github.com/skycoin/skycoin-cx/t5-cx-library-fileopcodes"    // thin: only PackageRegister, FileOpcodeDispatch; registration list and opcode switch, keep
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"      // thin: only CoreOpcodeDispatch; the opcode switch, keep
	"github.com/skycoin/skycoin-cx/t6-cx-library-graphics"       // thin: only CxfxOpcodeDispatch; the opcode switch, keep

	// "github.com/skycoin/cx/cx/packages/http"

	"github.com/skycoin/skycoin-cx/t4-cx-library-regexp" // thin: only PackageRegister; registration list, keep
)

// PackagesRegister adds the native packages to prgrm, registers the system
// libraries and sets the opcode switch of cx.
func PackagesRegister(prgrm *ast.CXProgram) {
	ast.OpcodeDispatch = OpcodeDispatch
	cipher.PackageRegister(prgrm)
	fileopcodes.PackageRegister(prgrm)
	printer.CompileErrorsRegister(prgrm)
	// http.PackageRegister()
	regexp.PackageRegister(prgrm)
	netopcodes.PackageRegister(prgrm)
	reflectopcodes.PackageRegister(prgrm)
}

// ProgramMake returns a new empty program. Call PackagesRegister on it to add
// the native packages.
// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
// Now a constructor, site S1460.
func ProgramMake() *ast.CXProgram {
	minHeapSize := ast.MinHeapSize()
	state := &ast.CxIrState{}
	state.CxIrStateInit()
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a heap literal, site S1461.
	newPrgrm := &ast.CXProgram{
		Packages:       make(map[stringpool.StringGid]ast.CXPackageIndex, 0),
		CurrentPackage: -1,
		// OBJMODEL: state.CXCallMng (manager storage) or []CXCallGid, phase 4 #366.
		// Now a slice of objects, site S1463.
		CallStack: make([]ast.CXCall, opcodetable.CallStackSize),
		Memory:    make([]byte, opcodetable.StackSize+minHeapSize),
		Stack: ast.StackSegmentStruct{
			Size: opcodetable.StackSize,
		},
		Data: ast.DataSegmentStruct{
			StartsAt: opcodetable.StackSize,
		},
		Heap: ast.HeapSegmentStruct{
			Size:    minHeapSize,
			Max:     opcodetable.MaxHeapSize,
			Pointer: opcodetable.NullHeapAddressOffset, // We can start adding objects to the heap after the NULL (nil) bytes.
		},
		Version:   opcodetable.LanguageVersionCurrent,
		CxIrState: state,
	}

	return newPrgrm
}

// OpcodeDispatch is the opcode switch of cx: the core switch, then the
// switches of the cipher, net, file and reflect libraries and of cxfx, each of which reports false in
// a build whose tags leave it out.
func OpcodeDispatch(prgrm *ast.CXProgram, code int, inputs []ast.CXValue, outputs []ast.CXValue) bool {
	return core.CoreOpcodeDispatch(prgrm, code, inputs, outputs) || cipher.CipherOpcodeDispatch(prgrm, code, inputs, outputs) || netopcodes.NetOpcodeDispatch(prgrm, code, inputs, outputs) || fileopcodes.FileOpcodeDispatch(prgrm, code, inputs, outputs) || graphics.CxfxOpcodeDispatch(prgrm, code, inputs, outputs) || reflectopcodes.ReflectOpcodeDispatch(prgrm, code, inputs, outputs)
}
