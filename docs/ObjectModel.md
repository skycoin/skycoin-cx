# Object model of the CX IR

This is the standard for how the compiler IR in `t3-ast` and the packages
around it hold their objects. Every IR object type has a definition, a
manager that owns every object of that type, and a generational index that
names one object. One struct, `CxIrState`, owns every manager.

The IR itself is described in [IR.md](IR.md). The migration to this standard
is tracked in [#299](https://github.com/skycoin/skycoin-cx/issues/299),
which replaces the earlier plan in #5 to #9. Every place in the code that does
not follow the standard yet is listed in
[objectmodel-sites.csv](objectmodel-sites.csv), see [Inventory](#inventory).

## Contents

* [Object types](#object-types)
* [Files](#files)
* [Def, the object](#def-the-object)
* [Gid, the generational index](#gid-the-generational-index)
* [Mng, the manager](#mng-the-manager)
* [Fon and Fof, helper functions](#fon-and-fof-helper-functions)
* [CxIrState, the single owner](#cxirstate-the-single-owner)
* [Rules](#rules)
* [Teardown](#teardown)
* [Fixed size](#fixed-size)
* [Naming](#naming)
* [Errors](#errors)
* [Compile benchmark](#compile-benchmark)
* [Migration](#migration)
* [Inventory](#inventory)

## Object types

These are the managed object types. The type tag is fixed forever, like an
opcode number. A removed type keeps its tag, and a new type takes the next
free one.

| Object | Type tag | Arena today | Index type today |
| --- | --- | --- | --- |
| `CXProgram` | `0x01` | none, one per compile | `*CXProgram` |
| `CXPackage` | `0x02` | `CXProgram.CXPackages` | `CXPackageIndex` |
| `CXFunction` | `0x03` | `CXProgram.CXFunctions` | `CXFunctionIndex` |
| `CXNativeFunction` | `0x04` | none, global `ast.Natives` | opcode number |
| `CXStruct` | `0x05` | `CXProgram.CXStructs`, and heap `*CXStruct` parameter lists | `CXStructIndex` |
| `CXExpression` | `0x06` | none, held by value in `CXFunction.Expressions` and `CXProgram.SysInitExprs` | none |
| `CXAtomicOperator` | `0x07` | `CXProgram.CXAtomicOps` | `int` |
| `CXLine` | `0x08` | `CxIrState.CXLineMng` since #311 | `int`, the slot index |
| `CXArgument` | `0x09` | `CXProgram.CXArgs` | `CXArgumentIndex` |
| `CXArgumentDebug` | `0x0A` | none, heap | `*CXArgumentDebug` |
| `CXTypeSignature` | `0x0B` | `CXProgram.CXTypeSignatures` | `CXTypeSignatureIndex` |
| `CXTypeSignature_Array` | `0x0C` | `CxIrState.CXTypeSignature_ArrayMng` since #316 | `int`, the slot index |
| `CXTypeSignature_Struct` | `0x0D` | `CxIrState.CXTypeSignature_StructMng` since #321 | `int`, the slot index |
| `CXCall` | `0x0E` | `CXProgram.CallStack` | `types.Pointer` (`CallCounter`) |

Tags `0x00` and `0x80` to `0xFF` are never used, see
[Gid](#gid-the-generational-index).

Not managed, with the reason.

* `CXValue` is the VM's view of one operand during one call, rebuilt for
  every expression in a reused scratch slice. It is not part of the IR. Its
  pointer fields `TypeSignature` and `Expr` become Gids in the Gid phase of
  the objects they point at.
* `DereferenceStruct` (`t3-ast/mem_offset.go`) lives on the stack of
  `GetFinalOffset` only.
* The `serialized*` structs in `t3-ast/serialize_structs.go` are the byte
  form. They hold offsets, not objects.
* `CXArgumentSlice`, `CXArgumentStruct` and `CXArgumentPointer` were
  unused and were removed with the Def files.

## Files

Each object type `<Obj>` has up to five files in the package that defines
it. `<obj>` is the type name in lower case with underscores removed, so
`CXTypeSignature_Array` is `cxtypesignaturearray`.

| File | Holds | Required |
| --- | --- | --- |
| `<obj>_def.go` | The struct `<Obj>` and nothing else | yes |
| `<obj>_mng.go` | The manager `<Obj>Mng` and its methods, generated | yes |
| `<obj>_gid.go` | The index type `<Obj>Gid`, its type tag and nil constants, generated | yes |
| `<obj>_fon.go` | Function-on, helpers that write to one object | no, may be empty |
| `<obj>_fof.go` | Function-of, helpers that only read one object | no, may be empty |

For example `cxfunction_def.go`, `cxfunction_mng.go`, `cxfunction_gid.go`,
`cxfunction_fon.go` and `cxfunction_fof.go`.

None of the suffixes is a `GOOS` or `GOARCH` name, so Go never treats them
as build constraints. A file that needs a build constraint adds a
`//go:build` line, as `arg_constants_ptr32.go` does. Code that works on
several object types, the compiler passes and the VM, stays in ordinary
files without these suffixes.

## Def, the object

`<obj>_def.go` holds the struct of the object. The struct keeps its plain
name, `CXArgument` and not `CXArgumentDef`, because a kind suffix on a type
is banned by the naming rules (#212, section 1.5). The file name says it is
the definition.

* A Def holds data only. Its methods live in the Fon and Fof files.
* A Def field that names another managed object is that object's `<Obj>Gid`,
  never a pointer and never a bare `int` index.
* A Def has no `Index` field of its own. The manager knows where an object
  is, and a Gid names it.
* A Def is never created outside its manager, see [Rules](#rules).

## Gid, the generational index

A Gid names one object of one type. It is an `int64`.

```
 63      56 55                     32 31                                  0
+----------+-------------------------+-------------------------------------+
| type tag |       generation        |                index                |
|  8 bits  |         24 bits         |               32 bits               |
+----------+-------------------------+-------------------------------------+
 \______________ high 32 bits _______/\___________ low 32 bits ____________/
```

* Bits 0 to 31 are the index of the object's slot in the manager's array.
* Bits 32 to 63 are the generation, 32 bits, whose top byte (bits 56 to 63)
  is the type tag. The tag is masked out wherever generations are compared,
  so the generation proper is bits 32 to 55, 24 bits.
* Type tags run from `0x01` to `0x7F`, so bit 63 is always 0 and a valid Gid
  is always positive. Every negative value and 0 is invalid.
* Generations start at 1. The generation of a slot goes up by one each time
  its object is released. A slot whose generation reaches
  `GidGenerationLast` is retired and never reused, so a generation never
  wraps and an old Gid can never name a new object.
* Gid 0 is the nil Gid of every type. It has generation 0, which no object
  ever has, so it can never pass a check.

The constants and helpers live in package `objmodel`, directory
`t1-cx-core-objmodel` (tier 1, it imports `t0-cx-core-assert`).

```go
package objmodel

type TypeTag uint8

const (
	GidIndexBits      = 32
	GidGenerationBits = 24
	GidTypeTagBits    = 8

	GidIndexMask      uint64 = 0x0000_0000_FFFF_FFFF // bits 0 to 31
	GidHighMask       uint64 = 0xFFFF_FFFF_0000_0000 // bits 32 to 63, generation and tag
	GidGenerationMask uint64 = 0x00FF_FFFF_0000_0000 // bits 32 to 55
	GidTypeTagMask    uint64 = 0xFF00_0000_0000_0000 // bits 56 to 63

	GidGenerationShift = 32
	GidTypeTagShift    = 56

	GidGenerationFirst uint32 = 1
	GidGenerationLast  uint32 = 0x00FF_FFFF

	GidTypeTagFirst TypeTag = 0x01
	GidTypeTagLast  TypeTag = 0x7F

	GidNil int64 = 0

	// GidCapMax is the largest Cap of a manager, so a slot count fits in uint32.
	GidCapMax uint64 = GidIndexMask
)

// GidCompose builds a Gid. It asserts the tag range and a generation from
// GidGenerationFirst to GidGenerationLast - 1, the generations a live object
// can have. Every uint32 is a valid index.
func GidCompose(typeTag TypeTag, generation uint32, index uint32) int64

// GidIndex returns bits 0 to 31.
func GidIndex(gid int64) uint32

// GidGeneration returns bits 32 to 55, the type tag masked out.
func GidGeneration(gid int64) uint32

// GidTypeTag returns bits 56 to 63.
func GidTypeTag(gid int64) TypeTag

// GidCheck asserts checks 1 and 2 below, and that gid is positive. The
// manager does checks 3 and 4 against its own slots.
func GidCheck(gid int64, typeTag TypeTag, typeName string, capacity int)
```

The bit operations work on `uint64(gid)`, so the masks are `uint64`
constants and no shift touches the sign bit.

Each object type's `<obj>_gid.go` defines its own named type, so the
compiler rejects a `CXFunctionGid` where a `CXArgumentGid` is wanted.

```go
// cxargument_gid.go
type CXArgumentGid int64

const (
	CXArgumentGidTypeTag objmodel.TypeTag = 0x09
	CXArgumentGidNil     CXArgumentGid    = 0
)
```

A Gid is valid for manager `m` when all of these hold, and every check is a
class B assert.

1. `GidTypeTag(gid)` is the manager's type tag.
2. `GidIndex(gid)` is below the manager's Cap.
3. The slot at that index holds a live object.
4. `GidGeneration(gid)` equals the slot's generation.

## Mng, the manager

`<obj>_mng.go` holds the manager, the only place where objects of that type
are allocated and released. A manager is a linear array of slots.

```go
// cxargument_mng.go
type CXArgumentMng struct {
	CXArgumentMngObjects     []CXArgument // one per slot, len is the Cap
	CXArgumentMngGenerations []uint32     // generation of each slot, 0 while never used
	CXArgumentMngLive        []bool       // slot holds a live object
	CXArgumentMngFree        []uint32     // released slots that can be reused
	CXArgumentMngNext        uint32       // first slot never used
	CXArgumentMngRetired     uint32       // slots retired at GidGenerationLast
	CXArgumentMngLen         int          // live objects
	CXArgumentMngGrowable    bool         // grow when full instead of asserting
}

func (manager *CXArgumentMng) CXArgumentMngInit(capacity int, growable bool)
func (manager *CXArgumentMng) CXArgumentAlloc() CXArgumentGid
func (manager *CXArgumentMng) CXArgumentRelease(gid CXArgumentGid)
func (manager *CXArgumentMng) CXArgumentGet(gid CXArgumentGid) *CXArgument
func (manager *CXArgumentMng) CXArgumentLen() int
func (manager *CXArgumentMng) CXArgumentCap() int
func (manager *CXArgumentMng) CXArgumentMngTeardown()
```

* `Init` sets the Cap and whether the manager may grow. A manager is used
  only after Init, and Init on a manager that holds objects is an assert.
* `Alloc` takes a free slot, or the next never used one, sets its object to
  the zero value, marks it live, raises Len by one and returns its Gid. When
  every slot is used, a growable manager doubles its Cap, from 0 to 1 and
  never past `GidCapMax`. A manager that cannot grow asserts.
* `Release` checks the Gid, sets the object to the zero value so its slices
  and strings can be collected, marks the slot free, raises its generation
  and lowers Len by one. It puts the slot on the free list unless its
  generation has reached `GidGenerationLast`, in which case the slot is
  retired.
* `Get` checks the Gid and returns a pointer to the object in its slot. The
  pointer is borrowed, see [Rules](#rules).
* `Len` is the number of live objects. `Cap` is the number of slots,
  retired ones included, at most `GidCapMax`.
* `MngTeardown` asserts that Len is 0 and drops the arrays, which leaves Cap
  at 0.

An object is filled after Alloc through the pointer Get returns, by its own
Fon functions or by the code that allocated it. There is no constructor that
takes field values.

Growing copies the array, so every pointer Get returned before is stale
after the next Alloc on the same manager. That is why a Get pointer is never
held across an Alloc.

Len is checked against the slots in O(1) after every Alloc and Release, as
`Next - len(Free) - Retired`, which is why the manager counts retired slots.
Counting the live flags would make every Alloc O(Cap).

Every check builds its message only when it fails, `if !ok {
assert.AssertPreconditionTrue(false, fmt.Sprintf(...)) }`, because Get and
Alloc are on the compiler's hot path once the sites use them.

### Generated

The 14 managers differ only in their type, so `<obj>_gid.go`,
`<obj>_mng.go` and the test `<obj>_mng_test.go` are generated from one
template by `t0-cx-core-objmodelgen` (`make generate-objmodel`), a tier 0
`package main` that holds the type table above. The generated files are
committed and start with a `Code generated ... DO NOT EDIT.` line.
`make check-objmodel` fails when one is stale, when a generated file is no
longer in the table, or when a `<obj>_def.go` has no table entry. A new
object type is a new row in that table. The generator was picked over 14
written copies because a fix to the manager is one template change that
reaches every type, and the check keeps the copies from drifting.

## Fon and Fof, helper functions

`<obj>_fon.go` (function-on) holds helpers that write to one object, such as
`SetType` on a `CXArgument`. `<obj>_fof.go` (function-of) holds helpers that
only read one object, such as `IsPointer` or a size computed from the
object's own fields.

* A Fon or Fof function takes the object as a pointer from Get, or a value,
  plus plain values. It may be a method on the Def.
* It does not use any manager or `CxIrState`. It never calls Alloc, Release
  or Get, of its own type or any other. In Go terms it names no `*Mng` type
  and no `CxIrState`, which is what "cannot import the manager" means inside
  one package. `make check-objmodel` checks it on every `_fon.go` and
  `_fof.go`, and also that every `_def.go` holds one type and no function.
* A Fof function does not write to the object or anything it reaches.
* Both files may be empty, or absent.

A function that needs a second object, or a manager, takes
`state *CxIrState` and lives in an ordinary file. Most of the compiler is
such code.

## CxIrState, the single owner

`CxIrState` (`t3-ast/cxirstate.go`) holds one manager of each type and is
the only owner of every IR object.

```go
type CxIrState struct {
	CXProgramMng             CXProgramMng
	CXPackageMng             CXPackageMng
	CXFunctionMng            CXFunctionMng
	CXNativeFunctionMng      CXNativeFunctionMng
	CXStructMng              CXStructMng
	CXExpressionMng          CXExpressionMng
	CXAtomicOperatorMng      CXAtomicOperatorMng
	CXLineMng                CXLineMng
	CXArgumentMng            CXArgumentMng
	CXArgumentDebugMng       CXArgumentDebugMng
	CXTypeSignatureMng       CXTypeSignatureMng
	CXTypeSignature_ArrayMng CXTypeSignature_ArrayMng
	CXTypeSignature_StructMng CXTypeSignature_StructMng
	CXCallMng                CXCallMng
}

func (state *CxIrState) CxIrStateInit()
func (state *CxIrState) CxIrStateTeardown1()
func (state *CxIrState) CxIrStateTeardown2()
func (state *CxIrState) CxIrStateManagerLens() []CxIrStateManagerLen

type CxIrStateManagerLen struct {
	Name string
	Len  int
}
```

* A manager is a value field of `CxIrState`, never a pointer, and no manager
  exists outside a `CxIrState`. The manager unit tests are the one exception.
  They test one manager on its own.
* One `CxIrState` holds one program, so `CXProgramMng` has a Cap of 1 and
  cannot grow. `CXCallMng` has the fixed Cap `constants.CALLSTACK_SIZE`,
  the call stack. Every other manager starts at Cap 0 and grows, as the
  arena slices it replaces do. A process that compiles several programs,
  such as the REPL or a test, makes one `CxIrState` for each.
* `*CxIrState` is passed down as a parameter, the way `prgrm *CXProgram` is
  passed today. A value that lives for one call, such as the lexer of one
  parse, may hold it for that call.
* The program's runtime data, `Memory`, the segments and the call stack,
  stays in `CXProgram`. Its arena slices become the managers.
* `ast.Natives` and the other opcode tables are package state today. Moving
  them into `CxIrState` is part of the `CXNativeFunction` allocation
  phase.

### Until the program is managed

The compiler passes `prgrm *CXProgram` to almost every function today,
and stage 4 does not change that. Each program points at its state
through `CXProgram.CxIrState`, which `MakeProgram`, `Deserialize`
and the test helpers that build a program set to a new, initialized
`CxIrState`. A site reaches its manager as `prgrm.CxIrState.<Obj>Mng`,
so stage 4 changes no signature except to give a constructor a program,
and two programs in one process never share a manager.

The field points at the state, not at a managed object, so rule 3 does not
apply to it. It goes in phase 5 of `CXProgram`, when `state *CxIrState`
replaces `prgrm *CXProgram` as the parameter and the program becomes the
one object in `state.CXProgramMng`. A constructor that had no program in scope
takes one, so `MakeArgument`, `MakeField` and `MakeGlobal` take `prgrm`
first. `Param` passes nil, because the natives it makes are shared by every
program until the `CXNativeFunction` allocation phase.

## Rules

1. **Every object is held by its manager.** Nothing else owns an object of a
   managed type, by pointer or by value.
2. **No New.** There is no `New<Obj>` or `Make<Obj>` constructor, no `&<Obj>{}`
   or `new(<Obj>)`, and no copy of a Def into a new variable that outlives
   the call. Every object comes from `<Obj>Alloc` and goes back through
   `<Obj>Release`.
3. **No pointers at rest.** A pointer to a managed object is never stored in
   a struct field, a package variable, a map, a slice or a closure that
   outlives the call. It is stored as a Gid. A pointer from Get is borrowed.
   It may live in a local variable or a parameter until the end of the call,
   and never across an Alloc on the same manager or a Release of that Gid.
4. **One owner.** Each object has exactly one owner, the object or the state
   whose teardown releases it. Any other holder keeps a borrowed Gid and does
   not release it. The owners are listed in [Teardown](#teardown).
5. **One state.** Every manager lives in the one `CxIrState`.
6. **No interfaces.** Managers, Gids and helpers are concrete types, by the
   repository rule. There is no `Manager` interface and no generic manager
   type.

## Teardown

Teardown is the release of every object, and it has two steps.

1. **Teardown1** (`CxIrStateTeardown1`) releases the program and, through it,
   every object it owns, by walking the owner relations below. Afterwards
   every manager's Len must be 0. Until #300 Teardown1 releases only the
   program, because the owner fields are not Gids yet.
2. **Teardown2** (`CxIrStateTeardown2`) calls every manager's `MngTeardown`,
   which asserts Len is 0 and drops the arrays.

Completion, the normal end of `cx` after a compile or a run, runs Teardown1
then Teardown2. A class B assert or a compile error that exits the process
does not tear down.

**Clean shutdown criterion.** After Teardown1, on every path that reaches
it, the Len of every manager is 0. A manager whose Len is not 0 holds an
object nobody owns, a leak, and that is a class B failure. The test helper
`CxIrStateManagerLens` returns each manager's name and Len, and the teardown
test asserts that all of them are 0 after compiling, and after running,
every program the compile benchmark uses and every test program that runs in
process. See [#300](https://github.com/skycoin/skycoin-cx/issues/300).

Owners, as intended. The teardown issue checks each one against the code
before Teardown1 is written.

| Object | Owned by |
| --- | --- |
| `CXProgram` | `CxIrState` |
| `CXPackage` | `CXProgram` |
| `CXFunction`, user | the `CXPackage` it is declared in |
| `CXFunction`, native copy | the `CXAtomicOperator` whose `Operator` it is |
| `CXNativeFunction` | `CxIrState` |
| `CXStruct`, declared | the `CXPackage` it is declared in |
| `CXStruct`, parameter list or globals | the `CXFunction`, `CXAtomicOperator` or `CXPackage` whose `Inputs`, `Outputs` or `Globals` it is |
| `CXExpression` | the `CXFunction` whose body it is in, or `CXProgram` for `SysInitExprs` |
| `CXAtomicOperator`, `CXLine` | the `CXExpression` that names it |
| `CXTypeSignature` | the `CXStruct` whose `Fields` it is in |
| `CXTypeSignature_Array`, `CXTypeSignature_Struct` | the `CXTypeSignature` whose `Meta` names it |
| `CXArgument` | the `CXTypeSignature` whose `Meta` names it (`TYPE_CXARGUMENT_DEPRECATE`), or the `CXArgument` whose `Fields`, `Indexes`, `Inputs` or `Outputs` hold it |
| `CXArgumentDebug` | the `CXArgument` it was made for. A `CXTypeSignature` that names it borrows it |
| `CXCall` | `CXProgram`, a fixed stack of `constants.CALLSTACK_SIZE` slots |

## Fixed size

A Def should be fixed size. A fixed size Def can live in an array slot with
nothing behind it, can be copied, compared and serialized as bytes, and
releases nothing but its slot.

A Def field breaks this when it is a slice, a map, a string, a pointer, an
interface or a function value. The current violations are listed per Def in
[#301](https://github.com/skycoin/skycoin-cx/issues/301). They are
recorded and not fixed until the object model is in place. Pointers become
Gids in the Gid phase, which already makes them fixed size. Slices, maps and
strings stay until a later normalization, for example a list becomes a first
Gid and a count in a manager, and a string becomes an index into a string
table.

## Naming

Names follow the verbose, prefixed style of
[#212](https://github.com/skycoin/skycoin-cx/issues/212) and
[ErrorHandling.md](ErrorHandling.md#naming).

* Types are `<Obj>`, `<Obj>Mng` and `<Obj>Gid`.
* Manager fields are `<Obj>Mng<Field>`, so they sort together.
* Manager methods are `<Obj><Verb>`, `CXArgumentAlloc`, `CXArgumentRelease`,
  `CXArgumentGet`, `CXArgumentLen`, `CXArgumentCap`, and manager lifecycle
  methods are `<Obj>Mng<Verb>`, `CXArgumentMngInit`, `CXArgumentMngTeardown`.
* Gid constants are `<Obj>GidTypeTag` and `<Obj>GidNil`.
* `CxIrState` methods are `CxIrState<Verb>`.
* Fon and Fof functions start with the object they work on. A method keeps
  its current name until the naming pass of #212 renames it.
* `objmodel` constants and functions start with `Gid` or `TypeTag`.

## Errors

Every object model failure is a programming error, class B in
[ErrorHandling.md](ErrorHandling.md), and crashes through
`t0-cx-core-assert`.

| Failure | Assert |
| --- | --- |
| Gid is nil, negative, has another type's tag, or an index at or above Cap | `AssertPreconditionTrue` in Get and Release |
| Gid's generation is not its slot's generation, a stale Gid | `AssertPreconditionTrue` in Get and Release |
| Release of a slot that is not live, a double release | `AssertPreconditionTrue` in Release |
| Alloc on a full manager that cannot grow | `AssertPreconditionTrue` in Alloc |
| Init on a manager that holds objects | `AssertPreconditionTrue` in Init |
| Len is not 0 at teardown | `AssertInvariantTrue` in `MngTeardown` and the teardown test |
| Len is not `Next - len(Free) - Retired`, the live slot count | `AssertInvariantTrue` in Alloc and Release |

No object model function returns an error value.

## Compile benchmark

`BenchmarkCompile` in `t11-cx/compile_bench_test.go` compiles seven programs
from `tests/` with no run. It registers the opcodes once, as `cx` does.
Each iteration makes a fresh program with `cxinit.MakeProgram` and
`cxinit.RegisterPackages` outside the timer, then times
`cxparsing.ParseSourceStrings` (passes one and two) and
`cxparsing.AddInitFunction`. It reports ns/op,
B/op, allocs/op and `irobjects/op`, the number of objects in the arenas after
the compile.

```
go test -run '^$' -bench BenchmarkCompile -count=10 ./t11-cx
```

Every phase PR runs it, compares with the baseline using `benchstat`, and
puts the comparison in the PR.

### Baseline

Measured on 2026-11-20 at `fb7a1243` (main before this standard), Go
1.27.3 linux/amd64, an Intel Core i7-3770S at 3.10 GHz with 8 threads, on
Linux 7.2. `-count=10`, times in milliseconds per compile.

| Program | Median ms | Min ms | Max ms | Spread | B/op | allocs/op | IR objects |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `hello-world.cx` | 0.345 | 0.298 | 0.373 | 21.5% | 134482 | 950 | 37 |
| `bubble-sort.cx` | 1.874 | 1.593 | 2.044 | 24.1% | 1109558 | 7731 | 732 |
| `fibonacci-recursive.cx` | 0.633 | 0.577 | 0.681 | 16.4% | 366694 | 2642 | 197 |
| `test-array.cx` | 3.859 | 3.523 | 4.083 | 14.5% | 1958511 | 16771 | 1692 |
| `test-slices.cx` | 15.225 | 12.097 | 16.284 | 27.5% | 9098722 | 58061 | 8546 |
| `test-gc.cx` | 15.464 | 13.632 | 16.841 | 20.8% | 10660948 | 64667 | 8291 |
| `testing-pointers-to-arrays-of-structs.cx` | 1.508 | 1.408 | 1.536 | 8.5% | 639370 | 4917 | 326 |

Spread is (max - min) / median. The machine was shared with other work
(load average 2.5 to 5.8 during the run), so times vary by up to 28%
between runs. IR objects were the same in all ten runs, and B/op and
allocs/op stayed within 1% and 3 allocations, so those are the steadier
signal. The table shows the last run's B/op and allocs/op. A phase PR
compares all three with `benchstat`, on an idle machine where it can.

## Migration

The phases run in this order, and every PR keeps `make check` green. The
detailed plan, its stages with entry and exit criteria, and the issue of each
object and phase are in
[#299](https://github.com/skycoin/skycoin-cx/issues/299).

1. **Def.** Each struct moves into its `<obj>_def.go`. No behavior change.
2. **Mng and Gid.** Package `objmodel`, each `<obj>_gid.go` and
   `<obj>_mng.go` with tests, and `CxIrState`. Nothing uses them yet.
3. **Comment.** Every site that breaks the standard gets a two line
   `// OBJMODEL:` comment naming its target call and its issue. Comments
   only. This phase is done first, right after the standard, so every
   later PR starts from marked sites.
4. **Allocation.** Each `New`, `Make`, `&<Obj>{}`, `Add*InArray` and
   `Remove*` site calls the manager. The arenas become the managers' arrays.
5. **Gid.** Each index type and each pointer at rest becomes a Gid.
6. **Teardown.** Teardown1, Teardown2 and the Len 0 test.

In phase 4 a field that names an object by an `int` index keeps the index,
which is the Gid's index bits, `int(objmodel.GidIndex(int64(gid)))`. The
old getter, such as `GetCXLine`, builds the Gid back from the slot's
generation and calls Get, so its callers wait for phase 5. A site that
fills an object takes the Get pointer after its last Alloc on that manager.

The objects go through phases 4 and 5 in this order, contained objects
first, so an object's allocation moves before the objects whose
constructors create it.

| Step | Objects (phase 4, phase 5 sites each) | Phase 4 sites | Phase 5 sites |
| --- | --- | --- | --- |
| 1 | `CXArgumentDebug` (20, 2), `CXLine` (11, 3), `CXTypeSignature_Array` (37, 61), `CXTypeSignature_Struct` (15, 35) | 83 | 101 |
| 2 | `CXArgument` (291, 196), `CXTypeSignature` (147, 179) | 438 | 375 |
| 3 | `CXStruct` (35, 13) | 35 | 13 |
| 4 | `CXAtomicOperator` (16, 91), `CXExpression` (119, 9) | 135 | 100 |
| 5 | `CXFunction` (45, 89), `CXNativeFunction` (6, 0) | 51 | 89 |
| 6 | `CXPackage` (49, 90), `CXCall` (9, 1), `CXProgram` (57, 2) | 115 | 93 |

`CXArgumentDebug` takes phases 4 and 5 in one step. Every allocation stores
the object in `ArgDetails`, a pointer field that a `CXArgument` and its
`CXTypeSignature` share and write through, so it cannot move into the manager
before that field is a `CXArgumentDebugGid` (#306, #307).



Phases 1 and 2 add files and touch no existing site, 14 Def moves and 28
Mng and Gid files, generated, see [Generated](#generated). Phase 3 comments
every site in the table, in one PR. Phase 6 adds the teardown functions and their test.


## Inventory

[objectmodel-sites.csv](objectmodel-sites.csv) has one row for every site
that breaks the standard today, 1628 in all, plus one row for
`DereferenceStruct.arg`, which lives on the stack and needs no change. Its
columns are

* `site`, a stable id such as `S0123`, which the `// OBJMODEL:` comment at
  the site repeats,
* `object`, the managed type the site is about,
* `category`, the kind of site, below,
* `file` and `line`, where the site is. A site in a generated parser points
  at its `.y` grammar,
* `function`, the enclosing function, type or `package var`,
* `current_form`, the code at the site,
* `target`, the manager call or Gid type that replaces it,
* `phase`, 4 for allocation and release or 5 for Gid,
* `issue`, the issue of that object and phase,
* `depends_on`, the phase that must be done first.

| Category | Meaning |
| --- | --- |
| `ctor-def` | a constructor, `Make*`, `MakeCall`, `Param`, `Slice`, `nativeArgCopy`, `Deserialize`, `deserializeArgument`, `deserializeExpression` or the test generator `GenerateSampleProgram` |
| `ctor-call` | a call to one of those constructors |
| `arena-add` | a call that inserts into an arena, `Add*InArray`, `AddCXLine`, `AddCXAtomicOp`, `AddPackage` and the `astapi` wrappers |
| `arena-append` | an `append` to an arena slice inside those functions |
| `alloc-literal-heap` | `&<Obj>{...}` |
| `alloc-literal-value` | `<Obj>{...}` that becomes an object |
| `alloc-copy` | `*ptr` copied into a new object |
| `alloc-var` | a local `var x <Obj>` that becomes an object |
| `make-slice` | `make([]<Obj>...)`, `make([]*<Obj>...)` or a slice literal of objects |
| `release-call` | a `Remove*` call |
| `drop` | an object dropped for Go's garbage collector, a list field truncated or spliced, a `delete` from a name map, or a program replaced |
| `get-call` | an arena getter, `Get*FromArray`, `GetCXArg`, `GetCXAtomicOp`, `GetCXLine` |
| `get-addr-arena` | `&prgrm.<arena>[i]` |
| `ptr-field` | a struct field that holds a pointer to an object |
| `ptr-global` | a package variable that holds a pointer to an object |
| `value-field` | a struct field that holds a list of objects by value |

The counts per object.

| Object | ctor-def | ctor-call | arena-add | arena-append | alloc-literal-heap | alloc-literal-value | alloc-copy | alloc-var | make-slice | release-call | drop | get-call | get-addr-arena | ptr-field | ptr-global | value-field | phase 4 | phase 5 | total |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `CXArgumentDebug` | 0 | 0 | 0 | 0 | 20 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 2 | 0 | 0 | 20 | 2 | 22 |
| `CXLine` | 0 | 0 | 2 | 1 | 8 | 0 | 0 | 0 | 0 | 0 | 0 | 2 | 1 | 0 | 0 | 0 | 11 | 3 | 14 |
| `CXTypeSignature_Array` | 0 | 0 | 18 | 1 | 12 | 2 | 4 | 0 | 0 | 0 | 0 | 60 | 1 | 0 | 0 | 0 | 37 | 61 | 98 |
| `CXTypeSignature_Struct` | 0 | 0 | 7 | 1 | 4 | 0 | 2 | 1 | 0 | 0 | 0 | 34 | 1 | 0 | 0 | 0 | 15 | 35 | 50 |
| `CXArgument` | 8 | 139 | 31 | 1 | 81 | 1 | 3 | 1 | 16 | 0 | 6 | 178 | 15 | 8 | 0 | 0 | 291 | 196 | 487 |
| `CXTypeSignature` | 0 | 0 | 89 | 1 | 21 | 4 | 14 | 8 | 1 | 0 | 9 | 176 | 1 | 2 | 0 | 0 | 147 | 179 | 326 |
| `CXStruct` | 1 | 15 | 2 | 1 | 12 | 0 | 0 | 3 | 0 | 0 | 1 | 0 | 6 | 7 | 0 | 0 | 35 | 13 | 48 |
| `CXAtomicOperator` | 0 | 0 | 3 | 1 | 12 | 0 | 0 | 0 | 0 | 0 | 0 | 90 | 1 | 0 | 0 | 0 | 16 | 91 | 107 |
| `CXExpression` | 3 | 79 | 0 | 0 | 2 | 0 | 7 | 1 | 19 | 2 | 3 | 0 | 0 | 4 | 0 | 8 | 119 | 9 | 128 |
| `CXFunction` | 1 | 7 | 21 | 2 | 4 | 0 | 3 | 1 | 1 | 2 | 1 | 87 | 1 | 3 | 0 | 0 | 45 | 89 | 134 |
| `CXNativeFunction` | 1 | 1 | 0 | 0 | 1 | 0 | 0 | 0 | 1 | 0 | 0 | 0 | 0 | 0 | 2 | 0 | 6 | 0 | 6 |
| `CXPackage` | 1 | 16 | 25 | 1 | 2 | 0 | 2 | 0 | 1 | 0 | 1 | 89 | 1 | 0 | 0 | 0 | 49 | 90 | 139 |
| `CXCall` | 2 | 2 | 0 | 0 | 0 | 1 | 0 | 0 | 4 | 0 | 0 | 0 | 0 | 0 | 0 | 1 | 9 | 1 | 10 |
| `CXProgram` | 3 | 33 | 0 | 0 | 3 | 0 | 0 | 0 | 0 | 0 | 18 | 0 | 0 | 0 | 2 | 0 | 57 | 2 | 59 |
| total | 20 | 292 | 198 | 10 | 182 | 8 | 35 | 15 | 43 | 4 | 39 | 716 | 28 | 26 | 4 | 9 | 857 | 771 | 1628 |



The list was made with a `go/ast` and `go/types` walk of every package,
generated parser sites mapped to their grammar through the `//line`
directives. The call counts agree with the codebase-memory call graph, for
example 30 `MakeArgument` callers, 34 `AddCXTypeSignatureInArray` callers,
19 `AddCXArgInArray` callers and 90 `GetCXArgFromArray` callers. GC here
means Go's garbage collector. `t3-ast/garbage_collector.go` collects the CX
heap in `CXProgram.Memory`, which holds no IR objects.

