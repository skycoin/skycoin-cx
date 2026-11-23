package init

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-opcodes"
)

// Every registered opcode sits on its live row of OpcodeLifecycleTable, and
// no deprecated number has a handler or a name.
func TestRegisteredOpcodesMatchLifecycleTable(t *testing.T) {
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, site S1465.
	prgrm := MakeProgram()
	RegisterPackages(prgrm)
	opcodes.RegisterOpcodes(prgrm)
	RegisterPackages(prgrm)

	for _, entry := range ast.NativeNames.NativeNameIndexAll() {
		code, name := entry.NativeNameEntryCode, entry.NativeNameEntryText
		if code < 0 {
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
		if ast.NativeNames.NativeNameIndexCodeText(row.Number) != "" {
			t.Errorf("deprecated opcode %d %q is registered", row.Number, row.Name)
		}
		if row.Number < len(ast.OpcodeHandlers) && ast.OpcodeHandlers[row.Number] != nil {
			t.Errorf("deprecated opcode %d %q has a handler", row.Number, row.Name)
		}
		if entry, found := ast.NativeNames.NativeNameIndexText(row.Name); found && entry.NativeNameEntryCode >= 0 {
			code := entry.NativeNameEntryCode
			t.Errorf("deprecated opcode name %q is registered as opcode %d", row.Name, code)
		}
	}
	if got := nativeCode("string.EqualityByByteValue"); got != 586 {
		t.Errorf("string.EqualityByByteValue is %d, want 586", got)
	}
	if got := nativeCode("ptr.uneq"); got != 590 {
		t.Errorf("ptr.uneq is %d, want 590", got)
	}
}

// Every live library function is registered under its key with a handler,
// and every registered library key is a live library function.
func TestRegisteredLibraryFunctionsMatchLifecycleTable(t *testing.T) {
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, as in TestRegisteredOpcodesMatchLifecycleTable.
	prgrm := MakeProgram()
	RegisterPackages(prgrm)
	opcodes.RegisterOpcodes(prgrm)

	for _, row := range constants.LibraryFunctionLifecycleTable {
		key := ast.LibraryFunctionKey(row.LibraryNumber, row.Number)
		live := row.VersionDeprecated == 0
		handlers := ast.LibraryFunctionHandlers[row.LibraryNumber]
		handled := row.Number < len(handlers) && handlers[row.Number] != nil
		if live != (nativeCode(row.Name) == key && ast.NativeNames.NativeNameIndexCodeText(key) == row.Name && handled) {
			t.Errorf("library %d function %d %q: live %v, registered %v", row.LibraryNumber, row.Number, row.Name, live, !live)
		}
		native := ast.Natives[key]
		if live && (native == nil || native.AtomicOPCode != constants.OP_LIBRARY_CALL || native.LibraryNumber != row.LibraryNumber || native.LibraryFunctionNumber != row.Number) {
			t.Errorf("library %d function %d %q: native %+v", row.LibraryNumber, row.Number, row.Name, native)
		}
	}
	for _, entry := range ast.NativeNames.NativeNameIndexAll() {
		code, name := entry.NativeNameEntryCode, entry.NativeNameEntryText
		if code >= 0 {
			continue
		}
		library, function := -code>>16, -code&0xffff
		if row, found := constants.LibraryFunctionLiveByNumber(library, function); !found || row.Name != name {
			t.Errorf("key %d %q is not a live library function", code, name)
		}
	}
}

// nativeCode returns the code registered for the built-in name, or 0.
func nativeCode(name string) int {
	entry, _ := ast.NativeNames.NativeNameIndexText(name)
	return entry.NativeNameEntryCode
}
