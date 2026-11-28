package reflecttable

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
)

// MetaRoom is the room each section of the table keeps for the records the
// meta library adds at run time (decision D151). A full section refuses the
// call, so meta never allocates.
type MetaRoom struct {
	MetaRoomTypes       int
	MetaRoomFields      int
	MetaRoomMethods     int
	MetaRoomFunctions   int
	MetaRoomParameters  int
	MetaRoomTypeNames   int
	MetaRoomMemberNames int
	MetaRoomPackages    int
	MetaRoomNameBytes   int
}

// MetaRoomDefault is the room of a program that calls the meta library, the
// same in every form, about 48 KiB of the data segment.
var MetaRoomDefault = MetaRoom{
	MetaRoomTypes:       256,
	MetaRoomFields:      1024,
	MetaRoomMethods:     256,
	MetaRoomFunctions:   256,
	MetaRoomParameters:  1024,
	MetaRoomTypeNames:   256,
	MetaRoomMemberNames: 1024,
	MetaRoomPackages:    16,
	MetaRoomNameBytes:   4096,
}

// PackageTag is the tag of a package handle of the meta library, in the
// reflect range of gid-variants D154.
const PackageTag objmodel.TypeTag = 0x42

// The words of a package record.
const (
	PackageName = iota
	PackageWordCount
)

// wordWrite sets the int32 at word index of table, nothing outside it.
func wordWrite(table []byte, index int64, value int32) {
	offset := index * 4
	if index >= 0 && offset+4 <= int64(len(table)) {
		binary.LittleEndian.PutUint32(table[offset:], uint32(value))
	}
}

// recordWrite sets word field of record number of section.
func recordWrite(table []byte, section int, number int32, size int, field int, value int32) {
	wordWrite(table, int64(word(table, int64(section)))/4+int64(number)*int64(size)+int64(field), value)
}

// sectionRoom reports whether the section whose count is at countWord has a
// free record below its capacity.
func sectionRoom(table []byte, countWord int, capacityWord int) bool {
	count := word(table, int64(countWord))
	return count >= 0 && count < word(table, int64(capacityWord))
}

// nameBytesOf returns the text of name entry index of the entries at
// entriesWord, nil when it lies outside table.
func nameBytesOf(table []byte, entriesWord int, index int32) []byte {
	offset := int64(word(table, HeaderNameBytesOffset)) + int64(record(table, entriesWord, index, NameWordCount, 0))
	length := int64(record(table, entriesWord, index, NameWordCount, 1))
	if offset < 0 || length < 0 || offset+length > int64(len(table)) {
		return nil
	}
	return table[offset : offset+length]
}

// memberNameIndex returns the member name number of name, false when it is
// no member name of table.
func memberNameIndex(table []byte, name int64) (int32, bool) {
	if objmodel.GidTypeTag(name) != MemberNameTag || objmodel.GidGeneration(name) != generation {
		return 0, false
	}
	index := int64(objmodel.GidIndex(name))
	if index >= int64(word(table, HeaderMemberNameCount)) {
		return 0, false
	}
	return int32(index), true
}

// StringPoolTextAppend returns the member name Gid of text, the name that
// is already in the table or a new one, 0 when text is empty or the table
// is full. Equal texts get equal Gids, so a name compares by its Gid.
func StringPoolTextAppend(table []byte, text []byte) int64 {
	if len(text) == 0 {
		return 0
	}
	count := word(table, HeaderMemberNameCount)
	for index := int32(0); index < count; index++ {
		if bytes.Equal(nameBytesOf(table, HeaderMemberNamesOffset, index), text) {
			return interned(MemberNameTag, index)
		}
	}
	used := word(table, HeaderNameByteCount)
	start := int64(word(table, HeaderNameBytesOffset)) + int64(used)
	if !sectionRoom(table, HeaderMemberNameCount, HeaderMemberNameCapacity) || int64(used)+int64(len(text)) > int64(word(table, HeaderNameByteCapacity)) ||
		start+int64(len(text)) > int64(len(table)) {
		return 0
	}
	copy(table[start:], text)
	recordWrite(table, HeaderMemberNamesOffset, count, NameWordCount, 0, used)
	recordWrite(table, HeaderMemberNamesOffset, count, NameWordCount, 1, int32(len(text)))
	wordWrite(table, HeaderNameByteCount, used+int32(len(text)))
	wordWrite(table, HeaderMemberNameCount, count+1)
	return interned(MemberNameTag, count)
}

