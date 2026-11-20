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
	prgrm := MakeProgram()
	RegisterPackages(prgrm)
	opcodes.RegisterOpcodes(prgrm)
	RegisterPackages(prgrm)

	for code, name := range ast.OpNames {
		row, found := constants.OpcodeLifecycleLiveByNumber(code)
		if !found || row.Name != name || ast.OpCodes[name] != code {
			t.Errorf("opcode %d %q is not its live table row", code, name)
		}
	}
	for _, row := range constants.OpcodeLifecycleTable {
		if row.VersionDeprecated == 0 {
			continue
		}
		if _, found := ast.OpNames[row.Number]; found {
			t.Errorf("deprecated opcode %d %q is registered", row.Number, row.Name)
		}
		if row.Number < len(ast.OpcodeHandlers) && ast.OpcodeHandlers[row.Number] != nil {
			t.Errorf("deprecated opcode %d %q has a handler", row.Number, row.Name)
		}
		if _, found := ast.OpCodes[row.Name]; found {
			t.Errorf("deprecated opcode name %q is registered", row.Name)
		}
	}
	if got := ast.OpCodes["str.equal_by_value"]; got != 586 {
		t.Errorf("str.equal_by_value is %d, want 586", got)
	}
	if got := ast.OpCodes["ptr.uneq"]; got != 590 {
		t.Errorf("ptr.uneq is %d, want 590", got)
	}
}
