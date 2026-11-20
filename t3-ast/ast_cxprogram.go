package ast

import (
	"errors"
	"fmt"

	"github.com/skycoin/skycoin-cx/t0-types"
)

type StackSegmentStruct struct {
	Size     types.Pointer // This field stores the size of a CX program's stack
	StartsAt types.Pointer // Offset at which the stack segment starts in a CX program's memory
	Pointer  types.Pointer // At what byte the current stack frame is
}

type DataSegmentStruct struct {
	Size     types.Pointer // This field stores the size of a CX program's data segment size
	StartsAt types.Pointer // Offset at which the data segment starts in a CX program's memory
}

type HeapSegmentStruct struct {
	Size     types.Pointer // This field stores the size of a CX program's heap
	StartsAt types.Pointer // Offset at which the heap starts in a CX program's memory (normally the stack size)
	Pointer  types.Pointer // At what offset a CX program can insert a new object to the heap
}

// ----------------------------------------------------------------
//                         `CXProgram` Package handling

// AddPackage ...
func (cxprogram *CXProgram) AddPackage(mod *CXPackage) CXPackageIndex {
	if idx, ok := cxprogram.Packages[mod.Name]; ok {
		cxprogram.CurrentPackage = idx
		return idx
	}

	// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
	// Now an arena insert, site S0188.
	index := cxprogram.AddPackageInArray(mod)

	cxprogram.Packages[mod.Name] = CXPackageIndex(index)
	cxprogram.CurrentPackage = CXPackageIndex(index)

	return index
}

func (cxprogram *CXProgram) AddPackageInArray(pkg *CXPackage) CXPackageIndex {
	// The index of pkg after it will be added in the array
	pkg.Index = len(cxprogram.CXPackages)
	// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
	// Now an arena append, site S0189.
	cxprogram.CXPackages = append(cxprogram.CXPackages, *pkg)

	return CXPackageIndex(pkg.Index)
}

func (cxprogram *CXProgram) GetPackageFromArray(index CXPackageIndex) (*CXPackage, error) {
	if int(index) > (len(cxprogram.CXPackages) - 1) {
		return nil, fmt.Errorf("error: CXPackages[%d]: index out of bounds", index)
	}

	// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
	// Now a pointer into an arena, site S0190.
	return &cxprogram.CXPackages[index], nil
}

// RemovePackage ...
// func (cxprogram *CXProgram) RemovePackage(modName string) {
// 	// If doesnt exist, return
// 	// if cxprogram.Packages[modName] == nil {
// 	// 	return
// 	// }

// 	// Check if it is the current pkg so when it
// 	// is deleted, it will be replaced with new pkg
// 	isCurrentPkg := cxprogram.Packages[modName] == cxprogram.CurrentPackage

// 	// Delete package
// 	delete(cxprogram.Packages, modName)

// 	// This means that we're removing the package set to be the CurrentPackage.
// 	// If it is removed from the program's map of packages, cxprogram.CurrentPackage
// 	// would be pointing to a package meant to be collected by the GC.
// 	// We fix this by pointing to random package in the program's map of packages.
// 	if isCurrentPkg {
// 		for _, pkg := range cxprogram.Packages {
// 			cxprogram.CurrentPackage = pkg
// 			break
// 		}
// 	}

// }

// ----------------------------------------------------------------
//
//	`CXProgram` Function handling
func (cxprogram *CXProgram) AddFunctionInArray(fn *CXFunction) CXFunctionIndex {
	// The index of fn after it will be added in the array
	fn.Index = len(cxprogram.CXFunctions)

	// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
	// Now an arena append, site S0191.
	cxprogram.CXFunctions = append(cxprogram.CXFunctions, *fn)

	return CXFunctionIndex(fn.Index)
}

// nativeArgCopy returns a copy of a native argument for one program.
// Natives are shared by every program, so their Index must stay unset.
// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
// Now a constructor, site S0192.
func nativeArgCopy(arg *CXArgument) *CXArgument {
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now an object copy, site S0193.
	cp := *arg
	cp.Index = -1
	return &cp
}

