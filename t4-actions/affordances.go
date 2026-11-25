package actions

import (
	"github.com/skycoin/skycoin-cx/t0-types"
	stringpool "github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// AffordanceClauseParsed is one clause of an aff literal as the parser
// reads it, name(argument).
type AffordanceClauseParsed struct {
	AffordanceClauseParsedName     string
	AffordanceClauseParsedArgument string
	AffordanceClauseParsedLine     int
	AffordanceClauseParsedColumn   int
}

// AffordanceLiteral compiles the aff literal #{clauses}. It writes a zero
// aff value to the data segment, which AffordanceLiteralsResolve fills once
// every function body is compiled.
func AffordanceLiteral(prgrm *ast.CXProgram, clauses []AffordanceClauseParsed, line int) []ast.CXExpression {
	literal := ast.AffordanceLiteral{
		AffordanceLiteralPackage:   prgrm.CurrentPackage,
		AffordanceLiteralFile:      CurrentFile,
		AffordanceLiteralFirstLine: line,
	}
	for _, clause := range clauses {
		kind := ast.AffordanceClauseCount
		for candidate, name := range ast.AffordanceClauseNames {
			if name == clause.AffordanceClauseParsedName {
				kind = ast.AffordanceClause(candidate)
			}
		}
		if kind == ast.AffordanceClauseCount {
			affordanceErrorAt(prgrm, clause.AffordanceClauseParsedLine, clause.AffordanceClauseParsedColumn, "unknown clause '"+clause.AffordanceClauseParsedName+"', expected mem, pkg, fn, var or strct")
			continue
		}
		argument := &literal.AffordanceLiteralClauses[kind]
		if argument.AffordanceClauseArgumentName != stringpool.StringGidNil {
			affordanceErrorAt(prgrm, clause.AffordanceClauseParsedLine, clause.AffordanceClauseParsedColumn, "clause '"+clause.AffordanceClauseParsedName+"' given twice")
			continue
		}
		argument.AffordanceClauseArgumentName = prgrm.CxIrState.StringPool.StringPoolIntern([]byte(clause.AffordanceClauseParsedArgument))
		argument.AffordanceClauseArgumentLine = clause.AffordanceClauseParsedLine
		argument.AffordanceClauseArgumentColumn = clause.AffordanceClauseParsedColumn
	}
	literal.AffordanceLiteralOffset = prgrm.Data.StartsAt + prgrm.Data.Size
	prgrm.CXProgramAffordanceLiteralAdd(literal)
	return WritePrimaryExprs(prgrm, types.AFF, make([]byte, types.AFF_SIZE), false)
}

// AffordanceLiteralsResolve writes the value of each aff literal compiled
// since the last call, or records why it has none.
func AffordanceLiteralsResolve(prgrm *ast.CXProgram) {
	for _, literal := range prgrm.CXProgramAffordanceLiteralsTake() {
		if value, ok := affordanceLiteralValue(prgrm, &literal); ok {
			ast.AffordanceValueWrite(prgrm.Memory, literal.AffordanceLiteralOffset, value)
		}
	}
}

// affordanceLiteralValue resolves the names of literal. mem stands alone, a
// package defaults to the one of the literal, and var names a global, or a
// variable of the function of fn.
func affordanceLiteralValue(prgrm *ast.CXProgram, literal *ast.AffordanceLiteral) (ast.AffordanceValue, bool) {
	value := ast.AffordanceValue{AffordanceValuePackage: -1, AffordanceValueItem: -1, AffordanceValueVariable: -1}
	clauses := &literal.AffordanceLiteralClauses
	present := func(clause ast.AffordanceClause) bool {
		return clauses[clause].AffordanceClauseArgumentName != stringpool.StringGidNil
	}
	name := func(clause ast.AffordanceClause) string {
		return prgrm.CXProgramNameText(clauses[clause].AffordanceClauseArgumentName)
	}
	fail := func(clause ast.AffordanceClause, text string) (ast.AffordanceValue, bool) {
		affordanceErrorFile(prgrm, literal.AffordanceLiteralFile, clauses[clause].AffordanceClauseArgumentLine, clauses[clause].AffordanceClauseArgumentColumn, text)
		return value, false
	}

	if present(ast.AffordanceClauseMemory) {
		for clause := ast.AffordanceClausePackage; clause < ast.AffordanceClauseCount; clause++ {
			if present(clause) {
				return fail(clause, "mem cannot be combined with "+ast.AffordanceClauseNames[clause])
			}
		}
		for segment, segmentName := range ast.AffordanceMemoryNames {
			if segmentName == name(ast.AffordanceClauseMemory) {
				value.AffordanceValueKind = ast.AffordanceKindMemory
				value.AffordanceValueItem = int32(segment)
				return value, true
			}
		}
		return fail(ast.AffordanceClauseMemory, "unknown memory '"+name(ast.AffordanceClauseMemory)+"', expected data, stack or heap")
	}
	if !present(ast.AffordanceClausePackage) && !present(ast.AffordanceClauseFunction) && !present(ast.AffordanceClauseVariable) && !present(ast.AffordanceClauseStruct) {
		value.AffordanceValueKind = ast.AffordanceKindProgram
		return value, true
	}
	if present(ast.AffordanceClauseStruct) && present(ast.AffordanceClauseFunction) {
		return fail(ast.AffordanceClauseStruct, "strct cannot be combined with fn")
	}
	if present(ast.AffordanceClauseStruct) && present(ast.AffordanceClauseVariable) {
		return fail(ast.AffordanceClauseStruct, "strct cannot be combined with var")
	}

	pkg, err := prgrm.GetPackageFromArray(literal.AffordanceLiteralPackage)
	if err != nil {
		return value, false
	}
	if present(ast.AffordanceClausePackage) && clauses[ast.AffordanceClausePackage].AffordanceClauseArgumentName != pkg.Name {
		imported, ok := pkg.Imports[clauses[ast.AffordanceClausePackage].AffordanceClauseArgumentName]
		if !ok {
			return fail(ast.AffordanceClausePackage, "package '"+name(ast.AffordanceClausePackage)+"' is not imported")
		}
		if pkg, err = prgrm.GetPackageFromArray(imported); err != nil {
			return value, false
		}
	}
	value.AffordanceValuePackage = ast.CXPackageIndex(pkg.Index)
	value.AffordanceValueKind = ast.AffordanceKindPackage

	if present(ast.AffordanceClauseStruct) {
		index, ok := pkg.Structs[clauses[ast.AffordanceClauseStruct].AffordanceClauseArgumentName]
		if !ok {
			return fail(ast.AffordanceClauseStruct, "struct '"+name(ast.AffordanceClauseStruct)+"' is not in package '"+prgrm.CXProgramNameText(pkg.Name)+"'")
		}
		value.AffordanceValueKind = ast.AffordanceKindStruct
		value.AffordanceValueItem = int32(index)
		return value, true
	}

	var fn *ast.CXFunction
	if present(ast.AffordanceClauseFunction) {
		index, ok := pkg.Functions[clauses[ast.AffordanceClauseFunction].AffordanceClauseArgumentName]
		if !ok {
			return fail(ast.AffordanceClauseFunction, "function '"+name(ast.AffordanceClauseFunction)+"' is not in package '"+prgrm.CXProgramNameText(pkg.Name)+"'")
		}
		fn = prgrm.CxIrState.CXFunctionMng.CXFunctionGet(index)
		value.AffordanceValueKind = ast.AffordanceKindFunction
		value.AffordanceValueItem = int32(ast.CXFunctionGidSlot(index))
	}

	if present(ast.AffordanceClauseVariable) {
		variableName := clauses[ast.AffordanceClauseVariable].AffordanceClauseArgumentName
		if fn == nil {
			if pkg.Globals != nil {
				for _, field := range pkg.Globals.Fields {
					if prgrm.GetCXTypeSignatureFromArray(field).Name == variableName {
						value.AffordanceValueKind = ast.AffordanceKindGlobal
						value.AffordanceValueVariable = field
						return value, true
					}
				}
			}
			return fail(ast.AffordanceClauseVariable, "global '"+name(ast.AffordanceClauseVariable)+"' is not in package '"+prgrm.CXProgramNameText(pkg.Name)+"'")
		}
		variable, found, ambiguous := affordanceFunctionVariable(prgrm, fn, variableName)
		if !found {
			return fail(ast.AffordanceClauseVariable, "variable '"+name(ast.AffordanceClauseVariable)+"' is not in function '"+name(ast.AffordanceClauseFunction)+"'")
		}
		if ambiguous {
			return fail(ast.AffordanceClauseVariable, "variable '"+name(ast.AffordanceClauseVariable)+"' is declared in more than one block of function '"+name(ast.AffordanceClauseFunction)+"'")
		}
		value.AffordanceValueKind = ast.AffordanceKindLocal
		value.AffordanceValueVariable = variable
	}
	return value, true
}

// affordanceFunctionVariable finds the parameter, result or local variable
// name of fn, as the first statement that writes it as a whole. ambiguous
// is set when sibling blocks of fn each declare a variable name at its own
// offset.
func affordanceFunctionVariable(prgrm *ast.CXProgram, fn *ast.CXFunction, name stringpool.StringGid) (variable ast.CXTypeSignatureIndex, found bool, ambiguous bool) {
	for _, parameters := range [2][]ast.CXTypeSignatureIndex{fn.GetInputs(prgrm), fn.GetOutputs(prgrm)} {
		for _, index := range parameters {
			if StripNameNumber(prgrm, prgrm.GetCXTypeSignatureFromArray(index).Name) == name {
				return index, true, false
			}
		}
	}
	variable = -1
	var offset types.Pointer
	for e := range fn.Expressions {
		if fn.Expressions[e].Type != ast.CX_ATOMIC_OPERATOR {
			continue
		}
		cxAtomicOp, err := prgrm.GetCXAtomicOp(fn.Expressions[e].Index)
		if err != nil {
			continue
		}
		for _, index := range cxAtomicOp.GetOutputs(prgrm) {
			signature := prgrm.GetCXTypeSignatureFromArray(index)
			if StripNameNumber(prgrm, signature.Name) != name || !ast.CXTypeSignatureWhole(prgrm, signature) {
				continue
			}
			if !found {
				variable, found, offset = index, true, signature.GetOffset(prgrm)
			} else if signature.GetOffset(prgrm) != offset {
				return variable, found, true
			}
		}
	}
	return variable, found, false
}

func affordanceErrorAt(prgrm *ast.CXProgram, line int, column int, text string) {
	affordanceErrorFile(prgrm, CurrentFile, line, column, text)
}

func affordanceErrorFile(prgrm *ast.CXProgram, file ast.SourceFileGid, line int, column int, text string) {
	prgrm.CXProgramCompileErrorAddPosition(ast.CompileErrorCodeAffordanceInvalid, file, line, column, ast.CompileErrorOperands{prgrm.CXProgramCompileErrorText(text)})
}
