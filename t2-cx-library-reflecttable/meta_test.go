package reflecttable

import (
	"testing"

	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
)

// metaTestTable is a table with the scalars i32 and i64 and the room of a
// program that calls the meta library.
func metaTestTable() ([]byte, int64, int64) {
	var builder Builder
	builder.BuilderMetaRoom = true
	i32 := builder.BuilderTypeAdd(Type{TypeKind: KindScalar, TypeSize: 4, TypeAlignment: 4, TypeName: builder.BuilderTypeNames.NamePoolIntern("i32"), TypeElement: -1, TypeLength: -1, TypeFunction: -1})
	i64 := builder.BuilderTypeAdd(Type{TypeKind: KindScalar, TypeSize: 8, TypeAlignment: 8, TypeName: builder.BuilderTypeNames.NamePoolIntern("i64"), TypeElement: -1, TypeLength: -1, TypeFunction: -1})
	table := builder.BuilderEncode()
	return table, TypeOfIndex(table, i32), TypeOfIndex(table, i64)
}

// TestPackageTagRange checks the package tag against the reflect range of
// D154 and that no other Gid uses it.
func TestPackageTagRange(t *testing.T) {
	if PackageTag < 0x40 || PackageTag > 0x5F || PackageTag == TypeTag || PackageTag == FunctionTag {
		t.Errorf("package tag %#x is outside the reflect range or taken", PackageTag)
	}
	if objmodel.GidTypeTagNameTable[PackageTag] != "" {
		t.Errorf("package tag %#x is taken by %s", PackageTag, objmodel.GidTypeTagNameTable[PackageTag])
	}
}

// TestMetaStructReadAndEncode builds a struct, reads it back with the
// reflect functions and encodes a value of it.
func TestMetaStructReadAndEncode(t *testing.T) {
	table, i32, i64 := metaTestTable()
	name := StringPoolTextAppend(table, []byte("Pair"))
	left := StringPoolTextAppend(table, []byte("left"))
	right := StringPoolTextAppend(table, []byte("right"))
	if StringPoolTextAppend(table, []byte("left")) != left {
		t.Fatal("the same text got a second name")
	}
	pair := StructDefinitionBegin(table, name)
	if pair == 0 || TypeKindGet(table, pair) != KindInvalid {
		t.Fatalf("open struct %#x reads as kind %d", pair, TypeKindGet(table, pair))
	}
	if StructDefinitionFieldAppend(table, pair, left, i32) != 0 || StructDefinitionFieldAppend(table, pair, right, ArrayTypeCreate(table, i64, 2)) != 1 {
		t.Fatal("field append refused")
	}
	if StructDefinitionFinish(table, pair) != 0 {
		t.Fatal("finish refused")
	}
	if StructDefinitionFieldAppend(table, pair, StringPoolTextAppend(table, []byte("late")), i32) != -1 {
		t.Error("a field after finish was accepted")
	}
	if TypeKindGet(table, pair) != KindStruct || TypeSizeInBytesGet(table, pair) != 20 || TypeAlignmentInBytesGet(table, pair) != 8 ||
		StructFieldCountGet(table, pair) != 2 || StructFieldOffsetInBytesGet(table, pair, 1) != 4 || StructFieldOrdinalByNameGet(table, pair, right) != 1 {
		t.Errorf("struct reads kind %d size %d alignment %d fields %d", TypeKindGet(table, pair), TypeSizeInBytesGet(table, pair),
			TypeAlignmentInBytesGet(table, pair), StructFieldCountGet(table, pair))
	}
	text := make([]byte, 8)
	if count := NameTextRead(table, TypeNameGet(table, pair), text); string(text[:count]) != "Pair" {
		t.Errorf("type name reads %q", text[:count])
	}
	value := []byte{1, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0}
	buffer := make([]byte, 32)
	if count := ValueEncode(table, nil, value, pair, buffer); count != 20 || string(buffer[:20]) != string(value) {
		t.Errorf("encode wrote %d bytes %v", count, buffer[:20])
	}
}

// TestMetaFull fills the type room and checks that every new type is then
// refused and nothing outside the table is written.
func TestMetaFull(t *testing.T) {
	table, i32, _ := metaTestTable()
	created := 0
	for length := int32(1); ArrayTypeCreate(table, i32, length) != 0; length++ {
		created++
	}
	if created != MetaRoomDefault.MetaRoomTypes {
		t.Errorf("%d types created, want %d", created, MetaRoomDefault.MetaRoomTypes)
	}
	if StructDefinitionBegin(table, StringPoolTextAppend(table, []byte("Late"))) != 0 || ArrayTypeCreate(table, i32, 1) == 0 {
		t.Error("a full table took a struct or lost an existing type")
	}
}

// TestMetaCompiledTableRefuses checks that a table without room takes no
// record, as for a program that does not call meta.
func TestMetaCompiledTableRefuses(t *testing.T) {
	table := testTable()
	i32 := TypeOfIndex(table, 0)
	if StringPoolTextAppend(table, []byte("new")) != 0 || ArrayTypeCreate(table, i32, 7) != 0 || FunctionDefinitionBegin(table, StructFieldNameGet(table, TypeOfIndex(table, 2), 0)) != 0 {
		t.Error("a table without room took a record")
	}
	if ArrayTypeCreate(table, i32, 3) != TypeOfIndex(table, 4) {
		t.Error("an existing array type was not found")
	}
}
