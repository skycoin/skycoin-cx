// Package reflecttable is the type table of the reflect library, sys/reflect
// (decision D150). The compiler builds it once from the program, and every
// form reads the same little endian bytes: the old loop from the library
// state, vm, an image and a native program from the data segment.
package reflecttable

import (
	"encoding/binary"

	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
)

// The kinds CxTypeKindGet returns. KindInvalid is the kind of no type.
const (
	KindInvalid  int32 = 0
	KindScalar   int32 = 1
	KindStruct   int32 = 2
	KindArray    int32 = 3
	KindSlice    int32 = 4
	KindPointer  int32 = 5
	KindFunction int32 = 6
)

// The type tags of the handles, see gid-variants D154. TypeTag and
// FunctionTag are in the reflect range 0x40 to 0x5F, TypeNameTag is the
// interned type name pool and MemberNameTag is the StringGid tag.
const (
	TypeTag       objmodel.TypeTag = 0x40
	FunctionTag   objmodel.TypeTag = 0x41
	TypeNameTag   objmodel.TypeTag = 0x11
	MemberNameTag objmodel.TypeTag = 0x0F
)

// The words of the table header, each an int32 at four times its number.
const (
	HeaderTypeCount = iota
	HeaderTypesOffset
	HeaderFieldsOffset
	HeaderMethodsOffset
	HeaderFunctionCount
	HeaderFunctionsOffset
	HeaderParametersOffset
	HeaderTypeNameCount
	HeaderTypeNamesOffset
	HeaderMemberNameCount
	HeaderMemberNamesOffset
	HeaderNameBytesOffset
	HeaderWordCount
)

// The words of a type record. TypeElement is the element of an array, a
// slice or a pointer, TypeLength the length of an array and TypeFunction the
// function record of a function type, each -1 when there is none.
const (
	TypeKind = iota
	TypeSize
	TypeAlignment
	TypeName
	TypeElement
	TypeLength
	TypeFieldFirst
	TypeFieldCount
	TypeMethodFirst
	TypeMethodCount
	TypeFunction
	TypeWordCount
)

// The words of a field record.
const (
	FieldName = iota
	FieldType
	FieldOffset
	FieldWordCount
)

// The words of a method record, its name and its function record.
const (
	MethodName = iota
	MethodFunction
	MethodWordCount
)

// The words of a function record. Its parameter types are the inputs and
// then the outputs from FunctionParameterFirst on.
const (
	FunctionName = iota
	FunctionInputCount
	FunctionOutputCount
	FunctionParameterFirst
	FunctionWordCount
)

// NameWordCount is the size of a name entry in words, its offset from the
// name bytes and its length.
const NameWordCount = 2

// generation is the generation of every handle, since nothing in the table
// is ever released.
const generation = 1

// word returns the int32 at word index of table, 0 outside it, so a table a
// program overwrote reads as empty rather than out of range.
func word(table []byte, index int64) int32 {
	offset := index * 4
	if index < 0 || offset+4 > int64(len(table)) {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(table[offset:]))
}

// record returns word field of record number of the section whose header
// word is section, records of size words each.
func record(table []byte, section int, number int32, size int, field int) int32 {
	return word(table, int64(word(table, int64(section)))/4+int64(number)*int64(size)+int64(field))
}

// TypeHandle returns the handle of type number index of kind, bits 0 to 31
// the index, 32 to 39 the kind, 40 to 55 the generation and 56 to 63 the tag
// (D153).
func TypeGidCompose(index int32, kind int32) int64 {
	return int64(uint64(TypeTag)<<56 | uint64(generation)<<40 | uint64(uint8(kind))<<32 | uint64(uint32(index)))
}

// interned returns the Gid of slot index of the pool with tag, generation 1.
func interned(tag objmodel.TypeTag, index int32) int64 {
	return int64(uint64(tag)<<56 | uint64(generation)<<32 | uint64(uint32(index)))
}

// TypeIndex returns the type number of handle, false when handle is no type
// of table. The kind bits must match the record.
func TypeIndex(table []byte, typeGid int64) (int32, bool) {
	if objmodel.GidTypeTag(typeGid) != TypeTag || uint64(typeGid)>>40&0xFFFF != generation {
		return 0, false
	}
	index := int64(objmodel.GidIndex(typeGid))
	if index >= int64(word(table, HeaderTypeCount)) {
		return 0, false
	}
	if record(table, HeaderTypesOffset, int32(index), TypeWordCount, TypeKind) != int32(uint64(typeGid)>>32&0xFF) {
		return 0, false
	}
	return int32(index), true
}

