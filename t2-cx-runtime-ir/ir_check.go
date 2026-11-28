package ir

import (
	"math"
)

// ImageCheckProblem says why ImageCheck refused an image.
type ImageCheckProblem uint32

const (
	// ImageCheckProblemNone is an image vm may run.
	ImageCheckProblemNone ImageCheckProblem = iota
	// ImageCheckProblemLayout is a stack, data segment and scratch that do
	// not fit one memory of at most 4 GiB with room after the stack.
	ImageCheckProblemLayout
	// ImageCheckProblemEntry is an entry that is no function with code.
	ImageCheckProblemEntry
	// ImageCheckProblemTables is a function whose lines or variables lie
	// outside the tables of the image, or a frame larger than the stack.
	ImageCheckProblemTables
	// ImageCheckProblemOpcode is OpcodeInvalid or a number past the last
	// opcode.
	ImageCheckProblemOpcode
	// ImageCheckProblemEnd is a function whose last instruction is neither
	// OpcodeReturn nor OpcodeJump, so the run could leave its code.
	ImageCheckProblemEnd
	// ImageCheckProblemJump is a jump outside its function.
	ImageCheckProblemJump
	// ImageCheckProblemCall is a call of a function that has no code, a call
	// or a library opcode without an OpcodeCallFrame before it in the same
	// block, or a frame that does not hold the callee.
	ImageCheckProblemCall
	// ImageCheckProblemOperand is an operand outside the memory or outside
	// the frame of its function.
	ImageCheckProblemOperand
	// ImageCheckProblemText is a print of bytes outside ImageText.
	ImageCheckProblemText
	// ImageCheckProblemOperation is a library opcode the image does not
	// enable in ImageOperationsEnabled (decision D119).
	ImageCheckProblemOperation
	// ImageCheckProblemSize is a function longer than the labels the caller
	// gave room for.
	ImageCheckProblemSize
	// ImageCheckProblemCount is one past the last problem.
	ImageCheckProblemCount
)

// ImageCheckProblemMessages describes each problem. Read only.
var ImageCheckProblemMessages = [ImageCheckProblemCount]string{
	ImageCheckProblemNone:      "",
	ImageCheckProblemLayout:    "the stack, data and scratch of the image do not fit its memory",
	ImageCheckProblemEntry:     "an entry of the image is no function with code",
	ImageCheckProblemTables:    "a function of the image has lines, variables or a frame outside the image",
	ImageCheckProblemOpcode:    "the image has an invalid opcode",
	ImageCheckProblemEnd:       "a function of the image does not end with a return or a jump",
	ImageCheckProblemJump:      "the image jumps outside a function",
	ImageCheckProblemCall:      "the image calls outside a prepared frame",
	ImageCheckProblemOperand:   "the image has an operand outside its memory or frame",
	ImageCheckProblemText:      "the image prints text it does not hold",
	ImageCheckProblemOperation: "the image uses a library operation it does not enable",
	ImageCheckProblemSize:      "a function of the image is too long",
}

// ImageCheckTailSize is the memory a machine keeps after the scratch. A
// library opcode reads and writes at most this many bytes of its block at
// sp, and a scalar operand at the end of a frame or of the memory at most 8.
const ImageCheckTailSize = 128

// imageCheckScalarSize is the widest scalar an operand of a scalar opcode
// reads or writes.
const imageCheckScalarSize = 8

// ImageCheckLabelWords returns the uint64 words of labels ImageCheck needs
// for the longest function of image.
func ImageCheckLabelWords(image *Image) int {
	longest := 0
	for i := range image.ImageFuncs {
		longest = max(longest, len(image.ImageFuncs[i].FuncCode))
	}
	return (longest + 63) / 64
}

