# Garbage collector

## Purpose

The interpreter and the IR base must not depend on the garbage collector.
The collector runs as a tick of the main loop and sees only a root table and
the heap. This document lists every place that reaches it, shows how the
packages depend on each other, and describes the tick and its interface.
Each reach carries a two line `GCREACH:` comment in the code, and `make
check-gcreach` fails when a new one appears without it. See #485.

## The collector

The heap is the part of `CXProgram.Memory` after the stack and the data
segment, described by `CXProgram.Heap` (`HeapSegmentStruct` in
t3-ast/ast_cxprogram.go). Objects are allocated by bumping `Heap.Pointer`.

`gc.GarbageCollectorCollect` in t2-cx-runtime-gc is a mark and compact
collector. It marks every object reachable from the roots, gives each live
object a new address, rewrites the references it recorded while marking, and
slides the objects down. It imports t0-types and t1-constants only.

### Interface

```go
func GarbageCollectorCollect(memory []byte, heapStartsAt types.Pointer,
	heapPointer types.Pointer, roots *GCRootTable, state *GCState) types.Pointer
```

- `memory` and the heap bounds `heapStartsAt` to `heapStartsAt+heapPointer`.
- `roots`, a `GCRootTable`: the global roots, each a `GCRoot` with an address
  and a `GCType`, the live frames, each a `GCRootFrame` with its frame
  pointer and the roots of its function, and the stack size.
- `state`, the collector's scratch memory. `GCStatePrepare` allocates it
  once for the heap cap, and a collection allocates nothing, see "The heap
  cap" below.

It returns the new heap pointer. A `GCType` says where a value holds
pointers, slices and strs. A nil `*GCType` holds none, so values without
references cost nothing to walk.

The collector sees no CX IR, no `CXProgram`, no call stack and no opcode.
t3-ast builds the root table: `gcGlobalRoots` and `gcFrameRoots` in
t3-ast/gc_type.go turn the globals and each function's `ListOfSlots` and
`ListOfPointers` into `GCRoot`s, cached in `Heap.Types`.
`GarbageCollectorPrepare` builds them for the globals and every function
before the run. `make
check-gcreach` fails when any package but t3-ast imports t2-cx-runtime-gc.

### The tick

The allocator never collects. `AllocateSeq` bumps the heap pointer, adds the
bytes to `Heap.Pressure`, and grows the heap when a request does not fit,
leaving `MIN_HEAP_FREE_RATIO` of it free. Past the heap cap it panics with
`RuntimeOutOfMemory`, the runtime error `CX_RUNTIME_HEAP_EXHAUSTED_ERROR`.

The old loop ticks after every step, one expression, in `runLoop` and
`runSysInitFunc` (t4-execute/execute.go). Between two steps every root is in a
global or a frame slot, so no Go code holds a heap address.

- `GarbageCollectorCheck` (t3-ast/gc_tick.go) reports `Heap.Pressure >
  Heap.PressureLimit`, two loads and a compare.
- `GarbageCollectorTrigger` builds the root table from the globals and the
  live frames, runs the collector, resizes the heap to keep between
  `MIN_HEAP_FREE_RATIO` and `MAX_HEAP_FREE_RATIO` of it free, and sets
  `PressureLimit` to the free heap, but to no more than half of what is left
  below the heap cap.

So the collection runs once the program has allocated what was free after
the last one, which is when the inline collector ran before, moved to the end
of the step. Near the cap the limit is lower, so the collection runs before
the heap is full instead of the next allocation failing. A heap whose live
data stays well under the cap collects as before. `-heap-initial 0`
still collects after every step that
allocates while the heap is small, so the `test-gc*.cx` programs stress it as
before.

The loop that `Callback` runs inside a native opcode does not tick, so no
object moves under an opcode in progress. `Heap.Pinned` is gone, as nothing
collects inside `AllocateStrAt`.

The trade-off is a single step that needs more than the room left below
the heap cap. The inline collector could free garbage for it in the middle
of the step. The tick collects only between steps, so such a step is a heap
exhausted error.

### The heap cap

The heap has a fixed cap, `Heap.Max` (D42). It is `MAX_HEAP_SIZE`, 64M,
unless `-heap-max` sets another, and `cx build` stores it in the built
program. `RunPrepare` calls `EnsureMinimumHeapSize`, which allocates once,
before the run:

- `prgrm.Memory` with the capacity of the stack, the data and the cap.
  `ResizeMemory` grows the heap by reslicing into it and zeroing the new
  bytes, and shrinks it by reslicing.