func (cxprogram *CXProgram) AddNativeFunctionInArray(fn *CXNativeFunction) CXFunctionIndex {
	// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
	// Now a heap literal, site S0194.
	fnNative := &CXFunction{
		Index:        len(cxprogram.CXFunctions),
		AtomicOPCode: fn.AtomicOPCode,
		// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
		// Now a heap literal, site S0195.
		Inputs: &CXStruct{},
		// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
		// Now a heap literal, site S0196.
		Outputs:        &CXStruct{},
		LocalVariables: []string{},
	}

	// Add inputs to cx arg array
	for _, argIn := range fn.Inputs {
		err := fnNative.AddLocalVariableName(argIn.Name)
		if err != nil {
			// TODO: improve error handling
			panic("error adding local variable name")
		}

		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now a constructor call, site S0197.
		newField := GetCXTypeSignatureRepresentationOfCXArg(cxprogram, nativeArgCopy(argIn))

		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
		// Now an arena insert, site S0198.
		newFieldIdx := cxprogram.AddCXTypeSignatureInArray(newField)
		fnNative.Inputs.Fields = append(fnNative.Inputs.Fields, newFieldIdx)
	}

	// Add outputs to cx arg array
	for _, argOut := range fn.Outputs {
		err := fnNative.AddLocalVariableName(argOut.Name)
		if err != nil {
			// TODO: improve error handling
			panic("error adding local variable name")
		}

		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now a constructor call, site S0199.
		newField := GetCXTypeSignatureRepresentationOfCXArg(cxprogram, nativeArgCopy(argOut))
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
		// Now an arena insert, site S0200.
		newFieldIdx := cxprogram.AddCXTypeSignatureInArray(newField)
		fnNative.Outputs.Fields = append(fnNative.Outputs.Fields, newFieldIdx)
	}

	// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
	// Now an arena append, site S0201.
	cxprogram.CXFunctions = append(cxprogram.CXFunctions, *fnNative)

	return CXFunctionIndex(fnNative.Index)
}

func (cxprogram *CXProgram) GetFunctionFromArray(index CXFunctionIndex) *CXFunction {
	if index == -1 {
		return nil
	}

	if int(index) > (len(cxprogram.CXFunctions) - 1) {
		panic(fmt.Errorf("error: CXFunctions[%d]: index out of bounds", index))
	}

	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now a pointer into an arena, site S0202.
	return &cxprogram.CXFunctions[index]
}

// ----------------------------------------------------------------
//
//	`CXProgram` Structs handling
func (cxprogram *CXProgram) AddStructInArray(strct *CXStruct) CXStructIndex {
	// The index of fn after it will be added in the array
	strct.Index = len(cxprogram.CXStructs)

	// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
	// Now an arena append, site S0203.
	cxprogram.CXStructs = append(cxprogram.CXStructs, *strct)

	return CXStructIndex(strct.Index)
}

func (cxprogram *CXProgram) GetStructFromArray(index CXStructIndex) *CXStruct {
	if index == -1 {
		return nil
	}

	if int(index) > (len(cxprogram.CXStructs) - 1) {
		panic(fmt.Errorf("error: CXStructs[%d]: index out of bounds", index))
	}

	// OBJMODEL: state.CXStructMng.CXStructGet(gid), phase 5 #337.
	// Now a pointer into an arena, site S0204.
	return &cxprogram.CXStructs[index]
}

// ----------------------------------------------------------------
//            `CXProgram` CXLines, CXArgs, and CXAtomicOps Handling

func (cxprogram *CXProgram) GetCXLine(index int) (*CXLine, error) {
	if index > (len(cxprogram.CXLines) - 1) {
		return nil, fmt.Errorf("error: CXLines[%d]: index out of bounds", index)
	}

	// OBJMODEL: state.CXLineMng.CXLineGet(gid), phase 5 #312.
	// Now a pointer into an arena, site S0205.
	return &cxprogram.CXLines[index], nil
}

func (cxprogram *CXProgram) GetCXArg(index CXArgumentIndex) *CXArgument {
	if int(index) > (len(cxprogram.CXArgs) - 1) {
		panic(fmt.Errorf("error: CXArgs[%d]: index out of bounds", index))
	}

	// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
	// Now a pointer into an arena, site S0206.
	return &cxprogram.CXArgs[index]
}

