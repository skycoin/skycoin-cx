package actions

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
)

func TestConstantsHaveTheirOwnCodes(t *testing.T) {
	InitConstants()
	names := map[int]string{}
	for name, code := range ConstCodes {
		if other, ok := names[code]; ok {
			t.Errorf("%s and %s share code %d", name, other, code)
		}
		names[code] = name
	}
	want := map[string]int32{
		"cx.SUCCESS":                  types.CX_SUCCESS,
		"cx.ASSERT":                   types.CX_ASSERT,
		"cx.RUNTIME_INVALID_ARGUMENT": types.CX_RUNTIME_INVALID_ARGUMENT,
		"cx.RUNTIME_NOT_IMPLEMENTED":  types.CX_RUNTIME_NOT_IMPLEMENTED,
		"cx.POINTER_SIZE":             types.Cast_ptr_to_i32(types.POINTER_SIZE),
	}
	for name, value := range want {
		code, ok := ConstCodes[name]
		if !ok {
			t.Errorf("%s is not registered", name)
			continue
		}
		if got := types.Read_i32(Constants[code].Value, 0); got != value {
			t.Errorf("%s = %d, want %d", name, got, value)
		}
	}
}
