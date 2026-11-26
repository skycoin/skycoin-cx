package cxhash

import (
	"encoding/binary"
	"math/bits"
)

// cxHashPortable is BLAKE3 with a 256 bit output in portable Go, from the
// BLAKE3 specification (https://github.com/BLAKE3-team/BLAKE3-specs,
// sections 2.1 to 2.6). A TinyGo build uses it, as TinyGo cannot link the
// assembly of lukechampine.com/blake3. Tests check it against that package.
type cxHashPortable struct {
	cxHashPortableChunk    cxHashPortableChunk
	cxHashPortableStack    [54][8]uint32
	cxHashPortableStackLen int
}

const (
	cxHashPortableBlockSize = 64
	cxHashPortableChunkSize = 1024

	cxHashPortableFlagChunkStart = 1 << 0
	cxHashPortableFlagChunkEnd   = 1 << 1
	cxHashPortableFlagParent     = 1 << 2
	cxHashPortableFlagRoot       = 1 << 3
)

var cxHashPortableIv = [8]uint32{0x6A09E667, 0xBB67AE85, 0x3C6EF372, 0xA54FF53A, 0x510E527F, 0x9B05688C, 0x1F83D9AB, 0x5BE0CD19}

var cxHashPortablePermutation = [16]int{2, 6, 3, 10, 7, 0, 4, 13, 1, 11, 12, 5, 9, 14, 15, 8}

func cxHashPortableMix(state *[16]uint32, a, b, c, d int, x, y uint32) {
	state[a] += state[b] + x
	state[d] = (state[d]^state[a])>>16 | (state[d]^state[a])<<16
	state[c] += state[d]
	state[b] = (state[b]^state[c])>>12 | (state[b]^state[c])<<20
	state[a] += state[b] + y
	state[d] = (state[d]^state[a])>>8 | (state[d]^state[a])<<24
	state[c] += state[d]
	state[b] = (state[b]^state[c])>>7 | (state[b]^state[c])<<25
}

// cxHashPortableCompress returns the 16 word state after compressing block
// into cv, section 2.2.
func cxHashPortableCompress(cv [8]uint32, block [16]uint32, counter uint64, blockLen uint32, flags uint32) [16]uint32 {
	state := [16]uint32{
		cv[0], cv[1], cv[2], cv[3], cv[4], cv[5], cv[6], cv[7],
		cxHashPortableIv[0], cxHashPortableIv[1], cxHashPortableIv[2], cxHashPortableIv[3],
		uint32(counter), uint32(counter >> 32), blockLen, flags,
	}
	message := block
	for round := 0; round < 7; round++ {
		cxHashPortableMix(&state, 0, 4, 8, 12, message[0], message[1])
		cxHashPortableMix(&state, 1, 5, 9, 13, message[2], message[3])
		cxHashPortableMix(&state, 2, 6, 10, 14, message[4], message[5])
		cxHashPortableMix(&state, 3, 7, 11, 15, message[6], message[7])
		cxHashPortableMix(&state, 0, 5, 10, 15, message[8], message[9])
		cxHashPortableMix(&state, 1, 6, 11, 12, message[10], message[11])
		cxHashPortableMix(&state, 2, 7, 8, 13, message[12], message[13])
		cxHashPortableMix(&state, 3, 4, 9, 14, message[14], message[15])
		var permuted [16]uint32
		for i, from := range cxHashPortablePermutation {
			permuted[i] = message[from]
		}
		message = permuted
	}
	for i := 0; i < 8; i++ {
		state[i] ^= state[i+8]
		state[i+8] ^= cv[i]
	}
	return state
}

func cxHashPortableWords(block [cxHashPortableBlockSize]byte) [16]uint32 {
	var words [16]uint32
	for i := range words {
		words[i] = binary.LittleEndian.Uint32(block[4*i:])
	}
	return words
}

