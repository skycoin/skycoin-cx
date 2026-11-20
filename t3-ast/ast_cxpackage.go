package ast

import (
	"errors"
	"fmt"
)

type CXPackageIndex int

// CXPackage is used to represent a CX package.
type CXPackage struct {
	// Metadata
	Name  string // Name of the package
	Index int    // Index of package inside the CXPackage array

	// Contents
	Imports   map[string]CXPackageIndex  // imported packages
	Functions map[string]CXFunctionIndex // declared functions in this package
	Structs   map[string]CXStructIndex   // declared structs in this package
	// OBJMODEL: CXStructGid, phase 5 #337.
	// Now a pointer at rest, site S0157.
	Globals *CXStruct // declared global variables in this package

	// Used by the REPL and cxgo
	CurrentFunction CXFunctionIndex
}

// Only Used by Affordances in op_aff.go
func (pkg *CXPackage) GetFunction(prgrm *CXProgram, fnName string) (*CXFunction, error) {
	if fnIdx, ok := pkg.Functions[fnName]; ok {
		// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
		// Now an arena getter, site S0158.
		fn := prgrm.GetFunctionFromArray(fnIdx)

		return fn, nil
	}

	// now checking in imported packages
	for _, impIdx := range pkg.Imports {
		// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
		// Now an arena getter, site S0159.
		imp, err := prgrm.GetPackageFromArray(impIdx)
		if err != nil {
			panic(err)
		}

		if fnIdx, ok := imp.Functions[fnName]; ok {
			// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
			// Now an arena getter, site S0160.
			fn := prgrm.GetFunctionFromArray(fnIdx)

			return fn, nil
		}
	}

	return nil, fmt.Errorf("function '%s' not found in package '%s' or its imports", fnName, pkg.Name)
}

// GetImport ...
func (pkg *CXPackage) GetImport(prgrm *CXProgram, impName string) (*CXPackage, error) {
	if _, ok := pkg.Imports[impName]; !ok {
		return nil, fmt.Errorf("package '%s' not imported", impName)
	}

	// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
	// Now an arena getter, site S0161.
	imp, err := prgrm.GetPackageFromArray(pkg.Imports[impName])
	if err != nil {
		panic(err)
	}
	return imp, nil
}

// GetMethod ...
func (pkg *CXPackage) GetMethod(prgrm *CXProgram, fnName string, receiverType string) (CXFunctionIndex, error) {
	if fnIdx, ok := pkg.Functions[fnName]; ok {
		// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
		// Now an arena getter, site S0162.
		fn := prgrm.GetFunctionFromArray(fnIdx)
		fnInputs := fn.GetInputs(prgrm)
		fnInputTypeSigIdx := fnInputs[0]
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0163.
		fnInputTypeSig := prgrm.GetCXTypeSignatureFromArray(fnInputTypeSigIdx)

		var fnInputStructType *CXStruct
		if fnInputTypeSig.Type == TYPE_CXARGUMENT_DEPRECATE {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0164.
			fnInput := prgrm.GetCXArgFromArray(CXArgumentIndex(fnInputTypeSig.Meta))
			fnInputStructType = fnInput.StructType
		} else if fnInputTypeSig.Type == TYPE_STRUCT {
			// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
			// Now an arena getter, site S0165.
			structDetails := prgrm.GetCXTypeSignatureStructFromArray(fnInputTypeSig.Meta)
			fnInputStructType = structDetails.StructType
		} else if fnInputTypeSig.Type == TYPE_POINTER_STRUCT {
			// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
			// Now an arena getter, site S0166.
			structDetails := prgrm.GetCXTypeSignatureStructFromArray(fnInputTypeSig.Meta)
			fnInputStructType = structDetails.StructType
		} else {
			panic("type is not known")
		}

		if len(fnInputs) > 0 && fnInputStructType != nil && fnInputStructType.Name == receiverType {
			return fnIdx, nil
		}
	}

	// Trying to find it in `Natives`.
	// Most likely a method from a core package.
	if opCode, found := OpCodes[pkg.Name+"."+fnName]; found {
		// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
		// Now an arena insert, site S0167.
		opFnIdx := prgrm.AddNativeFunctionInArray(Natives[opCode])
		return opFnIdx, nil
	}

	return -1, fmt.Errorf("method '%s' not found in package '%s'", fnName, pkg.Name)
}

