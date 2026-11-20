package astapi

import (
	"errors"

	"github.com/skycoin/skycoin-cx/t0-types" // thin: only Code; no depth gain
	"github.com/skycoin/skycoin-cx/t3-ast"
	cxast "github.com/skycoin/skycoin-cx/t3-ast"
	cxparseractions "github.com/skycoin/skycoin-cx/t4-actions" // thin: only AST, WritePrimary; no depth gain
)

// AddNativeInputToExpression adds native input to an expression
// in a function in cx program.
//
// The inputs are
// cxprogram - in the form of cxast.CXProgram.
// packageName - the name of the package where the function is located.
// functionName - the name of the function where the expression is located.
// inputName - the name of input to be added.
// inputType - the type of the input to be added.
// lineNumber - the line number of the expression in the function.
//
// Example:
// We have this CX Program:
// 0.- Package: main
//
//	Functions
//	    0.- Function: TestFunction (inputOne i32) (outputOne i16)
//	        0.- Expression: z i16 = add(x i16)
//
// We use AddNativeInputToExpression(cxprogram, "main", "TestFunction", "y", cxtypes.I16, 0).
// The Result will be:
// 0.- Package: main
//
//	Functions
//	    0.- Function: TestFunction (inputOne i32) (outputOne i16)
//	        0.- Expression: z i16 = add(x i16, y i16)
//
// Note the y i16 added as an input to the expression in line 0.
func AddNativeInputToExpression(cxprogram *cxast.CXProgram, packageName, functionName, inputName string, inputType types.Code, lineNumber int) error {
	pkg, err := FindPackage(cxprogram, packageName)
	if err != nil {
		return err
	}

	fn, err := FindFunction(cxprogram, functionName)
	if err != nil {
		return err
	}

	expr, err := fn.GetExpressionByLine(lineNumber)
	if err != nil {
		return err
	}

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1561.
	arg := cxast.MakeField(cxprogram, inputName, inputType, "", -1).SetType(types.Code(inputType))
	arg.Package = cxast.CXPackageIndex(pkg.Index)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now an arena insert, site S1562.
	argIdx := cxprogram.AddCXArgInArray(arg)
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
	// Now an arena getter, site S1563.
	cxAtomicOp, err := cxprogram.GetCXAtomicOp(expr.Index)
	if err != nil {
		panic(err)
	}

	// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
	// Now an arena getter, site S1564.
	typeSigIdx := cxast.AllocCXTypeSignatureOfCXArg(cxprogram, cxprogram.GetCXArgFromArray(cxast.CXArgumentIndex(argIdx)))
	cxAtomicOp.AddInput(cxprogram, typeSigIdx)

	return nil
}

// RemoveInputFromExpression removes an input from an
// expression in a function in cx program.
//
// The inputs are
// cxprogram - in the form of cxast.CXProgram.
// functionName - the name of the function where the expression is located.
// lineNumber - the line number of the expression in the function.
//
// Example:
// We have this CX Program:
// 0.- Package: main
//     Functions
//         0.- Function: TestFunction (inputOne i32) (outputOne i16)
//             0.- Expression: z i16 = add(x i16, y i16)
//
// We use RemoveInputFromExpression(cxprogram, "TestFunction", 0).
// The Result will be:
// 0.- Package: main
//     Functions
//         0.- Function: TestFunction (inputOne i32) (outputOne i16)
//             0.- Expression: z i16 = add(x i16)
//
// Note the y i16 removed from the expression in line 0.
// func RemoveInputFromExpression(cxprogram *cxast.CXProgram, functionName string, lineNumber int) error {
// 	fn, err := FindFunction(cxprogram, functionName)
// 	if err != nil {
// 		return err
// 	}

// 	expr, err := fn.GetExpressionByLine(lineNumber)
// 	if err != nil {
// 		return err
// 	}

// 	cxAtomicOp, err := cxprogram.GetCXAtomicOp(expr.Index)
// 	if err != nil {
// 		panic(err)
// 	}

// 	// cxAtomicOp.RemoveInput()
// 	return nil
// }

