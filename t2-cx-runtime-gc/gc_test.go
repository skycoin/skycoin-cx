package gc

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
)

// gcTestHeap builds a memory with a root slot at 0 and a heap at 64 that
// holds a garbage str "dead" and then a live str "live" the root points to.
func gcTestHeap() (memory []byte, heapStartsAt types.Pointer, heapPointer types.Pointer, live types.Pointer) {
	heapStartsAt = 64
	size := types.OBJECT_HEADER_SIZE + 4
	memory = make([]byte, 256)
	garbage := heapStartsAt + constants.NULL_HEAP_ADDRESS_OFFSET
	live = garbage + size
	types.Write_obj_data(memory, garbage, size, []byte("dead"))
	types.Write_obj_data(memory, live, size, []byte("live"))
	types.Write_ptr(memory, 0, live)
	return memory, heapStartsAt, live + size - heapStartsAt, live
}

func TestGarbageCollectorCollectMovesLiveObjectDown(t *testing.T) {
	memory, heapStartsAt, heapPointer, _ := gcTestHeap()
	roots := &GCRootTable{GCRootTableGlobals: []GCRoot{{GCRootOffset: 0, GCRootType: &GCType{GCTypeKind: GCKindStr}}}, GCRootTableStackSize: 32}
	var state GCState
	pointer := GarbageCollectorCollect(memory, heapStartsAt, heapPointer, roots, &state)
	moved := heapStartsAt + constants.NULL_HEAP_ADDRESS_OFFSET
	if want := constants.NULL_HEAP_ADDRESS_OFFSET + types.OBJECT_HEADER_SIZE + 4; pointer != want {
		t.Fatalf("heap pointer %d, want %d", pointer, want)
	}
	if got := types.Read_ptr(memory, 0); got != moved {
		t.Fatalf("root points to %d, want %d", got, moved)
	}
	if got := string(memory[moved+types.OBJECT_HEADER_SIZE : moved+types.OBJECT_HEADER_SIZE+4]); got != "live" {
		t.Fatalf("moved object holds %q, want live", got)
	}
}

func TestGarbageCollectorCollectFreesUnreachable(t *testing.T) {
	memory, heapStartsAt, heapPointer, _ := gcTestHeap()
	var state GCState
	if pointer := GarbageCollectorCollect(memory, heapStartsAt, heapPointer, &GCRootTable{GCRootTableStackSize: 32}, &state); pointer != constants.NULL_HEAP_ADDRESS_OFFSET {
		t.Fatalf("heap pointer %d, want %d", pointer, constants.NULL_HEAP_ADDRESS_OFFSET)
	}
}

// A frame root at a frame offset is read relative to its frame pointer.
func TestGarbageCollectorCollectFrameRoot(t *testing.T) {
	memory, heapStartsAt, heapPointer, live := gcTestHeap()
	types.Write_ptr(memory, 0, 0)
	types.Write_ptr(memory, 16+8, live)
	roots := &GCRootTable{
		GCRootTableFrames:    []GCRootFrame{{GCRootFramePointer: 16, GCRootFrameRoots: []GCRoot{{GCRootOffset: 8, GCRootType: &GCType{GCTypeKind: GCKindStr}}}}},
		GCRootTableStackSize: 32,
	}
	var state GCState
	GarbageCollectorCollect(memory, heapStartsAt, heapPointer, roots, &state)
	if got, want := types.Read_ptr(memory, 16+8), heapStartsAt+constants.NULL_HEAP_ADDRESS_OFFSET; got != want {
		t.Fatalf("frame slot points to %d, want %d", got, want)
	}
}
