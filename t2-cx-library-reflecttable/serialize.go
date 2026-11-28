package reflecttable

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
)

// The encoding of the serialize library, sys/serialize (decision D150). A
// scalar is its bytes, little endian, a str the 8 byte value of its slot, a
// struct its fields in order with no padding, an array its elements, and a
// slice a 4 byte element count then its elements. A pointer or a function
// does not encode.
const (
	SerializeStrSizeInBytes        = 8
	SerializeSliceCountSizeInBytes = 4
)

// serializeDepthLimit bounds the nesting of the types a walk follows, so a
// slice that holds itself fails rather than recursing without end.
const serializeDepthLimit = 32

// typeNameStr and typeNameBool are the type names typeNamed looks for.
var (
	typeNameStr  = [...]byte{'s', 't', 'r'}
	typeNameBool = [...]byte{'b', 'o', 'o', 'l'}
)

// serializeFailed is the result of a walk that cannot encode or decode.
const serializeFailed = -1

// typeRecordWord returns word field of type number index, which TypeIndex
// or a field record gave.
func typeRecordWord(table []byte, index int32, field int) int32 {
	return record(table, HeaderTypesOffset, index, TypeWordCount, field)
}

// typeRecordValid reports whether index is a type number of table.
func typeRecordValid(table []byte, index int32) bool {
	return index >= 0 && index < word(table, HeaderTypeCount)
}

// typeNamed reports whether type number index is named text, which tells
// str and bool from the other scalars of their size.
func typeNamed(table []byte, index int32, text []byte) bool {
	name := typeRecordWord(table, index, TypeName)
	if name < 0 || name >= word(table, HeaderTypeNameCount) {
		return false
	}
	offset := int64(word(table, HeaderNameBytesOffset)) + int64(record(table, HeaderTypeNamesOffset, name, NameWordCount, 0))
	length := int64(record(table, HeaderTypeNamesOffset, name, NameWordCount, 1))
	return length == int64(len(text)) && offset >= 0 && offset+length <= int64(len(table)) && bytes.Equal(table[offset:offset+length], text)
}

// TypeEncodedSizeInBytesGet returns the encoded size of a value of
// type, -1 for no type, a type that does not encode, or one with a slice,
// whose size depends on the value.
func TypeEncodedSizeInBytesGet(table []byte, typeGid int64) int32 {
	index, found := TypeIndex(table, typeGid)
	if !found {
		return serializeFailed
	}
	size := serializeSize(table, index, 0)
	if size < 0 || size > math.MaxInt32 {
		return serializeFailed
	}
	return int32(size)
}

func serializeSize(table []byte, index int32, depth int) int64 {
	if depth > serializeDepthLimit || !typeRecordValid(table, index) {
		return serializeFailed
	}
	switch typeRecordWord(table, index, TypeKind) {
	case KindScalar:
		if typeNamed(table, index, typeNameStr[:]) {
			return SerializeStrSizeInBytes
		}
		if size := typeRecordWord(table, index, TypeSize); size > 0 {
			return int64(size)
		}
	case KindStruct:
		first := typeRecordWord(table, index, TypeFieldFirst)
		count := typeRecordWord(table, index, TypeFieldCount)
		total := int64(0)
		for ordinal := int32(0); ordinal < count; ordinal++ {
			size := serializeSize(table, record(table, HeaderFieldsOffset, first+ordinal, FieldWordCount, FieldType), depth+1)
			if size < 0 {
				return serializeFailed
			}
			if total += size; total > math.MaxInt32 {
				return serializeFailed
			}
		}
		return total
	case KindArray:
		length := typeRecordWord(table, index, TypeLength)
		size := serializeSize(table, typeRecordWord(table, index, TypeElement), depth+1)
		if size < 0 || length < 0 || int64(length)*size > math.MaxInt32 {
			return serializeFailed
		}
		return int64(length) * size
	}
	return serializeFailed
}