func (cxprogram *CXProgram) GetCXAtomicOp(index int) (*CXAtomicOperator, error) {
	if index > (len(cxprogram.CXAtomicOps) - 1) {
		return nil, fmt.Errorf("error: CXAtomicOps[%d]: index out of bounds", index)
	}

	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
	// Now a pointer into an arena, site S0207.
	return &cxprogram.CXAtomicOps[index], nil
}

func (cxprogram *CXProgram) AddCXLine(CXLine *CXLine) int {
	// OBJMODEL: state.CXLineMng.CXLineAlloc(), phase 4 #311.
	// Now an arena append, site S0208.
	cxprogram.CXLines = append(cxprogram.CXLines, *CXLine)

	return len(cxprogram.CXLines) - 1
}

// func (cxprogram *CXProgram) AddCXArg(CXArg *CXArgument) int {
// 	cxprogram.CXArgs = append(cxprogram.CXArgs, *CXArg)

// 	return len(cxprogram.CXArgs) - 1
// }

func (cxprogram *CXProgram) AddCXAtomicOp(CXAtomicOp *CXAtomicOperator) int {
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
	// Now an arena append, site S0209.
	cxprogram.CXAtomicOps = append(cxprogram.CXAtomicOps, *CXAtomicOp)

	return len(cxprogram.CXAtomicOps) - 1
}

// ----------------------------------------------------------------
//
//	`CXProgram` CXTypeSignatures handling
func (cxprogram *CXProgram) AddCXTypeSignatureInArray(typeSignature *CXTypeSignature) CXTypeSignatureIndex {
	typeSignature.Index = CXTypeSignatureIndex(len(cxprogram.CXTypeSignatures))
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now an arena append, site S0210.
	cxprogram.CXTypeSignatures = append(cxprogram.CXTypeSignatures, *typeSignature)

	return typeSignature.Index
}

func (cxprogram *CXProgram) GetCXTypeSignatureFromArray(index CXTypeSignatureIndex) *CXTypeSignature {
	if index == -1 {
		return nil
	}

	if int(index) > (len(cxprogram.CXTypeSignatures) - 1) {
		panic(fmt.Errorf("error: CXTypeSignature[%d]: index out of bounds", index))
	}

	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now a pointer into an arena, site S0211.
	return &cxprogram.CXTypeSignatures[index]
}

// ----------------------------------------------------------------
//
//	`CXProgram` TypeSignature_Array handling
func (cxprogram *CXProgram) AddCXTypeSignatureArrayInArray(typeSignatureArray *CXTypeSignature_Array) int {
	// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc(), phase 4 #316.
	// Now an arena append, site S0212.
	cxprogram.TypeSignatureForArrays = append(cxprogram.TypeSignatureForArrays, *typeSignatureArray)

	return len(cxprogram.TypeSignatureForArrays) - 1
}

func (cxprogram *CXProgram) GetCXTypeSignatureArrayFromArray(index int) *CXTypeSignature_Array {
	if index == -1 {
		return nil
	}

	if int(index) > (len(cxprogram.TypeSignatureForArrays) - 1) {
		panic(fmt.Errorf("error: TypeSignatureForArrays[%d]: index out of bounds", index))
	}

	// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
	// Now a pointer into an arena, site S0213.
	return &cxprogram.TypeSignatureForArrays[index]
}

// ----------------------------------------------------------------
//
//	`CXProgram` TypeSignature_Struct handling
func (cxprogram *CXProgram) AddCXTypeSignatureStructInArray(typeSignatureStruct *CXTypeSignature_Struct) int {
	// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructAlloc(), phase 4 #321.
	// Now an arena append, site S0214.
	cxprogram.TypeSignatureForStructs = append(cxprogram.TypeSignatureForStructs, *typeSignatureStruct)

	return len(cxprogram.TypeSignatureForStructs) - 1
}