// typeNameIndex returns the type name number of a type name or member name
// Gid, adding a type name entry over the same bytes for a member name.
func typeNameIndex(table []byte, name int64) (int32, bool) {
	count := word(table, HeaderTypeNameCount)
	if objmodel.GidTypeTag(name) == TypeNameTag && objmodel.GidGeneration(name) == generation {
		index := int64(objmodel.GidIndex(name))
		return int32(index), index < int64(count)
	}
	member, found := memberNameIndex(table, name)
	if !found {
		return 0, false
	}
	text := nameBytesOf(table, HeaderMemberNamesOffset, member)
	for index := int32(0); index < count; index++ {
		if bytes.Equal(nameBytesOf(table, HeaderTypeNamesOffset, index), text) {
			return index, true
		}
	}
	if !sectionRoom(table, HeaderTypeNameCount, HeaderTypeNameCapacity) {
		return 0, false
	}
	recordWrite(table, HeaderTypeNamesOffset, count, NameWordCount, 0, record(table, HeaderMemberNamesOffset, member, NameWordCount, 0))
	recordWrite(table, HeaderTypeNamesOffset, count, NameWordCount, 1, record(table, HeaderMemberNamesOffset, member, NameWordCount, 1))
	wordWrite(table, HeaderTypeNameCount, count+1)
	return count, true
}

// typeAdd appends a type record and returns its number, -1 when the table
// is full.
func typeAdd(table []byte, values [TypeWordCount]int32) int32 {
	if !sectionRoom(table, HeaderTypeCount, HeaderTypeCapacity) {
		return -1
	}
	index := word(table, HeaderTypeCount)
	for field, value := range values {
		recordWrite(table, HeaderTypesOffset, index, TypeWordCount, field, value)
	}
	wordWrite(table, HeaderTypeCount, index+1)
	return index
}

// structOpenIndex returns the type number of the struct being defined when
// structType is its handle.
func structOpenIndex(table []byte, structType int64) (int32, bool) {
	open := word(table, HeaderStructOpen)
	if open < 0 || objmodel.GidTypeTag(structType) != TypeTag || uint64(structType)>>40&0xFFFF != generation ||
		uint64(structType)>>32&0xFF != uint64(KindStruct) || int64(objmodel.GidIndex(structType)) != int64(open) {
		return 0, false
	}
	return open, true
}

// StructDefinitionBegin opens a struct type named by a type name or member
// name Gid and returns its handle, 0 when another struct is open, the name
// is no name of the table or the table is full. Reflect sees no such type
// until StructDefinitionFinish.
func StructDefinitionBegin(table []byte, structName int64) int64 {
	if word(table, HeaderStructOpen) >= 0 {
		return 0
	}
	name, found := typeNameIndex(table, structName)
	if !found {
		return 0
	}
	index := typeAdd(table, [TypeWordCount]int32{
		TypeKind: KindInvalid, TypeAlignment: 1, TypeName: name, TypeElement: -1, TypeLength: -1,
		TypeFieldFirst: word(table, HeaderFieldCount), TypeMethodFirst: word(table, HeaderMethodCount), TypeFunction: -1,
	})
	if index < 0 {
		return 0
	}
	wordWrite(table, HeaderStructOpen, index)
	return TypeGidCompose(index, KindStruct)
}

// structMemberNamed reports whether the open struct has a member named by
// member name number name in the section of its first and count words.
func structMemberNamed(table []byte, index int32, name int32, first int, count int, section int, size int) bool {
	start := record(table, HeaderTypesOffset, index, TypeWordCount, first)
	total := record(table, HeaderTypesOffset, index, TypeWordCount, count)
	for ordinal := int32(0); ordinal < total; ordinal++ {
		if record(table, section, start+ordinal, size, 0) == name {
			return true
		}
	}
	return false
}

