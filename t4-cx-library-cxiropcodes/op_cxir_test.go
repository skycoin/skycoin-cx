package cxiropcodes

import (
	"testing"

	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
	reflecttable "github.com/skycoin/skycoin-cx/t2-cx-library-reflecttable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// TestTagsCollision checks the tags of sys/cxir against their ranges of
// gid-variants D154 and every other tag in use.
func TestTagsCollision(t *testing.T) {
	if ir.ImageTag < 0x40 || ir.ImageTag > 0x5F {
		t.Errorf("image tag %#x is outside the reflect, meta and cxir range 0x40 to 0x5F", ir.ImageTag)
	}
	if ir.ImageNameTag < 0x01 || ir.ImageNameTag > 0x1F {
		t.Errorf("name tag %#x is outside the pool range 0x01 to 0x1F", ir.ImageNameTag)
	}
	used := map[objmodel.TypeTag]string{}
	for tag, name := range objmodel.GidTypeTagNameTable {
		if name != "" {
			used[objmodel.TypeTag(tag)] = name
		}
	}
	for tag, name := range map[objmodel.TypeTag]string{
		reflecttable.TypeTag: "reflect type", reflecttable.FunctionTag: "reflect function", reflecttable.TypeNameTag: "reflect type name",
		reflecttable.PackageTag: "meta package",
	} {
		if other, taken := used[tag]; taken {
			t.Fatalf("tag %#x of %s is taken by %s", tag, name, other)
		}
		used[tag] = name
	}
	for _, tag := range []objmodel.TypeTag{ir.ImageTag, ir.ImageNameTag} {
		if other, taken := used[tag]; taken {
			t.Errorf("cxir tag %#x is taken by %s", tag, other)
		}
		used[tag] = "cxir"
	}
}

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