func (cxprogram *CXProgram) GetCXTypeSignatureStructFromArray(index int) *CXTypeSignature_Struct {
	if index == -1 {
		return nil
	}

	if int(index) > (len(cxprogram.TypeSignatureForStructs) - 1) {
		panic(fmt.Errorf("error: TypeSignatureForArrays[%d]: index out of bounds", index))
	}

	// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
	// Now a pointer into an arena, site S0215.
	return &cxprogram.TypeSignatureForStructs[index]
}

// ----------------------------------------------------------------
//
//	`CXProgram` CXArgument handling
func (cxprogram *CXProgram) AddCXArgInArray(cxArg *CXArgument) CXArgumentIndex {
	// The index of fn after it will be added in the array
	cxArg.Index = len(cxprogram.CXArgs)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now an arena append, site S0216.
	cxprogram.CXArgs = append(cxprogram.CXArgs, *cxArg)

	return CXArgumentIndex(cxArg.Index)
}

func (cxprogram *CXProgram) GetCXArgFromArray(index CXArgumentIndex) *CXArgument {
	if index == -1 {
		return nil
	}

	if int(index) > (len(cxprogram.CXArgs) - 1) {
		panic(fmt.Errorf("error: CXArgument[%d]: index out of bounds", index))
	}

	// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
	// Now a pointer into an arena, site S0217.
	return &cxprogram.CXArgs[index]
}

func (cxprogram *CXProgram) ConvertIndexArgsToPointerArgs(idxs []CXArgumentIndex) []*CXArgument {
	var cxArgs []*CXArgument
	for _, idx := range idxs {
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S0218.
		arg := cxprogram.GetCXArgFromArray(idx)
		cxArgs = append(cxArgs, arg)
	}
	return cxArgs
}

func (cxprogram *CXProgram) AddPointerArgsToCXArgsArray(cxArgs []*CXArgument) []CXArgumentIndex {
	var cxArgsIdxs []CXArgumentIndex
	for _, cxArg := range cxArgs {
		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now an arena insert, site S0219.
		cxArgIdx := cxprogram.AddCXArgInArray(cxArg)
		cxArgsIdxs = append(cxArgsIdxs, cxArgIdx)
	}
	return cxArgsIdxs
}

// Temporary only for intitial new struct def implementation.
func (cxprogram *CXProgram) ConvertIndexTypeSignaturesToPointerArgs(idxs []CXTypeSignatureIndex) []*CXArgument {
	var cxArgs []*CXArgument
	for _, typeSignatureIdx := range idxs {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0220.
		typeSignature := cxprogram.GetCXTypeSignatureFromArray(typeSignatureIdx)
		if typeSignature.Type == TYPE_CXARGUMENT_DEPRECATE {
			idx := typeSignature.Meta
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0221.
			arg := cxprogram.GetCXArgFromArray(CXArgumentIndex(idx))
			cxArgs = append(cxArgs, arg)
		}
	}
	return cxArgs
}

// Temporary only for intitial new struct def implementation.
func (cxprogram *CXProgram) AddPointerArgsToTypeSignaturesArray(cxArgs []*CXArgument) []CXTypeSignatureIndex {
	var cxTypeSignaturesIdxs []CXTypeSignatureIndex
	for _, cxArg := range cxArgs {
		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now an arena insert, site S0222.
		cxArgIdx := cxprogram.AddCXArgInArray(cxArg)

		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
		// Now a heap literal, site S0223.
		newCXTypeSignature := &CXTypeSignature{
			Name:       cxArg.Name,
			Package:    cxArg.Package,
			Offset:     cxArg.Offset,
			Type:       TYPE_CXARGUMENT_DEPRECATE,
			Meta:       int(cxArgIdx),
			ArgDetails: cxArg.ArgDetails,
		}

		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
		// Now an arena insert, site S0224.
		newCXTypeSignatureIdx := cxprogram.AddCXTypeSignatureInArray(newCXTypeSignature)
		cxTypeSignaturesIdxs = append(cxTypeSignaturesIdxs, newCXTypeSignatureIdx)
	}
	return cxTypeSignaturesIdxs
}

// ----------------------------------------------------------------
//                             `CXProgram` Getters