// AddNativeOutputToExpression adds native output to
// an expression in a function in cx program.
//
// The inputs are
// cxprogram - in the form of cxast.CXProgram.
// packageName - the name of the package where the function is located.
// functionName - the name of the function where the expression is located.
// outputName - the name of output to be added.
// outputType - the type of the output to be added.
// lineNumber - the line number of the expression in the function.
//
// Example:
// We have this CX Program:
// 0.- Package: main
//
//	Functions
//	    0.- Function: TestFunction (inputOne i32) (outputOne i16)
//	        0.- Expression: add(x i16, y i16)
//
// We use AddNativeOutputToExpression(cxprogram, "main", "TestFunction", "z", cxtypes.I16, 0).
// The Result will be:
// 0.- Package: main
//
//	Functions
//	    0.- Function: TestFunction (inputOne i32) (outputOne i16)
//	        0.- Expression: z i16 = add(x i16, y i16)
//
// Note the z i16 added as an output of the expression in line 0.
func AddNativeOutputToExpression(cxprogram *cxast.CXProgram, packageName, functionName, outputName string, outputType types.Code, lineNumber int) error {
	pkg, err := FindPackage(cxprogram, packageName)
	if err != nil {
		return err
	}

	fn, err := FindFunction(cxprogram, functionName)
	if err != nil {
		return err
	}

	expr, err := fn.GetExpressionByLine(lineNumber)
	if err != nil {
		return err
	}

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1566.
	arg := cxast.MakeField(cxprogram, outputName, outputType, "", -1).SetType(types.Code(outputType))
	arg.Package = cxast.CXPackageIndex(pkg.Index)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now an arena insert, site S1567.
	argIdx := cxprogram.AddCXArgInArray(arg)
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
	// Now an arena getter, site S1568.
	cxAtomicOp, err := cxprogram.GetCXAtomicOp(expr.Index)
	if err != nil {
		panic(err)
	}

	// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
	// Now an arena getter, site S1569.
	typeSigIdx := cxast.AllocCXTypeSignatureOfCXArg(cxprogram, cxprogram.GetCXArgFromArray(cxast.CXArgumentIndex(argIdx)))
	cxAtomicOp.AddOutput(cxprogram, typeSigIdx)

	return nil
}

// RemoveOutputFromExpression removes an output
// from an expression in a function in cx program.
//
// The inputs are
// cxprogram - in the form of cxast.CXProgram.
// functionName - the name of the function where the expression is located.
// lineNumber - the line number of the expression in the function.
//
// Example:
// We have this CX Program:
// 0.- Package: main
//     Functions
//         0.- Function: TestFunction (inputOne i32) (outputOne i16)
//             0.- Expression: z i16 = add(x i16, y i16)
//
// We use RemoveOutputFromExpression(cxprogram, "TestFunction", 0).
// The Result will be:
// 0.- Package: main
//     Functions
//         0.- Function: TestFunction (inputOne i32) (outputOne i16)
//             0.- Expression: add(x i16, y i16)
//
// Note the z i16 removed as an output from the expression in line 0.
// func RemoveOutputFromExpression(cxprogram *cxast.CXProgram, functionName string, lineNumber int) error {
// 	fn, err := FindFunction(cxprogram, functionName)
// 	if err != nil {
// 		return err
// 	}

// 	expr, err := fn.GetExpressionByLine(lineNumber)
// 	if err != nil {
// 		return err
// 	}

// 	cxAtomicOp, err := cxprogram.GetCXAtomicOp(expr.Index)
// 	if err != nil {
// 		panic(err)
// 	}

// 	cxAtomicOp.RemoveOutput()
// 	return nil
// }

