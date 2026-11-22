package ast

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
)

// TestStructAddFieldDuplicate checks that a repeated field is recorded once
// and left out, so the compiler goes on.
func TestStructAddFieldDuplicate(t *testing.T) {
	state := &CxIrState{}
	state.CxIrStateInit()
	prgrm := &CXProgram{CxIrState: state}
	file := prgrm.CXProgramSourceFileIntern("a.cx")
	name := prgrm.CXProgramNameIntern("x")
	strct := MakeStruct(prgrm.CXProgramNameIntern("S"))
	_, first := MakeField(prgrm, name, types.I32, file, 3)
	strct.AddField(prgrm, first)
	_, second := MakeField(prgrm, name, types.I32, file, 4)
	strct.AddField(prgrm, second)
	if len(strct.Fields) != 1 {
		t.Errorf("got %d fields, want 1", len(strct.Fields))
	}
	records := state.CompileErrorList.CompileErrorListRecords
	if len(records) != 1 || records[0].CompileErrorRecordCode != CompileErrorCodeStructFieldDuplicate || records[0].CompileErrorRecordLine != 4 {
		t.Errorf("got records %v, want one CompileErrorCodeStructFieldDuplicate at line 4", records)
	}
}
