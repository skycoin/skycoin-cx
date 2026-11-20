package actions

import (
	"fmt"
	"os"
	"strings"

	cxpackages "github.com/skycoin/skycoin-cx/t0-packages" // thin: only IsDefaultPackage, SysPrefix; no depth gain
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// PostfixExpressionArray handles the postfix expression arrays.
//
// Input arguments description:
//
//		prgrm - a CXProgram that contains all the data and arrays of the program.
//		prevExprs - the previous expressions that compose the postfix expression.
//	 postExprs - the array expressions.
func PostfixExpressionArray(prgrm *ast.CXProgram, prevExprs []ast.CXExpression, postExprs []ast.CXExpression) []ast.CXExpression {
	var elt *ast.CXArgument

	prevExpressionIdx := prevExprs[len(prevExprs)-1].Index
	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S1094.
	prevExpressionOperator := prgrm.GetFunctionFromArray(prgrm.CXAtomicOps[prevExpressionIdx].Operator)
	prevExprCXLine, _ := prgrm.GetPreviousCXLine(prevExprs, len(prevExprs)-1)

	if prevExpressionOperator != nil && len(prgrm.CXAtomicOps[prevExpressionIdx].GetOutputs(prgrm)) == 0 {
		genName := generateTempVarName(constants.LOCAL_PREFIX)

		// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
		// Now a constructor call, site S1095.
		useExprCXLine := ast.MakeCXLineExpression(prgrm, prevExprCXLine.FileName, prevExprCXLine.LineNumber, prevExprCXLine.LineStr)
		// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
		// Now a constructor call, site S1096.
		useExpr := ast.MakeAtomicOperatorExpression(prgrm, nil)
		useExpressionIdx := useExpr.Index
		prgrm.CXAtomicOps[useExpressionIdx].Package = prgrm.CXAtomicOps[prevExpressionIdx].Package

		prevExpressionOperatorOutputs := prevExpressionOperator.GetOutputs(prgrm)
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S1097.
		prevExpressionOperatorOutputTypeSig := prgrm.GetCXTypeSignatureFromArray(prevExpressionOperatorOutputs[0])
		if prevExpressionOperatorOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1098.
			prevExpressionOperatorOutputArg := prgrm.GetCXArgFromArray(ast.CXArgumentIndex(prevExpressionOperatorOutputTypeSig.Meta))

			prevExpressionOperatorOutput := prevExpressionOperatorOutputArg
			// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
			// Now a constructor call, site S1099.
			out := ast.MakeArgument(prgrm, genName, prevExprCXLine.FileName, prevExprCXLine.LineNumber-1).SetType(prevExpressionOperatorOutput.Type)

			out.DeclarationSpecifiers = prevExpressionOperatorOutput.DeclarationSpecifiers
			out.StructType = prevExpressionOperatorOutput.StructType
			out.Size = prevExpressionOperatorOutput.Size
			out.Lengths = prevExpressionOperatorOutput.Lengths
			out.PreviouslyDeclared = true
			out.Package = prevExpressionOperatorOutput.Package

			typeSigIdx := ast.AllocCXTypeSignatureOfCXArg(prgrm, out)
			prgrm.CXAtomicOps[prevExpressionIdx].AddOutput(prgrm, typeSigIdx)

			// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
			// Now a constructor call, site S1101.
			inp := ast.MakeArgument(prgrm, genName, prevExprCXLine.FileName, prevExprCXLine.LineNumber).SetType(prevExpressionOperatorOutput.Type)

			inp.DeclarationSpecifiers = prevExpressionOperatorOutput.DeclarationSpecifiers
			inp.StructType = prevExpressionOperatorOutput.StructType
			inp.Size = prevExpressionOperatorOutput.Size
			inp.Lengths = prevExpressionOperatorOutput.Lengths
			inp.PreviouslyDeclared = true

			typeSigIdx = ast.AllocCXTypeSignatureOfCXArg(prgrm, inp)
			prgrm.CXAtomicOps[useExpressionIdx].AddOutput(prgrm, typeSigIdx)
		} else if prevExpressionOperatorOutputTypeSig.Type == ast.TYPE_SLICE_ATOMIC {
			// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
			// Now an arena getter, site S1103.
			sliceDetails := prgrm.GetCXTypeSignatureArrayFromArray(prevExpressionOperatorOutputTypeSig.Meta)

			newTypeSigSliceGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
			newTypeSigSlice := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(newTypeSigSliceGid)
			newTypeSigSlice.Meta = sliceDetails.Meta
			newTypeSigSlice.Lengths = sliceDetails.Lengths
			newTypeSigSliceIdx := int(objmodel.GidIndex(int64(newTypeSigSliceGid)))

			argDetailsGid := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugAlloc()
			argDetails := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugGet(argDetailsGid)
			argDetails.FileName = prevExprCXLine.FileName
			argDetails.FileLine = prevExprCXLine.LineNumber - 1
			_, newTypeSig := prgrm.AllocCXTypeSignature()
			newTypeSig.ArgDetails = argDetailsGid
			newTypeSig.Name = genName
			newTypeSig.Package = prevExpressionOperatorOutputTypeSig.Package
			newTypeSig.Type = prevExpressionOperatorOutputTypeSig.Type
			newTypeSig.Meta = newTypeSigSliceIdx

			outNewTypeSigIdx := newTypeSig.Index
			prgrm.CXAtomicOps[prevExpressionIdx].AddOutput(prgrm, outNewTypeSigIdx)

			inpNewTypeSigSliceGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
			inpNewTypeSigSlice := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(inpNewTypeSigSliceGid)
			inpNewTypeSigSlice.Meta = sliceDetails.Meta
			inpNewTypeSigSlice.Lengths = sliceDetails.Lengths
			inpNewTypeSigSliceIdx := int(objmodel.GidIndex(int64(inpNewTypeSigSliceGid)))

			inpArgDetailsGid := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugAlloc()
			inpArgDetails := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugGet(inpArgDetailsGid)
			inpArgDetails.FileName = prevExprCXLine.FileName
			inpArgDetails.FileLine = prevExprCXLine.LineNumber - 1
			_, inpNewTypeSig := prgrm.AllocCXTypeSignature()
			inpNewTypeSig.ArgDetails = inpArgDetailsGid
			inpNewTypeSig.Name = genName
			inpNewTypeSig.Package = prevExpressionOperatorOutputTypeSig.Package
			inpNewTypeSig.Type = prevExpressionOperatorOutputTypeSig.Type
			inpNewTypeSig.Meta = inpNewTypeSigSliceIdx
			inpNewTypeSigIdx := inpNewTypeSig.Index
			prgrm.CXAtomicOps[useExpressionIdx].AddOutput(prgrm, inpNewTypeSigIdx)
		} else {
			panic("type is not known\n")
		}

		prevExprs = append(prevExprs, *useExprCXLine, *useExpr)
	}

	prevExpression, err := prgrm.GetCXAtomicOpFromExpressions(prevExprs, len(prevExprs)-1)
	if err != nil {
		panic(err)
	}
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now an arena getter, site S1114.
	prevExpressionOutputTypeSig := prgrm.GetCXTypeSignatureFromArray(prevExpression.GetOutputs(prgrm)[0])

	// nil for a slice output, which has no fields.
	var prevExpressionOutputArg *ast.CXArgument
	if prevExpressionOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S1116.
		prevExpressionOutputArg = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(prevExpressionOutputTypeSig.Meta))

		if len(prevExpressionOutputArg.Fields) > 0 {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1117.
			elt = prgrm.GetCXArgFromArray(prevExpressionOutputArg.Fields[len(prevExpressionOutputArg.Fields)-1])
		} else {
			elt = prevExpressionOutputArg
		}

		// elt.IsArray = false
		elt.DereferenceOperations = append(elt.DereferenceOperations, constants.DEREF_ARRAY)
		elt.DeclarationSpecifiers = append(elt.DeclarationSpecifiers, constants.DECL_INDEXING)

	} else if prevExpressionOutputTypeSig.Type == ast.TYPE_ATOMIC || prevExpressionOutputTypeSig.Type == ast.TYPE_POINTER_ATOMIC {
		// TODO: implement correct way when it is possible
		// temporarily return back to cx arg deprecate
		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now a constructor call, site S1118.
		arg := ast.MakeArgument(prgrm, prevExpressionOutputTypeSig.Name, CurrentFile, LineNo) // fix: line numbers in errors sometimes report +1 or -1. Issue #195
		arg.SetType(types.IDENTIFIER)
		arg.Name = prevExpressionOutputTypeSig.Name
		arg.Package = prevExpressionOutputTypeSig.Package
		arg.PassBy = prevExpressionOutputTypeSig.PassBy

		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now an arena insert, site S1119.
		argIdx := prgrm.AddCXArgInArray(arg)
		prevExpressionOutputTypeSig.Type = ast.TYPE_CXARGUMENT_DEPRECATE
		prevExpressionOutputTypeSig.Meta = int(argIdx)

		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S1120.
		prevExpressionOutputArg = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(prevExpressionOutputTypeSig.Meta))

		elt = prevExpressionOutputArg
		// elt.IsArray = false
		elt.DereferenceOperations = append(elt.DereferenceOperations, constants.DEREF_ARRAY)
		elt.DeclarationSpecifiers = append(elt.DeclarationSpecifiers, constants.DECL_INDEXING)
	} else if prevExpressionOutputTypeSig.Type == ast.TYPE_SLICE_ATOMIC {
		prevExpressionOutputTypeSig.IsDeref = true
	} else {
		panic("type is not known")
	}

	postExpressionIdx := postExprs[len(postExprs)-1].Index
	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S1123.
	postExpressionOperator := prgrm.GetFunctionFromArray(prgrm.CXAtomicOps[postExpressionIdx].Operator)
	postExpressionOperatorOutputs := postExpressionOperator.GetOutputs(prgrm)

	if prevExpressionOutputArg != nil && len(prevExpressionOutputArg.Fields) > 0 {
		fldIdx := prevExpressionOutputArg.Fields[len(prevExpressionOutputArg.Fields)-1]

		if postExpressionOperator == nil {
			postExpressionOutputIndex := prgrm.CXAtomicOps[postExpressionIdx].GetOutputs(prgrm)[0]
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1756.
			prgrm.GetCXArg(ast.CXArgumentIndex(fldIdx)).Indexes = append(prgrm.GetCXArg(ast.CXArgumentIndex(fldIdx)).Indexes, postExpressionOutputIndex)
		} else {
			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
			// Now an arena getter, site S1124.
			postExpressionOperatorOutputTypeSig := prgrm.GetCXTypeSignatureFromArray(postExpressionOperatorOutputs[0])
			var postExpressionOperatorOutputArg *ast.CXArgument
			if postExpressionOperatorOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S1126.
				postExpressionOperatorOutputArg = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(postExpressionOperatorOutputTypeSig.Meta))
			} else {
				panic("type is not cx argument deprecate\n\n")
			}

			// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
			// Now a constructor call, site S1127.
			sym := ast.MakeArgument(prgrm, generateTempVarName(constants.LOCAL_PREFIX), CurrentFile, LineNo).SetType(postExpressionOperatorOutputArg.Type)
			sym.Package = prgrm.CXAtomicOps[postExpressionIdx].Package
			sym.PreviouslyDeclared = true

			typeSigIdx := ast.AllocCXTypeSignatureOfCXArg(prgrm, sym)
			prgrm.CXAtomicOps[postExpressionIdx].AddOutput(prgrm, typeSigIdx)

			prevExprs = append(postExprs, prevExprs...)

			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1757.
			prgrm.GetCXArg(ast.CXArgumentIndex(fldIdx)).Indexes = append(prgrm.GetCXArg(ast.CXArgumentIndex(fldIdx)).Indexes, typeSigIdx)
		}
	} else {
		if len(prgrm.CXAtomicOps[postExpressionIdx].GetOutputs(prgrm)) < 1 {
			// then it's an expression (e.g. i32.add(0, 0))
			// we create a gensym for it

			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
			// Now an arena getter, site S1129.
			postExpressionOperatorOutputTypeSig := prgrm.GetCXTypeSignatureFromArray(postExpressionOperatorOutputs[0])
			var typeSigIdx ast.CXTypeSignatureIndex
			if postExpressionOperatorOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S1130.
				postExpressionOperatorOutputArg := prgrm.GetCXArgFromArray(ast.CXArgumentIndex(postExpressionOperatorOutputTypeSig.Meta))
				// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
				// Now a constructor call, site S1131.
				idxSym := ast.MakeArgument(prgrm, generateTempVarName(constants.LOCAL_PREFIX), CurrentFile, LineNo).SetType(postExpressionOperatorOutputArg.Type)
				idxSym.Size = postExpressionOperatorOutputArg.Size

				idxSym.Package = prgrm.CXAtomicOps[postExpressionIdx].Package
				idxSym.PreviouslyDeclared = true

				typeSigIdx = ast.AllocCXTypeSignatureOfCXArg(prgrm, idxSym)
			} else if postExpressionOperatorOutputTypeSig.Type == ast.TYPE_ATOMIC {
				argDetailsGid := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugAlloc()
				argDetails := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugGet(argDetailsGid)
				argDetails.FileName = CurrentFile
				argDetails.FileLine = LineNo
				_, newTypeSig := prgrm.AllocCXTypeSignature()
				newTypeSig.ArgDetails = argDetailsGid
				newTypeSig.Name = generateTempVarName(constants.LOCAL_PREFIX)
				newTypeSig.Package = prgrm.CXAtomicOps[postExpressionIdx].Package
				newTypeSig.Type = postExpressionOperatorOutputTypeSig.Type
				newTypeSig.Meta = postExpressionOperatorOutputTypeSig.Meta
				newTypeSig.Offset = postExpressionOperatorOutputTypeSig.Offset
				typeSigIdx = newTypeSig.Index
			} else if postExpressionOperatorOutputTypeSig.Type == ast.TYPE_POINTER_ATOMIC {
				argDetailsGid := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugAlloc()
				argDetails := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugGet(argDetailsGid)
				argDetails.FileName = CurrentFile
				argDetails.FileLine = LineNo
				_, newTypeSig := prgrm.AllocCXTypeSignature()
				newTypeSig.ArgDetails = argDetailsGid
				newTypeSig.Name = generateTempVarName(constants.LOCAL_PREFIX)
				newTypeSig.Package = prgrm.CXAtomicOps[postExpressionIdx].Package
				newTypeSig.Type = postExpressionOperatorOutputTypeSig.Type
				newTypeSig.Meta = postExpressionOperatorOutputTypeSig.Meta
				newTypeSig.Offset = postExpressionOperatorOutputTypeSig.Offset
				typeSigIdx = newTypeSig.Index
			} else {
				panic("type is not known")
			}

			prgrm.CXAtomicOps[postExpressionIdx].AddOutput(prgrm, typeSigIdx)

			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
			// Now an arena getter, site S1139.
			prevExpressionOutputTypeSig := prgrm.GetCXTypeSignatureFromArray(prevExpression.GetOutputs(prgrm)[0])
			var prevExpressionOutputArg *ast.CXArgument
			if prevExpressionOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S1141.
				prevExpressionOutputArg = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(prevExpressionOutputTypeSig.Meta))
			} else {
				panic("type is not cx argument deprecate\n\n")
			}

			prevExpressionOutput := prevExpressionOutputArg
			prevExpressionOutput.Indexes = append(prevExpressionOutput.Indexes, typeSigIdx)

			// we push the index expression
			prevExprs = append(postExprs, prevExprs...)
		} else {
			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
			// Now an arena getter, site S1142.
			prevOutsTypeSig := prgrm.GetCXTypeSignatureFromArray(prevExpression.GetOutputs(prgrm)[0])
			if prevOutsTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
				prevOutsIdx := ast.CXArgumentIndex(prevOutsTypeSig.Meta)

				postOutsIndex := prgrm.CXAtomicOps[postExpressionIdx].GetOutputs(prgrm)[0]
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S1758.
				prgrm.GetCXArg(ast.CXArgumentIndex(prevOutsIdx)).Indexes = append(prgrm.GetCXArg(ast.CXArgumentIndex(prevOutsIdx)).Indexes, postOutsIndex)
			} else if prevOutsTypeSig.Type == ast.TYPE_ATOMIC || prevOutsTypeSig.Type == ast.TYPE_POINTER_ATOMIC {
				panic("type is not cx argument deprecate\n\n")
			} else if prevOutsTypeSig.Type == ast.TYPE_SLICE_ATOMIC {
				// OBJMODEL: state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(gid), phase 5 #317.
				// Now an arena getter, site S1143.
				sliceDetails := prgrm.GetCXTypeSignatureArrayFromArray(prevOutsTypeSig.Meta)

				postOutsIndex := prgrm.CXAtomicOps[postExpressionIdx].GetOutputs(prgrm)[0]
				sliceDetails.Indexes = append(sliceDetails.Indexes, postOutsIndex)
			} else {
				panic("type is not known")
			}

		}
	}

	return prevExprs
}

