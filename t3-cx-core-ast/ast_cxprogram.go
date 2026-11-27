package ast

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	stringpool "github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
	gc "github.com/skycoin/skycoin-cx/t2-cx-runtime-gc"
)

type StackSegmentStruct struct {
	Size     valuetypes.Pointer // This field stores the size of a CX program's stack
	StartsAt valuetypes.Pointer // Offset at which the stack segment starts in a CX program's memory
	Pointer  valuetypes.Pointer // At what byte the current stack frame is
}

type DataSegmentStruct struct {
	Size     valuetypes.Pointer // This field stores the size of a CX program's data segment size
	StartsAt valuetypes.Pointer // Offset at which the data segment starts in a CX program's memory
}

type HeapSegmentStruct struct {
	Size     valuetypes.Pointer // This field stores the size of a CX program's heap
	StartsAt valuetypes.Pointer // Offset at which the heap starts in a CX program's memory (normally the stack size)
	Pointer  valuetypes.Pointer // At what offset a CX program can insert a new object to the heap
	// Max is the heap cap, the most bytes the heap grows to (D42). It is
	// set when the program is compiled from -heap-max and stored in a built
	// program. EnsureMinimumHeapSize allocates memory for it once.
	Max valuetypes.Pointer
	// Pressure is the bytes allocated since the last collection.
	Pressure valuetypes.Pointer
	// PressureLimit is the free heap after the last collection. The next
	// GarbageCollectorCheck past it asks for a collection.
	PressureLimit valuetypes.Pointer
	// Types holds the GCTypes of the roots, built before the run by
	// GarbageCollectorPrepare.
	Types *GCTypeCache
	// Roots is the root table of the collection running, kept for reuse.
	Roots gc.RootTable
	// Collector is the scratch memory of the collector.
	Collector gc.State
}

// ----------------------------------------------------------------
//                         `CXProgram` Package handling

// PackageAdd ...
func (cxprogram *CXProgram) PackageAdd(mod *CXPackage) CXPackageIndex {
	if idx, ok := cxprogram.Packages[mod.Name]; ok {
		cxprogram.CurrentPackage = idx
		return idx
	}

	// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
	// Now an arena insert, site S0188.
	index := cxprogram.PackageInArrayAdd(mod)

	cxprogram.Packages[mod.Name] = CXPackageIndex(index)
	cxprogram.CurrentPackage = CXPackageIndex(index)

	return index
}

func (cxprogram *CXProgram) PackageInArrayAdd(pkg *CXPackage) CXPackageIndex {
	// The index of pkg after it will be added in the array
	pkg.Index = len(cxprogram.CXPackages)
	// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
	// Now an arena append, site S0189.
	cxprogram.CXPackages = append(cxprogram.CXPackages, *pkg)

	return CXPackageIndex(pkg.Index)
}

