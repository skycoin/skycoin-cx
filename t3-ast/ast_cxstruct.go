package ast

import (
	"fmt"
	"os"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
)

type CXStructIndex int

// CXStruct is used to represent a CX struct.
type CXStruct struct {
	// Metadata
	Index   int
	Name    string         // Name of the struct
	Package CXPackageIndex // The package this struct belongs to

	// Contents
	Fields []CXTypeSignatureIndex // The fields of the struct
}

// ----------------------------------------------------------------
//                             `CXStruct` Getters

// GetField ...
func (strct *CXStruct) GetField(prgrm *CXProgram, name string) (*CXArgument, error) {
	for _, typeSignatureIdx := range strct.Fields {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0254.
		typeSignature := prgrm.GetCXTypeSignatureFromArray(typeSignatureIdx)
		if typeSignature.Name == name {
			// If type is struct
			if typeSignature.Type == TYPE_STRUCT {
				// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
				// Now an arena getter, site S0255.
				structDetails := prgrm.GetCXTypeSignatureStructFromArray(typeSignature.Meta)
				// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
				// Now a heap literal, site S0256.
				return &CXArgument{
					Name:                  typeSignature.Name,
					Type:                  types.STRUCT,
					DeclarationSpecifiers: []int{constants.DECL_STRUCT},
					StructType:            structDetails.StructType,
					Offset:                typeSignature.Offset,
				}, nil

			} else if typeSignature.Type == TYPE_POINTER_STRUCT {
				// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
				// Now an arena getter, site S0257.
				structDetails := prgrm.GetCXTypeSignatureStructFromArray(typeSignature.Meta)
				// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
				// Now a heap literal, site S0258.
				return &CXArgument{
					Name:                  typeSignature.Name,
					Type:                  types.STRUCT,
					DeclarationSpecifiers: []int{constants.DECL_STRUCT},
					StructType:            structDetails.StructType,
					Offset:                typeSignature.Offset,
				}, nil
			} else if typeSignature.Type != TYPE_CXARGUMENT_DEPRECATE {
				// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
				// Now a heap literal, site S0259.
				return &CXArgument{
					Name:                  typeSignature.Name,
					Type:                  types.Code(typeSignature.Meta),
					DeclarationSpecifiers: []int{constants.DECL_BASIC},
					StructType:            nil,
					Offset:                typeSignature.Offset,
				}, nil
			}

			// If type is cxargument deprecate
			fldIdx := typeSignature.Meta
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now a pointer into an arena, site S0260.
			return &prgrm.CXArgs[fldIdx], nil
		}
	}

	return nil, fmt.Errorf("field '%s' not found in struct '%s'", name, strct.Name)
}

// ----------------------------------------------------------------
//                     `CXStruct` Member handling

// MakeStruct ...
// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
// Now a constructor, site S0261.
func MakeStruct(name string) *CXStruct {
	// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
	// Now a heap literal, site S0262.
	return &CXStruct{
		Name: name,
	}
}

// AddField ...
func (strct *CXStruct) AddField(prgrm *CXProgram, cxArgument *CXArgument) *CXStruct {
	// Check if field already exist
	for _, typeSignatureIdx := range strct.Fields {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0263.
		typeSignature := prgrm.GetCXTypeSignatureFromArray(typeSignatureIdx)
		if typeSignature.Name == cxArgument.Name {
			fmt.Printf("%s : duplicate field", CompilationError(typeSignature.ArgDetails.FileName, typeSignature.ArgDetails.FileLine))
			os.Exit(types.CX_COMPILATION_ERROR)
		}
	}

	newCXTypeSignature := GetCXTypeSignatureRepresentationOfCXArg_ForStructs(prgrm, cxArgument)
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now an arena insert, site S0264.
	typeSignatureIdx := prgrm.AddCXTypeSignatureInArray(newCXTypeSignature)
	strct.Fields = append(strct.Fields, typeSignatureIdx)

	// TODO: Found out the effect and completely remove this.
	// Dont remove this yet as removing this gives another error on cxfx
	// even though all cx tests passed already.
	// numFlds := len(strct.Fields)
	// if numFlds > 2 {
	// 	// Pre-compiling the offset of the field.
	// 	lastTypeSignature := strct.Fields[numFlds-2]
	// 	currentTypeSignature := strct.Fields[numFlds-1]
	// 	if lastTypeSignature.Type == TYPE_CXARGUMENT_DEPRECATE && currentTypeSignature.Type == TYPE_CXARGUMENT_DEPRECATE {
	// 		lastFldIdx := lastTypeSignature.Meta
	// 		currFldIdx := currentTypeSignature.Meta
	// 		prgrm.CXArgs[currFldIdx].Offset = prgrm.CXArgs[lastFldIdx].Offset + prgrm.CXArgs[lastFldIdx].TotalSize
	// 	} else if lastTypeSignature.Type == TYPE_CXARGUMENT_DEPRECATE && currentTypeSignature.Type != TYPE_CXARGUMENT_DEPRECATE {
	// 		lastFldIdx := lastTypeSignature.Meta
	// 		currentTypeSignature.Offset = prgrm.CXArgs[lastFldIdx].Offset + prgrm.CXArgs[lastFldIdx].TotalSize
	// 	} else if lastTypeSignature.Type != TYPE_CXARGUMENT_DEPRECATE && currentTypeSignature.Type == TYPE_CXARGUMENT_DEPRECATE {
	// 		currFldIdx := currentTypeSignature.Meta
	// 		prgrm.CXArgs[currFldIdx].Offset = lastTypeSignature.Offset + lastTypeSignature.GetSize(prgrm, false)
	// 	} else if lastTypeSignature.Type != TYPE_CXARGUMENT_DEPRECATE && currentTypeSignature.Type != TYPE_CXARGUMENT_DEPRECATE {
	// 		currentTypeSignature.Offset = lastTypeSignature.Offset + lastTypeSignature.GetSize(prgrm, false)
	// 	}
	// }

	return strct
}

