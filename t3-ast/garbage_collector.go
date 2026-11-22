package ast

import (
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
)

// MarkAndCompact ...
func MarkAndCompact(prgrm *CXProgram) {
	gcObjectStartsBuild(prgrm)
	if prgrm.Heap.Visited == nil {
		prgrm.Heap.Visited = map[HeapVisit]bool{}
	}
	clear(prgrm.Heap.Visited)
	var fp types.Pointer
	var faddr = constants.NULL_HEAP_ADDRESS_OFFSET

	// marking, setting forward addresses and updating references
	// global variables
	for _, pkgIdx := range prgrm.Packages {
		// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
		// Now an arena getter, site S0335.
		pkg, err := prgrm.GetPackageFromArray(pkgIdx)
		if err != nil {
			panic(err)
		}

		for _, glblFldIdx := range pkg.Globals.Fields {
			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
			// Now an arena getter, site S0336.
			glblFld := prgrm.GetCXTypeSignatureFromArray(glblFldIdx)

			var glbl *CXArgument
			if glblFld.Type == TYPE_CXARGUMENT_DEPRECATE {
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S0338.
				glbl = prgrm.GetCXArg(CXArgumentIndex(glblFld.Meta))
			} else {
				panic("type is not cxargument deprecate.")
			}

			if gcGlobalPointer(glbl) {
				MarkObjectsTree(prgrm, glbl.Offset, glbl.Type, nil)
				continue
			}

			if (glbl.IsString() || glbl.IsSlice() || glbl.Type == types.STR) && glbl.StructType == nil {
				// Getting the offset to the object in the heap
				heapOffset := types.Read_ptr(prgrm.Memory, glbl.Offset)
				if heapOffset < prgrm.Heap.StartsAt {
					continue
				}

				glblType := glbl.Type
				if glbl.Type == types.POINTER {
					glblType = glbl.PointerTargetType
				}
				MarkObjectsTree(prgrm, glbl.Offset, glblType, glbl.DeclarationSpecifiers[1:])
			}

			// If `ptr` has fields, we need to navigate the heap and mark its fields too.
			if glbl.StructType != nil {
				for fieldIndex, typeSignatureIdx := range glbl.StructType.Fields {
					// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
					// Now an arena getter, site S0339.
					typeSignature := prgrm.GetCXTypeSignatureFromArray(typeSignatureIdx)

					var fld *CXArgument
					if typeSignature.Type == TYPE_CXARGUMENT_DEPRECATE {
						fldIdx := typeSignature.Meta
						// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
						// Now an arena getter, site S0341.
						fld = prgrm.GetCXArg(CXArgumentIndex(fldIdx))
					} else {
						fld = typeSignature.GetCXArgFormat(prgrm)
					}

					offset := glbl.Offset + StructFieldOffset(prgrm, glbl.StructType, fieldIndex)
					// Getting the offset to the object in the heap
					heapOffset := types.Read_ptr(prgrm.Memory, offset)
					if heapOffset < prgrm.Heap.StartsAt {
						continue
					}

					if fld.IsString() || fld.IsSlice() || fld.Type == types.STR {
						fldType := fld.Type
						if fld.Type == types.POINTER {
							fldType = fld.PointerTargetType
						}
						MarkObjectsTree(prgrm, offset, fldType, fld.DeclarationSpecifiers[1:])
					}
				}
			}
		}
	}

	// marking, setting forward addresses and updating references
	// local variables
	for c := types.Pointer(0); c <= prgrm.CallCounter; c++ {
		op := prgrm.CallStack[c].Operator

		// TODO: Some standard library functions "manually" add a function
		// call (callbacks) to `PRGRM.CallStack`. These functions do not have an
		// operator associated to them. This can be considered as a bug or as an
		// undesirable mechanic.
		// [2019-06-24 Mon 22:39] Actually, if the GC is triggered in the middle
		// of a callback, things will certainly break.
		if op == nil {
			continue
		}

		for _, ptr := range op.ListOfPointers {
			offset, found := gcFrameAddress(prgrm, fp, ptr.Offset)
			if !found {
				continue
			}

			ptrIsPointer := ptr.IsPointer()

			// Checking if we need to mark `ptr`.
			if ptrIsPointer {
				// If `ptr` has fields, we need to navigate the heap and mark its fields too.
				if ptr.StructType != nil {
					// Getting the offset to the object in the heap
					heapOffset := types.Read_ptr(prgrm.Memory, offset)
					if heapOffset >= prgrm.Heap.StartsAt {
						gcMarkStruct(prgrm, heapOffset+types.OBJECT_HEADER_SIZE, ptr.StructType)
					}
				}

				MarkObjectsTree(prgrm, offset, ptr.Type, ptr.DeclarationSpecifiers[1:])
			}

			// Checking if the field being accessed needs to be marked.
			// If the root (`ptr`) is a pointer, this step is unnecessary.
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0345.
			if len(ptr.Fields) == 1 && !ptrIsPointer && prgrm.GetCXArgFromArray(ptr.Fields[0]).IsPointer() {
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S0346.
				fld := prgrm.GetCXArgFromArray(ptr.Fields[len(ptr.Fields)-1])
				MarkObjectsTree(prgrm, offset+fld.Offset, fld.Type, fld.DeclarationSpecifiers[1:])
				if pointer := types.Read_ptr(prgrm.Memory, offset+fld.Offset); fld.StructType != nil && pointer > prgrm.Heap.StartsAt {
					gcMarkStruct(prgrm, pointer+types.OBJECT_HEADER_SIZE, fld.StructType)
				}
			}
		}

		for _, slot := range op.ListOfSlots {
			if base, found := gcFrameAddress(prgrm, fp, slot.CXFunctionSlotOffset); found {
				gcSlotMark(prgrm, base+slot.CXFunctionSlotInner, slot)
			}
		}

		fp += op.Size
	}

	// Relocation of live objects.
	for c := prgrm.Heap.StartsAt + constants.NULL_HEAP_ADDRESS_OFFSET; c < prgrm.Heap.StartsAt+prgrm.Heap.Pointer; {
		objSize := types.Read_obj_size(prgrm.Memory, c)

		if prgrm.Memory[c] == 1 {
			forwardingAddress := types.Read_obj_forwarding_address(prgrm.Memory, c)

			// We update the pointers that are pointing to the just moved object.
			updatePointers(prgrm, forwardingAddress, prgrm.Heap.StartsAt+faddr, objSize)

			// setting the mark back to 0
			prgrm.Memory[c] = 0
			// then it's alive and we'll relocate the object
			for i := types.Pointer(0); i < objSize; i++ {
				prgrm.Memory[faddr+prgrm.Heap.StartsAt+i] = prgrm.Memory[c+i]
			}

			faddr += objSize
		}

		c += objSize
	}

	prgrm.Heap.Pointer = faddr
}