func (cxprogram *CXProgram) PackageFromArrayGet(index CXPackageIndex) (*CXPackage, error) {
	if int(index) > (len(cxprogram.CXPackages) - 1) {
		return nil, CXProgramLookupError{CXProgramLookupErrorCodePackageIndex, int(index)}
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

// CXProgramFunctionAlloc allocates a function in the program's CXFunctionMng,
// sets its Index to the slot and returns its Gid and the pointer to fill it
// through.
func (cxprogram *CXProgram) CXProgramFunctionAlloc() (CXFunctionGid, *CXFunction) {
	functionMng := &cxprogram.CxIrState.CXFunctionMng
	gid := functionMng.CXFunctionAlloc()
	fn := functionMng.CXFunctionGet(gid)
	fn.Index = int(objmodel.GidIndex(int64(gid)))
	return gid, fn
}

// CXProgramFunctionRelease releases the function gid names.
func (cxprogram *CXProgram) CXProgramFunctionRelease(gid CXFunctionGid) {
	cxprogram.CxIrState.CXFunctionMng.CXFunctionRelease(gid)
}

// CXProgramFunctionSlotCount is the number of slots of the program's
// CXFunctionMng ever used. A function's slot is below it.
func (cxprogram *CXProgram) CXProgramFunctionSlotCount() int {
	return int(cxprogram.CxIrState.CXFunctionMng.CXFunctionMngNext)
}

// CXProgramFunctionSlotLive returns the function in slot index, or nil for a
// released slot, for a walk over every function.
func (cxprogram *CXProgram) CXProgramFunctionSlotLive(index int) *CXFunction {
	functionMng := &cxprogram.CxIrState.CXFunctionMng
	if !functionMng.CXFunctionMngLive[index] {
		return nil
	}
	return &functionMng.CXFunctionMngObjects[index]
}

// CXProgramFunctionSlotGid returns the Gid of the live function in slot
// index, the slot the lowered image and the vm number functions by.
func (cxprogram *CXProgram) CXProgramFunctionSlotGid(index uint32) CXFunctionGid {
	return CXFunctionGid(objmodel.GidCompose(CXFunctionGidTypeTag, cxprogram.CxIrState.CXFunctionMng.CXFunctionMngGenerations[index], index))
}

// CXProgramFunctionGid returns the Gid of fn, a live function of the program.
func (cxprogram *CXProgram) CXProgramFunctionGid(fn *CXFunction) CXFunctionGid {
	return cxprogram.CXProgramFunctionSlotGid(uint32(fn.Index))
}

// CxIrStateFunctionGetOrNil returns the function gid names, or nil for
// CXFunctionGidNil, the operator of a declaration.
func (state *CxIrState) CxIrStateFunctionGetOrNil(gid CXFunctionGid) *CXFunction {
	if gid == CXFunctionGidNil {
		return nil
	}
	return state.CXFunctionMng.CXFunctionGet(gid)
}

// CXFunctionGidSlot returns the slot of the function gid names, its number in
// the lowered image.
func CXFunctionGidSlot(gid CXFunctionGid) uint32 {
	return objmodel.GidIndex(int64(gid))
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

// NativeFunctionInArrayAdd allocates the function a call of the native fn
// calls, and returns its Gid.
func (cxprogram *CXProgram) NativeFunctionInArrayAdd(fn *CXNativeFunction) CXFunctionGid {
	if fn.AtomicOPCode == opcodetable.OpLibraryCall {
		cxprogram.ProgramLibrariesUsed[fn.LibraryNumber] = true
	} else if library := opcodetable.OpcodeLibraryOf(fn.AtomicOPCode); library != 0 {
		cxprogram.ProgramLibrariesUsed[library] = true
	}
	gid, fnNative := cxprogram.CXProgramFunctionAlloc()
	fnNative.AtomicOPCode = fn.AtomicOPCode
	fnNative.LibraryNumber = fn.LibraryNumber
	fnNative.LibraryFunctionNumber = fn.LibraryFunctionNumber
	// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
	// Now a heap literal, site S0195.
	fnNative.Inputs = &CXStruct{}
	// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
	// Now a heap literal, site S0196.
	fnNative.Outputs = &CXStruct{}
	fnNative.LocalVariables = []stringpool.StringGid{}

	// Add inputs to cx arg array
	for _, argIn := range fn.Inputs {
		err := fnNative.LocalVariableNameAdd(argIn.Name)
		if err != nil {
			// TODO: improve error handling
			panic("error adding local variable name")
		}

		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now a constructor call, site S0197.
		newField := CXTypeSignatureRepresentationOfCXArgGet(cxprogram, nativeArgCopy(argIn))

		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
		// Now an arena insert, site S0198.
		newFieldIdx := cxprogram.CXTypeSignatureInArrayAdd(newField)
		fnNative.Inputs.Fields = append(fnNative.Inputs.Fields, newFieldIdx)
	}

	// Add outputs to cx arg array
	for _, argOut := range fn.Outputs {
		err := fnNative.LocalVariableNameAdd(argOut.Name)
		if err != nil {
			// TODO: improve error handling
			panic("error adding local variable name")
		}

		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now a constructor call, site S0199.
		newField := CXTypeSignatureRepresentationOfCXArgGet(cxprogram, nativeArgCopy(argOut))
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
		// Now an arena insert, site S0200.
		newFieldIdx := cxprogram.CXTypeSignatureInArrayAdd(newField)
		fnNative.Outputs.Fields = append(fnNative.Outputs.Fields, newFieldIdx)
	}

	return gid
}

// ----------------------------------------------------------------
//
//	`CXProgram` Structs handling
func (cxprogram *CXProgram) StructInArrayAdd(strct *CXStruct) CXStructIndex {
	// The index of fn after it will be added in the array
	strct.Index = len(cxprogram.CXStructs)

	// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
	// Now an arena append, site S0203.
	cxprogram.CXStructs = append(cxprogram.CXStructs, *strct)

	return CXStructIndex(strct.Index)
}

func (cxprogram *CXProgram) StructFromArrayGet(index CXStructIndex) *CXStruct {
	if index == -1 {
		return nil
	}

	if int(index) > (len(cxprogram.CXStructs) - 1) {
		panic(CXProgramLookupError{CXProgramLookupErrorCodeStructIndex, int(index)})
	}

	// OBJMODEL: state.CXStructMng.CXStructGet(gid), phase 5 #337.
	// Now a pointer into an arena, site S0204.
	return &cxprogram.CXStructs[index]
}

// ----------------------------------------------------------------
//            `CXProgram` CXLines, CXArgs, and CXAtomicOps Handling

// CXLineGet returns the line in slot index of the program's CXLineMng. An
// expression keeps the slot index, not the Gid, until phase 5 of CXLine.
func (cxprogram *CXProgram) CXLineGet(index int) (*CXLine, error) {
	lineMng := &cxprogram.CxIrState.CXLineMng
	if index < 0 || index >= int(lineMng.CXLineMngNext) {
		return nil, CXProgramLookupError{CXProgramLookupErrorCodeLineIndex, index}
	}

	gid := objmodel.GidCompose(CXLineGidTypeTag, lineMng.CXLineMngGenerations[index], uint32(index))
	return lineMng.CXLineGet(CXLineGid(gid)), nil
}

// CXArgGet returns the argument in slot index of the program's
// CXArgumentMng. An argument is named by its slot index, not its Gid, until
// phase 5 of CXArgument.
func (cxprogram *CXProgram) CXArgGet(index CXArgumentIndex) *CXArgument {
	argumentMng := &cxprogram.CxIrState.CXArgumentMng
	if index < 0 || int(index) >= int(argumentMng.CXArgumentMngNext) {
		panic(CXProgramLookupError{CXProgramLookupErrorCodeArgumentIndex, int(index)})
	}

	gid := objmodel.GidCompose(CXArgumentGidTypeTag, argumentMng.CXArgumentMngGenerations[index], uint32(index))
	return argumentMng.CXArgumentGet(CXArgumentGid(gid))
}

func (cxprogram *CXProgram) CXAtomicOpGet(index int) (*CXAtomicOperator, error) {
	if index > (len(cxprogram.CXAtomicOps) - 1) {
		return nil, CXProgramLookupError{CXProgramLookupErrorCodeAtomicOperatorIndex, index}
	}

	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
	// Now a pointer into an arena, site S0207.
	return &cxprogram.CXAtomicOps[index], nil
}

func (cxprogram *CXProgram) CXAtomicOpAdd(CXAtomicOp *CXAtomicOperator) int {
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
	// Now an arena append, site S0209.
	cxprogram.CXAtomicOps = append(cxprogram.CXAtomicOps, *CXAtomicOp)

	return len(cxprogram.CXAtomicOps) - 1
}

// ----------------------------------------------------------------
//
//	`CXProgram` CXTypeSignatures handling

// CXTypeSignatureInArrayAdd allocates a type signature in the program's
// CXTypeSignatureMng and copies typeSignature into it. typeSignature.Index is
// set to the slot index, which it returns.
func (cxprogram *CXProgram) CXTypeSignatureInArrayAdd(typeSignature *CXTypeSignature) CXTypeSignatureIndex {
	gid := cxprogram.CxIrState.CXTypeSignatureMng.CXTypeSignatureAlloc()
	typeSignature.Index = CXTypeSignatureIndex(objmodel.GidIndex(int64(gid)))
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now an object copy, site S1775.
	*cxprogram.CxIrState.CXTypeSignatureMng.CXTypeSignatureGet(gid) = *typeSignature

	return typeSignature.Index
}

// CXTypeSignatureAlloc allocates a type signature in the program's
// CXTypeSignatureMng, sets its Index to the slot and returns its Gid and the
// pointer to fill it through.
func (cxprogram *CXProgram) CXTypeSignatureAlloc() (CXTypeSignatureGid, *CXTypeSignature) {
	typeSignatureMng := &cxprogram.CxIrState.CXTypeSignatureMng
	gid := typeSignatureMng.CXTypeSignatureAlloc()
	typeSignature := typeSignatureMng.CXTypeSignatureGet(gid)
	typeSignature.Index = CXTypeSignatureIndex(objmodel.GidIndex(int64(gid)))
	return gid, typeSignature
}

// CXTypeSignatureCopyAlloc allocates a type signature that copies src, which
// lives in the manager, and returns it like CXTypeSignatureAlloc.
func (cxprogram *CXProgram) CXTypeSignatureCopyAlloc(src *CXTypeSignature) (CXTypeSignatureGid, *CXTypeSignature) {
	typeSignatureMng := &cxprogram.CxIrState.CXTypeSignatureMng
	gid := typeSignatureMng.CXTypeSignatureAlloc()
	typeSignature := typeSignatureMng.CXTypeSignatureGet(gid)
	*typeSignature = *src
	typeSignature.Index = CXTypeSignatureIndex(objmodel.GidIndex(int64(gid)))
	return gid, typeSignature
}

// CXTypeSignatureFromArrayGet returns the type signature in slot index of
// the program's CXTypeSignatureMng, or nil for -1. A type signature is named
// by its slot index, not its Gid, until phase 5 of CXTypeSignature.
func (cxprogram *CXProgram) CXTypeSignatureFromArrayGet(index CXTypeSignatureIndex) *CXTypeSignature {
	if index == -1 {
		return nil
	}

	typeSignatureMng := &cxprogram.CxIrState.CXTypeSignatureMng
	if index < 0 || int(index) >= int(typeSignatureMng.CXTypeSignatureMngNext) {
		panic(CXProgramLookupError{CXProgramLookupErrorCodeTypeSignatureIndex, int(index)})
	}

	gid := objmodel.GidCompose(CXTypeSignatureGidTypeTag, typeSignatureMng.CXTypeSignatureMngGenerations[index], uint32(index))
	return typeSignatureMng.CXTypeSignatureGet(CXTypeSignatureGid(gid))
}

// ----------------------------------------------------------------
//
//	`CXProgram` TypeSignature_Array handling

// CXTypeSignatureArrayFromArrayGet returns the array signature in slot index
// of the program's CXTypeSignature_ArrayMng, or nil for -1. A type signature's
// Meta keeps the slot index, not the Gid, until phase 5 of
// CXTypeSignature_Array.
func (cxprogram *CXProgram) CXTypeSignatureArrayFromArrayGet(index int) *CXTypeSignature_Array {
	if index == -1 {
		return nil
	}

	typeSignatureArrayMng := &cxprogram.CxIrState.CXTypeSignature_ArrayMng
	if index < 0 || index >= int(typeSignatureArrayMng.CXTypeSignature_ArrayMngNext) {
		panic(CXProgramLookupError{CXProgramLookupErrorCodeTypeSignatureArrayIndex, index})
	}

	gid := objmodel.GidCompose(CXTypeSignature_ArrayGidTypeTag, typeSignatureArrayMng.CXTypeSignature_ArrayMngGenerations[index], uint32(index))
	return typeSignatureArrayMng.CXTypeSignature_ArrayGet(CXTypeSignature_ArrayGid(gid))
}

// ----------------------------------------------------------------
//
//	`CXProgram` TypeSignature_Struct handling

// CXTypeSignatureStructFromArrayGet returns the struct signature in slot index
// of the program's CXTypeSignature_StructMng, or nil for -1. A type
// signature's Meta keeps the slot index, not the Gid, until phase 5 of
// CXTypeSignature_Struct.
func (cxprogram *CXProgram) CXTypeSignatureStructFromArrayGet(index int) *CXTypeSignature_Struct {
	if index == -1 {
		return nil
	}

	typeSignatureStructMng := &cxprogram.CxIrState.CXTypeSignature_StructMng
	if index < 0 || index >= int(typeSignatureStructMng.CXTypeSignature_StructMngNext) {
		panic(CXProgramLookupError{CXProgramLookupErrorCodeTypeSignatureStructIndex, index})
	}

	gid := objmodel.GidCompose(CXTypeSignature_StructGidTypeTag, typeSignatureStructMng.CXTypeSignature_StructMngGenerations[index], uint32(index))
	return typeSignatureStructMng.CXTypeSignature_StructGet(CXTypeSignature_StructGid(gid))
}

// ----------------------------------------------------------------
//
//	`CXProgram` CXArgument handling

// CXArgumentAlloc allocates an argument in the program's CXArgumentMng, sets
// its Index to the slot and returns its Gid and the pointer to fill it
// through.
func (cxprogram *CXProgram) CXArgumentAlloc() (CXArgumentGid, *CXArgument) {
	argumentMng := &cxprogram.CxIrState.CXArgumentMng
	gid := argumentMng.CXArgumentAlloc()
	arg := argumentMng.CXArgumentGet(gid)
	arg.Index = int(objmodel.GidIndex(int64(gid)))
	return gid, arg
}

// CXArgInArrayAdd returns the slot of cxArg. An argument that already lives
// in the program's CXArgumentMng is not copied. A heap argument, which only
// the natives, serialization and tests still make, is copied into a new slot
// and cxArg.Index is set to it.
func (cxprogram *CXProgram) CXArgInArrayAdd(cxArg *CXArgument) CXArgumentIndex {
	if cxprogram.cxArgumentManaged(cxArg) {
		return CXArgumentIndex(cxArg.Index)
	}

	gid := cxprogram.CxIrState.CXArgumentMng.CXArgumentAlloc()
	cxArg.Index = int(objmodel.GidIndex(int64(gid)))
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now an object copy, site S1632.
	*cxprogram.CxIrState.CXArgumentMng.CXArgumentGet(gid) = *cxArg

	return CXArgumentIndex(cxArg.Index)
}

// CXArgRelease releases the argument in slot index of the program's
// CXArgumentMng, the counterpart of CXArgGet.
func (cxprogram *CXProgram) CXArgRelease(index CXArgumentIndex) {
	argumentMng := &cxprogram.CxIrState.CXArgumentMng
	gid := objmodel.GidCompose(CXArgumentGidTypeTag, argumentMng.CXArgumentMngGenerations[index], uint32(index))
	argumentMng.CXArgumentRelease(CXArgumentGid(gid))
}

// CXArgWithDebugRelease releases the argument in slot index and its debug
// record, for an argument that is dropped before anything borrowed the
// record.
func (cxprogram *CXProgram) CXArgWithDebugRelease(index CXArgumentIndex) {
	// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
	// Now an arena getter, site S1782.
	argDetails := cxprogram.CXArgGet(index).ArgDetails
	cxprogram.CXArgRelease(index)
	if argDetails != CXArgumentDebugGidNil {
		cxprogram.CxIrState.CXArgumentDebugMng.CXArgumentDebugRelease(argDetails)
	}
}

// CXArgBuilderRelease releases the managed builder cxArg and its debug record,
// when nothing took it. A heap argument is left alone.
func (cxprogram *CXProgram) CXArgBuilderRelease(cxArg *CXArgument) {
	if cxprogram.cxArgumentManaged(cxArg) {
		cxprogram.CXArgWithDebugRelease(CXArgumentIndex(cxArg.Index))
	}
}

// CXArgUnlessNamedRelease releases the managed argument cxArg unless the type
// signature in slot typeSigIdx names it. The signature keeps the debug
// record. A heap argument is left alone.
func (cxprogram *CXProgram) CXArgUnlessNamedRelease(cxArg *CXArgument, typeSigIdx CXTypeSignatureIndex) {
	if !cxprogram.cxArgumentManaged(cxArg) {
		return
	}
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now an arena getter, site S1783.
	typeSig := cxprogram.CXTypeSignatureFromArrayGet(typeSigIdx)
	if typeSig.Type != TypeCXArgumentDeprecate || typeSig.Meta != cxArg.Index {
		cxprogram.CXArgRelease(CXArgumentIndex(cxArg.Index))
	}
}

// CXTypeSignatureRelease releases the type signature in slot index of the
// program's CXTypeSignatureMng, the counterpart of
// CXTypeSignatureFromArrayGet.
func (cxprogram *CXProgram) CXTypeSignatureRelease(index CXTypeSignatureIndex) {
	typeSignatureMng := &cxprogram.CxIrState.CXTypeSignatureMng
	gid := objmodel.GidCompose(CXTypeSignatureGidTypeTag, typeSignatureMng.CXTypeSignatureMngGenerations[index], uint32(index))
	typeSignatureMng.CXTypeSignatureRelease(CXTypeSignatureGid(gid))
}

// cxArgumentManaged reports whether cxArg is the object in its own slot of the
// program's CXArgumentMng, as opposed to an argument on the heap.
func (cxprogram *CXProgram) cxArgumentManaged(cxArg *CXArgument) bool {
	if cxArg == nil {
		return false
	}
	objects := cxprogram.CxIrState.CXArgumentMng.CXArgumentMngObjects
	return cxArg.Index >= 0 && cxArg.Index < len(objects) && &objects[cxArg.Index] == cxArg
}

// cxArgumentSlot returns the slot that a type signature names cxArg by. A
// heap argument is copied in when it has no slot yet, or always when
// copyHeap is set.
func (cxprogram *CXProgram) cxArgumentSlot(cxArg *CXArgument, copyHeap bool) CXArgumentIndex {
	if cxprogram.cxArgumentManaged(cxArg) || !copyHeap && cxArg.Index != -1 && cxArg.Index != 0 {
		return CXArgumentIndex(cxArg.Index)
	}

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now an arena insert, site S0332.
	return cxprogram.CXArgInArrayAdd(cxArg)
}

func (cxprogram *CXProgram) CXArgFromArrayGet(index CXArgumentIndex) *CXArgument {
	if index == -1 {
		return nil
	}

	// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
	// Now an arena getter, site S1633.
	return cxprogram.CXArgGet(index)
}

func (cxprogram *CXProgram) IndexArgsToPointerArgsConvert(idxs []CXArgumentIndex) []*CXArgument {
	var cxArgs []*CXArgument
	for _, idx := range idxs {
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S0218.
		arg := cxprogram.CXArgFromArrayGet(idx)
		cxArgs = append(cxArgs, arg)
	}
	return cxArgs
}

func (cxprogram *CXProgram) PointerArgsToCXArgsArrayAdd(cxArgs []*CXArgument) []CXArgumentIndex {
	var cxArgsIdxs []CXArgumentIndex
	for _, cxArg := range cxArgs {
		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now an arena insert, site S0219.
		cxArgIdx := cxprogram.CXArgInArrayAdd(cxArg)
		cxArgsIdxs = append(cxArgsIdxs, cxArgIdx)
	}
	return cxArgsIdxs
}

// Temporary only for intitial new struct def implementation.
func (cxprogram *CXProgram) IndexTypeSignaturesToPointerArgsConvert(idxs []CXTypeSignatureIndex) []*CXArgument {
	var cxArgs []*CXArgument
	for _, typeSignatureIdx := range idxs {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0220.
		typeSignature := cxprogram.CXTypeSignatureFromArrayGet(typeSignatureIdx)
		if typeSignature.Type == TypeCXArgumentDeprecate {
			idx := typeSignature.Meta
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0221.
			arg := cxprogram.CXArgFromArrayGet(CXArgumentIndex(idx))
			cxArgs = append(cxArgs, arg)
		}
	}
	return cxArgs
}

// Temporary only for intitial new struct def implementation.
func (cxprogram *CXProgram) PointerArgsToTypeSignaturesArrayAdd(cxArgs []*CXArgument) []CXTypeSignatureIndex {
	var cxTypeSignaturesIdxs []CXTypeSignatureIndex
	for _, cxArg := range cxArgs {
		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now an arena insert, site S0222.
		cxArgIdx := cxprogram.CXArgInArrayAdd(cxArg)

		_, newCXTypeSignature := cxprogram.CXTypeSignatureAlloc()
		newCXTypeSignature.Name = cxArg.Name
		newCXTypeSignature.Package = cxArg.Package
		newCXTypeSignature.Offset = cxArg.Offset
		newCXTypeSignature.Type = TypeCXArgumentDeprecate
		newCXTypeSignature.Meta = int(cxArgIdx)
		newCXTypeSignature.ArgDetails = cxArg.ArgDetails
		cxTypeSignaturesIdxs = append(cxTypeSignaturesIdxs, newCXTypeSignature.Index)
	}
	return cxTypeSignaturesIdxs
}

// ----------------------------------------------------------------
//                             `CXProgram` Getters

func (cxprogram *CXProgram) OperationGet(expr *CXExpression) (*CXAtomicOperator, *CXLine, error) {
	switch expr.Type {
	case CXOperationTypeAtomicOperator:
		// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
		// Now an arena getter, site S0225.
		cxAtomicOp, err := cxprogram.CXAtomicOpGet(expr.Index)
		if err != nil {
			// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
			// Now a heap literal, site S0226.
			return &CXAtomicOperator{}, nil, err
		}

		return cxAtomicOp, nil, nil
	case CXOperationTypeLine:
		// OBJMODEL: state.CXLineMng.CXLineGet(gid), phase 5 #312.
		// Now an arena getter, site S0231.
		cxLine, err := cxprogram.CXLineGet(expr.Index)
		if err != nil {
			// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
			// Now a heap literal, site S0232.
			return &CXAtomicOperator{}, nil, err
		}

		// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
		// Now a heap literal, site S0235.
		return &CXAtomicOperator{}, cxLine, nil
		// case CXOperationTypeArgument:
	}

	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
	// Now a heap literal, site S0237.
	return &CXAtomicOperator{}, nil, CXProgramLookupError{CXProgramLookupErrorCodeOperationTypeMissing, 0}
}

func (cxprogram *CXProgram) PreviousCXLineGet(exprs []CXExpression, currIndex int) (*CXLine, error) {
	// A runtime error while the inputs of an empty function are copied stops at
	// line 0 of a call with no expressions.
	if currIndex >= len(exprs) {
		currIndex = len(exprs) - 1
	}
	for i := currIndex; i >= 0; i-- {
		if exprs[i].Type == CXOperationTypeLine {
			_, cxLine, err := cxprogram.OperationGet(&exprs[i])
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
	return &CXLine{}, CXProgramLookupError{CXProgramLookupErrorCodeLineMissing, 0}
}

func (cxprogram *CXProgram) CXAtomicOpFromExpressionsGet(exprs []CXExpression, currIndex int) (*CXAtomicOperator, error) {
	for i := currIndex; i < len(exprs); i++ {
		if exprs[i].Type == CXOperationTypeAtomicOperator {
			// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
			// Now an arena getter, site S0242.
			cxAtomicOp, err := cxprogram.CXAtomicOpGet(exprs[i].Index)
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

func (cxprogram *CXProgram) PreviousCXAtomicOpFromExpressionsGet(exprs []CXExpression, currIndex int) (*CXAtomicOperator, error) {
	for i := currIndex; i >= 0; i-- {
		if exprs[i].Type == CXOperationTypeAtomicOperator {
			// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
			// Now an arena getter, site S0245.
			cxAtomicOp, err := cxprogram.CXAtomicOpGet(exprs[i].Index)
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
func (cxprogram *CXProgram) SelectPackage(name stringpool.StringGid) (*CXPackage, error) {
	if _, ok := cxprogram.Packages[name]; !ok {
		return nil, fmt.Errorf("Package '%s' does not exist", cxprogram.CXProgramNameText(name))
	}

	cxprogram.CurrentPackage = cxprogram.Packages[name]

	// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
	// Now an arena getter, site S0248.
	return cxprogram.PackageFromArrayGet(cxprogram.Packages[name])
}

// CurrentPackageGet ...
func (cxprogram *CXProgram) CurrentPackageGet() (*CXPackage, error) {
	if cxprogram.CurrentPackage == -1 {
		return nil, errors.New("current package is nil")
	}

	// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
	// Now an arena getter, site S0249.
	return cxprogram.PackageFromArrayGet(cxprogram.CurrentPackage)
}

// CurrentFunctionGet ...
func (cxprogram *CXProgram) CurrentFunctionGet() (*CXFunction, error) {
	if cxprogram.CurrentPackage == -1 {
		return nil, errors.New("current package is nil")
	}

	// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
	// Now an arena getter, site S0250.
	currentPackage, err := cxprogram.PackageFromArrayGet(cxprogram.CurrentPackage)
	if err != nil {
		return nil, err
	}

	if currentPackage.CurrentFunction == CXFunctionGidNil {
		return nil, errors.New("current function is nil")
	}

	return cxprogram.CxIrState.CXFunctionMng.CXFunctionGet(currentPackage.CurrentFunction), nil

}

// PackageGet ...
func (cxprogram *CXProgram) PackageGet(pkgName stringpool.StringGid) (*CXPackage, error) {
	if _, ok := cxprogram.Packages[pkgName]; !ok {
		return nil, fmt.Errorf("package '%s' not found", cxprogram.CXProgramNameText(pkgName))
	}

	// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
	// Now an arena getter, site S0253.
	return cxprogram.PackageFromArrayGet(cxprogram.Packages[pkgName])
}

// StructGet ...
func (cxprogram *CXProgram) StructGet(strctName stringpool.StringGid, pkgName stringpool.StringGid) (*CXStruct, error) {
	pkg, err := cxprogram.PackageGet(pkgName)
	if err != nil {
		return nil, err
	}

	strct, err := pkg.StructGet(cxprogram, strctName)
	if err != nil {
		return nil, err
	}

	return strct, nil
}

// FunctionGet ...
func (cxprogram *CXProgram) FunctionGet(fnName stringpool.StringGid, pkgName stringpool.StringGid) (*CXFunction, error) {
	pkg, err := cxprogram.PackageGet(pkgName)
	if err != nil {
		return nil, err
	}

	fn, err := pkg.FunctionGet(cxprogram, fnName)
	if err != nil {
		return nil, err
	}

	return fn, nil
}

// CurrentCallGet returns the current CXCall
// TODO: What does this do?
// TODO: Only used in OpJmp
func (cxprogram *CXProgram) CurrentCallGet() *CXCall {
	return &cxprogram.CallStack[cxprogram.CallCounter]
}

// GlobalGet ...
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

// ProgramPrint prints the abstract syntax tree of a CX program in a
// human-readable format.
func (cxprogram *CXProgram) ProgramPrint() {
	fmt.Println(ToString(cxprogram))
}

// ProgramStdinInit makes reader the input of read() for the rest of the run.
func (prgrm *CXProgram) ProgramStdinInit(reader io.Reader) {
	prgrm.ProgramStdin = bufio.NewReader(reader)
}

// CXProgramFunctionLiveSlots returns which slots of the program's
// CXFunctionMng hold a live function, for CXProgramFunctionReleaseNew.
func (cxprogram *CXProgram) CXProgramFunctionLiveSlots() []bool {
	functionMng := &cxprogram.CxIrState.CXFunctionMng
	return append([]bool(nil), functionMng.CXFunctionMngLive[:functionMng.CXFunctionMngNext]...)
}

// CXProgramFunctionReleaseNew releases every function allocated since
// liveBefore was taken, the highest slot first, so the next Allocs take the
// slots back in ascending order.
func (cxprogram *CXProgram) CXProgramFunctionReleaseNew(liveBefore []bool) {
	functionMng := &cxprogram.CxIrState.CXFunctionMng
	for index := int(functionMng.CXFunctionMngNext) - 1; index >= 0; index-- {
		if functionMng.CXFunctionMngLive[index] && (index >= len(liveBefore) || !liveBefore[index]) {
			cxprogram.CXProgramFunctionRelease(cxprogram.CXProgramFunctionSlotGid(uint32(index)))
		}
	}
}