// cxHashPortableNode is the input of the last compression of a chunk or a
// parent, which gives either its chaining value or the root output.
type cxHashPortableNode struct {
	cxHashPortableNodeCv       [8]uint32
	cxHashPortableNodeBlock    [16]uint32
	cxHashPortableNodeCounter  uint64
	cxHashPortableNodeBlockLen uint32
	cxHashPortableNodeFlags    uint32
}

func (node cxHashPortableNode) chainingValue() [8]uint32 {
	state := cxHashPortableCompress(node.cxHashPortableNodeCv, node.cxHashPortableNodeBlock, node.cxHashPortableNodeCounter, node.cxHashPortableNodeBlockLen, node.cxHashPortableNodeFlags)
	return [8]uint32(state[:8])
}

func cxHashPortableParent(left, right [8]uint32) cxHashPortableNode {
	node := cxHashPortableNode{cxHashPortableNodeCv: cxHashPortableIv, cxHashPortableNodeBlockLen: cxHashPortableBlockSize, cxHashPortableNodeFlags: cxHashPortableFlagParent}
	for i := 0; i < 8; i++ {
		node.cxHashPortableNodeBlock[i], node.cxHashPortableNodeBlock[i+8] = left[i], right[i]
	}
	return node
}

// cxHashPortableChunk is the chunk being hashed, section 2.4.
type cxHashPortableChunk struct {
	cxHashPortableChunkCv               [8]uint32
	cxHashPortableChunkCounter          uint64
	cxHashPortableChunkBlock            [cxHashPortableBlockSize]byte
	cxHashPortableChunkBlockLen         int
	cxHashPortableChunkBlocksCompressed int
}

func cxHashPortableChunkNew(counter uint64) cxHashPortableChunk {
	return cxHashPortableChunk{cxHashPortableChunkCv: cxHashPortableIv, cxHashPortableChunkCounter: counter}
}

func (chunk *cxHashPortableChunk) length() int {
	return cxHashPortableBlockSize*chunk.cxHashPortableChunkBlocksCompressed + chunk.cxHashPortableChunkBlockLen
}

func (chunk *cxHashPortableChunk) startFlag() uint32 {
	if chunk.cxHashPortableChunkBlocksCompressed == 0 {
		return cxHashPortableFlagChunkStart
	}
	return 0
}

func (chunk *cxHashPortableChunk) update(input []byte) {
	for len(input) > 0 {
		if chunk.cxHashPortableChunkBlockLen == cxHashPortableBlockSize {
			words := cxHashPortableWords(chunk.cxHashPortableChunkBlock)
			state := cxHashPortableCompress(chunk.cxHashPortableChunkCv, words, chunk.cxHashPortableChunkCounter, cxHashPortableBlockSize, chunk.startFlag())
			chunk.cxHashPortableChunkCv = [8]uint32(state[:8])
			chunk.cxHashPortableChunkBlocksCompressed++
			chunk.cxHashPortableChunkBlock = [cxHashPortableBlockSize]byte{}
			chunk.cxHashPortableChunkBlockLen = 0
		}
		taken := copy(chunk.cxHashPortableChunkBlock[chunk.cxHashPortableChunkBlockLen:], input)
		chunk.cxHashPortableChunkBlockLen += taken
		input = input[taken:]
	}
}

func (chunk *cxHashPortableChunk) output() cxHashPortableNode {
	return cxHashPortableNode{
		cxHashPortableNodeCv:       chunk.cxHashPortableChunkCv,
		cxHashPortableNodeBlock:    cxHashPortableWords(chunk.cxHashPortableChunkBlock),
		cxHashPortableNodeCounter:  chunk.cxHashPortableChunkCounter,
		cxHashPortableNodeBlockLen: uint32(chunk.cxHashPortableChunkBlockLen),
		cxHashPortableNodeFlags:    chunk.startFlag() | cxHashPortableFlagChunkEnd,
	}
}

func cxHashPortableNew() *cxHashPortable {
	return &cxHashPortable{cxHashPortableChunk: cxHashPortableChunkNew(0)}
}

