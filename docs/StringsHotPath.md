# Strings on the run path

## Purpose

The interpreter uses no string operations on its run path, only fixed size
integers and Gids (#407, #411 rule 5). Strings appear at the boundaries:
error text, printing, and the Go and OS calls of the system libraries. This
document defines the STRHOT codes of #412 that mark every string operation
the old loop can still reach, and the check that keeps the list complete.

## The run path

The run path is `RunCxAst` in t4-execute and every opcode handler, a
function with the `OpcodeHandler` signature, followed through static calls
into this module. `t0-cx-tools-strhot` type checks the packages of the `cx`
build with the default tags and walks those calls. Interface calls and
function values other than handlers are not followed. t6-cxfx is built only
with `-tags cxfx`, and its codes have the status `cxfx`, open in that build
only. `./check-strhot.sh -tags cxfx` checks it (#432).

A string operation is a comparison, concatenation, switch, index, slice or
range on a string, a conversion between a string and bytes, a lookup in a
map keyed by strings, a call into `fmt`, `strings`, `errors` or `log`, and a
call that passes or returns a string. A constant expression and `panic` of a
constant are not operations.

## Codes

[strhot-codes.txt](../strhot-codes.txt) lists one code per function with its
class and status. A code keeps its number. A new finding takes the next
free number, and a fixed one is marked removed, never deleted.

- `I`, every instruction, success path.
- `O`, every run of that opcode, success path.
- `P`, printing a value, the print and printf opcodes and the stack dump.
- `E`, only on a failure branch: the text of a panic, an assert or an error.
- `B`, a boundary. The string goes to Go, the OS or compiler code that an
  opcode runs: file and process opcodes, regexp, tcp, the CPU
  profile, read, serialize and the affordance queries.

I and O are what #412 removes. P, E and B may stay with their comment. No P
code is open. Printing appends to the format buffer of the program, names
viewed from the string pool and the byte tables of the type, built-in and
library function names, and numbers with `strconv.Append*`.

Each site has a comment naming its code and class, either directly above
its statement or above its function, which then covers every site in it.

```go
// VIOLATION STRHOT-234: E, error or assert text, built on a failure branch only, #412.
func (cxprogram *CXProgram) GetCXArg(index CXArgumentIndex) *CXArgument {
```

The manager functions are generated, so their codes are in the table of
t0-cx-core-objmodelgen.

## Check

`make check-strhot`, part of `make check`, runs
[check-strhot.sh](../check-strhot.sh). It checks the default build, then the
cxfx build when cgo and the GL, X11 and OpenAL headers are there, and
`make check-strhot-gui`, which the vet-gui job of CI runs, fails when the
cxfx build does not compile. It fails when a site on the run path
has no code, when a comment names a code or class that differs from
strhot-codes.txt, and when an open code has no site left. `./check-strhot.sh
-l` lists every site with its code, and `-inventory -why` lists the sites
without one with a suggested class and the call chain that reaches them.
