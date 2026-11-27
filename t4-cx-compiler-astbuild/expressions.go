package astbuild

import (
	"slices"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	stringpool "github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// ReturnExpressions stores the `Size` of the return arguments represented by `Expressions`.
// For example: `return foo() + bar()` is a set of 3 expressions and they represent a single return argument
type ReturnExpressions struct {
	Size int
	// Line and Column are where the return keyword is, or 0.
	Line   int
	Column int
	// OBJMODEL: []CXExpressionGid or CXExpressionGid, phase 5 #347.
	// Now objects held by value, site S0607.
	Expressions []ast.CXExpression
}

// IterationExpressions creates series of expressions that will create
// a for loop condition.
//
// Input arguments description:
//
//		prgrm - a CXProgram that contains all the data and arrays of the program.
//		initializeExprs - contains the initialization of the variables used
//						  for the condition of the iteration expression.
//	 conditionExprs - contains the condition of the iteration expression.
//	 incrementExprs - contains the increment expr for the variable used
//						 in the condition.
//	 statementExprs - contains the statements inside the iteration expression.
func IterationExpressions(prgrm *ast.CXProgram,
	initializeExprs []ast.CXExpression,
	conditionExprs []ast.CXExpression,
	incrementExprs []ast.CXExpression,
	statementExprs []ast.CXExpression) []ast.CXExpression {
	jmpFn := ast.NativeFunctionOf(prgrm, opcodetable.OpJmp)

	pkg, err := prgrm.CurrentPackageGet()
	if err != nil {
		panic(err)
	}

	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0608.
	upExprCXLine := CXLineExpressionCurrentMake(prgrm)
	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0609.
	upExpr := ast.AtomicOperatorExpressionMake(prgrm, jmpFn)
	upExpressionIdx := upExpr.Index

	prgrm.CXAtomicOps[upExpressionIdx].Package = ast.CXPackageIndex(pkg.Index)

	lastCondExpressionIdx := conditionExprs[len(conditionExprs)-1].Index
	lastCondExpressionOperator := prgrm.CxIrState.CxIrStateFunctionGetOrNil(prgrm.CXAtomicOps[lastCondExpressionIdx].Operator)
	// A literal or variable condition is read in place, as in SelectionExpressions.
	// Running it would zero it as a declaration (skycoin-cx#20).
	loopConditionExprs := conditionExprs
	conditionInPlace := lastCondExpressionOperator == nil
	if conditionInPlace {
		loopConditionExprs = conditionExprs[:len(conditionExprs)-1]
	}

	// -2 for the cx line expression addition for up and down expr
	upLines := ((len(statementExprs) + len(incrementExprs) + len(loopConditionExprs) + 2) * -1) - 2
	downLines := 0

	trueArg := PrimaryWrite(prgrm, valuetypes.BOOL, encoder.Serialize(true), false)
	typeSigIdx := ast.CXTypeSignatureOfCXArgBuilderAlloc(prgrm, trueArg)
	prgrm.CXAtomicOps[upExpressionIdx].InputAdd(prgrm, typeSigIdx)
	prgrm.CXAtomicOps[upExpressionIdx].ThenLines = upLines
	prgrm.CXAtomicOps[upExpressionIdx].ElseLines = downLines

	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0611.
	downExprCXLine := CXLineExpressionCurrentMake(prgrm)
	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0612.
	downExpr := ast.AtomicOperatorExpressionMake(prgrm, jmpFn)
	downExpressionIdx := downExpr.Index

	prgrm.CXAtomicOps[downExpressionIdx].Package = ast.CXPackageIndex(pkg.Index)

	lastCondExpressionOperatorOutputs := lastCondExpressionOperator.OutputsGet(prgrm)
	if len(prgrm.CXAtomicOps[lastCondExpressionIdx].OutputsGet(prgrm)) < 1 {
		if len(lastCondExpressionOperatorOutputs) == 0 {
			// At the condition, which the parser rule of the loop makes the token position.
			list := &prgrm.CxIrState.CompileErrorList
			prgrm.CXProgramCompileErrorAddPosition(ast.CompileErrorCodeLoopConditionNoValue, list.CompileErrorListTokenFile, list.CompileErrorListTokenLine, list.CompileErrorListTokenColumn, ast.CompileErrorOperands{})
			// The condition reads a placeholder, so it reports nothing more.
			lastCondExpressionOperatorOutputs = []ast.CXTypeSignatureIndex{invalidPlaceholderTypeSignature(prgrm, ast.CXPackageIndex(pkg.Index))}
		}
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0614.
		lastCondExpressionOperatorOutputTypeSig := prgrm.CXTypeSignatureFromArrayGet(lastCondExpressionOperatorOutputs[0])

		var lastCondExpressionOperatorOutputType valuetypes.Code
		var typeSigIdx ast.CXTypeSignatureIndex
		if lastCondExpressionOperatorOutputTypeSig.Type == ast.TypeCXArgumentDeprecate {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0615.
			lastCondExpressionOperatorOutputArg := prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(lastCondExpressionOperatorOutputTypeSig.Meta))
			lastCondExpressionOperatorOutputType = lastCondExpressionOperatorOutputArg.Type
			_, predicate := ast.ArgumentMake(prgrm, temporaryVariableNameGenerate(prgrm, opcodetable.LocalPrefix), CurrentFile, LineNo)
			predicate.TypeSet(lastCondExpressionOperatorOutputType)
			predicate.Package = ast.CXPackageIndex(pkg.Index)
			predicate.PreviouslyDeclared = true

			typeSigIdx = ast.CXTypeSignatureOfCXArgBuilderAlloc(prgrm, predicate)
		} else if lastCondExpressionOperatorOutputTypeSig.Type == ast.TypeAtomic {
			_, newTypeSig := prgrm.CXTypeSignatureCopyAlloc(lastCondExpressionOperatorOutputTypeSig)
			newTypeSig.Name = temporaryVariableNameGenerate(prgrm, opcodetable.LocalPrefix)
			newTypeSig.Package = ast.CXPackageIndex(pkg.Index)
			newTypeSig.Offset = valuetypes.Pointer(0)
			typeSigIdx = newTypeSig.Index
		} else if lastCondExpressionOperatorOutputTypeSig.Type == ast.TypePointerAtomic {
			_, newTypeSig := prgrm.CXTypeSignatureCopyAlloc(lastCondExpressionOperatorOutputTypeSig)
			newTypeSig.Name = temporaryVariableNameGenerate(prgrm, opcodetable.LocalPrefix)
			newTypeSig.Package = ast.CXPackageIndex(pkg.Index)
			newTypeSig.Offset = valuetypes.Pointer(0)
			typeSigIdx = newTypeSig.Index
		} else {
			panic("type is not known")
		}

		prgrm.CXAtomicOps[lastCondExpressionIdx].OutputAdd(prgrm, typeSigIdx)
		prgrm.CXAtomicOps[downExpressionIdx].InputAdd(prgrm, typeSigIdx)
	} else if conditionInPlace {
		prgrm.CXAtomicOps[downExpressionIdx].InputAdd(prgrm, prgrm.CXAtomicOps[lastCondExpressionIdx].OutputsGet(prgrm)[0])
	} else {
		predicateTypeSigIdx := prgrm.CXAtomicOps[lastCondExpressionIdx].OutputsGet(prgrm)[0]
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0624.
		predicateTypeSig := prgrm.CXTypeSignatureFromArrayGet(predicateTypeSigIdx)
		var predicateIdx ast.CXArgumentIndex
		if predicateTypeSig.Type == ast.TypeCXArgumentDeprecate {
			predicateIdx = ast.CXArgumentIndex(predicateTypeSig.Meta)
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1675.
			prgrm.CXArgGet(ast.CXArgumentIndex(predicateIdx)).Package = ast.CXPackageIndex(pkg.Index)
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1676.
			prgrm.CXArgGet(ast.CXArgumentIndex(predicateIdx)).PreviouslyDeclared = true
		} else if predicateTypeSig.Type == ast.TypeAtomic || predicateTypeSig.Type == ast.TypePointerAtomic {
			// do nothing
		} else {
			panic("type is not known")
		}

		prgrm.CXAtomicOps[downExpressionIdx].InputAdd(prgrm, predicateTypeSigIdx)
	}

	thenLines := 0
	// + 1 for the cx line expression addition for the down expr jmp
	elseLines := len(incrementExprs) + len(statementExprs) + 1 + 1

	// A resolved break or continue becomes a true jmp, so an enclosing loop
	// does not retarget it (skycoin-cx#39).
	for i, statementExpr := range statementExprs {
		if statementExpr.Type == ast.CXOperationTypeLine {
			continue
		}
		statementExpressionIdx := statementExpr.Index
		if statementExpr.BreakOperator(prgrm) {
			prgrm.CXAtomicOps[statementExpressionIdx].ThenLines = elseLines - i - 1
			prgrm.CXAtomicOps[statementExpressionIdx].Operator = prgrm.NativeFunctionInArrayAdd(jmpFn)
		} else if statementExpr.ContinueOperator(prgrm) {
			prgrm.CXAtomicOps[statementExpressionIdx].ThenLines = len(statementExprs) - i - 1
			prgrm.CXAtomicOps[statementExpressionIdx].Operator = prgrm.NativeFunctionInArrayAdd(jmpFn)
		}
	}

	prgrm.CXAtomicOps[downExpressionIdx].ThenLines = thenLines
	prgrm.CXAtomicOps[downExpressionIdx].ElseLines = elseLines

	exprs := initializeExprs
	exprs = append(exprs, loopConditionExprs...)
	exprs = append(exprs, *downExprCXLine, *downExpr)
	exprs = append(exprs, statementExprs...)
	exprs = append(exprs, incrementExprs...)
	exprs = append(exprs, *upExprCXLine, *upExpr)

	DefineNewScope(exprs)

	return exprs
}

