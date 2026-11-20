package actions

import (
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// hasDeclSpec determines if an argument has certain declaration specifier.
func hasDeclSpec(arg *ast.CXArgument, spec int) bool {
	found := false
	for _, s := range arg.DeclarationSpecifiers {
		if s == spec {
			found = true
		}
	}
	return found
}

// hasDerefOp determines if an argument has certain dereference operation
func hasDerefOp(arg *ast.CXArgument, spec int) bool {
	found := false
	for _, s := range arg.DereferenceOperations {
		if s == spec {
			found = true
		}
	}
	return found
}

// WritePrimary writes those bytes to prgrm.Data and returns a CXArg.
func WritePrimary(prgrm *ast.CXProgram, typeCode types.Code, byts []byte, isSlice bool) *ast.CXArgument {
	pkg, err := prgrm.GetCurrentPackage()
	if err != nil {
		panic(err)
	}

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1052.
	arg := ast.MakeArgument(prgrm, "", CurrentFile, LineNo)
	arg.SetType(typeCode)
	arg.Package = ast.CXPackageIndex(pkg.Index)

	size := types.Cast_int_to_ptr(len(byts))

	arg.Size = typeCode.Size()
	arg.Offset = prgrm.Data.Size + prgrm.Data.StartsAt

	if arg.Type == types.STR || arg.Type == types.AFF {
		arg.Size = types.POINTER_SIZE
		if isSlice == false {

			types.Write_ptr(byts, 0, arg.Offset)
		}
	}

	// A CX program allocates min(INIT_HEAP_SIZE, MAX_HEAP_SIZE) bytes
	// after the stack segment. These bytes are used to allocate the data segment
	// at compile time. If the data segment is bigger than min(INIT_HEAP_SIZE, MAX_HEAP_SIZE),
	// we'll start appending the bytes to prgrm.Memory.
	// After compilation, we calculate how many bytes we need to add to have a heap segment
	// equal to `minHeapSize()` that is allocated after the data segment.
	memSize := types.Cast_int_to_ptr(len(prgrm.Memory))
	if (size + prgrm.Data.Size + prgrm.Data.StartsAt) > memSize {
		var i types.Pointer
		// First we need to fill the remaining free bytes in
		// the current `prgrm.Memory` slice.
		for i = types.Pointer(0); i < memSize-(prgrm.Data.Size+prgrm.Data.StartsAt); i++ {
			prgrm.Memory[prgrm.Data.Size+prgrm.Data.StartsAt+i] = byts[i]
		}
		// Then we append the bytes that didn't fit.
		prgrm.Memory = append(prgrm.Memory, byts[i:]...)
	} else {
		for i, byt := range byts {
			prgrm.Memory[prgrm.Data.Size+prgrm.Data.StartsAt+types.Cast_int_to_ptr(i)] = byt
		}
	}
	prgrm.Data.Size += size
	prgrm.Heap.StartsAt = prgrm.Data.Size + prgrm.Data.StartsAt

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now an arena insert, site S1053.
	argIdx := prgrm.AddCXArgInArray(arg)
	// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
	// Now an arena getter, site S1054.
	argReturn := prgrm.GetCXArgFromArray(argIdx)

	return argReturn
}

// WritePrimary writes those bytes to prgrm.Data and returns a []CXExpression.
func WritePrimaryExprs(prgrm *ast.CXProgram, typeCode types.Code, byts []byte, isSlice bool) []ast.CXExpression {
	pkg, err := prgrm.GetCurrentPackage()
	if err != nil {
		panic(err)
	}

	arg := WritePrimary(prgrm, typeCode, byts, isSlice)

	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S1055.
	expr := ast.MakeAtomicOperatorExpression(prgrm, nil)
	prgrm.CXAtomicOps[expr.Index].Package = ast.CXPackageIndex(pkg.Index)

	typeSigIdx := ast.AllocCXTypeSignatureOfCXArg(prgrm, arg)
	prgrm.CXAtomicOps[expr.Index].AddOutput(prgrm, typeSigIdx)

	// OBJMODEL: state.CXExpressionMng (manager storage) or []CXExpressionGid, phase 4 #346.
	// Now a slice of objects, site S1057.
	return []ast.CXExpression{*expr}
}

func AffordanceStructs(prgrm *ast.CXProgram, pkg *ast.CXPackage, currentFile string, lineNo int) {
	// Argument type
	// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
	// Now a constructor call, site S1058.
	argStrct := ast.MakeStruct("Argument")
	// argStrct.Size = cxcore.GetArgSize(cxcore.TYPE_STR) + cxcore.GetArgSize(cxcore.TYPE_STR)

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1059.
	argFldName := ast.MakeField(prgrm, "Name", types.STR, "", 0)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1060.
	argFldIndex := ast.MakeField(prgrm, "Index", types.I32, "", 0)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1061.
	argFldType := ast.MakeField(prgrm, "Type", types.STR, "", 0)

	argStrct.AddField(prgrm, argFldName)
	argStrct.AddField(prgrm, argFldIndex)
	argStrct.AddField(prgrm, argFldType)

	pkg.AddStruct(prgrm, argStrct)

	// Expression type
	// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
	// Now a constructor call, site S1062.
	exprStrct := ast.MakeStruct("Expression")
	// exprStrct.Size = cxcore.GetArgSize(cxcore.TYPE_STR)

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1063.
	exprFldOperator := ast.MakeField(prgrm, "Operator", types.STR, "", 0)

	exprStrct.AddField(prgrm, exprFldOperator)

	pkg.AddStruct(prgrm, exprStrct)

	// Function type
	// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
	// Now a constructor call, site S1064.
	fnStrct := ast.MakeStruct("Function")
	// fnStrct.Size = cxcore.GetArgSize(cxcore.TYPE_STR) + cxcore.GetArgSize(cxcore.TYPE_STR) + cxcore.GetArgSize(cxcore.TYPE_STR)

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1065.
	fnFldName := ast.MakeField(prgrm, "Name", types.STR, "", 0)

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1066.
	fnFldInpSig := ast.MakeField(prgrm, "InputSignature", types.STR, "", 0)
	fnFldInpSig.Size = types.STR.Size()
	fnFldInpSig = DeclarationSpecifiers(fnFldInpSig, []types.Pointer{0}, constants.DECL_SLICE)

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1067.
	fnFldOutSig := ast.MakeField(prgrm, "OutputSignature", types.STR, "", 0)
	fnFldOutSig.Size = types.STR.Size()
	fnFldOutSig = DeclarationSpecifiers(fnFldOutSig, []types.Pointer{0}, constants.DECL_SLICE)

	fnStrct.AddField(prgrm, fnFldName)
	fnStrct.AddField(prgrm, fnFldInpSig)

	fnStrct.AddField(prgrm, fnFldOutSig)

	pkg.AddStruct(prgrm, fnStrct)

	// Structure type
	// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
	// Now a constructor call, site S1068.
	strctStrct := ast.MakeStruct("Structure")
	// strctStrct.Size = cxcore.GetArgSize(cxcore.TYPE_STR)

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1069.
	strctFldName := ast.MakeField(prgrm, "Name", types.STR, "", 0)

	strctStrct.AddField(prgrm, strctFldName)

	pkg.AddStruct(prgrm, strctStrct)

	// Package type
	// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
	// Now a constructor call, site S1070.
	pkgStrct := ast.MakeStruct("Structure")
	// pkgStrct.Size = cxcore.GetArgSize(cxcore.TYPE_STR)

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1071.
	pkgFldName := ast.MakeField(prgrm, "Name", types.STR, "", 0)

	pkgStrct.AddField(prgrm, pkgFldName)

	pkg.AddStruct(prgrm, pkgStrct)

	// Caller type
	// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
	// Now a constructor call, site S1072.
	callStrct := ast.MakeStruct("Caller")
	// callStrct.Size = cxcore.GetArgSize(cxcore.TYPE_STR) + cxcore.GetArgSize(cxcore.TYPE_I32)

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1073.
	callFldFnName := ast.MakeField(prgrm, "FnName", types.STR, "", 0)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1074.
	callFldFnSize := ast.MakeField(prgrm, "FnSize", types.I32, "", 0)

	callStrct.AddField(prgrm, callFldFnName)
	callStrct.AddField(prgrm, callFldFnSize)

	pkg.AddStruct(prgrm, callStrct)

	// Program type
	// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
	// Now a constructor call, site S1075.
	prgrmStrct := ast.MakeStruct("Program")
	// prgrmStrct.Size = cxcore.GetArgSize(cxcore.TYPE_I32) + cxcore.GetArgSize(cxcore.TYPE_I64)

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1076.
	prgrmFldCallCounter := ast.MakeField(prgrm, "CallCounter", types.I32, "", 0)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1077.
	prgrmFldFreeHeap := ast.MakeField(prgrm, "HeapUsed", types.I64, "", 0)

	// prgrmFldCaller := cxcore.MakeField("Caller", cxcore.TYPE_STRUCT, "", 0)
	// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
	// Now an arena getter, site S1078.
	strctPkg, err := prgrm.GetPackageFromArray(callStrct.Package)
	if err != nil {
		panic(err)
	}
	prgrmFldCaller := DeclarationSpecifiersStruct(prgrm, callStrct.Name, strctPkg.Name, false, currentFile, lineNo)
	prgrmFldCaller.Name = "Caller"

	prgrmStrct.AddField(prgrm, prgrmFldCallCounter)
	prgrmStrct.AddField(prgrm, prgrmFldFreeHeap)
	prgrmStrct.AddField(prgrm, prgrmFldCaller)

	pkg.AddStruct(prgrm, prgrmStrct)
}

// PrimaryIdentifier creates an identifier expression with an output name of 'ident'.
func PrimaryIdentifier(prgrm *ast.CXProgram, ident string) []ast.CXExpression {
	pkg, err := prgrm.GetCurrentPackage()
	if err != nil {
		panic(err)
	}

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1079.
	arg := ast.MakeArgument(prgrm, ident, CurrentFile, LineNo) // fix: line numbers in errors sometimes report +1 or -1. Issue #195
	arg.SetType(types.IDENTIFIER)
	arg.Name = ident
	arg.Package = ast.CXPackageIndex(pkg.Index)

	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S1080.
	expr := ast.MakeAtomicOperatorExpression(prgrm, nil)
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
	// Now an arena getter, site S1081.
	expression, err := prgrm.GetCXAtomicOp(expr.Index)
	if err != nil {
		panic(err)
	}

	typeSigIdx := ast.AllocCXTypeSignatureOfCXArg(prgrm, arg)
	expression.AddOutput(prgrm, typeSigIdx)
	expression.Package = ast.CXPackageIndex(pkg.Index)
	// OBJMODEL: state.CXExpressionMng (manager storage) or []CXExpressionGid, phase 4 #346.
	// Now a slice of objects, site S1083.
	return []ast.CXExpression{*expr}
}

// IsAllArgsBasicTypes checks if all the input arguments in an expressions are of basic type.
func IsAllArgsBasicTypes(prgrm *ast.CXProgram, expr *ast.CXExpression) bool {
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
	// Now an arena getter, site S1084.
	expression, err := prgrm.GetCXAtomicOp(expr.Index)
	if err != nil {
		panic(err)
	}

	for _, inputIdx := range expression.GetInputs(prgrm) {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S1085.
		input := prgrm.GetCXTypeSignatureFromArray(inputIdx)

		var inpType types.Code

		var inp *ast.CXArgument
		if input.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1087.
			inp = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(input.Meta))

			inpType = inp.Type
			if inp.Type == types.POINTER {
				inpType = inp.PointerTargetType
			}
		} else {
			inpType = input.GetType(prgrm)
		}

		// TODO: Check why STR is considered as basic type.
		if !inpType.IsPrimitive() && inpType != types.STR {
			return false
		}
	}
	return true
}