// ImageCheck is the loader check of an image that did not come from the
// compiler, run once before the run (decision D153). It checks what vm
// trusts: the layout, the entries and tables, every opcode, jump, call and
// operand, the text a print reads and the library opcodes against
// ImageOperationsEnabled. An image that passes cannot make vm index outside
// its memory, code or tables. labels is scratch of ImageCheckLabelWords
// words, so the check allocates nothing.
func ImageCheck(image *Image, labels []uint64) ImageCheckProblem {
	memory := uint64(image.ImageDataStart) + uint64(len(image.ImageData)) + uint64(image.ImageScratchSize)
	if image.ImageDataStart < image.ImageStackSize || memory+ImageCheckTailSize > math.MaxUint32 {
		return ImageCheckProblemLayout
	}
	if !imageCheckEntry(image, image.ImageEntryMain) || (image.ImageEntryInit != ImageFuncNone && !imageCheckEntry(image, image.ImageEntryInit)) {
		return ImageCheckProblemEntry
	}
	for function := range image.ImageFuncs {
		if problem := imageCheckFunction(image, uint32(function), memory, labels); problem != ImageCheckProblemNone {
			return problem
		}
	}
	return ImageCheckProblemNone
}

// ImageCheckNames checks that the lines and variables of image name files
// and variables of its name tables, which a runner reads to print an error.
// ImageCheck leaves them out, as a run reads no name.
func ImageCheckNames(image *Image) ImageCheckProblem {
	for _, line := range image.ImageLines {
		if uint64(line.LineFile) >= uint64(len(image.ImageFileNames)) {
			return ImageCheckProblemTables
		}
	}
	for _, local := range image.ImageLocals {
		if uint64(local.LocalName) >= uint64(len(image.ImageLocalNames)) || uint64(local.LocalTypeName) >= uint64(len(image.ImageLocalNames)) {
			return ImageCheckProblemTables
		}
	}
	return ImageCheckProblemNone
}

// imageCheckEntry reports whether entry is a function with code.
func imageCheckEntry(image *Image, entry uint32) bool {
	return uint64(entry) < uint64(len(image.ImageFuncs)) && len(image.ImageFuncs[entry].FuncCode) > 0
}

// imageCheckFunction checks one function, see ImageCheck.
func imageCheckFunction(image *Image, function uint32, memory uint64, labels []uint64) ImageCheckProblem {
	fn := &image.ImageFuncs[function]
	if uint64(fn.FuncLineFirst)+uint64(fn.FuncLineCount) > uint64(len(image.ImageLines)) ||
		uint64(fn.FuncLocalFirst)+uint64(fn.FuncLocalCount) > uint64(len(image.ImageLocals)) || fn.FuncFrameSize > image.ImageStackSize {
		return ImageCheckProblemTables
	}
	code := fn.FuncCode
	if len(code) == 0 {
		return ImageCheckProblemNone
	}
	words := (len(code) + 63) / 64
	if words > len(labels) {
		return ImageCheckProblemSize
	}
	clear(labels[:words])
	count := uint32(len(code))
	for pc := range code {
		in := &code[pc]
		switch in.InstrOpcode {
		case OpcodeJump:
			if in.InstrM >= count {
				return ImageCheckProblemJump
			}
			labels[in.InstrM/64] |= 1 << (in.InstrM % 64)
		case OpcodeJumpIf:
			if in.InstrM >= count || in.InstrN >= count {
				return ImageCheckProblemJump
			}
			labels[in.InstrM/64] |= 1 << (in.InstrM % 64)
			labels[in.InstrN/64] |= 1 << (in.InstrN % 64)
		}
	}
	if last := code[len(code)-1].InstrOpcode; last != OpcodeReturn && last != OpcodeJump {
		return ImageCheckProblemEnd
	}
	operands := imageCheckOperands{memory: memory, frame: uint64(fn.FuncFrameSize)}
	// pending is the frame of the call being prepared and returned the frame
	// of the call that returned last, -1 when there is none. A jump target
	// starts a block, which knows neither.
	pending, returned := int64(-1), int64(-1)
	for pc := range code {
		if labels[pc/64]&(1<<(pc%64)) != 0 {
			pending, returned = -1, -1
		}
		in := &code[pc]
		op := in.InstrOpcode
		switch {
		case op == OpcodeInvalid || op >= OpcodeCount:
			return ImageCheckProblemOpcode
		case op == OpcodeCallFrame:
			pending = int64(in.InstrN)
		case op == OpcodeCallArg:
			if pending < 0 || uint64(in.InstrC)+uint64(in.InstrN) > uint64(pending) {
				return ImageCheckProblemCall
			}
		case op == OpcodeCall:
			if !imageCheckEntry(image, in.InstrM) || pending < int64(image.ImageFuncs[in.InstrM].FuncFrameSize) {
				return ImageCheckProblemCall
			}
			pending, returned = -1, pending
		case op == OpcodeCallResult:
			if returned < 0 || uint64(in.InstrA)+uint64(in.InstrN) > uint64(returned) {
				return ImageCheckProblemCall
			}
		case OpcodeLibrary(op):
			if pending < 0 {
				return ImageCheckProblemCall
			}
			if !image.ImageOperationsEnabled.OpcodeSetContains(OpcodeLibraryOperation(op)) {
				return ImageCheckProblemOperation
			}
			pending, returned = -1, pending
		case op == OpcodePrintText:
			if uint64(in.InstrN)+uint64(in.InstrM) > uint64(len(image.ImageText)) {
				return ImageCheckProblemText
			}
		}
		if !operands.instrValid(in) {
			return ImageCheckProblemOperand
		}
	}
	return ImageCheckProblemNone
}

