package ast

import (
	"sort"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	reflecttable "github.com/skycoin/skycoin-cx/t2-cx-library-reflecttable"
)

// ReflectTable is the type table of the reflect library for one program
// (decision D150). ReflectTableTypes gives the type number of the value
// each reflect.CxTypeOfValueGet call passes, by the signature of its input.
type ReflectTable struct {
	ReflectTableBytes []byte
	ReflectTableTypes map[CXTypeSignatureIndex]int32
}

// reflectTableComposite names an array, slice or pointer type by its kind,
// element and length.
type reflectTableComposite struct {
	reflectTableCompositeKind    int32
	reflectTableCompositeElement int32
	reflectTableCompositeLength  int32
}

// reflectTableBuilder holds the type numbers given so far, so each type has
// one record.
type reflectTableBuilder struct {
	program    *CXProgram
	builder    reflecttable.Builder
	scalars    map[valuetypes.Code]int32
	structs    map[*CXStruct]int32
	composites map[reflectTableComposite]int32
	functions  map[*CXFunction]int32
}

// ReflectTableBuild builds the type table of prgrm from the values its
// reflect.CxTypeOfValueGet calls pass and every type they reach, in function
// and expression order, so every form builds the same bytes. It runs before
// the run, never in the run loop.
func ReflectTableBuild(prgrm *CXProgram) ReflectTable {
	rb := &reflectTableBuilder{
		program:    prgrm,
		scalars:    map[valuetypes.Code]int32{},
		structs:    map[*CXStruct]int32{},
		composites: map[reflectTableComposite]int32{},
		functions:  map[*CXFunction]int32{},
	}
	table := ReflectTable{ReflectTableTypes: map[CXTypeSignatureIndex]int32{}}
	for index := 0; index < prgrm.CXProgramFunctionSlotCount(); index++ {
		fn := prgrm.CXProgramFunctionSlotLive(index)
		if fn == nil || fn.BuiltIn() {
			continue
		}
		for line := range fn.Expressions {
			expr := &fn.Expressions[line]
			if expr.Type != CXOperationTypeAtomicOperator {
				continue
			}
			cxAtomicOp, err := prgrm.CXAtomicOpGet(expr.Index)
			if err != nil {
				continue
			}
			operator := prgrm.CxIrState.CxIrStateFunctionGetOrNil(cxAtomicOp.Operator)
			if operator == nil || operator.AtomicOPCode != opcodetable.OpcodeReflectCxTypeOfValueGet {
				continue
			}
			inputs := cxAtomicOp.InputsGet(prgrm)
			if len(inputs) != 1 {
				continue
			}
			table.ReflectTableTypes[inputs[0]] = rb.valueType(prgrm.CXTypeSignatureFromArrayGet(inputs[0]))
		}
	}
	table.ReflectTableBytes = rb.builder.BuilderEncode()
	return table
}

// valueType returns the type of the value an expression operand names,
// after its field access, indexing and dereferences.
func (rb *reflectTableBuilder) valueType(signature *CXTypeSignature) int32 {
	prgrm := rb.program
	if signature.Type == TypeCXArgumentDeprecate {
		arg := prgrm.CXArgFromArrayGet(CXArgumentIndex(signature.Meta)).AssignmentElementGet(prgrm)
		var specs []int
		lengths := arg.Lengths
		for _, spec := range arg.DeclarationSpecifiers {
			switch spec {
			case opcodetable.DeclarationDeref, opcodetable.DeclarationIndexing:
				if len(specs) == 0 {
					continue
				}
				top := specs[len(specs)-1]
				specs = specs[:len(specs)-1]
				if top == opcodetable.DeclarationArray && len(lengths) > 0 {
					lengths = lengths[1:]
				} else if top == opcodetable.DeclarationSlice && len(lengths) > reflectArrayCount(specs) {
					lengths = lengths[1:]
				}
			default:
				specs = append(specs, spec)
			}
		}
		typ := arg.Type
		if typ == valuetypes.POINTER {
			typ = arg.PointerTargetType
		}
		return rb.declarationType(specs, typ, arg.StructType, lengths)
	}
	if signature.Type == TypeStruct || signature.Type == TypePointerStruct {
		if fields := prgrm.CXTypeSignatureStructFromArrayGet(signature.Meta).Fields; len(fields) > 0 {
			return rb.valueType(&CXTypeSignature{Type: TypeCXArgumentDeprecate, Meta: int(fields[len(fields)-1])})
		}
	}
	specs, typ, strct, lengths := SignatureGCDeclaration(prgrm, signature)
	switch signature.Type {
	case TypePointerAtomic, TypePointerStruct:
		if signature.Deref && len(specs) > 0 {
			specs = specs[:len(specs)-1]
		}
	case TypeArrayAtomic, TypeSliceAtomic:
		if indexes := len(prgrm.CXTypeSignatureArrayFromArrayGet(signature.Meta).Indexes); indexes > 0 && indexes <= len(lengths) {
			specs = specs[:len(specs)-indexes]
			lengths = lengths[indexes:]
		}
	}
	return rb.declarationType(specs, typ, strct, lengths)
}

