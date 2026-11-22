# Error handling

Every failure in this module is one of two classes. A failure the program
cannot prevent is class A and is returned to the caller. Every other failure
is class B and crashes the process at once. Nothing recovers from a crash.

The error design and its phases are in skycoin/skycoin-cx#196.

## Class A, environmental failures

A failure of something outside the program's control. A path that does not
exist, a file that cannot be opened, read or written, a database that is
locked, a connection that is refused or drops, a peer that sends bad data.
These can fail in a correct program, so the library handles them and the
caller decides what to do.

An operation that can fail this way returns a pointer to its module's error
struct, nil on success. It does not return the Go `error` interface.

### Error structs

Errors are plain structs. There is no interface, and there is no shared
error package that every module imports.

Each module or system area defines its own error struct in its own package.
`FileError` in `t0-util` is the only class A struct so far, see
[Error records today](#error-records-today). Its fields are

- the failure code, a specific code for each failure,
- the operation code, which operation failed,
- the error string, the message,
- any context fields the area needs, such as a path or an address.

```go
// in t0-util, next to the file calls
type FileErrorCode uint32
type FileOperationCode uint32

const (
	FileErrorCodeNotFound         FileErrorCode = 1001
	FileErrorCodePermissionDenied FileErrorCode = 1002
)

const (
	FileOperationOpen   FileOperationCode = 1501
	FileOperationCreate FileOperationCode = 1502
)

type FileError struct {
	FileErrorCode      FileErrorCode
	FileErrorOperation FileOperationCode
	FileErrorMessage   string
	FileErrorPath      string
}

func FileOpen(path string) (*os.File, *FileError)
```

```go
file, fileError := workdir.FileOpen(path)
if fileError != nil {
	if fileError.FileErrorCode == workdir.FileErrorCodeNotFound {
		...
	}
}
```

Each module owns its failure codes and operation codes as constants next to
its struct. Each failure has its own code and each operation has its own
code. The constants are prefixed by their area, `FileErrorCode*` for failures
and `FileOperation*` for operations.

Because each module defines its own struct and codes, a module imports no
other package to report an error, and no package changes tier. The codes stay
unique across the whole module through the code ranges below.

### Code ranges

Each area has a block of 1000 codes. Failure codes use the lower half of the
block and operation codes the upper half, so a code alone names its area.
A new area takes the next free block and is added here in the same change.

| Block | Area | Struct | Package | Status |
|---|---|---|---|---|
| 0 - 999 | CX exit and runtime codes | none, `CX_*` constants | `t0-types` | exists |
| 1000 - 1999 | Files | `FileError` | `t0-util` | exists |
| 2000 - 2999 | Paths and the working directory | `PathError` | `t0-util` | planned |
| 3000 - 3999 | Networking | `NetworkingError` | `t5-tcp` | planned |
| 4000 - 4999 | Package store database | `PackageStoreError` | `t0-plbolt` | planned |
| 5000 - 5999 | Package loader | `PackageLoaderError` | `t1-plloader` | planned |
| 6000 - 6999 | LZ4 compression | `LZ4CodecError` | `t0-lz4codec` | planned |
| 7000 - 7999 | Compiler | `CompileErrorRecord`, held as a list | `t3-ast` | exists |

A planned struct does not exist yet. Its block and name are reserved for the
phases of #196. Until then those packages return a Go `error` or an error
string.

In the block starting at B, failure codes run from B+1 to B+499 and operation
codes from B+500 to B+999, so files use 1001 to 1499 and 1500 to 1999. No
check enforces the blocks yet, see #270.

Pick the struct in this order.

1. Use the module's error struct whenever possible.
2. A module may add a more specific error struct for a situation that needs
   special handling, for example one that returns a list of errors, such as
   every compile error of a program.
3. A generic error, such as Go's `error`, may exist, but it is not used where
   a module or system error exists or is required.

### Wrapping at the module boundary

The file and network libraries of Go, and vendored packages, return Go
`error` values. The module that calls them wraps each one into its own struct
right where it is received, at the module boundary, with the failure and
operation codes and the message. No Go `error` crosses the boundary.

Only a binary's `main` turns a class A error into an exit code.

## Error records today

Three error records exist.

- `FileError` in `t0-util/file_error.go` is the class A struct for file
  calls. Its failure codes are `FileErrorCodeNotFound` (1001) to
  `FileErrorCodeFileSystemCallFailure` (1006), and its operation codes are
  `FileOperationOpen` (1501) to `FileOperationSandboxRootOpen` (1505).
- `CompileErrorRecord` in `t3-ast/compileerror.go` holds one compile error.
  Its fields are `CompileErrorRecordCode`, `CompileErrorRecordFile`,
  `CompileErrorRecordLine` and `CompileErrorRecordOperands`. The code is a
  `CompileErrorCode` from `CompileErrorCodeMessage` (7001) to
  `CompileErrorCodeImportGuiBuildMissing` (7048). 7041 is
  `CompileErrorCodeGotoLabelMissing`, a goto to a label that does not exist,
  7042 a slice built-in argument or result of the wrong type, 7043
  `CompileErrorCodeCallResultCall`, a call of a call result such as
  `f() ()`, 7044 `CompileErrorCodeMethodResultMissing`, a method with no
  output used as a value, 7045 `CompileErrorCodeAssignmentCount`, a
  multiple assignment whose targets and values differ in number, 7046
  `CompileErrorCodeArrayLiteralLength`, an array literal with more elements
  than its length, 7047 `CompileErrorCodeStructLiteralFieldRepeated`, a
  struct literal that names a field twice, and 7048
  `CompileErrorCodeImportGuiBuildMissing`, an import of `sys/gl`, `sys/glfw`,
  `sys/al` or `sys/gltext` in a cx built without the cxfx tag. 7000 is
  `CompileErrorCodeNil` and `CompileErrorCodeEnd` is one past the last code.
  The records are kept in a `CompileErrorList` in `CxIrState.CompileErrorList`.
- `RuntimeErrorRecord` in `t3-ast/error.go` holds the runtime error of the
  old loop in `CXProgram.RuntimeError`. Its code is a `CX_RUNTIME_*` or
  `CX_ASSERT` exit code from `t0-types`, and it also keeps the panic value,
  the call stack position and the expression index.

The two records are described in [CX programs](#cx-programs) below.

## Class B, everything else

A failure that cannot happen in a correct program. These are bugs, so they
crash with a strict assert and nothing handles them.

- Precondition violations. A function is called with input that breaks its
  contract, such as a nil program, a negative count or an unknown opcode.
- Invariant violations. A state that must always hold does not, such as a
  stack pointer below its frame or a type code outside the table.
- Postcondition violations. A function's result breaks its contract.
- Mathematical violations. Division by zero, an index out of range, an
  overflow the operation forbids.
- Impossible states. The default case of a switch over every known value, or
  an internal lookup that cannot fail, such as finding a package the compiler
  has just added.

A class B failure is never returned as an error value, never logged and
carried on from, and never turned into a default value.

## Naming

Generic names are banned. Error structs, their fields, methods and code
constants use verbose, lexicographically prefixed, Java style names, so
related names sort together.

- An error struct is named for its area, `<Area>Error`, for example
  `FileError`. A more specific struct keeps the area prefix, so an import
  cycle error of the planned package loader struct would be
  `PackageLoaderImportCycleError`.
- Failure constants are `<Area>ErrorCode<Failure>` and operation constants
  `<Area>Operation<Operation>`, for example `FileErrorCodeNotFound` and
  `FileOperationOpen`. Their types are `<Area>ErrorCode` and
  `<Area>OperationCode`.
- Methods, where a struct needs any, are named `<Noun><Verb>` or
  `<Noun><Noun><Verb>`, such as `ErrorCodeGet` or `ErrorMessageFormat`. No
  error struct has methods yet.
- `Error`, `Unwrap`, `Code`, `Is`, `As` and any other one word generic
  method name are banned on error structs.
- No interface for errors. There is no `Coded` interface. Each caller names
  the concrete struct it handles.
- Errors are not built with `errors.New` or `fmt.Errorf`.

## Nil on success

An operation that succeeds returns a nil error struct pointer. A non-nil
pointer always means failure, and its fields are always filled in. There is
no zero value error struct that means success.

## Asserts

Package `t0-cx-core-assert` (package `assert`) holds the class B checks.

```go
assert.AssertPreconditionTrue(condition bool, message string)
assert.AssertInvariantTrue(condition bool, message string)
assert.AssertPostconditionTrue(condition bool, message string)
assert.AssertUnreachable(message string)
```

```go
func StackFramePop(prgrm *CXProgram) {
	assert.AssertPreconditionTrue(prgrm.CallCounter > 0, "pop of the entry frame")
	...
}
```

A failed assert prints to stderr

```
assert failed: precondition violated at /path/t3-ast/stack.go:42: pop of the entry frame

goroutine 1 [running]:
...
```

and ends the process with `os.Exit(assert.AssertExitCode)`, which is 3, the
same value as `CX_INTERNAL_ERROR`.

The assert calls `os.Exit` rather than `panic`. A panic can be caught by any
`recover` on the stack, including the ones in vendored code that wraps a
callback, and it runs every deferred function on the way out. `os.Exit` cannot
be caught, so a violation can never be hidden, and the output is the same in
every caller. Deferred functions do not run, which is what no recovery means.

The message says what was violated in terms of the code, not what the user
should do. The file, line and stack come from the assert.

## No recover

No code in this module calls `recover`. A test that needs to see a crash runs
the crashing code in a subprocess and checks its exit status and stderr, as
`t0-cx-core-assert/assert_test.go` does. `t1-cxtest` already runs CX programs
this way.

`make check-recovers` fails on a `recover(` call outside `vendor/`. The sites
that still break the rule are listed in `recover-allowlist.txt`, which only
shrinks.

## Panics

A Go `panic` is not used to report an error. The existing panics are converted
in the phases of #196. A panic used as a class B check becomes an assert, and
a panic that reports a class A failure becomes a returned error struct.

## CX programs

The same two classes apply to a running CX program.

Class B failures in a CX program crash it. Division by zero, an index out of
range, a failed cast, stack overflow and heap exhaustion print the CX file and
line, the CX call stack and the message to stderr, then end the process with
the matching `CX_RUNTIME_*` exit code. The CX `assert`, `test` and `panic`
builtins do the same with `CX_ASSERT` or `CX_PANIC`. This holds under the REPL
and any embedding host too.

A compile error is data too. The compiler appends a `CompileErrorRecord`, a
`CompileErrorCode`, a `SourceFileGid`, a line and up to four `StringGid` or
integer operands, to `CxIrState.CompileErrorList` without building text.
`printer.PrinterCompileErrorsPrint` prints the records through a table of
formats indexed by the code when compilation returns. A compile error the
compiler cannot go on after calls `CXProgramCompileErrorExit`, which prints
through the printer the program registered and exits, see #413. A Go error
or lexer message that has no code of its own is the operand of
`CompileErrorCodeMessage`.

A runtime error is data until the run has returned. The old loop records the
exit code, the panic value and the call stack position in a
`RuntimeErrorRecord` in `CXProgram.RuntimeError`, and vm returns a `Stop`. Package
`t5-cx-runtime-printer` then prints the text and the caller exits with the
code, see #413. The old loop still unwinds by panic to the one recover at
the run boundary, which the phases of #196 remove.

Class A failures in a CX program are returned to it. A CX library call that
touches a file, a path or the network, such as `os.Open`, `os.ReadStr`,
`os.Create` or the `tcp` package calls, returns an error value as one of its
results, and the CX program checks it. Today the `os` calls return a success
`bool` or an invalid handle and the `tcp` calls an error string. The phases
of #196 give each of them a coded error value.
