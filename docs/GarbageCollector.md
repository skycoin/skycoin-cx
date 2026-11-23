# Garbage collector

## Purpose

The interpreter and the IR base must not depend on the garbage collector.
This document lists every place where they reach it today, shows how the
packages depend on each other, and proposes the separation that makes the
collector a tick of the main loop. Each reach carries a two line `GCREACH:`
comment in the code, and `make check-gcreach` fails when a new one appears
without it.

## The collector today

The heap is the part of `CXProgram.Memory` after the stack and the data
segment, described by `CXProgram.Heap` (`HeapSegmentStruct` in
t3-ast/ast_cxprogram.go). Objects are allocated by bumping `Heap.Pointer`.

`MarkAndCompact` (t3-ast/garbage_collector.go) marks every object reachable
from the roots, gives each live object a new address, rewrites the references
it recorded while marking, and slides the objects down. The roots are

- the globals, with the `GCType` of each built by `gcGlobalRoots`, and
- the frame of every call on `CallStack`, with the roots of each function
  built by `gcFrameRoots` from its `ListOfSlots` and `ListOfPointers`.

Both live in t3-ast/gc_type.go and are cached in `Heap.Types`.

The collector runs inline. `AllocateSeq` calls `MarkAndCompact` whenever a
request does not fit, then grows or shrinks the heap. Every allocation in
every opcode can therefore move every heap object while Go code in the middle
of that opcode holds addresses. `AllocateStrAt` works around this for one
address with `Heap.Pinned`.

## Which packages are GC free

| Package | Imports t3-ast | Reaches the GC |
| --- | --- | --- |
| t2-cx-runtime-ir, the IR base | no | no |
| t3-cx-runtime-vm, the vm executor | no | no. The machine has no heap |
| t4-cx-compiler-lower | yes | no. It reads the program and rejects heap operands (`ReasonOperandHeap`) |
| t4-execute, the old loop | yes | yes, through every opcode it runs and `feedOSArgs` |
| t4-opcodes | yes | yes, 17 of its 21 files allocate |
| t6-cx-runtime-core, the executor | yes | yes, through the old loop |

`make check-gcreach` fails when t2-cx-runtime-ir or t3-cx-runtime-vm imports
t3-ast. A program that uses the heap never lowers, so `--executor=auto` runs
it on the old loop.

## Reach list

Lines are as of this document. Regenerate them with `./check-gcreach.sh -l`.

### Imports of t3-ast by the interpreter and the IR base

- t4-cx-compiler-lower/lower.go:20, lower_printf.go:12. Reads only
- t4-execute/execute.go:11, callback.go:9
- t6-cx-runtime-core/core.go:15, executor.go:12
- t4-opcodes, allocating: op_aff.go:11, op_f32.go:11, op_f64.go:11,
  op_fmt.go:10, op_i8.go:10, op_i16.go:10, op_i32.go:10, op_i64.go:10,
  op_serialize.go:8, op_slice.go:8, op_stdin.go:10, op_str.go:12,
  op_testing.go:10, op_ui8.go:11, op_ui16.go:11, op_ui32.go:10, op_ui64.go:10
- t4-opcodes, not allocating: op_bool.go:8, op_misc.go:7, opcodes.go:11,
  opcodes_init.go:8

### Allocation that may collect

- t3-ast/ast_memory.go:78 `AllocateSeq` calls `MarkAndCompact`, and :102 and
  :113 resize the heap after it
- t3-ast/ast_memory.go:132 `AllocateStr`, :142 to :144 `AllocateStrAt` and
  `Heap.Pinned`
- t3-ast/ast_cxvalue.go:210 `CXValue.Set_str`, the path of every opcode that
  writes a str
- t3-ast/mem_escape.go:57 `EscapedAllocate`, and its caller
  t3-ast/ast_cxcall.go:96 in `wipeDeclarationMemory`
- t3-ast/mem_slice.go:104 `SliceResizeEx`, :152 `SliceResize`, :271
  `WriteToSlice`
- t4-execute/execute.go:84 `feedOSArgs`
- t4-opcodes/op_aff.go, 27 lines in `queryParam`, `QueryExpressions`,
  `getSignatureSlice`, `queryStructsInPackage`, `QueryFunction`,
  `QueryCaller`, `QueryProgram` and `opAffQuery`
- t4-opcodes/op_serialize.go:24 `opSerialize`
- t4-opcodes/op_str.go:159 `opStrBytesOfString`
- t0-types/types.go:149 the `Allocator` hook, set by t11-cx/main.go:94 and
  t7-cx-core-run/main.go:27, called by t0-types/types_object.go:60 and :81
  and types_string.go:8
