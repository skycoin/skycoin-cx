package actions

import (
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// gcRootsDeclare adds a GC root for each pointer, slice and str inside a new
// local, found from its declared type, also inside an escaped local.
func gcRootsDeclare(prgrm *ast.CXProgram, fn *ast.CXFunction, typeSig *ast.CXTypeSignature, sym *ast.CXArgument) {
	offset := typeSig.Offset
	specs := escapeSignatureSpecifiers(prgrm, typeSig)
	strct := typeSig.GetStructType(prgrm)
	var typ types.Code
	if typeSig.Type == ast.TYPE_ATOMIC || typeSig.Type == ast.TYPE_POINTER_ATOMIC {
		typ = types.Code(typeSig.Meta)
	}
	var lengths []types.Pointer
	switch {
	case sym != nil:
		offset = sym.Offset
		specs, typ, strct, lengths = sym.DeclarationSpecifiers, sym.Type, sym.StructType, sym.Lengths
	case typeSig.Type == ast.TYPE_ARRAY_ATOMIC, typeSig.Type == ast.TYPE_POINTER_ARRAY_ATOMIC, typeSig.Type == ast.TYPE_SLICE_ATOMIC, typeSig.Type == ast.TYPE_POINTER_SLICE_ATOMIC:
		details := prgrm.GetCXTypeSignatureArrayFromArray(typeSig.Meta)
		typ, lengths = types.Code(details.Meta), details.Lengths
	}
	if offset >= prgrm.Stack.Size && !ast.OffsetEscaped(offset) {
		return
	}
	for _, slot := range gcPointerOffsets(prgrm, specs, typ, strct, lengths, 0, nil, 0) {
		gcRootAdd(fn, offset, slot)
	}
}

// gcRootAdd adds slot, inside the variable at offset, to fn.ListOfSlots.
func gcRootAdd(fn *ast.CXFunction, offset types.Pointer, slot gcPointerSlot) {
	fn.ListOfSlots = append(fn.ListOfSlots, ast.CXFunctionSlot{
		CXFunctionSlotOffset:     offset,
		CXFunctionSlotInner:      slot.gcPointerSlotOffset,
		CXFunctionSlotType:       slot.gcPointerSlotType,
		CXFunctionSlotSpecifiers: slot.gcPointerSlotSpecs,
		CXFunctionSlotStruct:     slot.gcPointerSlotStruct,
	})
}

// gcPointerDepthLimit bounds the struct nesting gcPointerOffsets follows.
const gcPointerDepthLimit = 16

// gcPointerSlotLimit bounds the roots of one variable.
const gcPointerSlotLimit = 4096

// gcPointerOffsets appends to offsets the offset, from base, of each pointer
// in a value of declaration specifiers specs, inner type first.
func gcPointerOffsets(prgrm *ast.CXProgram, specs []int, typ types.Code, strct *ast.CXStruct, lengths []types.Pointer, base types.Pointer, offsets []gcPointerSlot, depth int) []gcPointerSlot {
	if depth > gcPointerDepthLimit || len(specs) == 0 || len(offsets) >= gcPointerSlotLimit {
		return offsets
	}
	switch specs[len(specs)-1] {
	case constants.DECL_POINTER:
		slot := gcPointerSlot{gcPointerSlotOffset: base, gcPointerSlotType: types.POINTER, gcPointerSlotSpecs: []int{constants.DECL_BASIC, constants.DECL_POINTER}}
		if len(specs) > 1 && specs[len(specs)-2] == constants.DECL_STRUCT {
			slot.gcPointerSlotStruct = strct
		}
		return append(offsets, slot)
	case constants.DECL_SLICE:
		return append(offsets, gcPointerSlot{gcPointerSlotOffset: base, gcPointerSlotType: typ, gcPointerSlotSpecs: specs})
	case constants.DECL_ARRAY:
		if len(lengths) == 0 {
			return offsets
		}
		element := specs[:len(specs)-1]
		size, found := gcValueSize(prgrm, element, typ, strct, lengths[1:])
		if !found {
			return offsets
		}
		for i := types.Pointer(0); i < lengths[0]; i++ {
			offsets = gcPointerOffsets(prgrm, element, typ, strct, lengths[1:], base+i*size, offsets, depth+1)
		}
		return offsets
	}
	if strct == nil {
		if typ == types.STR {
			return append(offsets, gcPointerSlot{gcPointerSlotOffset: base, gcPointerSlotType: types.STR, gcPointerSlotSpecs: []int{constants.DECL_BASIC}})
		}
		return offsets
	}
	for fieldIndex, fieldIdx := range strct.Fields {
		field := gcField(prgrm, fieldIdx)
		offsets = gcPointerOffsets(prgrm, field.DeclarationSpecifiers, field.Type, field.StructType, field.Lengths, base+ast.StructFieldOffset(prgrm, strct, fieldIndex), offsets, depth+1)
	}
	return offsets
}

// gcValueSize returns the size of a value that holds a pointer, and false
// for one that holds none.
func gcValueSize(prgrm *ast.CXProgram, specs []int, typ types.Code, strct *ast.CXStruct, lengths []types.Pointer) (types.Pointer, bool) {
	if !gcValueReferences(prgrm, specs, typ, strct, 0) {
		return 0, false
	}
	switch specs[len(specs)-1] {
	case constants.DECL_POINTER, constants.DECL_SLICE:
		return types.POINTER_SIZE, true
	case constants.DECL_ARRAY:
		size, found := gcValueSize(prgrm, specs[:len(specs)-1], typ, strct, lengths[1:])
		return size * lengths[0], found
	}
	if strct == nil {
		return types.STR_SIZE, true
	}
	return strct.GetStructSize(prgrm), true
}

// gcField returns the argument of a struct field.
func gcField(prgrm *ast.CXProgram, fieldIdx ast.CXTypeSignatureIndex) *ast.CXArgument {
	sig := prgrm.GetCXTypeSignatureFromArray(fieldIdx)
	if sig.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
		return prgrm.GetCXArg(ast.CXArgumentIndex(sig.Meta))
	}
	return sig.GetCXArgFormat(prgrm)
}

// gcPointerSlot is a pointer inside a variable, and the struct it points to.
type gcPointerSlot struct {
	gcPointerSlotOffset types.Pointer
	gcPointerSlotStruct *ast.CXStruct
	// gcPointerSlotType and gcPointerSlotSpecs type a slice or str slot for the GC.
	gcPointerSlotType  types.Code
	gcPointerSlotSpecs []int
}

// gcValueReferences reports whether a value of declaration specifiers specs
// holds a pointer, slice or str.
func gcValueReferences(prgrm *ast.CXProgram, specs []int, typ types.Code, strct *ast.CXStruct, depth int) bool {
	if depth > gcPointerDepthLimit || len(specs) == 0 {
		return false
	}
	switch specs[len(specs)-1] {
	case constants.DECL_POINTER, constants.DECL_SLICE:
		return true
	case constants.DECL_ARRAY:
		return gcValueReferences(prgrm, specs[:len(specs)-1], typ, strct, depth+1)
	}
	if strct == nil {
		return typ == types.STR
	}
	for _, fieldIdx := range strct.Fields {
		field := gcField(prgrm, fieldIdx)
		if gcValueReferences(prgrm, field.DeclarationSpecifiers, field.Type, field.StructType, depth+1) {
			return true
		}
	}
	return false
}
