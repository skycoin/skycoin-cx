package generator

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"strconv"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	cxconstants "github.com/skycoin/skycoin-cx/t1-constants"
	cxast "github.com/skycoin/skycoin-cx/t3-ast"
	cxinit "github.com/skycoin/skycoin-cx/t7-init"                           // thin: only MakeProgram, RegisterPackages; no depth gain
	cxparsingcompletor "github.com/skycoin/skycoin-cx/t8-cxparsingcompletor" // thin: only InitCXCore; no depth gain
	"github.com/skycoin/skycoin-cx/t9-astapi"
)

// GenerateRandomExpressions appends random expressions built from fns to
// inputFn. Each expression owns a stack slot named after its position in
// inputFn.Expressions, and the last one writes to inputFn's first output.
func GenerateRandomExpressions(prgrm *cxast.CXProgram, inputFn *cxast.CXFunction, inputPkg *cxast.CXPackage, fns []*cxast.CXNativeFunction, numExprs int) {
	// Adding an expression can grow CXFunctions and CXAtomicOps, so both
	// are looked up by index after every addition.
	fnIdx := cxast.CXFunctionIndex(inputFn.Index)
	fnOutputs := inputFn.GetOutputs(prgrm)
	newExprs := numExprs - len(inputFn.Expressions)

	for i := 0; i < newExprs; i++ {
		last := i == newExprs-1
		op := getRandFn(fns)
		// Last expression output must be the same as function output.
		if last && len(fnOutputs) > 0 {
			want := sigType(prgrm, fnOutputs[0])
			for len(op.Outputs) == 0 || op.Outputs[0].Type != want {
				op = getRandFn(fns)
			}
		}

		exprCXLine := cxast.MakeCXLineExpression(prgrm, "", -1, "")
		expr := cxast.MakeAtomicOperatorExpression(prgrm, op)
		cxAtomicOp := getAtomicOp(prgrm, expr.Index)
		cxAtomicOp.Package = cxast.CXPackageIndex(inputPkg.Index)
		cxAtomicOp.Function = fnIdx

		for _, inp := range op.Inputs {
			inpIdx := getRandInp(prgrm, fnIdx, op.AtomicOPCode, inp.Type)
			getAtomicOp(prgrm, expr.Index).AddInput(prgrm, inpIdx)
		}

		// if operator is jmp, add then and else lines
		if IsJumpOperator(op.AtomicOPCode) {
			lineNumOptions := numExprs - len(prgrm.GetFunctionFromArray(fnIdx).Expressions)
			if lineNumOptions < 0 {
				lineNumOptions = (lineNumOptions * -1) - 2
			}
			randThenLineIndex := 0
			if lineNumOptions > 0 {
				randThenLineIndex = rand.Intn(lineNumOptions)
			}

			cxAtomicOp = getAtomicOp(prgrm, expr.Index)
			cxAtomicOp.ThenLines = 1
			cxAtomicOp.ElseLines = randThenLineIndex
		}

		// The expression is added before its outputs are chosen, so its
		// own slot is one of the options.
		fn := prgrm.GetFunctionFromArray(fnIdx)
		fn.AddExpression(prgrm, exprCXLine)
		fn.AddExpression(prgrm, expr)

		cxAtomicOp = getAtomicOp(prgrm, expr.Index)
		if last && len(fnOutputs) > 0 {
			cxAtomicOp.AddOutput(prgrm, fnOutputs[0])
		} else {
			for _, out := range op.Outputs {
				cxAtomicOp.AddOutput(prgrm, getRandOut(prgrm, fnIdx, out.Type))
			}
		}
	}

	fn := prgrm.GetFunctionFromArray(fnIdx)
	fn.Size = calcFnSize(prgrm, fn)
	fn.LineCount = len(fn.Expressions)
	*inputFn = *fn
}

func IsJumpOperator(opCode int) bool {
	switch opCode {
	case cxconstants.OP_JMP,
		cxconstants.OP_ABS_JMP,
		cxconstants.OP_JMP_EQ,
		cxconstants.OP_JMP_UNEQ,
		cxconstants.OP_JMP_GT,
		cxconstants.OP_JMP_GTEQ,
		cxconstants.OP_JMP_LT,
		cxconstants.OP_JMP_LTEQ,
		cxconstants.OP_JMP_ZERO,
		cxconstants.OP_JMP_NOT_ZERO:
		return true
	default:
		return false
	}
}

func getRandFn(fnSet []*cxast.CXNativeFunction) *cxast.CXNativeFunction {
	return fnSet[rand.Intn(len(fnSet))]
}

func getAtomicOp(prgrm *cxast.CXProgram, index int) *cxast.CXAtomicOperator {
	cxAtomicOp, err := prgrm.GetCXAtomicOp(index)
	if err != nil {
		panic(err)
	}
	return cxAtomicOp
}

