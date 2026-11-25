package ast

import (
	"reflect"
	"testing"
)

// TestSerializeFieldCounts fails when a type the serializer writes field by
// field gains or loses a field, so that the new field is not left out of the
// serialized form, as CXFunction.ListOfSlots and EscapeInputs once were.
// Update serialize.go, serializeFormatVersion and docs/IR.md, then the count.
func TestSerializeFieldCounts(t *testing.T) {
	counts := []struct {
		value  interface{}
		fields int
	}{
		{CXProgram{}, 32},
		{StackSegmentStruct{}, 3},
		{DataSegmentStruct{}, 2},
		{HeapSegmentStruct{}, 9},
		{CxIrState{}, 25},
		{CXPackage{}, 8},
		{CXPackageMethodKey{}, 2},
		{CXStruct{}, 7},
		{CXFunction{}, 18},
		{CXFunctionSlot{}, 5},
		{CXExpression{}, 5},
		{CXAtomicOperator{}, 8},
		{CXLine{}, 4},
		{CXArgument{}, 18},
		{CXArgumentDebug{}, 3},
		{CXTypeSignature{}, 9},
		{CXTypeSignature_Array{}, 3},
		{CXTypeSignature_Struct{}, 2},
		{CXCall{}, 3},
		{CXLineMng{}, 8},
	}
	for _, count := range counts {
		if got := reflect.TypeOf(count.value).NumField(); got != count.fields {
			t.Errorf("%T has %d fields, the serializer was written for %d", count.value, got, count.fields)
		}
	}
}