- the root table: the roots of the globals and of every function, and
  `GCRootTableFrames` for a full call stack.
- the scratch, `GCState`, by `GCStatePrepare`.

An embedding program that loads programs one after another calls
`ast.ProgramMemoryRelease` on a program it is done with. Its memory and
scratch go to a reserve that the next `EnsureMinimumHeapSize` takes, so the
process allocates them once instead of once per load (D106). The reserve is
not cleared. The heap clears the bytes it grows into, and a collection leaves
the bitmaps clear, so only scratch from a collection that did not finish is
dropped. The REPL, the t11-cx tests and the playground release each
program before the next.

| Scratch | Size | Use |
| --- | --- | --- |
| `GCStateStarts` | a bit per byte of the heap cap, cap/8 bytes | where each object starts, for the object a pointer points into. It replaces the sorted list of object addresses |
| `GCStateSlots` and `GCStateSlotWords` | a bit per byte of memory, and a bit per word of that | the references the mark read. Its set bits are the references to update, in address order, so the sort of the old list is gone |
| `GCStateWork` | cap/1024 entries, at least 256 | objects to walk. The type to walk an object with is in its forwarding address field until the mark ends, which replaces the map of followed pointers |
| `GCStateViews` | memory/512 entries, at least 256 | pointers into an object, as `&s[1]`, and pointers to an object already queued with another type |

For the default cap of 64M the scratch is about 19 MB of Go memory. Pages
the collection does not touch are not committed on Linux, macOS and Windows.

The mark follows a reference only the first time it reads its slot. CX gives
a memory location one type, so the first read queued everything the
reference leads to, and the slot bitmap ends the walk of a cycle. An object
that does not fit a full `GCStateWork` is left marked pending, and the mark
scans the heap for pending objects once the queues are empty, so a deep list
needs no more scratch. More pointers into objects than `GCStateViews` holds
at once grow it, the one Go allocation a collection can make, VMALLOC-011 and
decision D97.

An allocation past the cap is the runtime error
`CX_RUNTIME_HEAP_EXHAUSTED_ERROR` with the message `out of memory: 80017
bytes requested with 4 taken do not fit the heap cap of 65536 bytes, raise it
with -heap-max`. Taken is the heap pointer, garbage included, since the
collector runs only between steps.

### The vm

The vm (t3-cx-runtime-vm) has no heap. `lower` rejects every heap operand
(`ReasonOperandHeap`) and every opcode outside its subset, so a program that
allocates runs on the old loop, and the vm has nothing to collect and no
tick. When it gains heap opcodes, its safe points are `OpcodeCall`,
`OpcodeReturn` and backward jumps, where every root is in a frame slot, and
the check stays two loads so `TestRunVmAllocs` stays 0.

## Which packages are GC free

| Package | Imports t3-ast | Reaches the GC |
| --- | --- | --- |
| t2-cx-runtime-ir, the IR base | no | no |
| t3-cx-runtime-vm, the vm executor | no | no. The machine has no heap |
| t2-cx-runtime-gc, the collector | no | it is the collector |
| t4-cx-compiler-lower | yes | no. It reads the program and rejects heap operands |
| t4-execute, the old loop | yes | the tick, and allocation through its opcodes |
| t4-opcodes | yes | allocation, in 17 of its 22 files |
| t5-cx-runtime-core, the executor | yes | through the old loop |

`make check-gcreach` fails when t2-cx-runtime-ir or t3-cx-runtime-vm imports
t3-ast or t2-cx-runtime-gc.

## Reach list

Lines are as of this document. Regenerate them with `./check-gcreach.sh -l`.

### The tick and the collector

- t4-execute/execute.go:46 to :47 the tick in `runLoop`, :78 to :79 in
  `runSysInitFunc`
- t3-ast/gc_tick.go:14 `GarbageCollectorCheck`, :26 and :29
  `GarbageCollectorPrepare` building the roots before the run, :50 to :63
  `GarbageCollectorTrigger` building the root table and calling the
  collector, :78 and :85 the heap resize after it

### Imports of t3-ast by the interpreter and the IR base