// trueJmpExpressions makes a true jump expression for BREAKS and CONTINUES.
//
// Input arguments description:
//
//		prgrm - a CXProgram that contains all the data and arrays of the program.
//		opcode - the opcode of the expression. Currently used as either a
//	 		 BREAK or CONTINUE expression.
func trueJmpExpressions(prgrm *ast.CXProgram, opcode int) []ast.CXExpression {
	pkg, err := prgrm.CurrentPackageGet()
	if err != nil {
		panic(err)
	}

	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0625.
	exprCXLine := CXLineExpressionCurrentMake(prgrm)
	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0626.
	expr := ast.AtomicOperatorExpressionMake(prgrm, ast.NativeFunctionOf(prgrm, opcode))
	expressionIdx := expr.Index

	trueArg := PrimaryWrite(prgrm, valuetypes.BOOL, encoder.Serialize(true), false)
	typeSigIdx := ast.CXTypeSignatureOfCXArgBuilderAlloc(prgrm, trueArg)
	prgrm.CXAtomicOps[expressionIdx].InputAdd(prgrm, typeSigIdx)
	prgrm.CXAtomicOps[expressionIdx].Package = ast.CXPackageIndex(pkg.Index)

	// OBJMODEL: state.CXExpressionMng (manager storage) or []CXExpressionGid, phase 4 #346.
	// Now a slice of objects, site S0628.
	return []ast.CXExpression{*exprCXLine, *expr}
}

// BreakExpressions makes a true jump expression for BREAKS.
//
// Input arguments description:
//
//	prgrm - a CXProgram that contains all the data and arrays of the program.
func BreakExpressions(prgrm *ast.CXProgram) []ast.CXExpression {
	exprs := trueJmpExpressions(prgrm, opcodetable.OpBreak)
	return exprs
}

// ContinueExpressions makes a true jump expression for CONTINUES.
//
// Input arguments description:
//
//	prgrm - a CXProgram that contains all the data and arrays of the program.
func ContinueExpressions(prgrm *ast.CXProgram) []ast.CXExpression {
	exprs := trueJmpExpressions(prgrm, opcodetable.OpContinue)
	return exprs
}

// SelectionExpressions creates series of expressions that will create
// an if else condition.
//
// Input arguments description:
//
//		prgrm - a CXProgram that contains all the data and arrays of the program.
//	 conditionExprs - contains the condition of the selection expression.
//		thenExprs - contains the statements if condition is true.
//	 elseExprs - contains the statements if condition is false.
func SelectionExpressions(prgrm *ast.CXProgram, conditionExprs []ast.CXExpression, thenExprs []ast.CXExpression, elseExprs []ast.CXExpression) []ast.CXExpression {
	DefineNewScope(thenExprs)
	DefineNewScope(elseExprs)

	jmpFn := ast.NativeFunctionOf(prgrm, opcodetable.OpJmp)
	pkg, err := prgrm.CurrentPackageGet()
	if err != nil {
		panic(err)
	}

	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0629.
	ifExprCXLine := CXLineExpressionCurrentMake(prgrm)
	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0630.
	ifExpr := ast.AtomicOperatorExpressionMake(prgrm, jmpFn)
	ifExpressionIdx := ifExpr.Index
	prgrm.CXAtomicOps[ifExpressionIdx].Package = ast.CXPackageIndex(pkg.Index)

	lastCondExpressionIdx := conditionExprs[len(conditionExprs)-1].Index
	lastCondExpressionOperator := prgrm.CxIrState.CxIrStateFunctionGetOrNil(prgrm.CXAtomicOps[lastCondExpressionIdx].Operator)

	var predicateTypeSigIdx ast.CXTypeSignatureIndex
	if lastCondExpressionOperator == nil {
		// then it's a literal
		predicateTypeSigIdx = prgrm.CXAtomicOps[lastCondExpressionIdx].OutputsGet(prgrm)[0]
	} else {
		// then it's an expression
		_, predicate := ast.ArgumentMake(prgrm, temporaryVariableNameGenerate(prgrm, opcodetable.LocalPrefix), CurrentFile, LineNo)
		lastCondExpressionOperatorOutputs := lastCondExpressionOperator.OutputsGet(prgrm)
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0633.
		lastCondExpressionOperatorOutputTypeSig := prgrm.CXTypeSignatureFromArrayGet(lastCondExpressionOperatorOutputs[0])

		var lastCondExpressionOperatorOutputArgType valuetypes.Code
		if lastCondExpressionOperatorOutputTypeSig.Type == ast.TypeCXArgumentDeprecate {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0634.
			lastCondExpressionOperatorOutputArg := prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(lastCondExpressionOperatorOutputTypeSig.Meta))
			lastCondExpressionOperatorOutputArgType = lastCondExpressionOperatorOutputArg.Type
		} else if lastCondExpressionOperatorOutputTypeSig.Type == ast.TypeAtomic {
			lastCondExpressionOperatorOutputArgType = valuetypes.Code(lastCondExpressionOperatorOutputTypeSig.Meta)
		} else if lastCondExpressionOperatorOutputTypeSig.Type == ast.TypePointerAtomic {
			lastCondExpressionOperatorOutputArgType = valuetypes.Code(lastCondExpressionOperatorOutputTypeSig.Meta)
		} else {
			panic("type is not known")
		}
		predicate.TypeSet(lastCondExpressionOperatorOutputArgType)
		predicate.PreviouslyDeclared = true
		predicate.Package = ast.CXPackageIndex(pkg.Index)

		predicateTypeSigIdx = ast.CXTypeSignatureOfCXArgBuilderAlloc(prgrm, predicate)
		prgrm.CXAtomicOps[lastCondExpressionIdx].OutputAdd(prgrm, predicateTypeSigIdx)
	}
	prgrm.CXAtomicOps[ifExpressionIdx].InputAdd(prgrm, predicateTypeSigIdx)

	thenLines := 0
	// + 1 for cx line expression addition
	elseLines := len(thenExprs) + 1 + 1

	prgrm.CXAtomicOps[ifExpressionIdx].ThenLines = thenLines
	prgrm.CXAtomicOps[ifExpressionIdx].ElseLines = elseLines

	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0636.
	skipExprCXLine := CXLineExpressionCurrentMake(prgrm)
	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0637.
	skipExpr := ast.AtomicOperatorExpressionMake(prgrm, jmpFn)
	skipExpressionIdx := skipExpr.Index

	prgrm.CXAtomicOps[skipExpressionIdx].Package = ast.CXPackageIndex(pkg.Index)

	skipLines := len(elseExprs)

	trueArg := PrimaryWrite(prgrm, valuetypes.BOOL, encoder.Serialize(true), false)
	typeSigIdx := ast.CXTypeSignatureOfCXArgBuilderAlloc(prgrm, trueArg)
	prgrm.CXAtomicOps[skipExpressionIdx].InputAdd(prgrm, typeSigIdx)
	prgrm.CXAtomicOps[skipExpressionIdx].ThenLines = skipLines
	prgrm.CXAtomicOps[skipExpressionIdx].ElseLines = 0

	var exprs []ast.CXExpression
	if lastCondExpressionOperator != nil {
		exprs = append(exprs, conditionExprs...)
	} else {
		exprs = append(exprs, conditionExprs[:len(conditionExprs)-1]...)
	}

	exprs = append(exprs, *ifExprCXLine, *ifExpr)
	exprs = append(exprs, thenExprs...)
	exprs = append(exprs, *skipExprCXLine, *skipExpr)
	exprs = append(exprs, elseExprs...)

	return exprs
}

