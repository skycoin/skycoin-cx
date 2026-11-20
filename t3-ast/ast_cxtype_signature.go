package ast

import (
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
)

// CXTypeSignature_TYPE enum contains CXTypeSignature types.
type CXTypeSignature_TYPE int

type CXTypeSignatureIndex int

const (
	TYPE_UNUSED CXTypeSignature_TYPE = iota
	TYPE_ATOMIC
	TYPE_POINTER_ATOMIC
	TYPE_ARRAY_ATOMIC
	TYPE_POINTER_ARRAY_ATOMIC
	TYPE_SLICE_ATOMIC
	TYPE_POINTER_SLICE_ATOMIC

	TYPE_STRUCT
	TYPE_POINTER_STRUCT
	TYPE_ARRAY_STRUCT
	TYPE_POINTER_ARRAY_STRUCT
	TYPE_SLICE_STRUCT
	TYPE_SLICE_POINTER_STRUCT

	TYPE_COMPLEX
	TYPE_POINTER_COMPLEX
	TYPE_ARRAY_COMPLEX
	TYPE_ARRAY_POINTER_COMPLEX
	TYPE_SLICE_COMPLEX
	TYPE_SLICE_POINTER_COMPLEX

	// For CXArgument usage
	// To be deprecated
	TYPE_CXARGUMENT_DEPRECATE
)

// ----------------------------------------------------------------
//                             `CXTypeSignature` Handlers

func (typeSignature *CXTypeSignature) GetSize(prgrm *CXProgram, IsForUpdateSymbolsTable bool) types.Pointer {
	switch typeSignature.Type {
	case TYPE_ATOMIC:
		return types.Code(typeSignature.Meta).Size()
	case TYPE_POINTER_ATOMIC:
		if typeSignature.IsDeref {
			return types.Code(typeSignature.Meta).Size()
		}

		return types.POINTER.Size()
	case TYPE_ARRAY_ATOMIC:
		if IsForUpdateSymbolsTable {
			return typeSignature.GetArraySize(prgrm)
		}

		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0273.
		arrDetails := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		if len(arrDetails.Indexes) > 0 {
			return types.Code(arrDetails.Meta).Size()
		}

		return typeSignature.GetArraySize(prgrm)
	case TYPE_POINTER_ARRAY_ATOMIC:
		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0274.
		arrDetails := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		if typeSignature.IsDeref && len(arrDetails.Indexes) > 0 {
			return types.Code(arrDetails.Meta).Size()
		} else if typeSignature.IsDeref {
			return typeSignature.GetArraySize(prgrm)
		}

		return types.POINTER.Size()
	case TYPE_SLICE_ATOMIC:
		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0275.
		sliceDetails := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)

		if typeSignature.IsDeref || len(sliceDetails.Indexes) > 0 {
			return types.Code(sliceDetails.Meta).Size()
		}

		return types.POINTER_SIZE
	case TYPE_POINTER_SLICE_ATOMIC:

		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0276.
		sliceDetails := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		if typeSignature.IsDeref || len(sliceDetails.Indexes) > 0 {
			return types.Code(sliceDetails.Meta).Size()
		}

		return types.POINTER.Size()
	case TYPE_STRUCT:
		// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
		// Now an arena getter, site S0277.
		structDetails := prgrm.GetCXTypeSignatureStructFromArray(typeSignature.Meta)

		lenFlds := len(structDetails.Fields)
		if lenFlds > 0 {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now a pointer into an arena, site S0278.
			return GetArgSize(prgrm, &prgrm.CXArgs[structDetails.Fields[lenFlds-1]])
		}
		return structDetails.StructType.GetStructSize(prgrm)
	case TYPE_POINTER_STRUCT:
		// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
		// Now an arena getter, site S0279.
		structDetails := prgrm.GetCXTypeSignatureStructFromArray(typeSignature.Meta)

		lenFlds := len(structDetails.Fields)
		if lenFlds > 0 {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now a pointer into an arena, site S0280.
			return GetArgSize(prgrm, &prgrm.CXArgs[structDetails.Fields[lenFlds-1]])
		}

		if typeSignature.IsDeref {
			return structDetails.StructType.GetStructSize(prgrm)
		}

		return types.POINTER.Size()
	case TYPE_ARRAY_STRUCT:
	case TYPE_POINTER_ARRAY_STRUCT:
	case TYPE_SLICE_STRUCT:
	case TYPE_SLICE_POINTER_STRUCT:

	case TYPE_COMPLEX:
	case TYPE_POINTER_COMPLEX:
	case TYPE_ARRAY_COMPLEX:
	case TYPE_ARRAY_POINTER_COMPLEX:
	case TYPE_SLICE_COMPLEX:
	case TYPE_SLICE_POINTER_COMPLEX:

	case TYPE_CXARGUMENT_DEPRECATE:
		argIdx := typeSignature.Meta
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now a pointer into an arena, site S0281.
		return GetArgSize(prgrm, &prgrm.CXArgs[argIdx])
	}

	return 0
}