- t4-cx-compiler-lower/lower.go:20, lower_printf.go:12. Reads only
- t4-execute/execute.go:11, callback.go:9
- t5-cx-runtime-core/core.go:15, executor.go:12
- t4-opcodes, allocating: op_aff.go:11, op_f32.go:10, op_f64.go:10,
  op_fmt.go:11, op_i8.go:9, op_i16.go:9, op_i32.go:9, op_i64.go:9,
  op_serialize.go:8, op_slice.go:8, op_stdin.go:10, op_str.go:12,
  op_testing.go:10, op_ui8.go:10, op_ui16.go:10, op_ui32.go:9, op_ui64.go:9
- t4-opcodes, not allocating: op_bool.go:8, op_format.go:8, op_misc.go:7,
  opcodes.go:11, opcodes_init.go:8

### Allocation, which adds to the pressure

- t3-ast/ast_memory.go `EnsureMinimumHeapSize` starts the pressure, :93
  to :94 `memoryReserveTake` and :120 to :127 `ProgramMemoryRelease` hand
  the collector scratch between programs, :176 to :179 `AllocateSeq` grows
  the heap and adds to it, :196 `AllocateStr`, :208 `AllocateStrAt`, :218
  `AllocateStrBytesAt`
- t3-ast/ast_cxvalue.go:219 `CXValue.Set_str`, used by the system library
  opcodes, :227 `Set_str_bytes` and :235 `Set_str_concat`, the path of every
  other opcode that writes a str
- t3-ast/mem_escape.go:57 `EscapedAllocate`, and its caller
  t3-ast/ast_cxcall.go:96 in `wipeDeclarationMemory`
- t3-ast/mem_slice.go:104 `SliceResizeEx`, :152 `SliceResize`, :271
  `WriteToSlice`
- t4-execute/execute.go:97 `feedOSArgs`
- t4-opcodes/op_aff.go, 27 lines in `queryParam`, `QueryExpressions`,
  `getSignatureSlice`, `queryStructsInPackage`, `QueryFunction`,
  `QueryCaller`, `QueryProgram` and `OpAffQuery`
- t4-opcodes/op_serialize.go:24 `OpSerialize`
- t4-opcodes/op_str.go:159 `OpStrBytesOfString`
- t4-regexp/op_regexp.go:32 `regexpCompile`
- t6-cxfx/event.go:64, op_glfw_desktop.go:237, op_openal.go:47,
  op_opengl.go:464

Through these, every opcode that writes a str or grows a slice allocates.
They are not marked one by one.

### GC root tables

- t3-ast/cxfunction_def.go:48 `ListOfPointers` and :57 `ListOfSlots`
- t4-actions/gcroot.go:31 adds to `ListOfSlots`, t4-actions/functions.go:1223
  to :1308 reads and adds to `ListOfPointers`
- t3-ast/serialize.go:736 to :751 writes and :807 to :831 reads both tables

## Before and after

This section records how the collector was wired before #485 and how it is
wired now, so the change can be reviewed as one structure against another.

### Before

This part describes the structure that PR #488 documented, where the
allocator ran the collector itself.

- `AllocateSeq` (t3-ast/ast_memory.go:65) called `MarkAndCompact`
  (t3-ast/garbage_collector.go:14) at line 78 whenever a request did not fit
  in `Heap.Size`, and then grew or shrank the heap.
- Every opcode that allocated could therefore move every heap object while Go
  code in the middle of that opcode held addresses.
- `AllocateStrAt` (t3-ast/ast_memory.go:141) protected one such address by
  storing it in `Heap.Pinned` at line 142, and nothing else was protected.
- t0-types held a function variable `Allocator` (t0-types/types.go:149), which
  t11-cx/main.go:94 and t6-cx-core-run/main.go:27 set to `ast.AllocateSeq`,
  so the lowest tier reached the collector through a hook.
- `MarkAndCompact` took the whole `*CXProgram`. It read the globals through
  `gcGlobalRoots` (t3-ast/gc_type.go:276), walked `CallStack` and each
  function's `ListOfSlots` and `ListOfPointers` (t3-ast/cxfunction_def.go:57
  and :48) through `gcFrameRoots` (t3-ast/gc_type.go:241), and kept its
  scratch memory in `HeapSegmentStruct` (t3-ast/ast_cxprogram.go:31).
- The old loop (t4-execute), t4-opcodes, t5-cxos, t4-regexp, t4-cxnet, t6-cxfx
  and t5-cx-runtime-core all reached the collector through allocation. The vm
  (t3-cx-runtime-vm) and the IR base (t2-cx-runtime-ir) did not.