// updateDisplaceReference performs the actual addition or subtraction of `plusOff` to the address being pointed by the element at `atOffset`.
func updateDisplaceReference(prgrm *CXProgram, updated *map[types.Pointer]types.Pointer, atOffset types.Pointer, plusOff types.Pointer) {
	// Checking if it was already updated.
	if _, found := (*updated)[atOffset]; found {
		return
	}

	// Extracting the address being pointed by element at `atOffset`
	dsCurrAddr := types.Read_ptr(prgrm.Memory, atOffset)

	// Adding `plusOff` to the address and updating the address pointed by
	// element at `atOffset`.
	types.Write_ptr(prgrm.Memory, atOffset, dsCurrAddr+plusOff)

	// Keeping a record of this address. We don't want to displace the object twice.
	// We're using a map to speed things up a tiny bit.
	(*updated)[atOffset] = atOffset
}

// doDisplaceReferences checks if the element at `atOffset` is pointing to an object on the heap and, if this is the case, it displaces it by `plusOff`. `updated` keeps a record of all the offsets that have already been updated.
func doDisplaceReferences(prgrm *CXProgram, updated *map[types.Pointer]types.Pointer, atOffset types.Pointer, plusOff types.Pointer, baseType types.Code, declSpecs []int) {
	var numDeclSpecs = len(declSpecs)

	// Getting the offset to the object in the heap.
	heapOffset := types.Read_ptr(prgrm.Memory, atOffset)

	// The whole displacement process is needed because the objects on the heap were
	// displaced by additional data segment bytes. These additional bytes need to be
	// considered if we want to read the objects on the heap. We need to check if the
	// displacement is positive or negative; if it is positive then this means a data
	// segment was added; if it is negative it means that we're done with any CX chains
	// process involving the addition of a data segment, and in some code snippets we can ignore the displacement.
	// TODO: Maybe this condition can be avoided by refactoring the code?
	var condPlusOff types.Pointer
	if plusOff > 0 {
		condPlusOff = plusOff
	}

	// Displace the address pointed by element at `atOffset`.
	updateDisplaceReference(prgrm, updated, atOffset, plusOff)

	// It can't be a tree of objects.
	if numDeclSpecs == 0 || heapOffset <= prgrm.Heap.StartsAt+condPlusOff {
		return
	}

	// Checking if it's a tree of objects.
	// TODO: We're not considering struct instances with pointer fields.
	if declSpecs[0] == constants.DECL_SLICE {
		if (numDeclSpecs > 1 &&
			(declSpecs[1] == constants.DECL_SLICE ||
				declSpecs[1] == constants.DECL_POINTER)) ||
			(numDeclSpecs == 1 && baseType == types.STR) {
			// Then we need to iterate each of the slice objects
			// and check if we need to update their address.
			sliceLen := GetSliceLen(prgrm, heapOffset+condPlusOff)

			offsetToElements := types.OBJECT_HEADER_SIZE + constants.SLICE_HEADER_SIZE

			for c := types.Pointer(0); c < sliceLen; c++ {
				cHeapOffset := types.Read_ptr(prgrm.Memory, heapOffset+condPlusOff+offsetToElements+c*types.POINTER_SIZE)
				if cHeapOffset <= prgrm.Heap.StartsAt+condPlusOff {
					// Then it's pointing to null or data segment
					continue
				}

				// Displacing this child element.
				updateDisplaceReference(prgrm, updated, heapOffset+offsetToElements+c*types.POINTER_SIZE, plusOff)
			}
		}
	}
}

