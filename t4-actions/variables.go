package actions

import (
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
	"github.com/skycoin/skycoin-cx/t2-globals" // thin: only FoundCompileErrors; no depth gain
	"github.com/skycoin/skycoin-cx/t3-ast"
)

func processGlobalInitializer(prgrm *ast.CXProgram, initializer []ast.CXExpression, glblIdx ast.CXTypeSignatureIndex) {
	initializerExpressionIdx := initializer[len(initializer)-1].Index
	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S1209.
	initializerExpressionOperator := prgrm.GetFunctionFromArray(prgrm.CXAtomicOps[initializerExpressionIdx].Operator)
	// then we just re-assign offsets
	if initializerExpressionOperator == nil {
		// then it's a literal
		typeSigIdx := prgrm.CXAtomicOps[initializerExpressionIdx].GetOutputs(prgrm)[0]
		prgrm.CXAtomicOps[initializerExpressionIdx].AddInput(prgrm, typeSigIdx)
		prgrm.CXAtomicOps[initializerExpressionIdx].Outputs = nil

		prgrm.CXAtomicOps[initializerExpressionIdx].AddOutput(prgrm, glblIdx)
		// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
		// Now an arena insert, site S1210.
		opIdx := prgrm.AddNativeFunctionInArray(ast.Natives[constants.OP_IDENTITY])
		prgrm.CXAtomicOps[initializerExpressionIdx].Operator = opIdx

		//add intialization statements, to array
		prgrm.SysInitExprs = append(prgrm.SysInitExprs, initializer...)
	} else {
		// then it's an expression
		if initializer[len(initializer)-1].IsStructLiteral() {
			// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
			// Now a heap literal, site S1211.
			outputStruct := &ast.CXStruct{}
			outputStruct.AddField_TypeSignature(prgrm, glblIdx)
			// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
			// Now a heap literal, site S1212 S1213.
			index := prgrm.AddCXAtomicOp(&ast.CXAtomicOperator{Outputs: outputStruct, Operator: -1, Function: -1})
			initializer = StructLiteralAssignment(prgrm,
				// OBJMODEL: state.CXExpressionMng (manager storage) or []CXExpressionGid, phase 4 #346.
				// Now a slice of objects, site S1214.
				[]ast.CXExpression{
					{
						Index: index,
						Type:  ast.CX_ATOMIC_OPERATOR,
					},
				},
				initializer,
			)
		} else {
			prgrm.CXAtomicOps[initializerExpressionIdx].Outputs = nil
			prgrm.CXAtomicOps[initializerExpressionIdx].AddOutput(prgrm, glblIdx)
		}

		//add intialization statements, to array
		prgrm.SysInitExprs = append(prgrm.SysInitExprs, initializer...)
	}
}

