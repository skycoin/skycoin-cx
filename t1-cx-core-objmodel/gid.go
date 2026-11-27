// Package objmodel holds the generational index of docs/ObjectModel.md. A Gid
// is an int64 with the slot index in bits 0 to 31, the generation in bits 32
// to 55 and the type tag in bits 56 to 63. Every check is a class B assert.
package objmodel

import (
	"fmt"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
)

// TypeTag names the object type of a Gid, 0x01 to 0x7F.
type TypeTag uint8

const (
	GidIndexBits      = 32
	GidGenerationBits = 24
	GidTypeTagBits    = 8

	GidIndexMask      uint64 = 0x0000_0000_FFFF_FFFF
	GidHighMask       uint64 = 0xFFFF_FFFF_0000_0000
	GidGenerationMask uint64 = 0x00FF_FFFF_0000_0000
	GidTypeTagMask    uint64 = 0xFF00_0000_0000_0000

	GidGenerationShift = 32
	GidTypeTagShift    = 56

	GidGenerationFirst uint32 = 1
	GidGenerationLast  uint32 = 0x00FF_FFFF

	GidTypeTagFirst TypeTag = 0x01
	GidTypeTagLast  TypeTag = 0x7F

	GidNil int64 = 0

	// GidCapMax is the largest Cap of a manager, so a slot count fits in uint32.
	GidCapMax uint64 = GidIndexMask
)

// GidCompose builds a Gid. It asserts the tag, generation and index ranges.
func GidCompose(typeTag TypeTag, generation uint32, index uint32) int64 {
	if typeTag < GidTypeTagFirst || typeTag > GidTypeTagLast || generation < GidGenerationFirst || generation >= GidGenerationLast {
		GidComposeFailureReport(typeTag, generation)
	}
	return int64(uint64(typeTag)<<GidTypeTagShift | uint64(generation)<<GidGenerationShift | uint64(index))
}

// GidComposeFailureReport asserts the failed range check of GidCompose.
// STRHOT REPORTER: builds the assert text from the integers, #412.
//
//go:noinline
func GidComposeFailureReport(typeTag TypeTag, generation uint32) {
	if typeTag < GidTypeTagFirst || typeTag > GidTypeTagLast {
		assert.PreconditionTrue(false, fmt.Sprintf("Gid type tag %#x outside %#x to %#x", typeTag, GidTypeTagFirst, GidTypeTagLast))
	}
	assert.PreconditionTrue(false, fmt.Sprintf("Gid generation %#x outside %#x to %#x", generation, GidGenerationFirst, GidGenerationLast-1))
}

// GidIndex returns bits 0 to 31.
func GidIndex(gid int64) uint32 {
	return uint32(uint64(gid) & GidIndexMask)
}

// GidGeneration returns bits 32 to 55, the type tag masked out.
func GidGeneration(gid int64) uint32 {
	return uint32((uint64(gid) & GidGenerationMask) >> GidGenerationShift)
}

// GidTypeTag returns bits 56 to 63.
func GidTypeTag(gid int64) TypeTag {
	return TypeTag((uint64(gid) & GidTypeTagMask) >> GidTypeTagShift)
}

// GidAssertFailure is the cold path of a manager's Release and of the Gid
// reporters. It asserts
// the first failed check of gid and returns when gid is positive, carries
// typeTag and indexes a slot below capacity, so the manager then reports a
// slot that is not live or a stale generation.
//
//go:noinline
func GidAssertFailure(gid int64, typeTag TypeTag, capacity int) {
	if gid <= GidNil {
		assert.PreconditionTrue(false, fmt.Sprintf("%s Gid %#x is nil or negative", GidTypeTagNameTable[typeTag], gid))
	}
	if GidTypeTag(gid) != typeTag {
		assert.PreconditionTrue(false,
			fmt.Sprintf("%s Gid %#x has type tag %#x, want %#x", GidTypeTagNameTable[typeTag], gid, GidTypeTag(gid), typeTag))
	}
	if uint64(GidIndex(gid)) >= uint64(capacity) {
		assert.PreconditionTrue(false,
			fmt.Sprintf("%s Gid %#x has index %d, Cap is %d", GidTypeTagNameTable[typeTag], gid, GidIndex(gid), capacity))
	}
}

// GidGetFailureReport asserts the failed check of a generated manager's Get
// of gid, given the Cap and the Live and Generations slices of the manager.
// STRHOT REPORTER: builds the assert text from the integers, #412.
//
//go:noinline
func GidGetFailureReport(gid int64, typeTag TypeTag, capacity int, live []bool, generations []uint32) {
	GidAssertFailure(gid, typeTag, capacity)
	index := GidIndex(gid)
	if !live[index] {
		assert.PreconditionTrue(false, fmt.Sprintf("%sGet of slot %d, which is not live, a stale Gid %#x", GidTypeTagNameTable[typeTag], index, gid))
	}
	assert.PreconditionTrue(false, fmt.Sprintf("%sGet of a stale Gid %#x, slot %d is at generation %d", GidTypeTagNameTable[typeTag], gid, index, generations[index]))
}

// GidMngInitFailureReport asserts the failed check of a generated
// manager's MngInit, given its Len and the capacity asked for.
// STRHOT REPORTER: builds the assert text from the integers, #412.
//
//go:noinline
func GidMngInitFailureReport(typeTag TypeTag, length int, capacity int) {
	if length != 0 {
		assert.PreconditionTrue(false, fmt.Sprintf("%sMngInit on a manager holding %d objects", GidTypeTagNameTable[typeTag], length))
	}
	assert.PreconditionTrue(false, fmt.Sprintf("%sMngInit capacity %d outside 0 to %d", GidTypeTagNameTable[typeTag], capacity, GidCapMax))
}

// GidFirstGenerationFailureReport asserts the failed check of a Gid that
// must carry typeTag, GidGenerationFirst and an index below capacity, as
// the Gids of the string pool and the source files do.
// STRHOT REPORTER: builds the assert text from the integers, #412.
//
//go:noinline
func GidFirstGenerationFailureReport(gid int64, typeTag TypeTag, capacity int) {
	GidAssertFailure(gid, typeTag, capacity)
	assert.PreconditionTrue(false, fmt.Sprintf("%s Gid %#x has generation %#x, want %#x", GidTypeTagNameTable[typeTag], gid, GidGeneration(gid), GidGenerationFirst))
}
