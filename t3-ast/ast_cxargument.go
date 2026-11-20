package ast

import (
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t2-globals" // thin: only DBG_GOLANG_STACK_TRACE, FoundCompileErrors, HeapOffset; core globals, keep
)

type CXArgumentIndex int

func (arg CXArgument) IsPointer() bool {
	return arg.Type == types.POINTER
}

func (arg CXArgument) IsString() bool {
	return arg.PointerTargetType == types.STR || arg.Type == types.STR
}

func (arg CXArgument) IsStruct() bool {
	return arg.StructType != nil
}

func (arg CXArgument) IsSlice() bool {
	if len(arg.Lengths) > 0 {
		for _, val := range arg.Lengths {
			if val != 0 {
				return false
			}
		}

		return true
	}

	return false
}

/*
	FileName              string
- filename and line number
- can be moved to CX AST annotations (comments to be skipped or map)

	FileLine
*/

/*

Note: PAssBy is not used too many place
Note: Low priority for deprecation
- isnt this same as "pointer"
*/

// ----------------------------------------------------------------
//                             `CXArgument` Getters

// GetAssignmentElement ...
func (arg *CXArgument) GetAssignmentElement(prgrm *CXProgram) *CXArgument {
	if len(arg.Fields) > 0 {
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S0068.
		return prgrm.GetCXArgFromArray(arg.Fields[len(arg.Fields)-1]).GetAssignmentElement(prgrm)
	}
	return arg

}

// GetType ...
func (arg *CXArgument) GetType(prgrm *CXProgram) types.Code {
	fieldCount := len(arg.Fields)
	if fieldCount > 0 {
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S0069.
		return prgrm.GetCXArgFromArray(arg.Fields[fieldCount-1]).GetType(prgrm)
	}

	if arg.Type == types.POINTER {
		return arg.PointerTargetType
	}
	return arg.Type
}

func (arg *CXArgument) GetSize() types.Pointer {
	if arg.Type != types.I8 {
		return arg.Type.Size()
	}

	return arg.Size
}

// ----------------------------------------------------------------
//                     `CXArgument` Member handling

// SetPackage sets CX package `pkg` of CX argument `arg`.
func (arg *CXArgument) SetPackage(pkg *CXPackage) *CXArgument {
	arg.Package = CXPackageIndex(pkg.Index)
	return arg
}

// SetType ...
func (arg *CXArgument) SetType(typeCode types.Code) *CXArgument {
	arg.Type = typeCode
	size := typeCode.Size()
	arg.Size = size
	arg.DeclarationSpecifiers = append(arg.DeclarationSpecifiers, constants.DECL_BASIC)
	return arg
}

// AddInput adds input parameters to `arg` in case arg is of type `TYPE_FUNC`.
func (arg *CXArgument) AddInput(prgrm *CXProgram, inp *CXArgument) *CXArgument {
	if inp.Package == -1 {
		inp.Package = arg.Package
	}

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now an arena insert, site S0070.
	inpIdx := prgrm.AddCXArgInArray(inp)
	arg.Inputs = append(arg.Inputs, inpIdx)

	return arg
}

// AddOutput adds output parameters to `arg` in case arg is of type `TYPE_FUNC`.
func (arg *CXArgument) AddOutput(prgrm *CXProgram, out *CXArgument) *CXArgument {
	if out.Package == -1 {
		out.Package = arg.Package
	}

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now an arena insert, site S0071.
	outIdx := prgrm.AddCXArgInArray(out)
	arg.Outputs = append(arg.Outputs, outIdx)

	return arg
}

// MakePointer takes an already defined `CXArgument` and turns it into a pointer.
// Only used once, deprecate
// TODO: only used by HTTP, create a better module system
func MakePointer(arg *CXArgument) *CXArgument {
	arg.DeclarationSpecifiers = append(arg.DeclarationSpecifiers, constants.DECL_POINTER)
	arg.PointerTargetType = arg.Type
	arg.Type = types.POINTER
	arg.Size = types.POINTER_SIZE

	return arg
}

// MakeStructParameter helper for creating a struct parameter. It creates a
// `CXArgument` named `argName`, that represents a structure instane of
// `strctName`, from package `pkgName`.
// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
// Now a constructor, site S0072.
func MakeStructParameter(prgrm *CXProgram, pkgName, strctName, argName string) *CXArgument {
	pkg, err := prgrm.GetPackage(pkgName)
	if err != nil {
		panic(err)
	}

	strct, err := pkg.GetStruct(prgrm, strctName)
	if err != nil {
		panic(err)
	}

	// Natives are shared by every program, so this argument belongs to none.
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S0073.
	arg := MakeArgument(nil, argName, "", -1).SetType(types.STRUCT)
	// arg.DeclarationSpecifiers = append(arg.DeclarationSpecifiers, constants.DECL_STRUCT)
	arg.Size = strct.GetStructSize(prgrm)
	arg.StructType = strct

	return arg
}