// sigType reads the atomic type of a type signature, deprecated CXArgument
// signatures included, which CXTypeSignature.GetType does not handle.
func sigType(prgrm *cxast.CXProgram, idx cxast.CXTypeSignatureIndex) types.Code {
	sig := prgrm.GetCXTypeSignatureFromArray(idx)
	if sig.Type == cxast.TYPE_CXARGUMENT_DEPRECATE {
		return prgrm.GetCXArgFromArray(cxast.CXArgumentIndex(sig.Meta)).Type
	}
	return sig.GetType(prgrm)
}

func sigSize(prgrm *cxast.CXProgram, idx cxast.CXTypeSignatureIndex) types.Pointer {
	return prgrm.GetCXTypeSignatureFromArray(idx).GetSize(prgrm, false)
}

// opOutput returns the first output of the operator behind expression exprPos
// of fn. Only one output per operator is considered, multiple returns in CX
// are buggy anyway.
func opOutput(prgrm *cxast.CXProgram, fn *cxast.CXFunction, exprPos int) (cxast.CXTypeSignatureIndex, bool) {
	expr := fn.Expressions[exprPos]
	if expr.Type != cxast.CX_ATOMIC_OPERATOR {
		return 0, false
	}
	operator := prgrm.GetFunctionFromArray(getAtomicOp(prgrm, expr.Index).Operator)
	outputs := operator.GetOutputs(prgrm)
	if len(outputs) == 0 {
		return 0, false
	}
	return outputs[0], true
}

// slotOffset is the stack offset of the slot owned by expression exprPos,
// after fn's inputs, outputs and the slots of the expressions before it.
func slotOffset(prgrm *cxast.CXProgram, fn *cxast.CXFunction, exprPos int) (offset types.Pointer) {
	for _, idx := range fn.GetInputs(prgrm) {
		offset += sigSize(prgrm, idx)
	}
	for _, idx := range fn.GetOutputs(prgrm) {
		offset += sigSize(prgrm, idx)
	}
	for p := 0; p < exprPos; p++ {
		if out, ok := opOutput(prgrm, fn, p); ok {
			offset += sigSize(prgrm, out)
		}
	}
	return offset
}

func calcFnSize(prgrm *cxast.CXProgram, fn *cxast.CXFunction) types.Pointer {
	return slotOffset(prgrm, fn, len(fn.Expressions))
}

// newSlot adds an atomic type signature for the slot of expression exprPos.
func newSlot(prgrm *cxast.CXProgram, fn *cxast.CXFunction, exprPos int, typ types.Code) cxast.CXTypeSignatureIndex {
	arg := cxast.MakeField(strconv.Itoa(exprPos), typ, "", -1).SetType(typ)
	arg.Package = fn.Package
	arg.Offset = slotOffset(prgrm, fn, exprPos)
	return prgrm.AddCXTypeSignatureInArray(cxast.GetCXTypeSignatureRepresentationOfCXArg(prgrm, arg))
}

// copySig adds a copy of the type signature at idx, so each use of a
// variable has its own signature.
func copySig(prgrm *cxast.CXProgram, idx cxast.CXTypeSignatureIndex) cxast.CXTypeSignatureIndex {
	sig := *prgrm.GetCXTypeSignatureFromArray(idx)
	return prgrm.AddCXTypeSignatureInArray(&sig)
}

// findArgOptions lists the inputs of fn and the outputs already written by
// its expressions that have type argTypeToFind.
func findArgOptions(prgrm *cxast.CXProgram, fn *cxast.CXFunction, argTypeToFind types.Code) []cxast.CXTypeSignatureIndex {
	var options []cxast.CXTypeSignatureIndex
	for _, idx := range fn.GetInputs(prgrm) {
		if sigType(prgrm, idx) == argTypeToFind && prgrm.GetCXTypeSignatureFromArray(idx).Name != "" {
			options = append(options, idx)
		}
	}

	for _, expr := range fn.Expressions {
		if expr.Type != cxast.CX_ATOMIC_OPERATOR {
			continue
		}
		outputs := getAtomicOp(prgrm, expr.Index).GetOutputs(prgrm)
		if len(outputs) > 0 && sigType(prgrm, outputs[0]) == argTypeToFind && prgrm.GetCXTypeSignatureFromArray(outputs[0]).Name != "" {
			options = append(options, outputs[0])
		}
	}
	return options
}

func getRandInp(prgrm *cxast.CXProgram, fnIdx cxast.CXFunctionIndex, opCode int, argType types.Code) cxast.CXTypeSignatureIndex {
	options := findArgOptions(prgrm, prgrm.GetFunctionFromArray(fnIdx), argType)

	// if no options available or if operator is jump, add new i32_LT expression.
	if len(options) == 0 || opCode == cxconstants.OP_JMP {
		// TODO: improve process when there's OP_JMP
		return addNewExpression(prgrm, fnIdx, cxast.OpCodes["i32.lt"])
	}

	return copySig(prgrm, options[rand.Intn(len(options))])
}