// DisplaceReferences displaces all the pointer-like variables, slice elements or field structures by `off`. `numPkgs` tells us the number of packages to consider for the reference desplacement (this number should equal to the number of packages that represent the blockchain code in a CX chain).
func DisplaceReferences(prgrm *CXProgram, off types.Pointer, numPkgs int) {
	// We're going to keep a record of all the references that were already updated.
	updated := make(map[types.Pointer]types.Pointer)

	count := 0
	for _, pkgIdx := range prgrm.Packages {
		// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
		// Now an arena getter, site S0347.
		pkg, err := prgrm.GetPackageFromArray(pkgIdx)
		if err != nil {
			panic(err)
		}

		if count > numPkgs {
			break
		}

		// In a CX chain we're only interested on considering global variables,
		// as any other object should be destroyed, as the program finished its
		// execution.
		for _, glblFldIdx := range pkg.Globals.Fields {
			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
			// Now an arena getter, site S0348.
			glblFld := prgrm.GetCXTypeSignatureFromArray(glblFldIdx)

			var glbl *CXArgument
			if glblFld.Type == TYPE_CXARGUMENT_DEPRECATE {
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S0350.
				glbl = prgrm.GetCXArg(CXArgumentIndex(glblFld.Meta))
			} else {
				panic("type is not cxargument deprecate.")
			}

			if glbl.IsString() || glbl.IsStruct() || glbl.IsSlice() {
				doDisplaceReferences(prgrm, &updated, glbl.Offset, off, glbl.Type, glbl.DeclarationSpecifiers[1:]) // TODO:PTR remove hardcoded offsets
			}

			// If it's a struct instance we need to displace each of its fields.
			if glbl.StructType != nil {
				for fieldIndex, typeSignatureIdx := range glbl.StructType.Fields {
					// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
					// Now an arena getter, site S0351.
					typeSignature := prgrm.GetCXTypeSignatureFromArray(typeSignatureIdx)

					var fld *CXArgument
					if typeSignature.Type == TYPE_CXARGUMENT_DEPRECATE {
						fldIdx := typeSignature.Meta
						// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
						// Now an arena getter, site S0353.
						fld = prgrm.GetCXArg(CXArgumentIndex(fldIdx))
					} else {
						fld = typeSignature.GetCXArgFormat(prgrm)
					}

					if fld.IsString() || fld.IsStruct() || fld.IsSlice() {
						doDisplaceReferences(prgrm, &updated, glbl.Offset+StructFieldOffset(prgrm, glbl.StructType, fieldIndex), off, fld.Type, fld.DeclarationSpecifiers[1:])
					}
				}
			}
		}

		count++
	}
}