// PostfixExpressionNative handles native postfix expression function calls.
//
// Input arguments description:
//
//		prgrm - a CXProgram that contains all the data and arrays of the program.
//		typeCode - type code of the native expression.
//	 opStrCode - name of the function to be called that belongs to the native
//					expression.
func PostfixExpressionNative(prgrm *ast.CXProgram, typeCode types.Code, opStrCode string) []ast.CXExpression {
	// these will always be native functions
	opCode, ok := ast.OpCodes[typeCode.Name()+"."+opStrCode]
	if !ok {
		if msg, renamed := renamedOpcodeMessage(typeCode.Name() + "." + opStrCode); renamed {
			println(ast.CompilationError(CurrentFile, LineNo), msg)
			return nil
		}
		println(ast.CompilationError(CurrentFile, LineNo) + " function '" +
			typeCode.Name() + "." + opStrCode + "' does not exist")
		return nil
		// panic(ok)
	}

	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S1144.
	exprCXLine := ast.MakeCXLineExpression(prgrm, CurrentFile, LineNo, LineStr)
	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S1145.
	expr := ast.MakeAtomicOperatorExpression(prgrm, ast.Natives[opCode])
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
	// Now an arena getter, site S1146.
	expression, err := prgrm.GetCXAtomicOp(expr.Index)
	if err != nil {
		panic(err)
	}

	pkg, err := prgrm.GetCurrentPackage()
	if err != nil {
		panic(err)
	}
	expression.Package = ast.CXPackageIndex(pkg.Index)
	// OBJMODEL: state.CXExpressionMng (manager storage) or []CXExpressionGid, phase 4 #346.
	// Now a slice of objects, site S1147.
	return []ast.CXExpression{*exprCXLine, *expr}
}