// DeclareGlobalInPackage creates a global variable in a specified package
//
// If `doesInitialize` is true, then `initializer` is used to initialize the
// new variable.
//
// Input arguments description:
//
//	prgrm - a CXProgram that contains all the data and arrays of the program.
//	pkg - the package where the global will belong.
//	declarator - contains the name of the global var.
//	declaration_specifiers - contains the type build of the global.
//	initializer - if doesInitialize is true then this contains the initial
//				  value of the global.
//	doesInitialize - true if global is initialized or given a value.
func DeclareGlobalInPackage(prgrm *ast.CXProgram, pkg *ast.CXPackage,
	declarator *ast.CXArgument, declaration_specifiers *ast.CXArgument,
	initializer []ast.CXExpression, doesInitialize bool) {
	if pkg == nil {
		var err error
		pkg, err = prgrm.GetCurrentPackage()
		if err != nil {
			panic(err)
		}
	}
	declaration_specifiers.Package = ast.CXPackageIndex(pkg.Index)

	totalSize := declaration_specifiers.Size
	for _, decl := range declaration_specifiers.DeclarationSpecifiers {
		switch decl {
		case constants.DECL_POINTER, constants.DECL_SLICE:
			totalSize = types.POINTER_SIZE
		case constants.DECL_ARRAY:
			totalSize = declaration_specifiers.Size * types.TotalLength(declaration_specifiers.Lengths)
		}
	}

	// Treat the name a bit different whether it's defined already or not.
	if glbl, err := pkg.GetGlobal(prgrm, declarator.Name); err == nil {
		// then it was only added a reference to the symbol

		// if offset is not valid, assign new offset
		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now a heap literal, site S1215.
		var glblArg *ast.CXArgument = &ast.CXArgument{}
		if !glbl.Offset.IsValid() {
			if declaration_specifiers.IsSlice() { // TODO:PTR move branch in WritePrimary
				glblArg = WritePrimary(prgrm, declaration_specifiers.Type,
					make([]byte, types.POINTER_SIZE), true)
			} else {
				glblArg = WritePrimary(prgrm, declaration_specifiers.Type,
					make([]byte, totalSize), false)
			}

		}

		if ast.IsTypeAtomic(declaration_specifiers) {
			glbl.Type = ast.TYPE_ATOMIC
			glbl.Meta = int(declaration_specifiers.Type)
			if !glbl.Offset.IsValid() {
				glbl.Offset = glblArg.Offset
			}

			glbl.PassBy = declaration_specifiers.PassBy
		} else if ast.IsTypePointerAtomic(declaration_specifiers) {
			glbl.Type = ast.TYPE_POINTER_ATOMIC
			glbl.Meta = int(declaration_specifiers.Type)
			if !glbl.Offset.IsValid() {
				glbl.Offset = glblArg.Offset
			}

			glbl.PassBy = declaration_specifiers.PassBy
		} else if ast.IsTypeArrayAtomic(declaration_specifiers) {
			glbl.Type = ast.TYPE_ARRAY_ATOMIC

			arrayGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
			typeSignatureForArray := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(arrayGid)
			typeSignatureForArray.Meta = int(declaration_specifiers.Type)
			typeSignatureForArray.Lengths = declaration_specifiers.Lengths
			typeSignatureForArray.Indexes = declaration_specifiers.Indexes

			glbl.Meta = int(objmodel.GidIndex(int64(arrayGid)))

			glbl.PassBy = declaration_specifiers.PassBy

			if !glbl.Offset.IsValid() {
				glbl.Offset = glblArg.Offset
			}
		} else if ast.IsTypePointerArrayAtomic(declaration_specifiers) {
			glbl.Type = ast.TYPE_POINTER_ARRAY_ATOMIC

			arrayGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
			typeSignatureForArray := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(arrayGid)
			typeSignatureForArray.Meta = int(declaration_specifiers.Type)
			typeSignatureForArray.Lengths = declaration_specifiers.Lengths
			typeSignatureForArray.Indexes = declaration_specifiers.Indexes
			glbl.Meta = int(objmodel.GidIndex(int64(arrayGid)))
			glbl.PassBy = declaration_specifiers.PassBy

			if !glbl.Offset.IsValid() {
				glbl.Offset = glblArg.Offset
			}
		} else if ast.IsTypeSliceAtomic(declaration_specifiers) {
			// If slice atomic type, i.e. []i32, []f64, etc.

			glbl.Type = ast.TYPE_SLICE_ATOMIC

			arrayGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
			typeSignatureForArray := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(arrayGid)
			typeSignatureForArray.Meta = int(declaration_specifiers.Type)
			typeSignatureForArray.Lengths = declaration_specifiers.Lengths
			typeSignatureForArray.Indexes = declaration_specifiers.Indexes
			glbl.Meta = int(objmodel.GidIndex(int64(arrayGid)))
			glbl.PassBy = declaration_specifiers.PassBy

			if !glbl.Offset.IsValid() {
				glbl.Offset = glblArg.Offset
			}
		} else if ast.IsTypePointerSliceAtomic(declaration_specifiers) {
			// If pointer slice atomic type, i.e. *[]i32, *[]f64, etc.

			glbl.Type = ast.TYPE_POINTER_SLICE_ATOMIC

			arrayGid := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayAlloc()
			typeSignatureForArray := prgrm.CxIrState.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayGet(arrayGid)
			typeSignatureForArray.Meta = int(declaration_specifiers.Type)
			typeSignatureForArray.Lengths = declaration_specifiers.Lengths
			typeSignatureForArray.Indexes = declaration_specifiers.Indexes
			glbl.Meta = int(objmodel.GidIndex(int64(arrayGid)))
			glbl.PassBy = declaration_specifiers.PassBy

			if !glbl.Offset.IsValid() {
				glbl.Offset = glblArg.Offset
			}
		} else {
			// its a cxargument_deprecate type
			var glblIdx int
			// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
			// Now a heap literal, site S1224.
			var glblCXArg *ast.CXArgument = &ast.CXArgument{}
			if glbl.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
				glblIdx = glbl.Meta
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S1225.
				glblCXArg = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(glblIdx))
			} else {
				panic("type is not type cx argument deprecate\n\n")
			}

			if glblCXArg.Size == 0 || totalSize == 0 {
				prgrm.CXArgs[glblIdx].Offset = glblArg.Offset
				prgrm.CXArgs[glblIdx].PassBy = glblArg.PassBy
			}

			declaration_specifiers.Name = prgrm.CXArgs[glblIdx].Name
			declaration_specifiers.Offset = prgrm.CXArgs[glblIdx].Offset
			declaration_specifiers.PassBy = prgrm.CXArgs[glblIdx].PassBy
			declaration_specifiers.Package = prgrm.CXArgs[glblIdx].Package

			// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
			// Now an object copy, site S1226.
			prgrm.CXArgs[glblIdx] = *declaration_specifiers
			prgrm.CXArgs[glblIdx].Index = glblIdx

		}

		// Checking if something is supposed to be initialized
		// and if `initializer` actually contains something.
		// If `initializer` is `nil`, this means that an expression
		// equivalent to nil was used, such as `[]i32{}`.
		if doesInitialize && initializer != nil {
			processGlobalInitializer(prgrm, initializer, glbl.Index)
		}
	} else {
		// then it hasn't been defined
		// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
		// Now a heap literal, site S1227.
		var glblArg *ast.CXArgument = &ast.CXArgument{}
		if declaration_specifiers.IsSlice() { // TODO:PTR move branch in WritePrimary
			glblArg = WritePrimary(prgrm, declaration_specifiers.Type, make([]byte, types.POINTER_SIZE), true)
		} else {
			glblArg = WritePrimary(prgrm, declaration_specifiers.Type, make([]byte, totalSize), false)
		}

		declaration_specifiers.Name = declarator.Name
		declaration_specifiers.ArgDetails.FileLine = declarator.ArgDetails.FileLine
		declaration_specifiers.Offset = glblArg.Offset
		declaration_specifiers.Size = glblArg.Size
		declaration_specifiers.Package = ast.CXPackageIndex(pkg.Index)

		typeSignature := ast.GetCXTypeSignatureRepresentationOfCXArg(prgrm, declaration_specifiers)
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
		// Now an arena insert, site S1228.
		typeSignatureIdx := prgrm.AddCXTypeSignatureInArray(typeSignature)
		pkg.AddGlobal_TypeSignature(prgrm, typeSignatureIdx)

		// Checking if something is supposed to be initialized
		// and if `initializer` actually contains something.
		// If `initializer` is `nil`, this means that an expression
		// equivalent to nil was used, such as `[]i32{}`.
		if doesInitialize && initializer != nil {
			processGlobalInitializer(prgrm, initializer, typeSignatureIdx)
		}
	}

}