// Mark marks the object located at `heapOffset` as alive and sets the object's referencing address to `heapOffset`.
func Mark(prgrm *CXProgram, heapOffset types.Pointer) {
	heapOffset, found := gcObjectContaining(prgrm, heapOffset)
	if !found {
		return
	}
	// Marking as alive.
	types.Write_obj_mark(prgrm.Memory, heapOffset, 1)

	// Setting forwarding address. This address is used to know where the object used to live on the heap. With it we can know what symbols were pointing to that dead object and then update their address.
	types.Write_obj_forwarding_address(prgrm.Memory, heapOffset, heapOffset)
}

// MarkObjectsTree traverses and marks a possible tree of heap objects (slices of slices, slices of pointers, etc.).
func MarkObjectsTree(prgrm *CXProgram, offset types.Pointer, baseType types.Code, declSpecs []int) {
	lenMem := types.Cast_int_to_ptr(len(prgrm.Memory))
	// Checking if it's a valid heap address. An invalid address
	// usually occurs in CX chains, with the split of blockchain
	// and transaction codes in a CX chain program state.
	if offset > lenMem || offset+types.POINTER_SIZE > lenMem {
		return
	}

	// Getting the offset to the object in the heap
	heapOffset := types.Read_ptr(prgrm.Memory, offset)

	// Then it's a pointer to an object in the stack and it should not be marked.
	if heapOffset <= prgrm.Heap.StartsAt {
		return
	}

	// marking the root object
	Mark(prgrm, heapOffset)

	var numDeclSpecs = len(declSpecs)
	if numDeclSpecs == 0 {
		return
	}

	// Then it's a tree of objects.
	// TODO: We're not considering struct instances with pointer fields.
	if declSpecs[0] == constants.DECL_SLICE {
		if (numDeclSpecs > 1 &&
			(declSpecs[1] == constants.DECL_SLICE ||
				declSpecs[1] == constants.DECL_POINTER)) ||
			(numDeclSpecs == 1 && baseType == types.STR) {
			// Then we need to iterate each of the slice objects and mark them as alive
			sliceLen := GetSliceLen(prgrm, heapOffset)

			for c := types.Pointer(0); c < sliceLen; c++ {
				offsetToElements := types.OBJECT_HEADER_SIZE + constants.SLICE_HEADER_SIZE
				cHeapOffset := types.Read_ptr(prgrm.Memory, heapOffset+offsetToElements+c*types.POINTER_SIZE)
				if cHeapOffset <= prgrm.Heap.StartsAt {
					// Then it's pointing to null or data segment
					continue
				}

				MarkObjectsTree(prgrm, heapOffset+offsetToElements+c*types.POINTER_SIZE, baseType, declSpecs[1:])
			}
		}
	}
}

// updatePointer changes the address of the pointer located at `atOffset` to `newAddress`.
func updatePointer(prgrm *CXProgram, atOffset types.Pointer, toAddress types.Pointer) {
	// Two roots can name one slot, and a pointer inside a moved object could
	// still be inside it after one update.
	for _, updated := range prgrm.Heap.UpdatedSlots {
		if updated == atOffset {
			return
		}
	}
	prgrm.Heap.UpdatedSlots = append(prgrm.Heap.UpdatedSlots, atOffset)
	types.Write_ptr(prgrm.Memory, atOffset, toAddress)
}