// PostfixExpressionEmptyFunCall handles postfix expression empty function calls
// or the function calls that doesn't have any input args.
//
// Input arguments description:
//
//	prgrm - a CXProgram that contains all the data and arrays of the program.
//	prevExprs - the previous expressions that compose the postfix expression.
func PostfixExpressionEmptyFunCall(prgrm *ast.CXProgram, prevExprs []ast.CXExpression) []ast.CXExpression {
	prevExpression, err := prgrm.GetCXAtomicOpFromExpressions(prevExprs, len(prevExprs)-1)
	if err != nil {
		panic(err)
	}

	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S1148.
	prevExpressionOperator := prgrm.GetFunctionFromArray(prevExpression.Operator)

	var firstPrevExpressionIdx int
	for i := 0; i < len(prevExprs); i++ {
		if prevExprs[i].Type == ast.CX_ATOMIC_OPERATOR {
			firstPrevExpressionIdx = prevExprs[i].Index
			break
		}
	}

	// nil when there is no output or it is not an argument.
	var prevExpressionOutputArg *ast.CXArgument
	if prevExpression.Outputs != nil {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S1150.
		prevExpressionOutputTypeSig := prgrm.GetCXTypeSignatureFromArray(prevExpression.GetOutputs(prgrm)[0])
		if prevExpression.Outputs != nil {
			if prevExpressionOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S1151.
				prevExpressionOutputArg = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(prevExpressionOutputTypeSig.Meta))
			}
		}
	}

	if prevExpressionOutputArg != nil && len(prevExpressionOutputArg.Fields) > 0 {
		// then it's a method call or function in field

	} else if prevExpressionOperator == nil {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S1153.
		prevExpressionOutputTypeSig := prgrm.GetCXTypeSignatureFromArray(prevExpression.GetOutputs(prgrm)[0])
		if opCode, ok := ast.OpCodes[prevExpressionOutputTypeSig.Name]; ok {
			if pkg, err := prgrm.GetCurrentPackage(); err == nil {
				prgrm.CXAtomicOps[firstPrevExpressionIdx].Package = ast.CXPackageIndex(pkg.Index)
			}
			prgrm.CXAtomicOps[firstPrevExpressionIdx].Outputs = nil
			// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
			// Now an arena insert, site S1154.
			opIdx := prgrm.AddNativeFunctionInArray(ast.Natives[opCode])
			prgrm.CXAtomicOps[firstPrevExpressionIdx].Operator = opIdx
		}

		prgrm.CXAtomicOps[firstPrevExpressionIdx].Inputs = nil
	}

	return FunctionCall(prgrm, prevExprs, nil)
}