// Write adds input to the hash, section 2.5. It never fails.
func (hasher *cxHashPortable) Write(input []byte) (int, error) {
	written := len(input)
	for len(input) > 0 {
		if hasher.cxHashPortableChunk.length() == cxHashPortableChunkSize {
			node := hasher.cxHashPortableChunk.output()
			cv := node.chainingValue()
			total := hasher.cxHashPortableChunk.cxHashPortableChunkCounter + 1
			for total&1 == 0 {
				hasher.cxHashPortableStackLen--
				parent := cxHashPortableParent(hasher.cxHashPortableStack[hasher.cxHashPortableStackLen], cv)
				cv = parent.chainingValue()
				total >>= 1
			}
			hasher.cxHashPortableStack[hasher.cxHashPortableStackLen] = cv
			hasher.cxHashPortableStackLen++
			hasher.cxHashPortableChunk = cxHashPortableChunkNew(hasher.cxHashPortableChunk.cxHashPortableChunkCounter + 1)
		}
		taken := min(cxHashPortableChunkSize-hasher.cxHashPortableChunk.length(), len(input))
		hasher.cxHashPortableChunk.update(input[:taken])
		input = input[taken:]
	}
	return written, nil
}

// Sum appends the 32 byte digest to b and leaves the hash as it was.
func (hasher *cxHashPortable) Sum(b []byte) []byte {
	node := hasher.cxHashPortableChunk.output()
	for i := hasher.cxHashPortableStackLen - 1; i >= 0; i-- {
		node = cxHashPortableParent(hasher.cxHashPortableStack[i], node.chainingValue())
	}
	state := cxHashPortableCompress(node.cxHashPortableNodeCv, node.cxHashPortableNodeBlock, 0, node.cxHashPortableNodeBlockLen, node.cxHashPortableNodeFlags|cxHashPortableFlagRoot)
	for i := 0; i < 8; i++ {
		b = binary.LittleEndian.AppendUint32(b, state[i])
	}
	return b
}

// Reset clears the hash.
func (hasher *cxHashPortable) Reset() {
	*hasher = cxHashPortable{cxHashPortableChunk: cxHashPortableChunkNew(0)}
}

// Size is the digest size, 32 bytes.
func (hasher *cxHashPortable) Size() int {
	return CxHashSize
}

// BlockSize is the BLAKE3 block size, 64 bytes.
func (hasher *cxHashPortable) BlockSize() int {
	return cxHashPortableBlockSize
}

// cxHashPortableSubtree returns the last node of the subtree over input,
// whose first chunk is number counter. The left subtree holds the largest
// power of two chunks that leaves the right one nonempty (BLAKE3
// specification, section 5.1.2). The recursion needs no chaining value
// stack, which TinyGo would allocate on the heap.
func cxHashPortableSubtree(input []byte, counter uint64) cxHashPortableNode {
	if len(input) <= cxHashPortableChunkSize {
		chunk := cxHashPortableChunkNew(counter)
		chunk.update(input)
		return chunk.output()
	}
	leftLen := cxHashPortableChunkSize << (bits.Len64(uint64((len(input)-1)/cxHashPortableChunkSize)) - 1)
	left := cxHashPortableSubtree(input[:leftLen], counter).chainingValue()
	right := cxHashPortableSubtree(input[leftLen:], counter+uint64(leftLen/cxHashPortableChunkSize)).chainingValue()
	return cxHashPortableParent(left, right)
}

// cxHashPortableOfBytes returns the digest of input. It allocates nothing.
func cxHashPortableOfBytes(input []byte) [CxHashSize]byte {
	node := cxHashPortableSubtree(input, 0)
	state := cxHashPortableCompress(node.cxHashPortableNodeCv, node.cxHashPortableNodeBlock, 0, node.cxHashPortableNodeBlockLen, node.cxHashPortableNodeFlags|cxHashPortableFlagRoot)
	var digest [CxHashSize]byte
	for i := 0; i < 8; i++ {
		binary.LittleEndian.PutUint32(digest[4*i:], state[i])
	}
	return digest
}
