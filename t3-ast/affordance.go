package ast

import (
	"encoding/binary"
	"slices"

	"github.com/skycoin/skycoin-cx/t0-types"
	stringpool "github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
)

// AffordanceKind is what an aff value selects, the first i32 of the value.
// The zero value selects the whole program, as #{} does. See "Affordances"
// in docs/IR.md.
type AffordanceKind int32

const (
	AffordanceKindProgram AffordanceKind = iota
	// AffordanceKindMemory is mem(data), mem(stack) or mem(heap),
	// AffordanceValueItem holds the AffordanceMemory.
	AffordanceKindMemory
	AffordanceKindPackage
	AffordanceKindFunction
	AffordanceKindStruct
	// AffordanceKindGlobal and AffordanceKindLocal are var(name), a global
	// or a variable of AffordanceValueItem, the function.
	AffordanceKindGlobal
	AffordanceKindLocal
	AffordanceKindCount
)

// AffordanceMemory is the segment mem(...) selects.
type AffordanceMemory int32

const (
	AffordanceMemoryData AffordanceMemory = iota
	AffordanceMemoryStack
	AffordanceMemoryHeap
	AffordanceMemoryCount
)

// AffordanceMemoryNames spells each AffordanceMemory as mem(...) takes it.
// Read only.
var AffordanceMemoryNames = [AffordanceMemoryCount]string{"data", "stack", "heap"}

// AffordanceValue is an aff value, types.AFF_SIZE bytes of program memory,
// four little endian i32 in field order. Indexes are -1 when unused.
type AffordanceValue struct {
	AffordanceValueKind     AffordanceKind
	AffordanceValuePackage  CXPackageIndex
	AffordanceValueItem     int32
	AffordanceValueVariable CXTypeSignatureIndex
}

// AffordanceValueWrite writes value at offset of memory.
func AffordanceValueWrite(memory []byte, offset types.Pointer, value AffordanceValue) {
	types.Write_i32(memory, offset, int32(value.AffordanceValueKind))
	types.Write_i32(memory, offset+4, int32(value.AffordanceValuePackage))
	types.Write_i32(memory, offset+8, value.AffordanceValueItem)
	types.Write_i32(memory, offset+12, int32(value.AffordanceValueVariable))
}

// AffordanceValueRead reads the aff value at offset of memory.
func AffordanceValueRead(memory []byte, offset types.Pointer) AffordanceValue {
	return AffordanceValue{
		AffordanceValueKind:     AffordanceKind(types.Read_i32(memory, offset)),
		AffordanceValuePackage:  CXPackageIndex(types.Read_i32(memory, offset+4)),
		AffordanceValueItem:     types.Read_i32(memory, offset+8),
		AffordanceValueVariable: CXTypeSignatureIndex(types.Read_i32(memory, offset+12)),
	}
}

// AffordanceClause names a clause of an aff literal.
type AffordanceClause int32

const (
	AffordanceClauseMemory AffordanceClause = iota
	AffordanceClausePackage
	AffordanceClauseFunction
	AffordanceClauseVariable
	AffordanceClauseStruct
	AffordanceClauseCount
)

// AffordanceClauseNames spells each AffordanceClause. Read only.
var AffordanceClauseNames = [AffordanceClauseCount]string{"mem", "pkg", "fn", "var", "strct"}

// AffordanceClauseArgument is the name in the parentheses of one clause and
// where the clause starts. A clause the literal leaves out has StringGidNil.
type AffordanceClauseArgument struct {
	AffordanceClauseArgumentName   stringpool.StringGid
	AffordanceClauseArgumentLine   int
	AffordanceClauseArgumentColumn int
}

// AffordanceLiteral is an aff literal whose value the compiler writes once
// every function body is compiled, at AffordanceLiteralOffset of the data
// segment, so that var can name a variable declared after the literal.
type AffordanceLiteral struct {
	AffordanceLiteralOffset    types.Pointer
	AffordanceLiteralPackage   CXPackageIndex
	AffordanceLiteralFile      SourceFileGid
	AffordanceLiteralClauses   [AffordanceClauseCount]AffordanceClauseArgument
	AffordanceLiteralFirstLine int
}