func (typeSignature *CXTypeSignature) GetArrayLength(prgrm *CXProgram) types.Pointer {
	switch typeSignature.Type {
	case TYPE_ARRAY_ATOMIC:
		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0282.
		typeSignatureForArray := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		return types.TotalLength(typeSignatureForArray.Lengths)
	}

	return 0
}

func (typeSignature *CXTypeSignature) GetArraySize(prgrm *CXProgram) types.Pointer {
	switch typeSignature.Type {
	case TYPE_ARRAY_ATOMIC, TYPE_POINTER_ARRAY_ATOMIC:
		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0283.
		typeSignatureForArray := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		arrType := types.Code(typeSignatureForArray.Meta)
		return types.TotalLength(typeSignatureForArray.Lengths) * arrType.Size()
	}

	return 0
}

func (typeSignature *CXTypeSignature) GetType(prgrm *CXProgram) types.Code {
	var atomicType types.Code
	switch typeSignature.Type {
	case TYPE_ATOMIC:
		atomicType = types.Code(typeSignature.Meta)
	case TYPE_POINTER_ATOMIC:
		atomicType = types.Code(typeSignature.Meta)
	case TYPE_ARRAY_ATOMIC:
		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0284.
		arrDetails := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		atomicType = types.Code(arrDetails.Meta)
	case TYPE_POINTER_ARRAY_ATOMIC:
		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0285.
		arrDetails := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		atomicType = types.Code(arrDetails.Meta)
	case TYPE_SLICE_ATOMIC:
		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0286.
		sliceDetails := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		atomicType = types.Code(sliceDetails.Meta)
	case TYPE_POINTER_SLICE_ATOMIC:
		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0287.
		sliceDetails := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		atomicType = types.Code(sliceDetails.Meta)
	case TYPE_STRUCT:
		atomicType = types.STRUCT

		// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
		// Now an arena getter, site S0288.
		structDetails := prgrm.GetCXTypeSignatureStructFromArray(typeSignature.Meta)
		fldLen := len(structDetails.Fields)
		if fldLen > 0 {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0289.
			fld := prgrm.GetCXArgFromArray(structDetails.Fields[fldLen-1])
			fld = fld.GetAssignmentElement(prgrm)

			atomicType = fld.Type
			if fld.Type == types.POINTER {
				atomicType = fld.PointerTargetType
			}
		}
	case TYPE_POINTER_STRUCT:
		atomicType = types.STRUCT

		// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
		// Now an arena getter, site S0290.
		structDetails := prgrm.GetCXTypeSignatureStructFromArray(typeSignature.Meta)
		fldLen := len(structDetails.Fields)
		if fldLen > 0 {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0291.
			fld := prgrm.GetCXArgFromArray(structDetails.Fields[fldLen-1])
			fld = fld.GetAssignmentElement(prgrm)

			atomicType = fld.Type
			if fld.Type == types.POINTER {
				atomicType = fld.PointerTargetType
			}
		}
	default:
		panic("type is not known")
	}

	return atomicType
}