// resolveTypeForUnd tries to determine the type that will be returned from an expression
func resolveTypeForUnd(prgrm *ast.CXProgram, expr *ast.CXExpression) valuetypes.Code {
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
	// Now an arena getter, site S0639.
	expression, err := prgrm.CXAtomicOpGet(expr.Index)
	if err != nil {
		panic(err)
	}

	expressionOperator := prgrm.CxIrState.CxIrStateFunctionGetOrNil(expression.Operator)

	expressionInputs := expression.InputsGet(prgrm)
	expressionOutputs := expression.OutputsGet(prgrm)
	if len(expressionInputs) > 0 {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0641.
		expressionInputTypeSig := prgrm.CXTypeSignatureFromArrayGet(expressionInputs[0])

		var expressionInputArgType valuetypes.Code
		if expressionInputTypeSig.Type == ast.TypeCXArgumentDeprecate {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0642.
			expressionInputArg := prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(expressionInputTypeSig.Meta))
			expressionInputArgType = expressionInputArg.Type
		} else if expressionInputTypeSig.Type == ast.TypeAtomic {
			expressionInputArgType = valuetypes.Code(expressionInputTypeSig.Meta)
		} else if expressionInputTypeSig.Type == ast.TypePointerAtomic {
			expressionInputArgType = valuetypes.Code(expressionInputTypeSig.Meta)
		} else {
			panic("type is not known")
		}

		// it's a literal
		return expressionInputArgType
	}

	if len(expressionOutputs) > 0 {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0643.
		expressionOutputTypeSig := prgrm.CXTypeSignatureFromArrayGet(expressionOutputs[0])
		var expressionOutputArgType valuetypes.Code
		if expressionOutputTypeSig.Type == ast.TypeCXArgumentDeprecate {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0644.
			expressionOutputArg := prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(expressionOutputTypeSig.Meta))
			expressionOutputArgType = expressionOutputArg.Type
		} else if expressionOutputTypeSig.Type == ast.TypeAtomic {
			expressionOutputArgType = valuetypes.Code(expressionOutputTypeSig.Meta)
		} else if expressionOutputTypeSig.Type == ast.TypePointerAtomic {
			expressionOutputArgType = valuetypes.Code(expressionOutputTypeSig.Meta)
		} else {
			panic("type is not known")
		}

		// it's an expression with an output
		return expressionOutputArgType
	}

	if expressionOperator == nil {
		// the expression doesn't return anything
		return -1
	}
	expressionOperatorOutputs := expressionOperator.OutputsGet(prgrm)
	if len(expressionOperatorOutputs) > 0 {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0645.
		expressionOperatorOutputTypeSig := prgrm.CXTypeSignatureFromArrayGet(expressionOperatorOutputs[0])

		var expressionOperatorOutputArgType valuetypes.Code
		if expressionOperatorOutputTypeSig.Type == ast.TypeCXArgumentDeprecate {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0646.
			expressionOperatorOutputArg := prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(expressionOperatorOutputTypeSig.Meta))
			expressionOperatorOutputArgType = expressionOperatorOutputArg.Type
		} else if expressionOperatorOutputTypeSig.Type == ast.TypeAtomic {
			expressionOperatorOutputArgType = valuetypes.Code(expressionOperatorOutputTypeSig.Meta)
		} else if expressionOperatorOutputTypeSig.Type == ast.TypePointerAtomic {
			expressionOperatorOutputArgType = valuetypes.Code(expressionOperatorOutputTypeSig.Meta)
		} else {
			panic("type is not known")
		}

		// always return first output's type
		return expressionOperatorOutputArgType
	}

	// error
	return -1
}

// TemporaryVariable ...
// TODO: Delete this function; only called by next function
func TemporaryVariable(prgrm *ast.CXProgram, nameGid stringpool.StringGid) bool {
	name := prgrm.CXProgramNameText(nameGid)
	if len(name) >= len(opcodetable.LocalPrefix) && name[:len(opcodetable.LocalPrefix)] == opcodetable.LocalPrefix {
		return true
	}
	return false
}