// CXProgramAffordanceLiteralAdd records literal for
// CXProgramAffordanceLiteralsTake.
func (prgrm *CXProgram) CXProgramAffordanceLiteralAdd(literal AffordanceLiteral) {
	prgrm.CxIrState.CxIrStateAffordanceLiterals = append(prgrm.CxIrState.CxIrStateAffordanceLiterals, literal)
}

// CXProgramAffordanceLiteralsTake returns the literals recorded since the
// last call and forgets them.
func (prgrm *CXProgram) CXProgramAffordanceLiteralsTake() []AffordanceLiteral {
	literals := prgrm.CxIrState.CxIrStateAffordanceLiterals
	prgrm.CxIrState.CxIrStateAffordanceLiterals = nil
	return literals
}

// serializeAffordanceMagic starts the bytes serialize gives for a package,
// function or struct, which Deserialize does not load.
const serializeAffordanceMagic = "CXSELECT"

// SerializeAffordance returns the bytes serialize gives for value: the whole
// program, the bytes in use of a memory segment, the bytes of a variable, or
// a package, function or struct as the serialized program encodes it. A
// variable of a function that has no call on the call stack, and an index
// outside the program, are an error.
// VIOLATION VMALLOC-019: the selection runs the encoder of the serializer, compiler code, #407.
func SerializeAffordance(prgrm *CXProgram, value AffordanceValue) ([]byte, bool) {
	switch value.AffordanceValueKind {
	case AffordanceKindProgram:
		b, err := SerializeCXProgram(prgrm, true, nil)
		return b, err == nil
	case AffordanceKindMemory:
		var first, last types.Pointer
		switch AffordanceMemory(value.AffordanceValueItem) {
		case AffordanceMemoryData:
			first, last = prgrm.Data.StartsAt, prgrm.Data.StartsAt+prgrm.Data.Size
		case AffordanceMemoryStack:
			first, last = prgrm.Stack.StartsAt, prgrm.Stack.Pointer
		case AffordanceMemoryHeap:
			first, last = prgrm.Heap.StartsAt, prgrm.Heap.StartsAt+prgrm.Heap.Pointer
		default:
			return nil, false
		}
		return affordanceBytes(prgrm, first, last)
	case AffordanceKindGlobal, AffordanceKindLocal:
		return affordanceVariableBytes(prgrm, value)
	}
	if int(value.AffordanceValuePackage) < 0 || int(value.AffordanceValuePackage) >= len(prgrm.CXPackages) {
		return nil, false
	}
	encoder := serializeEncoder{serializeEncoderBytes: make([]byte, 0, 256)}
	encoder.serializeEncoderBytes = append(encoder.serializeEncoderBytes, serializeAffordanceMagic...)
	encoder.serializeEncoderBytes = binary.LittleEndian.AppendUint32(encoder.serializeEncoderBytes, serializeFormatVersion)
	encoder.serializeEncoderBytes = binary.LittleEndian.AppendUint32(encoder.serializeEncoderBytes, uint32(types.POINTER_SIZE))
	encoder.serializeEncoderBytes = binary.LittleEndian.AppendUint64(encoder.serializeEncoderBytes, uint64(prgrm.Version))
	encoder.serializeEncoderUint(uint64(value.AffordanceValueKind))
	pkg := &prgrm.CXPackages[value.AffordanceValuePackage]
	switch value.AffordanceValueKind {
	case AffordanceKindPackage:
		serializePackage(&encoder, pkg)
		structs := affordanceIndexesSorted(pkg.Structs)
		encoder.serializeEncoderLength(len(structs), false)
		for _, index := range structs {
			serializeStruct(&encoder, &prgrm.CXStructs[index])
		}
		functions := affordanceIndexesSorted(pkg.Functions)
		encoder.serializeEncoderLength(len(functions), false)
		for _, index := range functions {
			if serializeFunction(prgrm, &encoder, &prgrm.CXFunctions[index]) != nil {
				return nil, false
			}
		}
	case AffordanceKindFunction:
		if int(value.AffordanceValueItem) < 0 || int(value.AffordanceValueItem) >= len(prgrm.CXFunctions) {
			return nil, false
		}
		if serializeFunction(prgrm, &encoder, &prgrm.CXFunctions[value.AffordanceValueItem]) != nil {
			return nil, false
		}
	case AffordanceKindStruct:
		if int(value.AffordanceValueItem) < 0 || int(value.AffordanceValueItem) >= len(prgrm.CXStructs) {
			return nil, false
		}
		serializeStruct(&encoder, &prgrm.CXStructs[value.AffordanceValueItem])
	default:
		return nil, false
	}
	return encoder.serializeEncoderBytes, true
}