// imageCheckOperands checks the operands of the instructions of one
// function against the memory and its frame.
type imageCheckOperands struct {
	memory uint64
	frame  uint64
}

// operandValid reports whether operand field of in, of size bytes, lies in
// the frame or the memory. A scalar operand of at most imageCheckScalarSize
// bytes needs only its first byte there, as the tail of the machine holds
// the rest.
func (operands imageCheckOperands) operandValid(in *Instr, field int, size uint64, scalar bool) bool {
	offset := uint64(in.InstrA)
	switch field {
	case 1:
		offset = uint64(in.InstrB)
	case 2:
		offset = uint64(in.InstrC)
	}
	if in.InstrMode&(InstrModeFrameA<<field) != 0 {
		if scalar {
			return offset < operands.frame
		}
		return offset+size <= operands.frame
	}
	if scalar {
		return offset < operands.memory
	}
	return offset+size <= operands.memory
}

// scalarsValid reports whether the scalar operands of in that fields marks,
// bit i for operand i, are valid.
func (operands imageCheckOperands) scalarsValid(in *Instr, fields uint8) bool {
	for field := 0; field < 3; field++ {
		if fields&(1<<field) != 0 && !operands.operandValid(in, field, imageCheckScalarSize, true) {
			return false
		}
	}
	return true
}

// instrValid reports whether every operand in reads or writes is valid.
func (operands imageCheckOperands) instrValid(in *Instr) bool {
	const a, b, c = 0, 1, 2
	const scalarA, scalarB, scalarC = 1 << a, 1 << b, 1 << c
	n, m := uint64(in.InstrN), uint64(in.InstrM)
	switch op := in.InstrOpcode; {
	case op == OpcodeZero:
		return operands.operandValid(in, a, n, false)
	case op == OpcodeMove:
		return operands.operandValid(in, a, n, false) && operands.operandValid(in, c, n, false)
	case op == OpcodeCallArg:
		return operands.operandValid(in, a, n, false)
	case op == OpcodeCallResult:
		return operands.operandValid(in, c, n, false)
	case op == OpcodeTestEqual:
		return operands.operandValid(in, a, n, false) && operands.operandValid(in, b, m, false) && operands.scalarsValid(in, scalarC)
	case op == OpcodeAddressLoad:
		return operands.scalarsValid(in, scalarA) && operands.operandValid(in, c, n, false)
	case op == OpcodeAddressStore:
		return operands.operandValid(in, a, n, false) && operands.scalarsValid(in, scalarC)
	case op == OpcodeJumpIf, op == OpcodePrintValue, op == OpcodePrintStr:
		return operands.scalarsValid(in, scalarA)
	case op == OpcodeAddressIndexStatic:
		return operands.scalarsValid(in, scalarB|scalarC)
	case op == OpcodeAddressIndex:
		return operands.scalarsValid(in, scalarA|scalarB|scalarC)
	case op == OpcodeAddressDerefStatic, op == OpcodeAddressDeref, op == OpcodeAddressPointer:
		return operands.scalarsValid(in, scalarA|scalarC)
	case op == OpcodeAddressOf:
		return operands.scalarsValid(in, scalarC)
	case op >= OpcodeBoolNot && op < OpcodeCount:
		return operands.scalarsValid(in, scalarA|scalarB|scalarC)
	}
	return true
}
