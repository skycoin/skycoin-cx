package reflecttable

import (
	"testing"

	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
)

// TestTypeHandleLayout checks the bits of a type handle against D153: the
// index in 0 to 31, the kind in 32 to 39, the generation in 40 to 55 and the
// tag in 56 to 63.
func TestTypeHandleLayout(t *testing.T) {
	typeGid := TypeGidCompose(0x12345678, KindPointer)
	if uint64(typeGid)&0xFFFFFFFF != 0x12345678 || uint64(typeGid)>>32&0xFF != uint64(KindPointer) ||
		uint64(typeGid)>>40&0xFFFF != 1 || objmodel.GidTypeTag(typeGid) != TypeTag || typeGid <= 0 {
		t.Errorf("type handle %#x does not have the D153 layout", typeGid)
	}
	if objmodel.GidIndex(typeGid) != 0x12345678 {
		t.Errorf("GidIndex reads %#x from a type handle", objmodel.GidIndex(typeGid))
	}
}

// TestTagRanges checks each tag against the ranges of D154.
func TestTagRanges(t *testing.T) {
	for _, tag := range []objmodel.TypeTag{TypeTag, FunctionTag} {
		if tag < 0x40 || tag > 0x5F {
			t.Errorf("tag %#x is outside the reflect range 0x40 to 0x5F", tag)
		}
	}
	for _, tag := range []objmodel.TypeTag{TypeNameTag, MemberNameTag} {
		if tag < 0x01 || tag > 0x1F {
			t.Errorf("pool tag %#x is outside 0x01 to 0x1F", tag)
		}
	}
	if objmodel.GidTypeTagNameTable[TypeNameTag] != "" {
		t.Errorf("tag %#x of the type name pool is taken by %s", TypeNameTag, objmodel.GidTypeTagNameTable[TypeNameTag])
	}
	if objmodel.GidTypeTagNameTable[MemberNameTag] != "String" {
		t.Errorf("tag %#x is not the StringGid tag", MemberNameTag)
	}
}

// testTable is a struct Node with a field value i32 and a field next *Node,
// a method Visit(Node, i32) bool, and an array [3]i32.
func testTable() []byte {
	var builder Builder
	i32 := builder.BuilderTypeAdd(Type{TypeKind: KindScalar, TypeSize: 4, TypeAlignment: 4, TypeName: builder.BuilderTypeNames.NamePoolIntern("i32"), TypeElement: -1, TypeLength: -1, TypeFunction: -1})
	boolean := builder.BuilderTypeAdd(Type{TypeKind: KindScalar, TypeSize: 1, TypeAlignment: 1, TypeName: builder.BuilderTypeNames.NamePoolIntern("bool"), TypeElement: -1, TypeLength: -1, TypeFunction: -1})
	node := builder.BuilderTypeAdd(Type{TypeKind: KindStruct, TypeSize: 8, TypeAlignment: 4, TypeName: builder.BuilderTypeNames.NamePoolIntern("Node"), TypeElement: -1, TypeLength: -1, TypeFunction: -1})
	pointer := builder.BuilderTypeAdd(Type{TypeKind: KindPointer, TypeSize: 4, TypeAlignment: 4, TypeName: -1, TypeElement: node, TypeLength: -1, TypeFunction: -1})
	builder.BuilderTypeAdd(Type{TypeKind: KindArray, TypeSize: 12, TypeAlignment: 4, TypeName: -1, TypeElement: i32, TypeLength: 3, TypeFunction: -1})
	visit := builder.BuilderFunctionAdd(Function{FunctionName: builder.BuilderMemberNames.NamePoolIntern("Node.Visit"), FunctionInputs: []int32{node, i32}, FunctionOutputs: []int32{boolean}})
	builder.BuilderTypes[node].TypeFields = []Field{
		{FieldName: builder.BuilderMemberNames.NamePoolIntern("value"), FieldType: i32, FieldOffset: 0},
		{FieldName: builder.BuilderMemberNames.NamePoolIntern("next"), FieldType: pointer, FieldOffset: 4},
	}
	builder.BuilderTypes[node].TypeMethods = []Method{{MethodName: builder.BuilderMemberNames.NamePoolIntern("Visit"), MethodFunction: visit}}
	return builder.BuilderEncode()
}

