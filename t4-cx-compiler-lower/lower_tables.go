package lower

import (
	"sort"
	"strconv"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-types"
	stringpool "github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	// GCREACH: t3-ast owns the heap and the GC roots. The side tables read only
	// type signatures and names, see docs/GarbageCollector.md.
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// The side tables of the image, decision D8: the extent, name and type of
// every frame variable, and the literals of the data segment.

// localAdd records the variable of signatureIndex, size bytes at offset in
// the frame of the current function.
func (s *state) localAdd(signatureIndex ast.CXTypeSignatureIndex, offset uint32, size uint32) {
	prgrm := s.statePrgrm
	signature := prgrm.GetCXTypeSignatureFromArray(signatureIndex)
	name := signature.Name
	code := types.UNDEFINED
	typeName := ""
	switch signature.Type {
	case ast.TYPE_ATOMIC:
		code = types.Code(signature.Meta)
		typeName = code.Name()
	case ast.TYPE_CXARGUMENT_DEPRECATE:
		arg := prgrm.GetCXArgFromArray(ast.CXArgumentIndex(signature.Meta))
		name = arg.Name
		typeName = localArgumentTypeName(prgrm, arg)
		if arg.StructType == nil && len(arg.Lengths) == 0 {
			code = arg.Type
		}
	default:
		typeName = signature.GetTypeString(prgrm)
		if signature.Type == ast.TYPE_POINTER_ATOMIC || signature.Type == ast.TYPE_POINTER_ARRAY_ATOMIC ||
			signature.Type == ast.TYPE_POINTER_SLICE_ATOMIC || signature.Type == ast.TYPE_POINTER_STRUCT {
			typeName = "*" + typeName
		}
	}
	if !localScalar(code) {
		code = types.UNDEFINED
	}
	s.stateLocals = append(s.stateLocals, ir.Local{
		LocalOffset:   offset,
		LocalSize:     size,
		LocalName:     s.localName(prgrm, name),
		LocalTypeName: s.localNameText(typeName),
		LocalType:     code,
	})
}

// localArgumentTypeName spells the type of arg, its lengths, a * for a
// pointer, then the struct or element type.
func localArgumentTypeName(prgrm *ast.CXProgram, arg *ast.CXArgument) string {
	text := ""
	for _, length := range arg.Lengths {
		text += "[" + strconv.FormatUint(uint64(length), 10) + "]"
	}
	element := arg.Type
	if arg.Type == types.POINTER {
		text += "*"
		element = arg.PointerTargetType
	}
	if arg.StructType != nil {
		return text + prgrm.CXProgramNameText(arg.StructType.Name)
	}
	return text + element.Name()
}

// localScalar reports whether code is a bool, integer or float type.
func localScalar(code types.Code) bool {
	switch code {
	case types.BOOL, types.I8, types.I16, types.I32, types.I64,
		types.UI8, types.UI16, types.UI32, types.UI64, types.F32, types.F64:
		return true
	}
	return false
}

func (s *state) localName(prgrm *ast.CXProgram, name stringpool.StringGid) uint32 {
	if name == stringpool.StringGidNil {
		return s.localNameText("")
	}
	// A local variable is n:name, see actions.StripNameNumber.
	text := prgrm.CXProgramNameText(name)
	if index := strings.IndexByte(text, ':'); index >= 0 {
		text = text[index+1:]
	}
	return s.localNameText(text)
}

// localNameText returns the index of text in ImageLocalNames, adding it when
// new.
func (s *state) localNameText(text string) uint32 {
	if index, found := s.stateLocalNames[text]; found {
		return index
	}
	index := uint32(len(s.stateImage.ImageLocalNames))
	s.stateImage.ImageLocalNames = append(s.stateImage.ImageLocalNames, text)
	s.stateLocalNames[text] = index
	return index
}

// localsEnd adds the variables of the current function to the image, sorted
// by offset with repeats of one variable left out, and empties the list.
func (s *state) localsEnd(lowered *ir.Func) {
	locals := s.stateLocals
	sort.SliceStable(locals, func(i, j int) bool {
		if locals[i].LocalOffset != locals[j].LocalOffset {
			return locals[i].LocalOffset < locals[j].LocalOffset
		}
		return locals[i].LocalSize < locals[j].LocalSize
	})
	lowered.FuncLocalFirst = uint32(len(s.stateImage.ImageLocals))
	for i, local := range locals {
		if i > 0 && local == locals[i-1] {
			continue
		}
		s.stateImage.ImageLocals = append(s.stateImage.ImageLocals, local)
	}
	lowered.FuncLocalCount = uint32(len(s.stateImage.ImageLocals)) - lowered.FuncLocalFirst
	s.stateLocals = s.stateLocals[:0]
}

// readOnlyAdd records a literal read from signatureIndex at the static
// place p. A literal is a type signature or argument without a name.
func (s *state) readOnlyAdd(signatureIndex ast.CXTypeSignatureIndex, p place) {
	if p.placeComputed || p.placeFrame || p.placeSize == 0 {
		return
	}
	prgrm := s.statePrgrm
	signature := prgrm.GetCXTypeSignatureFromArray(signatureIndex)
	name := signature.Name
	if signature.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
		arg := prgrm.GetCXArgFromArray(ast.CXArgumentIndex(signature.Meta))
		if len(arg.Indexes) > 0 || len(arg.Fields) > 0 || len(arg.DereferenceOperations) > 0 {
			return
		}
		name = arg.Name
	} else if signature.Type != ast.TYPE_ATOMIC {
		return
	}
	if name != stringpool.StringGidNil {
		return
	}
	s.stateReadOnly = append(s.stateReadOnly, ir.Span{SpanStart: p.placeOffset, SpanEnd: p.placeOffset + p.placeSize})
}