// typeWord returns word field of the type of handle, and false when handle
// is no type or not of kind, any kind when kind is KindInvalid.
func typeWord(table []byte, typeGid int64, kind int32, field int) (int32, bool) {
	index, found := TypeIndex(table, typeGid)
	if !found {
		return 0, false
	}
	if kind != KindInvalid && record(table, HeaderTypesOffset, index, TypeWordCount, TypeKind) != kind {
		return 0, false
	}
	return record(table, HeaderTypesOffset, index, TypeWordCount, field), true
}

// typeHandleOf returns the handle of type number index, 0 for -1.
func typeGidOf(table []byte, index int32) int64 {
	if index < 0 || index >= word(table, HeaderTypeCount) {
		return 0
	}
	return TypeGidCompose(index, record(table, HeaderTypesOffset, index, TypeWordCount, TypeKind))
}

// TypeOfIndex returns the handle of type number index, which the compiler
// gives CxTypeOfValueGet.
func TypeOfIndex(table []byte, index int32) int64 {
	return typeGidOf(table, index)
}

// TypeKindGet returns the kind of type, KindInvalid for no type.
func TypeKindGet(table []byte, typeGid int64) int32 {
	kind, _ := typeWord(table, typeGid, KindInvalid, TypeKind)
	return kind
}

// TypeNameGet returns the type name Gid of type, 0 when it has no name.
func TypeNameGet(table []byte, typeGid int64) int64 {
	name, found := typeWord(table, typeGid, KindInvalid, TypeName)
	if !found || name < 0 {
		return 0
	}
	return interned(TypeNameTag, name)
}

// TypeSizeInBytesGet returns the size of a value of type, -1 for no type.
func TypeSizeInBytesGet(table []byte, typeGid int64) int32 {
	return typeWordOrMinus(table, typeGid, KindInvalid, TypeSize)
}

// TypeAlignmentInBytesGet returns the natural alignment of type, -1 for no
// type. CX packs struct fields, so a field offset need not be a multiple.
func TypeAlignmentInBytesGet(table []byte, typeGid int64) int32 {
	return typeWordOrMinus(table, typeGid, KindInvalid, TypeAlignment)
}

func typeWordOrMinus(table []byte, typeGid int64, kind int32, field int) int32 {
	value, found := typeWord(table, typeGid, kind, field)
	if !found {
		return -1
	}
	return value
}

// StructFieldCountGet returns the field count of a struct type, -1 for any
// other handle.
func StructFieldCountGet(table []byte, structType int64) int32 {
	return typeWordOrMinus(table, structType, KindStruct, TypeFieldCount)
}

// structField returns word field of field ordinal of a struct type.
func structField(table []byte, structType int64, ordinal int32, field int) (int32, bool) {
	first, found := typeWord(table, structType, KindStruct, TypeFieldFirst)
	if !found {
		return 0, false
	}
	count, _ := typeWord(table, structType, KindStruct, TypeFieldCount)
	if ordinal < 0 || ordinal >= count {
		return 0, false
	}
	return record(table, HeaderFieldsOffset, first+ordinal, FieldWordCount, field), true
}

// StructFieldNameGet returns the name Gid of field ordinal, 0 when there is
// no such field.
func StructFieldNameGet(table []byte, structType int64, ordinal int32) int64 {
	name, found := structField(table, structType, ordinal, FieldName)
	if !found {
		return 0
	}
	return interned(MemberNameTag, name)
}

// StructFieldTypeGet returns the type of field ordinal, 0 when there is no
// such field.
func StructFieldTypeGet(table []byte, structType int64, ordinal int32) int64 {
	index, found := structField(table, structType, ordinal, FieldType)
	if !found {
		return 0
	}
	return typeGidOf(table, index)
}

// StructFieldOffsetInBytesGet returns the offset of field ordinal in the
// struct, -1 when there is no such field.
func StructFieldOffsetInBytesGet(table []byte, structType int64, ordinal int32) int32 {
	offset, found := structField(table, structType, ordinal, FieldOffset)
	if !found {
		return -1
	}
	return offset
}

