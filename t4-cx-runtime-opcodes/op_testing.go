package opcodes

import (
	"bytes"
	"os"
	"strconv"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	// GCREACH: t3-cx-core-ast owns the heap and the GC roots. This file allocates strs, which
	// adds to the GC pressure, see docs/GarbageCollector.md.
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// VIOLATION: state outside State. Owner: t4-cx-runtime-opcodes; moves to TestMng in phase 2.
var assertSuccess = true

// AssertFailed ...
func AssertFailed() bool {
	return !assertSuccess
}

// TODO: Rework
func assert(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) (same bool) {
	var byts1, byts2 []byte
	var inp0 *ast.CXArgument
	if inputs[0].TypeSignature.Type == ast.TypeCXArgumentDeprecate {
		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S1341.
		inp0 = prgrm.CXArgFromArrayGet(ast.CXArgumentIndex(inputs[0].TypeSignature.Meta))

		if inp0.Type == valuetypes.STR || inp0.PointerTargetType == valuetypes.STR {
			byts1 = inputs[0].StrBytesGet(prgrm)
			byts2 = inputs[1].StrBytesGet(prgrm)
		} else {
			byts1 = inputs[0].BytesGet(prgrm)
			byts2 = inputs[1].BytesGet(prgrm)
		}

	} else if inputs[0].TypeSignature.Type == ast.TypeAtomic || inputs[0].TypeSignature.Type == ast.TypePointerAtomic {
		byts1 = inputs[0].BytesGet(prgrm)
		byts2 = inputs[1].BytesGet(prgrm)
	} else if inputs[0].TypeSignature.Type == ast.TypeArrayAtomic {
		byts1 = inputs[0].BytesGet(prgrm)
		byts2 = inputs[1].BytesGet(prgrm)
	} else if inputs[0].TypeSignature.Type == ast.TypePointerArrayAtomic {
		byts1 = inputs[0].BytesGet(prgrm)
		byts2 = inputs[1].BytesGet(prgrm)
	} else if inputs[0].TypeSignature.Type == ast.TypeSliceAtomic {
		byts1 = inputs[0].BytesGet(prgrm)
		byts2 = inputs[1].BytesGet(prgrm)
	} else if inputs[0].TypeSignature.Type == ast.TypeStruct || inputs[0].TypeSignature.Type == ast.TypePointerStruct {
		var strTyped bool = false
		// OBJMODEL: state.CXTypeSignature_StructMng.CXTypeSignature_StructGet(gid), phase 5 #322.
		// Now an arena getter, site S1342.
		structDetails := prgrm.CXTypeSignatureStructFromArrayGet(inputs[0].TypeSignature.Meta)

		lenFlds := len(structDetails.Fields)
		if lenFlds > 0 {
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S1343.
			fld := prgrm.CXArgFromArrayGet(structDetails.Fields[lenFlds-1])
			if fld.Type == valuetypes.STR || fld.PointerTargetType == valuetypes.STR {
				strTyped = true
			}
		}

		if strTyped {
			byts1 = inputs[0].StrBytesGet(prgrm)
			byts2 = inputs[1].StrBytesGet(prgrm)
		} else {
			byts1 = inputs[0].BytesGet(prgrm)
			byts2 = inputs[1].BytesGet(prgrm)
		}
	} else {
		panic("type is not known")
	}

	same = bytes.Equal(byts1, byts2)
	if !same {
		assertFailurePrint(prgrm, inputs[2].StrBytesGet(prgrm))
	}

	assertSuccess = assertSuccess && same
	return same
}

// assertFailurePrint writes "file: line: result was not equal to the expected
// value", then "; message" when message is not empty, in one write as
// fmt.Printf did. It appends to the format buffer, so it builds no string.
func assertFailurePrint(prgrm *ast.CXProgram, message []byte) {
	call := prgrm.CurrentCallGet()
	cxLine, _ := prgrm.PreviousCXLineGet(prgrm.CxIrState.CXFunctionMng.CXFunctionGet(call.Operator).Expressions, call.Line)
	line := append(formatBuffer(prgrm), prgrm.CXProgramSourceFileBytes(cxLine.FileName)...)
	line = append(line, ": "...)
	line = strconv.AppendInt(line, int64(cxLine.LineNumber), 10)
	line = append(line, ": result was not equal to the expected value"...)
	if len(message) != 0 {
		line = append(line, "; "...)
		line = append(line, message...)
	}
	line = append(line, '\n')
	os.Stdout.Write(line)
	formatBufferKeep(prgrm, line)
}

func OpAssertValue(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	same := assert(prgrm, inputs, outputs)
	outputs[0].BoolSet(prgrm, same)
}

func OpTest(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	assert(prgrm, inputs, outputs)
}

func OpPanic(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	if !assert(prgrm, inputs, outputs) {
		panic(valuetypes.CXAssert)
	}
}

// panicIf/panicIfNot implementation
func panicIf(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue, condition bool) {
	if inputs[0].BoolGet(prgrm) == condition {
		// The printer prints the line and the message after the loop.
		prgrm.RuntimeError.RuntimeErrorRecordAssertLine = true
		prgrm.RuntimeError.RuntimeErrorRecordAssertMessage = inputs[1].PtrGet(prgrm)
		panic(valuetypes.CXAssert)
	}
}

// panic with CXAssert exit code if condition is true
func OpPanicIf(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	panicIf(prgrm, inputs, outputs, true)
}

// panic with CXAssert exit code if condition is false
func OpPanicIfNot(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	panicIf(prgrm, inputs, outputs, false)
}

// VIOLATION STRHOT-120: B, the text of an error code, #412.
func OpStrError(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].StrSet(prgrm, ast.ErrorString(int(inputs[0].I32Get(prgrm))))
}