// updatePointerTree changes the address of the pointer located at `atOffset` to `newAddress` and checks if it is the
// root of a tree of objects, such as a slice or the instance of a struct where some of its fields are pointers.
func updatePointerTree(prgrm *CXProgram, atOffset types.Pointer, oldAddr, newAddr, oldSize types.Pointer, baseType types.Code, declSpecs []int) {
	var numDeclSpecs = len(declSpecs)

	// Getting the offset to the object in the heap
	heapOffset := types.Read_ptr(prgrm.Memory, atOffset)

	if heapOffset >= oldAddr && heapOffset < oldAddr+oldSize {
		// Updating the root pointer, which may point inside the object.
		updatePointer(prgrm, atOffset, heapOffset-oldAddr+newAddr)
	}

	// It can't be a tree of objects.
	if numDeclSpecs == 0 || heapOffset <= prgrm.Heap.StartsAt {
		return
	}

	// Checking if it's a tree of objects.
	// TODO: We're not considering struct instances with pointer fields.
	if declSpecs[0] == constants.DECL_SLICE {
		if (numDeclSpecs > 1 &&
			(declSpecs[1] == constants.DECL_SLICE ||
				declSpecs[1] == constants.DECL_POINTER)) ||
			(numDeclSpecs == 1 && baseType == types.STR) {
			// Then we need to iterate each of the slice objects
			// and check if we need to update their address.
			sliceLen := GetSliceLen(prgrm, heapOffset)
			offsetToElements := types.OBJECT_HEADER_SIZE + constants.SLICE_HEADER_SIZE

			for c := types.Pointer(0); c < sliceLen; c++ {
				cHeapOffset := types.Read_ptr(prgrm.Memory, heapOffset+offsetToElements+c*types.POINTER_SIZE)
				if cHeapOffset <= prgrm.Heap.StartsAt {
					// Then it's pointing to null or data segment
					continue
				}

				// Then it's not pointing to the object moved by the GC or it's pointing to
				// an object in the stack segment or nil.
				if cHeapOffset >= oldAddr && cHeapOffset < oldAddr+oldSize {
					updatePointerTree(prgrm, heapOffset+offsetToElements+c*types.POINTER_SIZE, oldAddr, newAddr, oldSize, baseType, declSpecs[1:]) // TODO:PTR remove hardcode
				}
			}
		}
	}
}

