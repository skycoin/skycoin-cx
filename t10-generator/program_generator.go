package generator

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"strconv"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	cxconstants "github.com/skycoin/skycoin-cx/t1-constants"
	stringpool "github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
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

		// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
		// Now a constructor call, site S0021.
		exprCXLine := cxast.MakeCXLineExpression(prgrm, cxast.SourceFileGidNil, -1, stringpool.StringGidNil)
		// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
		// Now a constructor call, site S0022.
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
			// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
			// Now an arena getter, site S0023.
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
		// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
		// Now an arena getter, site S0024.
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

	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S0025.
	fn := prgrm.GetFunctionFromArray(fnIdx)
	fn.Size = calcFnSize(prgrm, fn)
	fn.LineCount = len(fn.Expressions)
	// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
	// Now an object copy, site S0026.
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
	// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
	// Now an arena getter, site S0027.
	cxAtomicOp, err := prgrm.GetCXAtomicOp(index)
	if err != nil {
		panic(err)
	}
	return cxAtomicOp
}

// sigType reads the atomic type of a type signature, deprecated CXArgument
// signatures included, which CXTypeSignature.GetType does not handle.
func sigType(prgrm *cxast.CXProgram, idx cxast.CXTypeSignatureIndex) types.Code {
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now an arena getter, site S0028.
	sig := prgrm.GetCXTypeSignatureFromArray(idx)
	if sig.Type == cxast.TYPE_CXARGUMENT_DEPRECATE {
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S0029.
		return prgrm.GetCXArgFromArray(cxast.CXArgumentIndex(sig.Meta)).Type
	}
	return sig.GetType(prgrm)
}

func sigSize(prgrm *cxast.CXProgram, idx cxast.CXTypeSignatureIndex) types.Pointer {
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
	// Now an arena getter, site S0030.
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
	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S0031.
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
	_, arg := cxast.MakeField(prgrm, prgrm.CXProgramNameIntern(strconv.Itoa(exprPos)), typ, cxast.SourceFileGidNil, -1)
	arg.SetType(typ)
	arg.Package = fn.Package
	arg.Offset = slotOffset(prgrm, fn, exprPos)
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now an arena insert, site S0033.
	return prgrm.AddCXTypeSignatureInArray(cxast.GetCXTypeSignatureRepresentationOfCXArg(prgrm, arg))
}

// copySig adds a copy of the type signature at idx, so each use of a
// variable has its own signature.
func copySig(prgrm *cxast.CXProgram, idx cxast.CXTypeSignatureIndex) cxast.CXTypeSignatureIndex {
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Also state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332. Site S0034 S0035.
	sig := *prgrm.GetCXTypeSignatureFromArray(idx)
	// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureAlloc(), phase 4 #331.
	// Now an arena insert, site S0036.
	return prgrm.AddCXTypeSignatureInArray(&sig)
}

// findArgOptions lists the inputs of fn and the outputs already written by
// its expressions that have type argTypeToFind.
func findArgOptions(prgrm *cxast.CXProgram, fn *cxast.CXFunction, argTypeToFind types.Code) []cxast.CXTypeSignatureIndex {
	var options []cxast.CXTypeSignatureIndex
	for _, idx := range fn.GetInputs(prgrm) {
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0037.
		if sigType(prgrm, idx) == argTypeToFind && prgrm.GetCXTypeSignatureFromArray(idx).Name != stringpool.StringGidNil {
			options = append(options, idx)
		}
	}

	for _, expr := range fn.Expressions {
		if expr.Type != cxast.CX_ATOMIC_OPERATOR {
			continue
		}
		outputs := getAtomicOp(prgrm, expr.Index).GetOutputs(prgrm)
		// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
		// Now an arena getter, site S0038.
		if len(outputs) > 0 && sigType(prgrm, outputs[0]) == argTypeToFind && prgrm.GetCXTypeSignatureFromArray(outputs[0]).Name != stringpool.StringGidNil {
			options = append(options, outputs[0])
		}
	}
	return options
}

func getRandInp(prgrm *cxast.CXProgram, fnIdx cxast.CXFunctionIndex, opCode int, argType types.Code) cxast.CXTypeSignatureIndex {
	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S0039.
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
	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0040.
	newExprCXLine := cxast.MakeCXLineExpression(prgrm, cxast.SourceFileGidNil, -1, stringpool.StringGidNil)
	// OBJMODEL: state.CXExpressionMng.CXExpressionAlloc(), phase 4 #346.
	// Now a constructor call, site S0041.
	newExpr := cxast.MakeAtomicOperatorExpression(prgrm, native)

	for _, inp := range native.Inputs {
		// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
		// Now an arena getter, site S0042.
		options := findArgOptions(prgrm, prgrm.GetFunctionFromArray(fnIdx), inp.Type)
		inpIdx := copySig(prgrm, options[rand.Intn(len(options))])
		getAtomicOp(prgrm, newExpr.Index).AddInput(prgrm, inpIdx)
	}

	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S0043.
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
	// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
	// Now an arena getter, site S0044.
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

// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
// Now a constructor, site S0045.
func GenerateSampleProgram(t *testing.T, withLiteral bool) *cxast.CXProgram {
	var cxProgram *cxast.CXProgram

	// Needed for AddNativeExpressionToFunction
	// because of dependency on cxast.OpNames
	cxparsingcompletor.InitCXCore()
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, site S0046.
	cxProgram = cxinit.MakeProgram()
	cxinit.RegisterPackages(cxProgram)

	// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
	// Now an arena insert, site S0047.
	err := astapi.AddEmptyPackage(cxProgram, "main")
	if err != nil {
		t.Errorf("want no error, got %v", err)
	}

	// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
	// Now an arena insert, site S0048.
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

	fn, _ := cxProgram.GetFunction(cxProgram.CXProgramNameIntern("TestFunction"), cxProgram.CxIrState.CxIrStateNameMain)
	pkg, _ := cxProgram.GetPackage(cxProgram.CxIrState.CxIrStateNameMain)
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
