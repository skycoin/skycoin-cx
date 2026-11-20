package ast

// CXExpression is used represent a CX expression.
//
// All statements in CX are expressions, including for loops and other control
// flow.
type CXExpression struct {
	ExpressionType CXEXPR_TYPE

	// For new CX AST Format
	Index int
	Type  CXOPERATION_TYPE
}