func (typeSignature *CXTypeSignature) GetTypeString(prgrm *CXProgram) string {
	var typeString string

	switch typeSignature.Type {
	case TYPE_ATOMIC:
		typeString = types.Code(typeSignature.Meta).Name()
	case TYPE_POINTER_ATOMIC:
		typeString = types.Code(typeSignature.Meta).Name()
	case TYPE_ARRAY_ATOMIC:
		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0292.
		arrDetails := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		typeString = types.Code(arrDetails.Meta).Name()
	case TYPE_POINTER_ARRAY_ATOMIC:
		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0293.
		arrDetails := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		typeString = types.Code(arrDetails.Meta).Name()
	case TYPE_SLICE_ATOMIC, TYPE_POINTER_SLICE_ATOMIC:
		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0294.
		arrDetails := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		typeString = types.Code(arrDetails.Meta).Name()
	case TYPE_STRUCT:
		// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
		// Now an arena getter, site S0295.
		structDetails := prgrm.GetCXTypeSignatureStructFromArray(typeSignature.Meta)
		fldsLen := len(structDetails.Fields)
		if fldsLen > 0 {
			fldIdx := structDetails.Fields[fldsLen-1]
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0296.
			fld := prgrm.GetCXArgFromArray(fldIdx)
			elt := fld.GetAssignmentElement(prgrm)

			if elt.StructType != nil {
				// then it's custom type
				typeString = elt.StructType.Name
			} else {
				// then it's native type
				typeString = elt.Type.Name()

				if elt.Type == types.POINTER {
					typeString = elt.PointerTargetType.Name()
				}
			}
		} else {
			typeString = structDetails.StructType.Name
		}

	case TYPE_POINTER_STRUCT:
		// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
		// Now an arena getter, site S0297.
		structDetails := prgrm.GetCXTypeSignatureStructFromArray(typeSignature.Meta)
		fldsLen := len(structDetails.Fields)
		if fldsLen > 0 {
			fldIdx := structDetails.Fields[fldsLen-1]
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0298.
			fld := prgrm.GetCXArgFromArray(fldIdx)
			elt := fld.GetAssignmentElement(prgrm)

			if elt.StructType != nil {
				// then it's custom type
				typeString = elt.StructType.Name
			} else {
				// then it's native type
				typeString = elt.Type.Name()

				if elt.Type == types.POINTER {
					typeString = elt.PointerTargetType.Name()
				}
			}
		} else {
			typeString = structDetails.StructType.Name
		}
	default:
		panic("type is not known")

	}

	return typeString
}

func (typeSignature *CXTypeSignature) GetCXArgFormat(prgrm *CXProgram) *CXArgument {
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a heap literal, site S0299.
	var arg *CXArgument = &CXArgument{}
	if typeSignature.Type == TYPE_ATOMIC {
		arg.Type = types.Code(typeSignature.Meta)
		arg.StructType = nil
		arg.Size = typeSignature.GetSize(prgrm, false)

		// TODO: this should not be needed.
		if len(arg.DeclarationSpecifiers) > 0 {
			arg.DeclarationSpecifiers = append([]int{constants.DECL_BASIC}, arg.DeclarationSpecifiers[1:]...)
		} else {
			arg.DeclarationSpecifiers = []int{constants.DECL_BASIC}
		}

	} else if typeSignature.Type == TYPE_POINTER_ATOMIC {
		arg.Type = types.POINTER
		arg.PointerTargetType = types.Code(typeSignature.Meta)
		arg.StructType = nil
		arg.Size = types.POINTER.Size()

		// TODO: this should not be needed.
		if len(arg.DeclarationSpecifiers) > 0 {
			arg.DeclarationSpecifiers = append([]int{constants.DECL_BASIC, constants.DECL_POINTER}, arg.DeclarationSpecifiers[1:]...)
		} else {
			arg.DeclarationSpecifiers = []int{constants.DECL_POINTER}
		}
	} else if typeSignature.Type == TYPE_ARRAY_ATOMIC {
		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0300.
		typeSignatureArray := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		arg.Type = types.Code(typeSignatureArray.Meta)
		arg.StructType = nil
		arg.Size = typeSignature.GetSize(prgrm, false)
		arg.Lengths = typeSignatureArray.Lengths
		arg.Indexes = typeSignatureArray.Indexes

		// TODO: this should not be needed.
		if len(arg.DeclarationSpecifiers) > 0 {
			arg.DeclarationSpecifiers = append([]int{constants.DECL_BASIC, constants.DECL_ARRAY}, arg.DeclarationSpecifiers[1:]...)
		} else {
			arg.DeclarationSpecifiers = []int{constants.DECL_BASIC}
		}

	} else if typeSignature.Type == TYPE_SLICE_ATOMIC {
		// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
		// Now an arena getter, site S0301.
		sliceDetails := prgrm.GetCXTypeSignatureArrayFromArray(typeSignature.Meta)
		arg.Type = types.Code(sliceDetails.Meta)
		arg.StructType = nil
		arg.Size = types.Code(sliceDetails.Meta).Size()
		arg.Lengths = sliceDetails.Lengths

		// TODO: this should not be needed.
		if len(arg.DeclarationSpecifiers) > 0 {
			arg.DeclarationSpecifiers = append([]int{constants.DECL_BASIC, constants.DECL_SLICE}, arg.DeclarationSpecifiers[1:]...)
		} else {
			arg.DeclarationSpecifiers = []int{constants.DECL_BASIC}
		}

		arg.DereferenceOperations = append([]int{constants.DEREF_POINTER}, arg.DereferenceOperations...)

	} else if typeSignature.Type == TYPE_STRUCT {
		arg.Type = types.STRUCT
		// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
		// Now an arena getter, site S0302.
		structDetails := prgrm.GetCXTypeSignatureStructFromArray(typeSignature.Meta)
		arg.StructType = structDetails.StructType
		// TODO: this should not be needed.
		if len(arg.DeclarationSpecifiers) > 0 {
			arg.DeclarationSpecifiers = append([]int{constants.DECL_STRUCT}, arg.DeclarationSpecifiers[1:]...)
		} else {
			arg.DeclarationSpecifiers = []int{constants.DECL_STRUCT}
		}

	} else if typeSignature.Type == TYPE_POINTER_STRUCT {
		arg.Type = types.POINTER
		// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
		// Now an arena getter, site S0303.
		structDetails := prgrm.GetCXTypeSignatureStructFromArray(typeSignature.Meta)
		arg.StructType = structDetails.StructType
		// TODO: this should not be needed.
		if len(arg.DeclarationSpecifiers) > 0 {
			arg.DeclarationSpecifiers = append([]int{constants.DECL_STRUCT, constants.DECL_POINTER}, arg.DeclarationSpecifiers[1:]...)
		} else {
			arg.DeclarationSpecifiers = []int{constants.DECL_STRUCT, constants.DECL_POINTER}
		}

	}
	arg.Offset = typeSignature.Offset

	return arg
}