// declarationType returns the type of declaration specifiers specs,
// innermost first, as GCTypeOf reads them.
func (rb *reflectTableBuilder) declarationType(specs []int, typ valuetypes.Code, strct *CXStruct, lengths []valuetypes.Pointer) int32 {
	if len(specs) == 0 {
		return rb.baseType(typ, strct)
	}
	inner := specs[:len(specs)-1]
	switch specs[len(specs)-1] {
	case opcodetable.DeclarationPointer:
		return rb.compositeType(reflecttable.KindPointer, rb.declarationType(inner, typ, strct, lengths), 0)
	case opcodetable.DeclarationSlice:
		if len(lengths) > reflectArrayCount(inner) {
			lengths = lengths[1:]
		}
		return rb.compositeType(reflecttable.KindSlice, rb.declarationType(inner, typ, strct, lengths), 0)
	case opcodetable.DeclarationArray:
		if len(lengths) == 0 {
			return rb.declarationType(inner, typ, strct, lengths)
		}
		return rb.compositeType(reflecttable.KindArray, rb.declarationType(inner, typ, strct, lengths[1:]), int32(lengths[0]))
	}
	return rb.baseType(typ, strct)
}

// reflectArrayCount returns how many arrays specs has.
func reflectArrayCount(specs []int) int {
	return gcArrayCount(specs)
}

func (rb *reflectTableBuilder) baseType(typ valuetypes.Code, strct *CXStruct) int32 {
	if strct != nil {
		return rb.structType(strct)
	}
	return rb.scalarType(typ)
}

// scalarType returns the type of an atomic value. A str is a pointer to its
// bytes, and a func value names no function.
func (rb *reflectTableBuilder) scalarType(typ valuetypes.Code) int32 {
	if index, found := rb.scalars[typ]; found {
		return index
	}
	kind, size := reflecttable.KindScalar, int32(0)
	if typ > valuetypes.UNUSED && typ < valuetypes.COUNT && typ.Size() != valuetypes.InvalidPointer {
		size = int32(typ.Size())
	}
	if typ == valuetypes.FUNC {
		kind = reflecttable.KindFunction
	}
	name := int32(-1)
	if typ > valuetypes.UNUSED && typ < valuetypes.COUNT {
		name = rb.builder.BuilderTypeNames.NamePoolIntern(typ.Name())
	}
	index := rb.builder.BuilderTypeAdd(reflecttable.Type{
		TypeKind: kind, TypeSize: size, TypeAlignment: max(size, 1), TypeName: name,
		TypeElement: -1, TypeLength: -1, TypeFunction: -1,
	})
	rb.scalars[typ] = index
	return index
}

func (rb *reflectTableBuilder) compositeType(kind int32, element int32, length int32) int32 {
	key := reflectTableComposite{kind, element, length}
	if index, found := rb.composites[key]; found {
		return index
	}
	elementRecord := rb.builder.BuilderTypes[element]
	size, alignment := int32(valuetypes.PointerSize), int32(valuetypes.PointerSize)
	if kind == reflecttable.KindArray {
		size, alignment = length*elementRecord.TypeSize, elementRecord.TypeAlignment
	} else {
		length = -1
	}
	index := rb.builder.BuilderTypeAdd(reflecttable.Type{
		TypeKind: kind, TypeSize: size, TypeAlignment: alignment, TypeName: -1,
		TypeElement: element, TypeLength: length, TypeFunction: -1,
	})
	rb.composites[key] = index
	return index
}