func (cxprogram *CXProgram) GetOperation(expr *CXExpression) (*CXAtomicOperator, *CXArgument, *CXLine, error) {
	switch expr.Type {
	case CX_ATOMIC_OPERATOR:
		// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
		// Now an arena getter, site S0225.
		cxAtomicOp, err := cxprogram.GetCXAtomicOp(expr.Index)
		if err != nil {
			// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
			// Also state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326. Also state.CXLineMng.CXLineAlloc(), phase 4 #311. Site S0226 S0227 S0228.
			return &CXAtomicOperator{}, &CXArgument{}, &CXLine{}, err
		}

		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Also state.CXLineMng.CXLineAlloc(), phase 4 #311. Site S0229 S0230.
		return cxAtomicOp, &CXArgument{}, &CXLine{}, nil
	case CX_LINE:
		// OBJMODEL: state.CXLineMng.CXLineGet(gid), phase 5 #312.
		// Now an arena getter, site S0231.
		cxLine, err := cxprogram.GetCXLine(expr.Index)
		if err != nil {
			// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
			// Also state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326. Also state.CXLineMng.CXLineAlloc(), phase 4 #311. Site S0232 S0233 S0234.
			return &CXAtomicOperator{}, &CXArgument{}, &CXLine{}, err
		}

		// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
		// Also state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326. Site S0235 S0236.
		return &CXAtomicOperator{}, &CXArgument{}, cxLine, nil
		// case CX_ARGUMENT:
	}

	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
	// Also state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326. Also state.CXLineMng.CXLineAlloc(), phase 4 #311. Site S0237 S0238 S0239.
	return &CXAtomicOperator{}, &CXArgument{}, &CXLine{}, fmt.Errorf("operation type is not found.")
}

func (cxprogram *CXProgram) GetPreviousCXLine(exprs []CXExpression, currIndex int) (*CXLine, error) {
	for i := currIndex; i >= 0; i-- {
		if exprs[i].Type == CX_LINE {
			_, _, cxLine, err := cxprogram.GetOperation(&exprs[i])
			if err != nil {
				// OBJMODEL: state.CXLineMng.CXLineAlloc(), phase 4 #311.
				// Now a heap literal, site S0240.
				return &CXLine{}, err
			}

			return cxLine, nil
		}
	}
	// OBJMODEL: state.CXLineMng.CXLineAlloc(), phase 4 #311.
	// Now a heap literal, site S0241.
	return &CXLine{}, fmt.Errorf("CXLine not found.")
}

func (cxprogram *CXProgram) GetCXAtomicOpFromExpressions(exprs []CXExpression, currIndex int) (*CXAtomicOperator, error) {
	for i := currIndex; i < len(exprs); i++ {
		if exprs[i].Type == CX_ATOMIC_OPERATOR {
			// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
			// Now an arena getter, site S0242.
			cxAtomicOp, err := cxprogram.GetCXAtomicOp(exprs[i].Index)
			if err != nil {
				// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
				// Now a heap literal, site S0243.
				return &CXAtomicOperator{}, err
			}

			return cxAtomicOp, nil
		}
	}
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
	// Now a heap literal, site S0244.
	return &CXAtomicOperator{}, fmt.Errorf("CXAtomicOperator not found.")
}

func (cxprogram *CXProgram) GetPreviousCXAtomicOpFromExpressions(exprs []CXExpression, currIndex int) (*CXAtomicOperator, error) {
	for i := currIndex; i >= 0; i-- {
		if exprs[i].Type == CX_ATOMIC_OPERATOR {
			// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
			// Now an arena getter, site S0245.
			cxAtomicOp, err := cxprogram.GetCXAtomicOp(exprs[i].Index)
			if err != nil {
				// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
				// Now a heap literal, site S0246.
				return &CXAtomicOperator{}, err
			}

			return cxAtomicOp, nil
		}
	}
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
	// Now a heap literal, site S0247.
	return &CXAtomicOperator{}, fmt.Errorf("CXAtomicOperator not found.")
}