func testNameText(table []byte, name int64) string {
	buffer := make([]byte, 32)
	length := NameTextRead(table, name, buffer)
	if length < 0 {
		return "(none)"
	}
	return string(buffer[:length])
}

// TestTableRead reads every record kind back from an encoded table.
func TestTableRead(t *testing.T) {
	table := testTable()
	node := TypeOfIndex(table, 2)
	if TypeKindGet(table, node) != KindStruct || TypeSizeInBytesGet(table, node) != 8 || testNameText(table, TypeNameGet(table, node)) != "Node" {
		t.Fatalf("struct reads kind %d size %d", TypeKindGet(table, node), TypeSizeInBytesGet(table, node))
	}
	if StructFieldCountGet(table, node) != 2 || StructFieldOffsetInBytesGet(table, node, 1) != 4 || testNameText(table, StructFieldNameGet(table, node, 1)) != "next" {
		t.Errorf("fields read wrong")
	}
	if PointerElementTypeGet(table, StructFieldTypeGet(table, node, 1)) != node {
		t.Errorf("next does not point back to Node")
	}
	if StructFieldOrdinalByNameGet(table, node, StructFieldNameGet(table, node, 1)) != 1 || StructFieldOrdinalByNameGet(table, node, TypeNameGet(table, node)) != -1 {
		t.Errorf("field ordinal by name reads wrong")
	}
	array := TypeOfIndex(table, 4)
	if ArrayLengthGet(table, array) != 3 || ArrayElementTypeGet(table, array) != TypeOfIndex(table, 0) || TypeNameGet(table, array) != 0 {
		t.Errorf("array reads wrong")
	}
	function := StructMethodFunctionGet(table, node, 0)
	if objmodel.GidTypeTag(function) != FunctionTag || FunctionInputCountGet(table, function) != 2 || FunctionOutputCountGet(table, function) != 1 ||
		FunctionInputTypeGet(table, function, 0) != node || testNameText(table, TypeNameGet(table, FunctionOutputTypeGet(table, function, 0))) != "bool" ||
		testNameText(table, FunctionNameGet(table, function)) != "Node.Visit" || testNameText(table, StructMethodNameGet(table, node, 0)) != "Visit" {
		t.Errorf("method function reads wrong")
	}
	value := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	buffer := make([]byte, 4)
	if StructFieldValueRead(table, value, node, 1, buffer) != 4 || buffer[0] != 5 {
		t.Errorf("field read copies %v", buffer)
	}
	if StructFieldValueWrite(table, value, node, 0, []byte{9, 9, 9, 9}) != 4 || value[0] != 9 || value[4] != 5 {
		t.Errorf("field write leaves %v", value)
	}
	if StructFieldValueRead(table, value[:6], node, 1, buffer) != -1 || StructFieldValueRead(table, value, node, 1, buffer[:2]) != -1 {
		t.Errorf("a short value or buffer is not refused")
	}
}

// TestTableInvalidHandles checks that a handle of the wrong tag, kind,
// generation or index reads as no type, and a truncated table as empty.
func TestTableInvalidHandles(t *testing.T) {
	table := testTable()
	node := TypeOfIndex(table, 2)
	wrongKind := node&^(0xFF<<32) | int64(KindArray)<<32
	wrongGeneration := node + 1<<40
	pastEnd := TypeGidCompose(100, KindStruct)
	for _, typeGid := range []int64{0, -1, wrongKind, wrongGeneration, pastEnd, StructFieldNameGet(table, node, 0)} {
		if TypeKindGet(table, typeGid) != KindInvalid || TypeSizeInBytesGet(table, typeGid) != -1 || StructFieldCountGet(table, typeGid) != -1 {
			t.Errorf("handle %#x reads as a type", typeGid)
		}
	}
	if StructFieldCountGet(table, TypeOfIndex(table, 0)) != -1 || ArrayLengthGet(table, node) != -1 || FunctionInputCountGet(table, node) != -1 {
		t.Errorf("a handle of another kind is not refused")
	}
	if TypeKindGet(table[:20], node) != KindInvalid || NameTextRead(table[:20], TypeNameGet(table, node), make([]byte, 8)) != -1 {
		t.Errorf("a truncated table reads past its end")
	}
}
