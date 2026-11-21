// Package lower turns a compiled CXProgram into an ir.Image, step 4 of
// skycoin/skycoin-cx#408. It lowers the scalar subset: integer, float and
// bool operators, conversions, print, identity, declarations, jumps, test and
// calls to other functions, with every operand a plain variable or literal.
// A function using anything else is reported as a Failure with a reason
// code, and its ir.Func gets no code.
package lower

import (
	"fmt"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// ReasonCode says why a function did not lower.
type ReasonCode uint32

const (
	ReasonNone ReasonCode = iota
	// ReasonOpcodeOutsideSubset is a built-in outside the subset,
	// FailureOpcode names it.
	ReasonOpcodeOutsideSubset
	// ReasonOperandNotScalar is an operand of pointer, array, slice or
	// struct type.
	ReasonOperandNotScalar
	// ReasonOperandIndexed is an operand with an index, a field or a
	// dereference.
	ReasonOperandIndexed
	// ReasonOperandOutsideData is an absolute operand outside the data
	// segment.
	ReasonOperandOutsideData
	// ReasonOperandByReference is an operand passed by reference.
	ReasonOperandByReference
	// ReasonOperandString is a str compared by test.
	ReasonOperandString
	// ReasonOperandCount is an operator with an unexpected operand count.
	ReasonOperandCount
	// ReasonOutputPointer is identity into a pointer.
	ReasonOutputPointer
	// ReasonDeclarationType is a declaration of a type the old loop panics on.
	ReasonDeclarationType
	// ReasonExpressionType is an expression that is neither an atomic
	// operator nor a line.
	ReasonExpressionType
	// ReasonCalleeLayout is a call to a function whose inputs or outputs are
	// not plain frame variables.
	ReasonCalleeLayout
	// ReasonParameterLayout is a function whose own inputs or outputs are
	// not plain frame variables.
	ReasonParameterLayout
	// ReasonCount is one past the last reason.
	ReasonCount
)

// ReasonNames names each ReasonCode for reports. Read only.
var ReasonNames = [ReasonCount]string{
	ReasonNone:                "none",
	ReasonOpcodeOutsideSubset: "opcode outside subset",
	ReasonOperandNotScalar:    "operand not scalar",
	ReasonOperandIndexed:      "operand indexed",
	ReasonOperandOutsideData:  "operand outside data",
	ReasonOperandByReference:  "operand by reference",
	ReasonOperandString:       "operand string",
	ReasonOperandCount:        "operand count",
	ReasonOutputPointer:       "output pointer",
	ReasonDeclarationType:     "declaration type",
	ReasonExpressionType:      "expression type",
	ReasonCalleeLayout:        "callee layout",
	ReasonParameterLayout:     "parameter layout",
}

// Failure is one function that did not lower. FailureExpression indexes its
// expressions. FailureOpcode is the language opcode for
// ReasonOpcodeOutsideSubset and 0 otherwise.
type Failure struct {
	FailureFunction   uint32
	FailureExpression uint32
	FailureReason     ReasonCode
	FailureOpcode     uint32
}

// operand is one lowered operand.
type operand struct {
	operandOffset uint32
	operandFrame  bool
	operandSize   uint32
}

// state is one Program call.
type state struct {
	statePrgrm     *ast.CXProgram
	stateTable     opcodeTable
	stateStackSize uint32
	stateDataStart uint32
	stateDataEnd   uint32
	stateImage     ir.Image
	stateFiles     map[string]uint32
}

// jumpFixup is a jump target still in expression indexes.
type jumpFixup struct {
	jumpFixupPc          uint32
	jumpFixupTrue        int
	jumpFixupFalse       int
	jumpFixupConditional bool
}

// Program lowers every function of prgrm that is not a built-in. Functions
// that do not lower are listed in the failures and have no code. prgrm must
// be compiled and not yet run.
func Program(prgrm *ast.CXProgram) (ir.Image, []Failure) {
	s := &state{
		statePrgrm:     prgrm,
		stateStackSize: pointerToUint32(prgrm.Stack.Size),
		stateDataStart: pointerToUint32(prgrm.Data.StartsAt),
		stateFiles:     map[string]uint32{},
	}
	s.stateDataEnd = s.stateDataStart + pointerToUint32(prgrm.Data.Size)
	opcodeTableBuild(&s.stateTable)

	image := &s.stateImage
	image.ImageStackSize = s.stateStackSize
	image.ImageDataStart = s.stateDataStart
	image.ImageData = append([]byte(nil), prgrm.Memory[s.stateDataStart:s.stateDataEnd]...)
	image.ImageFuncs = make([]ir.Func, len(prgrm.CXFunctions))
	image.ImageFuncNames = make([]string, len(prgrm.CXFunctions))
	image.ImageEntryInit = ir.ImageFuncNone
	image.ImageEntryMain = ir.ImageFuncNone

	if packageIndex, found := prgrm.Packages[constants.MAIN_PKG]; found {
		mainPackage, err := prgrm.GetPackageFromArray(packageIndex)
		assert.AssertInvariantTrue(err == nil, "lower: main package index out of range")
		if functionIndex, found := mainPackage.Functions[constants.SYS_INIT_FUNC]; found {
			image.ImageEntryInit = uint32(functionIndex)
		}
		if functionIndex, found := mainPackage.Functions[constants.MAIN_FUNC]; found {
			image.ImageEntryMain = uint32(functionIndex)
		}
	}

	var failures []Failure
	for functionIndex := range prgrm.CXFunctions {
		fn := &prgrm.CXFunctions[functionIndex]
		image.ImageFuncNames[functionIndex] = fn.Name
		if fn.IsBuiltIn() {
			continue
		}
		if failure, failed := s.function(uint32(functionIndex), fn); failed {
			failures = append(failures, failure)
		}
	}
	return s.stateImage, failures
}

// function lowers fn into stateImage.ImageFuncs[functionIndex].
func (s *state) function(functionIndex uint32, fn *ast.CXFunction) (Failure, bool) {
	failure := Failure{FailureFunction: functionIndex}
	lowered := ir.Func{FuncFrameSize: pointerToUint32(fn.Size)}

	var reason ReasonCode
	if lowered.FuncInputs, reason = s.slots(fn.GetInputs(s.statePrgrm)); reason != ReasonNone {
		failure.FailureReason = ReasonParameterLayout
		return failure, true
	}
	if lowered.FuncOutputs, reason = s.slots(fn.GetOutputs(s.statePrgrm)); reason != ReasonNone {
		failure.FailureReason = ReasonParameterLayout
		return failure, true
	}

	lineCount := fn.LineCount
	assert.AssertInvariantTrue(lineCount >= 0 && lineCount <= len(fn.Expressions), "lower: LineCount outside the expressions of "+fn.Name)
	expressionPc := make([]uint32, lineCount+1)
	var code []ir.Instr
	var fixups []jumpFixup
	var lines []ir.Line

	for line := 0; line < lineCount; line++ {
		expressionPc[line] = uint32(len(code))
		failure.FailureExpression = uint32(line)
		expr := &fn.Expressions[line]

		switch expr.Type {
		case ast.CX_LINE:
			cxLine, err := s.statePrgrm.GetCXLine(expr.Index)
			assert.AssertInvariantTrue(err == nil, "lower: CX_LINE index out of range")
			lines = append(lines, ir.Line{
				LinePc:     uint32(len(code)),
				LineFile:   s.file(cxLine.FileName),
				LineNumber: uint32(cxLine.LineNumber),
			})
			continue
		case ast.CX_ATOMIC_OPERATOR:
		default:
			failure.FailureReason = ReasonExpressionType
			return failure, true
		}

		cxAtomicOp, err := s.statePrgrm.GetCXAtomicOp(expr.Index)
		assert.AssertInvariantTrue(err == nil, "lower: atomic operator index out of range")
		operator := s.statePrgrm.GetFunctionFromArray(cxAtomicOp.Operator)

		switch {
		case operator == nil:
			code, reason = s.declaration(code, cxAtomicOp)
		case operator.IsBuiltIn():
			var fixup jumpFixup
			var jump bool
			code, fixup, jump, reason, failure.FailureOpcode = s.builtIn(code, line, cxAtomicOp, operator)
			if jump {
				fixups = append(fixups, fixup)
			}
		default:
			code, reason = s.call(code, cxAtomicOp, operator)
		}
		if reason != ReasonNone {
			failure.FailureReason = reason
			return failure, true
		}
	}

	expressionPc[lineCount] = uint32(len(code))
	code = append(code, ir.Instr{InstrOpcode: ir.OpcodeReturn})

	for _, fixup := range fixups {
		in := &code[fixup.jumpFixupPc]
		in.InstrM = expressionPc[jumpTarget(fixup.jumpFixupTrue, lineCount)]
		if fixup.jumpFixupConditional {
			in.InstrN = expressionPc[jumpTarget(fixup.jumpFixupFalse, lineCount)]
		}
	}

	lowered.FuncCode = code
	lowered.FuncLineFirst = uint32(len(s.stateImage.ImageLines))
	lowered.FuncLineCount = uint32(len(lines))
	s.stateImage.ImageLines = append(s.stateImage.ImageLines, lines...)
	s.stateImage.ImageFuncs[functionIndex] = lowered
	return Failure{}, false
}

// jumpTarget clamps a target expression to lineCount, where the old loop
// returns. A negative target would index out of range in the old loop.
func jumpTarget(target int, lineCount int) int {
	assert.AssertInvariantTrue(target >= 0, fmt.Sprintf("lower: jump to expression %d", target))
	if target > lineCount {
		return lineCount
	}
	return target
}

// declaration zeroes a declared variable, as wipeDeclarationMemory does. It
// always adds the frame pointer, also for an offset outside the stack.
func (s *state) declaration(code []ir.Instr, cxAtomicOp *ast.CXAtomicOperator) ([]ir.Instr, ReasonCode) {
	outputs := cxAtomicOp.GetOutputs(s.statePrgrm)
	if len(outputs) == 0 {
		return code, ReasonOperandCount
	}
	signature := s.statePrgrm.GetCXTypeSignatureFromArray(outputs[0])
	var offset types.Pointer
	switch signature.Type {
	case ast.TYPE_CXARGUMENT_DEPRECATE:
		offset = s.statePrgrm.GetCXArg(ast.CXArgumentIndex(signature.Meta)).Offset
	case ast.TYPE_ATOMIC, ast.TYPE_POINTER_ATOMIC, ast.TYPE_ARRAY_ATOMIC, ast.TYPE_POINTER_ARRAY_ATOMIC,
		ast.TYPE_SLICE_ATOMIC, ast.TYPE_POINTER_SLICE_ATOMIC, ast.TYPE_STRUCT, ast.TYPE_POINTER_STRUCT:
		offset = signature.Offset
	default:
		return code, ReasonDeclarationType
	}
	return append(code, ir.Instr{
		InstrOpcode: ir.OpcodeZero,
		InstrMode:   ir.InstrModeFrameA,
		InstrA:      pointerToUint32(offset),
		InstrN:      pointerToUint32(signature.GetSize(s.statePrgrm, false)),
	}), ReasonNone
}

// builtIn lowers a built-in operator. A jump returns its fixup.
func (s *state) builtIn(code []ir.Instr, line int, cxAtomicOp *ast.CXAtomicOperator, operator *ast.CXFunction) ([]ir.Instr, jumpFixup, bool, ReasonCode, uint32) {
	inputs := cxAtomicOp.GetInputs(s.statePrgrm)
	outputs := cxAtomicOp.GetOutputs(s.statePrgrm)
	opcode := operator.AtomicOPCode
	pc := uint32(len(code))
	noFixup := jumpFixup{}

	switch opcode {
	case constants.OP_NOP:
		return code, noFixup, false, ReasonNone, 0
	case constants.OP_GOTO, constants.OP_BREAK, constants.OP_CONTINUE:
		code = append(code, ir.Instr{InstrOpcode: ir.OpcodeJump})
		return code, jumpFixup{jumpFixupPc: pc, jumpFixupTrue: line + cxAtomicOp.ThenLines + 1}, true, ReasonNone, 0
	case constants.OP_JMP, constants.OP_ABS_JMP:
		if len(inputs) != 1 {
			return code, noFixup, false, ReasonOperandCount, 0
		}
		condition, reason := s.operand(inputs[0])
		if reason != ReasonNone {
			return code, noFixup, false, reason, 0
		}
		in := ir.Instr{InstrOpcode: ir.OpcodeJumpIf}
		operandSet(&in, 0, condition)
		fixup := jumpFixup{
			jumpFixupPc:          pc,
			jumpFixupTrue:        line + cxAtomicOp.ThenLines + 1,
			jumpFixupFalse:       line + cxAtomicOp.ElseLines + 1,
			jumpFixupConditional: true,
		}
		if opcode == constants.OP_ABS_JMP {
			fixup.jumpFixupTrue = cxAtomicOp.ThenLines + 1
			fixup.jumpFixupFalse = cxAtomicOp.ElseLines + 1
		}
		return append(code, in), fixup, true, ReasonNone, 0
	case constants.OP_IDENTITY:
		code, reason := s.identity(code, inputs, outputs)
		return code, noFixup, false, reason, 0
	case constants.OP_TEST:
		code, reason := s.test(code, inputs)
		return code, noFixup, false, reason, 0
	}

	entry := opcodeEntry{}
	if opcode > 0 && opcode < opcodeTableSize {
		entry = s.stateTable[opcode]
	}
	if entry.opcodeEntryOpcode == ir.OpcodeInvalid {
		return code, noFixup, false, ReasonOpcodeOutsideSubset, uint32(opcode)
	}
	if len(inputs) != entry.opcodeEntryInputs || len(outputs) != entry.opcodeEntryOutputs {
		return code, noFixup, false, ReasonOperandCount, 0
	}
	in := ir.Instr{InstrOpcode: entry.opcodeEntryOpcode}
	for i, signatureIndex := range inputs {
		lowered, reason := s.operand(signatureIndex)
		if reason != ReasonNone {
			return code, noFixup, false, reason, 0
		}
		operandSet(&in, i, lowered)
	}
	if len(outputs) == 1 {
		lowered, reason := s.operand(outputs[0])
		if reason != ReasonNone {
			return code, noFixup, false, reason, 0
		}
		operandSet(&in, 2, lowered)
	}
	return append(code, in), noFixup, false, ReasonNone, 0
}

// identity lowers opIdentity for a copy by value. Taking an address, into a
// pointer or by reference, stays outside the subset.
func (s *state) identity(code []ir.Instr, inputs []ast.CXTypeSignatureIndex, outputs []ast.CXTypeSignatureIndex) ([]ir.Instr, ReasonCode) {
	if len(inputs) != 1 || len(outputs) != 1 {
		return code, ReasonOperandCount
	}
	output := s.statePrgrm.GetCXTypeSignatureFromArray(outputs[0])
	if output.Type == ast.TYPE_CXARGUMENT_DEPRECATE && s.statePrgrm.GetCXArgFromArray(ast.CXArgumentIndex(output.Meta)).IsPointer() {
		return code, ReasonOutputPointer
	}
	source, reason := s.operand(inputs[0])
	if reason != ReasonNone {
		return code, reason
	}
	destination, reason := s.operand(outputs[0])
	if reason != ReasonNone {
		return code, reason
	}
	in := ir.Instr{InstrOpcode: ir.OpcodeMove, InstrN: source.operandSize}
	operandSet(&in, 0, source)
	operandSet(&in, 2, destination)
	return append(code, in), ReasonNone
}

// test lowers opTest. assert compares a str argument by its characters,
// which needs the heap, so that stays outside the subset.
func (s *state) test(code []ir.Instr, inputs []ast.CXTypeSignatureIndex) ([]ir.Instr, ReasonCode) {
	if len(inputs) != 3 {
		return code, ReasonOperandCount
	}
	first := s.statePrgrm.GetCXTypeSignatureFromArray(inputs[0])
	if first.Type == ast.TYPE_CXARGUMENT_DEPRECATE && s.statePrgrm.GetCXArgFromArray(ast.CXArgumentIndex(first.Meta)).IsString() {
		return code, ReasonOperandString
	}
	in := ir.Instr{InstrOpcode: ir.OpcodeTestEqual}
	for i, signatureIndex := range inputs {
		lowered, reason := s.operand(signatureIndex)
		if reason != ReasonNone {
			return code, reason
		}
		operandSet(&in, i, lowered)
		switch i {
		case 0:
			in.InstrN = lowered.operandSize
		case 1:
			in.InstrM = lowered.operandSize
		}
	}
	return append(code, in), ReasonNone
}

// call lowers a call of a function, as processNonAtomicOperators and
// popStack do: wipe the callee frame, copy the inputs, call, then copy the
// outputs the caller receives.
func (s *state) call(code []ir.Instr, cxAtomicOp *ast.CXAtomicOperator, callee *ast.CXFunction) ([]ir.Instr, ReasonCode) {
	inputs := cxAtomicOp.GetInputs(s.statePrgrm)
	outputs := cxAtomicOp.GetOutputs(s.statePrgrm)
	calleeInputs := callee.GetInputs(s.statePrgrm)
	calleeOutputs := callee.GetOutputs(s.statePrgrm)
	if len(inputs) > len(calleeInputs) {
		return code, ReasonOperandCount
	}

	code = append(code, ir.Instr{InstrOpcode: ir.OpcodeCallFrame, InstrN: pointerToUint32(callee.Size)})
	for i, signatureIndex := range inputs {
		signature := s.statePrgrm.GetCXTypeSignatureFromArray(signatureIndex)
		if signature.PassBy == constants.PASSBY_REFERENCE {
			return code, ReasonOperandByReference
		}
		source, reason := s.operand(signatureIndex)
		if reason != ReasonNone {
			return code, reason
		}
		destination, reason := s.operand(calleeInputs[i])
		if reason != ReasonNone || !destination.operandFrame {
			return code, ReasonCalleeLayout
		}
		in := ir.Instr{InstrOpcode: ir.OpcodeCallArg, InstrC: destination.operandOffset, InstrN: source.operandSize}
		operandSet(&in, 0, source)
		code = append(code, in)
	}

	code = append(code, ir.Instr{InstrOpcode: ir.OpcodeCall, InstrM: uint32(cxAtomicOp.Operator)})
	for i, signatureIndex := range calleeOutputs {
		if i >= len(outputs) {
			continue
		}
		source, reason := s.operand(signatureIndex)
		if reason != ReasonNone || !source.operandFrame {
			return code, ReasonCalleeLayout
		}
		destination, reason := s.operand(outputs[i])
		if reason != ReasonNone {
			return code, reason
		}
		in := ir.Instr{InstrOpcode: ir.OpcodeCallResult, InstrA: source.operandOffset, InstrN: source.operandSize}
		operandSet(&in, 2, destination)
		code = append(code, in)
	}
	return code, ReasonNone
}

// slots lowers the inputs or outputs of a function to frame slots.
func (s *state) slots(signatures []ast.CXTypeSignatureIndex) ([]ir.Slot, ReasonCode) {
	slots := make([]ir.Slot, len(signatures))
	for i, signatureIndex := range signatures {
		lowered, reason := s.operand(signatureIndex)
		if reason != ReasonNone {
			return nil, reason
		}
		if !lowered.operandFrame {
			return nil, ReasonOperandOutsideData
		}
		slots[i] = ir.Slot{SlotOffset: lowered.operandOffset, SlotSize: lowered.operandSize}
	}
	return slots, ReasonNone
}

// operand lowers an operand whose address GetFinalOffset computes from its
// offset and the frame pointer alone, a plain variable or literal.
func (s *state) operand(signatureIndex ast.CXTypeSignatureIndex) (operand, ReasonCode) {
	signature := s.statePrgrm.GetCXTypeSignatureFromArray(signatureIndex)
	var offset types.Pointer
	switch signature.Type {
	case ast.TYPE_ATOMIC:
		if signature.PassBy == constants.PASSBY_REFERENCE {
			return operand{}, ReasonOperandByReference
		}
		offset = signature.Offset
	case ast.TYPE_CXARGUMENT_DEPRECATE:
		arg := s.statePrgrm.GetCXArg(ast.CXArgumentIndex(signature.Meta))
		if arg.PassBy == constants.PASSBY_REFERENCE {
			return operand{}, ReasonOperandByReference
		}
		if len(arg.Fields) > 0 || len(arg.Indexes) > 0 {
			return operand{}, ReasonOperandIndexed
		}
		for _, deref := range arg.DereferenceOperations {
			if deref == constants.DEREF_POINTER {
				return operand{}, ReasonOperandIndexed
			}
		}
		if arg.IsPointer() || arg.IsStruct() || len(arg.Lengths) > 0 {
			return operand{}, ReasonOperandNotScalar
		}
		offset = arg.Offset
	default:
		return operand{}, ReasonOperandNotScalar
	}

	lowered := operand{
		operandOffset: pointerToUint32(offset),
		operandSize:   pointerToUint32(signature.GetSize(s.statePrgrm, false)),
	}
	if lowered.operandOffset < s.stateStackSize {
		lowered.operandFrame = true
	} else if lowered.operandOffset < s.stateDataStart || lowered.operandOffset+lowered.operandSize > s.stateDataEnd {
		return operand{}, ReasonOperandOutsideData
	}
	return lowered, ReasonNone
}

// operandSet puts lowered in operand slot i of in, 0 to 3 for A to D.
func operandSet(in *ir.Instr, i int, lowered operand) {
	var mode ir.InstrMode
	switch i {
	case 0:
		in.InstrA = lowered.operandOffset
		mode = ir.InstrModeFrameA
	case 1:
		in.InstrB = lowered.operandOffset
		mode = ir.InstrModeFrameB
	case 2:
		in.InstrC = lowered.operandOffset
		mode = ir.InstrModeFrameC
	case 3:
		in.InstrD = lowered.operandOffset
		mode = ir.InstrModeFrameD
	default:
		assert.AssertUnreachable(fmt.Sprintf("lower: operand slot %d", i))
	}
	if lowered.operandFrame {
		in.InstrMode |= mode
	}
}

// file returns the index of name in ImageFileNames, adding it when new.
func (s *state) file(name string) uint32 {
	if index, found := s.stateFiles[name]; found {
		return index
	}
	index := uint32(len(s.stateImage.ImageFileNames))
	s.stateImage.ImageFileNames = append(s.stateImage.ImageFileNames, name)
	s.stateFiles[name] = index
	return index
}

// pointerToUint32 narrows an offset or size. An image addresses 4 GB.
func pointerToUint32(pointer types.Pointer) uint32 {
	assert.AssertInvariantTrue(uint64(pointer) <= uint64(^uint32(0)), "lower: offset does not fit in 32 bits")
	return uint32(pointer)
}
