# Execution records

Every way of running a CX program must give the result the program gives
itself. An execution record holds that result once, as hashes of the
program, its input and its output, and `TestExecutionRecords` in `t11-cx`
runs every recorded program in every form and checks it against its record.
See [skycoin-cx#507](https://github.com/skycoin/skycoin-cx/issues/507).

## The tuple

| Hash | Of |
| --- | --- |
| program | the serialized program as `cx build` appends it to the core, from `buildProgramImage` in `t11-cx/build.go` (decision D44) |
| source | the `.cx` files the program names on the command line, one after the other. It is kept beside the program hash for reference and is not part of the record hash |
| input | the decimal number of arguments and a zero byte, each argument and a zero byte, then stdin, and for a program with a net replay record a zero byte, the decimal length of the record, a zero byte and the record (decision D53) |
| output | stdout |
| record | the program, input and output hashes as 32 bytes each, then the exit code as a little endian int32. It names the record |

The input is what the program sees, `os.Args` and stdin. A program that uses the net library may have a replay record beside its first `.cx` file, `test-net-tcp.netreplay` beside `test-net-tcp.cx`, which holds every batch of a live run, see [Networking.md](Networking.md). The harness then runs every form from the record and opens no socket: cx and the image forms with `--net-replay`, a built program with `CX_NET_REPLAY`. The record is part of the input. cx routes an
argument that starts with `++` to the program, and a built program gets its
arguments unchanged, so `./program ++a` sees what `cx program.cx ++a` sees.
Most programs have no input.

The output is stdout and the exit code. Stderr is not part of it. Runtime
errors and failed tests print to stdout, so they are recorded. Stderr holds
what differs from form to form and is no result of the program: the Go
stack of a crash, with its addresses, the refusal of vm for a program that
does not lower, and the messages of `cx build` and of the core.

Every hash is the hash of CX, `cxhash.CxHashOfBytes` in `t0-cxhash`, which
the cipher library hashes with too. It is SHA-256 for now. The lead asked
for a BLAKE hash and which one is decision D23, which changes `t0-cxhash`
only. The file names its hash in the format section, so a file written with
another hash is refused, not misread.

## The record file

[tests/execution-records.txt](../tests/execution-records.txt) is text. Each
section is a header line in brackets naming the section and its columns,
then one entry per line, and a blank line before the next header. Lines
starting with `#` are comments. `t0-textsections` reads and writes this
form, for the package loader too (#509). A program is named by its cx arguments
relative to `tests/`, its flags and files, and the name is the last column
because it may hold spaces.

| Section | Columns |
| --- | --- |
| `[format]` | the format version, 1, and the hash name |
| `[inputs]` | stdin and each argument as Go quoted strings, the argument count before the arguments, then the program. Edited by hand |
| `[records]` | record hash, program hash, source hash, input hash, output hash, exit code, program |
| `[reasons]` | the key of each exclusion reason and its text |
| `[excluded]` | reason key, program |

The recorded stdout of every record is kept in `tests/execution-records/`,
one file named by its output hash, so a mismatch can show the first line
that differs. `.gitattributes` keeps those files byte for byte.

## The programs

The programs are every `.cx` file under `tests/` on its own, and every
`t1-cxtest` entry that passes flags or several files, without its
`--executor` flag, since the forms choose the executor. Each one is either a
record or excluded with a reason. The ones excluded now are these.

| Reason | Why |
| --- | --- |
| compile | does not compile on its own: a compile error test, a file of a package, or a flag only test |
| gui | opens a window, needs the cxfx build and a display |
| network | uses the network and has no net replay record beside it |
| random | prints random values |
| clock | prints or depends on the clock |
| process | runs other cx programs |
| shared | writes a fixed file that parallel runs share |
| ast | prints the AST while it compiles, which a built program does not |
| varies | two runs on the old loop print differently |
| crash | the old loop crashes with a Go panic, which is a bug, not a result |
| timeout | does not finish in two minutes |
| gostack | prints a Go stack trace by design |
| path | prints the path of its source file, which has another separator on Windows |

`make records` finds every reason but `gostack` and `path` from the source,
the compiler and two runs on the old loop. Those two are listed by hand in
`executionRecordExcludedByHand`.

## The forms

| Form | How it runs |
| --- | --- |
| old | `cx --executor=old` |
| vm | `cx --executor=vm`, when every function lowers |
| image-old | compiled, turned into the image `cx build` appends, loaded again with `ast.Deserialize` and run on the old loop |
| image-vm | the same on vm, when every function lowers |
| built | the executable `cx build` writes, which runs vm when the program lowers and the old loop otherwise. Not for a program that passes a run time flag of cx, such as `--executor`, which a built program cannot take, or one that uses an opcode the core leaves out. `-heap-max` is stored in the built program, so programs that pass it are compared |

A later native form, such as the LLVM target, is added as one more form.

## The harness

`TestExecutionRecords` runs in `make check` as part of `go test ./...`. For
each record it runs every form on the input and compares the exit code and
the hash of stdout with the record. A mismatch names the record, the form
and the first line that differs from the recorded stdout.

```
record examples/hello-world.cx, form old: stdout differs from the record, first differing line 2: "Hello Earth!\n", recorded "Hello World!\n"
```

It fails when a program of `tests/` is neither recorded nor excluded, when a
listed program is gone, when an input or record hash does not match its
columns, and when a recorded stdout is missing or does not hash to its name.
A record whose program or source hash differs from what the compiler or the
source gives now is counted in the log and does not fail, since every change
of the language version or the serialized form changes every program hash.
The output is what is checked.

It builds the sixteen cores of `cx build` first, then logs the runs checked by
form, the forms not run and why, the core sizes and the sizes of the
appended programs.

## make records

`make records` writes the record file and the recorded stdout from two runs
of each program on the old loop. It keeps the inputs section and never runs
as part of `make check`. Run it after adding, removing or changing a test
program, or when a change of output is intended, and review the diff before
committing it. A changed output hash is a changed result.

## What it absorbed

| Test | Now |
| --- | --- |
| `TestSerializeRoundTrip` | removed. The image-old and image-vm forms run every program it ran after the round trip, through the image a built program loads |
| `TestBuildPrograms` | removed. The built form runs every program it built, and the harness logs the core and program sizes it logged |
| `TestLowerCompare` | keeps the memory comparison after both loops and the lowering statistics. Its comparison of stdout and exit code of `cx --executor=old` and `cx --executor=vm` moved to the vm form |
| `TestCompareExpected` | stays. It checks the hashes of `t0-benchmarks/compare/expected.txt` against the math, an oracle outside CX, and those programs run for seconds each at their one scale |
| `TestBuildCoreRefuses`, `TestBuildLibraries` | stay. They check refusals and library linking, not results |
| `make test` (`t1-cxtest`) | stays. It checks the expected exit code and output of each test, written by hand |