// structType returns the type of strct. Its number is given before its
// fields, so a field that points back to strct finds it.
func (rb *reflectTableBuilder) structType(strct *CXStruct) int32 {
	if index, found := rb.structs[strct]; found {
		return index
	}
	prgrm := rb.program
	index := rb.builder.BuilderTypeAdd(reflecttable.Type{
		TypeKind: reflecttable.KindStruct, TypeSize: int32(strct.StructSizeGet(prgrm)), TypeAlignment: 1,
		TypeName:    rb.builder.BuilderTypeNames.NamePoolIntern(prgrm.CXProgramNameText(strct.Name)),
		TypeElement: -1, TypeLength: -1, TypeFunction: -1,
	})
	rb.structs[strct] = index
	fields := make([]reflecttable.Field, len(strct.Fields))
	alignment := int32(1)
	for ordinal, fieldIndex := range strct.Fields {
		signature := prgrm.CXTypeSignatureFromArrayGet(fieldIndex)
		specs, typ, fieldStruct, lengths := SignatureGCDeclaration(prgrm, signature)
		fieldType := rb.declarationType(specs, typ, fieldStruct, lengths)
		alignment = max(alignment, rb.builder.BuilderTypes[fieldType].TypeAlignment)
		fields[ordinal] = reflecttable.Field{
			FieldName:   rb.builder.BuilderMemberNames.NamePoolIntern(prgrm.CXProgramNameText(signature.Name)),
			FieldType:   fieldType,
			FieldOffset: int32(StructFieldOffset(prgrm, strct, ordinal)),
		}
	}
	methods := rb.structMethods(strct)
	record := &rb.builder.BuilderTypes[index]
	record.TypeAlignment = alignment
	record.TypeFields = fields
	record.TypeMethods = methods
	return index
}

// structMethods returns the methods of strct in name order.
func (rb *reflectTableBuilder) structMethods(strct *CXStruct) []reflecttable.Method {
	prgrm := rb.program
	pkg, err := prgrm.PackageFromArrayGet(strct.Package)
	if err != nil {
		return nil
	}
	type namedMethod struct {
		name     string
		function *CXFunction
	}
	var named []namedMethod
	for key, gid := range pkg.Methods {
		if key.CXPackageMethodKeyStruct != strct.Name {
			continue
		}
		if fn := prgrm.CxIrState.CxIrStateFunctionGetOrNil(gid); fn != nil {
			named = append(named, namedMethod{prgrm.CXProgramNameText(key.CXPackageMethodKeyMethod), fn})
		}
	}
	sort.Slice(named, func(i, j int) bool { return named[i].name < named[j].name })
	methods := make([]reflecttable.Method, len(named))
	for i, method := range named {
		methods[i] = reflecttable.Method{
			MethodName:     rb.builder.BuilderMemberNames.NamePoolIntern(method.name),
			MethodFunction: rb.functionRecord(method.function),
		}
	}
	return methods
}

// functionRecord returns the function record of fn, its inputs with the
// receiver of a method first.
func (rb *reflectTableBuilder) functionRecord(fn *CXFunction) int32 {
	if index, found := rb.functions[fn]; found {
		return index
	}
	prgrm := rb.program
	index := rb.builder.BuilderFunctionAdd(reflecttable.Function{
		FunctionName: rb.builder.BuilderMemberNames.NamePoolIntern(prgrm.CXProgramNameText(fn.Name)),
	})
	rb.functions[fn] = index
	parameters := func(signatures []CXTypeSignatureIndex) []int32 {
		types := make([]int32, len(signatures))
		for i, signatureIndex := range signatures {
			specs, typ, strct, lengths := SignatureGCDeclaration(prgrm, prgrm.CXTypeSignatureFromArrayGet(signatureIndex))
			types[i] = rb.declarationType(specs, typ, strct, lengths)
		}
		return types
	}
	inputs := parameters(fn.InputsGet(prgrm))
	outputs := parameters(fn.OutputsGet(prgrm))
	rb.builder.BuilderFunctions[index].FunctionInputs = inputs
	rb.builder.BuilderFunctions[index].FunctionOutputs = outputs
	return index
}
