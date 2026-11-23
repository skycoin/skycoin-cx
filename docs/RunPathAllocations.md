# Go allocations on the run path

## Purpose

The interpreter makes no Go heap allocation and starts no Go garbage
collection while it runs a program (#407). Everything a run needs is sized
before it starts. Allocations in CX memory (`AllocateSeq`) are program data
and are not Go allocations. This document lists the VMALLOC codes of #407
that mark the Go allocations the old loop can still make, and the tests
that keep the run free of the others.

## Before the run

`execute.RunPrepare`, which `ExecutorRun` calls before `RunCompiled`, sizes
the heap of program memory (`EnsureMinimumHeapSize`), the operand buffers to
the largest arity of an expression and the format buffer that print, printf,
sprintf and the str conversions append to (`RunBuffersPrepare`).
`RunCompiled` calls it as well, and it allocates nothing once the program is
prepared.

## Tests

- `TestRunAllocs` runs every `BenchmarkRun` program and the
  `t0-benchmarks/compare` programs, shrunk to a size that runs in
  milliseconds, with `testing.AllocsPerRun` around `RunCompiled` and requires
  0. test-gc.cx is the exception, see VMALLOC-010 and VMALLOC-011.
- `TestRunNoGC` collects, runs each program that allocates nothing, and
  requires `runtime.MemStats.NumGC` unchanged, also with `GOGC=1`.
- `TestRunVmAllocs` requires 0 for the vm.

## Codes

A `// VIOLATION VMALLOC-nnn:` comment above a function marks the Go
allocations in it. A failure branch that builds an error or panic text
allocates only when it fails, and is marked by its STRHOT E code instead
(see [StringsHotPath.md](StringsHotPath.md)).

| Code | Status | Where |
| --- | --- | --- |
| VMALLOC-001 | fixed in #422 | the expression copy in `CXCall.Call` |
| VMALLOC-002 | fixed for the run | operand buffers, sized by `RunBuffersPrepare`. A `Callback` loop of t6-cxfx still grows its own |
| VMALLOC-003 | open, boundary | `Read_str`, `Read_str_data`, `CXValue.Get_str`, Go string copies the system library and affordance opcodes use. The str opcodes read views |
| VMALLOC-004 | fixed | `fmt.Println` in the print opcodes |
| VMALLOC-005 | fixed but `%v` | printf and sprintf append to the format buffer. `%v` and extra arguments are VMALLOC-015 |
| VMALLOC-006 | fixed | the number to str conversions |
| VMALLOC-007 | fixed | `feedOSArgs` built an error when `os` is not imported |
| VMALLOC-008 | fixed | `rand.Seed`, a no-op since Go 1.24, removed |
| VMALLOC-009 | fixed for the run | `EnsureMinimumHeapSize`, called by `RunPrepare` |
| VMALLOC-010 | open | `ResizeMemory` grows program memory when the heap outgrows it. Needs a decision on a fixed cap |
| VMALLOC-011 | open | the CX collector builds its root table and type cache and grows its scratch in Go memory. test-gc.cx makes 90 allocations a run here and in VMALLOC-010 |
| VMALLOC-012 | open, boundary | `ReadSlice_*`, used by the serialize and file opcodes |
| VMALLOC-013 | not on the run path | `Make_obj` and `Write_str_data` run at compile time |
| VMALLOC-014 | failure only | the getters build an error only on a miss |
| VMALLOC-015 | open, printing | `GetPrintableValue` and the stack dump build Go strings |
| VMALLOC-016 | fixed | assert builds its text only when it fails |
| VMALLOC-017 | open, boundary | regexp compile and find |
| VMALLOC-018 | open, boundary | the file, json, process, profile, tcp and read opcodes |
| VMALLOC-019 | open, boundary | the affordance opcodes run compiler code |
| VMALLOC-020 | not heap | CXArgument stack temporaries, #408 |
| VMALLOC-021 | open, t6-cxfx | `Callback` runs a nested loop with its own buffers |
| VMALLOC-022 | open, once | a print longer than `ProgramFormatBufferSize` grows the format buffer once |
| VMALLOC-023 | open, boundary | the serialize opcodes run the serializer |

## Import depth

Measured with `go list` on the default build.

```
t4-execute -> t3-ast -> t2-cx-core-stringpool -> t1-cx-core-objmodel -> t0-cx-core-assert
t4-opcodes -> t3-ast, t3-cx-runtime-vm
t3-cx-runtime-vm -> t2-cx-runtime-ir -> t1-constants -> t0-types
```

The old loop is five levels deep and the vm four. t3-ast is tier 3
because of three tier 2 packages, the string pool, the collector and
t2-globals, so dropping t2-globals alone no longer lowers it, as it would
have when #411 was written. Each package name carries its tier, so any
flattening renames packages. It is left to #408, which separates the IR
definitions from the builders. Tiers cost nothing at run time, call depth
per instruction does.
