# Gid variants

Every handle in CX is an `int64`. Bits 0 to 31 are always the slot index and
bits 56 to 63 the type tag, so `objmodel.GidIndex`, `objmodel.GidTypeTag`
and a tag check work on every handle. Bits 32 to 55 differ by variant
(decision D154). Tag `0x00` is `GidNil` and bit 63 stays 0, so every valid
handle is positive.

The tags are one table, `gidtag.Table` in
[t0-cx-core-gidtag/gidtag.go](../t0-cx-core-gidtag/gidtag.go). Each row has a
name, a value, a variant, a range and the package that owns it.
`make generate-gidtags` writes a constant per tag and `gidtag.NameTable`
into `gidtag_gen.go`, and `make check-gidtags`, part of `make check`, fails
when that file is stale. `TestTableTags` fails when two tags share a value
or a tag lies outside its range. A package declares its tag constant from
the table, `ImageTag uint8 = gidtag.CxirImage`, and never as a number.

* [Variants](#variants)
* [Tag ranges](#tag-ranges)
* [Tags in use](#tags-in-use)
* [Adding a tag](#adding-a-tag)

## Variants

### Object Gid

```
63      56 55                     32 31                              0
| tag    |       generation         |              index             |
```

The Gid of [ObjectModel.md](ObjectModel.md). A manager in t3-cx-core-ast
releases and reuses slots, and the generation, 1 to `0xFFFFFE`, goes up by
one on each release, so a stale Gid is caught. A slot whose generation
reaches `objmodel.GidGenerationLast` (`0xFFFFFF`) is retired and never
reused. `objmodel.GidCompose` builds one.

### Interned Gid

The object layout with the generation fixed at 1 (`GidGenerationFirst`),
since a slot is never released. Equal content has an equal Gid, so a
compare is one integer compare. The string pool, the source files, the
type name and member name pools of sys/reflect and the name pool of
sys/cxir are interned. The function and package handles of sys/reflect and
sys/meta have this layout too, since a function or package record is never
released.

### Type handle

```
63      56 55          40 39      32 31                              0
| tag    |  generation   |  kind    |              index             |
```

The type handle of sys/reflect (decision D153), tag `ReflectType`. The
index is the type number in the reflect table, the kind byte is the
`reflecttable.Kind*` of the type, so it is read with a shift and a mask,
and the 16 bit generation stays 1 because types are never released.
`reflecttable.TypeGidCompose` builds one, and `reflecttable.TypeIndex`
refuses a handle whose kind byte differs from its record.

### Library resource handle

The object layout, for a resource a system library frees. The generation
starts at 1 and goes up on each release, and after `0xFFFFFF` it starts
again at 1 instead of retiring the slot. In use for the images of
sys/cxir and the ASTs of sys/compiler. The running image is index 0,
generation 1, and a writable image is its store slot plus 1.

Files, sockets and cxfx windows are not Gids today. They are `i32` handles
into fixed tables, so the range `0x20` to `0x3F` is still empty.

t2-cx-runtime-ir and t4-cx-library-compileropcodes compose and read their
handles with their own shifts instead of objmodel, which brings `fmt` into
cx-core-image, too large for a microcontroller.

## Tag ranges

| Range | Use |
| --- | --- |
| `0x01` to `0x1F` | compiler and IR objects, interned pools |
| `0x20` to `0x3F` | system library resources |
| `0x40` to `0x5F` | sys/reflect, sys/meta, sys/cxir and sys/compiler handles |
| `0x60` to `0x7F` | reserved |

`0x00` is `GidNil`, and `0x80` to `0xFF` are never used. A range is a
block of tags for one part of CX, not one variant: `0x40` to `0x5F` holds
type, interned and library resource handles.

## Tags in use

| Tag | Name | Variant | Owner | Handle |
| --- | --- | --- | --- | --- |
| `0x01` | `CXProgram` | object | t3-cx-core-ast | `CXProgramGid` |
| `0x02` | `CXPackage` | object | t3-cx-core-ast | `CXPackageGid` |
| `0x03` | `CXFunction` | object | t3-cx-core-ast | `CXFunctionGid` |
| `0x04` | `CXNativeFunction` | object | t3-cx-core-ast | `CXNativeFunctionGid` |
| `0x05` | `CXStruct` | object | t3-cx-core-ast | `CXStructGid` |
| `0x06` | `CXExpression` | object | t3-cx-core-ast | `CXExpressionGid` |
| `0x07` | `CXAtomicOperator` | object | t3-cx-core-ast | `CXAtomicOperatorGid` |
| `0x08` | `CXLine` | object | t3-cx-core-ast | `CXLineGid` |
| `0x09` | `CXArgument` | object | t3-cx-core-ast | `CXArgumentGid` |
| `0x0A` | `CXArgumentDebug` | object | t3-cx-core-ast | `CXArgumentDebugGid` |
| `0x0B` | `CXTypeSignature` | object | t3-cx-core-ast | `CXTypeSignatureGid` |
| `0x0C` | `CXTypeSignature_Array` | object | t3-cx-core-ast | `CXTypeSignature_ArrayGid` |
| `0x0D` | `CXTypeSignature_Struct` | object | t3-cx-core-ast | `CXTypeSignature_StructGid` |
| `0x0E` | `CXCall` | object | t3-cx-core-ast | `CXCallGid` |
| `0x0F` | `String` | interned | t2-cx-core-stringpool | `stringpool.StringGid`, and the member names of sys/reflect |
| `0x10` | `SourceFile` | interned | t3-cx-core-ast | `SourceFileGid` |
| `0x11` | `ReflectTypeName` | interned | t2-cx-library-reflecttable | type name, `reflect.CxTypeNameGet` |
| `0x12` | `CxirName` | interned | t2-cx-runtime-ir | name of the sys/cxir image store |
| `0x40` | `ReflectType` | type | t2-cx-library-reflecttable | type, `reflect.CxTypeOfValueGet` |
| `0x41` | `ReflectFunction` | interned | t2-cx-library-reflecttable | function, `reflect.CxStructMethodFunctionGet`, `meta.CxFunctionDefinitionBegin` |
| `0x42` | `MetaPackage` | interned | t2-cx-library-reflecttable | package, `meta.CxPackageDefinitionBegin` |
| `0x43` | `CxirImage` | library resource | t2-cx-runtime-ir | image, `cxir.CxProgramImageCurrentGet`, `cxir.CxProgramImageCreate` |
| `0x44` | `CompilerAst` | library resource | t4-cx-library-compileropcodes | AST, `compiler.CxCompilerSourceParse` |

The member names of sys/reflect, fields, methods, functions and packages,
carry the `String` tag but index the member name pool of the reflect
table, not the string pool of the IR. A member name Gid and a `StringGid`
with the same index name different strings.

## Adding a tag

Take the next free value of the range, add a row to `gidtag.Table`, run
`make generate-gidtags`, and declare the constant of the owning package as
`gidtag.<Name>`. A tag is fixed forever, and a removed handle keeps its tag.
