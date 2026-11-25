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

// gcTestState returns a GCState prepared for memory and its heap.
func gcTestState(memory []byte, heapStartsAt types.Pointer) *GCState {
	var state GCState
	GCStatePrepare(&state, types.Cast_int_to_ptr(len(memory))-heapStartsAt, types.Cast_int_to_ptr(len(memory)))
	return &state
}

func TestGarbageCollectorCollectMovesLiveObjectDown(t *testing.T) {
	memory, heapStartsAt, heapPointer, _ := gcTestHeap()
	roots := &GCRootTable{GCRootTableGlobals: []GCRoot{{GCRootOffset: 0, GCRootType: &GCType{GCTypeKind: GCKindStr}}}, GCRootTableStackSize: 32}
	pointer := GarbageCollectorCollect(memory, heapStartsAt, heapPointer, roots, gcTestState(memory, heapStartsAt))
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
	if pointer := GarbageCollectorCollect(memory, heapStartsAt, heapPointer, &GCRootTable{GCRootTableStackSize: 32}, gcTestState(memory, heapStartsAt)); pointer != constants.NULL_HEAP_ADDRESS_OFFSET {
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
	GarbageCollectorCollect(memory, heapStartsAt, heapPointer, roots, gcTestState(memory, heapStartsAt))
	if got, want := types.Read_ptr(memory, 16+8), heapStartsAt+constants.NULL_HEAP_ADDRESS_OFFSET; got != want {
		t.Fatalf("frame slot points to %d, want %d", got, want)
	}
}

// gcTestList builds a heap at 64 of count garbage strs and count list nodes,
// each holding a pointer to the next node and a str, with a root at 0 that
// points to the first node. It returns the root type.
func gcTestList(state *GCState, count int) (memory []byte, heapStartsAt types.Pointer, heapPointer types.Pointer, root *GCType) {
	heapStartsAt = 64
	node := &GCType{GCTypeKind: GCKindStruct}
	pointer := &GCType{GCTypeKind: GCKindPointer, GCTypeElem: node}
	GCStateTypeAdd(state, pointer)
	node.GCTypeFields = []GCTypeField{{GCTypeFieldOffset: 0, GCTypeFieldType: pointer}, {GCTypeFieldOffset: types.POINTER_SIZE, GCTypeFieldType: &GCType{GCTypeKind: GCKindStr}}}
	strSize := types.OBJECT_HEADER_SIZE + 1
	nodeSize := types.OBJECT_HEADER_SIZE + 2*types.POINTER_SIZE
	memory = make([]byte, heapStartsAt+constants.NULL_HEAP_ADDRESS_OFFSET+types.Cast_int_to_ptr(count)*(2*strSize+nodeSize))
	c := heapStartsAt + constants.NULL_HEAP_ADDRESS_OFFSET
	previous := types.Pointer(0)
	for i := 0; i < count; i++ {
		types.Write_obj_data(memory, c, strSize, []byte{'g'})
		c += strSize
		str := c
		types.Write_obj_data(memory, str, strSize, []byte{byte('a' + i%26)})
		c += strSize
		types.Write_obj_size(memory, c, nodeSize)
		types.Write_ptr(memory, c+types.OBJECT_HEADER_SIZE+types.POINTER_SIZE, str)
		if previous == 0 {
			types.Write_ptr(memory, 0, c)
		} else {
			types.Write_ptr(memory, previous+types.OBJECT_HEADER_SIZE, c)
		}
		previous = c
		c += nodeSize
	}
	return memory, heapStartsAt, c - heapStartsAt, pointer
}

// gcTestListCheck walks the list from the root at 0 and fails unless it has
// count nodes whose strs read a, b, c and on.
func gcTestListCheck(t *testing.T, memory []byte, count int) {
	t.Helper()
	node := types.Read_ptr(memory, 0)
	for i := 0; i < count; i++ {
		if node == 0 {
			t.Fatalf("list ends after %d nodes, want %d", i, count)
		}
		str := types.Read_ptr(memory, node+types.OBJECT_HEADER_SIZE+types.POINTER_SIZE)
		if got, want := memory[str+types.OBJECT_HEADER_SIZE], byte('a'+i%26); got != want {
			t.Fatalf("node %d holds %q, want %q", i, got, want)
		}
		node = types.Read_ptr(memory, node+types.OBJECT_HEADER_SIZE)
	}
}

// A list collects with no allocation, and with a GCStateWork of one entry,
// which leaves nodes for the scan of the heap, it keeps the same objects.
func TestGarbageCollectorCollectList(t *testing.T) {
	const count = 300
	for _, work := range []int{0, 1} {
		var state GCState
		memory, heapStartsAt, heapPointer, root := gcTestList(&state, count)
		GCStatePrepare(&state, types.Cast_int_to_ptr(len(memory))-heapStartsAt, types.Cast_int_to_ptr(len(memory)))
		if work != 0 {
			state.GCStateWork = make([]types.Pointer, 0, work)
		}
		roots := &GCRootTable{GCRootTableGlobals: []GCRoot{{GCRootOffset: 0, GCRootType: root}}, GCRootTableStackSize: 32}
		var pointer types.Pointer
		allocs := testing.AllocsPerRun(1, func() {
			pointer = GarbageCollectorCollect(memory, heapStartsAt, heapPointer, roots, &state)
		})
		if allocs != 0 {
			t.Errorf("work %d: %.0f allocations per collection, want 0", work, allocs)
		}
		live := constants.NULL_HEAP_ADDRESS_OFFSET + count*(types.OBJECT_HEADER_SIZE+1+types.OBJECT_HEADER_SIZE+2*types.POINTER_SIZE)
		if pointer != live {
			t.Errorf("work %d: heap pointer %d, want %d", work, pointer, live)
		}
		gcTestListCheck(t, memory, count)
		// A second collection moves nothing and keeps the list.
		if again := GarbageCollectorCollect(memory, heapStartsAt, pointer, roots, &state); again != live {
			t.Errorf("work %d: second collection heap pointer %d, want %d", work, again, live)
		}
		gcTestListCheck(t, memory, count)
	}
}

// Pointers into an object past the capacity of GCStateViews grow it, and
// the collection keeps the object and moves the pointers into it.
func TestGarbageCollectorCollectViewsGrow(t *testing.T) {
	memory, heapStartsAt, heapPointer, live := gcTestHeap()
	inner := &GCType{GCTypeKind: GCKindPointer, GCTypeElem: &GCType{GCTypeKind: GCKindStr}}
	roots := &GCRootTable{GCRootTableStackSize: 32}
	for i := types.Pointer(0); i < 3; i++ {
		types.Write_ptr(memory, i*8, live+1+i)
		roots.GCRootTableGlobals = append(roots.GCRootTableGlobals, GCRoot{GCRootOffset: i * 8, GCRootType: inner})
	}
	state := gcTestState(memory, heapStartsAt)
	state.GCStateViews = make([]GCVisit, 0, 2)
	GarbageCollectorCollect(memory, heapStartsAt, heapPointer, roots, state)
	moved := heapStartsAt + constants.NULL_HEAP_ADDRESS_OFFSET
	for i := types.Pointer(0); i < 3; i++ {
		if got, want := types.Read_ptr(memory, i*8), moved+1+i; got != want {
			t.Errorf("pointer %d is %d, want %d", i, got, want)
		}
	}
	if got := string(memory[moved+types.OBJECT_HEADER_SIZE : moved+types.OBJECT_HEADER_SIZE+4]); got != "live" {
		t.Fatalf("moved object holds %q, want live", got)
	}
}
