package reflecttable

import (
	"bytes"
)

// FunctionOfIndex returns the handle of function record index, 0 when there
// is none.
func FunctionOfIndex(table []byte, index int32) int64 {
	if index < 0 || index >= word(table, HeaderFunctionCount) {
		return 0
	}
	return interned(FunctionTag, index)
}

// tableRoomFor reports whether table has room for every record and name of
// source, counting each name as new.
func tableRoomFor(table []byte, source []byte) bool {
	sections := [...][2]int{
		{HeaderTypeCount, HeaderTypeCapacity}, {HeaderFieldCount, HeaderFieldCapacity}, {HeaderMethodCount, HeaderMethodCapacity},
		{HeaderFunctionCount, HeaderFunctionCapacity}, {HeaderParameterCount, HeaderParameterCapacity},
		{HeaderTypeNameCount, HeaderTypeNameCapacity}, {HeaderMemberNameCount, HeaderMemberNameCapacity},
		{HeaderNameByteCount, HeaderNameByteCapacity},
	}
	for _, section := range sections {
		count, capacity, added := int64(word(table, int64(section[0]))), int64(word(table, int64(section[1]))), int64(word(source, int64(section[0])))
		if count < 0 || added < 0 || count+added > capacity {
			return false
		}
	}
	return word(table, HeaderStructOpen) < 0 && word(table, HeaderFunctionOpen) < 0
}

// nameAppend returns the number of text in the name section at entriesWord
// and countWord, the entry already there or a new one over new bytes.
// tableRoomFor has checked the room.
func nameAppend(table []byte, entriesWord int, countWord int, text []byte) int32 {
	count := word(table, int64(countWord))
	for index := int32(0); index < count; index++ {
		if bytes.Equal(nameBytesOf(table, entriesWord, index), text) {
			return index
		}
	}
	used := word(table, HeaderNameByteCount)
	copy(table[int64(word(table, HeaderNameBytesOffset))+int64(used):], text)
	recordWrite(table, entriesWord, count, NameWordCount, 0, used)
	recordWrite(table, entriesWord, count, NameWordCount, 1, int32(len(text)))
	wordWrite(table, HeaderNameByteCount, used+int32(len(text)))
	wordWrite(table, int64(countWord), count+1)
	return count
}

// nameMapped returns the number in table of name number name of source in
// the section at entriesWord and countWord, -1 for -1.
func nameMapped(table []byte, source []byte, entriesWord int, countWord int, name int32) int32 {
	if name < 0 {
		return -1
	}
	return nameAppend(table, entriesWord, countWord, nameBytesOf(source, entriesWord, name))
}

// TableAppend adds every type and function record of source, a table the
// compiler built for another program, to table, in the room the compiler
// library keeps (decision D152). typeMap and functionMap get the number in
// table of each record of source and must hold one entry per record. It
// writes nothing and returns false when table lacks the room.
func TableAppend(table []byte, source []byte, typeMap []int32, functionMap []int32) bool {
	types, functions := word(source, HeaderTypeCount), word(source, HeaderFunctionCount)
	if !tableRoomFor(table, source) || int(types) > len(typeMap) || int(functions) > len(functionMap) {
		return false
	}
	typeFirst, functionFirst := word(table, HeaderTypeCount), word(table, HeaderFunctionCount)
	fieldFirst, methodFirst, parameterFirst := word(table, HeaderFieldCount), word(table, HeaderMethodCount), word(table, HeaderParameterCount)
	for i := int32(0); i < types; i++ {
		typeMap[i] = typeFirst + i
	}
	for i := int32(0); i < functions; i++ {
		functionMap[i] = functionFirst + i
	}
	mapped := func(index int32, indexes []int32) int32 {
		if index < 0 || int(index) >= len(indexes) {
			return -1
		}
		return indexes[index]
	}
	for i := int32(0); i < types; i++ {
		var values [TypeWordCount]int32
		for field := range values {
			values[field] = record(source, HeaderTypesOffset, i, TypeWordCount, field)
		}
		values[TypeName] = nameMapped(table, source, HeaderTypeNamesOffset, HeaderTypeNameCount, values[TypeName])
		values[TypeElement] = mapped(values[TypeElement], typeMap)
		values[TypeFieldFirst] += fieldFirst
		values[TypeMethodFirst] += methodFirst
		values[TypeFunction] = mapped(values[TypeFunction], functionMap)
		typeAdd(table, values)
	}
	for i := int32(0); i < word(source, HeaderFieldCount); i++ {
		number := word(table, HeaderFieldCount)
		recordWrite(table, HeaderFieldsOffset, number, FieldWordCount, FieldName, nameMapped(table, source, HeaderMemberNamesOffset, HeaderMemberNameCount, record(source, HeaderFieldsOffset, i, FieldWordCount, FieldName)))
		recordWrite(table, HeaderFieldsOffset, number, FieldWordCount, FieldType, mapped(record(source, HeaderFieldsOffset, i, FieldWordCount, FieldType), typeMap))
		recordWrite(table, HeaderFieldsOffset, number, FieldWordCount, FieldOffset, record(source, HeaderFieldsOffset, i, FieldWordCount, FieldOffset))
		wordWrite(table, HeaderFieldCount, number+1)
	}
	for i := int32(0); i < word(source, HeaderMethodCount); i++ {
		number := word(table, HeaderMethodCount)
		recordWrite(table, HeaderMethodsOffset, number, MethodWordCount, MethodName, nameMapped(table, source, HeaderMemberNamesOffset, HeaderMemberNameCount, record(source, HeaderMethodsOffset, i, MethodWordCount, MethodName)))
		recordWrite(table, HeaderMethodsOffset, number, MethodWordCount, MethodFunction, mapped(record(source, HeaderMethodsOffset, i, MethodWordCount, MethodFunction), functionMap))
		wordWrite(table, HeaderMethodCount, number+1)
	}
	for i := int32(0); i < functions; i++ {
		number := word(table, HeaderFunctionCount)
		recordWrite(table, HeaderFunctionsOffset, number, FunctionWordCount, FunctionName, nameMapped(table, source, HeaderMemberNamesOffset, HeaderMemberNameCount, record(source, HeaderFunctionsOffset, i, FunctionWordCount, FunctionName)))
		recordWrite(table, HeaderFunctionsOffset, number, FunctionWordCount, FunctionInputCount, record(source, HeaderFunctionsOffset, i, FunctionWordCount, FunctionInputCount))
		recordWrite(table, HeaderFunctionsOffset, number, FunctionWordCount, FunctionOutputCount, record(source, HeaderFunctionsOffset, i, FunctionWordCount, FunctionOutputCount))
		recordWrite(table, HeaderFunctionsOffset, number, FunctionWordCount, FunctionParameterFirst, record(source, HeaderFunctionsOffset, i, FunctionWordCount, FunctionParameterFirst)+parameterFirst)
		wordWrite(table, HeaderFunctionCount, number+1)
	}
	for i := int32(0); i < word(source, HeaderParameterCount); i++ {
		number := word(table, HeaderParameterCount)
		wordWrite(table, int64(word(table, HeaderParametersOffset))/4+int64(number), mapped(word(source, int64(word(source, HeaderParametersOffset))/4+int64(i)), typeMap))
		wordWrite(table, HeaderParameterCount, number+1)
	}
	return true
}
