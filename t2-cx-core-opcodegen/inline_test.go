package main

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t1-constants"
)

func TestInlineCase(t *testing.T) {
	for _, test := range []struct {
		row  constants.OpcodeDefinition
		call string
		want string
	}{
		{constants.OpcodeDefinition{Name: "gl.Clear", Inputs: "i32"}, "gl.Clear(uint32(%0))",
			"gl.Clear(uint32(types.Read_i32(prgrm.Memory, inputs[0].Offset)))"},
		{constants.OpcodeDefinition{Name: "gl.End"}, "gl.End()", "gl.End()"},
		{constants.OpcodeDefinition{Name: "glfw.ShouldClose", Inputs: "i32", Outputs: "bool"}, "cxfxWindowGet(%0).ShouldClose()",
			"types.Write_bool(prgrm.Memory, outputs[0].Offset, cxfxWindowGet(types.Read_i32(prgrm.Memory, inputs[0].Offset)).ShouldClose())"},
		{constants.OpcodeDefinition{Name: "gl.Uniform1f", Inputs: "i32 f32"}, "gl.Uniform1f(%0, %1)",
			"gl.Uniform1f(types.Read_i32(prgrm.Memory, inputs[0].Offset), types.Read_f32(prgrm.Memory, inputs[1].Offset))"},
	} {
		got, err := inlineCase(test.row, test.call)
		if err != nil || got != test.want {
			t.Errorf("%s: got %q, %v", test.row.Name, got, err)
		}
	}
	for _, test := range []struct {
		row  constants.OpcodeDefinition
		call string
	}{
		{constants.OpcodeDefinition{Name: "past the inputs", Inputs: "i32"}, "f(%1)"},
		{constants.OpcodeDefinition{Name: "a str input", Inputs: "str"}, "f(%0)"},
		{constants.OpcodeDefinition{Name: "two outputs", Outputs: "i32 i32"}, "f()"},
		{constants.OpcodeDefinition{Name: "a slice output", Outputs: "[]i32"}, "f()"},
	} {
		if _, err := inlineCase(test.row, test.call); err == nil {
			t.Errorf("%s: no error", test.row.Name)
		}
	}
	used := map[string]bool{}
	inlinePackages("int32(cxfxWindowGet(%0).GetKey(glfw.Key(%1)))", used)
	if !used["glfw"] || used["gl"] || len(used) != 1 {
		t.Errorf("packages %v", used)
	}
}
