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

Errors are plain structs. There is no interface.

Each module or system area has one error struct, for example a file system
error, a networking error and a package loader error. Its fields are

- the error type code, a specific code for each failure,
- the operation code, which operation failed,
- the error string, the message,
- any context fields the area needs, such as a path or an address.

```go
// in the file system module
type FileSystemError struct {
	FileSystemErrorCode      cxerrors.ErrorTypeCode      // FileSystemErrorCodeNotFound, ...
	FileSystemErrorOperation cxerrors.ErrorOperationCode // FileSystemOperationOpen, ...
	FileSystemErrorMessage   string
	FileSystemErrorPath      string
}

func FileSystemFileOpen(path string) (*FileSystemFile, *FileSystemError)
```

```go
file, fileSystemError := FileSystemFileOpen(path)
if fileSystemError != nil {
	if fileSystemError.FileSystemErrorCode == cxerrors.FileSystemErrorCodeNotFound {
		...
	}
}
```

The failure codes and operation codes come from the global code table of
#196. Each failure has its own code, and each operation has its own code. The
constants are prefixed by their area, such as `FileSystemErrorCode*` for
failures and `FileSystemOperation*` for operations.

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
  `FileSystemError`, `NetworkError`, `PackageLoaderError`. A more specific
  struct keeps the area prefix, for example `PackageLoaderImportCycleError`.
- Failure constants are `<Area>ErrorCode<Failure>` and operation constants
  `<Area>Operation<Operation>`, for example `FileSystemErrorCodeNotFound` and
  `FileSystemOperationOpen`.
- Methods, where a struct needs any, are named `<Noun><Verb>` or
  `<Noun><Noun><Verb>`, for example `ErrorCodeGet` or `ErrorMessageFormat`.
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
and any embedding host too, since nothing recovers. The failing opcode prints
and exits itself, instead of panicking up to a recover at the run boundary.

Class A failures in a CX program are returned to it. A CX library call that
touches a file, a path or the network, such as `os.Open`, `os.ReadStr`,
`os.Create` or the `tcp` package calls, returns an error value as one of its
results, and the CX program checks it. Today the `os` calls return a success
`bool` or an invalid handle and the `tcp` calls an error string. The phases
of #196 give each of them a coded error value.