// Only two users, both in cx/execute.go
func (cxprogram *CXProgram) SelectPackage(name string) (*CXPackage, error) {
	if _, ok := cxprogram.Packages[name]; !ok {
		return nil, fmt.Errorf("Package '%s' does not exist", name)
	}

	cxprogram.CurrentPackage = cxprogram.Packages[name]

	// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
	// Now an arena getter, site S0248.
	return cxprogram.GetPackageFromArray(cxprogram.Packages[name])
}

// GetCurrentPackage ...
func (cxprogram *CXProgram) GetCurrentPackage() (*CXPackage, error) {
	if cxprogram.CurrentPackage == -1 {
		return nil, errors.New("current package is nil")
	}

	// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
	// Now an arena getter, site S0249.
	return cxprogram.GetPackageFromArray(cxprogram.CurrentPackage)
}

// GetCurrentFunction ...
func (cxprogram *CXProgram) GetCurrentFunction() (*CXFunction, error) {
	if cxprogram.CurrentPackage == -1 {
		return nil, errors.New("current package is nil")
	}

	// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
	// Now an arena getter, site S0250.
	currentPackage, err := cxprogram.GetPackageFromArray(cxprogram.CurrentPackage)
	if err != nil {
		// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
		// Now a heap literal, site S0251.
		return &CXFunction{}, err
	}

	if currentPackage.CurrentFunction == -1 {
		return nil, errors.New("current function is nil")
	}

	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S0252.
	return cxprogram.GetFunctionFromArray(currentPackage.CurrentFunction), nil

}

// GetPackage ...
func (cxprogram *CXProgram) GetPackage(pkgName string) (*CXPackage, error) {
	if _, ok := cxprogram.Packages[pkgName]; !ok {
		return nil, fmt.Errorf("package '%s' not found", pkgName)
	}

	// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
	// Now an arena getter, site S0253.
	return cxprogram.GetPackageFromArray(cxprogram.Packages[pkgName])
}

// GetStruct ...
func (cxprogram *CXProgram) GetStruct(strctName string, pkgName string) (*CXStruct, error) {
	pkg, err := cxprogram.GetPackage(pkgName)
	if err != nil {
		return nil, err
	}

	strct, err := pkg.GetStruct(cxprogram, strctName)
	if err != nil {
		return nil, err
	}

	return strct, nil
}

// GetFunction ...
func (cxprogram *CXProgram) GetFunction(fnName string, pkgName string) (*CXFunction, error) {
	pkg, err := cxprogram.GetPackage(pkgName)
	if err != nil {
		return nil, err
	}

	fn, err := pkg.GetFunction(cxprogram, fnName)
	if err != nil {
		return nil, err
	}

	return fn, nil
}

// GetCurrentCall returns the current CXCall
// TODO: What does this do?
// TODO: Only used in OP_JMP
func (cxprogram *CXProgram) GetCurrentCall() *CXCall {
	return &cxprogram.CallStack[cxprogram.CallCounter]
}

// GetGlobal ...
/*
func (cxprogram *CXProgram) GetGlobal(name string) (*CXArgument, error) {
	mod, err := cxprogram.GetCurrentPackage()
	if err != nil {
		return nil, err
	}

	var foundArgument *CXArgument
	for _, def := range mod.Globals {
		if def.Name == name {
			foundArgument = def
			break
		}
	}

	for _, imp := range mod.Imports {
		for _, def := range imp.Globals {
			if def.Name == name {
				foundArgument = def
				break
			}
		}
	}

	if foundArgument == nil {
		return nil, fmt.Errorf("global '%s' not found", name)
	}
	return foundArgument, nil
}
*/

/*
// GetCurrentOpCode returns the current OpCode
func (cxprogram *CXProgram) GetCurrentOpCode() int {
	return cxprogram.GetCurrentExpression2().Operator.OpCode
}
*/

/*
//not used
func (cxprogram *CXProgram) GetFramePointer() int {
	return cxprogram.GetCurrentCall().FramePointer
}
*/

// ----------------------------------------------------------------
//                             `CXProgram` Debugging

// PrintProgram prints the abstract syntax tree of a CX program in a
// human-readable format.
func (cxprogram *CXProgram) PrintProgram() {
	fmt.Println(ToString(cxprogram))
}
