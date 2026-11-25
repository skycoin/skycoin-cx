package init

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
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

// The registered opcodes, their handlers, signatures and operators are the
// rows of constants.OpcodeDefinitionTable, as opcodegen reads them (D21).
func TestRegisteredOpcodesMatchDefinitionTable(t *testing.T) {
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, as in TestRegisteredOpcodesMatchLifecycleTable.
	prgrm := MakeProgram()
	RegisterPackages(prgrm)
	opcodes.RegisterOpcodes(prgrm)

	handlerPackages := map[string]string{"opcodes": "t4-opcodes", "printer": "t5-cx-runtime-printer", "cxoscore": "t5-cxos-core", "regexp": "t5-regexp", "cxfx": "t6-cxfx"}
	registered := map[int]bool{}
	for _, entry := range ast.NativeNames.NativeNameIndexAll() {
		if entry.NativeNameEntryCode >= 0 {
			registered[entry.NativeNameEntryCode] = true
		}
	}
	defined := 0
	for _, row := range constants.OpcodeDefinitionTable {
		native := ast.Natives[row.Number]
		if native == nil {
			// Build tags leave out the cxfx, cxoscore and regexp packages.
			if pkg, _, _ := strings.Cut(row.Handler, "."); pkg != "cxfx" && pkg != "cxoscore" && pkg != "regexp" {
				t.Errorf("opcode %d %q has a definition row but no registration", row.Number, row.Name)
			}
			continue
		}
		defined++
		var handler ast.OpcodeHandler
		if row.Number < len(ast.OpcodeHandlers) {
			handler = ast.OpcodeHandlers[row.Number]
		}
		if handler == nil {
			if row.Handler != "" {
				t.Errorf("opcode %q: no handler registered, definition %q", row.Name, row.Handler)
			}
		} else {
			pkg, function, _ := strings.Cut(row.Handler, ".")
			want := "github.com/skycoin/skycoin-cx/" + handlerPackages[pkg] + "." + function
			if got := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name(); got != want {
				t.Errorf("opcode %q: handler %s, definition %s", row.Name, got, want)
			}
		}
		signature := constants.OpcodeSignatureOf(row.Number)
		if signature.OpcodeSignatureHandled != (handler != nil) {
			t.Errorf("opcode %q: handled %v in the signature table", row.Name, signature.OpcodeSignatureHandled)
		}
		parametersCompare(t, prgrm, row.Name+" inputs", native.Inputs, signature.OpcodeSignatureInputs())
		parametersCompare(t, prgrm, row.Name+" outputs", native.Outputs, signature.OpcodeSignatureOutputs())
		if constants.OpcodeNameOf(row.Number) != row.Name || string(constants.OpcodeNameBytesOf(row.Number)) != row.Name {
			t.Errorf("opcode %d: name %q, want %q", row.Number, constants.OpcodeNameOf(row.Number), row.Name)
		}
	}
	if defined != len(registered) {
		t.Errorf("%d registered opcodes, %d of them in the definition table", len(registered), defined)
	}
	for code := types.BOOL; code < types.COUNT; code++ {
		for operator := constants.START_OF_OPERATORS + 1; operator < constants.END_OF_OPERATORS; operator++ {
			// RegisterOperator put the i32 jumps, which are no operators, in unused slots.
			if _, live := constants.OpcodeLifecycleLiveByNumber(operator); !live {
				continue
			}
			want := 0
			if native := ast.GetTypedOperator(code, operator); native != nil {
				want = native.AtomicOPCode
			}
			if got := constants.OpcodeOperatorOf(code, operator); got != want {
				t.Errorf("operator %d for %s: %d, registered %d", operator, code.Name(), got, want)
			}
		}
	}
	var names []string
	for _, name := range constants.NativeNameTable {
		library, _ := constants.LibraryFunctionKeyParts(name.NativeNameCode)
		if (name.NativeNameCode < 0 && ast.LibraryLinked(library)) || registered[name.NativeNameCode] {
			names = append(names, fmt.Sprintf("%s %d", name.NativeNameText, name.NativeNameCode))
		}
	}
	var want []string
	for _, entry := range ast.NativeNames.NativeNameIndexAll() {
		want = append(want, fmt.Sprintf("%s %d", entry.NativeNameEntryText, entry.NativeNameEntryCode))
	}
	if strings.Join(names, "\n") != strings.Join(want, "\n") {
		t.Errorf("NativeNameTable differs from the registered names")
	}
}

// parametersCompare checks registered native parameters against the
// parameters of a signature.
func parametersCompare(t *testing.T, prgrm *ast.CXProgram, what string, registered []*ast.CXArgument, parameters []constants.OpcodeParameter) {
	t.Helper()
	if len(registered) != len(parameters) {
		t.Errorf("%s: %d registered, %d in the table", what, len(registered), len(parameters))
		return
	}
	for i, parameter := range parameters {
		arg := registered[i]
		slice := len(arg.DeclarationSpecifiers) > 0 && arg.DeclarationSpecifiers[len(arg.DeclarationSpecifiers)-1] == constants.DECL_SLICE
		form := constants.OpcodeParameterFormAtomic
		if slice {
			form = constants.OpcodeParameterFormSlice
		} else if arg.Type == types.STRUCT {
			form = constants.OpcodeParameterFormStruct
		}
		if form != parameter.OpcodeParameterForm || arg.Type != parameter.OpcodeParameterType {
			t.Errorf("%s %d: registered form %d type %s, table form %d type %s", what, i, form, arg.Type.Name(), parameter.OpcodeParameterForm, parameter.OpcodeParameterType.Name())
		}
		if form == constants.OpcodeParameterFormStruct {
			structType := constants.OpcodeParameterStructTable[parameter.OpcodeParameterStruct]
			if prgrm.CXProgramNameText(arg.StructType.Name) != structType.OpcodeParameterStructType || prgrm.CXProgramNameText(arg.Name) != structType.OpcodeParameterStructName {
				t.Errorf("%s %d: registered struct differs from %+v", what, i, structType)
			}
		}
	}
}