// OperatorExpression creates an expression for an expression OP expression instance.
// i.e. ((expr+expr) * (expr-expr))
//
// Input arguments description:
//
//		prgrm - a CXProgram that contains all the data and arrays of the program.
//	 leftExprs - exprs on the left side of the operator expression
//		rightExprs - exprs on the right side of the operator expression
//		opcode - the opcode of the operator.
func OperatorExpression(prgrm *ast.CXProgram, leftExprs []ast.CXExpression, rightExprs []ast.CXExpression, opcode int) (out []ast.CXExpression) {
	// An empty side failed to compile and its error is recorded.
	if len(leftExprs) == 0 || len(rightExprs) == 0 {
		return nil
	}
	pkg, err := prgrm.CurrentPackageGet()
	if err != nil {
		panic(err)
	}

	outputlessValueCheck(prgrm, leftExprs)
	outputlessValueCheck(prgrm, rightExprs)
	lastLeftExpressionIdx := leftExprs[len(leftExprs)-1].Index
	lastLeftExpressionOperator := prgrm.CxIrState.CxIrStateFunctionGetOrNil(prgrm.CXAtomicOps[lastLeftExpressionIdx].Operator)
	if len(prgrm.CXAtomicOps[lastLeftExpressionIdx].OutputsGet(prgrm)) < 1 {
		operandResultOutputAdd(prgrm, pkg, leftExprs, lastLeftExpressionOperator)
	}

	lastRightExpressionIdx := rightExprs[len(rightExprs)-1].Index
	lastRightExpressionOperator := prgrm.CxIrState.CxIrStateFunctionGetOrNil(prgrm.CXAtomicOps[lastRightExpressionIdx].Operator)
	if len(prgrm.CXAtomicOps[lastRightExpressionIdx].OutputsGet(prgrm)) < 1 {
		operandResultOutputAdd(prgrm, pkg, rightExprs, lastRightExpressionOperator)
	}

	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0670.
	exprCXLine := CXLineExpressionCurrentMake(prgrm)
	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0671.
	expr := ast.AtomicOperatorExpressionMake(prgrm, ast.NativeFunctionOf(prgrm, opcode))
	expressionIdx := expr.Index

	// we can't know the type until we compile the full function
	prgrm.CXAtomicOps[expressionIdx].Package = ast.CXPackageIndex(pkg.Index)

	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now an arena getter, site S0672.
	lastLeftExpressionOutputTypeSig := prgrm.CXTypeSignatureFromArrayGet(prgrm.CXAtomicOps[lastLeftExpressionIdx].OutputsGet(prgrm)[0])
	switch lastLeftExpressionOutputTypeSig.Type {
	case ast.TypeCXArgumentDeprecate, ast.TypeAtomic, ast.TypePointerAtomic, ast.TypeSliceAtomic:
	default:
		panic("type is not known")
	}

	if operandIndexed(prgrm, lastLeftExpressionOutputTypeSig) || lastLeftExpressionOperator != nil {
		typeSigIdx := prgrm.CXAtomicOps[lastLeftExpressionIdx].OutputsGet(prgrm)[0]
		// then it's a function call or an array access
		prgrm.CXAtomicOps[expressionIdx].InputAdd(prgrm, typeSigIdx)

		// The placeholder of f()[i] has no operator and is dropped like a variable.
		if lastLeftExpressionOperator != nil && TemporaryVariable(prgrm, lastLeftExpressionOutputTypeSig.Name) {
			out = append(out, leftExprs...)
		} else {
			out = append(out, leftExprs[:len(leftExprs)-1]...)
		}
	} else {
		typeSigIdx := prgrm.CXAtomicOps[lastLeftExpressionIdx].OutputsGet(prgrm)[0]
		prgrm.CXAtomicOps[expressionIdx].InputAdd(prgrm, typeSigIdx)
		// A variable read after a method call keeps the call.
		out = append(out, leftExprs[:len(leftExprs)-1]...)
	}
	leftCount := len(out)

	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now an arena getter, site S0676.
	lastRightExpressionOutputTypeSig := prgrm.CXTypeSignatureFromArrayGet(prgrm.CXAtomicOps[lastRightExpressionIdx].OutputsGet(prgrm)[0])
	// nil for an atomic output.
	var lastRightExpressionOutputArg *ast.CXArgument
	if lastRightExpressionOutputTypeSig.Type == ast.TypeCXArgumentDeprecate {
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S0678.
		lastRightExpressionOutputArg = prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(lastRightExpressionOutputTypeSig.Meta))
	} else if lastRightExpressionOutputTypeSig.Type == ast.TypeAtomic || lastRightExpressionOutputTypeSig.Type == ast.TypePointerAtomic {
		// An atomic output has no indexes.
	} else if lastRightExpressionOutputTypeSig.Type != ast.TypeSliceAtomic {
		panic("type is not known")
	}

	if (opcode == opcodetable.OpBoolAnd || opcode == opcodetable.OpBoolOr) && lastRightExpressionOperator == nil && (operandIndexed(prgrm, lastRightExpressionOutputTypeSig) || lastRightExpressionOutputArg != nil && len(lastRightExpressionOutputArg.DereferenceOperations) > 0) {
		lastRightExpressionOperator = shortCircuitOperandExpression(prgrm, lastRightExpressionIdx)
		lastRightExpressionOutputTypeSig = prgrm.CXTypeSignatureFromArrayGet(prgrm.CXAtomicOps[lastRightExpressionIdx].OutputsGet(prgrm)[0])
	}

	if operandIndexed(prgrm, lastRightExpressionOutputTypeSig) || lastRightExpressionOperator != nil {
		typeSigIdx := prgrm.CXAtomicOps[lastRightExpressionIdx].OutputsGet(prgrm)[0]
		// then it's a function call or an array access
		prgrm.CXAtomicOps[expressionIdx].InputAdd(prgrm, typeSigIdx)

		if lastRightExpressionOperator != nil && TemporaryVariable(prgrm, lastRightExpressionOutputTypeSig.Name) {
			out = append(out, rightExprs...)
		} else {
			out = append(out, rightExprs[:len(rightExprs)-1]...)
		}
	} else {
		typeSigIdx := prgrm.CXAtomicOps[lastRightExpressionIdx].OutputsGet(prgrm)[0]
		prgrm.CXAtomicOps[expressionIdx].InputAdd(prgrm, typeSigIdx)
		out = append(out, rightExprs[:len(rightExprs)-1]...)
	}

	if opcode == opcodetable.OpBoolAnd || opcode == opcodetable.OpBoolOr {
		out = shortCircuitExpressions(prgrm, out, leftCount, prgrm.CXAtomicOps[expressionIdx].InputsGet(prgrm)[0], opcode)
	}
	out = append(out, *exprCXLine, *expr)

	return
}

// operandResultOutputAdd gives the last expression of exprs, a call or an
// operator with no output yet, a temporary output of the type it returns.
func operandResultOutputAdd(prgrm *ast.CXProgram, pkg *ast.CXPackage, exprs []ast.CXExpression, operator *ast.CXFunction) {
	exprIdx := exprs[len(exprs)-1].Index
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now an arena getter, site S0648.
	resultTypeSig := prgrm.CXTypeSignatureFromArrayGet(operator.OutputsGet(prgrm)[0])

	var typeSigIdx ast.CXTypeSignatureIndex
	if resultTypeSig.Type == ast.TypeCXArgumentDeprecate {
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S0649.
		resultArg := prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(resultTypeSig.Meta))

		_, out := ast.ArgumentMake(prgrm, temporaryVariableNameGenerate(prgrm, opcodetable.LocalPrefix), CurrentFile, LineNo)
		out.TypeSet(resolveTypeForUnd(prgrm, &exprs[len(exprs)-1]))
		out.Size = resultArg.Size
		out.Type = resultArg.Type
		out.PointerTargetType = resultArg.PointerTargetType
		out.Package = ast.CXPackageIndex(pkg.Index)
		out.PreviouslyDeclared = true

		typeSigIdx = ast.CXTypeSignatureOfCXArgBuilderAlloc(prgrm, out)
	} else if resultTypeSig.Type == ast.TypeAtomic {
		argDetailsGid := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugAlloc()
		argDetails := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugGet(argDetailsGid)
		argDetails.FileName = CurrentFile
		argDetails.FileLine = LineNo
		argDetails.FileColumn = prgrm.CXProgramCompileErrorColumn(argDetails.FileName, argDetails.FileLine)
		_, typeSig := prgrm.CXTypeSignatureAlloc()
		typeSig.ArgDetails = argDetailsGid
		typeSig.Name = temporaryVariableNameGenerate(prgrm, opcodetable.LocalPrefix)
		typeSig.Package = resultTypeSig.Package
		typeSig.Type = ast.TypeAtomic
		typeSig.Meta = resultTypeSig.Meta
		typeSig.Offset = resultTypeSig.Offset

		typeSigIdx = typeSig.Index
	} else if resultTypeSig.Type == ast.TypePointerAtomic {
		argDetailsGid := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugAlloc()
		argDetails := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugGet(argDetailsGid)
		argDetails.FileName = CurrentFile
		argDetails.FileLine = LineNo
		argDetails.FileColumn = prgrm.CXProgramCompileErrorColumn(argDetails.FileName, argDetails.FileLine)
		_, typeSig := prgrm.CXTypeSignatureAlloc()
		typeSig.ArgDetails = argDetailsGid
		typeSig.Name = temporaryVariableNameGenerate(prgrm, opcodetable.LocalPrefix)
		typeSig.Package = resultTypeSig.Package
		typeSig.Type = ast.TypePointerAtomic
		typeSig.Meta = resultTypeSig.Meta
		typeSig.Offset = resultTypeSig.Offset

		typeSigIdx = typeSig.Index
	} else {
		panic("type is not known")
	}

	prgrm.CXAtomicOps[exprIdx].OutputAdd(prgrm, typeSigIdx)
}

// operandIndexed reports whether the operand sig reads an element, as a[i]
// or the slice placeholder of f()[i] do.
func operandIndexed(prgrm *ast.CXProgram, sig *ast.CXTypeSignature) bool {
	switch sig.Type {
	case ast.TypeCXArgumentDeprecate:
		return len(prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(sig.Meta)).Indexes) > 0
	case ast.TypeSliceAtomic:
		return len(prgrm.CXTypeSignatureArrayFromArrayGet(sig.Meta).Indexes) > 0
	}
	return false
}