// PostfixExpressionFunCall handles postfix expression function calls.
//
// Input arguments description:
//
//		prgrm - a CXProgram that contains all the data and arrays of the program.
//		prevExprs - the previous expressions that compose the postfix expression.
//	 args - input args list for the function call.
func PostfixExpressionFunCall(prgrm *ast.CXProgram, prevExprs []ast.CXExpression, args []ast.CXExpression) []ast.CXExpression {
	// An unresolved callee has already been reported as a compile error.
	if len(prevExprs) == 0 {
		return nil
	}

	lastPrevExpression, err := prgrm.GetCXAtomicOpFromExpressions(prevExprs, len(prevExprs)-1)
	if err != nil {
		panic(err)
	}
	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S1155.
	lastPrevExpressionOperator := prgrm.GetFunctionFromArray(lastPrevExpression.Operator)

	firstPrevExpressionIdx := prevExprs[0].Index

	var lastPrevExpressionOutputTypeSig *ast.CXTypeSignature

	// nil when there is no output or it is atomic.
	var lastPrevExpressionOutputArg *ast.CXArgument
	if lastPrevExpression.Outputs != nil {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S1157.
		lastPrevExpressionOutputTypeSig = prgrm.GetCXTypeSignatureFromArray(lastPrevExpression.GetOutputs(prgrm)[0])

		if lastPrevExpressionOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1158.
			lastPrevExpressionOutputArg = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(lastPrevExpressionOutputTypeSig.Meta))
		} else if lastPrevExpressionOutputTypeSig.Type == ast.TYPE_ATOMIC || lastPrevExpressionOutputTypeSig.Type == ast.TYPE_POINTER_ATOMIC {
			// do nothing
		} else {
			panic("type is not known")
		}
	}

	if lastPrevExpressionOutputArg != nil && len(lastPrevExpressionOutputArg.Fields) > 0 {
		// then it's a method
		// prevExprs[len(prevExprs) - 1].IsMethodCall = true

	} else if lastPrevExpressionOperator == nil {
		if opCode, ok := ast.OpCodes[lastPrevExpressionOutputTypeSig.Name]; ok {
			if pkg, err := prgrm.GetCurrentPackage(); err == nil {
				prgrm.CXAtomicOps[firstPrevExpressionIdx].Package = ast.CXPackageIndex(pkg.Index)
			}
			prgrm.CXAtomicOps[firstPrevExpressionIdx].Outputs = nil
			// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
			// Now an arena insert, site S1159.
			opIdx := prgrm.AddNativeFunctionInArray(ast.Natives[opCode])
			prgrm.CXAtomicOps[firstPrevExpressionIdx].Operator = opIdx
		}

		prgrm.CXAtomicOps[firstPrevExpressionIdx].Inputs = nil
	}

	return FunctionCall(prgrm, prevExprs, args)
}