func processLocalInitialization(prgrm *ast.CXProgram, pkg *ast.CXPackage, initializer *[]ast.CXExpression, localVarIdx ast.CXTypeSignatureIndex, declSpecIdx ast.CXArgumentIndex) {
	initializerExpression, err := prgrm.GetCXAtomicOpFromExpressions(*initializer, len(*initializer)-1)
	if err != nil {
		panic(err)
	}

	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S1229.
	initializerExpressionOperator := prgrm.GetFunctionFromArray(initializerExpression.Operator)
	if initializerExpressionOperator == nil {
		// THEN it's a literal, e.g. var foo i32 = 10;

		// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
		// Now a constructor call, site S1230.
		exprCXLine := ast.MakeCXLineExpression(prgrm, CurrentFile, LineNo, LineStr)

		// we need to create an expression that links the initializer expressions
		// with the declared variable
		// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
		// Now a constructor call, site S1231.
		expr := ast.MakeAtomicOperatorExpression(prgrm, ast.Natives[constants.OP_IDENTITY])
		cxExprAtomicOpIdx := expr.Index
		prgrm.CXAtomicOps[cxExprAtomicOpIdx].Package = ast.CXPackageIndex(pkg.Index)

		var initOutTypeSigIdx ast.CXTypeSignatureIndex
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S1232.
		initializerExpressionOutputTypeSig := prgrm.GetCXTypeSignatureFromArray(initializerExpression.GetOutputs(prgrm)[0])
		if initializerExpressionOutputTypeSig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1233.
			initializerExpressionOutputArg := prgrm.GetCXArgFromArray(ast.CXArgumentIndex(initializerExpressionOutputTypeSig.Meta))
			initOut := initializerExpressionOutputArg
			initOutIdx := ast.CXArgumentIndex(initializerExpressionOutputTypeSig.Meta)

			// CX checks the output of an expression to determine if it's being passed
			// by value or by reference, so we copy this property from the initializer's
			// output, in case of something like var foo *i32 = &bar
			prgrm.CXArgs[declSpecIdx].PassBy = initOut.PassBy

			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1234.
			typeSig := ast.GetCXTypeSignatureRepresentationOfCXArg(prgrm, prgrm.GetCXArgFromArray(initOutIdx))
			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
			// Now an arena insert, site S1235.
			initOutTypeSigIdx = prgrm.AddCXTypeSignatureInArray(typeSig)
		} else if initializerExpressionOutputTypeSig.Type == ast.TYPE_ATOMIC || initializerExpressionOutputTypeSig.Type == ast.TYPE_POINTER_ATOMIC {
			initOutTypeSigIdx = initializerExpression.GetOutputs(prgrm)[0]
		} else if initializerExpressionOutputTypeSig.Type == ast.TYPE_STRUCT {
			initOutTypeSigIdx = initializerExpression.GetOutputs(prgrm)[0]
		} else {
			panic("type is not known")
		}

		prgrm.CXAtomicOps[cxExprAtomicOpIdx].AddOutput(prgrm, localVarIdx)
		prgrm.CXAtomicOps[cxExprAtomicOpIdx].AddInput(prgrm, initOutTypeSigIdx)

		// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
		// Now an object copy, site S1236.
		(*initializer)[len(*initializer)-1] = *exprCXLine
		(*initializer) = append(*initializer, *expr)

	} else {
		// ELSE it's an expression with an operator

		expr := (*initializer)[len(*initializer)-1]
		// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
		// Now an arena getter, site S1237.
		cxExprAtomicOp, err := prgrm.GetCXAtomicOp(expr.Index)
		if err != nil {
			panic(err)
		}

		// THEN the expression has outputs created from the result of
		// handling a dot notation initializer, and it needs to be replaced
		// ELSE we simply add it using `AddOutput`
		if len(cxExprAtomicOp.GetOutputs(prgrm)) > 0 {
			// declSpecIdx := prgrm.AddCXArgInArray(declarationSpecifiers)
			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureRelease(gid), phase 4 #331.
			// Now a dropped object, site S1238.
			cxExprAtomicOp.Outputs.Fields = nil

		}
		cxExprAtomicOp.AddOutput(prgrm, localVarIdx)
	}
}