// IsInnerReference determines if it is an inner reference
// for example: &slice[0] or &struct.field
func (typeSignature *CXTypeSignature) IsInnerReference(prgrm *CXProgram) bool {
	switch typeSignature.Type {
	case TYPE_CXARGUMENT_DEPRECATE:
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S0304.
		arg := prgrm.GetCXArgFromArray(CXArgumentIndex(typeSignature.Meta))

		if arg.PassBy == constants.PASSBY_REFERENCE && len(arg.Indexes) > 0 {
			for _, decl := range arg.DeclarationSpecifiers {
				if decl == constants.DECL_POINTER {
					return true
				}
			}

		}
	}

	return false
}

// Added _ForStructs for now so it will be separate from others for smooth transition.
// TODO: Remove _ForStructs
func GetCXTypeSignatureRepresentationOfCXArg_ForStructs(prgrm *CXProgram, cxArgument *CXArgument) *CXTypeSignature {
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now a value literal, site S0305.
	newCXTypeSignature := CXTypeSignature{
		Name:       cxArgument.Name,
		Package:    cxArgument.Package,
		PassBy:     cxArgument.PassBy,
		ArgDetails: cxArgument.ArgDetails,
	}

	fieldType := cxArgument.Type
	if IsTypeAtomic(cxArgument) {
		// If atomic type. i.e. i8, i16, i32, f32, etc.

		newCXTypeSignature.Type = TYPE_ATOMIC
		newCXTypeSignature.Meta = int(fieldType)

	} else if IsTypePointerAtomic(cxArgument) {
		// If pointer atomic, i.e. *i32, *f32, etc.

		newCXTypeSignature.Type = TYPE_POINTER_ATOMIC
		newCXTypeSignature.Offset = cxArgument.Offset

		newCXTypeSignature.Meta = int(cxArgument.Type)
		if cxArgument.Type == types.STR || cxArgument.StructType != nil {
			newCXTypeSignature.Meta = int(cxArgument.PointerTargetType)
		}

	} else if IsTypeArrayAtomic(cxArgument) {
		// If simple array atomic type, i.e. [5]i32, [2]f64, etc.

		newCXTypeSignature.Type = TYPE_ARRAY_ATOMIC

		arrayGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
		typeSignatureForArray := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(arrayGid)
		typeSignatureForArray.Meta = int(fieldType)
		typeSignatureForArray.Lengths = cxArgument.Lengths
		typeSignatureForArray.Indexes = cxArgument.Indexes

		newCXTypeSignature.Meta = int(objmodel.GidIndex(int64(arrayGid)))

	} else if IsTypePointerArrayAtomic(cxArgument) {
		// If pointer array atomic type, i.e. *[5]i32, *[2]f64, etc.

		newCXTypeSignature.Type = TYPE_POINTER_ARRAY_ATOMIC

		arrayGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
		typeSignatureForArray := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(arrayGid)
		typeSignatureForArray.Meta = int(fieldType)
		typeSignatureForArray.Lengths = cxArgument.Lengths
		typeSignatureForArray.Indexes = cxArgument.Indexes

		newCXTypeSignature.Meta = int(objmodel.GidIndex(int64(arrayGid)))

	} else if IsTypeSliceAtomic(cxArgument) {
		// If slice atomic type, i.e. []i32, []f64, etc.

		newCXTypeSignature.Type = TYPE_SLICE_ATOMIC

		arrayGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
		typeSignatureForArray := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(arrayGid)
		typeSignatureForArray.Meta = int(cxArgument.Type)
		typeSignatureForArray.Lengths = cxArgument.Lengths
		typeSignatureForArray.Indexes = cxArgument.Indexes

		newCXTypeSignature.Meta = int(objmodel.GidIndex(int64(arrayGid)))
	} else if IsTypePointerSliceAtomic(cxArgument) {
		// If pointer slice atomic type, i.e. *[]i32, *[]f64, etc.

		newCXTypeSignature.Type = TYPE_POINTER_SLICE_ATOMIC

		arrayGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
		typeSignatureForArray := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(arrayGid)
		typeSignatureForArray.Meta = int(cxArgument.Type)
		typeSignatureForArray.Lengths = cxArgument.Lengths
		typeSignatureForArray.Indexes = cxArgument.Indexes

		newCXTypeSignature.Meta = int(objmodel.GidIndex(int64(arrayGid)))
	} else if IsTypeStruct(cxArgument) {
		// If type is struct
		newCXTypeSignature.Type = TYPE_STRUCT

		structGid := prgrm.CxIrState.CXTypeSignature_StructMng.CXTypeSignature_StructAlloc()
		typeSignatureForStruct := prgrm.CxIrState.CXTypeSignature_StructMng.CXTypeSignature_StructGet(structGid)
		typeSignatureForStruct.Fields = cxArgument.Fields
		typeSignatureForStruct.StructType = cxArgument.StructType

		newCXTypeSignature.Meta = int(objmodel.GidIndex(int64(structGid)))
	} else if IsTypePointerStruct(cxArgument) {
		// If type is struct
		newCXTypeSignature.Type = TYPE_POINTER_STRUCT

		structGid := prgrm.CxIrState.CXTypeSignature_StructMng.CXTypeSignature_StructAlloc()
		typeSignatureForStruct := prgrm.CxIrState.CXTypeSignature_StructMng.CXTypeSignature_StructGet(structGid)
		typeSignatureForStruct.Fields = cxArgument.Fields
		typeSignatureForStruct.StructType = cxArgument.StructType

		newCXTypeSignature.Meta = int(objmodel.GidIndex(int64(structGid)))
	} else {
		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now an arena insert, site S0318.
		fldIdx := prgrm.AddCXArgInArray(cxArgument)

		// All are TYPE_CXARGUMENT_DEPRECATE for now.
		// FieldIdx or the CXArg ID is in Meta field.
		newCXTypeSignature.Type = TYPE_CXARGUMENT_DEPRECATE
		newCXTypeSignature.Meta = int(fldIdx)
	}

	newCXTypeSignature.Offset = newCXTypeSignature.GetSize(prgrm, false)

	return &newCXTypeSignature
}