// StructFieldOrdinalByNameGet returns the ordinal of the field whose name
// Gid is name, -1 when the struct has none.
func StructFieldOrdinalByNameGet(table []byte, structType int64, name int64) int32 {
	count := StructFieldCountGet(table, structType)
	for ordinal := int32(0); ordinal < count; ordinal++ {
		if StructFieldNameGet(table, structType, ordinal) == name {
			return ordinal
		}
	}
	return -1
}

// structFieldSpan returns the offset and size of field ordinal, false when
// it does not fit in a value of valueSize bytes.
func structFieldSpan(table []byte, structType int64, ordinal int32, valueSize int) (int, int, bool) {
	offset, found := structField(table, structType, ordinal, FieldOffset)
	if !found {
		return 0, 0, false
	}
	size := TypeSizeInBytesGet(table, StructFieldTypeGet(table, structType, ordinal))
	if offset < 0 || size < 0 || int64(offset)+int64(size) > int64(valueSize) {
		return 0, 0, false
	}
	return int(offset), int(size), true
}

// StructFieldValueRead copies field ordinal of the struct value into buffer
// and returns its size, -1 when there is no such field or buffer is short.
func StructFieldValueRead(table []byte, value []byte, structType int64, ordinal int32, buffer []byte) int32 {
	offset, size, found := structFieldSpan(table, structType, ordinal, len(value))
	if !found || size > len(buffer) {
		return -1
	}
	return int32(copy(buffer, value[offset:offset+size]))
}

// StructFieldValueWrite copies the first bytes of buffer into field ordinal
// of the struct value and returns the field size, -1 when there is no such
// field or buffer is short.
func StructFieldValueWrite(table []byte, value []byte, structType int64, ordinal int32, buffer []byte) int32 {
	offset, size, found := structFieldSpan(table, structType, ordinal, len(value))
	if !found || size > len(buffer) {
		return -1
	}
	return int32(copy(value[offset:offset+size], buffer))
}

// StructMethodCountGet returns the method count of a struct type, -1 for
// any other handle.
func StructMethodCountGet(table []byte, structType int64) int32 {
	return typeWordOrMinus(table, structType, KindStruct, TypeMethodCount)
}

// structMethod returns word field of method ordinal of a struct type.
func structMethod(table []byte, structType int64, ordinal int32, field int) (int32, bool) {
	first, found := typeWord(table, structType, KindStruct, TypeMethodFirst)
	if !found {
		return 0, false
	}
	count, _ := typeWord(table, structType, KindStruct, TypeMethodCount)
	if ordinal < 0 || ordinal >= count {
		return 0, false
	}
	return record(table, HeaderMethodsOffset, first+ordinal, MethodWordCount, field), true
}

// StructMethodNameGet returns the name Gid of method ordinal, 0 when there
// is no such method. The methods are in name order.
func StructMethodNameGet(table []byte, structType int64, ordinal int32) int64 {
	name, found := structMethod(table, structType, ordinal, MethodName)
	if !found {
		return 0
	}
	return interned(MemberNameTag, name)
}

// StructMethodFunctionGet returns the function handle of method ordinal, 0
// when there is no such method.
func StructMethodFunctionGet(table []byte, structType int64, ordinal int32) int64 {
	function, found := structMethod(table, structType, ordinal, MethodFunction)
	if !found {
		return 0
	}
	return interned(FunctionTag, function)
}

// ArrayLengthGet returns the length of an array type, -1 for any other
// handle.
func ArrayLengthGet(table []byte, arrayType int64) int32 {
	return typeWordOrMinus(table, arrayType, KindArray, TypeLength)
}

func elementGet(table []byte, typeGid int64, kind int32) int64 {
	element, found := typeWord(table, typeGid, kind, TypeElement)
	if !found {
		return 0
	}
	return typeGidOf(table, element)
}

// ArrayElementTypeGet returns the element type of an array type, 0 for any
// other handle.
func ArrayElementTypeGet(table []byte, arrayType int64) int64 {
	return elementGet(table, arrayType, KindArray)
}

// SliceElementTypeGet returns the element type of a slice type, 0 for any
// other handle.
func SliceElementTypeGet(table []byte, sliceType int64) int64 {
	return elementGet(table, sliceType, KindSlice)
}