// updatePointers updates all the references to objects on the heap to their new addresses after calling the garbage collector.
// For example, if `foo` was pointing to an object located at address 5151 and after calling the garbage collector it was
// moved to address 4141, every symbol in a `CXProgram`'s `CallStack` and in its global variables need to be updated to point now to
// 4141 instead of 5151.
func updatePointers(prgrm *CXProgram, oldAddr, newAddr, oldSize types.Pointer) {
	if oldAddr == newAddr {
		return
	}
	prgrm.Heap.UpdatedSlots = prgrm.Heap.UpdatedSlots[:0]
	clear(prgrm.Heap.Visited)
	// TODO: `oldAddr` could be received as a slice of bytes that represent the old address of the object,
	// as it needs to be converted to bytes later on anyways. However, I'm sticking to an int32
	// for a bit more of clarity.
	for _, pkgIdx := range prgrm.Packages {
		// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
		// Now an arena getter, site S0354.
		pkg, err := prgrm.GetPackageFromArray(pkgIdx)
		if err != nil {
			panic(err)
		}

		for _, glblFldIdx := range pkg.Globals.Fields {
			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
			// Now an arena getter, site S0355.
			glblFld := prgrm.GetCXTypeSignatureFromArray(glblFldIdx)

			var glbl *CXArgument
			if glblFld.Type == TYPE_CXARGUMENT_DEPRECATE {
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S0357.
				glbl = prgrm.GetCXArg(CXArgumentIndex(glblFld.Meta))
			} else {
				panic("type is not cxargument deprecate.")
			}

			if gcGlobalPointer(glbl) {
				updatePointerTree(prgrm, glbl.Offset, oldAddr, newAddr, oldSize, glbl.Type, nil)
				continue
			}

			if (glbl.IsPointer() || glbl.IsSlice() || glbl.Type == types.STR) && glbl.StructType == nil {
				// Getting the offset to the object in the heap
				heapOffset := types.Read_ptr(prgrm.Memory, glbl.Offset)
				if heapOffset < prgrm.Heap.StartsAt {
					continue
				}

				glblType := glbl.Type
				if glbl.Type == types.POINTER {
					glblType = glbl.PointerTargetType
				}
				updatePointerTree(prgrm, glbl.Offset, oldAddr, newAddr, oldSize, glblType, glbl.DeclarationSpecifiers[1:]) // TODO:PTR remove hardcoded 1 value
			}

			// If `ptr` has fields, we need to navigate the heap and mark its fields too.
			if glbl.StructType != nil {
				for fieldIndex, typeSignatureIdx := range glbl.StructType.Fields {
					// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
					// Now an arena getter, site S0358.
					typeSignature := prgrm.GetCXTypeSignatureFromArray(typeSignatureIdx)

					var fld *CXArgument
					if typeSignature.Type == TYPE_CXARGUMENT_DEPRECATE {
						fldIdx := typeSignature.Meta
						// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
						// Now an arena getter, site S0360.
						fld = prgrm.GetCXArg(CXArgumentIndex(fldIdx))
					} else {
						fld = typeSignature.GetCXArgFormat(prgrm)
					}

					if !fld.IsPointer() {
						continue
					}
					offset := glbl.Offset + StructFieldOffset(prgrm, glbl.StructType, fieldIndex)

					// Getting the offset to the object in the heap
					heapOffset := types.Read_ptr(prgrm.Memory, offset)
					if heapOffset < prgrm.Heap.StartsAt {
						continue
					}

					if fld.IsPointer() || fld.IsSlice() || fld.Type == types.STR {
						fldType := fld.Type
						if fld.Type == types.POINTER {
							fldType = fld.PointerTargetType
						}
						updatePointerTree(prgrm, offset, oldAddr, newAddr, oldSize, fldType, fld.DeclarationSpecifiers[1:])
					}
				}
			}
		}
	}

	var fp types.Pointer
	for c := types.Pointer(0); c <= prgrm.CallCounter; c++ {
		op := prgrm.CallStack[c].Operator

		// TODO: Some standard library functions "manually" add a function
		// call (callbacks) to `PRGRM.CallStack`. These functions do not have an
		// operator associated to them. This can be considered as a bug or as an
		// undesirable mechanic.
		// [2019-06-24 Mon 22:39] Actually, if the GC is triggered in the middle
		// of a callback, things will certainly break.
		if op == nil {
			continue
		}

		for _, ptr := range op.ListOfPointers {
			offset, found := gcFrameAddress(prgrm, fp, ptr.Offset)
			if !found {
				continue
			}

			ptrIsPointer := ptr.IsPointer()

			// Checking if we need to mark `ptr`.
			if ptrIsPointer {
				// Getting the offset to the object in the heap
				heapOffset := types.Read_ptr(prgrm.Memory, offset)
				if heapOffset > prgrm.Heap.StartsAt {
					updatePointerTree(prgrm, offset, oldAddr, newAddr, oldSize, ptr.Type, ptr.DeclarationSpecifiers[1:])

					// If `ptr` has fields, we need to navigate the heap and mark its fields too.
					if ptr.StructType != nil {
						if heapOffset >= prgrm.Heap.StartsAt {
							gcUpdateStruct(prgrm, heapOffset+types.OBJECT_HEADER_SIZE, ptr.StructType, oldAddr, newAddr, oldSize)
						}
					}
				}
			}

			// Checking if the field being accessed needs to be marked.
			// If the root (`ptr`) is a pointer, this step is unnecessary.
			// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
			// Now an arena getter, site S0364.
			if len(ptr.Fields) == 1 && !ptrIsPointer && prgrm.GetCXArgFromArray(ptr.Fields[0]).IsPointer() {
				// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
				// Now an arena getter, site S0365.
				fld := prgrm.GetCXArgFromArray(ptr.Fields[len(ptr.Fields)-1])

				// Getting the offset to the object in the heap
				heapOffset := types.Read_ptr(prgrm.Memory, offset+fld.Offset)
				if heapOffset > prgrm.Heap.StartsAt {
					updatePointerTree(prgrm, offset+fld.Offset, oldAddr, newAddr, oldSize, fld.Type, fld.DeclarationSpecifiers[1:])
					if fld.StructType != nil {
						gcUpdateStruct(prgrm, heapOffset+types.OBJECT_HEADER_SIZE, fld.StructType, oldAddr, newAddr, oldSize)
					}
				}
			}

		}

		for _, slot := range op.ListOfSlots {
			if base, found := gcFrameAddress(prgrm, fp, slot.CXFunctionSlotOffset); found {
				gcSlotUpdate(prgrm, base+slot.CXFunctionSlotInner, slot, oldAddr, newAddr, oldSize)
			}
		}

		fp += op.Size
	}
}