// shortCircuitLinesPending marks the skip of a short-circuit jmp until ShortCircuitJumpsResolve sets it.
const shortCircuitLinesPending = -1 << 30

// shortCircuitExpressions puts a jmp on the left operand before the right operand's expressions.
// The bool op still runs, and when the right side is skipped the left operand alone decides it.
func shortCircuitExpressions(prgrm *ast.CXProgram, exprs []ast.CXExpression, leftCount int, leftTypeSigIdx ast.CXTypeSignatureIndex, opcode int) []ast.CXExpression {
	if leftCount == len(exprs) {
		return exprs
	}
	pkg, err := prgrm.CurrentPackageGet()
	if err != nil {
		panic(err)
	}

	jmpCXLine := CXLineExpressionCurrentMake(prgrm)
	jmpExpr := ast.AtomicOperatorExpressionMake(prgrm, ast.NativeFunctionOf(prgrm, opcodetable.OpJmp))
	jmpOp := &prgrm.CXAtomicOps[jmpExpr.Index]
	jmpOp.Package = ast.CXPackageIndex(pkg.Index)
	jmpOp.InputAdd(prgrm, shortCircuitOperandCopy(prgrm, leftTypeSigIdx))
	if opcode == opcodetable.OpBoolOr {
		jmpOp.ThenLines = shortCircuitLinesPending
	} else {
		jmpOp.ElseLines = shortCircuitLinesPending
	}

	out := make([]ast.CXExpression, 0, len(exprs)+2)
	out = append(out, exprs[:leftCount]...)
	out = append(out, *jmpCXLine, *jmpExpr)
	return append(out, exprs[leftCount:]...)
}

// shortCircuitOperandCopy returns a signature of its own for the jmp of the
// left operand leftTypeSigIdx. The pass resolves the fields of an argument
// in place, so the jmp and the bool op cannot share item.flag.
func shortCircuitOperandCopy(prgrm *ast.CXProgram, leftTypeSigIdx ast.CXTypeSignatureIndex) ast.CXTypeSignatureIndex {
	sig := prgrm.CXTypeSignatureFromArrayGet(leftTypeSigIdx)
	if sig.Type != ast.TypeCXArgumentDeprecate {
		return leftTypeSigIdx
	}
	original := prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(sig.Meta))
	arg := argumentCopy(prgrm, original)
	argumentDebugPositionCopy(prgrm, arg.ArgDetails, original.ArgDetails)
	copied := ast.CXTypeSignatureOfCXArgDeprecateAlloc(prgrm, arg)
	prgrm.CXTypeSignatureFromArrayGet(copied).PassBy = sig.PassBy
	return copied
}

// shortCircuitOperandSame reports whether the jmp operand jmpSig is the
// left operand boolSig of its bool op, or the copy shortCircuitOperandCopy
// made of it.
func shortCircuitOperandSame(prgrm *ast.CXProgram, jmpSig ast.CXTypeSignatureIndex, boolSig ast.CXTypeSignatureIndex) bool {
	if jmpSig == boolSig {
		return true
	}
	jmp, left := prgrm.CXTypeSignatureFromArrayGet(jmpSig), prgrm.CXTypeSignatureFromArrayGet(boolSig)
	if StripNameNumber(prgrm, jmp.Name) != StripNameNumber(prgrm, left.Name) {
		return false
	}
	jmpFile, jmpLine, jmpColumn := ast.CXArgumentDebugPosition(prgrm, jmp.ArgDetails)
	leftFile, leftLine, leftColumn := ast.CXArgumentDebugPosition(prgrm, left.ArgDetails)
	return jmpFile == leftFile && jmpLine == leftLine && jmpColumn == leftColumn
}

// shortCircuitOperandExpression copies an indexed or dereferenced right operand into a bool temp,
// so the index or pointer is read only when the right side runs.
func shortCircuitOperandExpression(prgrm *ast.CXProgram, exprIdx int) *ast.CXFunction {
	pkg, err := prgrm.CurrentPackageGet()
	if err != nil {
		panic(err)
	}
	operandOp := &prgrm.CXAtomicOps[exprIdx]
	operandOp.Inputs = nil
	operandOp.InputAdd(prgrm, operandOp.OutputsGet(prgrm)[0])
	operandOp.Outputs = nil

	_, temporary := ast.ArgumentMake(prgrm, temporaryVariableNameGenerate(prgrm, opcodetable.LocalPrefix), CurrentFile, LineNo)
	temporary.TypeSet(valuetypes.BOOL)
	temporary.PreviouslyDeclared = true
	temporary.Package = ast.CXPackageIndex(pkg.Index)
	operandOp.OutputAdd(prgrm, ast.CXTypeSignatureOfCXArgBuilderAlloc(prgrm, temporary))

	operandOp.ThenLines = shortCircuitLinesPending
	operandOp.ElseLines = shortCircuitLinesPending
	identityIdx := prgrm.NativeFunctionInArrayAdd(ast.NativeFunctionOf(prgrm, opcodetable.OpIdentity))
	operandOp.Operator = identityIdx
	return prgrm.CxIrState.CXFunctionMng.CXFunctionGet(identityIdx)
}

// ShortCircuitJumpCheck reports whether exprs[currIndex] is a short-circuit jmp. Its bool op
// reads the same left operand and reports a type error on it, so the jmp is not type checked.
func ShortCircuitJumpCheck(prgrm *ast.CXProgram, exprs []ast.CXExpression, currIndex int) bool {
	jmpOp := &prgrm.CXAtomicOps[exprs[currIndex].Index]
	jmpOperator := prgrm.CxIrState.CxIrStateFunctionGetOrNil(jmpOp.Operator)
	if jmpOperator == nil || jmpOperator.AtomicOPCode != opcodetable.OpJmp || len(jmpOp.InputsGet(prgrm)) != 1 {
		return false
	}
	boolOpcode := opcodetable.OpBoolOr
	target := currIndex + jmpOp.ThenLines + 1
	if jmpOp.ThenLines == 0 {
		boolOpcode = opcodetable.OpBoolAnd
		target = currIndex + jmpOp.ElseLines + 1
	}
	if target <= currIndex+1 || target >= len(exprs) || exprs[target].Type != ast.CXOperationTypeAtomicOperator {
		return false
	}
	boolOp := &prgrm.CXAtomicOps[exprs[target].Index]
	boolOperator := prgrm.CxIrState.CxIrStateFunctionGetOrNil(boolOp.Operator)
	return boolOperator != nil && boolOperator.AtomicOPCode == boolOpcode && len(boolOp.InputsGet(prgrm)) > 0 && shortCircuitOperandSame(prgrm, jmpOp.InputsGet(prgrm)[0], boolOp.InputsGet(prgrm)[0])
}

// ShortCircuitJumpsResolve points each short-circuit jmp at its bool op, once the function's expressions are final.
// A jmp whose bool op is not found skips nothing, so both operands run.
func ShortCircuitJumpsResolve(prgrm *ast.CXProgram, exprs []ast.CXExpression) {
	for i := range exprs {
		if exprs[i].Type != ast.CXOperationTypeAtomicOperator {
			continue
		}
		jmpOp := &prgrm.CXAtomicOps[exprs[i].Index]
		if jmpOp.ThenLines != shortCircuitLinesPending && jmpOp.ElseLines != shortCircuitLinesPending {
			continue
		}
		if jmpOp.ThenLines == shortCircuitLinesPending && jmpOp.ElseLines == shortCircuitLinesPending {
			// An operand copy from shortCircuitOperandExpression.
			jmpOp.ThenLines = 0
			jmpOp.ElseLines = 0
			continue
		}
		boolOpcode := opcodetable.OpBoolAnd
		if jmpOp.ThenLines == shortCircuitLinesPending {
			boolOpcode = opcodetable.OpBoolOr
		}
		jmpOp.ThenLines = 0
		jmpOp.ElseLines = 0
		leftTypeSigIdx := jmpOp.InputsGet(prgrm)[0]
		for j := i + 1; j < len(exprs); j++ {
			if exprs[j].Type != ast.CXOperationTypeAtomicOperator {
				continue
			}
			boolOp := &prgrm.CXAtomicOps[exprs[j].Index]
			boolOperator := prgrm.CxIrState.CxIrStateFunctionGetOrNil(boolOp.Operator)
			if boolOperator == nil || boolOperator.AtomicOPCode != boolOpcode || len(boolOp.InputsGet(prgrm)) == 0 || !shortCircuitOperandSame(prgrm, leftTypeSigIdx, boolOp.InputsGet(prgrm)[0]) {
				continue
			}
			if boolOpcode == opcodetable.OpBoolOr {
				jmpOp.ThenLines = j - i - 1
			} else {
				jmpOp.ElseLines = j - i - 1
			}
			break
		}
	}
}