// PostfixExpressionIncDec handles the incrementing or decrementing of the
// postfix expression.
//
// Input arguments description:
//
//		prgrm - a CXProgram that contains all the data and arrays of the program.
//		prevExprs - the previous expressions that compose the postfix expression.
//	 isInc - true if expression is to be incremented, false if to be decremented.
func PostfixExpressionIncDec(prgrm *ast.CXProgram, prevExprs []ast.CXExpression, isInc bool) []ast.CXExpression {
	pkg, err := prgrm.GetCurrentPackage()
	if err != nil {
		panic(err)
	}

	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S1160.
	exprCXLine := ast.MakeCXLineExpression(prgrm, CurrentFile, LineNo, LineStr)
	var expr *ast.CXExpression
	if isInc {
		// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
		// Now a constructor call, site S1161.
		expr = ast.MakeAtomicOperatorExpression(prgrm, ast.Natives[constants.OP_ADD])
	} else {
		// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
		// Now a constructor call, site S1162.
		expr = ast.MakeAtomicOperatorExpression(prgrm, ast.Natives[constants.OP_SUB])
	}

	var valB [4]byte
	types.Write_i32(valB[:], 0, 1)
	valArg := WritePrimary(prgrm, types.I32, valB[:], false)

	lastPrevExpression, err := prgrm.GetCXAtomicOpFromExpressions(prevExprs, len(prevExprs)-1)
	if err != nil {
		panic(err)
	}

	expressionIdx := expr.Index
	prgrm.CXAtomicOps[expressionIdx].Package = ast.CXPackageIndex(pkg.Index)

	typeSigIdx := lastPrevExpression.GetOutputs(prgrm)[0]
	prgrm.CXAtomicOps[expressionIdx].AddInput(prgrm, typeSigIdx)

	typeSigIdx = ast.AllocCXTypeSignatureOfCXArg(prgrm, valArg)
	prgrm.CXAtomicOps[expressionIdx].AddInput(prgrm, typeSigIdx)

	typeSigIdx = lastPrevExpression.GetOutputs(prgrm)[0]
	prgrm.CXAtomicOps[expressionIdx].AddOutput(prgrm, typeSigIdx)

	// OBJMODEL: state.CXExpressionMng (manager storage) or []CXExpressionGid, phase 4 #346.
	// Now a slice of objects, site S1164.
	exprs := append([]ast.CXExpression{}, *exprCXLine, *expr)
	return exprs
}