// Slice Helper function for creating parameters for standard library operators.
// The current standard library only uses basic types and slices. If more options are needed, modify this function
// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
// Now a constructor, site S0074.
func Slice(typeCode types.Code) *CXArgument {
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S0075.
	arg := Param(typeCode)
	arg.DeclarationSpecifiers = append(arg.DeclarationSpecifiers, constants.DECL_SLICE)
	return arg
}

// Func Helper function for creating function parameters for standard library operators.
// The current standard library only uses basic types and slices. If more options are needed, modify this function
// func Func(pkg *CXPackage, inputs []*CXArgument, outputs []*CXArgument) *CXArgument {
// 	arg := Param(types.FUNC)
// 	arg.Package = CXPackageIndex(pkg.Index)
// 	arg.Inputs = inputs
// 	arg.Outputs = outputs
// 	return arg
// }

// Param ...
// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
// Now a constructor, site S0076.
func Param(typeCode types.Code) *CXArgument {
	// Natives are shared by every program, so this argument belongs to none.
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S0077.
	arg := MakeArgument(nil, "", "", -1).SetType(typeCode)
	// arg.IsLocalDeclaration = true
	return arg
}

// MakeArgument makes an argument of prgrm. The natives are shared by every
// program, so Param and MakeStructParameter pass nil and the argument gets
// no debug record.
// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
// Now a constructor, site S0078.
func MakeArgument(prgrm *CXProgram, name string, fileName string, fileLine int) *CXArgument {
	argDetailsGid := argumentDebugAlloc(prgrm, fileName, fileLine)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a heap literal, site S0079.
	return &CXArgument{
		Name:       name,
		Package:    -1,
		ArgDetails: argDetailsGid,
	}

}

// MakeField ...
// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
// Now a constructor, site S0081.
func MakeField(prgrm *CXProgram, name string, typeCode types.Code, fileName string, fileLine int) *CXArgument {
	argDetailsGid := argumentDebugAlloc(prgrm, fileName, fileLine)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a heap literal, site S0082.
	return &CXArgument{
		Name:       name,
		Package:    -1,
		ArgDetails: argDetailsGid,
		Type:       typeCode,
	}
}

// MakeGlobal ...
// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
// Now a constructor, site S0084.
func MakeGlobal(prgrm *CXProgram, name string, typeCode types.Code, fileName string, fileLine int) *CXArgument {
	argDetailsGid := argumentDebugAlloc(prgrm, fileName, fileLine)
	size := typeCode.Size()
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a heap literal, site S0085.
	global := &CXArgument{
		Name:       name,
		Package:    -1,
		ArgDetails: argDetailsGid,
		Type:       typeCode,
		Size:       size,
		Offset:     globals.HeapOffset,
	}
	globals.HeapOffset += size
	return global
}

// ------------------------------------------------------------------------------------------
//           Special functions to determine its type (atomic, array etomic, etc)

func IsTypeAtomic(arg *CXArgument) bool {
	// TODO: implement including types.IDENTIFIER
	// return (arg.Type.IsPrimitive() || arg.Type == types.IDENTIFIER) && !arg.IsSlice() && len(arg.Lengths) == 0 && len(arg.Fields) == 0 && len(arg.DereferenceOperations) == 0 && (len(arg.DeclarationSpecifiers) == 0 || (len(arg.DeclarationSpecifiers) == 1 && arg.DeclarationSpecifiers[0] == constants.DECL_BASIC))

	return arg.Type.IsPrimitive() && !arg.IsSlice() && len(arg.Lengths) == 0 && len(arg.Fields) == 0 && len(arg.DereferenceOperations) == 0 && (len(arg.DeclarationSpecifiers) == 0 || (len(arg.DeclarationSpecifiers) == 1 && arg.DeclarationSpecifiers[0] == constants.DECL_BASIC))
}

func IsTypePointerAtomic(arg *CXArgument) bool {
	return arg.Type.IsPrimitive() && arg.PointerTargetType == 0 && arg.StructType == nil && !arg.IsSlice() && len(arg.Lengths) == 0 && len(arg.Fields) == 0 && len(arg.DereferenceOperations) == 0 && (len(arg.DeclarationSpecifiers) == 2 && arg.DeclarationSpecifiers[0] == constants.DECL_BASIC && arg.DeclarationSpecifiers[1] == constants.DECL_POINTER)
}