// StructDefinitionFieldAppend appends a field of fieldType to the open
// struct and returns its ordinal, -1 when structType is not open, the name
// is no member name or is taken, fieldType is no finished type or the table
// is full.
func StructDefinitionFieldAppend(table []byte, structType int64, fieldName int64, fieldType int64) int32 {
	index, open := structOpenIndex(table, structType)
	name, named := memberNameIndex(table, fieldName)
	typeIndex, typed := TypeIndex(table, fieldType)
	if !open || !named || !typed || structMemberNamed(table, index, name, TypeFieldFirst, TypeFieldCount, HeaderFieldsOffset, FieldWordCount) ||
		!sectionRoom(table, HeaderFieldCount, HeaderFieldCapacity) {
		return -1
	}
	count := word(table, HeaderFieldCount)
	recordWrite(table, HeaderFieldsOffset, count, FieldWordCount, FieldName, name)
	recordWrite(table, HeaderFieldsOffset, count, FieldWordCount, FieldType, typeIndex)
	recordWrite(table, HeaderFieldsOffset, count, FieldWordCount, FieldOffset, -1)
	wordWrite(table, HeaderFieldCount, count+1)
	ordinal := record(table, HeaderTypesOffset, index, TypeWordCount, TypeFieldCount)
	recordWrite(table, HeaderTypesOffset, index, TypeWordCount, TypeFieldCount, ordinal+1)
	return ordinal
}

// StructDefinitionMethodAppend appends a method to the open struct and
// returns its ordinal, -1 when structType is not open, the name is no
// member name or is taken, function is no finished function or the table is
// full. Methods keep the order they are appended in.
func StructDefinitionMethodAppend(table []byte, structType int64, methodName int64, function int64) int32 {
	index, open := structOpenIndex(table, structType)
	name, named := memberNameIndex(table, methodName)
	functionNumber, found := functionIndex(table, function)
	if !open || !named || !found || structMemberNamed(table, index, name, TypeMethodFirst, TypeMethodCount, HeaderMethodsOffset, MethodWordCount) ||
		!sectionRoom(table, HeaderMethodCount, HeaderMethodCapacity) {
		return -1
	}
	count := word(table, HeaderMethodCount)
	recordWrite(table, HeaderMethodsOffset, count, MethodWordCount, MethodName, name)
	recordWrite(table, HeaderMethodsOffset, count, MethodWordCount, MethodFunction, functionNumber)
	wordWrite(table, HeaderMethodCount, count+1)
	ordinal := record(table, HeaderTypesOffset, index, TypeWordCount, TypeMethodCount)
	recordWrite(table, HeaderTypesOffset, index, TypeWordCount, TypeMethodCount, ordinal+1)
	return ordinal
}

// StructDefinitionFinish lays the fields of the open struct out one after
// the other with no padding, as CX packs a struct (D139), and makes it a
// type reflect and serialize read. It returns 0, or -1 when structType is
// not open or the struct is larger than an int32.
func StructDefinitionFinish(table []byte, structType int64) int32 {
	index, open := structOpenIndex(table, structType)
	if !open {
		return -1
	}
	first := record(table, HeaderTypesOffset, index, TypeWordCount, TypeFieldFirst)
	count := record(table, HeaderTypesOffset, index, TypeWordCount, TypeFieldCount)
	size, alignment := int64(0), int32(1)
	for ordinal := int32(0); ordinal < count; ordinal++ {
		fieldType := record(table, HeaderFieldsOffset, first+ordinal, FieldWordCount, FieldType)
		size += int64(record(table, HeaderTypesOffset, fieldType, TypeWordCount, TypeSize))
		alignment = max(alignment, record(table, HeaderTypesOffset, fieldType, TypeWordCount, TypeAlignment))
	}
	if size > math.MaxInt32 {
		return -1
	}
	offset := int32(0)
	for ordinal := int32(0); ordinal < count; ordinal++ {
		recordWrite(table, HeaderFieldsOffset, first+ordinal, FieldWordCount, FieldOffset, offset)
		offset += record(table, HeaderTypesOffset, record(table, HeaderFieldsOffset, first+ordinal, FieldWordCount, FieldType), TypeWordCount, TypeSize)
	}
	recordWrite(table, HeaderTypesOffset, index, TypeWordCount, TypeSize, int32(size))
	recordWrite(table, HeaderTypesOffset, index, TypeWordCount, TypeAlignment, alignment)
	recordWrite(table, HeaderTypesOffset, index, TypeWordCount, TypeKind, KindStruct)
	wordWrite(table, HeaderStructOpen, -1)
	return 0
}