// serializeWalk is one encode or decode of a value. memory is all of
// program memory, where a slice points; buffer the encoded bytes and at the
// next of them. A decode with write false only checks the buffer.
type serializeWalk struct {
	walkTable  []byte
	walkMemory []byte
	walkBuffer []byte
	walkAt     int
	walkDecode bool
	walkWrite  bool
}

// ValueEncode encodes value, the bytes of a value of type, into
// buffer and returns the bytes written, -1 when buffer is short or the type
// does not encode.
func ValueEncode(table []byte, memory []byte, value []byte, typeGid int64, buffer []byte) int32 {
	walk := serializeWalk{walkTable: table, walkMemory: memory, walkBuffer: buffer}
	return walk.serializeRun(value, typeGid)
}

// ValueDecode decodes buffer into value, the bytes of a value of
// type, and returns the bytes read, -1 when buffer is short or bad. A bad
// buffer leaves value as it was.
func ValueDecode(table []byte, memory []byte, buffer []byte, value []byte, typeGid int64) int32 {
	walk := serializeWalk{walkTable: table, walkMemory: memory, walkBuffer: buffer, walkDecode: true}
	if walk.serializeRun(value, typeGid) < 0 {
		return serializeFailed
	}
	walk.walkAt, walk.walkWrite = 0, true
	return walk.serializeRun(value, typeGid)
}

func (walk *serializeWalk) serializeRun(value []byte, typeGid int64) int32 {
	index, found := TypeIndex(walk.walkTable, typeGid)
	if !found || int64(typeRecordWord(walk.walkTable, index, TypeSize)) > int64(len(value)) {
		return serializeFailed
	}
	if !walk.serializeValue(value, index, 0) || walk.walkAt > math.MaxInt32 {
		return serializeFailed
	}
	return int32(walk.walkAt)
}

// serializeBytes returns the next size bytes of the buffer, false when it
// is short.
func (walk *serializeWalk) serializeBytes(size int) ([]byte, bool) {
	if size > len(walk.walkBuffer)-walk.walkAt {
		return nil, false
	}
	encoded := walk.walkBuffer[walk.walkAt : walk.walkAt+size]
	walk.walkAt += size
	return encoded, true
}

// serializeValue encodes or decodes value, of type number index.
func (walk *serializeWalk) serializeValue(value []byte, index int32, depth int) bool {
	table := walk.walkTable
	if depth > serializeDepthLimit || !typeRecordValid(table, index) {
		return false
	}
	size := int64(typeRecordWord(table, index, TypeSize))
	if size <= 0 || size > int64(len(value)) {
		return false
	}
	value = value[:size]
	switch typeRecordWord(table, index, TypeKind) {
	case KindScalar:
		if typeNamed(table, index, typeNameStr[:]) {
			return walk.serializeString(value)
		}
		encoded, found := walk.serializeBytes(len(value))
		if !found {
			return false
		}
		switch {
		case !walk.walkDecode:
			copy(encoded, value)
		case len(value) == int(valuetypes.BoolSize) && typeNamed(table, index, typeNameBool[:]) && encoded[0] > 1:
			return false
		case walk.walkWrite:
			copy(value, encoded)
		}
		return true
	case KindStruct:
		first := typeRecordWord(table, index, TypeFieldFirst)
		count := typeRecordWord(table, index, TypeFieldCount)
		for ordinal := int32(0); ordinal < count; ordinal++ {
			fieldType := record(table, HeaderFieldsOffset, first+ordinal, FieldWordCount, FieldType)
			offset := int64(record(table, HeaderFieldsOffset, first+ordinal, FieldWordCount, FieldOffset))
			if !typeRecordValid(table, fieldType) || offset < 0 || offset > size {
				return false
			}
			if !walk.serializeValue(value[offset:], fieldType, depth+1) {
				return false
			}
		}
		return true
	case KindArray:
		return walk.serializeElements(value, typeRecordWord(table, index, TypeElement), int64(typeRecordWord(table, index, TypeLength)), depth)
	case KindSlice:
		return walk.serializeSlice(value, typeRecordWord(table, index, TypeElement), depth)
	}
	return false
}