// UnaryExpression creates an expression for unary operations,
// operations that only need one input, '*', '&', '!', and '-'.
//
// Input arguments description:
//
//		prgrm - a CXProgram that contains all the data and arrays of the program.
//	 op - the unary operator, '*', '&', '!', and '-'.
//	 prevExprs - the array of expressions the unary expression belongs.
func UnaryExpression(prgrm *ast.CXProgram, op string, prevExprs []ast.CXExpression) []ast.CXExpression {
	if len(prevExprs) == 0 {
		return nil
	}
	lastPrevExpression, err := prgrm.CXAtomicOpFromExpressionsGet(prevExprs, len(prevExprs)-1)
	if err != nil {
		panic(err)
	}

	if op == "-" || op == "!" {
		if exprs, ok := unaryOperatorOnValue(prgrm, op, prevExprs); ok {
			return exprs
		}
	}

	if len(lastPrevExpression.OutputsGet(prgrm)) == 0 {
		prgrm.CXProgramCompileErrorAdd(ast.CompileErrorCodeInvalidIndirection, CurrentFile, LineNo, ast.CompileErrorOperands{})
		// A stand in typed types.INVALID calls the operand, so its uses report nothing more.
		lastPrevExpression.Operator = invalidFunctionIndex(prgrm, lastPrevExpression.Package, prgrm.CXProgramNameIntern(op))
		return prevExprs
	}

	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now an arena getter, site S0680.
	lastPrevExpressionOutputTypeSig := prgrm.CXTypeSignatureFromArrayGet(lastPrevExpression.OutputsGet(prgrm)[0])
	var lastPrevExpressionOutputArg *ast.CXArgument
	if lastPrevExpressionOutputTypeSig.Type == ast.TypeCXArgumentDeprecate {
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S0682.
		lastPrevExpressionOutputArg = prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(lastPrevExpressionOutputTypeSig.Meta))

		// Some properties need to be read from the base argument
		// due to how we calculate dereferences at the moment.
		lastPrevExpressionOutput := lastPrevExpressionOutputArg
		baseOut := lastPrevExpressionOutput
		exprOut := lastPrevExpressionOutput.AssignmentElementGet(prgrm)
		switch op {
		case "*":
			exprOut.DereferenceOperations = append(exprOut.DereferenceOperations, opcodetable.DerefPointer)
			exprOut.DeclarationSpecifiers = append(exprOut.DeclarationSpecifiers, opcodetable.DeclarationDeref)
		case "&":
			baseOut.PassBy = opcodetable.PassByReference

			exprOut.DeclarationSpecifiers = append(exprOut.DeclarationSpecifiers, opcodetable.DeclarationPointer)
		case "!":
			if pkg, err := prgrm.CurrentPackageGet(); err == nil {
				// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
				// Now a constructor call, site S0683.
				expr := ast.AtomicOperatorExpressionMake(prgrm, ast.NativeFunctionOf(prgrm, opcodetable.OpBoolNot))
				// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
				// Now an arena getter, site S0684.
				expression, err := prgrm.CXAtomicOpGet(expr.Index)
				if err != nil {
					panic(err)
				}
				expression.Package = ast.CXPackageIndex(pkg.Index)

				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S0685.
				typeSigIdx := ast.CXTypeSignatureOfCXArgAlloc(prgrm, prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(exprOut.Index)))
				expression.InputAdd(prgrm, typeSigIdx)
				// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
				// Now an object copy, site S0687.
				prevExprs[len(prevExprs)-1] = *expr
			} else {
				panic(err)
			}
		case "-":
			if pkg, err := prgrm.CurrentPackageGet(); err == nil {
				// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
				// Now a constructor call, site S0688.
				expr := ast.AtomicOperatorExpressionMake(prgrm, ast.NativeFunctionOf(prgrm, opcodetable.OpNeg))
				// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
				// Now an arena getter, site S0689.
				expression, err := prgrm.CXAtomicOpGet(expr.Index)
				if err != nil {
					panic(err)
				}
				expression.Package = ast.CXPackageIndex(pkg.Index)
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S0690.
				typeSigIdx := ast.CXTypeSignatureOfCXArgAlloc(prgrm, prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(exprOut.Index)))
				expression.InputAdd(prgrm, typeSigIdx)
				// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
				// Now an object copy, site S0692.
				prevExprs[len(prevExprs)-1] = *expr
			} else {
				panic(err)
			}
		}
	} else if lastPrevExpressionOutputTypeSig.Type == ast.TypeAtomic {
		switch op {
		case "*":
			// TODO: what is the most efficient alternative for this
			// exprOut.DereferenceOperations = append(exprOut.DereferenceOperations, constants.DerefPointer)
			// exprOut.DeclarationSpecifiers = append(exprOut.DeclarationSpecifiers, constants.DeclarationDeref)

			// a pointer type that we want to get its value
			lastPrevExpressionOutputTypeSig.Deref = true
		case "&":
			lastPrevExpressionOutputTypeSig.PassBy = opcodetable.PassByReference
		case "!":
			if pkg, err := prgrm.CurrentPackageGet(); err == nil {
				// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
				// Now a constructor call, site S0693.
				expr := ast.AtomicOperatorExpressionMake(prgrm, ast.NativeFunctionOf(prgrm, opcodetable.OpBoolNot))
				// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
				// Now an arena getter, site S0694.
				expression, err := prgrm.CXAtomicOpGet(expr.Index)
				if err != nil {
					panic(err)
				}
				expression.Package = ast.CXPackageIndex(pkg.Index)

				typeSigIdx := lastPrevExpressionOutputTypeSig.Index
				expression.InputAdd(prgrm, typeSigIdx)
				// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
				// Now an object copy, site S0695.
				prevExprs[len(prevExprs)-1] = *expr
			} else {
				panic(err)
			}
		case "-":
			if pkg, err := prgrm.CurrentPackageGet(); err == nil {
				// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
				// Now a constructor call, site S0696.
				expr := ast.AtomicOperatorExpressionMake(prgrm, ast.NativeFunctionOf(prgrm, opcodetable.OpNeg))
				// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
				// Now an arena getter, site S0697.
				expression, err := prgrm.CXAtomicOpGet(expr.Index)
				if err != nil {
					panic(err)
				}
				expression.Package = ast.CXPackageIndex(pkg.Index)

				typeSigIdx := lastPrevExpressionOutputTypeSig.Index
				expression.InputAdd(prgrm, typeSigIdx)
				// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
				// Now an object copy, site S0698.
				prevExprs[len(prevExprs)-1] = *expr
			} else {
				panic(err)
			}
		}

	} else {
		panic("type is not cx argument deprecate\n\n")
	}

	return prevExprs
}

