package parsingcompletor

import (
	stringpool "github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
)

// ParserActionAssignmentExpression is the action of an assignment, shared by
// assignment_expression and the for clauses.
func ParserActionAssignmentExpression(left []ast.CXExpression, operator string, right []ast.CXExpression) []ast.CXExpression {
	result := left

	if right == nil {
		result = nil
	}
	if right != nil {
		for i := 0; i < len(right); i++ {
			if right[i].Type != ast.CX_LINE {
				if right[i].IsArrayLiteral() {
					if operator != "=" && operator != ":=" {
						panic("")
					}
					if operator == ":=" {
						for _, from := range right {
							if from.Type == ast.CX_LINE {
								continue
							}

							// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
							// Now an arena getter, site S1493.
							fromExpression, err := actions.AST.GetCXAtomicOp(from.Index)
							if err != nil {
								panic(err)
							}

							// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
							// Now an arena getter, site S1494.
							fromExpressionOutputTypeSig := actions.AST.GetCXTypeSignatureFromArray(fromExpression.GetOutputs(actions.AST)[0])
							fromExpressionOutputIdx := fromExpressionOutputTypeSig.Meta
							// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
							// Now an arena getter, site S1773.
							actions.AST.GetCXArg(ast.CXArgumentIndex(fromExpressionOutputIdx)).PreviouslyDeclared = true
						}
					}
					result = actions.ArrayLiteralAssignment(actions.AST, left, right)
				} else if right[len(right)-1].IsStructLiteral() {
					if operator != "=" && operator != ":=" {
						panic("")
					}
					if operator == ":=" {
						for _, from := range right {
							if from.Type == ast.CX_LINE {
								continue
							}

							// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
							// Now an arena getter, site S1495.
							fromExpression, err := actions.AST.GetCXAtomicOp(from.Index)
							if err != nil {
								panic(err)
							}

							// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
							// Now an arena getter, site S1496.
							fromExpressionOutputTypeSig := actions.AST.GetCXTypeSignatureFromArray(fromExpression.GetOutputs(actions.AST)[0])
							fromExpressionOutputIdx := fromExpressionOutputTypeSig.Meta
							// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
							// Now an arena getter, site S1774.
							actions.AST.GetCXArg(ast.CXArgumentIndex(fromExpressionOutputIdx)).PreviouslyDeclared = true
						}
					}
					result = actions.StructLiteralAssignment(actions.AST, left, right)
				} else {
					result = actions.AssignmentStatement(actions.AST, left, operator, right)
				}

				break
			}
		}
	}
	return result
}

// ParserActionExpressionList joins the outputs of a comma separated
// expression list into its last expression.
func ParserActionExpressionList(left []ast.CXExpression, right []ast.CXExpression) []ast.CXExpression {
	lastOfThirdAtomicOp, err := actions.AST.GetCXAtomicOpFromExpressions(right, len(right)-1)
	if err != nil {
		panic(err)
	}

	lastOfFirstAtomicOp, err := actions.AST.GetCXAtomicOpFromExpressions(left, len(left)-1)
	if err != nil {
		panic(err)
	}

	lastOfThirdAtomicOp.Outputs.Fields = append(lastOfFirstAtomicOp.Outputs.Fields, lastOfThirdAtomicOp.Outputs.Fields...)
	return right
}

// ParserActionExpressionStatement is the action of an expression statement,
// shared by expression_statement and the for clauses.
func ParserActionExpressionStatement(exprs []ast.CXExpression) []ast.CXExpression {
	exprs = actions.MethodCallStatement(actions.AST, exprs)
	var result []ast.CXExpression
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorAlloc(), phase 4 #341.
	// Now a heap literal, site S1497.
	var lastFirstAtomicOp *ast.CXAtomicOperator = &ast.CXAtomicOperator{Operator: -1}
	var err error

	if len(exprs) > 0 {
		lastFirstAtomicOp, err = actions.AST.GetCXAtomicOpFromExpressions(exprs, len(exprs)-1)
		if err != nil {
			panic(err)
		}
	}
	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S1498.
	lastFirstAtomicOpOperator := actions.AST.GetFunctionFromArray(lastFirstAtomicOp.Operator)

	if len(exprs) > 0 && lastFirstAtomicOpOperator == nil && !exprs[len(exprs)-1].IsMethodCall() {
		outs := lastFirstAtomicOp.GetOutputs(actions.AST)

		if len(outs) > 0 {
			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
			// Now an arena getter, site S1499.
			outTypeSig := actions.AST.GetCXTypeSignatureFromArray(outs[0])
			actions.AST.CXProgramCompileErrorAddDebug(ast.CompileErrorCodeInvalidExpression, outTypeSig.ArgDetails, ast.CompileErrorOperands{})
		} else {
			actions.AST.CXProgramCompileErrorAdd(ast.CompileErrorCodeInvalidExpression, actions.CurrentFile, actions.LineNo, ast.CompileErrorOperands{})
		}
		result = nil
	} else if len(exprs) > 0 && exprs[0].Type != ast.CX_LINE {
		// test() and assert report the CX_LINE before them (skycoin-cx#254)
		// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
		// Also state.CXExpressionMng (manager storage) or []CXExpressionGid, phase 4 #346. Site S1500 S1501.
		result = append([]ast.CXExpression{*actions.MakeCXLineExpressionCurrent(actions.AST)}, exprs...)
	} else {
		result = exprs
	}
	return result
}

// ParserActionLoopCondition keeps a literal or variable loop condition as
// it is, which IterationExpressions reads in place (skycoin-cx#21).
func ParserActionLoopCondition(exprs []ast.CXExpression) []ast.CXExpression {
	if len(exprs) > 0 && exprs[len(exprs)-1].Type != ast.CX_LINE && !exprs[len(exprs)-1].IsMethodCall() {
		cxAtomicOp, err := actions.AST.GetCXAtomicOpFromExpressions(exprs, len(exprs)-1)
		if err != nil {
			panic(err)
		}
		if actions.AST.GetFunctionFromArray(cxAtomicOp.Operator) == nil {
			return exprs
		}
	}
	return ParserActionExpressionStatement(exprs)
}

// ParserActionLabeledStatement labels every expression of a statement that
// has no label yet, so a nested goto keeps its target and a nested label stays.
func ParserActionLabeledStatement(label string, exprs []ast.CXExpression) []ast.CXExpression {
	labelGid := actions.AST.CXProgramNameIntern(label)
	first := -1
	labeled := false
	for i, expr := range exprs {
		if expr.Type == ast.CX_LINE {
			continue
		}
		if first < 0 {
			first = i
		}
		if actions.AST.CXAtomicOps[expr.Index].Label == stringpool.StringGidNil {
			actions.AST.CXAtomicOps[expr.Index].Label = labelGid
			labeled = true
		}
	}
	if !labeled && first >= 0 {
		actions.AST.CXAtomicOps[exprs[first].Index].Label = labelGid
	}
	return exprs
}