// gcFrameAddress returns the address of the frame variable at offset. It
// returns false for an escaped local whose declaration has not run yet.
func gcFrameAddress(prgrm *CXProgram, fp types.Pointer, offset types.Pointer) (types.Pointer, bool) {
	if !OffsetEscaped(offset) {
		return offset + fp, true
	}
	object := types.Read_ptr(prgrm.Memory, fp+OffsetEscapedSlot(offset))
	if object < prgrm.Heap.StartsAt+constants.NULL_HEAP_ADDRESS_OFFSET || object >= prgrm.Heap.StartsAt+prgrm.Heap.Pointer {
		return 0, false
	}
	return object + types.OBJECT_HEADER_SIZE, true
}

// gcGlobalPointer reports whether glbl is a pointer to a value that is
// neither a str nor a slice, such as the address of an escaped local.
func gcGlobalPointer(glbl *CXArgument) bool {
	return glbl.IsPointer() && !glbl.IsString() && !glbl.IsSlice()
}

// gcObjectStartsBuild lists the address of each heap object in
// prgrm.Heap.ObjectStarts.
func gcObjectStartsBuild(prgrm *CXProgram) {
	starts := prgrm.Heap.ObjectStarts[:0]
	for c := prgrm.Heap.StartsAt + constants.NULL_HEAP_ADDRESS_OFFSET; c < prgrm.Heap.StartsAt+prgrm.Heap.Pointer; {
		starts = append(starts, c)
		objSize := types.Read_obj_size(prgrm.Memory, c)
		if objSize == 0 {
			break
		}
		c += objSize
	}
	prgrm.Heap.ObjectStarts = starts
}

// gcObjectContaining returns the address of the heap object that pointer
// points into. A pointer inside an object, as &s[1] is, keeps it alive.
func gcObjectContaining(prgrm *CXProgram, pointer types.Pointer) (types.Pointer, bool) {
	starts := prgrm.Heap.ObjectStarts
	low, high := 0, len(starts)
	for low < high {
		middle := int(uint(low+high) >> 1)
		if starts[middle] <= pointer {
			low = middle + 1
		} else {
			high = middle
		}
	}
	if low == 0 {
		return 0, false
	}
	start := starts[low-1]
	if pointer >= start+types.Read_obj_size(prgrm.Memory, start) {
		return 0, false
	}
	return start, true
}

// gcVisit reports whether the struct value strct at address was followed
// already, and records it.
func gcVisit(prgrm *CXProgram, address types.Pointer, strct *CXStruct) bool {
	key := HeapVisit{HeapVisitAddress: address, HeapVisitStruct: strct}
	if prgrm.Heap.Visited[key] {
		return true
	}
	prgrm.Heap.Visited[key] = true
	return false
}

// gcStructField returns the argument of a struct field.
func gcStructField(prgrm *CXProgram, fieldIdx CXTypeSignatureIndex) *CXArgument {
	sig := prgrm.GetCXTypeSignatureFromArray(fieldIdx)
	if sig.Type == TYPE_CXARGUMENT_DEPRECATE {
		return prgrm.GetCXArg(CXArgumentIndex(sig.Meta))
	}
	return sig.GetCXArgFormat(prgrm)
}

