package init

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t3-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
)

// Every opcode name sits on its live row of OpcodeLifecycleTable, and no
// deprecated number has a handler or a name.
func TestOpcodeNamesMatchLifecycleTable(t *testing.T) {
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, site S1465.
	prgrm := MakeProgram()
	RegisterPackages(prgrm)

	for _, entry := range ast.NativeNames.NativeNameIndexAll() {
		code, name := entry.NativeNameEntryCode, entry.NativeNameEntryText
		if code < 0 || !ast.NativeLinked(code) {
			continue
		}
		row, found := constants.OpcodeLifecycleLiveByNumber(code)
		if !found || row.Name != name || nativeCode(name) != code {
			t.Errorf("opcode %d %q is not its live table row", code, name)
		}
	}
	for _, row := range constants.OpcodeLifecycleTable {
		if row.VersionDeprecated == 0 {
			continue
		}
		if constants.OpcodeNameOf(row.Number) != "" || constants.OpcodeHandled(row.Number) {
			t.Errorf("deprecated opcode %d %q has a name or a handler", row.Number, row.Name)
		}
		// A name may come back only on a live opcode added after the
		// deprecation, as os.Open did when the file library became opcodes
		// (D119).
		if live, found := constants.OpcodeLifecycleLiveByName(row.Name); found && live.VersionAdded > row.VersionDeprecated {
			continue
		}
		if entry, found := ast.NativeNames.NativeNameIndexText(row.Name); found && entry.NativeNameEntryCode >= 0 {
			t.Errorf("deprecated opcode name %q is opcode %d", row.Name, entry.NativeNameEntryCode)
		}
	}
	if got := nativeCode("string.EqualityByByteValue"); got != 586 {
		t.Errorf("string.EqualityByByteValue is %d, want 586", got)
	}
	if got := nativeCode("ptr.uneq"); got != 590 {
		t.Errorf("ptr.uneq is %d, want 590", got)
	}
	if got := nativeCode("LibraryCall"); got != 0 {
		t.Errorf("LibraryCall has the name code %d, want none", got)
	}
}

// Every live function of a linked library is named under its key and has a
// native made from its signature, and every linked library key is a live
// library function. The library switches report false for a number without
// a definition row.
func TestLibraryFunctionsMatchLifecycleTable(t *testing.T) {
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, as in TestOpcodeNamesMatchLifecycleTable.
	prgrm := MakeProgram()
	RegisterPackages(prgrm)

	for _, row := range constants.LibraryFunctionLifecycleTable {
		key := ast.LibraryFunctionKey(row.LibraryNumber, row.Number)
		live := row.VersionDeprecated == 0 && ast.LibraryLinked(row.LibraryNumber)
		named := nativeCode(row.Name) == key && ast.NativeNames.NativeNameIndexCodeText(key) == row.Name && ast.LibraryFunctionLinked(row.LibraryNumber, row.Number)
		if live != named {
			t.Errorf("library %d function %d %q: live and linked %v, named %v", row.LibraryNumber, row.Number, row.Name, live, named)
		}
		native := ast.NativeFunctionOf(prgrm, key)
		signature := constants.LibraryFunctionSignatureOf(row.LibraryNumber, row.Number)
		if live && (native == nil || native.AtomicOPCode != constants.OP_LIBRARY_CALL || native.LibraryNumber != row.LibraryNumber || native.LibraryFunctionNumber != row.Number || len(native.Inputs) != len(signature.LibraryFunctionSignatureInputs()) || len(native.Outputs) != len(signature.LibraryFunctionSignatureOutputs())) {
			t.Errorf("library %d function %d %q: native %+v", row.LibraryNumber, row.Number, row.Name, native)
		}
		if !live && native != nil {
			t.Errorf("library %d function %d %q: native %+v without a live linked row", row.LibraryNumber, row.Number, row.Name, native)
		}
		if row.VersionDeprecated != 0 && ast.LibraryLinked(row.LibraryNumber) && (ast.LibraryDispatches[row.LibraryNumber](prgrm, row.Number, nil, nil) || vm.LibraryDispatchesVm[row.LibraryNumber](uint32(row.Number), nil, 0)) {
			t.Errorf("library %d function %d %q: deprecated, ran in the switch", row.LibraryNumber, row.Number, row.Name)
		}
	}
	for library := 1; library < constants.LibraryCount; library++ {
		if !ast.LibraryLinked(library) {
			continue
		}
		for _, function := range []int{0, constants.LibraryFunctionTableSize} {
			if ast.LibraryDispatches[library](prgrm, function, nil, nil) || vm.LibraryDispatchesVm[library](uint32(function), nil, 0) {
				t.Errorf("library %d function %d ran in the switch", library, function)
			}
		}
	}
	for _, entry := range ast.NativeNames.NativeNameIndexAll() {
		code, name := entry.NativeNameEntryCode, entry.NativeNameEntryText
		if code >= 0 || !ast.NativeLinked(code) {
			continue
		}
		library, function := constants.LibraryFunctionKeyParts(code)
		if row, found := constants.LibraryFunctionLiveByNumber(library, function); !found || row.Name != name {
			t.Errorf("key %d %q is not a live library function", code, name)
		}
	}
}

// NativeFunctionOf makes the natives of OpcodeDefinitionTable from their
// signatures, and the switch reports false for an opcode without a handler.
func TestOpcodeNativesAndDispatch(t *testing.T) {
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, as in TestOpcodeNamesMatchLifecycleTable.
	prgrm := MakeProgram()
	RegisterPackages(prgrm)

	for _, row := range constants.OpcodeDefinitionTable {
		if !constants.OpcodeLinked(row.Number) {
			continue
		}
		native := ast.NativeFunctionOf(prgrm, row.Number)
		signature := constants.OpcodeSignatureOf(row.Number)
		if native == nil || native.AtomicOPCode != row.Number || len(native.Inputs) != len(signature.OpcodeSignatureInputs()) || len(native.Outputs) != len(signature.OpcodeSignatureOutputs()) {
			t.Errorf("opcode %d %q: native %+v", row.Number, row.Name, native)
		}
	}
	compile := ast.NativeFunctionOf(prgrm, nativeCode("regexp.Compile"))
	if constants.OpcodeLinked(nativeCode("regexp.Compile")) && (compile.Outputs[0].Type != types.STRUCT || prgrm.CXProgramNameText(compile.Outputs[0].Name) != "r") {
		t.Errorf("regexp.Compile returns %+v", compile.Outputs[0])
	}
	for _, code := range []int{0, constants.OP_JMP_EQ, constants.OP_ADD, constants.OP_LIBRARY_CALL, 4096} {
		if ast.OpcodeDispatch(prgrm, code, nil, nil) {
			t.Errorf("opcode %d ran in the switch", code)
		}
	}
	if constants.OpcodeOperatorOf(types.I32, constants.OP_ADD) != nativeCode("i32.add") {
		t.Errorf("i32 add is not i32.add")
	}
}

// nativeCode returns the code of the linked built-in name, or 0.
func nativeCode(name string) int {
	entry, _ := ast.NativeNames.NativeNameIndexText(name)
	return entry.NativeNameEntryCode
}