func IsTypeArrayAtomic(arg *CXArgument) bool {
	isThereDeclPointer := false
	for _, decl := range arg.DeclarationSpecifiers {
		if decl == constants.DECL_POINTER {
			isThereDeclPointer = true
		}
	}

	return !isThereDeclPointer && arg.Type.IsPrimitive() && !arg.IsSlice() && len(arg.Lengths) > 0 && len(arg.Fields) == 0 && len(arg.DereferenceOperations) == 0 && (len(arg.DeclarationSpecifiers) >= 2 && arg.DeclarationSpecifiers[0] == constants.DECL_BASIC && arg.DeclarationSpecifiers[1] == constants.DECL_ARRAY)
}

func IsTypePointerArrayAtomic(arg *CXArgument) bool {
	isThereDeclPointer := false
	for _, decl := range arg.DeclarationSpecifiers {
		if decl == constants.DECL_POINTER {
			isThereDeclPointer = true
		}
	}

	return isThereDeclPointer && arg.Type.IsPrimitive() && !arg.IsSlice() && len(arg.Lengths) > 0 && len(arg.Fields) == 0 && len(arg.DereferenceOperations) == 0 && (len(arg.DeclarationSpecifiers) >= 2 && arg.DeclarationSpecifiers[0] == constants.DECL_BASIC && arg.DeclarationSpecifiers[1] == constants.DECL_ARRAY)
}

func IsTypeSliceAtomic(arg *CXArgument) bool {
	isThereDeclPointer := false
	for _, decl := range arg.DeclarationSpecifiers {
		if decl == constants.DECL_POINTER {
			isThereDeclPointer = true
		}
	}

	return !isThereDeclPointer && arg.Type.IsPrimitive() && arg.IsSlice() && len(arg.Lengths) > 0 && len(arg.Fields) == 0 && len(arg.DereferenceOperations) == 0 && (len(arg.DeclarationSpecifiers) >= 2 && arg.DeclarationSpecifiers[0] == constants.DECL_BASIC && arg.DeclarationSpecifiers[1] == constants.DECL_SLICE)
}

func IsTypePointerSliceAtomic(arg *CXArgument) bool {
	isThereDeclPointer := false
	for _, decl := range arg.DeclarationSpecifiers {
		if decl == constants.DECL_POINTER {
			isThereDeclPointer = true
		}
	}

	return isThereDeclPointer && arg.Type.IsPrimitive() && arg.IsSlice() && len(arg.Lengths) > 0 && len(arg.Fields) == 0 && len(arg.DereferenceOperations) == 0 && (len(arg.DeclarationSpecifiers) >= 2 && arg.DeclarationSpecifiers[0] == constants.DECL_BASIC && arg.DeclarationSpecifiers[1] == constants.DECL_SLICE)
}

func IsTypeStruct(arg *CXArgument) bool {
	isThereDeclPointer := false
	IsThereDeclSlice := false
	for _, decl := range arg.DeclarationSpecifiers {
		if decl == constants.DECL_POINTER {
			isThereDeclPointer = true
		}

		if decl == constants.DECL_SLICE {
			IsThereDeclSlice = true
		}
	}

	return !isThereDeclPointer && !IsThereDeclSlice && !arg.IsSlice() && len(arg.Lengths) == 0 && arg.StructType != nil && len(arg.DeclarationSpecifiers) >= 1 && arg.DeclarationSpecifiers[0] == constants.DECL_STRUCT
}

func IsTypePointerStruct(arg *CXArgument) bool {
	isThereDeclPointer := false
	IsThereDeclSlice := false
	for _, decl := range arg.DeclarationSpecifiers {
		if decl == constants.DECL_POINTER {
			isThereDeclPointer = true
		}

		if decl == constants.DECL_SLICE {
			IsThereDeclSlice = true
		}
	}

	return isThereDeclPointer && !IsThereDeclSlice && !arg.IsSlice() && len(arg.Lengths) == 0 && arg.StructType != nil && len(arg.DeclarationSpecifiers) >= 1 && arg.DeclarationSpecifiers[0] == constants.DECL_STRUCT
}

// argumentDebugAlloc allocates the debug record of an argument in prgrm's
// CXArgumentDebugMng. A nil prgrm, for a native, gets CXArgumentDebugGidNil.
func argumentDebugAlloc(prgrm *CXProgram, fileName string, fileLine int) CXArgumentDebugGid {
	if prgrm == nil {
		return CXArgumentDebugGidNil
	}
	manager := &prgrm.CxIrState.CXArgumentDebugMng
	gid := manager.CXArgumentDebugAlloc()
	details := manager.CXArgumentDebugGet(gid)
	details.FileName = fileName
	details.FileLine = fileLine
	return gid
}

// CXArgumentDebugFileLine returns the file name and line that gid names in
// prgrm. A native parameter has none and gets "" and -1, as Param gave it.
func CXArgumentDebugFileLine(prgrm *CXProgram, gid CXArgumentDebugGid) (string, int) {
	if gid == CXArgumentDebugGidNil {
		return "", -1
	}
	details := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugGet(gid)
	return details.FileName, details.FileLine
}