// PointerElementTypeGet returns the type a pointer type points to, 0 for
// any other handle.
func PointerElementTypeGet(table []byte, pointerType int64) int64 {
	return elementGet(table, pointerType, KindPointer)
}

// functionIndex returns the function record of a function handle or of a
// function type, false for any other handle.
func functionIndex(table []byte, function int64) (int32, bool) {
	if objmodel.GidTypeTag(function) == TypeTag {
		index, found := typeWord(table, function, KindFunction, TypeFunction)
		return index, found && index >= 0
	}
	if objmodel.GidTypeTag(function) != FunctionTag || objmodel.GidGeneration(function) != generation {
		return 0, false
	}
	index := int64(objmodel.GidIndex(function))
	if index >= int64(word(table, HeaderFunctionCount)) {
		return 0, false
	}
	return int32(index), true
}

// FunctionNameGet returns the name Gid of function, 0 when it has none.
func FunctionNameGet(table []byte, function int64) int64 {
	index, found := functionIndex(table, function)
	if !found {
		return 0
	}
	name := record(table, HeaderFunctionsOffset, index, FunctionWordCount, FunctionName)
	if name < 0 {
		return 0
	}
	return interned(MemberNameTag, name)
}

func functionWordOrMinus(table []byte, function int64, field int) int32 {
	index, found := functionIndex(table, function)
	if !found {
		return -1
	}
	return record(table, HeaderFunctionsOffset, index, FunctionWordCount, field)
}

// FunctionInputCountGet returns the input count of function, the receiver
// of a method first, -1 for no function.
func FunctionInputCountGet(table []byte, function int64) int32 {
	return functionWordOrMinus(table, function, FunctionInputCount)
}

// FunctionOutputCountGet returns the output count of function, -1 for no
// function.
func FunctionOutputCountGet(table []byte, function int64) int32 {
	return functionWordOrMinus(table, function, FunctionOutputCount)
}

// functionParameterGet returns the type of parameter ordinal after skip
// parameters, of count.
func functionParameterGet(table []byte, function int64, ordinal int32, outputs bool) int64 {
	index, found := functionIndex(table, function)
	if !found {
		return 0
	}
	inputCount := record(table, HeaderFunctionsOffset, index, FunctionWordCount, FunctionInputCount)
	count, skip := inputCount, int32(0)
	if outputs {
		count, skip = record(table, HeaderFunctionsOffset, index, FunctionWordCount, FunctionOutputCount), inputCount
	}
	if ordinal < 0 || ordinal >= count {
		return 0
	}
	first := record(table, HeaderFunctionsOffset, index, FunctionWordCount, FunctionParameterFirst)
	return typeGidOf(table, record(table, HeaderParametersOffset, first+skip+ordinal, 1, 0))
}

// FunctionInputTypeGet returns the type of input ordinal of function, 0 when
// there is no such input.
func FunctionInputTypeGet(table []byte, function int64, ordinal int32) int64 {
	return functionParameterGet(table, function, ordinal, false)
}

// FunctionOutputTypeGet returns the type of output ordinal of function, 0
// when there is no such output.
func FunctionOutputTypeGet(table []byte, function int64, ordinal int32) int64 {
	return functionParameterGet(table, function, ordinal, true)
}

// NameTextRead copies the text of a type name or member name Gid into
// buffer and returns the bytes copied, -1 for no name of table. A longer
// name is cut at the size of buffer.
func NameTextRead(table []byte, name int64, buffer []byte) int32 {
	countWord, entriesWord := HeaderMemberNameCount, HeaderMemberNamesOffset
	switch objmodel.GidTypeTag(name) {
	case TypeNameTag:
		countWord, entriesWord = HeaderTypeNameCount, HeaderTypeNamesOffset
	case MemberNameTag:
	default:
		return -1
	}
	index := int64(objmodel.GidIndex(name))
	if objmodel.GidGeneration(name) != generation || index >= int64(word(table, int64(countWord))) {
		return -1
	}
	offset := int64(word(table, HeaderNameBytesOffset)) + int64(record(table, entriesWord, int32(index), NameWordCount, 0))
	length := int64(record(table, entriesWord, int32(index), NameWordCount, 1))
	if offset < 0 || length < 0 || offset+length > int64(len(table)) {
		return -1
	}
	return int32(copy(buffer, table[offset:offset+length]))
}
