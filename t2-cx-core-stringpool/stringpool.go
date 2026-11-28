// Package stringpool interns the names of the compiler IR, see #413 and
// docs/ObjectModel.md. A StringGid names one interned byte string, and equal
// strings have equal Gids, so comparing two names is comparing two integers.
package stringpool

import (
	"bytes"
	"fmt"
	"unsafe"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
	gidtag "github.com/skycoin/skycoin-cx/t0-cx-core-gidtag"
	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
)

// StringGid names one interned string. It has the Gid layout of
// docs/ObjectModel.md. Every slot keeps GidGenerationFirst, since an interned
// string is never released. StringGidNil is the empty string.
type StringGid int64

const (
	StringGidTypeTag objmodel.TypeTag = gidtag.String
	StringGidNil     StringGid        = 0
)

// StringPool holds every interned string in one byte arena, with the offset
// and length of each slot. No Go string is stored. StringPoolBuckets is an
// open addressed hash table of slot+1 by the FNV-1a hash of the bytes, 0 for
// an empty bucket, with a power of two length, see D139 on #412.
type StringPool struct {
	StringPoolArena   []byte
	StringPoolOffsets []uint32
	StringPoolLengths []uint32
	StringPoolBuckets []uint32
	StringPoolFrozen  bool
}

// stringPoolBucketsMinimum is the smallest hash table, in buckets.
const stringPoolBucketsMinimum = 16

// StringPoolInit sets up an empty pool with room for slotCap strings and
// arenaCap bytes before the first growth.
func (pool *StringPool) StringPoolInit(slotCap int, arenaCap int) {
	pool.StringPoolArena = make([]byte, 0, arenaCap)
	pool.StringPoolOffsets = make([]uint32, 0, slotCap)
	pool.StringPoolLengths = make([]uint32, 0, slotCap)
	buckets := stringPoolBucketsMinimum
	for buckets < 2*slotCap {
		buckets *= 2
	}
	pool.StringPoolBuckets = make([]uint32, buckets)
	pool.StringPoolFrozen = false
}

// StringPoolIntern returns the Gid of bytes, interning them on first use.
// The empty string is StringGidNil. It asserts on a frozen pool.
func (pool *StringPool) StringPoolIntern(bytes []byte) StringGid {
	if pool.StringPoolFrozen {
		stringPoolInternFrozenReport(bytes)
	}
	if len(bytes) == 0 {
		return StringGidNil
	}
	if len(pool.StringPoolBuckets) == 0 {
		pool.StringPoolBuckets = make([]uint32, stringPoolBucketsMinimum)
	}
	bucket, found := pool.stringPoolBucketFind(bytes)
	if found {
		return stringPoolGidOf(pool.StringPoolBuckets[bucket] - 1)
	}
	slot := len(pool.StringPoolOffsets)
	if uint64(slot) >= objmodel.GidCapMax || uint64(len(pool.StringPoolArena))+uint64(len(bytes)) > uint64(^uint32(0)) {
		stringPoolFullReport(slot, len(pool.StringPoolArena))
	}
	pool.StringPoolOffsets = append(pool.StringPoolOffsets, uint32(len(pool.StringPoolArena)))
	pool.StringPoolLengths = append(pool.StringPoolLengths, uint32(len(bytes)))
	pool.StringPoolArena = append(pool.StringPoolArena, bytes...)
	pool.StringPoolBuckets[bucket] = uint32(slot) + 1
	if 2*(slot+1) > len(pool.StringPoolBuckets) {
		pool.stringPoolBucketsGrow()
	}
	return stringPoolGidOf(uint32(slot))
}

// StringPoolFind returns the Gid of text without interning it, and false
// when it was never interned.
func (pool *StringPool) StringPoolFind(text []byte) (StringGid, bool) {
	if len(text) == 0 {
		return StringGidNil, true
	}
	if len(pool.StringPoolBuckets) == 0 {
		return StringGidNil, false
	}
	bucket, found := pool.stringPoolBucketFind(text)
	if !found {
		return StringGidNil, false
	}
	return stringPoolGidOf(pool.StringPoolBuckets[bucket] - 1), true
}

// stringPoolBucketFind returns the bucket that holds text, or the empty
// bucket where it goes and false.
func (pool *StringPool) stringPoolBucketFind(text []byte) (int, bool) {
	mask := uint32(len(pool.StringPoolBuckets) - 1)
	for bucket := stringPoolHash(text) & mask; ; bucket = (bucket + 1) & mask {
		entry := pool.StringPoolBuckets[bucket]
		if entry == 0 {
			return int(bucket), false
		}
		if bytes.Equal(pool.stringPoolSlotBytes(entry-1), text) {
			return int(bucket), true
		}
	}
}