func GetCXTypeSignatureRepresentationOfCXArg(prgrm *CXProgram, cxArgument *CXArgument) *CXTypeSignature {
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now a value literal, site S0319.
	newCXTypeSignature := CXTypeSignature{
		Name:       cxArgument.Name,
		Package:    cxArgument.Package,
		PassBy:     cxArgument.PassBy,
		ArgDetails: cxArgument.ArgDetails,
	}

	if IsTypeAtomic(cxArgument) {
		// If atomic type. i.e. i8, i16, i32, f32, etc.
		newCXTypeSignature.Type = TYPE_ATOMIC
		newCXTypeSignature.Meta = int(cxArgument.Type)
		newCXTypeSignature.Offset = cxArgument.Offset
	} else if IsTypePointerAtomic(cxArgument) {
		// If pointer atomic, i.e. *i32, *f32, etc.
		newCXTypeSignature.Type = TYPE_POINTER_ATOMIC
		newCXTypeSignature.Offset = cxArgument.Offset

		newCXTypeSignature.Meta = int(cxArgument.Type)
		if cxArgument.Type == types.STR || cxArgument.StructType != nil {
			newCXTypeSignature.Meta = int(cxArgument.PointerTargetType)
		}
	} else if IsTypeArrayAtomic(cxArgument) {
		// If simple array atomic type, i.e. [5]i32, [2]f64, etc.
		newCXTypeSignature.Type = TYPE_ARRAY_ATOMIC

		arrayGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
		typeSignatureForArray := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(arrayGid)
		typeSignatureForArray.Meta = int(cxArgument.Type)
		typeSignatureForArray.Lengths = cxArgument.Lengths
		typeSignatureForArray.Indexes = cxArgument.Indexes

		newCXTypeSignature.Meta = int(objmodel.GidIndex(int64(arrayGid)))
	} else if IsTypePointerArrayAtomic(cxArgument) {
		// If pointer array atomic type, i.e. *[5]i32, *[2]f64, etc.

		newCXTypeSignature.Type = TYPE_POINTER_ARRAY_ATOMIC

		arrayGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
		typeSignatureForArray := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(arrayGid)
		typeSignatureForArray.Meta = int(cxArgument.Type)
		typeSignatureForArray.Lengths = cxArgument.Lengths
		typeSignatureForArray.Indexes = cxArgument.Indexes

		newCXTypeSignature.Meta = int(objmodel.GidIndex(int64(arrayGid)))
	} else if IsTypeSliceAtomic(cxArgument) {
		// If slice atomic type, i.e. []i32, []f64, etc.

		newCXTypeSignature.Type = TYPE_SLICE_ATOMIC

		arrayGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
		typeSignatureForArray := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(arrayGid)
		typeSignatureForArray.Meta = int(cxArgument.Type)
		typeSignatureForArray.Lengths = cxArgument.Lengths
		typeSignatureForArray.Indexes = cxArgument.Indexes

		newCXTypeSignature.Meta = int(objmodel.GidIndex(int64(arrayGid)))
	} else if IsTypePointerSliceAtomic(cxArgument) {
		// If pointer slice atomic type, i.e. *[]i32, *[]f64, etc.

		newCXTypeSignature.Type = TYPE_POINTER_SLICE_ATOMIC

		arrayGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
		typeSignatureForArray := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(arrayGid)
		typeSignatureForArray.Meta = int(cxArgument.Type)
		typeSignatureForArray.Lengths = cxArgument.Lengths
		typeSignatureForArray.Indexes = cxArgument.Indexes

		newCXTypeSignature.Meta = int(objmodel.GidIndex(int64(arrayGid)))
	} else if IsTypeStruct(cxArgument) {
		// If type is struct
		newCXTypeSignature.Type = TYPE_STRUCT

		structGid := prgrm.CxIrState.CXTypeSignature_StructMng.CXTypeSignature_StructAlloc()
		typeSignatureForStruct := prgrm.CxIrState.CXTypeSignature_StructMng.CXTypeSignature_StructGet(structGid)
		typeSignatureForStruct.Fields = cxArgument.Fields
		typeSignatureForStruct.StructType = cxArgument.StructType

		newCXTypeSignature.Meta = int(objmodel.GidIndex(int64(structGid)))
	} else if IsTypePointerStruct(cxArgument) {
		// If type is struct
		newCXTypeSignature.Type = TYPE_POINTER_STRUCT

		structGid := prgrm.CxIrState.CXTypeSignature_StructMng.CXTypeSignature_StructAlloc()
		typeSignatureForStruct := prgrm.CxIrState.CXTypeSignature_StructMng.CXTypeSignature_StructGet(structGid)
		typeSignatureForStruct.Fields = cxArgument.Fields
		typeSignatureForStruct.StructType = cxArgument.StructType

		newCXTypeSignature.Meta = int(objmodel.GidIndex(int64(structGid)))
	} else {
		// TYPE_CXARGUMENT_DEPRECATE
		// FieldIdx or the CXArg ID is in Meta field.
		cxArgumentIndex := cxArgument.Index
		if cxArgument.Index == -1 || cxArgument.Index == 0 {
			// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
			// Now an arena insert, site S0332.
			cxArgumentIndex = int(prgrm.AddCXArgInArray(cxArgument))
		}
		newCXTypeSignature.Type = TYPE_CXARGUMENT_DEPRECATE
		newCXTypeSignature.Meta = cxArgumentIndex
		newCXTypeSignature.Offset = cxArgument.Offset
	}

	return &newCXTypeSignature
}