// affordanceBytes copies the bytes of memory from first up to last.
func affordanceBytes(prgrm *CXProgram, first types.Pointer, last types.Pointer) ([]byte, bool) {
	if first > last || int(last) > len(prgrm.Memory) {
		return nil, false
	}
	return append([]byte{}, prgrm.Memory[first:last]...), true
}

// affordanceVariableBytes copies the bytes of a global, or of a variable in
// the innermost call of its function.
func affordanceVariableBytes(prgrm *CXProgram, value AffordanceValue) ([]byte, bool) {
	if value.AffordanceValueVariable < 0 || uint32(value.AffordanceValueVariable) >= prgrm.CxIrState.CXTypeSignatureMng.CXTypeSignatureMngNext {
		return nil, false
	}
	variable := prgrm.GetCXTypeSignatureFromArray(value.AffordanceValueVariable)
	var framePointer types.Pointer
	if value.AffordanceValueKind == AffordanceKindLocal {
		found := false
		for call := int(prgrm.CallCounter); call >= 0 && call < len(prgrm.CallStack); call-- {
			operator := prgrm.CallStack[call].Operator
			if operator != nil && operator.Index == int(value.AffordanceValueItem) {
				framePointer = prgrm.CallStack[call].FramePointer
				found = true
				break
			}
		}
		if !found {
			return nil, false
		}
	}
	address := frameAddress(prgrm, framePointer, variable.GetOffset(prgrm))
	return affordanceBytes(prgrm, address, address+variable.GetSize(prgrm, true))
}

// affordanceIndexesSorted returns the indexes of names in increasing order.
func affordanceIndexesSorted[Index ~int](names map[stringpool.StringGid]Index) []Index {
	indexes := make([]Index, 0, len(names))
	for _, index := range names {
		indexes = append(indexes, index)
	}
	slices.Sort(indexes)
	return indexes
}

// CXTypeSignatureWhole reports whether sig names a whole variable, without
// an index, a field or a dereference.
func CXTypeSignatureWhole(prgrm *CXProgram, sig *CXTypeSignature) bool {
	switch sig.Type {
	case TYPE_ATOMIC, TYPE_SLICE_ATOMIC:
		return sig.Type == TYPE_ATOMIC || len(prgrm.GetCXTypeSignatureArrayFromArray(sig.Meta).Indexes) == 0
	case TYPE_POINTER_ATOMIC, TYPE_POINTER_SLICE_ATOMIC:
		return !sig.IsDeref
	case TYPE_ARRAY_ATOMIC, TYPE_POINTER_ARRAY_ATOMIC:
		return !sig.IsDeref && len(prgrm.GetCXTypeSignatureArrayFromArray(sig.Meta).Indexes) == 0
	case TYPE_STRUCT, TYPE_POINTER_STRUCT:
		return !sig.IsDeref && len(prgrm.GetCXTypeSignatureStructFromArray(sig.Meta).Fields) == 0
	case TYPE_CXARGUMENT_DEPRECATE:
		arg := prgrm.GetCXArgFromArray(CXArgumentIndex(sig.Meta))
		return len(arg.Fields) == 0 && len(arg.Indexes) == 0 && len(arg.DereferenceOperations) == 0
	}
	return false
}