// serializeElements encodes or decodes count elements of type number
// element, one after the other in elements.
func (walk *serializeWalk) serializeElements(elements []byte, element int32, count int64, depth int) bool {
	if !typeRecordValid(walk.walkTable, element) {
		return false
	}
	size := int64(typeRecordWord(walk.walkTable, element, TypeSize))
	if size <= 0 || count < 0 || count*size > int64(len(elements)) {
		return false
	}
	for i := int64(0); i < count; i++ {
		if !walk.serializeValue(elements[i*size:], element, depth+1) {
			return false
		}
	}
	return true
}

// serializeString encodes the str slot value as 8 bytes, or decodes 8 bytes
// into it. A decode takes 0 or the address of an object in memory.
func (walk *serializeWalk) serializeString(value []byte) bool {
	encoded, found := walk.serializeBytes(SerializeStrSizeInBytes)
	if !found {
		return false
	}
	if !walk.walkDecode {
		binary.LittleEndian.PutUint64(encoded, uint64(valuetypes.PtrRead(value, 0)))
		return true
	}
	address := binary.LittleEndian.Uint64(encoded)
	if address != 0 && !serializeObjectValid(walk.walkMemory, address) {
		return false
	}
	if walk.walkWrite {
		valuetypes.PtrWrite(value, 0, valuetypes.Pointer(address))
	}
	return true
}

// serializeObjectValid reports whether a heap object starts at address and
// ends inside memory, by the size in its header.
func serializeObjectValid(memory []byte, address uint64) bool {
	const header = uint64(valuetypes.ObjectHeaderSize)
	if address >= uint64(valuetypes.InvalidPointer) || address+header > uint64(len(memory)) {
		return false
	}
	size := uint64(valuetypes.PtrRead(memory, valuetypes.Pointer(address+header)-valuetypes.PointerSize))
	return size >= header && address+size <= uint64(len(memory))
}

// serializeSlice encodes the slice whose pointer is value as its count and
// elements, or decodes them into it. A decode needs a slice with room for
// the count and sets its length.
func (walk *serializeWalk) serializeSlice(value []byte, element int32, depth int) bool {
	memory := walk.walkMemory
	address := uint64(valuetypes.PtrRead(value, 0))
	const lengthAt = uint64(valuetypes.ObjectHeaderSize) + uint64(valuetypes.PointerSize)
	const elementsAt = uint64(valuetypes.ObjectHeaderSize) + uint64(opcodetable.SliceHeaderSize)
	capacity, length := uint64(0), uint64(0)
	if address != 0 {
		if !serializeObjectValid(memory, address) || address+elementsAt > uint64(len(memory)) {
			return false
		}
		capacity = uint64(valuetypes.PtrRead(memory, valuetypes.Pointer(address)+valuetypes.ObjectHeaderSize))
		length = uint64(valuetypes.PtrRead(memory, valuetypes.Pointer(address+lengthAt)))
	}
	encoded, found := walk.serializeBytes(SerializeSliceCountSizeInBytes)
	if !found {
		return false
	}
	if !walk.walkDecode {
		if length > math.MaxInt32 {
			return false
		}
		binary.LittleEndian.PutUint32(encoded, uint32(length))
	} else {
		length = uint64(binary.LittleEndian.Uint32(encoded))
		if length > capacity {
			return false
		}
		if walk.walkWrite && address != 0 {
			valuetypes.PtrWrite(memory, valuetypes.Pointer(address+lengthAt), valuetypes.Pointer(length))
		}
	}
	if length == 0 {
		return true
	}
	return walk.serializeElements(memory[address+elementsAt:], element, int64(length), depth)
}