// AssociateReturnExpressions associate the output of `retExprs` to the
// `idx`th output parameter of the current function.
//
// Input arguments description:
//
//		prgrm - a CXProgram that contains all the data and arrays of the program.
//	 idx - referring to the nth output parameter of the current function.
//		retExprs - the return expressions to be associated or linked to the
//				   output parameter of the current function.
func AssociateReturnExpressions(prgrm *ast.CXProgram, idx int, retExprs []ast.CXExpression) []ast.CXExpression {
	var pkg *ast.CXPackage
	var fn *ast.CXFunction
	var err error

	pkg, err = prgrm.CurrentPackageGet()
	if err != nil {
		panic(err)
	}

	fn, err = pkg.CurrentFunctionGet(prgrm)
	if err != nil {
		panic(err)
	}

	fnOutputs := fn.OutputsGet(prgrm)
	// ReturnExpressionsJmpAdd reports a return count that does not match.
	if idx >= len(fnOutputs) || len(retExprs) == 0 {
		return retExprs
	}
	lastExpr := retExprs[len(retExprs)-1]

	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now an arena getter, site S0699.
	fnOutputTypeSig := prgrm.CXTypeSignatureFromArrayGet(fnOutputs[idx])
	var outParam *ast.CXArgument
	// pointerArrayOut is the result of a *[N]T function, which return &a
	// writes the address to, as r = &a does.
	var pointerArrayOut *ast.CXArgument
	var typeSigIdx ast.CXTypeSignatureIndex
	if fnOutputTypeSig.Type == ast.TypeCXArgumentDeprecate {
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S0701.
		outParam = prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(fnOutputTypeSig.Meta))

		_, out := ast.ArgumentMake(prgrm, outParam.Name, CurrentFile, LineNo)
		out.TypeSet(outParam.Type)
		out.StructType = outParam.StructType
		out.PreviouslyDeclared = true

		if ast.CXArgumentTypeArrayAtomic(outParam) {
			// return [2]i32{1, 2} checks the literal against the whole result type.
			out.Lengths = slices.Clone(outParam.Lengths)
			out.DeclarationSpecifiers = slices.Clone(outParam.DeclarationSpecifiers)
			typeSigIdx = ast.CXTypeSignatureOfCXArgDeprecateAlloc(prgrm, out)
		} else if ast.CXArgumentTypePointerArrayAtomic(outParam) {
			// The result takes its pointer specifiers from its declaration.
			typeSigIdx = ast.CXTypeSignatureOfCXArgDeprecateAlloc(prgrm, out)
			pointerArrayOut = out
		} else {
			typeSigIdx = ast.CXTypeSignatureOfCXArgBuilderAlloc(prgrm, out)
		}
	} else if fnOutputTypeSig.Type == ast.TypeAtomic {
		argDetailsGid := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugAlloc()
		argDetails := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugGet(argDetailsGid)
		argDetails.FileName = CurrentFile
		argDetails.FileLine = LineNo
		argDetails.FileColumn = prgrm.CXProgramCompileErrorColumn(argDetails.FileName, argDetails.FileLine)
		_, newCXTypeSig := prgrm.CXTypeSignatureAlloc()
		newCXTypeSig.ArgDetails = argDetailsGid
		newCXTypeSig.Name = fnOutputTypeSig.Name
		newCXTypeSig.Package = fnOutputTypeSig.Package
		newCXTypeSig.Type = ast.TypeAtomic
		newCXTypeSig.Meta = int(fnOutputTypeSig.Type)
		newCXTypeSig.Offset = fnOutputTypeSig.Offset
		typeSigIdx = newCXTypeSig.Index
	} else if fnOutputTypeSig.Type == ast.TypePointerAtomic {
		argDetailsGid := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugAlloc()
		argDetails := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugGet(argDetailsGid)
		argDetails.FileName = CurrentFile
		argDetails.FileLine = LineNo
		argDetails.FileColumn = prgrm.CXProgramCompileErrorColumn(argDetails.FileName, argDetails.FileLine)
		_, newCXTypeSig := prgrm.CXTypeSignatureAlloc()
		newCXTypeSig.ArgDetails = argDetailsGid
		newCXTypeSig.Name = fnOutputTypeSig.Name
		newCXTypeSig.Package = fnOutputTypeSig.Package
		newCXTypeSig.Type = ast.TypePointerAtomic
		newCXTypeSig.Meta = int(fnOutputTypeSig.Type)
		newCXTypeSig.Offset = fnOutputTypeSig.Offset
		typeSigIdx = newCXTypeSig.Index
	} else if fnOutputTypeSig.Type == ast.TypeSliceAtomic {
		argDetailsGid := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugAlloc()
		argDetails := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugGet(argDetailsGid)
		argDetails.FileName = CurrentFile
		argDetails.FileLine = LineNo
		argDetails.FileColumn = prgrm.CXProgramCompileErrorColumn(argDetails.FileName, argDetails.FileLine)
		_, newCXTypeSig := prgrm.CXTypeSignatureAlloc()
		newCXTypeSig.ArgDetails = argDetailsGid
		newCXTypeSig.Name = fnOutputTypeSig.Name
		newCXTypeSig.Package = fnOutputTypeSig.Package
		newCXTypeSig.Type = ast.TypeSliceAtomic
		newCXTypeSig.Meta = fnOutputTypeSig.Meta
		newCXTypeSig.Offset = fnOutputTypeSig.Offset
		typeSigIdx = newCXTypeSig.Index
	} else if fnOutputTypeSig.Type == ast.TypeStruct || fnOutputTypeSig.Type == ast.TypeArrayAtomic || fnOutputTypeSig.Type == ast.TypePointerStruct || fnOutputTypeSig.Type == ast.TypePointerArrayAtomic || fnOutputTypeSig.Type == ast.TypePointerSliceAtomic {
		// A struct, array or pointer output, as in return T{x: 1} or return &T{}.
		_, newCXTypeSig := prgrm.CXTypeSignatureCopyAlloc(fnOutputTypeSig)
		argDetailsGid := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugAlloc()
		argDetails := prgrm.CxIrState.CXArgumentDebugMng.CXArgumentDebugGet(argDetailsGid)
		argDetails.FileName = CurrentFile
		argDetails.FileLine = LineNo
		argDetails.FileColumn = prgrm.CXProgramCompileErrorColumn(argDetails.FileName, argDetails.FileLine)
		newCXTypeSig.ArgDetails = argDetailsGid
		typeSigIdx = newCXTypeSig.Index
	} else {
		panic("type is not known")
	}

	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
	// Now an arena getter, site S0713.
	lastExpression, err := prgrm.CXAtomicOpGet(lastExpr.Index)
	if err != nil {
		panic(err)
	}

	lastExpressionOperator := prgrm.CxIrState.CxIrStateFunctionGetOrNil(lastExpression.Operator)
	if lastExpressionOperator == nil {
		opIdx := prgrm.NativeFunctionInArrayAdd(ast.NativeFunctionOf(prgrm, opcodetable.OpIdentity))
		lastExpression.Operator = opIdx

		lastExpression.Inputs = lastExpression.Outputs
		lastExpression.Outputs = nil

		if inputs := lastExpression.InputsGet(prgrm); pointerArrayOut != nil && len(inputs) > 0 && ast.CXTypeSignaturePassBy(prgrm, prgrm.CXTypeSignatureFromArrayGet(inputs[0])) == opcodetable.PassByReference {
			pointerArrayOut.PassBy = opcodetable.PassByReference
			prgrm.CXTypeSignatureFromArrayGet(typeSigIdx).PassBy = opcodetable.PassByReference
		}
		lastExpression.OutputAdd(prgrm, typeSigIdx)
		if len(retExprs) < 2 || retExprs[len(retExprs)-2].Type != ast.CXOperationTypeLine {
			// return s[0][1] has no line of its own for a runtime error to report.
			lineExprs := append([]ast.CXExpression{}, retExprs[:len(retExprs)-1]...)
			return append(lineExprs, *CXLineExpressionCurrentMake(prgrm), lastExpr)
		}
		return retExprs
	} else if lastExpr.TypeArrayLiteral() && len(lastExpression.OutputsGet(prgrm)) > 0 {
		// return [2]i32{1, 2} writes the literal to the result.
		lastExpression.Outputs = nil
		lastExpression.OutputAdd(prgrm, typeSigIdx)
		retExprs[len(retExprs)-1].ExpressionType = ast.CXExpressionTypeUnused
		return retExprs
	} else if len(lastExpression.OutputsGet(prgrm)) > 0 {
		// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
		// Now a constructor call, site S0716.
		exprCXLine := CXLineExpressionCurrentMake(prgrm)
		// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
		// Now a constructor call, site S0717.
		expr := ast.AtomicOperatorExpressionMake(prgrm, ast.NativeFunctionOf(prgrm, opcodetable.OpIdentity))
		// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
		// Now an arena getter, site S0718.
		expression, err := prgrm.CXAtomicOpGet(expr.Index)
		if err != nil {
			panic(err)
		}

		exprOutTypeSig := lastExpression.OutputsGet(prgrm)[0]
		expression.InputAdd(prgrm, exprOutTypeSig)
		expression.OutputAdd(prgrm, typeSigIdx)

		return append(retExprs, *exprCXLine, *expr)
	} else {
		lastExpression.OutputAdd(prgrm, typeSigIdx)
		return retExprs
	}
}

