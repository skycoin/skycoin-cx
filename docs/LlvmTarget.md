# The LLVM target

`cx build --target=llvm` compiles a program to LLVM IR and, with clang, to a
native executable. It compiles the image vm runs, so it takes exactly the
programs vm takes (decision D8), and a native run prints what vm prints.

```
./bin/cx build --target=llvm hello.cx               # hello.ll and hello
./bin/cx build --target=llvm hello.cx -o hello     # the same
./bin/cx build --target=llvm hello.cx -o hello.ll  # hello.ll alone
```

An output ending in `.ll` gets the IR alone, for reading, for `opt` and `llc`,
or for another target triple. Any other output also gets the IR beside it,
`.ll` in place of `.exe`, and the executable that clang compiles from it and
the C runtime (decision D9). `CX_CLANG` names the clang to run, `clang` on
the `PATH` by default. `--llvm-stats` prints how each scalar opcode of the
program was emitted. `--llvm-debug` adds DWARF debug information: a subprogram
per function, the source line of each instruction, and the name and type of
each variable kept in a register, from `ImageLocals`, so a debugger such as
gdb or lldb shows CX names.

Without a clang, or with `--llvm-linker=tinygo`, cx build links with
TinyGo instead, and `--llvm-tinygo-target=pico` cross compiles for a TinyGo
target, see [Microcontrollers.md](Microcontrollers.md#the-llvm-target-with-tinygo). TinyGo builds
with `-no-debug` unless `--llvm-debug` is given, since its DWARF made a wasip1
fib 1.34 MB instead of 74 KB and did not change the speed. `CX_TINYGO_FLAGS`
adds flags after it, so `CX_TINYGO_FLAGS=-no-debug=false` keeps the DWARF.

## What it compiles

`t3-cx-compiler-llvm` reads the `ir.Image` that `lower.Program` makes, see
[IR.md](IR.md#the-runtime-image), and its three side tables. It is pure Go
and links no LLVM.

* Each function of the image becomes an LLVM function that takes the frame
  pointer, the call depth and its inputs, and returns its outputs.
* Memory is one global, `cx_mem`, laid out as the vm memory: the stack, the
  data segment, then the scratch. The data segment starts as the image data.
* A frame variable of `ImageLocals` whose address is never taken becomes an
  alloca, which LLVM keeps in registers, named after the variable. A variable
  whose address is taken stays in `cx_mem` at its vm offset from the frame
  pointer, so every pointer keeps its vm value and the address checks of vm
  hold. A byte no variable covers whose address is taken keeps the whole
  frame in memory.
* A load of a literal of `ImageReadOnly` becomes a constant.
* Every check of vm stops the native run the same way: the call depth and
  stack size of a call, an index outside its array, an address outside the
  stack and data segment, and an integer division by zero. An index check is
  one compare and one branch: a mask when the length is a power of two, an
  unsigned compare with the length otherwise, with a comment in the IR
  naming the check (optimizations 0006 and 0007). The run prints the line vm
  prints and exits with its exit code.

## Templates and the runtime

Each scalar opcode has a runtime function in the C runtime,
`cx_rt_i32_add` for `i32.add`, which computes what vm computes. An opcode
may also have a template, inline LLVM IR, and then compiles to the template
(decision D10). `llvmTemplate` in `t2-cx-core-opcodegen` holds the templates,
and `make generate-opcodes` writes them into the opcode table of the target
with the type and operation of each opcode, from its row of
`OpcodeDefinitionTable`. Adding a template moves one opcode to inline code
and changes nothing else.

279 of the 296 scalar opcodes have a template. The 17 without are the 11
prints and the float mod, max and min, which follow `math.Mod`, `math.Max`
and `math.Min`. A template keeps Go semantics where LLVM leaves a case
undefined or poison: a shift by the width or more, `MinInt / -1` and the
modulo of it. A float out of the range of the integer it converts to is left
to the platform by Go, and `freeze` gives some value.

`TestLlvmScalarOpcodes` runs every scalar opcode on edge operands on vm and
natively, by its template and by its runtime function, with the operands in
memory and as folded literals, and checks that all print the same.

The C runtime, `runtime.c.txt`, also prints, runs `test` and stops. Its
output buffer is static, as the vm output buffer is sized before the run, and
nothing in it allocates. cx build compiles it once per clang into the user
cache and links the object with each program.

## What it refuses

* A program outside the vm subset, with the message vm gives:
  `cx build: cannot run on vm: opcode outside subset append in main`.
* A program that calls `os.Run` of the process library, which is still
  called through `LibraryCall` (795). It never lowers, as it takes and
  returns a `str`, so vm refuses it first.

## System libraries

A function of a system library that is its own opcode and runs on vm, such
as `cipher.HashStateNew` (decision D119), has a native form: a C function
of `runtime.c.txt` that takes the block at sp, its inputs then its outputs,
as the vm form does, `cx_rt_cipher_hash_state_new` and so on. It is
compiled into the executable with the rest of the runtime, so nothing is
loaded from outside. A program that uses a library the build does not
enable becomes an executable that prints the refusal cx prints and exits
with `CX_RUNTIME_LIBRARY_DISABLED` (12).

The terminal functions read the `argc` and `argv` of `main`, which the
entry passes to `cx_rt_start`, and standard input with `getchar`. Linked
by TinyGo they call back into the Go side for `os.Args` and `os.Stdin`.

`OpcodeOutputHandOff` and `OpcodeOutputFlush` call `cx_rt_output_hand_off`
and `cx_rt_output_flush`. For `--output-mode=thread` cx build compiles the
runtime with `CX_RT_OUTPUT_THREAD` and `-pthread`, and the hand off copies
the output to a ring that a POSIX thread writes. On Windows and with
`--llvm-linker=tinygo` the runtime has no second thread and writes at the
hand off, as `instant` does.

## Differences from vm

After a runtime error vm prints the call stack with the value of every
variable, read from its memory. A native program keeps most variables in
registers, so it prints the error line alone. The execution records check
the native output up to the call stack.

## Execution records

The `native` form of [ExecutionRecords.md](ExecutionRecords.md) builds every
record that lowers with `cx build --target=llvm` and runs it. It needs clang,
and on a machine without one the harness lists the form as not run with the
reason.

## Performance

`make bench-compare` times the native form as `cx-native`. On one core of an
x86-64 desktop with clang 23.1.4, net milliseconds, the median of 5 runs:

| benchmark | cx old loop | cx vm | cx native | C -O2 | native over C |
|---|---:|---:|---:|---:|---:|
| count | 19631 | 819 | 0 (folded) | 0.1 (folded) | - |
| fib | 13442 | 1102 | 28.2 | 23.9 | 1.2 |
| array | 10844 | 862 | 6.6 | 4.9 | 1.3 |
| struct | 10540 | 951 | 2.4 | 2.0 | 1.2 |
| memtest | 24034 | 1738 | 9.8 | 15.8 | 0.6 |
| primes | 12066 | 740 | 55.5 | 57.3 | 1.0 |
