# Changelog

### Unreleased

#### Language

- Language version 2611200002. Every built-in library lives under `sys/` and needs an explicit import, for example `import "sys/os"`. A bare import of a built-in package, an unknown `sys/` path and a built-in package used without its import are compile errors (#380)
- The `str` functions moved to `sys/string` with verbose camel case names, for example `str.concat` is `string.Concatenation` and `str.compare_by_value` is `string.OrderingByByteValue`, and `glfw.func_i32_i32` and `glfw.call_i32_i32` are `glfw.RegisterCallbackI32I32` and `glfw.InvokeCallbackI32I32`. Opcodes keep their numbers, `OpcodeRenameTable` and the `former_names` column of `docs/opcodes.csv` record the renames, and an old name is a compile error that names the new one (#380, #382, #391)
- `string.ByteLength`, opcode 792 (#382)
- Language version 2611200001, formerly revision 1, see the Language versions section of LanguageGuide.md (#183, #296)
- Opcode numbers are fixed in `constants.OpcodeLifecycleTable` and `docs/opcodes.csv`. Every opcode keeps the number it had before #197, deleted opcodes leave gaps that are not reused before their reassignable version, `ptr.eq` and `ptr.uneq` are 589 and 590, and the cxfx opcodes are 591 to 791 whatever the build tags (#296)
- Strings have no built-in operators. `==`, `!=`, `<`, `>`, `<=`, `>=`, `+` and every other arithmetic or bitwise operator on `str` is a compile error that names the function to use (#183, #197)
- `<`, `>`, `<=` and `>=` on pointers are compile errors. Pointer `==` and `!=` compare addresses, and now work for every pointer type instead of only `*i32` (#183, #197)

#### Added

- The object model managers, stage 3 of #299. Package `objmodel` (`t1-cx-core-objmodel`) holds the Gid layout and checks, and `t3-ast` has a generated `<obj>_gid.go`, `<obj>_mng.go` and manager test for each of the 14 IR object types, plus `CxIrState` with every manager. Nothing in the compiler uses them yet (#299)
- `t0-cx-core-objmodelgen` generates the Gid and manager files from one template, `make generate-objmodel` runs it, and `make check-objmodel`, part of `make check`, fails on stale generated files, a Def file with more than its struct, or a Fon or Fof file that names a manager or `CxIrState` (#299)
- `docs/ObjectModel.md`, the object model standard for the compiler IR. Def, Mng and Gid files per object type, the generational index layout, one `CxIrState` owning every manager, no pointers at rest, and every manager empty at teardown. `docs/objectmodel-sites.csv` lists every site still to migrate, and `BenchmarkCompile` in `t11-cx` is the compile speed baseline (#299)
- `docs/IR.md`, what the compiler emits. It describes the in-memory program the VM runs, its arenas, expressions, operands and memory layout, a compiled example, and the serialized form
- `str.equal_by_value`, `str.equal_by_pointer` and `str.compare_by_value` (#184)
- Serialized programs carry the language version `constants.LanguageVersionCurrent`, the integer yymmddnnnn, in `CXProgram.Version`, and `Deserialize` rejects a program from another version (#183, #197, #296)
- `docs/Conventions.md`, every repository rule in one list, linked from the README (#296)
- `docs/ErrorHandling.md`, the two-class error policy: path, file and network failures return the error struct of their own module, with failure and operation codes from that module's code range, every other failure is an assert that crashes, and nothing recovers. `t0-cx-core-assert` holds the asserts, and `make check-recovers` fails on a new `recover(` call (#270)

#### Removed

- `t3-packageloader/makefile`, whose targets pointed at directories that no longer exist and used `sudo rm`. `go test ./t3-packageloader` now runs load, tree, graph and save end to end (#113)
- `loader.FileHashMap` and `loader.PackageHashMap`, which were written but never read. FileHashMap was keyed by base file name, so files of the same name in different packages overwrote each other (#94)
- `ast.Debug`, `ast.CopyProgramState`, `ast.GetSerializedStackSize`, `ast.GetSerializedDataSize` and `CXProgram.PrintAllObjects`, which nothing called (#162)
- The `t4-assert` package, which nothing imported (#107)
- The `str.eq`, `str.uneq`, `str.lt`, `str.lteq`, `str.gt` and `str.gteq` opcodes, deprecated in version 2611200001. Their numbers stay empty, and serialized programs from before that version must be recompiled (#183, #197, #296)

#### Changed

- `CXTypeSignature_Array` objects live in `CxIrState.CXTypeSignature_ArrayMng`. Array and slice signatures are allocated there, `CXProgram.TypeSignatureForArrays` and `AddCXTypeSignatureArrayInArray` are gone, and `GetCXTypeSignatureArrayFromArray` reads the manager by slot index (#299, #316)
- `CXLine` objects live in `CxIrState.CXLineMng`. `MakeCXLineExpression` and `Deserialize` allocate them there, `CXProgram.CXLines` and `CXProgram.AddCXLine` are gone, and `GetCXLine` reads the manager by slot index. `GetOperation` returns a nil line for an expression that is not a line instead of a new empty one (#299, #311)
- Every program has its own `CxIrState`, reached through `CXProgram.CxIrState` until phase 5 of `CXProgram`. `MakeProgram` and `Deserialize` make and initialize it, see "Until the program is managed" in `docs/ObjectModel.md`. The `CXArgumentDebug` allocation sites stay as they are until its Gid phase, because the object is shared through a pointer field (#299, #306)
- Each IR struct of `t3-ast` lives in its own `<obj>_def.go`, for example `cxargument_def.go`, as the object model asks. `CXArgumentSlice`, `CXArgumentStruct` and `CXArgumentPointer`, which nothing used, are removed (#299)
- Every site the object model will change, 1628 in all, carries a two line `// OBJMODEL:` comment with its target manager call or Gid, its phase, its issue and its id in `docs/objectmodel-sites.csv`. The `// VIOLATION:` comments for IR objects are gone (#299)
- `t3-ast/tostring.go` prints struct values through one helper and reads atomic values through one switch, and the array and pointer array branches of `GetPrintableValue` are one. The output is unchanged (#238)
- `loader.ReadProgram` reads a stored program with its packages and files. The graph, tree, encoder and file output use it instead of four copies of the same database walk, and `encoder.SaveFilesToDisk` takes a `loader.StoredPackage` (#92)
- `bolt.Init` takes the database file path and keeps the database open until `bolt.Close`, and `packageloader` needs `-db`. The database was opened on every call and kept next to the executable, so `go run` lost it (#99)
- The v2 package loader, tree and CLI replace v1 under the v1 paths, and `t1-plloader2`, `t2-pltree2` and `t3-packageloader2` are removed (#181)
- LZ4 state lives in `lz4codec.Codec` (`t0-lz4codec`), which the caller initializes with explicit size limits, passes to `SerializeCXProgram` and `Deserialize`, and closes. Using it before Init or after Close returns an error. The serialized bytes are unchanged (#190)

#### Fixed

- The f64 `%` operator was registered under the name `f32.mod`, so calling `f32.mod` by name ran the f64 handler. It is `f64.mod` now (#296)
- Taking the address of a slice, `ps = &s`, panicked the compiler with "type is not known", and so did passing `&s` to a function. Assigning through and from `*ps` now works, and `printf("%v", *ps)` prints the slice (#262)
- `*p` on a struct pointer kept the pointer type, so `q = *p`, `*p = q` and passing `*p` as a struct were compile errors, and `printf("%v", *p)` printed the address. It is now the struct value (#261)
- A `test()`, `assert` or `panic` call whose first two arguments differ in type, and an operator on struct values, reported the last line of the function instead of the line of the call (#259)
- `printf("%v")` printed wrong values for structs, arrays and slices of structs, global arrays, struct fields reached through a field path and a dereferenced array pointer, and panicked on a struct with an array or struct field. Assigning `&array` to an array pointer copied the first element instead of the address (#256, #19)
- A failing `test()` printed the line of the statement before it, or no file and line 0 when it was the first statement. Every expression statement now starts with its own line marker (#254)
- `StartCPUProfile(path, rate)` in CX wrote `<argv0>_<path>_cpu.pprof` next to the cx executable instead of `path`. It now writes the path it is given (#118)
- A failing `test()`, `assert` or `panic` printed the raw bytes of both values to stdout before its message. Only the message is printed now (#239)
- A call to a function that does not exist in an imported package, such as `os.NoSuchFunction`, printed "struct not found" and the program still ran and exited 0. It is now a compile error naming the identifier and its line, and cx exits 1 (#221)
- `cx.POINTER_SIZE` used the pointer size as its constant code, so it overwrote `cx.ASSERT` (4) or, with ptr64, `cx.RUNTIME_INVALID_ARGUMENT` (8). It now has its own code after the exit codes (#193)
- `SerializeCXProgram` gave different bytes for the same program on each call, and a round trip could move functions and structs to another package. Packages, structs, functions and imports are now walked in name order, and the scratch name maps are left out of the bytes (#186, #250)
- `packageloader` ignored a stray positional argument and rejected extra ones only by counting flags. Any positional argument is now an error (#102)
- `packageloader -graph` listed an external import such as `os` once for every package that imported it, each time under a new id. It now gets one id (#105)
- The package loader kept its own list of built in packages, which lacked `regexp`, so a program importing it failed to load with "package regexp not found". It now uses the `t0-packages` list the compiler uses (#95)
- The package loader, `-graph` and `-tree` took `import` inside a block comment as an import, so the loader rejected a valid program. One scanner, `loader.Imports`, now skips comments and string literals for all three (#98)
- The package loader found only self imports and loops of two packages, so a -> b -> c -> a loaded. It now walks the imports depth first and names the whole loop (#97)
- The package loader stored `Blake2Hash` as raw bytes in a JSON string, so it read back changed. It is now hex encoded, which changes stored data (#93)
- The structser benchmark wrote 0-byte files and timed empty output, because its structs had only unexported fields, which the encoder skips (#178)
- The `t0-util` file helpers joined the working dir three different ways, so with one set `CXCreateFile` doubled it and the others dropped the separator. All six now use `filepath.Join` (#89)
- `os.Run` with an empty command returned `OS_RUN_START_FAILED` instead of `OS_RUN_EMPTY_CMD`. It split the command on single spaces, so two spaces passed an empty argument and quotes were passed through. It now splits on runs of spaces and tabs, and single or double quotes group a word (#118)
- Every `regexp.Compile`, `regexp.MustCompile` and `Regexp.Find` call added an argument to the program, so a loop grew it without bound. A failed `regexp.Compile` stored a nil `Regexp`, and `Find` on it was a Go nil dereference. It is now a `CX_RUNTIME_ERROR` (#120)
- Running with an op limit (`RunCompiled` with `maxOps` above 0) printed a step trace into the program's output, grew the program, and could stop an unfinished program as terminated. The limit now only counts operations, and the tracer is removed (#110)
- Some compile errors and runtime errors reported line 0, because identifier and number tokens lost their line in the lexer (#217)
- A slice of slices literal with exactly two inner literals, such as `[][]i32{[]i32{0, 1}, []i32{2, 3}}`, left the second inner slice empty (#216)
- `str.substr` with a negative, reversed or too large range gave a raw Go slice panic. It now fails with `CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE` (#179)
- Calling a function that does not exist in a built in package, such as `str.nosuchfn`, reported the error and then panicked. It now stops with the compile error (#177)
- CX programs can call the `regexp` package. Using a package type without importing the package is now a compile error, not a compiler panic, and a runtime error in a function with a struct local no longer panics while printing the stack (#188)
- The REPL `Eval` ran a program that failed to compile and returned whatever it printed. It now returns `compilation error` (#189)
- The REPL leaked operator state from one `Eval` into the next, so an unrelated second program failed to type check (#174)
- Slices of slices were wrong with ptr64 builds, and `append(s, a, b, c)` wrote `a` three times. `make check` now also runs the CX tests with a ptr64 build (#176)
- The package loader silently dropped a file whose package did not match its directory. It now reports the mismatch (#181)

### skycoin-cx, 2026-11-17 to 2026-11-20

skycoin/skycoin-cx was created on 2026-11-17 by merging cx, cx-chains,
cx-tracker, cx-playground, cx-gui-helper, cx-benchmark, cx-games and
cx-game into one repository, starting from skycoin/cx develop 0c08a443.
Commits without a pull request are cited by hash. Everything below this
section is skycoin/cx history.

#### Language

- String `<`, `>`, `<=` and `>=` work. The four operators were never registered, and `>` computed `>=` (#163)
- Calls to functions with no inputs, or with atomic typed inputs, now get the argument count check they skipped (#34)
- The constant `cx.RUNTIME_NOT_IMPLEMENTED` resolves. It was registered as `cx.RUNTIME_NOT_INPLEMENTED` (#169)

#### Added

- `ParseSourceStrings` parses source held in memory and returns a compile error instead of exiting, so the parser can be embedded, for example by the playground server (a7be6405)
- The CX unit test library `lib/utest`, moved in from cx-games (5eab2468)
- The CX and Go benchmarks from cx-benchmark, now in `t0-benchmarks` (06b8c9a6, #1)
- `CLEANUP.md`, listing every path of the merged CX repositories with its action (cef93cad)
- CI runs `make check` on Linux, macOS and Windows, plus a GUI vet job using the new `make vet-gui` (#37)
- The tier rule. `check-tiers.sh` checks that a package only imports lower tiers and sits at `<dir>/<file>.go` (#1)
- The init rule. `check-inits.sh` fails on `func init()` outside package main (#10, #85)
- The interface rule. `make check-ifaces` fails when an interface embeds another interface (#82)
- The State rule, with a VIOLATION comment at every package var, pointer at rest and object creation outside a manager (#11)

#### Changed

- Packages now set up from explicit `Init` calls that `t11-cx` makes in order after parsing flags, so `--heap-initial` sizes the program. `MakeProgram` only returns a new program (#10, #85)
- The 32 bit pointer build is the default and `ptr64` opts in. `cipher`, `cxos` and `regexp` are built by default with `nocipher`, `nocxos` and `noregexp` to opt out. `cxfx` stays opt in (09408949, #37)
- The package loader returns wrapped errors instead of dropping them or exiting, and `ParseArgsForCX` gains an error return (#70)
- The package loader checks its arguments at entry, and `t3-packageloader` requires exactly one action and `-name` (#76)
- Bolt is the only package store. The database argument is gone from `LoadCXProgram`, `SavePackagesToDisk`, `SaveFilesToDisk`, `GetImportTree`, `GetImportGraph` and `GetImportFiles` (#114)
- `t0-plbolt` uses `go.etcd.io/bbolt` in place of the archived `boltdb/bolt`, and dependencies are updated and vendored (b91dc227, #61)
- The random program generator works on type signatures (#53)
- The module is renamed to skycoin-cx and every package is a top level `t<N>-<name>` directory, where N is one more than the highest tier it imports. `cmd/cx` is now `t11-cx` (09408949, #1)
- `main` is the only long-lived branch. CI, the README badge and the contributing guide moved from develop to main (#69)
- The README describes skycoin-cx, with install and build steps, test targets, status and history, and spelling fixes (#2, #62)
- The parsers are generated with `golang.org/x/tools/cmd/goyacc`, a go.mod tool (09408949)
- The packageloader2 copies of bolt, encoder, file output and graph are shared with packageloader (09408949)
- `make check` passes. Go tests that still fail are commented out with their cause and issue (#33, #53)
- Failing CX program tests are tagged as known issues #13 to #29 (#34)
- Package loader oddities carry NOTE comments pointing at their issues, and unclear local names are renamed (#142)
- `t7-cxmobile` is marked as not compiling, see #35 (#36)
- The docs note that bolt supersedes redis (#180)

#### Removed

- Redis support in the package loader, including `t0-plredis`, the `-redis` flag of `t3-packageloader` and the CI redis server (#79, #114)
- The modernc goyacc copy in `cmd/goyacc`, which did not build (09408949)

#### Fixed

- `cx` with no arguments starts the REPL instead of exiting silently (#168)
- `cx` matches subcommands exactly, so `cx conversions.cx` runs the program instead of printing the version (#4)
- `cx` no longer panics at startup under js/wasm (#3)
- `-stack-size` was never applied (#34)
- Lexer errors and a void call used as a `for` condition panicked instead of reporting an error (#34)
- A precedence bug in `WritePrimary` wrote past the end of memory (#34)
- `os.GetWorkingDirectory` always returned an empty string (#164)
- `tcp.Dial` and `tcp.Listen` ignored their arguments, closed what they opened and crashed when the dial or listen failed (#171)
- REPL output over 64 KiB blocked until Eval timed out, and errors left stdout pointing at a pipe (#166)
- Stepping a program with `RunCompiled` panicked after returning from a call on the caller's last line (#172)
- LZ4 compression panicked on input it could not shrink, and program deserialization ignored decode errors (#173)
- `ExtractAllDeclarations` hung on a file with a bad package line (#165)
- The random program generator panicked on every run (#53)
- The package loader hung on an import loop, and the tree printer overflowed the stack on a cycle (#76)
- Package loader paths panicked on a file with no directory or a short `-path` and depended on the working directory (#72)
- The package loader tests crashed under `-race` with `boltdb/bolt` (#61)
- The CX test runner ran tests when any tag was enabled, passed a program that exited 0 when an error was expected, and never timed out (#34)
- The test runner was given a relative cx path, so 44 tests failed (bdab488b)
- The build failed without `-tags ptr32` (09408949)
- The Go Ackermann benchmark computed the wrong function (#167)
- The struct serialization benchmark wrote its output with mode 01204 instead of 0644 (#170)

#### Security

- `SavePackagesToDisk` refuses stored file names that are not a single path element, so a database entry cannot write outside the save directory (#72)
- CI actions are pinned to commit SHAs (#37)

### v0.8.0
* Additions
  * Preperation for CX 1.0 Milestone
  * Massive internal compiler refactoring. Too many changes to list. See commit log

### v0.7.3
* Additions
  * Preperation for CX 1.0 Milestone

### v0.7.2
* Additions
  * CX package manager:
    - Added a flag for setting a CX workspace. This flag overrides the environment variable .
	- Import statements are now aware of the possibility of importing libraries that are defined in a CX workspace.
	- If the user has not set a CXPATH through an environment variable or --cxpath flag, CX will use by default `~/cx` and create all the necessary subdirectories.
	- If the directory supplied by the user to be used as CXPATH does not contain the following subdirectories: `src/`, `pkg/` and `bin/`, CX will create these subdirectories.
	- Increased opengl version from 2.1 to 3.2.
	- Increased glfw version from 3.2 to 3.3.
    - Added opcodes for reading/writing from/to binary file.
    - Added .hdr file loader.
	- Added opengl bindings : glClearBufferI, glClearBufferUI, glClearBufferF, glBlendFuncSeparate, glDrawBuffers.
	- Added glfw bindings : glfw.GetWindowContentScale, glfw.GetMonitorContentScale.
	- Added glfw constants : glfw.CocoaRetinaFramebuffer, glfw.ScaleToMonitor.
	- Added opengl constants : GL_NONE, GL_RED, GL_RGBA16F, GL_HALF_FLOAT, GL_UNSIGNED_INT_24_8, GL_R8.
	- Added opcodes for reading binary files : os.Seek, os.ReadUI16, os.ReadUI32, os.ReadF32, os.ReadUI16Slice, os.ReadUI32Slice, os.ReadF32Slice.
    - Added TextureGetPixel function.
    - Added basic Regexp library.
	- Added basic Cipher library.
* Changes
  * Updated affordances to play well with newer language features.
  * Removed cx-games as a module. It was just confusing as users would be
    redirected to an outdated version of the repo and the games are already
    mentioned in the README.
  * Improvements to `PrintProgram` (the function associated to the meta-command `:dp` in the REPL); the types printed to input/output parameters now print indexes, dereferences to fields and if they come from an external package.
  * Improvements to memory management when dealing with CX chains.
  * Multidimensional array literals are now working properly.
  * Multidimensional slice literals are now working properly.
  * Added i8/16/ui8/ui16/ui32/ui64 types.
  * Added str.lastindex built-in function.
Libraries
  * Add json bindings for reading json files: Open, Close, More, Next, Delim, Bool, Float64, Int64, Str.
  * Add json cx library to ease json parsing in cx.
  * Add HTTP library.
* Fixed issues
  * #306: Can't print double quotes.
  * #321: Can't do math inside 2nd square brackets (indexer) of an expression.
  * #419: Initializing array or slice with values equivalent to nil.
  * #481: Array literals: Problem with temporary variables in multi-dimensional arrays.
  * #482: Slice literals: Problem with temporary variables in multi-dimensional slices.
  * #131: Panic when using arithmetic operations.

* Documentation
  * ...
* Miscellaneous
  * ...

### v0.7.1 released 2019-07-26
* Additions
  * Added capability of storing heap objects in a CX chain's program state.
  * Added a function that prints information about the heap for debugging purposes (`debugHeap` in cx/utilities.go).
* Changes
  * Redesign of CX's garbage collector.
  * Changes to several functions that relied on allocating objects on the heap.
  * In previous versions of CX the data segment was living inside the heap segment. Now the data segment is properly separated from the heap segment.
  * Moved CX book sources to github.com/skycoin/cx-book. The releases and any code updates will be pushed to that repository.
    * The README file notifies the users about this change.
* Libraries
  * ...
* Fixed issues
  * #286: Compilation error when struct field is named 'input' or 'output'
  * #323: Installation issues on Windows after merging #320
* Documentation
  * ...
* Miscellaneous
  * ...

### v0.7.0 released 2019-06-15
* Additions
  * CX can now generate addresses to be used by CX chains.
  * New debug option --debug-lexer or -Dl to see which tokens that are
	returned by the lexer.
  * New debug option --debug-profile or -Dp to print timings information and dump cpu/mem
	profiles during compilation.
* Changes
  * Removed `cmd/cli`. The CX executable should now be used to generate CX chain addresses.
  * Running a CX chain with the --broadcast flag no longer runs the transaction code locally; it simply
	broadcasts the transaction code and it is run in the peer node to update the program state. If the user
	wants to test transaction code, the --transaction flag must be used.
  * Updated the style of the CX roadmap.
  * Changed max transaction size to 128 Mb. CX chains are storing their program state in two different unspent outputs in their skycoin fork. This means that if a CX program to be stored on a CX chain needs 64 Mb, then the CX chain will need at least a max transaction size of 128 Mb. This behavior needs to be corrected immediately in the next version of CX and the user needs a way to set these parameters via flags (they're hardcoded at the moment).
* Libraries
  * Add cx arg parsing library
* Fixed issues
  * #357: Error running cx in blockchain broadcast mode.
  * #360: Panic when package keyword is misspelled
  * #373: Error in address used to generate a CSRF token. Port was 6001 instead of 6421.
  * #338: Issues with slice of type T where sizeof T is different than 4.
  * #388: Array fail on cx --blockchain.
  * #389: CX chains errors with big programs.
* Documentation
  * New file `documentation/BLOCKCHAIN-OVERVIEW.md` which describes the processes and modules involved in CX chains.
  * The blockchain tutorial
	[documentation/BLOCKCHAIN.md](https://github.com/skycoin/cx/blob/develop/documentation/BLOCKCHAIN.md)
	will be used to reflect the state in the CX source code (`develop` branch)
  * The blockchain tutorial in the
	[wiki](https://github.com/skycoin/cx/wiki/CX-Chains-Tutorial) will
	be used as a tutorial for the latest CX release. 
* Miscellaneous
  * ...

### v0.7beta released 2019-05-19
* Additions
  * First release of CX chains, i.e. CX programs stored on a Skycoin fiber blockchain
  * Added/forked the newcoin and skycoin-cli commands to the CX
    repositoryand adapted it to CX chains.
  * CX can now create a wallet by running `cx --create-wallet --wallet-seed $WALLET_SEED`
  * Added --wallet-id flag. This parameter replaces having to set the WALLET environment variable for CX chains.
* Changes
  * Transaction and block default sizes for CX chains changed from 32 Kb to 5 Mb.
* Libraries
  * Add math bindings for f32 and f64 types: isnan, rand, acos, asin.
  * Add glfw bindings: Fullscreen, GetWindowPos, GetWindowSize, SetFramebufferSizeCallback, SetWindowPosCallback, SetWindowSizeCallback.
* Fixed issues
  * #292: Compilation error when left hand side of an assignment expression is a struct field.
  * #309: Serialization is not taking into account non-default stack sizes.
  * #310: Splitting a serialized program into its blockchain and transaction parts.
  * #311: `CurrentFunction` and `CurrentStruct` are causing errors in programs with more than 1 package.
  * #312: Deserialization is not setting correctly the sizes for the CallStack, HeapStartsAt and StackSize fields of the CXProgram structure.
* Documentation
* IDE
  * Removed the current version of the IDE. We'll move to a textmate-based
    solution in the future.
* Miscellaneous
  * Vendored all external packages

### v0.6.2
* Additions
  * Support for exponents in floating point numbers (f32 and f64). Also
	improved decimal point parsing
	examples: 1.0 1. .1 1e+2 .1e-3
  * Support for multi-dimensional slices.
* Libraries
* Fixed issues
  * #58: No compilation error when indexing an array with a non integral var.
  * #70: Inline field and index "dereferences" to function calls' outputs.
  * #72: Multi-dimensional slices don't work.
  * #76: Using int literal 0 where 0.0 was needed gave no error.
  * #133: Panic when inserting a newline into a string literal.
  * #134: Panic when declaring a variable of an unknown type.
  * #155: Panic when trying to assign return value of a function returning void.
  * #156: Panic when using a function declared in another package without importing the package.
  * #166: Panic when calling a function from another package where the package name alias a local variable name
  * #271: CX floats cannot handle exponents
  * #284: Concatenation of str variables with + operator doesn't work.
  * #285: Short declaration doesn't compile with opcode return value.
  * #288: No compilation error when using float value in place of boolean expression.
  * #289: Panic when package contains duplicate function signature.
* Documentation
* IDE (WiP)
* Miscellaneous
  * CXFX, the game development toolkit, has been moved to its own
	repository at https://github.com/skycoin/FIXME

### v0.6.1
* Additions
  * Support for lexical scoping
  * `if/elseif` and `if/elseif/else` statements now work properly.
* Libraries
* Fixed issues
  * #54: No compilation error when defining a struct with duplicate fields.
  * #67: No compilation error when var is accessed outside of its declaring scope.
  * #69: glfw.GetCursorPos() throws error
  * #82: Empty code blocks (even if they contain commented-out lines) crash like this.
  * #99: Short variable declarations are not working with calls to methods or functions.
  * #102: String concatenation using the + operator doesn't work.
  * #135: No compilation error when using arithmetic operators on struct instances.
  * #153: Panic in when assigning an empty initializer list to a []i32 variable.
  * #169: No compilation error when assigning a i32 value to a []i32 variable.
  * #170: No compilation error when comparing value of different types.
  * #247: No compilation error when variables are inline initialized.
  * #244: Crash when using a constant expression in a slice literal expression.
	* The problem actually involved the incapability of using expressions as
	values in slice literals
* Documentation
* IDE (WiP)
* Miscellaneous

### v0.6
* Additions
  * Serialization and deserialization
  * Functions as first-class objects
  * Callbacks
  * Improve error reporting system
  * Add slice resize/copy/insert/remove functions
  * Add ReadF32Slice (gl.BufferData and gl.BufferSubData accept slice arguments)
  * Add slice helpers
  * Switch len and size in slice header to avoid unecessary alloc/copy.
  * Refactor cx/op_und.go::op_append
  * Refactor cx/utilities.go::WriteToSlice
  * Add runtime bound checks on slices
  * Print exit code string when a runtime error is thrown
  * CX process returns CX_RUNTIME_* exit code when a runtime error is thrown
  * Add strerror opcode returning a string matching the CX_* error code passed as argument
* Libraries
  * Added GIF support to OpenGL
* Fixed issues
  * #32: Panic if return value is used in an expression
  * #40: Slice keeps growing though it's cleared inside the loop
  * #41: Scope not working in loops
  * #50: No compilation error when using an invalid identifier
  * #51: Silent name clash between packages
  * #52: Some implicit casts were not being caught at compile time
  * #53: CX was not catching an error involving invalid indirections
  * #55: Single character declarations are now allowed
  * #59: Fields of a struct returned by a function call can now be accessed
  * #61: No compilation error when passing *i32 as an i32 arg and conversely
  * #62: No compilation error when dereferencing an i32 var
  * #63: Fixed a problem where inline initializations didn't work with dereferences
  * #65: Return statements now work in CX, with and without return arguments
  * #77: Fixed errors related to sending references of structs to functions and
        assigning references to struct literals
  * #101: Using different types in shorthands now throws an error
  * #104: Dubious error message when indexing an array with a substraction expression
  * #105: Dubious error message when inline initializing a slice
  * #108: Solved a bug that occured when two functions were named the same in
    different packages
  * #131: Problem with struct literals in short variable declarations
  * #132: Short declarations can now be assigned values coming from function calls
  * #154: Sending pointers to slices to functions is now possible
  * #167: Passing the address of a slice element is now possible
  * #199: Trying to call an undefined function no longer throws a segfault
  * #214: Fixed an error related to type deduction in references to struct fields
  * #218: Type checking now works with receiving variables of unexpected types
* Documentation
  * CONTRIBUTING.md: Information about how to contribute to CX
* IDE (WiP)
  * Added a simple guide
* CX GUI helper moved to its own repository at https://github.com/skycoin/cx-gui-helper

### v0.5.18 [2018-11-27 Tue 21:33]
* **Affordances**:
  * Support for `affordances-of`: argument -> argument
  * Support for `affordances-of`: argument -> program
  * Support for `affordances-of`: argument -> struct
  * Support for `affordances-on`: argument -> argument
* `break`: Implemented
* `continue`: Implemented
* `printf`: Added %v format directive which tries to deduce an argument's type and prints its value
* `printf()`: Tell the type and value of extra arguments
* **Labels**: can now be anywhere in a function's block of statements
* Fixed issues:
  * Fix #39: Short-variable declarations now work properly with arrays, e.g. `bar := foo[0]`
  * Fix #49: Trying to access fields of non-struct instances now throws an appropriate error
  * Fix #84: `++` suffix operator now working
  * Fix #92: Conflict when calling multiple callbacks using GLFW
  * Fix #98: CX now throws an error when trying to redeclare a variable
  * Fix #112: `printf` now prints either a `MISSING` or `EXTRA` when there are
              fewer or more arguments than format directives, respectively
  * Fix #120: CX was throwing redeclaration errors in multiple return expressions

### v0.5.17
* Fixed issues:
  * Fix #90: `goto` now works properly on Windows
  * Fix #91: Methods with pointer receivers are working now
  * Fix #111: Trying to use global variables from other packages is no longer
              allowed without their owner package prefixed to them, i.e. `foo` was
	      allowed, now it must be written as `pkg.foo`