// readOnlyEnd sorts and joins the literal spans into ImageReadOnly. It takes
// out every byte an instruction writes at an absolute address and every run
// of literals whose address an instruction takes, so the table holds only
// data no instruction changes.
func (s *state) readOnlyEnd() {
	start, end := s.stateDataStart, s.stateDataEnd
	marks := make([]bool, end-start)
	for _, span := range s.stateReadOnly {
		for address := max(span.SpanStart, start); address < span.SpanEnd && address < end; address++ {
			marks[address-start] = true
		}
	}
	for functionIndex := range s.stateImage.ImageFuncs {
		code := s.stateImage.ImageFuncs[functionIndex].FuncCode
		for pc := range code {
			if address, found := instrAbsoluteAddress(&code[pc]); found && address >= start && address < end {
				for low := address - start; low < uint32(len(marks)) && marks[low]; low-- {
					marks[low] = false
					if low == 0 {
						break
					}
				}
				for high := address - start + 1; high < uint32(len(marks)) && marks[high]; high++ {
					marks[high] = false
				}
			}
			offset, size, writes := instrAbsoluteWrite(&code[pc])
			if !writes {
				continue
			}
			for address := uint64(offset); address < uint64(offset)+uint64(size) && address < uint64(end); address++ {
				if address >= uint64(start) {
					marks[address-uint64(start)] = false
				}
			}
		}
	}
	var spans []ir.Span
	for i := 0; i < len(marks); i++ {
		if !marks[i] {
			continue
		}
		j := i
		for j < len(marks) && marks[j] {
			j++
		}
		spans = append(spans, ir.Span{SpanStart: start + uint32(i), SpanEnd: start + uint32(j)})
		i = j
	}
	s.stateImage.ImageReadOnly = spans
}

// instrAbsoluteWriteMax is the most bytes a scalar opcode writes, an i64,
// ui64 or f64.
const instrAbsoluteWriteMax = 8

// instrAbsoluteWrite returns the operand in writes at an absolute address
// and an upper bound of its size. A write through a computed address is not
// one, the lowering only computes addresses of variables.
func instrAbsoluteWrite(in *ir.Instr) (uint32, uint32, bool) {
	if ir.OpcodeLibrary(in.InstrOpcode) {
		return 0, 0, false
	}
	switch in.InstrOpcode {
	case ir.OpcodeZero:
		return in.InstrA, in.InstrN, in.InstrMode&ir.InstrModeFrameA == 0
	case ir.OpcodeReturn, ir.OpcodeJump, ir.OpcodeJumpIf, ir.OpcodeCallFrame, ir.OpcodeCallArg, ir.OpcodeCall,
		ir.OpcodeTestEqual, ir.OpcodeAddressStore, ir.OpcodePrintText, ir.OpcodePrintValue, ir.OpcodePrintStr,
		ir.OpcodeLibraryCall:
		return 0, 0, false
	}
	size := uint32(instrAbsoluteWriteMax)
	switch in.InstrOpcode {
	case ir.OpcodeMove, ir.OpcodeCallResult, ir.OpcodeAddressLoad:
		size = in.InstrN
	}
	return in.InstrC, size, in.InstrMode&ir.InstrModeFrameC == 0
}

// instrAbsoluteAddress returns the absolute address whose value in takes as
// a pointer or as the base of an index.
func instrAbsoluteAddress(in *ir.Instr) (uint32, bool) {
	switch in.InstrOpcode {
	case ir.OpcodeAddressOf, ir.OpcodeAddressIndexStatic:
		return in.InstrA, in.InstrMode&ir.InstrModeFrameA == 0
	}
	return 0, false
}