```
t11-cx, t6-cx-core-run ----- set types.Allocator = ast.AllocateSeq
        |
t5-cx-runtime-core
   |            \
   |             t4-cx-compiler-lower --> t2-cx-runtime-ir   (GC free)
   |                                           ^
   |             t3-cx-runtime-vm -------------+             (GC free)
   |
t4-execute (old loop) --> t4-opcodes, t5-*, t6-cxfx
   |                          |
   +--------------------------+--> t3-ast
                                     AllocateSeq --> MarkAndCompact(*CXProgram)
                                                        |
                                      gcGlobalRoots, gcFrameRoots, CallStack
                                      ListOfSlots, ListOfPointers
                                      Heap.Types, Heap.Pinned, Heap.Visited ...
t0-types: Allocator hook --> AllocateSeq (through the function variable)
```

### After

This part describes the structure of this change, where the main loop runs
the collector between steps.

- `AllocateSeq` (t3-ast/ast_memory.go:74) never collects. It grows the heap at
  line 84 when a request does not fit, adds the bytes to `Heap.Pressure` at
  line 87, and panics with `HEAP_EXHAUSTED_ERROR` past `MAX_HEAP_SIZE`.
- `runLoop` (t4-execute/execute.go:27) calls `GarbageCollectorCheck` after
  every step at line 46, and `runSysInitFunc` (t4-execute/execute.go:56) does
  the same at line 78. The loop that `Callback` nests inside a native opcode
  does not tick, because `gcTick` at line 31 is false there.
- `GarbageCollectorCheck` (t3-ast/gc_tick.go:13) compares `Heap.Pressure` with
  `Heap.PressureLimit`, which costs two loads and a compare.
- `GarbageCollectorTrigger` (t3-ast/gc_tick.go:23) builds a `GCRootTable` from
  `gcGlobalRoots` (t3-ast/gc_type.go:232) and, for each live call, the call's
  `FramePointer` and `gcFrameRoots` (t3-ast/gc_type.go:197). It then calls the
  collector and resizes the heap with `heapFreeRatioKeep`
  (t3-ast/gc_tick.go:48).
- `GarbageCollectorCollect` (t2-cx-runtime-gc/gc.go:108) receives the memory,
  the heap bounds, the `GCRootTable` (t2-cx-runtime-gc/gc.go:68) and its own
  `GCState` (t2-cx-runtime-gc/gc.go:82). It imports only t0-types and
  t1-constants and sees no CX IR, no `CXProgram` and no call stack.
- `Heap.Pinned`, the `types.Allocator` hook, `types.Write_obj`,
  `types.Write_str`, `types.AllocWrite_obj_data` and
  `types.AllocWrite_str_data` are removed. The unused `DisplaceReferences` is
  removed with the old collector file.
- Only t3-ast may import t2-cx-runtime-gc. The old loop, t4-opcodes, t5-cxos,
  t4-regexp, t4-cxnet and t6-cxfx still allocate through t3-ast, which adds to
  the pressure but never collects. The vm and the IR base still reach nothing,
  and `make check-gcreach` enforces both rules.

```
t11-cx, t6-cx-core-run
        |
t5-cx-runtime-core
   |            \
   |             t4-cx-compiler-lower --> t2-cx-runtime-ir   (GC free)
   |                                           ^
   |             t3-cx-runtime-vm -------------+             (GC free)
   |
t4-execute: runLoop step, then GarbageCollectorCheck / GarbageCollectorTrigger
   |    \
   |     t4-opcodes, t5-*, t6-cxfx: allocate, add to Heap.Pressure
   |          |
   +----------+--> t3-ast: AllocateSeq (grows, counts), gc_type.go (root
                   table), gc_tick.go (check, trigger, heap resize)
                      |
                      v  memory, heap bounds, GCRootTable, GCState
                 t2-cx-runtime-gc: GarbageCollectorCollect
                      |
                 t0-types, t1-constants
```

### Measured cost of the tick

This part gives the cost of the change as measured on the base commit
64762a20 and on this branch.

- The operation counts of a `-tags cxopcount` build do not change. test-gc.cx
  runs 765405 operations, test-gc-nested.cx 562126, test-gc-slice-resize.cx
  2312 and a program with a non-empty `*init` frame 2367, on both builds.
- `TestRunAllocs` reports the same Go allocations per run on both builds, for
  example 7 for count, 6 for fib and 202756 for test-gc.cx.
  `TestRunVmAllocs` stays 0, because the vm code does not change.
