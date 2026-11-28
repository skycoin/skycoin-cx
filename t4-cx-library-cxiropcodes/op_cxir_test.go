package cxiropcodes

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// TestStructSizes checks that the structs sys/cxir declares have the sizes
// of the records the store reads and writes.
func TestStructSizes(t *testing.T) {
	sizes := map[string]int{"CxInstruction": ir.ImageInstructionSize, "CxFrameSlot": ir.ImageSlotSize, "CxFrameLocal": ir.ImageLocalSize, "CxSourceLine": ir.ImageLineSize}
	for _, declaration := range structDeclarations {
		size := 0
		for _, field := range declaration.structDeclarationFields {
			size += int(field.structFieldType.Size())
		}
		if size != sizes[declaration.structDeclarationName] {
			t.Errorf("%s is %d bytes, the store uses %d", declaration.structDeclarationName, size, sizes[declaration.structDeclarationName])
		}
	}
}