- t6-cxfx/event.go:64, op_glfw_desktop.go:237, op_openal.go:47,
  op_opengl.go:464

Through these, 90 functions reach `AllocateSeq`: 33 in t4-opcodes, 16 in
t5-cxos, 12 in t6-cxfx, 11 in t3-ast, 6 in t4-execute, 3 in t5-tcp, 2 in
t5-regexp, 2 in t10-cxrepl, 2 in t11-cx, 2 in t7-cx-core-run and 1 in
t6-cx-runtime-core. They are not marked one by one.

### GC root tables

- t3-ast/cxfunction_def.go:48 `ListOfPointers` and :57 `ListOfSlots`
- t4-actions/gcroot.go:31 adds to `ListOfSlots`, t4-actions/functions.go:1223
  to :1308 reads and adds to `ListOfPointers`
- t3-ast/serialize.go:736 to :751 writes and :807 to :831 reads both tables

## Dependencies

```
t11-cx, t7-cx-core-run ----- set types.Allocator = ast.AllocateSeq
        |
t6-cx-runtime-core
   |            \
   |             t4-cx-compiler-lower --> t2-cx-runtime-ir   (GC free)
   |                     |                     ^
   |                     |                     |
   |             t3-cx-runtime-vm -------------+             (GC free)
   |
t4-execute (old loop) --> t4-opcodes, t5-*, t6-cxfx
   |                          |
   +--------------------------+--> t3-ast
                                     AllocateSeq --> MarkAndCompact
                                                        |
                                      gcGlobalRoots, gcFrameRoots
                                      ListOfSlots, ListOfPointers
                                      Heap.Types, Heap.Pinned
t0-types: Allocator hook --> AllocateSeq (through the function variable)
```

## Hazards of the inline collection

- An opcode that allocates twice, or reads an address and then allocates,
  holds an address the collection may move. `Heap.Pinned` covers one such
  address in `AllocateStrAt` and nothing else.
- `types.Write_obj` and `types.AllocWrite_obj_data` write through the
  `memory` slice the caller passed before the allocation. When the
  allocation grows `prgrm.Memory`, the str and the pointer to it go to the
  old array and are lost. `regexp.Compile` and two cxfx opcodes use them.
- `Callback` runs a nested old loop inside a native opcode, so a collection
  can run while that opcode holds addresses.

## Proposed separation

1. The allocator only allocates. It bumps `Heap.Pointer`, grows the heap when
   a request does not fit, and records the pressure, the bytes allocated
   since the last collection. When the heap cannot grow past
   `MAX_HEAP_SIZE` it raises the CX runtime error
   `CX_RUNTIME_HEAP_EXHAUSTED_ERROR`.
2. The main loop owns the collection. Between two steps of the old loop every
   root is in a frame slot or a global, so that is the safe point.
   `GarbageCollectorCheck` compares the pressure with a threshold and
   `GarbageCollectorTrigger` collects when it says so.
3. The collector moves to its own package below t3-ast. It sees a root table,
   the frame pointer and roots of each live frame and the global roots, and
   the heap. It imports no CX IR and no program state.
4. The `Allocator` hook goes. Its callers allocate through t3-ast with the
   current `prgrm.Memory`.
5. t2-cx-runtime-ir and t3-cx-runtime-vm keep importing neither t3-ast nor
   the collector, which `make check-gcreach` enforces.

## The check

`make check-gcreach` runs [check-gcreach.sh](../check-gcreach.sh) as part of
`make check`. A reach is

- an import of t3-ast in t4-execute, t6-cx-runtime-core,
  t4-cx-compiler-lower or t4-opcodes,
- a call of `MarkAndCompact`, `AllocateSeq`, `AllocateStr`, `AllocateStrAt`,
  `EscapedAllocate`, `SliceResizeEx`, `ResizeMemory`, `gcFrameRoots`,
  `gcGlobalRoots`, `AllocWrite_obj_data` or `AllocWrite_str_data`, or any use
  of `Allocator`,
- any use of `ListOfSlots`, `ListOfPointers` or the GC state of the heap,
  `Heap.Pinned`, `Types`, `Visited`, `Slots`, `Work` and `ObjectStarts`.

The collector's own files, t3-ast/garbage_collector.go and gc_type.go, are
not scanned. A reach needs a comment containing `GCREACH:` directly above it,
or in the doc comment of its function.