// PostfixExpressionField handles the dot notation that can be followed by an identifier.
// Examples are: `foo.bar`, `foo().bar`, `pkg.foo`
//
// Input arguments description:
//
//		prgrm - a CXProgram that contains all the data and arrays of the program.
//		prevExprs - the previous expressions that compose the left expression
//					i.e. 'foo' in the example above.
//	 ident - field name or the identifier after the dot notation.
func PostfixExpressionField(prgrm *ast.CXProgram, prevExprs []ast.CXExpression, ident string) []ast.CXExpression {
	lastExpr := prevExprs[len(prevExprs)-1]

	lastExpressionIdx := lastExpr.Index
	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S1165.
	lastExpressionOperator := prgrm.GetFunctionFromArray(prgrm.CXAtomicOps[lastExpressionIdx].Operator)

	lastExprCXLine, _ := prgrm.GetPreviousCXLine(prevExprs, len(prevExprs)-1)

	// Then it's a function call, e.g. foo().fld
	// and we need to create some auxiliary variables to hold the result from
	// the function call
	if lastExpressionOperator != nil {
		lastExpressionOperatorOutputs := lastExpressionOperator.GetOutputs(prgrm)
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S1166.
		lastExpressionOperatorOutputTypeSig := prgrm.GetCXTypeSignatureFromArray(lastExpressionOperatorOutputs[0])

		symName := generateTempVarName(constants.LOCAL_PREFIX)
		if lastExpressionOperatorOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1167.
			lastExpressionOperatorOutputArg := prgrm.GetCXArgFromArray(ast.CXArgumentIndex(lastExpressionOperatorOutputTypeSig.Meta))

			opOut := lastExpressionOperatorOutputArg

			// we associate the result of the function call to the aux variable
			// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
			// Now a constructor call, site S1168.
			out := ast.MakeArgument(prgrm, symName, lastExprCXLine.FileName, lastExprCXLine.LineNumber)
			out.SetType(opOut.Type)
			out.DeclarationSpecifiers = opOut.DeclarationSpecifiers
			out.StructType = opOut.StructType
			out.Size = opOut.Size
			// out.IsArray = opOut.IsArray
			// out.IsReference = opOut.IsReference
			out.Lengths = opOut.Lengths
			out.Package = prgrm.CXAtomicOps[lastExpressionIdx].Package
			out.PreviouslyDeclared = true

			typeSigIdx := ast.AllocCXTypeSignatureOfCXArg(prgrm, out)
			prgrm.CXAtomicOps[lastExpressionIdx].AddOutput(prgrm, typeSigIdx)

			// we need to create an expression to hold all the modifications
			// that will take place after this if statement
			// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
			// Now a constructor call, site S1170.
			inp := ast.MakeArgument(prgrm, symName, lastExprCXLine.FileName, lastExprCXLine.LineNumber)
			inp.SetType(opOut.Type)
			inp.DeclarationSpecifiers = opOut.DeclarationSpecifiers
			inp.StructType = opOut.StructType
			inp.Size = opOut.Size
			inp.Package = prgrm.CXAtomicOps[lastExpressionIdx].Package

			// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
			// Now a constructor call, site S1171.
			exprCXLine := ast.MakeCXLineExpression(prgrm, lastExprCXLine.FileName, lastExprCXLine.LineNumber, lastExprCXLine.LineStr)
			// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
			// Now a constructor call, site S1172.
			expr := ast.MakeAtomicOperatorExpression(prgrm, nil)

			// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
			// Now an arena getter, site S1173.
			expression, err := prgrm.GetCXAtomicOp(expr.Index)
			if err != nil {
				panic(err)
			}

			expression.Package = prgrm.CXAtomicOps[lastExpressionIdx].Package
			typeSigIdx = ast.AllocCXTypeSignatureOfCXArg(prgrm, inp)
			expression.AddOutput(prgrm, typeSigIdx)
			prevExprs = append(prevExprs, *exprCXLine, *expr)

			lastExpr = prevExprs[len(prevExprs)-1]

			lastExpressionIdx = lastExpr.Index
		} else if lastExpressionOperatorOutputTypeSig.Type == ast.TYPE_STRUCT {
			// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
			// Now an arena getter, site S1176.
			structDetails := prgrm.GetCXTypeSignatureStructFromArray(lastExpressionOperatorOutputTypeSig.Meta)

			outStructGid := prgrm.CxIrState.CXTypeSignature_StructMng.CXTypeSignature_StructAlloc()
			outStructDetails := prgrm.CxIrState.CXTypeSignature_StructMng.CXTypeSignature_StructGet(outStructGid)
			outStructDetails.Fields = structDetails.Fields
			outStructDetails.StructType = structDetails.StructType
			outStructDetailsIdx := int(objmodel.GidIndex(int64(outStructGid)))

			_, outTypeSig := prgrm.AllocCXTypeSignatureCopy(lastExpressionOperatorOutputTypeSig)
			outTypeSig.Name = symName
			outTypeSig.Meta = outStructDetailsIdx
			outTypeSigIdx := outTypeSig.Index
			prgrm.CXAtomicOps[lastExpressionIdx].AddOutput(prgrm, outTypeSigIdx)

			// we need to create an expression to hold all the modifications
			// that will take place after this if statement
			_, inpTypeSig := prgrm.AllocCXTypeSignatureCopy(lastExpressionOperatorOutputTypeSig)
			inpTypeSig.Name = symName
			inpStructGid := prgrm.CxIrState.CXTypeSignature_StructMng.CXTypeSignature_StructAlloc()
			inpStructDetails := prgrm.CxIrState.CXTypeSignature_StructMng.CXTypeSignature_StructGet(inpStructGid)
			inpStructDetails.Fields = structDetails.Fields
			inpStructDetails.StructType = structDetails.StructType
			inpTypeSig.Meta = int(objmodel.GidIndex(int64(inpStructGid)))
			inpTypeSigIdx := inpTypeSig.Index

			// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
			// Now a constructor call, site S1184.
			exprCXLine := ast.MakeCXLineExpression(prgrm, lastExprCXLine.FileName, lastExprCXLine.LineNumber, lastExprCXLine.LineStr)
			// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
			// Now a constructor call, site S1185.
			expr := ast.MakeAtomicOperatorExpression(prgrm, nil)

			// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
			// Now an arena getter, site S1186.
			expression, err := prgrm.GetCXAtomicOp(expr.Index)
			if err != nil {
				panic(err)
			}

			expression.Package = prgrm.CXAtomicOps[lastExpressionIdx].Package
			expression.AddOutput(prgrm, inpTypeSigIdx)
			prevExprs = append(prevExprs, *exprCXLine, *expr)

			lastExpr = prevExprs[len(prevExprs)-1]

			lastExpressionIdx = lastExpr.Index
		} else {
			panic("type is not cx argument deprecate\n\n")
		}

	}

	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now an arena getter, site S1187.
	leftExprOutputTypeSig := prgrm.GetCXTypeSignatureFromArray(prgrm.CXAtomicOps[lastExpressionIdx].GetOutputs(prgrm)[0])

	// then left is a first (e.g first.rest) and right is a rest
	// let's check if left is a package
	pkg, err := prgrm.GetCurrentPackage()
	if err != nil {
		panic(err)
	}

	if imp, err := pkg.GetImport(prgrm, leftExprOutputTypeSig.Name); err == nil {
		// the external property will be propagated to the following arguments
		// this way we avoid considering these arguments as module names

		if cxpackages.IsDefaultPackage(leftExprOutputTypeSig.Name) {
			//TODO: constants.ConstCodes[leftExprOutputTypeSig.Name+"."+ident]
			//TODO: only play ConstCodes are used
			//Is used for constant declaration? But only for core packages?
			if code, ok := ConstCodes[leftExprOutputTypeSig.Name+"."+ident]; ok {
				constant := Constants[code]
				valArg := WritePrimary(prgrm, constant.Type, constant.Value, false)

				// TODO: to be removed and implemented with correct process to change the index
				// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
				// Now an arena getter, site S1188.
				lastExpressionOutputFieldTypeSig := prgrm.GetCXTypeSignatureFromArray(prgrm.CXAtomicOps[lastExpressionIdx].Outputs.Fields[0])
				if lastExpressionOutputFieldTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
					lastExpressionOutputFieldTypeSig.Meta = valArg.Index
					lastExpressionOutputFieldTypeSig.Name = valArg.Name
					lastExpressionOutputFieldTypeSig.Offset = valArg.Offset
				} else if lastExpressionOutputFieldTypeSig.Type == ast.TYPE_ATOMIC {
					// TODO: Review if this is the correct one
					typeSigIdx := ast.AllocCXTypeSignatureOfCXArg(prgrm, valArg)
					// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
					// Now an arena getter, site S1778.
					prgrm.GetCXTypeSignatureFromArray(typeSigIdx).Offset = valArg.Offset

					prgrm.CXAtomicOps[lastExpressionIdx].Outputs.Fields[0] = typeSigIdx
				} else if lastExpressionOutputFieldTypeSig.Type == ast.TYPE_POINTER_ATOMIC {
					// TODO: Review if this is the correct one
					typeSigIdx := ast.AllocCXTypeSignatureOfCXArg(prgrm, valArg)
					// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
					// Now an arena getter, site S1779.
					prgrm.GetCXTypeSignatureFromArray(typeSigIdx).Offset = valArg.Offset

					prgrm.CXAtomicOps[lastExpressionIdx].Outputs.Fields[0] = typeSigIdx
				} else {
					panic("type is not known")
				}

				return prevExprs
			} else if _, ok := ast.OpCodes[leftExprOutputTypeSig.Name+"."+ident]; ok {
				// then it's a native
				// TODO: we'd be referring to the function itself, not a function call
				// (functions as first-class objects)
				leftExprOutputTypeSigName := leftExprOutputTypeSig.Name + "." + ident
				leftExprOutputTypeSig.Name = leftExprOutputTypeSigName
				if leftExprOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
					leftExprIdx := leftExprOutputTypeSig.Meta
					// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
					// Now an arena getter, site S1759.
					prgrm.GetCXArg(ast.CXArgumentIndex(leftExprIdx)).Name = leftExprOutputTypeSigName
				}

				return prevExprs
			}
		}

		leftExprOutputTypeSig.Package = ast.CXPackageIndex(imp.Index)
		if leftExprOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
			leftExprIdx := leftExprOutputTypeSig.Meta
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1760.
			prgrm.GetCXArg(ast.CXArgumentIndex(leftExprIdx)).Package = ast.CXPackageIndex(imp.Index)
		}

		if glbl, err := imp.GetGlobal(prgrm, ident); err == nil {
			// then it's a global
			// prevExprs[len(prevExprs)-1].ProgramOutput[0] = glbl
			if glbl.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S1191.
				glblArg := prgrm.GetCXArgFromArray(ast.CXArgumentIndex(glbl.Meta))

				// then it's a global
				// prevExprs[len(prevExprs)-1].ProgramOutput[0] = glbl
				// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
				// Now an arena getter, site S1192.
				lastExpressionOutputTypeSig := prgrm.GetCXTypeSignatureFromArray(prgrm.CXAtomicOps[lastExpressionIdx].GetOutputs(prgrm)[0])

				var lastExpressionOutputArg *ast.CXArgument
				if lastExpressionOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
					// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
					// Now an arena getter, site S1194.
					lastExpressionOutputArg = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(lastExpressionOutputTypeSig.Meta))
					lastExpressionOutputArg.Name = glblArg.Name
					lastExpressionOutputTypeSig.Name = glblArg.Name

					lastExpressionOutputArg.Type = glblArg.Type
					lastExpressionOutputArg.StructType = glblArg.StructType
					lastExpressionOutputArg.Size = glblArg.Size
					lastExpressionOutputArg.PointerTargetType = glblArg.PointerTargetType
					lastExpressionOutputArg.Package = glblArg.Package
				} else if lastExpressionOutputTypeSig.Type == ast.TYPE_ATOMIC || lastExpressionOutputTypeSig.Type == ast.TYPE_POINTER_ATOMIC {
					lastExpressionOutputTypeSig.Name = glbl.Name
					lastExpressionOutputTypeSig.Type = glbl.Type
					lastExpressionOutputTypeSig.Meta = glbl.Meta
					lastExpressionOutputTypeSig.Offset = glbl.Offset
				} else {
					panic("type is not known")
				}

			} else if glbl.Type == ast.TYPE_ATOMIC || glbl.Type == ast.TYPE_POINTER_ATOMIC {
				// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
				// Now an arena getter, site S1195.
				lastExpressionOutputTypeSig := prgrm.GetCXTypeSignatureFromArray(prgrm.CXAtomicOps[lastExpressionIdx].GetOutputs(prgrm)[0])
				lastExpressionOutputTypeSig.Name = glbl.Name
				lastExpressionOutputTypeSig.Type = glbl.Type
				lastExpressionOutputTypeSig.Meta = glbl.Meta
				lastExpressionOutputTypeSig.Offset = glbl.Offset
			} else {
				panic("type is not known")
			}
		} else if fn, err := imp.GetFunction(prgrm, ident); err == nil {
			// then it's a function
			// not sure about this next line
			prgrm.CXAtomicOps[lastExpressionIdx].Outputs = nil
			prgrm.CXAtomicOps[lastExpressionIdx].Operator = ast.CXFunctionIndex(fn.Index)
		} else if strct, err := prgrm.GetStruct(ident, imp.Name); err == nil {
			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
			// Now an arena getter, site S1196.
			lastExpressionOutputTypeSig := prgrm.GetCXTypeSignatureFromArray(prgrm.CXAtomicOps[lastExpressionIdx].GetOutputs(prgrm)[0])
			var lastExpressionOutputArg *ast.CXArgument
			if lastExpressionOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S1198.
				lastExpressionOutputArg = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(lastExpressionOutputTypeSig.Meta))
			} else {
				panic("type is not cx argument deprecate\n\n")
			}

			lastExpressionOutputArg.StructType = strct
		} else {
			file, line := CurrentFile, LineNo
			if details := leftExprOutputTypeSig.ArgDetails; details != ast.CXArgumentDebugGidNil {
				file, line = ast.CXArgumentDebugFileLine(prgrm, details)
			}
			if msg, renamed := renamedOpcodeMessage(imp.Name + "." + ident); renamed {
				println(ast.CompilationError(file, line), msg)
				return nil
			}
			println(ast.CompilationError(file, line), fmt.Sprintf("identifier '%s.%s' does not exist", imp.Name, ident))
			return nil
		}
	} else {
		// then left is not a package name
		if cxpackages.IsDefaultPackage(leftExprOutputTypeSig.Name) {
			println(ast.CompilationError(ast.CXArgumentDebugFileLine(prgrm, leftExprOutputTypeSig.ArgDetails)),
				fmt.Sprintf("built-in package %s is used without importing it, add import \"%s%s\"",
					leftExprOutputTypeSig.Name, cxpackages.SysPrefix, leftExprOutputTypeSig.Name))
			os.Exit(types.CX_COMPILATION_ERROR)
		}

		// then it's a struct
		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now a constructor call, site S1199.
		leftExprField := ast.MakeArgument(prgrm, ident, CurrentFile, LineNo)
		// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
		// Now an arena getter, site S1200.
		leftPkg, err := prgrm.GetPackageFromArray(leftExprOutputTypeSig.Package)
		if err != nil {
			panic(err)
		}
		leftExprField.SetType(types.IDENTIFIER)
		leftExprField.SetPackage(leftPkg)
		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now an arena insert, site S1201.
		leftExprFieldIdx := prgrm.AddCXArgInArray(leftExprField)

		var leftExprIdx int
		if leftExprOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
			leftExprIdx = leftExprOutputTypeSig.Meta

			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1761.
			prgrm.GetCXArg(ast.CXArgumentIndex(leftExprIdx)).Fields = append(prgrm.GetCXArg(ast.CXArgumentIndex(leftExprIdx)).Fields, leftExprFieldIdx)
		} else if leftExprOutputTypeSig.Type == ast.TYPE_ATOMIC || leftExprOutputTypeSig.Type == ast.TYPE_POINTER_ATOMIC {
			// TODO: implement correct way when it is possible
			// temporarily return back to cx arg deprecate
			// since the arg will become a struct
			// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
			// Now a constructor call, site S1202.
			arg := ast.MakeArgument(prgrm, leftExprOutputTypeSig.Name, CurrentFile, LineNo) // fix: line numbers in errors sometimes report +1 or -1. Issue #195
			arg.SetType(types.IDENTIFIER)
			arg.Name = leftExprOutputTypeSig.Name
			arg.Package = leftExprOutputTypeSig.Package
			arg.PassBy = leftExprOutputTypeSig.PassBy

			// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
			// Now an arena insert, site S1203.
			argIdx := prgrm.AddCXArgInArray(arg)
			leftExprOutputTypeSig.Type = ast.TYPE_CXARGUMENT_DEPRECATE
			leftExprOutputTypeSig.Meta = int(argIdx)

			leftExprIdx = int(argIdx)
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1762.
			prgrm.GetCXArg(ast.CXArgumentIndex(leftExprIdx)).Fields = append(prgrm.GetCXArg(ast.CXArgumentIndex(leftExprIdx)).Fields, leftExprFieldIdx)
		} else if leftExprOutputTypeSig.Type == ast.TYPE_STRUCT {
			// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
			// Now an arena getter, site S1204.
			structDetails := prgrm.GetCXTypeSignatureStructFromArray(leftExprOutputTypeSig.Meta)
			structDetails.Fields = append(structDetails.Fields, leftExprFieldIdx)
		} else {
			panic("type is not known")
		}
	}

	return prevExprs
}

// renamedOpcodeMessage returns the compile error for name when it is a former
// opcode name, see constants.OpcodeRenameTable.
func renamedOpcodeMessage(name string) (string, bool) {
	rename, found := constants.OpcodeRenamed(name)
	if !found {
		return "", false
	}
	msg := fmt.Sprintf("%s was renamed to %s in language version %d", name, rename.NewName, rename.Version)
	oldPkg, _, _ := strings.Cut(name, ".")
	if pkg, _, cut := strings.Cut(rename.NewName, "."); cut && pkg != oldPkg && cxpackages.IsDefaultPackage(pkg) {
		msg += fmt.Sprintf(", import \"%s%s\"", cxpackages.SysPrefix, pkg)
	}
	return msg, true
}
