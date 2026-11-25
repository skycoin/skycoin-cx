# The CX intermediate representation

This document describes what the CX compiler produces. That is the
in-memory program the VM executes, a `*ast.CXProgram` from `t3-ast`, and
the byte form `ast.SerializeCXProgram` writes. Every type, field and
function named here is in this repository, and the worked example was
produced by compiling a real program and dumping it.

Opcode numbers are fixed by `constants.OpcodeLifecycleTable`, so the
numbers quoted below are the same in every build. See [Opcodes](#opcodes).

## Contents

* [Pipeline](#pipeline)
* [The program](#the-program)
* [Packages, functions and structs](#packages-functions-and-structs)
* [Expressions](#expressions)
* [Arguments and type signatures](#arguments-and-type-signatures)
* [Memory layout](#memory-layout)
* [How the VM runs an expression](#how-the-vm-runs-an-expression)
* [Worked example](#worked-example)
* [Serialized form](#serialized-form)
  * [Affordances](#affordances)
  * [Built programs](#built-programs)
* [Opcodes](#opcodes)
* [The runtime image](#the-runtime-image)
* [Known oddities](#known-oddities)

## Pipeline

The `cx` command (`t11-cx/main.go`) builds the program in this order.

1. `actions.InitConstants`, `cxos.InitConstants` and `cxfx.InitConstants`
   fill the constant tables.
2. `cxinit.MakeProgram` (`t7-init/init.go`) returns an empty
   `*ast.CXProgram` and stores it in the package variable `actions.AST`.
   Every later stage writes into `actions.AST`.
3. `cxinit.RegisterPackages` adds the native packages and registers the
   system libraries `cipher`, `cxos` and `tcp`, and sets `ast.OpcodeDispatch`
   to the opcode switch. The opcodes are fixed when cx is compiled: their
   names, handlers and signatures are the rows of
   `constants.OpcodeDefinitionTable`, and `ast.NativeNames` is built from
   the generated `constants.NativeNameTable` (decision D21). The compiler finds a built-in by the name Gids of
   its parts with `CXProgram.CXProgramNativeFind`, which compares the bytes
   of a name with the sorted names of `ast.NativeNames` once and caches the
   result by Gids in `CxIrState.CxIrStateNativeNames`. Constants such as
   `cx.SUCCESS` are found the same way in `actions.ConstNames`.
4. `compileSourceStrings` (`t11-cx/helpers.go`), which the cx command and
   the tests share, calls `cxparsing.ParseSourceStrings`, which runs the
   two passes below.
5. `cxparsing.AddInitFunction` (`t9-cxparsing/utils.go`) adds the function
   `*init` (`constants.SYS_INIT_FUNC`) to package `main`. Its body is
   `CXProgram.SysInitExprs`, the global initializers collected by
   `actions.DeclareGlobalInPackage`.
6. `execute.RunCompiled` (`t4-execute/execute.go`) calls
   `CXProgram.EnsureMinimumHeapSize`, runs `*init`, pushes `main` onto the
   call stack and runs it.

### Pass 1, declarations

`cxparsing.Preliminarystage` (`t9-cxparsing/utils.go`) first scans every
file with regular expressions for `package`, `type`, `var` and `import`
lines. It then calls `cxparsing.Passone` for each file, which calls
`cxpartialparsing.Parse` (`t5-cxpartialparsing`, generated from
`cxpartialparsing.y`). The pass 1 grammar only calls
`actions.DeclarePackage`, `actions.DeclareImport`, `actions.DeclareStruct`,
`actions.DeclareGlobalInPackage` and the `actions.DeclarationSpecifiers*`
helpers. After pass 1 the program holds every package, import, struct and
global, but no function.

### Pass 2, function bodies

`ParseSourceStrings` then calls `cxparsing.Passtwo` for each file, which
calls `cxparsingcompletor.Parse(cxparsingcompletor.NewLexer(b))`
(`t8-cxparsingcompletor`, generated from `parsingcompletor.y`). The grammar
actions call into `t4-actions`. They build expression lists bottom up, for
example `actions.OperatorExpression`, `actions.PostfixExpressionFunCall`,
`actions.SelectionExpressions`, `actions.IterationExpressions`,
`actions.WritePrimaryExprs` and `actions.Assignment`.

A function is finished by `actions.FunctionHeader`, which adds the
`CXFunction`, and `actions.FunctionDeclaration`
(`t4-actions/functions.go`), which

* adds the parameters with `FunctionAddParameters`,
* resolves `goto` targets with `ProcessGoTos`,
* appends the expressions with `AddExprsToFunction`,
* gives every parameter and local a frame offset through
  `ProcessFunctionParameters`, `ProcessExpressionArguments` and
  `UpdateSymbolsTable`, renaming each local to `<n>:<name>`,
* type checks each expression (`CheckTypes`, `checkOperatorOperands`) and
  replaces untyped operators with typed ones in `ProcessTypedOperator`,
  so `+` on `i32` becomes `i32.add`,
* sets `CXFunction.LineCount = len(fn.Expressions)` and `CXFunction.Size`
  to the frame size.

## The program

`ast.CXProgram` (`t3-ast/cxprogram_def.go`) is the root of the IR.

| Field | Type | Contents |
| --- | --- | --- |
| `Stack`, `Data`, `Heap` | `StackSegmentStruct`, `DataSegmentStruct`, `HeapSegmentStruct` | Segment bounds in `Memory`, see [Memory layout](#memory-layout) |
| `Packages` | `map[string]CXPackageIndex` | Package name to index in `CXPackages` |
| `ProgramInput` | `[]CXArgumentIndex` | OS arguments |
| `Memory` | `[]byte` | Stack, data and heap segments |
| `CallStack` | `[]CXCall` | `constants.CALLSTACK_SIZE` (1000) frames, outside `Memory` |
| `CallCounter` | `types.Pointer` | Index of the running frame in `CallStack` |
| `Terminated` | `bool` | Set by `popStack` when `main` returns |
| `Version` | `int64` | `constants.LanguageVersionCurrent`, see [Serialized form](#serialized-form) |
| `ProgramLibrariesUsed` | `[constants.LibraryCount]bool` | System libraries the program calls, by library number |
| `ProgramLibrariesEnabled` | `[constants.LibraryCount]bool` | System libraries the run may call, from `--enable` |
| `CurrentPackage` | `CXPackageIndex` | Package being parsed, or the REPL's package |
| `SysInitExprs` | `[]CXExpression` | Global initializers, the body of `*init` |

The rest of the program lives in flat slices, called arenas here. IR
objects refer to each other by index into these slices, as integer types
such as `CXFunctionIndex`.

| Arena | Element | Index type | Getter |
| --- | --- | --- | --- |
| `CXPackages` | `CXPackage` | `CXPackageIndex` | `GetPackageFromArray` |
| `CXFunctions` | `CXFunction` | `CXFunctionIndex` | `GetFunctionFromArray` (nil for -1) |
| `CXStructs` | `CXStruct` | `CXStructIndex` | direct index |
| `CXAtomicOps` | `CXAtomicOperator` | `int` | `GetCXAtomicOp` |
| `CxIrState.CXLineMng`, was `CXLines` | `CXLine` | `int`, the slot index | `GetCXLine` |
| `CXArgs` | `CXArgument` | `CXArgumentIndex` | `GetCXArg`, `GetCXArgFromArray` |
| `CXTypeSignatures` | `CXTypeSignature` | `CXTypeSignatureIndex` | `GetCXTypeSignatureFromArray` |
| `CxIrState.CXTypeSignature_ArrayMng`, was `TypeSignatureForArrays` | `CXTypeSignature_Array` | `int`, the slot index | `GetCXTypeSignatureArrayFromArray` |
| `CxIrState.CXTypeSignature_StructMng`, was `TypeSignatureForStructs` | `CXTypeSignature_Struct` | `int`, the slot index | `GetCXTypeSignatureStructFromArray` |

Several fields still hold Go pointers instead of indexes. Each is marked
in the source and becomes a generational index under the object model in
[ObjectModel.md](ObjectModel.md), tracked in
[#299](https://github.com/skycoin/skycoin-cx/issues/299). They are
`CXPackage.Globals`, `CXFunction.Inputs`, `CXFunction.Outputs`,
`CXFunction.ListOfPointers`, `CXAtomicOperator.Inputs`,
`CXAtomicOperator.Outputs`, `CXArgument.StructType`,
`CXArgument.ArgDetails`, `CXTypeSignature.ArgDetails`,
`CXTypeSignature_Struct.StructType` and `CXCall.Operator`.

## Packages, functions and structs

`ast.CXPackage` (`t3-ast/cxpackage_def.go`) has a `Name`, its own `Index`
in `CXPackages`, and three name maps, `Imports` (to `CXPackageIndex`),
`Functions` (to `CXFunctionIndex`) and `Structs` (to `CXStructIndex`).
`Globals` is a `*CXStruct` whose `Fields` are the global variables.
`CurrentFunction` is the function being parsed.

`ast.CXFunction` (`t3-ast/cxfunction_def.go`) has

* `Index`, `Name` and `Package`,
* `AtomicOPCode`, the opcode number, 0 for a user function.
  `CXFunction.IsBuiltIn` is `AtomicOPCode != 0`,
* `LibraryNumber` and `LibraryFunctionNumber`, the system library function
  that an `AtomicOPCode` of `constants.OP_LIBRARY_CALL` (795) calls, 0
  otherwise,
* `Inputs` and `Outputs`, each a `*CXStruct` whose `Fields` are
  `CXTypeSignatureIndex` values for the parameters,
* `Expressions []CXExpression`, the body,
* `LineCount`, equal to `len(Expressions)`,
* `Size types.Pointer`, the frame size in bytes, parameters included,
* `FileName`, `FileLine`, `FileColumn` and `LocalVariables`, used while compiling,
* `ListOfPointers []*CXArgument` and `ListOfSlots []CXFunctionSlot`, the GC
  roots of the frame. A slot is a local with its declared type, and the GC
  walks it by the `ast.GCType` that `ast.GCTypeOf` builds from that type
  (`t3-ast/gc_type.go`).

`CXFunctions` holds more than the user's functions. Every expression that
calls a native operator gets its own copy of the native, made by
`CXProgram.AddNativeFunctionInArray` from `ast.NativeFunctionOf(prgrm, opcode)`,
which makes it from `constants.OpcodeSignatureTable`. The copy has
`AtomicOPCode` set and no `Expressions`. A system library function is the
native `ast.NativeFunctionOf(prgrm, ast.LibraryFunctionKey(library, function))`
makes from `constants.LibraryFunctionSignatureTable`, a negative key that
is also its code in `ast.NativeNames`. Its copy has `AtomicOPCode`
`OP_LIBRARY_CALL` and the two library numbers, and adding it marks the
library in `CXProgram.ProgramLibrariesUsed`.

`ast.CXStruct` (`t3-ast/cxstruct_def.go`) has `Index`, `Name`, `Package`,
`Fields []CXTypeSignatureIndex`, and `FileName`, `FileLine` and
`FileColumn`, the position of a declared type name, used while compiling.
The same type is reused as the parameter list of a function or operator and
as a package's globals.

## Expressions

A function body is a flat `[]CXExpression`. There are no nested blocks.
Scopes, branches and loops become jumps over this list.

`ast.CXExpression` (`t3-ast/cxexpression_def.go`) has three fields.

| Field | Type | Meaning |
| --- | --- | --- |
| `Type` | `CXOPERATION_TYPE` | `CX_ATOMIC_OPERATOR` (1) or `CX_LINE` (3) |
| `Index` | `int` | Index into `CXAtomicOps` for `CX_ATOMIC_OPERATOR`, the slot in `CxIrState.CXLineMng` for `CX_LINE` |
| `ExpressionType` | `CXEXPR_TYPE` | `CXEXPR_UNUSED` (0), `CXEXPR_METHOD_CALL` (1), `CXEXPR_STRUCT_LITERAL` (2), `CXEXPR_ARRAY_LITERAL` (3), `CXEXPR_SCOPE_NEW` (4), `CXEXPR_SCOPE_DEL` (5) |

`CXOPERATION_TYPE` (`t3-ast/ast_cxoperation.go`) also defines `UNUSED`
(0) and `CX_ARGUMENT` (2), which no expression uses. `ExpressionType`
only matters while compiling. `CXEXPR_SCOPE_NEW` and `CXEXPR_SCOPE_DEL`
open and close a symbol scope in `FunctionDeclaration`.

### Line markers

A `CX_LINE` expression points at an `ast.CXLine`
(`t3-ast/cxline_def.go`) with `FileName`, `LineNumber`, `LineColumn` and
`LineStr`. `ast.MakeCXLineExpression` creates them. The parser puts one before each
statement, and error messages use the nearest preceding marker. The VM
skips them, but they take a slot in `Expressions`, so jump distances
count them.

### Atomic operators

A `CX_ATOMIC_OPERATOR` expression points at an `ast.CXAtomicOperator`
(`t3-ast/cxatomicoperator_def.go`).

| Field | Type | Meaning |
| --- | --- | --- |
| `Operator` | `CXFunctionIndex` | What runs, see below |
| `Inputs`, `Outputs` | `*CXStruct` | `Fields` are the `CXTypeSignatureIndex` of each operand. Read them with `GetInputs(prgrm)` and `GetOutputs(prgrm)` |
| `Function` | `CXFunctionIndex` | The function this expression belongs to |
| `Package` | `CXPackageIndex` | The package of that function |
| `Label` | `string` | `goto` label, empty otherwise |
| `ThenLines`, `ElseLines` | `int` | Relative jump distances for `jmp`, `goto`, `break` and `continue` |

`Operator` takes one of three forms.

* `-1`. `GetFunctionFromArray` returns nil and the expression is a
  variable declaration. It has one output and no inputs, and the VM
  zeroes the output's bytes (`wipeDeclarationMemory`).
* A `CXFunction` with `AtomicOPCode != 0`. The expression is a native
  operation and `ast.OpcodeDispatch` runs `AtomicOPCode` through the generated
  switch.
  For `OP_LIBRARY_CALL` the loop first checks
  `CXProgram.ProgramLibrariesEnabled[LibraryNumber]`, one indexed load and
  compare, then calls the generated switch of the library,
  `ast.LibraryDispatches[LibraryNumber]`, with `LibraryFunctionNumber`.
* A `CXFunction` with `AtomicOPCode == 0`. The expression calls that user
  function.

Inputs are read and outputs written in order. A call `y = f(a, b)` has
inputs `a, b` and output `y`. An expression whose result goes nowhere has
no outputs, and nested calls get temporaries named `*tmp_<n>`
(`constants.LOCAL_PREFIX`).

### Jumps and labels

All control flow uses four opcodes, each with handlers in
`t4-opcodes/op_misc.go`.

| Opcode | Number | Handler | Effect on `CXCall.Line` before the VM adds 1 |
| --- | --- | --- | --- |
| `jmp` | `OP_JMP` = 2 | `OpJmp` | `+ ThenLines` if input 0 is true, `+ ElseLines` if false |
| `goto` | `OP_GOTO` = 13 | `OpGoto` | `+ ThenLines` |
| `break` | `OP_BREAK` = 14 | `OpBreak` | `+ ThenLines` |
| `continue` | `OP_CONTINUE` = 15 | `OpContinue` | `+ ThenLines` |

`CXCall.Call` adds 1 to `Line` after every native, so a jump at index `i`
with distance `n` continues at `i + n + 1`. `n = 0` falls through.

* `if` (`actions.SelectionExpressions`) emits the condition, a `jmp` on
  its result with `ThenLines = 0` and `ElseLines = len(then) + 2`, the
  then block, and a `jmp` on a literal `true` with
  `ThenLines = len(else)` that skips the else block.
* `for` (`actions.IterationExpressions`) emits a `jmp` at the top that
  leaves the loop when the condition is false, and a `jmp` on `true` at
  the bottom with a negative `ThenLines` back to the condition.
  `break` and `continue` get their distances there and become a `jmp` on
  `true`, so an enclosing loop leaves them alone. One that no loop resolved
  is a compile error. A literal or variable condition is read in place.
* `goto L` is an `OP_GOTO` expression whose `Label` is `L`, and the
  statement labelled `L` has the same `Label` on each expression that has
  none yet. `actions.ProcessGoTos`
  finds that expression at index `j` and sets `ThenLines = j - i - 1`.
* `return` (`actions.AddJmpToReturnExpressions`) is an `OP_GOTO` with a
  generated label `*lbl_<n>` and `ThenLines = types.MAX_INT32`, which
  moves `Line` past `LineCount` and so returns.

`OP_ABS_JMP` (3, `OpAbsJmp`) sets `Line` to `ThenLines` or `ElseLines`
directly, but the compiler never emits it.

## Arguments and type signatures

An operand is a `CXTypeSignature`. An older form, `CXArgument`, is still
used for many operands. A type signature can point at one.

### CXTypeSignature

`ast.CXTypeSignature` (`t3-ast/cxtypesignature_def.go`) has `Index`,
`Name`, `Offset`, `Type`, `Package`, `PassBy`
(`constants.PASSBY_VALUE` 0 or `PASSBY_REFERENCE` 1), `IsDeref`, `Meta`
and `ArgDetails`. `Type` is a `CXTypeSignature_TYPE`, and `Meta` depends
on it.

| `Type` | Value | `Meta` |
| --- | --- | --- |
| `TYPE_ATOMIC` | 1 | A `types.Code`, for example `types.I32` (4) |
| `TYPE_POINTER_ATOMIC` | 2 | The `types.Code` pointed at |
| `TYPE_ARRAY_ATOMIC` | 3 | Slot in `CxIrState.CXTypeSignature_ArrayMng` |
| `TYPE_POINTER_ARRAY_ATOMIC` | 4 | Slot in `CxIrState.CXTypeSignature_ArrayMng` |
| `TYPE_SLICE_ATOMIC` | 5 | Slot in `CxIrState.CXTypeSignature_ArrayMng` |
| `TYPE_POINTER_SLICE_ATOMIC` | 6 | Slot in `CxIrState.CXTypeSignature_ArrayMng` |
| `TYPE_STRUCT` | 7 | Slot in `CxIrState.CXTypeSignature_StructMng` |
| `TYPE_POINTER_STRUCT` | 8 | Slot in `CxIrState.CXTypeSignature_StructMng` |
| `TYPE_ARRAY_STRUCT` to `TYPE_SLICE_POINTER_COMPLEX` | 9 to 18 | Declared, not produced |
| `TYPE_CXARGUMENT_DEPRECATE` | 19 | Index into `CXArgs` |

`ast.CXTypeSignature_Array` holds `Meta` (the element `types.Code` or a
`CXTypeSignature_StructMng` slot), `Lengths []types.Pointer` (one entry per
dimension, 0 for a slice dimension) and `Indexes`, the index operands when
the signature reads an element. `ast.CXTypeSignature_Struct` holds
`StructType` and `Fields`, the field access chain.

`ast.GetCXTypeSignatureRepresentationOfCXArg` converts a `CXArgument` to
the most specific signature it can. Anything it cannot express, such as
a struct field access, a slice of structs or a function value, becomes
`TYPE_CXARGUMENT_DEPRECATE`.

### CXArgument

`ast.CXArgument` (`t3-ast/cxargument_def.go`) is the general operand. The
fields the VM reads are

* `Type` and `PointerTargetType`, both `types.Code`,
* `Size` and `Offset`, both `types.Pointer`,
* `DeclarationSpecifiers []int`, the type built from the left, with
  `constants.DECL_POINTER` (1), `DECL_DEREF` (2), `DECL_ARRAY` (3),
  `DECL_SLICE` (4), `DECL_STRUCT` (5), `DECL_INDEXING` (6),
  `DECL_BASIC` (7) and `DECL_FUNC` (8). For example `[5][]*Point` is
  built from these,
* `DereferenceOperations []int`, the accesses applied to the value, with
  `constants.DEREF_ARRAY` (1), `DEREF_FIELD` (2), `DEREF_POINTER` (3),
  `DEREF_DEREF` (4) and `DEREF_SLICE` (5),
* `Lengths`, `Indexes` and `Fields` for arrays, element access and field
  access,
* `Inputs` and `Outputs` for a function value,
* `StructType` and `PassBy`.

`ast.GetFinalOffset` (`t3-ast/mem_offset.go`) turns an operand and a frame
pointer into an absolute address in `Memory`, applying the dereferences.

### Type codes

`types.Code` (`t0-types/types.go`) numbers the base types. `UNUSED` 0,
`BOOL` 1, `I8` 2, `I16` 3, `I32` 4, `I64` 5, `UI8` 6, `UI16` 7, `UI32` 8,
`UI64` 9, `F32` 10, `F64` 11, `POINTER` 12, `STR` 13, `ARRAY` 14,
`SLICE` 15, `STRUCT` 16, `FUNC` 17, `AFF` 18, `UNDEFINED` 19,
`IDENTIFIER` 20. `Code.Size` gives the byte size. `POINTER`, `STR`,
`ARRAY`, `SLICE` and `IDENTIFIER` are `types.POINTER_SIZE` bytes, a
string being a pointer to a heap or data object. `AFF` is
`types.AFF_SIZE`, 16 bytes, see [Affordances](#affordances).

## Memory layout

`CXProgram.Memory` is one byte slice.

```
0                    Data.StartsAt          Heap.StartsAt
|  stack             |  data                |  heap                     |
   Stack.Size bytes     Data.Size bytes        Heap.Size bytes
```

`cxinit.MakeProgram` allocates `constants.STACK_SIZE` plus
`ast.MinHeapSize()` bytes and sets `Stack.Size = constants.STACK_SIZE`,
`Data.StartsAt = constants.STACK_SIZE` and
`Heap.Pointer = constants.NULL_HEAP_ADDRESS_OFFSET`. `STACK_SIZE` defaults
to 1048576 and `INIT_HEAP_SIZE` to 2097152, and `-stack-size`,
`-heap-initial` and `-heap-max` change them. `Heap.Max` is the heap cap,
`MAX_HEAP_SIZE` (64M) or `-heap-max`, and `EnsureMinimumHeapSize` gives
`Memory` the capacity of the stack, the data and the cap once before the
run, so the heap grows without a Go allocation (D42). `Stack.StartsAt` stays 0.

* Stack. Each call gets a frame of `CXFunction.Size` bytes at
  `CXCall.FramePointer`. Parameters, then outputs, then locals and
  temporaries get offsets from 0 in declaration order. A local's
  `Offset` is relative to the frame.
* Data. `actions.WritePrimary` (`t4-actions/misc.go`) appends every
  literal and global to the data segment at compile time, grows
  `Data.Size` and moves `Heap.StartsAt` to `Data.StartsAt + Data.Size`.
  A literal's `Offset` is absolute.
* Heap. `ast.AllocateSeq` (`t3-ast/ast_memory.go`) allocates objects.
  `Heap.Pointer` is relative to `Heap.StartsAt`, and the first
  `NULL_HEAP_ADDRESS_OFFSET` bytes are the `nil` address. Each object has
  a header of `types.OBJECT_HEADER_SIZE` bytes (`t0-types/types_object.go`),
  a 1 byte GC mark, a forwarding address and the object size.

`GetFinalOffset` tells a frame offset from an absolute one by value. An
offset below `Stack.Size` gets the frame pointer added, anything else is
used as it is.

### Pointer size

`types.Pointer` is `uint32` and `types.POINTER_SIZE` is 4 in the default
build (`t0-types/types_ptr32.go`). With `-tags ptr64` it is `uint64` and 8
(`t0-types/types_ptr64.go`). The pointer size changes the size of every
pointer, string, slice and array header, the object header, and with them
every offset after such an operand. `make check` runs the tests in both
builds.

## How the VM runs an expression

`ast.CXCall` (`t3-ast/cxcall_def.go`) is a frame with `Operator`
(`*CXFunction`), `Line` and `FramePointer`. `CXCall.Call` runs
`Operator.Expressions[Line]`.

1. If `Line >= Operator.LineCount`, `popStack` copies the callee's outputs
   into the caller's output operands, resets `Stack.Pointer` to the
   frame pointer and moves the caller to its next line.
2. A `CX_LINE` expression only increments `Line`.
3. A declaration zeroes its output and increments `Line`.
4. A native fills two `[]CXValue` with the operands' type signature,
   size, final offset and type in `processBuiltInOperators`, calls
   `OpcodeDispatch(prgrm, AtomicOPCode, inputs, outputs)`, and increments `Line`.
5. A user call (`processNonAtomicOperators`) increments `CallCounter`,
   sets the new frame at `Stack.Pointer`, adds `Operator.Size` to
   `Stack.Pointer`, zeroes the frame and copies the inputs into the
   callee's parameter offsets. A `PASSBY_REFERENCE` input is copied as its
   address.

## Worked example

This 10 line program was compiled at commit `64bd5701` in the default
build.

```
 1  package main
 2  func sum(a i32, b i32) (c i32) {
 3  	c = a + b
 4  }
 5  func main() {
 6  	var x i32 = sum(3, 4)
 7  	if x > 6 {
 8  		i32.print(x)
 9  	}
10  }
```

`cx -ast` prints the program with `ast.ToString`, which shows names and
types but not offsets or opcode numbers.

```
0.- Package: main
	Functions
		0.- Function: main () ()
			0.- Line: 6: var x i32 = sum(3, 4)
			1.- Declaration: 1:x i32
			2.- Expression: 1:x i32 = sum(*lit i32, *lit i32)
			3.- Line: 7: if x > 6 {
			4.- Expression: *tmp_0 bool = i32.gt(1:x i32, *lit i32)
			5.- Line: 9: }
			6.- Expression: jmp(*tmp_0 bool)
			7.- Line: 8: i32.print
			8.- Expression: i32.print(1:x i32)
			9.- Line: 9: }
			10.- Expression: jmp(*lit bool)
		1.- Function: sum (1:a i32, 2:b i32) (3:c i32)
			0.- Line: 3: c = a + b
			1.- Expression: 3:c i32 = i32.add(1:a i32, 2:b i32)
```

The listing below came from a throwaway test that runs the same steps as
`cx` (`InitConstants`, `MakeProgram`, `RegisterPackages`,
`ParseSourceStrings`, `AddInitFunction`) and walks
`CXFunction.Expressions`, `CXAtomicOps`, `CXLines` and `CXTypeSignatures`.

Segments after compiling. `Stack.Size` 1048576, `Data.StartsAt` 1048576,
`Data.Size` 13, `Heap.StartsAt` 1048589, `Heap.Pointer` 4. The data
segment holds the four literals.

| Address | Bytes | Literal |
| --- | --- | --- |
| 1048576 | `03 00 00 00` | `3` |
| 1048580 | `04 00 00 00` | `4` |
| 1048584 | `06 00 00 00` | `6` |
| 1048588 | `01` | `true`, made by `SelectionExpressions` |

`sum` is `CXFunctions[0]`, `Size` 12, `LineCount` 2. Parameters `1:a` at
frame offset 0 and `2:b` at 4, output `3:c` at 8, all `TYPE_ATOMIC` with
`Meta` 4 (`i32`).

| # | `Type` | Arena index | Operator | Inputs | Outputs | Then, Else |
| --- | --- | --- | --- | --- | --- | --- |
| 0 | `CX_LINE` | `CXLines[0]` | | line 3 | | |
| 1 | `CX_ATOMIC_OPERATOR` | `CXAtomicOps[3]` | opcode 383 `i32.add` | `1:a` @0, `2:b` @4 | `3:c` @8 | 0, 0 |

`main` is `CXFunctions[1]`, `Size` 5, `LineCount` 11. Local `1:x` (i32)
is at frame offset 0 and the temporary `*tmp_0` (bool) at 4.

| # | `Type` | Arena index | Operator | Inputs | Outputs | Then, Else |
| --- | --- | --- | --- | --- | --- | --- |
| 0 | `CX_LINE` | `CXLines[2]` | | line 6 | | |
| 1 | `CX_ATOMIC_OPERATOR` | `CXAtomicOps[7]` | -1, declaration | | `1:x` @0 | 0, 0 |
| 2 | `CX_ATOMIC_OPERATOR` | `CXAtomicOps[4]` | user function `CXFunctions[0]` `sum` | literal @1048576, literal @1048580 | `1:x` @0 | 0, 0 |
| 3 | `CX_LINE` | `CXLines[3]` | | line 7 | | |
| 4 | `CX_ATOMIC_OPERATOR` | `CXAtomicOps[10]` | opcode 389 `i32.gt` | `1:x` @0, literal @1048584 | `*tmp_0` @4 | 0, 0 |
| 5 | `CX_LINE` | `CXLines[5]` | | line 9 | | |
| 6 | `CX_ATOMIC_OPERATOR` | `CXAtomicOps[13]` | opcode 2 `jmp` | `*tmp_0` @4 | | 0, 4 |
| 7 | `CX_LINE` | `CXLines[4]` | | line 8 | | |
| 8 | `CX_ATOMIC_OPERATOR` | `CXAtomicOps[11]` | opcode 393 `i32.print` | `1:x` @0 | | 0, 0 |
| 9 | `CX_LINE` | `CXLines[6]` | | line 9 | | |
| 10 | `CX_ATOMIC_OPERATOR` | `CXAtomicOps[14]` | opcode 2 `jmp` | literal `true` @1048588 | | 0, 0 |

Every operand here is `TYPE_ATOMIC`. Offsets below 1048576 are frame
offsets and the others are data addresses.

When `x > 6` is false, the `jmp` at 6 adds `ElseLines` 4 and the VM adds
1, so `Line` becomes 11, which equals `LineCount`, and `main` returns.
When it is true, `Line` becomes 7 and the then block runs. The `jmp` at
10 has an empty else block to skip, so its `ThenLines` is 0.

`sum` and `i32.add` resolve differently although both are called the same
way. `sum` stays a user function call at index 2, and `+` becomes the
native `i32.add` in `ProcessTypedOperator`. The arenas end up with 8
`CXFunctions`, three user functions (`sum`, `main`, `*init`) and five
per-expression native copies.

## Serialized form

### API

`ast.SerializeCXProgram(prgrm, includeMemory, codec)`,
`ast.Deserialize(b, codec)`, `ast.SerializedHeaderCheck(b)` and
`ast.SerializeDebugInfo(prgrm, includeMemory, wholeMemory)` are in
`t3-ast/serialize.go`.

* A serialized program is a snapshot of the arenas and managers of a
  compiled program. `Deserialize` gives back a program that lowers and runs
  as the original does, on vm and on the old loop. The image-old and
  image-vm forms of `TestExecutionRecords` in `t11-cx` check this for every
  recorded program, see [ExecutionRecords.md](ExecutionRecords.md).
* With `includeMemory` the memory goes in too, from its first to its last
  nonzero byte. Without it `Deserialize` makes zeroed memory of the same
  size, so literals in the data segment read as zero.
* With a non-nil `ast.SerializeCodec`, such as a `*lz4codec.Codec`
  (`t0-lz4codec`), the memory always goes
  in and the result is one LZ4 block from `Codec.Compress`. The block does
  not store the original length, so `Codec.Decompress` relies on the limits
  set in `lz4codec.Config`. `Deserialize` must be given a codec when the
  bytes were compressed.
* `Deserialize` leaves `CompileErrorPrint` nil. The caller sets it, as
  `printer.PrinterCompileErrorsRegister` does.
* `Deserialize` returns an error, never panics, for bytes that are
  truncated or malformed, and for a program from another format, pointer
  size or language version.
* `SerializeDebugInfo` returns the byte size of the header and of each
  section. They add up to the length of the uncompressed bytes.

The CX functions `serialize(aff) []ui8` and `deserialize([]ui8)`
(`t4-opcodes/op_serialize.go`) serialize what an aff value selects, see
[Affordances](#affordances), and check that bytes hold a program this cx
can load. Bytes that do not are the runtime error
`CX_RUNTIME_INVALID_ARGUMENT`. Both take a slice, so they run on the old
loop only, and `cx --executor=vm` refuses a program that calls them with
`opcode outside subset serialize`.

### Affordances

An aff value selects what `serialize` writes (decision D5, #500). It is
`types.AFF_SIZE`, 16 bytes, four little endian i32 that `ast.AffordanceValue`
names: the kind, the package index, the item and the type signature index of
a variable, -1 where unused. The compiler writes the value of each aff
literal `#{...}` into the data segment once every function body is compiled
(`actions.AffordanceLiteralsResolve`), so no name is looked up at run time.
An aff value moves like any other value, on the old loop and on vm.

| Literal | Kind | serialize gives |
| --- | --- | --- |
| `#{}`, or the zero value | program | the whole program with its memory, which `deserialize` loads |
| `#{mem(data)}`, `#{mem(stack)}`, `#{mem(heap)}` | memory | the bytes in use of the segment, the stack up to `Stack.Pointer` and the heap up to `Heap.Pointer` |
| `#{pkg(p)}` | package | the header, then the package, its structs and its functions as the packages, structs and functions sections encode them |
| `#{pkg(p) fn(f)}` | function | the header, then the function |
| `#{pkg(p) strct(s)}` | struct | the header, then the struct |
| `#{pkg(p) var(v)}` | global | the bytes of the global |
| `#{pkg(p) fn(f) var(v)}` | local | the bytes of the parameter, result or local variable `v` in the innermost running call of `f` |

The header is laid out as the 24 bytes of a program header with the magic
`CXSELECT`, followed by the kind as a uvarint. `pkg` defaults to the package
of the literal. `deserialize` loads only a whole program, so the bytes of
any other selection are `CX_RUNTIME_INVALID_ARGUMENT`, as is a local of a
function with no running call.

### Encoding

The bytes start with a fixed 24 byte header, little endian.

| Bytes | Field | Value |
| --- | --- | --- |
| 0 to 7 | magic | `CXPROGRM` |
| 8 to 11 | format | `serializeFormatVersion`, now 7 |
| 12 to 15 | pointer size | `types.POINTER_SIZE`, 4, or 8 with the `ptr64` tag |
| 16 to 23 | language version | `CXProgram.Version` |

Nine sections follow, each a uvarint byte length and its bytes. Inside a
section a signed integer is a zigzag varint, an unsigned one or a
`types.Pointer` a uvarint, and a bool one byte. A slice or map is a varint
length, -1 for nil, and its elements. Maps are written in key order, so two
serializations of the same program are byte for byte equal
(`TestSerializeIsDeterministic`).

| Section | Contents |
| --- | --- |
| program | stack, data and heap bounds, the heap cap `Heap.Max`, `Packages`, `CurrentPackage`, `ProgramInput`, the call stack length, `CallCounter` and the entries in use, `Terminated`, `ProgramLibrariesUsed` and `ProgramLibrariesEnabled` as a count and one bool per library, `SysInitExprs` |
| strings | every string pool slot in order, whether the pool is frozen, the four `CxIrStateName*` Gids and `SourceFileMng` |
| packages | `CXPackages`, with imports, functions, structs and methods by name Gid, and the globals struct |
| structs | `CXStructs` |
| functions | `CXFunctions`, with `AtomicOPCode`, `LibraryNumber`, `LibraryFunctionNumber`, inputs, outputs, expressions, local variables, `ListOfPointers`, `EscapeInputs` and `ListOfSlots` |
| expressions | `CXAtomicOps` and `CXLineMng` |
| arguments | `CXArgumentMng` and `CXArgumentDebugMng` |
| type signatures | `CXTypeSignatureMng`, `CXTypeSignature_ArrayMng` and `CXTypeSignature_StructMng` |
| memory | the memory size, whether the bytes are included, and the offset and bytes from the first to the last nonzero byte |

Every index and Gid is written as it is in memory. The string pool is
interned again in slot order and a manager keeps its Cap, generations, live
flags, free list, `Next`, `Retired`, `Len` and growable flag, so each Gid
names the same object after `Deserialize`. Offsets keep
`constants.ESCAPE_OFFSET_FLAG`. A `*CXStruct` that names a struct of
`CXStructs`, such as `CXArgument.StructType`, `CXTypeSignature_Struct.StructType`
and `CXFunctionSlot.CXFunctionSlotStruct`, is written as its index and points
into `CXStructs` again. A struct that is owned through a pointer, such as
`CXFunction.Inputs` or `CXPackage.Globals`, is written in place. A call
stack entry writes the index of its function.

`SerializeCXProgram` fails when one of the managers no compile stage fills
yet, such as `CXFunctionMng`, holds an object. `TestSerializeFieldCounts`
in `t3-ast` fails when a type the serializer writes gains a field.

### Language version and pointer size

`Deserialize` checks the header with `SerializedHeaderCheck` before it
builds anything, and returns an error unless the format, the pointer size
and the language version are this cx's. The language version is the
integer `yymmddnnnn` of `constants.LanguageVersionCurrent`
(`t1-constants/constants.go`), and `constants.LanguageVersionValid` checks
it. The versions are listed in the
[Language versions](../LanguageGuide.md#language-versions) section of the
language guide, and the rule is item 9 of
[Conventions.md](Conventions.md). The `ptr64` build tag changes the size
of every pointer, string, slice and array header and with it every offset
after one, so a program only loads in a build with its pointer size
([#295](https://github.com/skycoin/skycoin-cx/issues/295)).

### What is not included

* The run time fields `RuntimeError`, `CompileErrorPrint`,
  `ProgramSandboxRoot`, `ProgramStdin`, `ProgramFormatBuffer`,
  `ProgramInputValues`, `ProgramOutputValues`, `ProgramCallbackValues`,
  `ProgramCallbackDepth` and `ProgramError`, and the
  collector state `Heap.Pressure`, `Heap.PressureLimit`, `Heap.Types`,
  `Heap.Roots` and `Heap.Collector`, see [GarbageCollector.md](GarbageCollector.md).
* `CxIrState.CompileErrorList`. A program with compile errors does not
  run.
* `CxIrState.CxIrStateGlobalsDeferred` and
  `CxIrState.CxIrStateGlobalInitializers`, which only the compiler reads
  while it types globals and orders their initializers.
* `CxIrState.CxIrStateNativeNames`, the cache of built-ins found by name
  Gids, which a program rebuilds when it looks a built-in up again.
* `CXProgram.ImportPositions`, where each package imports each other
  package, which only the shadowing errors of #486 read.
* `CXProgram.ImportScoped`, the imports into scope of #487, which only
  the compiler reads. Every shorthand is compiled as `pkg.name`.
* The build tags. A program that calls an opcode from a package the
  reading build leaves out, such as `sys/os` under `nocxos`, finds no
  handler for it.
* The flags that sized the memory. The program keeps the stack, data and
  heap bounds it was compiled with.

### Built programs

`cx build` (`t11-cx/build.go`) writes one executable file: the core binary
of `t6-cx-core-run`, then the serialized program with its memory and no
compression, then a 32 byte trailer, little endian.

| Offset from the trailer start | Size | Field |
|---|---|---|
| 0 | 8 | offset of the program from the start of the file, the core size |
| 8 | 8 | length of the program in bytes |
| 16 | 4 | CRC-32 (IEEE) of the program |
| 20 | 4 | trailer format version, 1 |
| 24 | 8 | magic `CXBUNDLE` |

The serialized form has no total length, so the trailer gives both the
offset and the length, and the offset plus the length plus 32 must be the
file size. `core.CoreTrailerAppend` writes it and `core.CoreTrailerRead`
(`t5-cx-runtime-core/trailer.go`) reads it from `os.Executable()` when the
core starts. A file without the magic is no built program, and the core
prints a message and exits with `CX_COMPILATION_ERROR`. A trailer that does
not fit the file or whose checksum fails is `FileErrorCodeContentInvalid`.

`core.CoreRegister` registers the opcodes of the core, the built-in ones,
`LibraryCall` and those of `sys/regexp`, `sys/time` and `os.Exit`
(`t4-cxos-core`), and the system libraries the core links. A library is
linked by the build tag `cxcorelibraryfile`, `cxcorelibrarynet`,
`cxcorelibraryprocess` or `cxcorelibrarycipher`, whose file in `t6-cx-core-run` adds its register
function. `make build` builds a core for every set, named by
`buildCoreFileName`, `cx-core-file-net` for file and net. `cx build` keeps
the enabled libraries the program calls, records them in the serialized
program and appends it to the core of that set. It refuses a program that
uses any other opcode and lists them, since build tags are not recorded. The
core refuses a program that enables a library it does not link, and opens
the working directory as the sandbox when the file library is enabled. The core runs the program on vm when it
lowers and on the old loop otherwise, as `cx --executor=auto` does, and
passes its command line arguments to the program as `os.Args`. It has no
flags, so the run time flags of cx, `--executor`, `-min-heap-free`,
`-max-heap-free` and `--debug-go-stack`, take their defaults. The heap cap
of `-heap-max` is stored in the program, `Heap.Max`, so it runs with the cap
it was built with (D42).

## Opcodes

This document does not list the opcodes. The list is
`constants.OpcodeLifecycleTable` in
[t1-constants/opcodes.go](../t1-constants/opcodes.go), mirrored in
[opcodes.csv](opcodes.csv). Each row is an `OpcodeLifecycle` with
`Number`, `Name`, `VersionAdded`, `VersionDeprecated` and
`VersionReassignable`. The numbering and lifecycle rules are items 7 and
8 of [Conventions.md](Conventions.md).

* The named opcodes are the `OP_*` constants in the same file, from
  `OP_IDENTITY` (1) to `OP_BOOL_NOT` (133). `END_OF_NAMED_OPCODES` (134)
  and the `START_OF_*` and `END_OF_*` markers bound ranges and are not
  opcodes.
* Every live opcode has a row in `constants.OpcodeDefinitionTable`
  ([t1-constants/opcodes_definitions.go](../t1-constants/opcodes_definitions.go))
  with its handler, signature and typed operator form. `make
  generate-opcodes` runs t2-cx-core-opcodegen, which writes the signature,
  operator and name tables of `t1-constants/opcodes_signatures_gen.go` and
  the opcode switch: `CoreOpcodeDispatch` in t5-cx-runtime-core, with the
  `nocxos` and `noregexp` groups in their own files, and `CxfxOpcodeDispatch`
  in t6-cxfx under the `cxfx` tag. Each case calls the handler directly,
  or, for a row whose handler is a call such as
  `cxfx.gl.Clear(uint32(%0))`, makes that call inline with the operands read
  at their offsets, which the simple GL and GLFW calls do (#432).
  Nothing registers an opcode at startup and a deprecated row has no case.
* A program that embeds CX, such as the Android playground with its `app`
  functions, adds opcodes outside the table. It names each one in
  `ast.NativeNames`, gives the compiler its native through
  `ast.NativeFunctionHost` and runs it from a wrapper of `ast.OpcodeDispatch`.
  These opcodes run on the old loop only. `TestNativeFunctionHost` in t11-cx
  shows the three steps.
* `LibraryCall` (795, `constants.OP_LIBRARY_CALL`) has no definition row
  and no name, so no program calls it by name, and the loop runs it. The system library functions have
  their own numbers in `constants.LibraryFunctionLifecycleTable` and compile
  to `LibraryCall`. `constants.LibraryFunctionDefinitionTable` in
  t1-constants/libraries_definitions.go gives each live one its handler, vm
  form and signature (decision D96). t2-cx-core-opcodegen writes the library
  signature tables of `t1-constants/libraries_signatures_gen.go` and one
  switch per library in the package of its handlers, under that package's
  build tag: `LibraryFileDispatch` and `LibraryProcessDispatch` in t5-cxos,
  `LibraryNetDispatch` in t5-cxnet and `LibraryCipherDispatch` in t5-cipher,
  each with a `Vm` switch of its vm forms. A binary links a library with
  `opcodes.LibraryLink`, which sets `ast.LibraryDispatches` and
  `vm.LibraryDispatchesVm` for its number. Nothing registers a library
  function. See item 16 of [Conventions.md](Conventions.md).
* A renamed opcode has its new name only.
  `constants.OpcodeRenameTable` keeps the old names, and the compiler
  turns a call by an old name into a compile error that names the new one.
* The typed operators are also reachable through
  `constants.OpcodeOperatorOf(typeCode, opcode)`, which is how
  `ProcessTypedOperator` turns `add` (82) into `i32.add` (383).

Before [#297](https://github.com/skycoin/skycoin-cx/pull/297) the
numbers above `END_OF_NAMED_OPCODES` came from registration order and
moved with build tags
([#296](https://github.com/skycoin/skycoin-cx/issues/296)).

## The runtime image

`lower.Program` (`t4-cx-compiler-lower`) turns a compiled program into an
`ir.Image` (`t2-cx-runtime-ir`), the flat form that `vm.Machine`
(`t3-cx-runtime-vm`) runs, steps 4 and 5 of
[#408](https://github.com/skycoin/skycoin-cx/issues/408). cx lowers every
program after compiling it. When every function lowers it runs the image on
vm, otherwise on the old loop. `cx --executor=old` or `--executor=vm` forces
one loop, and vm fails on a program that does not lower.

* `ir.Instr` is a fixed 32 byte record. `InstrOpcode` is an `ir.Opcode`,
  `InstrA` to `InstrD` are operand byte offsets, `InstrN` and `InstrM` hold
  a size, a jump target or a function id, and `InstrAux` indexes a side
  table. Bit i of `InstrMode` adds the frame pointer to operand i.
* `ir.Opcode` numbers are dense and grouped by family, and exist only in an
  image. They are not the language opcode numbers above, which are sparse,
  fixed for serialized programs and have no number for moves, calls or
  returns. The lowering maps a language opcode to an image opcode by its
  live name.
* `ir.Func` holds the code, the frame size and the input and output slots.
  `ImageFuncs` is indexed like `CXProgram.CXFunctions`, so a function id is
  the same in both. A function that does not lower has no code.
* `ir.Image` holds the data segment bytes and where they start, the stack
  size, the ids of `*init` and `main`, and the line and name tables. It
  holds no pointer to an IR object.
* `CX_LINE` expressions become `ir.Line` entries, a pc where a source line
  starts. A jump target, counted in expressions by `ThenLines` and
  `ElseLines`, becomes a pc. A target at or past `LineCount` becomes the
  final `OpcodeReturn`.
* A call becomes `OpcodeCallFrame` (zero the callee frame at the stack
  pointer), one `OpcodeCallArg` per input, `OpcodeCall`, then one
  `OpcodeCallResult` per output the caller receives, the same copies
  `processNonAtomicOperators` and `popStack` make.
* A system library call lowers when its function has a vm form in
  `constants.LibraryFunctionDefinitionTable` and its library is linked, see
  `vm.LibraryFunctionVmLinked`. The vm forms today are `os.Close`, `os.Seek`, `os.Sync`,
  `cipher.HashStateNew`, `cipher.HashStateRelease` and every function of
  `sys/net`. A `str` input is a pointer the vm form reads from the data
  segment, and an input of undefined type is a byte buffer, a `ui8` array
  lowered as its address and its byte size, two pointers that
  `OpcodeAddressOf` writes, see [Networking.md](Networking.md). It becomes
  `OpcodeCallFrame` for a block of the inputs then the outputs, packed as
  `ast.NativeFunctionOf` lays them out, one `OpcodeCallArg` per input,
  `OpcodeLibraryCall` with the library in M and the function in N, then
  `OpcodeCallResult` per output. `OpcodeLibraryCall` runs in the slow path.
  It checks `ImageLibraryEnabled[M]`, which `lower.Program` copies from
  `ProgramLibrariesEnabled`, and stops with `StopLibraryDisabled` when the
  library is not enabled. Any other library function is
  `ReasonLibraryFunctionOutsideSubset`.

The subset is the bool, integer and float operators, conversions and print,
`string.PrintLine`, `printf` with a str literal format, identity, declarations,
the jumps, `test` on non-string values and calls. Anything else gives a
`lower.Failure` with a `ReasonCode`.

* An operand may index arrays, select struct fields and dereference pointers,
  the steps `GetFinalOffset` takes. Field offsets fold into a static offset.
  An index or dereference becomes `OpcodeAddressIndex` or
  `OpcodeAddressDeref` code that writes the address to a scratch slot, and
  `OpcodeAddressLoad` or `OpcodeAddressStore` moves the value. The
  addresses are computed before a built-in runs and after a call returns, as
  in the old loop. A slice or the heap stays outside the subset.
* The scratch is `ImageScratchSize` bytes after the data segment, used by one
  expression at a time. The old loop has no scratch, so it is not compared.
* A stack operand must fit the frame of its function. The old loop lets a
  larger one, from a recursive struct type, run into the data segment.
* Taking an address, in identity or a call by reference, is
  `OpcodeAddressOf` or `OpcodeAddressPointer`, by the type tests `OpIdentity`
  and `processNonAtomicOperators` make.
* `printf` reads its format at lower time as `buildString` does. Constant text
  goes to `ImageText` and `OpcodePrintText`, each value to
  `OpcodePrintValue` or `OpcodePrintStr`. `%v` on a whole array, struct or
  pointer stays outside. `TestLowerCompare` lowers every program
in `tests/` and, for each one that lowers completely, compares the memory
after the old loop and vm. `TestExecutionRecords` checks stdout and the exit
code of both against the execution records, see
[ExecutionRecords.md](ExecutionRecords.md).

`vm.Machine` holds the memory (stack, data segment, then scratch), a fixed array
of `ir.ImageCallDepthMax` frames and an output buffer, all sized before the
run. `Run` runs one entry function and returns a `vm.Stop` code and a pc,
never a string. The caller runs `*init`, then `main` in the frame after it,
as `execute.RunCompiled` does.

* `StopOutputFull` and `StopTestFailed` are resumable. The caller writes the
  output buffer, prints a failed test from the line table and
  `Machine.TestMessage`, the str bytes read from memory, then calls `Resume`.
* A str print that does not fit the output keeps its place and stops with
  `StopOutputFull`, and the same instruction goes on after `Resume`.
* `StopIndexNegative` is a negative index, `CX_RUNTIME_INVALID_CAST` as in
  `types.Cast_i32_to_ptr`. `StopAddressInvalid` is a computed address or str
  outside the stack and data segment, where the old loop would read the heap.
* `StopDivideByZero` and `StopStackOverflow` end the run. `printer.PrinterVmErrorPrint` prints them as
  the old loop does, the line from `ir.LineAt` and the call stack by copying
  the memory and frames into the program for `PrinterStackPrint`. `vm.StopExitCodes` gives the exit code.
* The loop has no call in it. Prints, `test`, float mod, max and min, address
  dereferences, loads and stores other than 4 bytes and every address error
  return to a slow path outside the loop, which runs the instruction and
  enters the loop again. A call in the loop makes Go spill the loop state on
  every instruction, and so does a loop past the size where Go stops
  inlining the memory accessors into it.
* A run allocates nothing (`TestRunVmAllocs`).

Two programs behave differently on purpose. At the call depth limit the old
loop crashes with an index out of range in `ast.RuntimeError`, and a runtime
error in `*init` crashes it because `runSysInitFunc` has no recover. vm
prints a stack overflow or runtime error for both.

## Known oddities

* A user function named like a named opcode, for example `add`, is never
  called. The call compiles to the built-in
  ([#294](https://github.com/skycoin/skycoin-cx/issues/294)).
* Serialization used to depend on map order and could move functions
  between packages on a round trip. It is now in key order
  ([#186](https://github.com/skycoin/skycoin-cx/issues/186),
  [#250](https://github.com/skycoin/skycoin-cx/issues/250)).
* Native operators were described by shared `CXArgument` values, and
  `GetCXTypeSignatureRepresentationOfCXArg` wrote one program's indexes
  into them, which broke the next REPL program
  ([#174](https://github.com/skycoin/skycoin-cx/issues/174), closed).
  Each expression still gets its own copy of the native function in
  `CXFunctions`.
* Pointers to slices are only partly modelled. There is
  `TYPE_POINTER_SLICE_ATOMIC` but no pointer to a slice of structs, which
  stays a `TYPE_CXARGUMENT_DEPRECATE`. A `*[]str` is a
  `TYPE_CXARGUMENT_DEPRECATE` too, and `OpIdentity` and the vm lowering test
  for it with `CXArgumentStringSlicePointer` so `p = &s` takes its address.
* `GetFinalOffset` decides whether an offset is in the frame by comparing
  it with `Stack.Size`, not by a flag on the operand.
* The line marker before the `jmp` of an `if` carries the line of the
  closing brace, line 9 in the worked example, because
  `SelectionExpressions` runs when the whole statement is reduced.
* `ast.ToString` walks `CXProgram.Packages` and `CXPackage.Functions` as
  maps, so `cx -ast` prints functions in a different order from run to
  run.
* `CX_ARGUMENT`, the `TYPE_ARRAY_STRUCT` to `TYPE_SLICE_POINTER_COMPLEX`
  signatures and `OP_ABS_JMP` are defined but never produced by the
  compiler.
* Several fields still hold Go pointers instead of arena indexes, and the
  getters return pointers into arenas that `append` can move. See
  [ObjectModel.md](ObjectModel.md)
  ([#299](https://github.com/skycoin/skycoin-cx/issues/299)).