// addNewExpression appends an expression of type expressionType to fn, with
// random inputs, and returns a signature reading its output.
func addNewExpression(prgrm *cxast.CXProgram, fnIdx cxast.CXFunctionIndex, expressionType int) cxast.CXTypeSignatureIndex {
	native := cxast.Natives[expressionType]
	newExprCXLine := cxast.MakeCXLineExpression(prgrm, "", -1, "")
	newExpr := cxast.MakeAtomicOperatorExpression(prgrm, native)

	for _, inp := range native.Inputs {
		options := findArgOptions(prgrm, prgrm.GetFunctionFromArray(fnIdx), inp.Type)
		inpIdx := copySig(prgrm, options[rand.Intn(len(options))])
		getAtomicOp(prgrm, newExpr.Index).AddInput(prgrm, inpIdx)
	}

	fn := prgrm.GetFunctionFromArray(fnIdx)
	fn.AddExpression(prgrm, newExprCXLine)
	fn.AddExpression(prgrm, newExpr)

	exprPos := len(fn.Expressions) - 1
	outIdx := newSlot(prgrm, fn, exprPos, native.Outputs[0].Type)
	getAtomicOp(prgrm, newExpr.Index).AddOutput(prgrm, outIdx)

	return copySig(prgrm, outIdx)
}

// getRandOut picks the slot of a random expression of fn whose operator
// returns outType, so expressions may overwrite each other's results.
func getRandOut(prgrm *cxast.CXProgram, fnIdx cxast.CXFunctionIndex, outType types.Code) cxast.CXTypeSignatureIndex {
	fn := prgrm.GetFunctionFromArray(fnIdx)

	var options []int
	for p := range fn.Expressions {
		if out, ok := opOutput(prgrm, fn, p); ok && sigType(prgrm, out) == outType {
			options = append(options, p)
		}
	}

	return newSlot(prgrm, fn, options[rand.Intn(len(options))], outType)
}

func GetFunctionSet(fnNames []string) (fns []*cxast.CXNativeFunction) {
	for _, fnName := range fnNames {
		fn := cxast.Natives[cxast.OpCodes[fnName]]
		if fn == nil {
			panic("standard library function not found.")
		}

		fns = append(fns, fn)
	}
	return fns
}

// VIOLATION: created outside manager; phase 1.
func GenerateSampleProgram(t *testing.T, withLiteral bool) *cxast.CXProgram {
	var cxProgram *cxast.CXProgram

	// Needed for AddNativeExpressionToFunction
	// because of dependency on cxast.OpNames
	cxparsingcompletor.InitCXCore()
	cxProgram = cxinit.MakeProgram()
	cxinit.RegisterPackages(cxProgram)

	err := astapi.AddEmptyPackage(cxProgram, "main")
	if err != nil {
		t.Errorf("want no error, got %v", err)
	}

	err = astapi.AddEmptyFunctionToPackage(cxProgram, "main", "TestFunction")
	if err != nil {
		t.Errorf("want no error, got %v", err)
	}

	err = astapi.AddNativeInputToFunction(cxProgram, "main", "TestFunction", "inputOne", types.Code(types.I32))
	if err != nil {
		t.Errorf("want no error, got %v", err)
	}

	err = astapi.AddNativeOutputToFunction(cxProgram, "main", "TestFunction", "outputOne", types.Code(types.I32))
	if err != nil {
		t.Errorf("want no error, got %v", err)
	}
	functionSetNames := []string{"i32.add", "i32.mul", "i32.sub", "i32.eq", "i32.uneq", "i32.gt", "i32.gteq", "i32.lt", "i32.lteq", "bool.not", "bool.or", "bool.and", "bool.uneq", "bool.eq", "i32.neg", "i32.abs", "i32.bitand", "i32.bitor", "i32.bitxor", "i32.bitclear", "i32.bitshl", "i32.bitshr", "i32.max", "i32.min", "i32.rand"}
	fns := GetFunctionSet(functionSetNames)

	fn, _ := cxProgram.GetFunction("TestFunction", "main")
	pkg, _ := cxProgram.GetPackage("main")
	GenerateRandomExpressions(cxProgram, fn, pkg, fns, 30)

	if withLiteral {
		buf := new(bytes.Buffer)
		var num int32 = 5
		binary.Write(buf, binary.LittleEndian, num)
		err = astapi.AddLiteralInputToExpression(cxProgram, "main", "TestFunction", buf.Bytes(), types.I32, 2)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}

	}

	cxProgram.PrintProgram()
	return cxProgram
}
