package reflecttable

import "encoding/binary"

// Type is a type record as the compiler builds it.
type Type struct {
	TypeKind      int32
	TypeSize      int32
	TypeAlignment int32
	TypeName      int32
	TypeElement   int32
	TypeLength    int32
	TypeFields    []Field
	TypeMethods   []Method
	TypeFunction  int32
}

// Field is a field record, its name a member name number.
type Field struct {
	FieldName   int32
	FieldType   int32
	FieldOffset int32
}

// Method is a method record, its name a member name number.
type Method struct {
	MethodName     int32
	MethodFunction int32
}

// Function is a function record. FunctionName is -1 for a function type.
type Function struct {
	FunctionName    int32
	FunctionInputs  []int32
	FunctionOutputs []int32
}

// Builder collects the records of a table. The compiler fills it before the
// run, so nothing here runs in the run loop.
type Builder struct {
	BuilderTypes       []Type
	BuilderFunctions   []Function
	BuilderTypeNames   NamePool
	BuilderMemberNames NamePool
}

// NamePool interns names in the order they are first seen.
type NamePool struct {
	NamePoolTexts   []string
	NamePoolNumbers map[string]int32
}

// NamePoolIntern returns the number of text, adding it when new.
func (pool *NamePool) NamePoolIntern(text string) int32 {
	if number, found := pool.NamePoolNumbers[text]; found {
		return number
	}
	if pool.NamePoolNumbers == nil {
		pool.NamePoolNumbers = map[string]int32{}
	}
	number := int32(len(pool.NamePoolTexts))
	pool.NamePoolTexts = append(pool.NamePoolTexts, text)
	pool.NamePoolNumbers[text] = number
	return number
}

// BuilderTypeAdd appends record and returns its type number.
func (builder *Builder) BuilderTypeAdd(record Type) int32 {
	builder.BuilderTypes = append(builder.BuilderTypes, record)
	return int32(len(builder.BuilderTypes) - 1)
}

// BuilderFunctionAdd appends record and returns its function number.
func (builder *Builder) BuilderFunctionAdd(record Function) int32 {
	builder.BuilderFunctions = append(builder.BuilderFunctions, record)
	return int32(len(builder.BuilderFunctions) - 1)
}

// BuilderEncode returns the table as the bytes every form reads.
func (builder *Builder) BuilderEncode() []byte {
	var fields []Field
	var methods []Method
	var parameters []int32
	types := make([]int32, 0, len(builder.BuilderTypes)*TypeWordCount)
	for _, record := range builder.BuilderTypes {
		types = append(types, record.TypeKind, record.TypeSize, record.TypeAlignment, record.TypeName,
			record.TypeElement, record.TypeLength, int32(len(fields)), int32(len(record.TypeFields)),
			int32(len(methods)), int32(len(record.TypeMethods)), record.TypeFunction)
		fields = append(fields, record.TypeFields...)
		methods = append(methods, record.TypeMethods...)
	}
	functions := make([]int32, 0, len(builder.BuilderFunctions)*FunctionWordCount)
	for _, record := range builder.BuilderFunctions {
		functions = append(functions, record.FunctionName, int32(len(record.FunctionInputs)), int32(len(record.FunctionOutputs)), int32(len(parameters)))
		parameters = append(parameters, record.FunctionInputs...)
		parameters = append(parameters, record.FunctionOutputs...)
	}

	var words []int32
	header := make([]int32, HeaderWordCount)
	section := func(headerWord int, values []int32) {
		header[headerWord] = int32(4 * (HeaderWordCount + len(words)))
		words = append(words, values...)
	}
	header[HeaderTypeCount] = int32(len(builder.BuilderTypes))
	section(HeaderTypesOffset, types)
	fieldWords := make([]int32, 0, len(fields)*FieldWordCount)
	for _, field := range fields {
		fieldWords = append(fieldWords, field.FieldName, field.FieldType, field.FieldOffset)
	}
	section(HeaderFieldsOffset, fieldWords)
	methodWords := make([]int32, 0, len(methods)*MethodWordCount)
	for _, method := range methods {
		methodWords = append(methodWords, method.MethodName, method.MethodFunction)
	}
	section(HeaderMethodsOffset, methodWords)
	header[HeaderFunctionCount] = int32(len(builder.BuilderFunctions))
	section(HeaderFunctionsOffset, functions)
	section(HeaderParametersOffset, parameters)

	var nameBytes []byte
	entries := func(pool *NamePool) []int32 {
		values := make([]int32, 0, len(pool.NamePoolTexts)*NameWordCount)
		for _, text := range pool.NamePoolTexts {
			values = append(values, int32(len(nameBytes)), int32(len(text)))
			nameBytes = append(nameBytes, text...)
		}
		return values
	}
	header[HeaderTypeNameCount] = int32(len(builder.BuilderTypeNames.NamePoolTexts))
	section(HeaderTypeNamesOffset, entries(&builder.BuilderTypeNames))
	header[HeaderMemberNameCount] = int32(len(builder.BuilderMemberNames.NamePoolTexts))
	section(HeaderMemberNamesOffset, entries(&builder.BuilderMemberNames))
	header[HeaderNameBytesOffset] = int32(4 * (HeaderWordCount + len(words)))

	table := make([]byte, 0, 4*(HeaderWordCount+len(words))+len(nameBytes))
	for _, value := range header {
		table = binary.LittleEndian.AppendUint32(table, uint32(value))
	}
	for _, value := range words {
		table = binary.LittleEndian.AppendUint32(table, uint32(value))
	}
	return append(table, nameBytes...)
}
