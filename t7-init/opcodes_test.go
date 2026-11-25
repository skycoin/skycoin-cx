package init

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t3-ast"
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

// Every live library function is registered under its key with a handler,
// and every linked library key is a live library function.
func TestRegisteredLibraryFunctionsMatchLifecycleTable(t *testing.T) {
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, as in TestOpcodeNamesMatchLifecycleTable.
	prgrm := MakeProgram()
	RegisterPackages(prgrm)

	for _, row := range constants.LibraryFunctionLifecycleTable {
		key := ast.LibraryFunctionKey(row.LibraryNumber, row.Number)
		live := row.VersionDeprecated == 0
		handlers := ast.LibraryFunctionHandlers[row.LibraryNumber]
		handled := row.Number < len(handlers) && handlers[row.Number] != nil
		if live != (nativeCode(row.Name) == key && ast.NativeNames.NativeNameIndexCodeText(key) == row.Name && handled) {
			t.Errorf("library %d function %d %q: live %v, registered %v", row.LibraryNumber, row.Number, row.Name, live, !live)
		}
		native := ast.NativeFunctionOf(prgrm, key)
		if live && (native == nil || native.AtomicOPCode != constants.OP_LIBRARY_CALL || native.LibraryNumber != row.LibraryNumber || native.LibraryFunctionNumber != row.Number) {
			t.Errorf("library %d function %d %q: native %+v", row.LibraryNumber, row.Number, row.Name, native)
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