// GetStruct ...
func (pkg *CXPackage) GetStruct(prgrm *CXProgram, strctName string) (*CXStruct, error) {
	if strctIdx, ok := pkg.Structs[strctName]; ok {
		// OBJMODEL: state.CXStructMng.CXStructGet(gid), phase 5 #337.
		// Now a pointer into an arena, site S0168.
		return &prgrm.CXStructs[strctIdx], nil
	}

	// looking in imports
	for _, impIdx := range pkg.Imports {
		// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
		// Now an arena getter, site S0169.
		imp, err := prgrm.GetPackageFromArray(impIdx)
		if err != nil {
			panic(err)
		}
		if strctIdx, ok := imp.Structs[strctName]; ok {
			// OBJMODEL: state.CXStructMng.CXStructGet(gid), phase 5 #337.
			// Now a pointer into an arena, site S0170.
			return &prgrm.CXStructs[strctIdx], nil
		}
	}

	return nil, fmt.Errorf("struct '%s' not found in package '%s'", strctName, pkg.Name)

}

// GetGlobal ...
func (pkg *CXPackage) GetGlobal(prgrm *CXProgram, defName string) (*CXTypeSignature, error) {
	for _, fieldIdx := range pkg.Globals.Fields {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0171.
		field := prgrm.GetCXTypeSignatureFromArray(fieldIdx)
		if field.Name == defName {
			return field, nil
		}
	}

	return nil, fmt.Errorf("global '%s' not found in package '%s'", defName, pkg.Name)
}

// GetCurrentFunction ...
func (pkg *CXPackage) GetCurrentFunction(prgrm *CXProgram) (*CXFunction, error) {
	if pkg.CurrentFunction == -1 {
		return nil, errors.New("current function is nil")
	}

	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S0172.
	return prgrm.GetFunctionFromArray(pkg.CurrentFunction), nil
}

// ----------------------------------------------------------------
//                             `CXPackage` Selectors

// SelectFunction ...
func (pkg *CXPackage) SelectFunction(prgrm *CXProgram, name string) (*CXFunction, error) {
	if _, ok := pkg.Functions[name]; !ok {
		return nil, fmt.Errorf("function '%s' does not exist", name)
	}

	idx := pkg.Functions[name]
	pkg.CurrentFunction = idx

	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S0173.
	return prgrm.GetFunctionFromArray(idx), nil
}

// MakePackage creates a new empty CXPackage.
//
// It can be filled in later with imports, structs, globals and functions.
// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
// Now a constructor, site S0174.
func MakePackage(name string) *CXPackage {
	// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
	// Now a heap literal, site S0175.
	return &CXPackage{
		Name: name,
		// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
		// Now a heap literal, site S0176.
		Globals:         &CXStruct{},
		Imports:         make(map[string]CXPackageIndex, 0),
		Structs:         make(map[string]CXStructIndex, 0),
		Functions:       make(map[string]CXFunctionIndex, 0),
		CurrentFunction: -1,
	}
}

// ----------------------------------------------------------------
//                             `CXPackage` Getters

// ----------------------------------------------------------------
//                     `CXPackage` Member handling

// AddImport ...
func (pkg *CXPackage) AddImport(prgrm *CXProgram, imp *CXPackage) *CXPackage {
	if _, ok := pkg.Imports[imp.Name]; !ok {
		// impIdx := prgrm.AddPackageInArray(imp)
		pkg.Imports[imp.Name] = CXPackageIndex(imp.Index)
	}

	return pkg
}

// RemoveImport ...
func (pkg *CXPackage) RemoveImport(impName string) {
	if _, ok := pkg.Imports[impName]; ok {
		// OBJMODEL: state.CXPackageMng.CXPackageRelease(gid), phase 4 #361.
		// Now a dropped object, site S0177.
		delete(pkg.Imports, impName)
	}
}

