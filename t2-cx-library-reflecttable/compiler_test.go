package reflecttable

import (
	"testing"
)

// compilerSourceTable is the table the compiler builds for the declarations
// of another program: struct Point {X i32, Next *Point} and func move(p
// Point) (q Point).
func compilerSourceTable() []byte {
	var builder Builder
	i32 := builder.BuilderTypeAdd(Type{TypeKind: KindScalar, TypeSize: 4, TypeAlignment: 4, TypeName: builder.BuilderTypeNames.NamePoolIntern("i32"), TypeElement: -1, TypeLength: -1, TypeFunction: -1})
	point := builder.BuilderTypeAdd(Type{TypeKind: KindStruct, TypeSize: 12, TypeAlignment: 8, TypeName: builder.BuilderTypeNames.NamePoolIntern("Point"), TypeElement: -1, TypeLength: -1, TypeFunction: -1})
	pointer := builder.BuilderTypeAdd(Type{TypeKind: KindPointer, TypeSize: 8, TypeAlignment: 8, TypeName: -1, TypeElement: point, TypeLength: -1, TypeFunction: -1})
	builder.BuilderTypes[point].TypeFields = []Field{
		{FieldName: builder.BuilderMemberNames.NamePoolIntern("X"), FieldType: i32, FieldOffset: 0},
		{FieldName: builder.BuilderMemberNames.NamePoolIntern("Next"), FieldType: pointer, FieldOffset: 4},
	}
	builder.BuilderFunctionAdd(Function{FunctionName: builder.BuilderMemberNames.NamePoolIntern("move"), FunctionInputs: []int32{point}, FunctionOutputs: []int32{point}})
	return builder.BuilderEncode()
}

// TestTableAppend appends the declarations of another program to a table
// with room and reads them back with the reflect functions.
func TestTableAppend(t *testing.T) {
	table, i32, _ := metaTestTable()
	source := compilerSourceTable()
	typeMap, functionMap := make([]int32, 3), make([]int32, 1)
	if !TableAppend(table, source, typeMap, functionMap) {
		t.Fatal("append refused")
	}
	point := TypeOfIndex(table, typeMap[1])
	text := make([]byte, 8)
	if count := NameTextRead(table, TypeNameGet(table, point), text); string(text[:count]) != "Point" {
		t.Errorf("struct name %q", text[:count])
	}
	next := StructFieldTypeGet(table, point, 1)
	if StructFieldCountGet(table, point) != 2 || PointerElementTypeGet(table, next) != point || StructFieldOffsetInBytesGet(table, point, 1) != 4 {
		t.Errorf("fields %d, Next points to %#x, want %#x", StructFieldCountGet(table, point), PointerElementTypeGet(table, next), point)
	}
	if TypeNameGet(table, StructFieldTypeGet(table, point, 0)) != TypeNameGet(table, i32) {
		t.Errorf("the i32 of the source does not share the name of the table")
	}
	move := FunctionOfIndex(table, functionMap[0])
	if count := NameTextRead(table, FunctionNameGet(table, move), text); string(text[:count]) != "move" || FunctionInputTypeGet(table, move, 0) != point {
		t.Errorf("function %q takes %#x", text[:count], FunctionInputTypeGet(table, move, 0))
	}
	if FunctionOfIndex(table, functionMap[0]+1) != 0 {
		t.Errorf("a function past the table has a handle")
	}
}

// TestTableAppendFull refuses a table without room and leaves it as it was.
func TestTableAppendFull(t *testing.T) {
	var builder Builder
	table := builder.BuilderEncode()
	before := string(table)
	if TableAppend(table, compilerSourceTable(), make([]int32, 3), make([]int32, 1)) {
		t.Fatal("a table without room took the records")
	}
	if string(table) != before {
		t.Errorf("a refused append changed the table")
	}
	if TableAppend(table, compilerSourceTable(), make([]int32, 2), make([]int32, 1)) {
		t.Errorf("a short type map was accepted")
	}
}