// DeclareLocal() creates a local variable inside a function.
//
// Returns a list of expressions that contains the initialization, if any.
//
// Input arguments description:
//
//	prgrm - a CXProgram that contains all the data and arrays of the program.
//	declarator - contains the name of the var.
//	declaration_specifiers - contains the type build of the var.
//	initializer - if doesInitialize is true then this contains the initial
//				  value of the var.
//	doesInitialize - true if var is initialized or given a value.
func DeclareLocal(prgrm *ast.CXProgram, declarator *ast.CXArgument, declarationSpecifiers *ast.CXArgument,
	initializer []ast.CXExpression, doesInitialize bool) []ast.CXExpression {
	if globals.FoundCompileErrors {
		return nil
	}

	currFn, err := prgrm.GetCurrentFunction()
	if err != nil {
		// TODO: improve error handling
		panic("error getting current function")
	}
	err = currFn.AddLocalVariableName(declarator.Name)
	if err != nil {
		panic("error adding local variable")
	}

	pkg, err := prgrm.GetCurrentPackage()
	if err != nil {
		panic(err)
	}

	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S1239.
	declCXLine := ast.MakeCXLineExpression(prgrm, CurrentFile, LineNo, LineStr)
	// Declaration expression to handle the inline initialization.
	// For example, `var foo i32 = 11` needs to be divided into two expressions:
	// one that declares `foo`, and another that assigns 11 to `foo`
	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S1240.
	decl := ast.MakeAtomicOperatorExpression(prgrm, nil)
	expressionIdx := decl.Index
	prgrm.CXAtomicOps[expressionIdx].Package = ast.CXPackageIndex(pkg.Index)

	declarationSpecifiers.Name = declarator.Name
	declarationSpecifiers.ArgDetails.FileLine = declarator.ArgDetails.FileLine
	declarationSpecifiers.Package = ast.CXPackageIndex(pkg.Index)
	declarationSpecifiers.PreviouslyDeclared = true
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now an arena insert, site S1241.
	declSpecIdx := prgrm.AddCXArgInArray(declarationSpecifiers)

	// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
	// Now an arena getter, site S1242.
	localVarTypeSig := ast.GetCXTypeSignatureRepresentationOfCXArg(prgrm, prgrm.GetCXArgFromArray(declSpecIdx))
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now an arena insert, site S1243.
	localVarTypeSigIdx := prgrm.AddCXTypeSignatureInArray(localVarTypeSig)
	prgrm.CXAtomicOps[expressionIdx].AddOutput(prgrm, localVarTypeSigIdx)

	// Checking if something is supposed to be initialized
	// and if `initializer` actually contains something.
	// If `initializer` is `nil`, this means that an expression
	// equivalent to nil was used, such as `[]i32{}`.
	if doesInitialize && initializer != nil {
		processLocalInitialization(prgrm, pkg, &initializer, localVarTypeSigIdx, declSpecIdx)

		// OBJMODEL: state.CXExpressionMng (manager storage) or []CXExpressionGid, phase 4 #346.
		// Now a slice of objects, site S1244.
		return append([]ast.CXExpression{*declCXLine, *decl}, initializer...)
	} else {
		// There is no initialization.
		// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
		// Now a constructor call, site S1245.
		expr := ast.MakeAtomicOperatorExpression(prgrm, nil)
		cxAtomicOpIdx := expr.Index
		prgrm.CXAtomicOps[cxAtomicOpIdx].Package = ast.CXPackageIndex(pkg.Index)
		prgrm.CXAtomicOps[cxAtomicOpIdx].AddOutput(prgrm, localVarTypeSigIdx)

		// OBJMODEL: state.CXExpressionMng (manager storage) or []CXExpressionGid, phase 4 #346.
		// Now a slice of objects, site S1246.
		return []ast.CXExpression{*declCXLine, *expr}
	}

}
