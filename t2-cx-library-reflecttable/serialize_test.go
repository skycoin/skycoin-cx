package reflecttable

import (
	"bytes"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
)

// serializeTestTable is a struct Record {id i32, ready bool, name str}, a
// slice []i32 and a pointer *i32, with the handles of each.
func serializeTestTable() (table []byte, record int64, slice int64, pointer int64) {
	var builder Builder
	scalar := func(name string, size int32) int32 {
		return builder.BuilderTypeAdd(Type{TypeKind: KindScalar, TypeSize: size, TypeAlignment: size, TypeName: builder.BuilderTypeNames.NamePoolIntern(name), TypeElement: -1, TypeLength: -1, TypeFunction: -1})
	}
	i32, boolean, str := scalar("i32", 4), scalar("bool", 1), scalar("str", int32(valuetypes.PointerSize))
	recordIndex := builder.BuilderTypeAdd(Type{TypeKind: KindStruct, TypeSize: 5 + int32(valuetypes.PointerSize), TypeAlignment: 4, TypeName: builder.BuilderTypeNames.NamePoolIntern("Record"), TypeElement: -1, TypeLength: -1, TypeFunction: -1,
		TypeFields: []Field{{FieldType: i32, FieldOffset: 0}, {FieldType: boolean, FieldOffset: 4}, {FieldType: str, FieldOffset: 5}}})
	sliceIndex := builder.BuilderTypeAdd(Type{TypeKind: KindSlice, TypeSize: int32(valuetypes.PointerSize), TypeAlignment: 4, TypeName: -1, TypeElement: i32, TypeLength: -1, TypeFunction: -1})
	pointerIndex := builder.BuilderTypeAdd(Type{TypeKind: KindPointer, TypeSize: int32(valuetypes.PointerSize), TypeAlignment: 4, TypeName: -1, TypeElement: i32, TypeLength: -1, TypeFunction: -1})
	return builder.BuilderEncode(), TypeGidCompose(recordIndex, KindStruct), TypeGidCompose(sliceIndex, KindSlice), TypeGidCompose(pointerIndex, KindPointer)
}

// serializeTestObject writes an object of payload bytes at address of
// memory, as the heap lays it out.
func serializeTestObject(memory []byte, address int, payload int) {
	valuetypes.PtrWrite(memory, valuetypes.Pointer(address)+valuetypes.ObjectHeaderSize-valuetypes.PointerSize, valuetypes.ObjectHeaderSize+valuetypes.Pointer(payload))
}

func TestSerializeRecordRoundTrip(t *testing.T) {
	table, record, _, pointer := serializeTestTable()
	memory := make([]byte, 256)
	serializeTestObject(memory, 100, 3)
	value := memory[0:16]
	valuetypes.I32Write(value, 0, -2)
	value[4] = 1
	valuetypes.PtrWrite(value, 5, 100)

	if size := TypeEncodedSizeInBytesGet(table, record); size != 13 {
		t.Fatalf("encoded size %d, want 13", size)
	}
	buffer := make([]byte, 16)
	if written := ValueEncode(table, memory, value, record, buffer); written != 13 {
		t.Fatalf("wrote %d, want 13", written)
	}
	if want := []byte{254, 255, 255, 255, 1, 100, 0, 0, 0, 0, 0, 0, 0}; !bytes.Equal(buffer[:13], want) {
		t.Fatalf("bytes %v, want %v", buffer[:13], want)
	}
	if written := ValueEncode(table, memory, value, record, buffer[:12]); written != -1 {
		t.Errorf("encode into 12 bytes %d, want -1", written)
	}
	back := memory[16:32]
	if read := ValueDecode(table, memory, buffer, back, record); read != 13 || !bytes.Equal(back[:5+valuetypes.PointerSize], value[:5+valuetypes.PointerSize]) {
		t.Fatalf("decode read %d, value %v, want 13 and %v", read, back, value)
	}
	if read := ValueDecode(table, memory, buffer[:12], back, record); read != -1 {
		t.Errorf("decode of 12 bytes %d, want -1", read)
	}

	// A bad bool or a str that names no object is refused and writes nothing.
	clear(back)
	for _, bad := range []func([]byte){
		func(b []byte) { b[4] = 2 },
		func(b []byte) { b[5] = 250 },
		func(b []byte) { b[12] = 1 },
	} {
		broken := bytes.Clone(buffer)
		bad(broken)
		if read := ValueDecode(table, memory, broken, back, record); read != -1 || !bytes.Equal(back, make([]byte, 16)) {
			t.Errorf("bad buffer %v read %d and wrote %v", broken[:13], read, back)
		}
	}

	if TypeEncodedSizeInBytesGet(table, pointer) != -1 || ValueEncode(table, memory, value, pointer, buffer) != -1 || TypeEncodedSizeInBytesGet(table, 0) != -1 {
		t.Error("a pointer or no type encodes")
	}
}

func TestSerializeSlice(t *testing.T) {
	table, _, slice, _ := serializeTestTable()
	memory := make([]byte, 256)
	const header = int(valuetypes.ObjectHeaderSize)
	const elements = header + int(opcodetable.SliceHeaderSize)
	for _, address := range []int{64, 128} {
		serializeTestObject(memory, address, int(opcodetable.SliceHeaderSize)+16)
		valuetypes.PtrWrite(memory, valuetypes.Pointer(address+header), 4)
	}
	valuetypes.PtrWrite(memory, valuetypes.Pointer(64+header)+valuetypes.PointerSize, 2)
	valuetypes.I32Write(memory, valuetypes.Pointer(64+elements), 1)
	valuetypes.I32Write(memory, valuetypes.Pointer(64+elements+4), 258)
	valuetypes.PtrWrite(memory, 0, 64)
	valuetypes.PtrWrite(memory, 8, 128)

	if size := TypeEncodedSizeInBytesGet(table, slice); size != -1 {
		t.Errorf("slice encoded size %d, want -1", size)
	}
	buffer := make([]byte, 16)
	if written := ValueEncode(table, memory, memory[0:8], slice, buffer); written != 12 || !bytes.Equal(buffer[:12], []byte{2, 0, 0, 0, 1, 0, 0, 0, 2, 1, 0, 0}) {
		t.Fatalf("wrote %d bytes %v", written, buffer[:12])
	}
	if read := ValueDecode(table, memory, buffer, memory[8:16], slice); read != 12 || !bytes.Equal(memory[128+elements:128+elements+8], memory[64+elements:64+elements+8]) {
		t.Fatalf("decode read %d", read)
	}
	if length := valuetypes.PtrRead(memory, valuetypes.Pointer(128+header)+valuetypes.PointerSize); length != 2 {
		t.Errorf("decoded length %d, want 2", length)
	}
	buffer[0] = 5
	if read := ValueDecode(table, memory, buffer, memory[8:16], slice); read != -1 {
		t.Errorf("a count past the capacity read %d, want -1", read)
	}
}