// gcMarkStruct marks what the pointers, slices and strings of the struct
// value strct at address point to, and follows pointers to structs.
func gcMarkStruct(prgrm *CXProgram, address types.Pointer, strct *CXStruct) {
	if strct == nil || gcVisit(prgrm, address, strct) {
		return
	}
	for fieldIndex, fieldIdx := range strct.Fields {
		fld := gcStructField(prgrm, fieldIdx)
		slot := address + StructFieldOffset(prgrm, strct, fieldIndex)
		switch {
		case cxArgumentPointerField(fld):
			pointer := types.Read_ptr(prgrm.Memory, slot)
			if pointer <= prgrm.Heap.StartsAt {
				continue
			}
			Mark(prgrm, pointer)
			gcMarkStruct(prgrm, pointer+types.OBJECT_HEADER_SIZE, fld.StructType)
		case fld.IsSlice() || fld.Type == types.STR:
			MarkObjectsTree(prgrm, slot, fld.Type, fld.DeclarationSpecifiers[1:])
		case fld.StructType != nil && len(fld.Lengths) == 0:
			gcMarkStruct(prgrm, slot, fld.StructType)
		}
	}
}

// gcUpdateStruct is updatePointers for the struct value strct at address. It
// follows each pointer from where it pointed before the update.
func gcUpdateStruct(prgrm *CXProgram, address types.Pointer, strct *CXStruct, oldAddr, newAddr, oldSize types.Pointer) {
	if strct == nil || gcVisit(prgrm, address, strct) {
		return
	}
	for fieldIndex, fieldIdx := range strct.Fields {
		fld := gcStructField(prgrm, fieldIdx)
		slot := address + StructFieldOffset(prgrm, strct, fieldIndex)
		switch {
		case cxArgumentPointerField(fld):
			pointer := types.Read_ptr(prgrm.Memory, slot)
			if pointer <= prgrm.Heap.StartsAt {
				continue
			}
			updatePointerTree(prgrm, slot, oldAddr, newAddr, oldSize, fld.Type, nil)
			gcUpdateStruct(prgrm, pointer+types.OBJECT_HEADER_SIZE, fld.StructType, oldAddr, newAddr, oldSize)
		case fld.IsSlice() || fld.Type == types.STR:
			updatePointerTree(prgrm, slot, oldAddr, newAddr, oldSize, fld.Type, fld.DeclarationSpecifiers[1:])
		case fld.StructType != nil && len(fld.Lengths) == 0:
			gcUpdateStruct(prgrm, slot, fld.StructType, oldAddr, newAddr, oldSize)
		}
	}
}

// gcSlotMark marks what the slot at address points to.
func gcSlotMark(prgrm *CXProgram, address types.Pointer, slot CXFunctionSlot) {
	MarkObjectsTree(prgrm, address, slot.CXFunctionSlotType, slot.CXFunctionSlotSpecifiers[1:])
	if pointer := types.Read_ptr(prgrm.Memory, address); slot.CXFunctionSlotStruct != nil && pointer > prgrm.Heap.StartsAt {
		gcMarkStruct(prgrm, pointer+types.OBJECT_HEADER_SIZE, slot.CXFunctionSlotStruct)
	}
}

// gcSlotUpdate is updatePointers for the slot at address.
func gcSlotUpdate(prgrm *CXProgram, address types.Pointer, slot CXFunctionSlot, oldAddr, newAddr, oldSize types.Pointer) {
	pointer := types.Read_ptr(prgrm.Memory, address)
	if pointer <= prgrm.Heap.StartsAt {
		return
	}
	updatePointerTree(prgrm, address, oldAddr, newAddr, oldSize, slot.CXFunctionSlotType, slot.CXFunctionSlotSpecifiers[1:])
	gcUpdateStruct(prgrm, pointer+types.OBJECT_HEADER_SIZE, slot.CXFunctionSlotStruct, oldAddr, newAddr, oldSize)
}