// ReturnExpressionsJmpAdd adds an jump expression that makes a function stop its execution.
//
// Input arguments description:
//
//		prgrm - a CXProgram that contains all the data and arrays of the program.
//	 exprs - contains the array of return expressions where the jump expr will be added.
func ReturnExpressionsJmpAdd(prgrm *ast.CXProgram, exprs ReturnExpressions) []ast.CXExpression {
	var pkg *ast.CXPackage
	var fn *ast.CXFunction
	var err error

	pkg, err = prgrm.CurrentPackageGet()
	if err != nil {
		panic(err)
	}

	fn, err = pkg.CurrentFunctionGet(prgrm)
	if err != nil {
		panic(err)
	}

	retExprs := exprs.Expressions
	fnOutputs := fn.OutputsGet(prgrm)
	if len(fnOutputs) != exprs.Size && exprs.Expressions != nil {
		operands := ast.CompileErrorOperands{int64(fn.Name), int64(len(fnOutputs)), int64(exprs.Size)}
		if exprs.Column > 0 {
			prgrm.CXProgramCompileErrorAddPosition(ast.CompileErrorCodeFunctionReturnCount, CurrentFile, exprs.Line, exprs.Column, operands)
		} else {
			file, line := CurrentFile, LineNo
			if lastExprCXLine, err := prgrm.PreviousCXLineGet(retExprs, len(retExprs)-1); err == nil {
				file, line = lastExprCXLine.FileName, lastExprCXLine.LineNumber
			}
			prgrm.CXProgramCompileErrorAdd(ast.CompileErrorCodeFunctionReturnCount, file, line, operands)
		}
	}

	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0719.
	exprCXLine := CXLineExpressionCurrentMake(prgrm)
	// expression to jump to the end of the embedding function
	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0720.
	expr := ast.AtomicOperatorExpressionMake(prgrm, ast.NativeFunctionOf(prgrm, opcodetable.OpGoto))
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
	// Now an arena getter, site S0721.
	expression, err := prgrm.CXAtomicOpGet(expr.Index)
	if err != nil {
		panic(err)
	}

	// simulating a label so it gets executed without evaluating a predicate
	expression.Label = temporaryVariableNameGenerate(prgrm, opcodetable.LabelPrefix)
	expression.ThenLines = ast.ThenLinesReturn
	expression.Package = ast.CXPackageIndex(pkg.Index)

	retExprs = append(retExprs, *exprCXLine, *expr)

	return retExprs
}

// ShortCircuitStrip turns the && and || of a struct literal field back into eager bool ops.
// A struct literal takes every expression of a field value as the field itself.
func ShortCircuitStrip(prgrm *ast.CXProgram, exprs []ast.CXExpression) []ast.CXExpression {
	out := make([]ast.CXExpression, 0, len(exprs))
	for i := 0; i < len(exprs); i++ {
		if exprs[i].Type != ast.CXOperationTypeAtomicOperator {
			out = append(out, exprs[i])
			continue
		}
		op := &prgrm.CXAtomicOps[exprs[i].Index]
		thenPending := op.ThenLines == shortCircuitLinesPending
		elsePending := op.ElseLines == shortCircuitLinesPending
		if !thenPending && !elsePending {
			out = append(out, exprs[i])
			continue
		}
		op.ThenLines = 0
		op.ElseLines = 0
		if thenPending && elsePending {
			copyOutput := op.OutputsGet(prgrm)[0]
			for j := i + 1; j < len(exprs); j++ {
				if exprs[j].Type != ast.CXOperationTypeAtomicOperator {
					continue
				}
				boolInputs := prgrm.CXAtomicOps[exprs[j].Index].InputsGet(prgrm)
				if len(boolInputs) == 2 && boolInputs[1] == copyOutput {
					boolInputs[1] = op.InputsGet(prgrm)[0]
					break
				}
			}
			continue
		}
		if len(out) > 0 && out[len(out)-1].Type == ast.CXOperationTypeLine {
			out = out[:len(out)-1]
		}
	}
	return out
}

// unaryOperatorOnValue applies - or ! to a call, an operator result or a
// struct field read. It reports false for the operands UnaryExpression handles.
func unaryOperatorOnValue(prgrm *ast.CXProgram, op string, prevExprs []ast.CXExpression) ([]ast.CXExpression, bool) {
	lastIdx := prevExprs[len(prevExprs)-1].Index
	operator := prgrm.CxIrState.CxIrStateFunctionGetOrNil(prgrm.CXAtomicOps[lastIdx].Operator)
	operand := prevExprs
	if operator != nil {
		if len(prgrm.CXAtomicOps[lastIdx].OutputsGet(prgrm)) == 0 {
			if len(operator.OutputsGet(prgrm)) == 0 {
				return nil, false
			}
			pkg, err := prgrm.CurrentPackageGet()
			if err != nil {
				panic(err)
			}
			operandResultOutputAdd(prgrm, pkg, prevExprs, operator)
		}
	} else {
		outputs := prgrm.CXAtomicOps[lastIdx].OutputsGet(prgrm)
		if len(outputs) == 0 {
			return nil, false
		}
		sig := prgrm.CXTypeSignatureFromArrayGet(outputs[0])
		if sig.Type != ast.TypeCXArgumentDeprecate || len(prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(sig.Meta)).Fields) == 0 {
			return nil, false
		}
		// The read is replaced, and the expressions before it still run.
		operand = prevExprs[:len(prevExprs)-1]
	}

	pkg, err := prgrm.CurrentPackageGet()
	if err != nil {
		panic(err)
	}
	opcode := opcodetable.OpNeg
	if op == "!" {
		opcode = opcodetable.OpBoolNot
	}
	expr := ast.AtomicOperatorExpressionMake(prgrm, ast.NativeFunctionOf(prgrm, opcode))
	prgrm.CXAtomicOps[expr.Index].Package = ast.CXPackageIndex(pkg.Index)
	prgrm.CXAtomicOps[expr.Index].InputAdd(prgrm, prgrm.CXAtomicOps[lastIdx].OutputsGet(prgrm)[0])
	return append(operand, *expr), true
}

// invalidPlaceholderTypeSignature returns a temporary of types.INVALID, the value
// of an expression whose compile error is recorded.
func invalidPlaceholderTypeSignature(prgrm *ast.CXProgram, pkgIdx ast.CXPackageIndex) ast.CXTypeSignatureIndex {
	_, placeholder := ast.ArgumentMake(prgrm, temporaryVariableNameGenerate(prgrm, opcodetable.LocalPrefix), CurrentFile, LineNo)
	placeholder.TypeSet(valuetypes.INVALID)
	placeholder.Package = pkgIdx
	placeholder.PreviouslyDeclared = true
	return ast.CXTypeSignatureOfCXArgBuilderAlloc(prgrm, placeholder)
}