// stringPoolBucketsGrow doubles the hash table and places every slot again.
func (pool *StringPool) stringPoolBucketsGrow() {
	pool.StringPoolBuckets = make([]uint32, 2*len(pool.StringPoolBuckets))
	mask := uint32(len(pool.StringPoolBuckets) - 1)
	for slot := range pool.StringPoolOffsets {
		bucket := stringPoolHash(pool.stringPoolSlotBytes(uint32(slot))) & mask
		for pool.StringPoolBuckets[bucket] != 0 {
			bucket = (bucket + 1) & mask
		}
		pool.StringPoolBuckets[bucket] = uint32(slot) + 1
	}
}

// stringPoolHash is the 32 bit FNV-1a hash of text,
// see https://datatracker.ietf.org/doc/html/draft-eastlake-fnv section 2.
func stringPoolHash(text []byte) uint32 {
	hash := uint32(2166136261)
	for _, b := range text {
		hash ^= uint32(b)
		hash *= 16777619
	}
	return hash
}

func (pool *StringPool) stringPoolSlotBytes(slot uint32) []byte {
	offset := pool.StringPoolOffsets[slot]
	end := offset + pool.StringPoolLengths[slot]
	return pool.StringPoolArena[offset:end:end]
}

func stringPoolGidOf(slot uint32) StringGid {
	return StringGid(objmodel.GidCompose(StringGidTypeTag, objmodel.GidGenerationFirst, slot))
}

// StringPoolFreeze ends interning. It is called when compilation ends, and the
// VM runs only on a frozen pool.
func (pool *StringPool) StringPoolFreeze() {
	pool.StringPoolFrozen = true
}

// StringPoolBytes returns a view into the arena of the string gid names. The
// caller must not write to it. StringGidNil gives an empty slice.
func (pool *StringPool) StringPoolBytes(gid StringGid) []byte {
	if gid == StringGidNil {
		return nil
	}
	slot := objmodel.GidIndex(int64(gid))
	if objmodel.GidTypeTag(int64(gid)) != StringGidTypeTag || objmodel.GidGeneration(int64(gid)) != objmodel.GidGenerationFirst || uint64(slot) >= uint64(len(pool.StringPoolOffsets)) {
		objmodel.GidFirstGenerationFailureReport(int64(gid), StringGidTypeTag, len(pool.StringPoolOffsets))
	}
	return pool.stringPoolSlotBytes(slot)
}

// StringPoolEqual reports whether a and b name equal strings.
func StringPoolEqual(a StringGid, b StringGid) bool {
	return a == b
}

// StringPoolLen returns the number of interned strings.
func (pool *StringPool) StringPoolLen() int {
	return len(pool.StringPoolOffsets)
}

// StringPoolText returns the string gid names as a Go string that views the
// arena, so it does not allocate. Arena bytes are never written again, which
// keeps the view valid for as long as it is held. It is for boundary code
// that still takes strings, such as diagnostics and the string keyed maps.
// VIOLATION STRHOT-278: B, name text for compiler code run by an opcode, #412.
func (pool *StringPool) StringPoolText(gid StringGid) string {
	bytes := pool.StringPoolBytes(gid)
	if len(bytes) == 0 {
		return ""
	}
	return unsafe.String(&bytes[0], len(bytes))
}

// stringPoolInternFrozenReport asserts the StringPoolIntern of text on a
// frozen pool.
// STRHOT REPORTER: builds the assert text from the bytes, #412.
//
//go:noinline
func stringPoolInternFrozenReport(text []byte) {
	assert.PreconditionTrue(false, fmt.Sprintf("StringPoolIntern of %q on a frozen pool", string(text)))
}

// stringPoolFullReport asserts a pool that holds slots strings in an arena
// of arenaLength bytes and has no room for one more.
// STRHOT REPORTER: builds the assert text from the integers, #412.
//
//go:noinline
func stringPoolFullReport(slots int, arenaLength int) {
	assert.InvariantTrue(false, fmt.Sprintf("StringPool full at %d strings and %d bytes", slots, arenaLength))
}

// StringPoolReset empties the pool and keeps its arrays, for a pool that
// holds the text of one call. No Gid it gave names a string afterwards.
func (pool *StringPool) StringPoolReset() {
	pool.StringPoolArena = pool.StringPoolArena[:0]
	pool.StringPoolOffsets = pool.StringPoolOffsets[:0]
	pool.StringPoolLengths = pool.StringPoolLengths[:0]
	clear(pool.StringPoolBuckets)
	pool.StringPoolFrozen = false
}

// StringPoolArenaFree returns the empty slice at the end of the arena. A
// caller may append the bytes of its next StringPoolIntern to it, so that
// building them takes no other buffer.
func (pool *StringPool) StringPoolArenaFree() []byte {
	return pool.StringPoolArena[len(pool.StringPoolArena):]
}
