// Package cipherstate holds which incremental hash states of the cipher
// library are taken, the handles that the old loop and the vm both give
// out, see docs/CipherLibrary.md. It uses only the standard library, so the
// vm reaches it in every build and every core. The hashes themselves are
// in t4-cipher, by the same handle.
package cipherstate

import (
	"fmt"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
)

// HashStateCountMax is the size of the incremental hash table. A program
// that holds more states at once fails with an assert.
const HashStateCountMax = 64

// CipherState marks the taken handles of the incremental hash table.
type CipherState struct {
	CipherHashStateTaken [HashStateCountMax]bool
}

// VIOLATION: state outside State. Owner: t1-cipherstate; moves to a per program state with the cxfx tables, see D45.
var cipherState CipherState

// hashStateAssertFailure ends the program for a handle out of range or
// naming a released or never created state.
// VIOLATION STRHOT-326: E, the assert text of a bad handle, built only when it fails, #510.
func hashStateAssertFailure(handle int32) {
	assert.AssertPreconditionTrue(false, fmt.Sprintf("cipher hash state %d is out of range, released or never created", handle))
}

// hashStateFullAssertFailure ends the program when every state is taken.
// VIOLATION STRHOT-327: E, the assert text of a full table, built only when it fails, #510.
func hashStateFullAssertFailure() {
	assert.AssertPreconditionTrue(false, fmt.Sprintf("cipher hash state table is full, it holds %d", HashStateCountMax))
}

// HashStateTakenCheck asserts that handle names a taken state.
func HashStateTakenCheck(handle int32) {
	if uint32(handle) >= HashStateCountMax || !cipherState.CipherHashStateTaken[handle] {
		hashStateAssertFailure(handle)
	}
}

// HashStateNew takes a free state and returns its handle.
func HashStateNew() int32 {
	for handle := range cipherState.CipherHashStateTaken {
		if !cipherState.CipherHashStateTaken[handle] {
			cipherState.CipherHashStateTaken[handle] = true
			return int32(handle)
		}
	}
	hashStateFullAssertFailure()
	return -1
}

// HashStateRelease frees the state handle names.
func HashStateRelease(handle int32) {
	HashStateTakenCheck(handle)
	cipherState.CipherHashStateTaken[handle] = false
}