// compositeTypeCreate returns the handle of the array, slice or pointer
// type of element, the record already in the table or a new one, 0 when
// element is no type or the table is full. A pointer may point to the open
// struct, so a struct can hold a pointer to itself.
func compositeTypeCreate(table []byte, kind int32, element int64, length int32) int64 {
	elementIndex, found := TypeIndex(table, element)
	if !found && kind == KindPointer {
		elementIndex, found = structOpenIndex(table, element)
	}
	if !found || length < -1 {
		return 0
	}
	size, alignment := int64(valuetypes.PointerSize), int32(valuetypes.PointerSize)
	if kind == KindArray {
		size = int64(length) * int64(record(table, HeaderTypesOffset, elementIndex, TypeWordCount, TypeSize))
		alignment = record(table, HeaderTypesOffset, elementIndex, TypeWordCount, TypeAlignment)
		if size > math.MaxInt32 {
			return 0
		}
	}
	count := word(table, HeaderTypeCount)
	for index := int32(0); index < count; index++ {
		if record(table, HeaderTypesOffset, index, TypeWordCount, TypeKind) == kind &&
			record(table, HeaderTypesOffset, index, TypeWordCount, TypeElement) == elementIndex &&
			record(table, HeaderTypesOffset, index, TypeWordCount, TypeLength) == length {
			return TypeGidCompose(index, kind)
		}
	}
	index := typeAdd(table, [TypeWordCount]int32{
		TypeKind: kind, TypeSize: int32(size), TypeAlignment: alignment, TypeName: -1, TypeElement: elementIndex,
		TypeLength: length, TypeFieldFirst: 0, TypeMethodFirst: 0, TypeFunction: -1,
	})
	if index < 0 {
		return 0
	}
	return TypeGidCompose(index, kind)
}

// ArrayTypeCreate returns the type of an array of length elements, 0 when
// element is no type, length is negative, the array is larger than an int32
// or the table is full.
func ArrayTypeCreate(table []byte, element int64, length int32) int64 {
	if length < 0 {
		return 0
	}
	return compositeTypeCreate(table, KindArray, element, length)
}

// SliceTypeCreate returns the type of a slice of element, 0 when element is
// no type or the table is full.
func SliceTypeCreate(table []byte, element int64) int64 {
	return compositeTypeCreate(table, KindSlice, element, -1)
}

// PointerTypeCreate returns the type of a pointer to element, 0 when element
// is no type or the table is full.
func PointerTypeCreate(table []byte, element int64) int64 {
	return compositeTypeCreate(table, KindPointer, element, -1)
}

// functionOpenIndex returns the function number of the function being
// defined when function is its handle.
func functionOpenIndex(table []byte, function int64) (int32, bool) {
	open := word(table, HeaderFunctionOpen)
	if open < 0 || objmodel.GidTypeTag(function) != FunctionTag || objmodel.GidGeneration(function) != generation ||
		int64(objmodel.GidIndex(function)) != int64(open) {
		return 0, false
	}
	return open, true
}

// FunctionDefinitionBegin opens a function signature named by a member name
// Gid and returns its handle, 0 when another function is open, the name is
// no member name or the table is full. It records a signature and creates
// no code. Reflect sees it once FunctionDefinitionFinish counts it.
func FunctionDefinitionBegin(table []byte, functionName int64) int64 {
	name, named := memberNameIndex(table, functionName)
	if word(table, HeaderFunctionOpen) >= 0 || !named || !sectionRoom(table, HeaderFunctionCount, HeaderFunctionCapacity) {
		return 0
	}
	index := word(table, HeaderFunctionCount)
	recordWrite(table, HeaderFunctionsOffset, index, FunctionWordCount, FunctionName, name)
	recordWrite(table, HeaderFunctionsOffset, index, FunctionWordCount, FunctionInputCount, 0)
	recordWrite(table, HeaderFunctionsOffset, index, FunctionWordCount, FunctionOutputCount, 0)
	recordWrite(table, HeaderFunctionsOffset, index, FunctionWordCount, FunctionParameterFirst, word(table, HeaderParameterCount))
	wordWrite(table, HeaderFunctionOpen, index)
	return interned(FunctionTag, index)
}