- benchstat compared `BenchmarkRun` and `BenchmarkRunVm` over three builds per
  side that differ only in `-ldflags=-funcalign=32`, 64 and 128, five
  interleaved rounds each, pinned to one core. No benchmark changed
  significantly except `Run/test-gc.cx`, which ran 12.3% faster (p=0.037). The
  geomean of the three layouts moved by +4.4%, -4.9% and -19.5%, so code
  placement and the machine's load (average 20 to 35 during the runs) move the
  results more than the tick does.
- The Go bytes allocated per run of test-gc.cx rose from 14.9 MiB to
  28.2 MiB. The heap now grows before it is collected, so `prgrm.Memory` is
  reallocated to larger sizes.
- The first version of the tick ran test-gc.cx with `-heap-initial 0` in 18 to
  26 seconds instead of 0.7 to 2.2. `-heap-initial 0` sets `INIT_HEAP_SIZE` to
  0, so every collection shrank the heap with a copy of all of memory and the
  next allocation grew it with another. `ResizeMemory` now shrinks by
  reslicing and copies only when that frees more than half of the array.
  After the fix test-gc.cx ran in 484 to 654 ms against 694 to 886 ms on
  base, and test-gc-nested.cx in 226 to 268 ms against 206 to 236 ms.

## Hazards removed

- An opcode that read an address and then allocated could see the object
  move under it. Nothing collects inside an opcode now.
- `types.Write_obj` and `types.AllocWrite_obj_data` wrote through the
  `memory` slice the caller passed before the allocation, so after a heap
  growth they wrote past the end of memory. `-heap-initial 0 test-regexp.cx`
  failed with `index out of range` and now runs. They are gone, with the
  `types.Allocator` hook they called. `regexp.Compile` and the two cxfx
  callers allocate through t3-ast with the current `prgrm.Memory`.
- `Callback` no longer lets a collection run inside the native opcode that
  called it.

## Reference counting by type

Reference counting by type would increment the count of the object a value
of a GC type points to when it is stored and decrement the one it replaces.
Values whose `GCType` is nil, atomic values and arrays of them, would cost
nothing. It is not implemented, and the choice is open.

| | Mark and compact tick | Reference counting by type |
| --- | --- | --- |
| Cost per store of a pointer, slice or str | none | an increment and a decrement, plus a check for 0, at every store into a frame slot, global, field or element of a GC type, and a decrement of every GC typed slot when a frame is popped |
| Cost per collection | proportional to the live objects and the references they hold | none, objects are freed when their count reaches 0 |
| Cycles | collected | leaked. `test-gc-nested.cx` builds a ring of pointers, so a tracing collector stays as a backup |
| Allocator | bump, because compaction keeps the heap contiguous | needs a free list or size classes, as freed objects leave holes |
| Object header | mark byte, forwarding address, size | a count as well |
| Pause | one collection per tick that passes the limit | none, but freeing a large structure frees it all at once |
| vm, zero allocation rule | no change to the hot loop | no Go allocation, but every move of a GC typed operand needs its own opcode with the count updates, in the loop whose timing moves with code placement (#470) |
| Freeing native resources | at the next collection | at once |

Counting by type keeps the collector out of every value that holds no
reference, which is most of what the benchmarks move. It adds work to every
store of one that does, and it still needs the tracing collector for cycles.
The tick costs nothing per store and two loads per step. A hybrid would count
references and keep the tick to collect cycles. The lead decides.

## The check

`make check-gcreach` runs [check-gcreach.sh](../check-gcreach.sh) as part of
`make check`. It fails when

- t2-cx-runtime-ir or t3-cx-runtime-vm imports t3-ast or t2-cx-runtime-gc,
- a package other than t3-ast imports t2-cx-runtime-gc, or
- a reach has no comment containing `GCREACH:` directly above it or in the
  doc comment of its function.

A reach is an import of t3-ast in t4-execute, t5-cx-runtime-core,
t4-cx-compiler-lower or t4-opcodes, a call of `GarbageCollectorCheck`,
`GarbageCollectorTrigger`, `GarbageCollectorCollect`, `AllocateSeq`,
`AllocateStr`, `AllocateStrAt`, `AllocateStrBytesAt`, `EscapedAllocate`, `SliceResizeEx`,
`ResizeMemory`, `gcFrameRoots` or `gcGlobalRoots`, or a use of `ListOfSlots`,
`ListOfPointers` or `Heap.Pressure`, `PressureLimit`, `Types`, `Roots` and
`Collector`. The collector package and t3-ast/gc_type.go are not scanned.