// AddFunction ...
func (pkg *CXPackage) AddFunction(prgrm *CXProgram, fn *CXFunction) (*CXPackage, CXFunctionIndex) {
	if _, ok := pkg.Functions[fn.Name]; ok {
		println(CompilationError(fn.FileName, fn.FileLine), "function redeclaration")
	}

	fn.Package = CXPackageIndex(pkg.Index)

	// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
	// Now an arena insert, site S0178.
	fnIdx := prgrm.AddFunctionInArray(fn)
	pkg.Functions[fn.Name] = fnIdx
	pkg.CurrentFunction = fnIdx

	return pkg, fnIdx
}

// RemoveFunction ...
func (pkg *CXPackage) RemoveFunction(fnName string) {
	if _, ok := pkg.Functions[fnName]; !ok {
		return
	}

	// OBJMODEL: state.CXFunctionMng.CXFunctionRelease(gid), phase 4 #351.
	// Now a dropped object, site S0179.
	delete(pkg.Functions, fnName)
}

// AddStruct ...
func (pkg *CXPackage) AddStruct(prgrm *CXProgram, strct *CXStruct) *CXPackage {
	if _, ok := pkg.Structs[strct.Name]; ok {
		return pkg
	}
	strct.Package = CXPackageIndex(pkg.Index)
	// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
	// Now an arena insert, site S0180.
	strctIdx := prgrm.AddStructInArray(strct)
	pkg.Structs[strct.Name] = strctIdx

	return pkg
}

// RemoveStruct ...
func (pkg *CXPackage) RemoveStruct(strctName string) {
	if _, ok := pkg.Structs[strctName]; !ok {
		return
	}
	// OBJMODEL: state.CXStructMng.CXStructRelease(gid), phase 4 #336.
	// Now a dropped object, site S0181.
	delete(pkg.Structs, strctName)
}

// AddGlobal ...
func (pkg *CXPackage) AddGlobal(prgrm *CXProgram, defIdx CXArgumentIndex) *CXPackage {
	// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
	// Now an arena getter, site S0182.
	defArg := prgrm.GetCXArgFromArray(defIdx)
	prgrm.CXArgs[defIdx].Package = CXPackageIndex(pkg.Index)

	for _, fieldIdx := range pkg.Globals.Fields {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0183.
		field := prgrm.GetCXTypeSignatureFromArray(fieldIdx)
		if field.Name == defArg.Name && field.Type == TYPE_CXARGUMENT_DEPRECATE {
			return pkg
		}
	}

	pkg.Globals.AddField_Globals_CXAtomicOps(prgrm, defIdx)

	return pkg
}

func (pkg *CXPackage) AddGlobal_TypeSignature(prgrm *CXProgram, typeSignatureIdx CXTypeSignatureIndex) *CXPackage {
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now an arena getter, site S0184.
	typeSignature := prgrm.GetCXTypeSignatureFromArray(typeSignatureIdx)
	typeSignature.Package = CXPackageIndex(pkg.Index)
	for _, fieldIdx := range pkg.Globals.Fields {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0185.
		field := prgrm.GetCXTypeSignatureFromArray(fieldIdx)
		if field.Name == typeSignature.Name {
			return pkg
		}
	}

	pkg.Globals.AddField_TypeSignature(prgrm, typeSignatureIdx)

	return pkg
}

// RemoveGlobal ...
// func (pkg *CXPackage) RemoveGlobal(prgrm *CXProgram, defName string) {
// 	lenGlobals := len(pkg.Globals)
// 	for i, defIdx := range pkg.Globals {
// 		def := prgrm.GetCXArg(defIdx)
// 		if def.Name == defName {
// 			if i == lenGlobals-1 {
// 				pkg.Globals = pkg.Globals[:len(pkg.Globals)-1]
// 			} else {
// 				pkg.Globals = append(pkg.Globals[:i], pkg.Globals[i+1:]...)
// 			}
// 			break
// 		}
// 	}
// }