// functionParameterAppend appends a parameter of parameterType to the open
// function and returns its ordinal among the inputs or the outputs. The
// name is 0 or a member name, checked and not kept, since the table holds
// the parameter types only.
func functionParameterAppend(table []byte, function int64, parameterName int64, parameterType int64, output bool) int32 {
	index, open := functionOpenIndex(table, function)
	_, named := memberNameIndex(table, parameterName)
	typeIndex, typed := TypeIndex(table, parameterType)
	if !open || !named && parameterName != 0 || !typed || !sectionRoom(table, HeaderParameterCount, HeaderParameterCapacity) {
		return -1
	}
	field := FunctionInputCount
	if output {
		field = FunctionOutputCount
	} else if record(table, HeaderFunctionsOffset, index, FunctionWordCount, FunctionOutputCount) > 0 {
		return -1
	}
	count := word(table, HeaderParameterCount)
	recordWrite(table, HeaderParametersOffset, count, 1, 0, typeIndex)
	wordWrite(table, HeaderParameterCount, count+1)
	ordinal := record(table, HeaderFunctionsOffset, index, FunctionWordCount, field)
	recordWrite(table, HeaderFunctionsOffset, index, FunctionWordCount, field, ordinal+1)
	return ordinal
}

// FunctionDefinitionInputAppend appends an input to the open function and
// returns its ordinal, -1 when function is not open, an output is already
// appended, the name is neither 0 nor a member name, inputType is no
// finished type or the table is full.
func FunctionDefinitionInputAppend(table []byte, function int64, inputName int64, inputType int64) int32 {
	return functionParameterAppend(table, function, inputName, inputType, false)
}

// FunctionDefinitionOutputAppend appends an output to the open function and
// returns its ordinal, -1 as FunctionDefinitionInputAppend.
func FunctionDefinitionOutputAppend(table []byte, function int64, outputName int64, outputType int64) int32 {
	return functionParameterAppend(table, function, outputName, outputType, true)
}

// FunctionDefinitionFinish closes the open function, so reflect reads it and
// a method can name it. It returns 0, or -1 when function is not open.
func FunctionDefinitionFinish(table []byte, function int64) int32 {
	index, open := functionOpenIndex(table, function)
	if !open {
		return -1
	}
	wordWrite(table, HeaderFunctionCount, index+1)
	wordWrite(table, HeaderFunctionOpen, -1)
	return 0
}

// packageIndex returns the package number of a package handle below limit.
func packageIndex(pkg int64, limit int32) (int32, bool) {
	if objmodel.GidTypeTag(pkg) != PackageTag || objmodel.GidGeneration(pkg) != generation {
		return 0, false
	}
	index := int64(objmodel.GidIndex(pkg))
	return int32(index), index < int64(limit)
}

// PackageDefinitionBegin opens a package named by a member name Gid and
// returns its handle, 0 when another package is open, the name is no member
// name or the table is full. A package records its name only.
func PackageDefinitionBegin(table []byte, packageName int64) int64 {
	name, named := memberNameIndex(table, packageName)
	if word(table, HeaderPackageOpen) >= 0 || !named || !sectionRoom(table, HeaderPackageCount, HeaderPackageCapacity) {
		return 0
	}
	index := word(table, HeaderPackageCount)
	recordWrite(table, HeaderPackagesOffset, index, PackageWordCount, PackageName, name)
	wordWrite(table, HeaderPackageOpen, index)
	return interned(PackageTag, index)
}

// PackageDefinitionFinish closes the open package. It returns 0, or -1 when
// pkg is not open.
func PackageDefinitionFinish(table []byte, pkg int64) int32 {
	open := word(table, HeaderPackageOpen)
	index, found := packageIndex(pkg, open+1)
	if open < 0 || !found || index != open {
		return -1
	}
	wordWrite(table, HeaderPackageCount, index+1)
	wordWrite(table, HeaderPackageOpen, -1)
	return 0
}

// PackageDefinitionNameGet returns the member name Gid of a finished
// package, 0 for any other handle.
func PackageDefinitionNameGet(table []byte, pkg int64) int64 {
	index, found := packageIndex(pkg, word(table, HeaderPackageCount))
	if !found {
		return 0
	}
	return interned(MemberNameTag, record(table, HeaderPackagesOffset, index, PackageWordCount, PackageName))
}