// AddField ...
func (strct *CXStruct) AddField_TypeSignature(prgrm *CXProgram, fieldIdx CXTypeSignatureIndex) *CXStruct {
	// Check if field already exist
	for _, typeSignatureIdx := range strct.Fields {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0265.
		typeSignature := prgrm.GetCXTypeSignatureFromArray(typeSignatureIdx)
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0266.
		field := prgrm.GetCXTypeSignatureFromArray(fieldIdx)
		if typeSignature.Name == field.Name {
			fmt.Printf("%s : duplicate field", CompilationError(typeSignature.ArgDetails.FileName, typeSignature.ArgDetails.FileLine))
			os.Exit(types.CX_COMPILATION_ERROR)
		}
	}

	strct.Fields = append(strct.Fields, fieldIdx)

	return strct
}

func (strct *CXStruct) AddField_CXAtomicOps(prgrm *CXProgram, fieldIdx CXTypeSignatureIndex) *CXStruct {
	strct.Fields = append(strct.Fields, fieldIdx)

	return strct
}

func (strct *CXStruct) AddField_Globals_CXAtomicOps(prgrm *CXProgram, cxArgIdx CXArgumentIndex) *CXStruct {
	// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
	// Now an arena getter, site S0267.
	cxArgument := prgrm.GetCXArgFromArray(cxArgIdx)

	// Check if field already exist
	for _, typeSignatureIdx := range strct.Fields {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0268.
		typeSignature := prgrm.GetCXTypeSignatureFromArray(typeSignatureIdx)
		if typeSignature.Name == cxArgument.Name {
			fmt.Printf("%s : duplicate field", CompilationError(typeSignature.ArgDetails.FileName, typeSignature.ArgDetails.FileLine))
			os.Exit(types.CX_COMPILATION_ERROR)
		}
	}

	newCXTypeSignature := GetCXTypeSignatureRepresentationOfCXArg(prgrm, cxArgument)
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now an arena insert, site S0269.
	newCXTypeSignatureIdx := prgrm.AddCXTypeSignatureInArray(newCXTypeSignature)
	strct.Fields = append(strct.Fields, newCXTypeSignatureIdx)

	return strct
}

// RemoveField ...
// func (strct *CXStruct) RemoveField(prgrm *CXProgram, fldName string) {
// 	if len(strct.Fields) > 0 {
// 		lenFlds := len(strct.Fields)
// 		for i, fldIdx := range strct.Fields {
// 			if prgrm.CXArgs[fldIdx].Name == fldName {
// 				if i == lenFlds-1 {
// 					strct.Fields = strct.Fields[:len(strct.Fields)-1]
// 				} else {
// 					strct.Fields = append(strct.Fields[:i], strct.Fields[i+1:]...)
// 				}
// 				break
// 			}
// 		}
// 	}
// }

func (strct *CXStruct) GetStructSize(prgrm *CXProgram) types.Pointer {
	var structSize types.Pointer
	for _, typeSignatureIdx := range strct.Fields {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0270.
		typeSignature := prgrm.GetCXTypeSignatureFromArray(typeSignatureIdx)
		structSize += typeSignature.GetSize(prgrm, false)
	}

	return structSize
}
