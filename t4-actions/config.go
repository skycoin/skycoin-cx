package actions

import (
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// VIOLATION: state outside State. Owner: t4-actions; moves to CXProgramMng in phase 2.
var AST *ast.CXProgram

// VIOLATION: state outside State. Owner: t4-actions; moves to ParseMng in phase 2.
var CurrentFile string

// VIOLATION: state outside State. Owner: t4-actions; moves to ParseMng in phase 2.
var LineNo int

// VIOLATION: state outside State. Owner: t4-actions; moves to ParseMng in phase 2.
var LineStr string