// MakeInputExpressionAPointer makes an input of an
// expression a pointer.
//
// The inputs are
// cxprogram - in the form of cxast.CXProgram.
// functionName - the name of the function where the expression is located.
// lineNumber - the line number of the expression in the function.
// expressionNumber
// inputNumber - the input number of the expression to be made as a pointer.
//
// Example:
// We have this CX Program:
// 0.- Package: main
//
//	Functions
//	    0.- Function: TestFunction (inputOne i32) (outputOne i16)
//	        0.- Expression: z i16 = add(x i16, y i16)
//
// We use MakeInputExpressionAPointer(cxprogram, "TestFunction", 0, 1).
// The Result will be:
// 0.- Package: main
//
//	Functions
//	    0.- Function: TestFunction (inputOne i32) (outputOne i16)
//	        0.- Expression: z i16 = add(x i16, y *i16)
//
// Note that y *i16 became a pointer.
func MakeInputExpressionAPointer(cxprogram *cxast.CXProgram, functionName string, lineNumber, inputNumber int) error {
	fn, err := FindFunction(cxprogram, functionName)
	if err != nil {
		return err
	}

	expr, err := fn.GetExpressionByLine(lineNumber)
	if err != nil {
		return err
	}

	if expr.Type == ast.CX_LINE {
		return errors.New("Expression is a CXLine")
	}
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
	// Now an arena getter, site S1571.
	cxAtomicOp, err := cxprogram.GetCXAtomicOp(expr.Index)
	if err != nil {
		panic(err)
	}

	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now an arena getter, site S1572.
	cxAtomicOpInputTypeSig := cxprogram.GetCXTypeSignatureFromArray(cxAtomicOp.GetInputs(cxprogram)[inputNumber])
	// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
	// Now an arena getter, site S1573.
	cxast.MakePointer(cxprogram.GetCXArgFromArray(cxast.CXArgumentIndex(cxAtomicOpInputTypeSig.Meta)))
	return nil
}

// MakeOutputExpressionAPointer makes an output
// of an expression a pointer.
//
// The inputs are
// cxprogram - in the form of cxast.CXProgram.
// functionName - the name of the function where the expression is located.
// lineNumber - the line number of the expression in the function.
// expressionNumber
// outputNumber - the output number of the expression to be made as a pointer.
//
// Example:
// We have this CX Program:
// 0.- Package: main
//
//	Functions
//	    0.- Function: TestFunction (inputOne i32) (outputOne i16)
//	        0.- Expression: z i16 = add(x i16, y i16)
//
// We use MakeOutputExpressionAPointer(cxprogram, "TestFunction", 0, 0).
// The Result will be:
// 0.- Package: main
//
//	Functions
//	    0.- Function: TestFunction (inputOne i32) (outputOne i16)
//	        0.- Expression: z *i16 = add(x i16, y i16)
//
// Note that z *i16 became a pointer.
func MakeOutputExpressionAPointer(cxprogram *cxast.CXProgram, functionName string, lineNumber, outputNumber int) error {
	fn, err := FindFunction(cxprogram, functionName)
	if err != nil {
		return err
	}

	expr, err := fn.GetExpressionByLine(lineNumber)
	if err != nil {
		return err
	}

	if expr.Type == ast.CX_LINE {
		return errors.New("Expression is a CXLine")
	}
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
	// Now an arena getter, site S1574.
	cxAtomicOp, err := cxprogram.GetCXAtomicOp(expr.Index)
	if err != nil {
		panic(err)
	}

	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now an arena getter, site S1575.
	cxAtomicOpOutputTypeSig := cxprogram.GetCXTypeSignatureFromArray(cxAtomicOp.GetOutputs(cxprogram)[outputNumber])
	// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
	// Now an arena getter, site S1576.
	cxast.MakePointer(cxprogram.GetCXArgFromArray(cxast.CXArgumentIndex(cxAtomicOpOutputTypeSig.Meta)))
	return nil
}

// GetAccessibleArgsForFunctionByType gets all accessible
// arguments in cx program for a function by specified
// argument type.
//
// The inputs are
// cxprogram - in the form of cxast.CXProgram.
// packageLocationName - the name of the package where the function is located.
// functionName - the name of the function where the argument will be used.
// argType - the type of argument we are looking for.
//
// Example:
// We have this CX Program:
// 0.- Package: main
//
//	Globals
//	        0.- Global: testGlobal
//	Functions
//	        0.- Function: TestFunction (inputOne i32) (outputOne i16)
//	                0.- Expression: add(x i16)
//
// We use GetAccessibleArgsForFunctionByType(cxprogram, "main", "TestFunction", cxtypes.I16).
// We then get a result of
// 1. testGlobal i16 arg in the form of cxast.CXTypeSignature.
// 2. x i16 arg in the form of cxast.CXTypeSignature.
func GetAccessibleArgsForFunctionByType(cxprogram *cxast.CXProgram, packageLocationName, functionName string, argType types.Code) ([]*cxast.CXTypeSignature, error) {
	var argsList []*cxast.CXTypeSignature

	// Get all globals
	pkg, err := FindPackage(cxprogram, packageLocationName)
	if err != nil {
		return nil, err
	}

	for _, glblFldIdx := range pkg.Globals.Fields {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S1577.
		argsList = appendArgsByType(cxprogram, argsList, cxprogram.GetCXTypeSignatureFromArray(glblFldIdx), argType)
	}

	for _, impIdx := range pkg.Imports {
		// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
		// Now an arena getter, site S1578.
		imp, err := cxprogram.GetPackageFromArray(impIdx)
		if err != nil {
			panic(err)
		}
		for _, glblFldIdx := range imp.Globals.Fields {
			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
			// Now an arena getter, site S1579.
			argsList = appendArgsByType(cxprogram, argsList, cxprogram.GetCXTypeSignatureFromArray(glblFldIdx), argType)
		}
	}

	fn, err := FindFunction(cxprogram, functionName)
	if err != nil {
		return nil, err
	}

	// Get all args from expression inputs
	for _, expr := range fn.Expressions {
		cxAtomicOp, _, _, err := cxprogram.GetOperation(&expr)
		if err != nil {
			panic(err)
		}
		for _, inputArgIdx := range cxAtomicOp.GetInputs(cxprogram) {
			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
			// Now an arena getter, site S1580.
			argsList = appendArgsByType(cxprogram, argsList, cxprogram.GetCXTypeSignatureFromArray(inputArgIdx), argType)
		}
	}

	return argsList, nil
}

// appendArgsByType appends typeSig, or its struct fields, to argsList if
// they are of argType. typeSig may be atomic or a deprecated CXArgument.
func appendArgsByType(cxprogram *cxast.CXProgram, argsList []*cxast.CXTypeSignature, typeSig *cxast.CXTypeSignature, argType types.Code) []*cxast.CXTypeSignature {
	switch typeSig.Type {
	case cxast.TYPE_ATOMIC, cxast.TYPE_POINTER_ATOMIC:
		if types.Code(typeSig.Meta) == argType {
			argsList = append(argsList, typeSig)
		}
	case cxast.TYPE_CXARGUMENT_DEPRECATE:
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S1581.
		arg := cxprogram.GetCXArgFromArray(cxast.CXArgumentIndex(typeSig.Meta))
		if arg.IsStruct() {
			for _, fieldIdx := range arg.StructType.Fields {
				// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
				// Now an arena getter, site S1582.
				argsList = appendArgsByType(cxprogram, argsList, cxprogram.GetCXTypeSignatureFromArray(fieldIdx), argType)
			}
		} else if arg.Type == argType {
			argsList = append(argsList, cxast.GetCXTypeSignatureRepresentationOfCXArg(cxprogram, arg))
		}
	}
	return argsList
}

func AddLiteralInputToExpression(cxprogram *cxast.CXProgram, packageName, functionName string, bytes []byte, argType types.Code, lineNumber int) error {
	pkg, err := FindPackage(cxprogram, packageName)
	if err != nil {
		return err
	}
	cxprogram.CurrentPackage = cxprogram.Packages[packageName]
	fn, err := FindFunction(cxprogram, functionName)
	if err != nil {
		return err
	}

	expr, err := fn.GetExpressionByLine(lineNumber)
	if err != nil {
		return err
	}

	cxparseractions.AST = cxprogram
	litArg := cxparseractions.WritePrimary(cxprogram, argType, bytes, false)

	cxAtomicOp2, _, _, err := cxprogram.GetOperation(expr)
	if err != nil {
		panic(err)
	}

	litArg.Package = cxast.CXPackageIndex(pkg.Index)

	// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
	// Now an arena getter, site S1583.
	typeSigIdx := cxast.AllocCXTypeSignatureOfCXArg(cxprogram, cxprogram.GetCXArgFromArray(cxast.CXArgumentIndex(litArg.Index)))
	cxAtomicOp2.AddInput(cxprogram, typeSigIdx)

	return nil
}
