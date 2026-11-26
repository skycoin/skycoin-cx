# Changelog

### Unreleased

#### Language

- Language version 2611230001. File access, networking and `os.Run` are system libraries, `file`, `net` and `process`, called through the one opcode `LibraryCall` (795) with a library and a function number. Source is unchanged, but a call runs only when `cx --enable=file,net,process` enables its library, and otherwise is the runtime error `CX_RUNTIME_LIBRARY_DISABLED` (12) naming the library, on the old loop and on vm. The 68 opcodes of `sys/os` files, `sys/json`, the CPU profile and `sys/tcp` are deprecated, `os.Exit` stays opcode 190. Serialized programs must be recompiled (#492)
- Language version 2611230002. A function declares every global it uses in a `global` statement at the top of its body, `global counter, config.Limit`, globals of its own package included, and reaches a global of another package only as `package.name`. An undeclared use, a declared global the function does not use and a `global` statement after another statement are compile errors (#486)
- Shadowing is a compile error. A local, parameter, result, loop variable or `:=` name that has the name of a global, an imported package, a function or a type of its package, or a variable of an enclosing block reports `'x' shadows global declared at file:line:column`. `tests/issue-51.cx` passes and every program in the repository was migrated (#486, #25)
- Language version 2611230003. An import brings names into scope, `import . "geometry"` (the form of Go) and `import (Point, Distance) "geometry"` for the package, and `import file . "geometry"` and `import file (Point, Distance) "geometry"` for its file. A shorthand compiles to `geometry.Point`. A name brought into scope that has the name of a local, parameter, result, global, function, type, imported package or a name of another import is the compile error `'Distance' masks the name imported at main.cx:3:16`, and a warning in a function under a `//cx:masking warning` directive (#487)
- Language version 2611240001. `sys/glfw`, `sys/gltext` and `sys/gl` name windows, fonts, gifs and texture pixel copies by `i32` handles into fixed tables instead of `str` names looked up in Go maps, so `window = glfw.CreateWindow(800, 600, "Title")` and `glfw.SwapBuffers(window)`. A callback is a function name the compiler resolves, `glfw.SetKeyboardCallback(window, onKey)`, and gets the window handle as its first input. A name that is not a function is compile error 7080, and a handle never returned, freed or out of range is an assert. `gl.ShaderSource` takes the source and hands GL a view of program memory, and `gl.Strs`, `gl.Free` and `gl.Destroy` (592 to 594) are deprecated. Shader programs that used `gl.Strs(source, "csources")` now compile their shaders, which the swapped key never did. `gltext.GlyphBounds` takes the font it read without one. The mobile files of t6-cxfx are removed. `make test-gui` and `make check-strhot-gui` run in the vet-gui job (#432)
- Language version 2611240002. JSON is not part of the core (D46). `sys/json` and its functions `json.Open` to `json.Str`, functions 52 to 61 of the `file` library, are removed and their numbers deprecated, and `tests/test-json.cx` with them. The package loader stores a file, package and package list in the CX text form of the new `t0-textsections`, LZ4 compressed, instead of JSON, and the execution records read and write through it too. `make check-json`, part of `make check`, fails when a package other than vendor and the `t0-cx-tools-*` developer tools imports `encoding/json` (#509)
- Language version 2611240003. `sys/cipher` is the system library `cipher` (4), enabled with `--enable=cipher`. `cipher.HashOfBytes` and the incremental `cipher.HashStateNew`, `HashStateWrite`, `HashStateSum` and `HashStateRelease` hash with SHA-256 until decision D23 names the BLAKE hash, through `t0-cxhash`, which the execution records use too. `cipher.SignatureVerify`, `SignatureRecoverPubKey` and `SignatureSign` work on secp256k1 signatures of Skycoin's `cipher` package (D22), and `cipher.KeyPairGenerate` replaces `cipher.GenerateKeyPair` (opcode 134, deprecated) and returns `[]ui8` instead of the removed `cipher.PubKey` and `cipher.SecKey` structs. Byte slices are read as views of program memory, and the hash functions make no Go allocation. `HashStateNew` and `HashStateRelease` run on vm. `make build` builds sixteen cores, eight with cipher, see docs/CipherLibrary.md (#510)
- Language version 2611240004. `sys/tcp` is replaced by `sys/net`, read through batched updates (#508). The runtime reads every socket in the background into fixed buffers, and `net.Update` makes what arrived one batch, sorted by socket handle, which the program reads as ready entries with `net.ReadyNext` and payloads with `net.Read` into a `ui8` array it owns. Writes are sent at the next update. TCP and UDP, many sockets, timeouts on a timer wheel and dropped datagram counts are new, and every function runs on vm. The update, the reads and the writes make no Go allocation, and the work of an update grows with the ready count and not with the number of sockets. `tcp.Dial`, `tcp.Listen`, `tcp.Accept` and `tcp.Close`, functions 1 to 4 of the net library, are deprecated. See docs/Networking.md
- Language version 2611250001. The `file` library has `os.OpenReadWrite` (64), which opens a file for reads and writes at any offset without truncating it and creates it when it is missing, `os.Sync` (65), which commits the writes of a handle and runs on vm without a Go allocation, and `os.Remove` (66), which removes a file or an empty directory through the sandbox root, so a path or a symlink that leaves the sandbox is refused and a symlink is removed itself. Every networking function of D19 already exists in `sys/net`, see docs/Networking.md
- Language version 2611250002. Affordances select what `serialize` writes (decision D5, #500). An aff literal `#{ ... }` takes the clauses `mem`, `pkg`, `fn`, `var` and `strct`, separated by spaces, semicolons or newlines, and `var` is a clause name. The compiler resolves every name, so a wrong clause is compile error 7082, and an `aff` is a 16 byte value that runs on vm. `serialize` gives the whole program, the bytes in use of a memory segment, a variable, or a package, function or struct after a `CXSELECT` header. `deserialize` still loads only a whole program. `aff.print` and `aff.query` (580 and 581) are deprecated, and the affordance query code and its examples are removed
- Language version 2611250005. The 56 functions of the `file` library are opcodes 820 to 875, `constants.OpcodeFileGetWorkingDirectory` to `constants.OpcodeFileRemove` in the new group `OpcodeGroupFile`, left out by `nocxos`, instead of functions of library 1 through `LibraryCall` (decision D119). The generated `cxos.FileOpcodeDispatch` runs them on the old loop. `os.Close`, `os.Seek` and `os.Sync` are the image opcodes `ir.OpcodeFileClose`, `ir.OpcodeFileSeek` and `ir.OpcodeFileSync`, which the vm runs from `vm_file.go` on the open files of the new `t0-cxfiles`, and their native forms are C functions of the LLVM runtime. A program that uses a file operation it does not enable is refused before it runs
- Language version 2611250004. The fifteen functions of the `net` library are opcodes 805 to 819, `constants.OpcodeNetUpdate` to `constants.OpcodeNetUDPPeerResolve` in the new group `OpcodeGroupNet`, instead of functions 5 to 19 of library 2 through `LibraryCall` (decision D119). The generated `cxnet.NetOpcodeDispatch` runs them on the old loop, and the vm runs the image opcodes `ir.OpcodeNetUpdate` to `ir.OpcodeNetUDPPeerResolve` from `t3-cx-runtime-vm/vm_net.go`. Their native forms are C functions of the LLVM runtime that poll the sockets, with Winsock on Windows, and replay `CX_NET_REPLAY`, so the net programs build natively and the native form of the execution records replays them. A program that uses a net operation it does not enable is refused before it runs. Cores are built with the new `cxcore` tag, which with `cxcorelibrarynet` decides whether the vm and the run start link the net library
- `cx --enable` takes single library operations as well as libraries, `--enable=cipher,file.Open` style, for sandboxing at library level and at opcode level (Synth, D119). An operation is the library name and the function name after its package, `cipher.HashStateNew`, and an unknown entry is an error. The enabled operations are `constants.OpcodeSet`, a fixed size bit set over the opcode table, in `CXProgram.ProgramOperationsEnabled`, `ir.Image.ImageOperationsEnabled` and the serialized program. A program that uses a library opcode outside the set is refused before it runs with `CX_RUNTIME_LIBRARY_DISABLED`, naming the library when it enables nothing of it and the operation otherwise, in cx, in a built program and in a native executable. `core.CoreLibraryRefused` and `core.CoreLibraryRefusalLine` are `core.CoreOperationRefused` and `core.CoreOperationRefusalLine`
- Language version 2611250003. The nine functions of the `cipher` library are opcodes 796 to 804, `constants.OpcodeCipherHashOfBytes` to `constants.OpcodeCipherKeyPairGenerate` in the new group `OpcodeGroupCipher`, instead of functions 1 to 9 of library 4 called through `LibraryCall` (decision D119). The source does not change. A program that uses one without `--enable=cipher` is refused before it runs with `CX_RUNTIME_LIBRARY_DISABLED` (12), and the run loop no longer checks the library. `cipher.HashStateNew` and `cipher.HashStateRelease` are the image opcodes `ir.OpcodeCipherHashStateNew` and `ir.OpcodeCipherHashStateRelease`, and their native forms are C functions of the LLVM runtime, so a program that uses only them builds with `cx build --target=llvm`
- `cx --net-record=FILE` writes every batch of the net library to a replay record, and `cx --net-replay=FILE` (`CX_NET_REPLAY` for a built program) runs the program from it without opening a socket and prints what the recorded run printed (D53). The execution records replay the record beside a program and hash it into the input, so `test-net-tcp.cx` and `test-net-udp.cx` are records checked in every form, and the `network` exclusion is only for programs without one (#508)
- The hash of CX is BLAKE3 with a 256 bit output, BLAKE3-256, instead of SHA-256 (decision D23). `cipher.HashOfBytes` and the hash states return BLAKE3-256 digests, through `t0-cxhash` and `lukechampine.com/blake3`. The execution records file is format 2, named `BLAKE3-256`, and a format 1 file of SHA-256 hashes is refused. Every record was regenerated, every hash changed and no output changed (#507, #510)
- A program that embeds CX can add opcodes outside the opcode table again, which #534 had removed with `ast.OpcodeHandlers`. It names them in `ast.NativeNames`, gives their natives through the new `ast.NativeFunctionHost` and runs them from a wrapper of `ast.OpcodeDispatch`. The Android playground needs this for its `app` functions
- `import "pkg"` alone no longer lets a file call a function or use a type of `pkg` by its name alone. It found one in any import, the first of several in map order. `tests/issue-270.cx` and `tests/examples/testing-packages.cx` now write `pkg.T`, and the error drops "or its imports"
- A `var` in a one line function body, `func f() { var s i32 }`, no longer declares a global `s` of the package as well
- The errors of a parameter and of a name a `:=` declares point at the name. They pointed at the token after the parameter list or at the value
- `printf` and `sprintf` print a `%s` argument as it is. They turned a `\n` or `\%` the str held as two bytes into a new line or `%`, and read the same escapes again in the format. `%%` prints `%`
- String literals take every escape of Go. `\x41`, `\u00e9` and `\U0001F600` were `invalid character 'x' in hexadecimal escape`. An invalid escape is reported once, with the message of Go, and a raw string drops its carriage returns
- `string.BytesOfString` and `string.StringOfBytes`, opcodes 793 and 794, convert between `str` and `[]ui8`
- `len`, `[i]` and `append` read the `[]T` result of a nested opcode or library call, `len(string.BytesOfString("abc"))` or `len(serialize(t))`, and `len(append(s, v))` reads the result of a nested `append`, `resize` or `insert`. They panicked with `index out of range [0] with length 0`, failed with `CX_RUNTIME_INVALID_ARGUMENT`, or stopped the compiler with `type is not known` for `append(f(), v)` and `var c ui8 = f()[1]`. The call stack of a runtime error prints such a result as a slice, `[]`, where it printed `0`
- `remove` into a different slice, `q = remove(r, 1)` or `len(remove(r, 1))`, removes the element from the input slice in place and returns it, as `append`, `insert` and `resize` do while the capacity holds the result. It shifted and shortened the output slice with the length of the input, which panicked with `slice bounds out of range [1:0]`. LanguageGuide.md says when a slice built-in shares the input
- `remove` and `resize` on a slice of slices or of pointers move 8 bytes an element in a ptr64 build. They moved 4, so `m = remove(m, 0)` on a `[][]i32` read a broken slice and `resize` dropped elements
- `(*pp).f` on a `**T` dereferences `pp` twice. It read the field at the address of the `*T`
- Composite literals take pointer and mixed element types, `[]*i32{&x}`, `[2]*T{&a}`, `[]*[2]i32{&a}`, `[]*T{&T{}}`, `[2][]i32{{1}, {2, 3}}` and `[][2][]i32{...}`, and `[]*T{{x: 1}}` leaves out `&T`. They were syntax errors
- An index of three levels mixing arrays and slices, `a[0][1][0]` on a `[][2][]i32`, `[2][][2]i32` and the others, reads the right element, and `(*s[i])[j]` on a `[]*[]T` reads through the pointer. They read the wrong address
- A global `str` without a value is empty. It held its own address, so reading it, also through a `*str`, panicked with `slice bounds out of range`
- The call stack of a runtime error prints a str, slice, array or struct pointer that leads outside program memory as `<invalid address>`. A local whose declaration had not run still held an earlier value of its stack slot, and printing it as a str panicked with `slice bounds out of range` in `appendAtomic` (#407)
- `:=` with several names assigns the names its block has and declares the others, as in Go, and one that declares nothing is `no new variables on left side of :=`. It reported `'old' redeclared`
- `a, b := two(), 1` is `multiple-value two() (value of type (a i32, b i32)) in single-value context` and `var y i32 = none()` is `function 'none' returns no value`, where they reported the receiving count of the call
- `item.flag && f()` and `||` with a field, element or pointer field on the left read the field. The jump of the short circuit shared the operand with the bool op, whose field was resolved twice, so `&&` gave false and `items[i].flag && f()` read out of range

- Pointers, slices, arrays and structs combine in any order as parameters, results, locals, globals, fields and elements. `x := f()` of a `*str`, `[]*T`, `[2]*i32` or `[N][]T` result reported a wrong receiving variable, a `*[]i32` result panicked with `type is not known`, and globals of such types could not be inferred. `[2]*[3]i32` parses, and an array whose elements point to arrays is sized and indexed as an array of pointers. `[]*[]i32` and `[][2][]i32` keep their whole type, where they took the type of a pointer to a slice and a slice of slices
- `var c = p` and `c := p` in a function take the whole type of `p`, such as `*Point` or `**i32`. They reported `cannot assign *Point to c of type struct`
- A call of a function without results used as a slice, array or struct literal element, an argument or an operand is the compile error `function 'f' returns no value`. A slice literal element panicked with `index out of range` and an array element was dropped without an error
- A local of a struct that does not exist, such as `var a [][]Missing`, is declared invalid, so only the missing struct is an error and its uses report nothing more
- `x, y := 1, 2, 3` and `x, y, z := two()` declare their new names invalid after the count error, so their uses report nothing more, and a name the function declared already is not reported `redeclared`
- A trailing comma ends a composite literal, a call argument list, a parameter list or a result list, as Go requires before a newline
- `%v` prints a pointer as Go does, `0x100000`, a nil pointer as `<nil>` and a pointer to a struct as `&{x: 1}`, where it printed the address as a decimal number. `%+v` prints as `%v`, where it printed `v`
- A global read only in a method orders the initializer that calls the method, `var first = obj.get()`, as in Go
- `var a, b = f()` and `var a, b T = f()` declare globals from the results of `f`. They were `assignment mismatch: 2 variables but 1 value`

- Function outputs in parentheses may give only their types, `func f() (i32)` and `func f() (i32, str)`, filled by `return 1, "a"` and read by `x, s := f()`, as in Go
- An output may be an array, an array of structs, a slice of arrays or a pointer to an array, named or not. `return [2]i32{1, 2}`, `return &a` of a local array and `r[1] = 2` on the line of the function header work, where they reported a wrong type, read a dead frame or panicked
- `var a, b i32`, `var a, b i32 = 1, 2`, `var a, b = 1, 2` and `var ( ... )` groups declare globals and locals, as in Go. A local without a type is a short declaration, and a wrong number of values is `assignment mismatch`
- A global without a type takes the type of any value, `var g = f()`, `var g = a + b`, `var g = T{...}` or `var g = []i32{...}`, as in Go. Such globals are typed after the function signatures and the other globals, each after the globals its value names, and a value with no type, such as a call of a function without results, is the compile error `cannot infer the type of global 'g' from its value`
- Globals are initialized in declaration order, except that a global comes after the globals its value refers to, directly or through the functions it calls, as in Go. `var c = a + b` before `var a = 1` reads 1, where it read 0. A cycle of such references is the compile error `initialization cycle for 'a': a refers to b, b refers to a`
- `x := a * b + c`, a short declaration whose value is an operator of another operator, declares `x`. It reported `identifier 'x' does not exist` on every use
- A method of a global, `box.Area()`, can be called in a function and in the value of a global. It reported `identifier 'box' does not exist`, and in a global value `identifier '*mrs_1' does not exist`
- `var a, b = 1` declares both names, local or global, as invalid, so only the count is an error and their uses report nothing more
- A global whose struct or slice literal starts a brace it closes on a later line, or that is declared in a `var ( ... )` group or as the second name of `var a, b = ...`, keeps its value. A struct global panicked with `type is not type cx argument deprecate` and a slice global was empty
- `h.c.radius` of a field `c` whose type comes from a failed import reports nothing more, and of a field of a basic type it says `illegal method call or field access on identifier 'c' of primitive type 'i32'`. It said `field 'radius' not found in struct 'Holder'`. A field a struct literal names that does not exist reports nothing more
- The errors of a struct, array or slice literal point at the literal, a struct that does not exist at the type name, a field or a repeated field at its name and too many elements at the start of the literal, and a return with the wrong count at `return`. They pointed at the closing brace or the token after it. The errors a one line function finds once it is parsed give no column, where they gave the column of the token after the function
- A struct literal in an if or for condition, `if T{a: 1}.a == 1 {`, says `composite literal in condition must be parenthesized` instead of a plain syntax error
- A literal of a struct that does not exist is the compile error `struct 'T' not found` instead of a panic
- `cx -ast` prints packages in the same order on every run, each after the packages it imports. `TestAstPrintsEveryProgram` runs `-ast` twice on every program of `tests/`
- A struct can be declared on one line, `type T struct { a i32; b i32 }`, and so can a function body or block, `func f() i32 { return 1 }`, since the semicolon before a closing `}` may be left out, as in Go. A function with one output may give only its type, `func f() i32`. A multi-line comment that holds a newline ends a statement like a newline
- A struct literal is an operand, so `T{a: 1}.m() + 1` and `f(T{a: 1}.a * 2)` work. An if or for header still needs it in parentheses, as in Go (skycoin-cx#21)
- A pointer method called on a struct literal or a call result, such as `T{a: 5}.p()`, is the compile error `cannot call pointer method p on T`, as in Go. `(&T{a: 5}).p()` works
- An array literal element of the wrong type says `in array literal` and points at the element
- Binary operators bind as in Go. `&`, `&^`, `<<` and `>>` bind like `*`, and `|` and `^` like `+`, all tighter than comparisons. `d & 1 == 1` was `d & (1 == 1)` and is `(d & 1) == 1`, and `1 + 2 << 3` was 24 and is 17. No program in the repository changes meaning
- Arrays of slices such as `[2][]i32` can be declared as variables, globals and struct fields, and the GC follows their slices
- Slices of arrays such as `[][2]i32` and `[][2]Point` take `append`, indexing, `len` and element field access, and the GC follows them
- Program file paths are confined to a sandbox directory, the working directory or `cx --sandbox <dir>`, opened once at run start with `os.OpenRoot`. Every path a program passes to `os.Open`, `os.Create`, `os.ReadAllText`, `json.Open`, `StartCPUProfile`, the `gl` texture and GIF loaders and `al.LoadWav` must be relative, `/` separated, without NUL, at most 4096 bytes, and may not leave the sandbox by `..` or a symlink. An absolute path such as one built from `os.GetWorkingDirectory` is now rejected. A rejected path fails like a missing file. `t0-util` has `FileError` with the first file failure and operation codes (#433)
- Language version 2611200002. Every built-in library lives under `sys/` and needs an explicit import, for example `import "sys/os"`. A bare import of a built-in package, an unknown `sys/` path and a built-in package used without its import are compile errors (#380)
- The `str` functions moved to `sys/string` with verbose camel case names, for example `str.concat` is `string.Concatenation` and `str.compare_by_value` is `string.OrderingByByteValue`, and `glfw.func_i32_i32` and `glfw.call_i32_i32` are `glfw.RegisterCallbackI32I32` and `glfw.InvokeCallbackI32I32`. Opcodes keep their numbers, `OpcodeRenameTable` and the `former_names` column of `docs/opcodes.csv` record the renames, and an old name is a compile error that names the new one (#380, #382, #391)
- `string.ByteLength`, opcode 792 (#382)
- Language version 2611200001, formerly revision 1, see the Language versions section of LanguageGuide.md (#183, #296)
- Opcode numbers are fixed in `constants.OpcodeLifecycleTable` and `docs/opcodes.csv`. Every opcode keeps the number it had before #197, deleted opcodes leave gaps that are not reused before their reassignable version, `ptr.eq` and `ptr.uneq` are 589 and 590, and the cxfx opcodes are 591 to 791 whatever the build tags (#296)
- Strings have no built-in operators. `==`, `!=`, `<`, `>`, `<=`, `>=`, `+` and every other arithmetic or bitwise operator on `str` is a compile error that names the function to use (#183, #197)
- `<`, `>`, `<=` and `>=` on pointers are compile errors. Pointer `==` and `!=` compare addresses, and now work for every pointer type instead of only `*i32` (#183, #197)

#### Added

- `cx build --target=llvm` links with TinyGo when there is no clang but a `tinygo`, or with `--llvm-linker=tinygo`, and `--llvm-tinygo-target` cross compiles for a TinyGo target such as `pico`, with its C library, startup code and linker script. The C runtime prints numbers without printf, which the C library of TinyGo cannot format (decision D131)
- `cx build --target=image` writes the image vm runs to a file, and `t4-cx-core-image` runs it on vm with no parser, compiler or AST, so it builds with TinyGo for a microcontroller. `make build-mcu` and `make run-mcu` embed an image in it for a TinyGo target and run it in QEMU or wasmtime, and the new `image-core` form of the execution records checks it (decision D129). See docs/Microcontrollers.md
- `make build-tinygo` and `make build-wasm-tinygo` build cx with TinyGo, an alternative to the Go builds, which stay the default. A TinyGo build checks sandbox paths in Go, as TinyGo has no `os.Root`, and hashes with a portable BLAKE3 that a Go test checks against `lukechampine.com/blake3` (decisions D130 and D132). `CX_EXECUTION_RECORDS_CX` runs the execution records with another cx, and a TinyGo cx passes all of them. See docs/Microcontrollers.md
- A `cx build` core with no program appended runs `cx-core --program FILE args`, where FILE is a program made with `cx build`, so a js/wasm host, which has no `os.Executable`, needs no separate command. A core with an appended program runs it with every argument as before
- `make check-names`, part of `make check`, fails on a Go name that breaks the Naming rules of AGENTS.md and docs/NamingPlan.md and is not in `names-allowlist.txt`, which only shrinks. docs/NamingPlan.md lists the renames to come (#569)
- Release archives. Pushing a `v*` tag publishes a GitHub release of cx and the 16 `cx build` cores, built with cgo off for linux, darwin and windows on amd64 and arm64, with SHA256SUMS. A tag with a `-`, such as `v0.9.0-alpha1`, is a prerelease. The workflow refuses a tag whose commit is not on main or whose check run did not pass. `make release-archives VERSION=v0.9.0` builds the same archives locally, and `make release-notes VERSION=v0.9.0` prints the notes. The GUI build (cxfx) is not included
- `cx --version` prints the release tag, which make release-archives sets with `-ldflags -X main.VERSION`
- The LLVM target, decisions D8 to D10. `cx build --target=llvm program.cx -o program` compiles a program that runs on vm to LLVM IR, `program.ll`, and with clang to a native executable that prints what vm prints, and refuses any other program with the vm message. Each scalar opcode is an inline template or a call into the C runtime that links with the program, 279 inline and 17 calls. `--llvm-debug` adds DWARF debug information with the CX names and types of variables. The execution records check a `native` form, see docs/LlvmTarget.md
- Execution records. `tests/execution-records.txt` holds 439 records, the SHA-256 of each test program as `cx build` serializes it, of its input and of its stdout, with the exit code, and the 327 programs left out with their reason. `TestExecutionRecords`, part of `make check`, runs every record on the old loop, on vm, after a round trip through the serialized program on both loops and as a `cx build` executable, and names the record, the form and the first differing line of any other result. `make records` writes the file from the old loop. It replaces `TestSerializeRoundTrip` and `TestBuildPrograms` and the stdout comparison of `TestLowerCompare`. See `docs/ExecutionRecords.md` (#507)
- `docs/StringsHotPath.md` and `strhot-codes.txt` give every string operation the old loop can reach a STRHOT code with a class, I every instruction, O every run of an opcode, P printing, E a failure branch and B a boundary, and a `VIOLATION STRHOT-nnn:` comment. `make check-strhot`, part of `make check`, fails when a new one has no code (#412)
- `docs/GarbageCollector.md` lists every place the interpreter and the IR base reach the garbage collector, with a `GCREACH:` comment at each, and proposes running the collector as a tick of the main loop. `make check-gcreach`, part of `make check`, fails when the vm or the IR base imports `t3-ast` or a new reach has no comment (#485)
- `cx build program.cx -o out` writes one executable file, the core binary `cx-core` with the serialized program and a 32 byte trailer appended, which runs the program and passes its arguments as `os.Args`. The core (`t7-cx-core-run`, `t6-cx-runtime-core`) has no parser, no file system opcodes and no module outside the Go standard library, and `cx build` refuses a program that uses an opcode it leaves out. `make build` builds `bin/cx-core`. `time` and `os.Exit` moved to `t5-cxos-core`, and `t3-ast` no longer imports `t0-lz4codec`, taking any `ast.SerializeCodec`. Whether the core may use outside modules and the file system is still pending
- Long benchmarks in `t0-benchmarks/compare`: a sieve, Miller-Rabin and Lucas-Lehmer prime test, a memtest86 style RAM test, and about one second versions of the count, fib, array sort and struct walk programs, each in CX, C, Python 3 and Bash and each printing an FNV-1a hash that `TestCompareExpected` derives from the math. `make bench-compare` times them. A cx built with `-tags cxopcount` prints the operations the old loop and vm ran (#470)
- Compile errors are data. The compiler records a `CompileErrorRecord` with a `CompileErrorCode`, a `SourceFileGid`, a line and `StringGid` or integer operands in `CxIrState.CompileErrorList`, and `t5-cx-runtime-printer` prints them through a table of formats when compilation returns or the compiler exits. `ast.CompilationError` is gone. Output and exit codes are unchanged (#413)
- `t5-cx-runtime-printer` prints runtime errors after the run has returned. The old loop records the code, the panic value and the call stack position in `CXProgram.RuntimeError` instead of printing inside its recover, and `PrintStack`, `ErrorHeader` and `RuntimeErrorInfo` moved there as `PrinterStackPrint`, `PrinterErrorHeader` and `PrinterRuntimeErrorPrint`. The `debug` opcode is registered by the printer. Output and exit codes are unchanged (#413)
- `t3-cx-runtime-vm` runs a lowered `ir.Image` in one switch loop with no allocation. cx runs a program on it when every function lowers and on the old loop otherwise, and `cx --executor=auto|old|vm` chooses. 71 of the 451 programs `TestLowerCompare` checks run on it (#408)
- The vm subset takes array indexes, struct fields and pointer dereferences into the stack and data segment, taking addresses, struct and array values, `string.PrintLine` and `printf` with a literal format. 149 of the 455 programs `TestLowerCompare` checks run on vm (#408)
- `BenchmarkRun` in `t11-cx` times `execute.RunCompiled` alone over count, fib and bubble sort programs and the `BenchmarkCompile` programs, compiling outside the timer. `TestRunAllocs` fails when the allocations per run of any of them go above the counts recorded when it was added. The target is 0 (#407, #417)
- The object model managers grow only at grow boundaries between compile stages and REPL iterations, to the next power of two when less than half of a manager is free, and print a warning on stderr when they do. Alloc on a full manager between two boundaries is an assert, see the Growth section of docs/ObjectModel.md (#299)
- The object model managers, stage 3 of #299. Package `objmodel` (`t1-cx-core-objmodel`) holds the Gid layout and checks, and `t3-ast` has a generated `<obj>_gid.go`, `<obj>_mng.go` and manager test for each of the 14 IR object types, plus `CxIrState` with every manager. Nothing in the compiler uses them yet (#299)
- `make check-objmodel` fails when `docs/objectmodel-sites.csv` and the site comments of the code disagree. 100 rows of sites rewritten away and the comments of 71 sites #429 fixed are gone, and 22 rows name the file their site moved to
- `t0-cx-core-objmodelgen` generates the Gid and manager files from one template, `make generate-objmodel` runs it, and `make check-objmodel`, part of `make check`, fails on stale generated files, a Def file with more than its struct, or a Fon or Fof file that names a manager or `CxIrState` (#299)
- `docs/ObjectModel.md`, the object model standard for the compiler IR. Def, Mng and Gid files per object type, the generational index layout, one `CxIrState` owning every manager, no pointers at rest, and every manager empty at teardown. `docs/objectmodel-sites.csv` lists every site still to migrate, and `BenchmarkCompile` in `t11-cx` is the compile speed baseline (#299)
- `docs/IR.md`, what the compiler emits. It describes the in-memory program the VM runs, its arenas, expressions, operands and memory layout, a compiled example, and the serialized form
- `str.equal_by_value`, `str.equal_by_pointer` and `str.compare_by_value` (#184)
- Serialized programs carry the language version `constants.LanguageVersionCurrent`, the integer yymmddnnnn, in `CXProgram.Version`, and `Deserialize` rejects a program from another version (#183, #197, #296)
- `docs/Conventions.md`, every repository rule in one list, linked from the README (#296)
- `docs/ErrorHandling.md`, the two-class error policy: path, file and network failures return the error struct of their own module, with failure and operation codes from that module's code range, every other failure is an assert that crashes, and nothing recovers. `t0-cx-core-assert` holds the asserts, and `make check-recovers` fails on a new `recover(` call (#270)

#### Removed

- `ir.OpcodeLibraryCall`, `ir.Image.ImageLibraryEnabled`, `vm.LibraryDispatchesVm`, `vm.LibraryDispatchVmFunction`, `vm.LibraryFunctionVmLinked`, `vm.LibraryStrView`, `vm.StopLibraryDisabled`, `llvm.LibraryCallRefusal`, the `HandlerVm` column of `constants.LibraryFunctionDefinitionTable`, `constants.LibraryFunctionSignature.LibraryFunctionSignatureVm`, `cxos.LibraryProcessDispatchVm` and the `dispatchVm` input of `opcodes.LibraryLink`. Once the file, net and cipher functions became opcodes nothing used them, and the vm runs no library call and checks no library (D119). `os.Run` of the process library is still a `LibraryCall` on the old loop
- `cxos.LibraryFileDispatch`, `cxos.LibraryFileDispatchVm`, `cxos.LibraryFileCloseVm`, `cxos.LibraryFileSeekVm`, `cxos.LibraryFileSyncVm` and `cxos.LibraryFileRegister`. The open files are `cxfiles.FileHandleOf` and the rest of t0-cxfiles, and `cxos.ValidFile` calls it (D119)
- `cxnet.LibraryNetDispatch`, `cxnet.LibraryNetDispatchVm`, the fifteen `cxnet.LibraryNet*Vm` vm forms, `cxnet.LibraryNetMode`, `cxnet.LibraryNetRecordPath`, `cxnet.LibraryNetRunStart` and `ast.LibraryRunStarts`. The run mode is `netbatch.NetRunMode` and `netbatch.NetRunRecordPath`, and the run start a static call of t5-cx-runtime-core (D119). `t5-cxnet` is `t4-cxnet`
- `cipher.LibraryCipherDispatch`, `cipher.LibraryCipherDispatchVm`, `cipher.LibraryCipherHashStateNewVm`, `cipher.LibraryCipherHashStateReleaseVm`, `cipher.CipherHashStateCountMax` and `cipher.CipherState`. The cipher opcodes run through the generated `cipher.CipherOpcodeDispatch`, and the taken handles of the hash states are in the new `t1-cipherstate`, which uses only the standard library and which the vm calls directly, and the hashes are `cipher.CipherHashTable` (D119)
- `t3-packageloader/makefile`, whose targets pointed at directories that no longer exist and used `sudo rm`. `go test ./t3-packageloader` now runs load, tree, graph and save end to end (#113)
- `loader.FileHashMap` and `loader.PackageHashMap`, which were written but never read. FileHashMap was keyed by base file name, so files of the same name in different packages overwrote each other (#94)
- `ast.Debug`, `ast.CopyProgramState`, `ast.GetSerializedStackSize`, `ast.GetSerializedDataSize` and `CXProgram.PrintAllObjects`, which nothing called (#162)
- The `t4-assert` package, which nothing imported (#107)
- The `str.eq`, `str.uneq`, `str.lt`, `str.lteq`, `str.gt` and `str.gteq` opcodes, deprecated in version 2611200001. Their numbers stay empty, and serialized programs from before that version must be recompiled (#183, #197, #296)

#### Changed

- Package `t0-util` is split by subject (#569). `t1-cx-runtime-sandbox` (package `sandbox`) holds the program paths, the sandbox root calls and `FileError`: `ProgramFileSystemPath` is `sandbox.Path` and `ProgramSandboxRootOpen`, `ProgramSandboxFileOpen` and the other `ProgramSandbox*` calls are `sandbox.RootOpen`, `sandbox.FileOpen` and so on. `ParseArgsForCX` is `sourceargs.Parse` in `t0-cx-compiler-sourceargs`. `CXOpenFile`, `CXCreateFile`, `CXStatFile` and `CXLogFile` are `filelog.Open`, `Create`, `Stat` and `Enable` in `t0-cx-runtime-filelog`. `ProgramTextStringView` is `stringpool.StringView`. `CXSetWorkingDir`, `CXRemoveFile`, `CXReadFile` and `CXMkdirAll` had no callers and are removed
- Go names of t0 to t2 renamed for the Naming rules (#569): `cxtestrunner.HasAll` is `BitsAllSet`, `packages.IsDefaultPackage` and `IsBuiltinImport` are `PackageDefault` and `ImportBuiltin`, `types.Code.IsPrimitive` and `types.Pointer.IsValid` are `Primitive` and `Valid`, `types.Get_obj_data`, `Read_obj_data`, `Write_obj_data` and `Write_str_data` are `ObjectPayloadGet`, `ObjectPayloadRead`, `ObjectPayloadWrite` and `StrPayloadWrite`, and the `Handler` field of `constants.OpcodeDefinition` and `constants.LibraryFunctionDefinition` is `GoFunction`
- Go names of t3-ast renamed for the Naming rules (#569). The `CXArgument` methods `IsPointer`, `IsString`, `IsStruct` and `IsSlice` are `PointerTyped`, `StringTyped`, `StructTyped` and `SliceTyped`, and `IsTypeAtomic` and the other `IsType*` functions are `CXArgumentTypeAtomic` and so on. The `CXExpression` methods `IsMethodCall`, `IsStructLiteral`, `IsArrayLiteral`, `IsBreak`, `IsContinue` and `IsUndType` are `TypeMethodCall`, `TypeStructLiteral`, `TypeArrayLiteral`, `BreakOperator`, `ContinueOperator` and `UndTyped`. `CXFunction.IsBuiltIn`, `IsAtomic` and `IsLocalVariable` are `BuiltIn`, `Atomic` and `LocalVariableDeclared`, `CXTypeSignature.IsInnerReference` and `IsDeref` are `InnerReference` and `Deref`, `IsValidSliceIndex` is `SliceIndexValid`, `GetSliceData` is `SliceElementsGet`, `IsOperator`, `IsArithmeticOperator` and `IsComparisonOperator` are `OpcodeOperator`, `OpcodeOperatorArithmetic` and `OpcodeOperatorComparison`, `OpcodeHandler` is `OpcodeGoFunction`, `SerializeDebugInfo` and `SerializedDataSize` are `SerializedSizesOf` and `SerializedSizes`, and `AffordanceValue.AffordanceValueItem` is `AffordanceValueTarget`
- Go names of t4-actions renamed for the Naming rules (#569). The `Process*` passes name what they do: `ProcessGoTos` is `GoTosResolve`, `ProcessTypedOperator` is `TypedOperatorResolve`, `ProcessMethodCall` is `MethodCallResolve`, `ProcessFunctionParameters` is `FunctionParametersDeclare`, `ProcessExpressionArguments` is `ExpressionArgumentsResolve`, `ProcessOperatorExpression` is `OperatorExpressionOutputsSize`, `ProcessPointerStructs` is `PointerStructsResolve`, `ProcessStringAssignment` is `StringAssignmentPassBySet`, `ProcessShortDeclaration` is `ShortDeclarationComplete`, `ProcessSlice` is `SliceDereferenceSet`, `ProcessSliceAssignment` is `SliceAssignmentComplete`, `ProcessTempVariable` is `TemporaryVariableComplete`, `ProcessSymbolFields` is `SymbolFieldsCopy`, `ProcessLoopJumpsOutsideLoop` is `LoopJumpsOutsideLoopReport` and `ProcessShortCircuitJumps` is `ShortCircuitJumpsResolve`. `SymbolsData` and `SymbolsDataKey` are `SymbolTable` and `SymbolTableKey`, `IsTempVar` is `TemporaryVariable` and `IsAllArgsBasicTypes` is `ArgsAllBasicTypes`
- `generator.IsJumpOperator` is `OpcodeOperatorJump`, and `cxfx.CxfxTexturePixels.CxfxTexturePixelsData` is `CxfxTexturePixelsValues` (#569)
- The exported functions and methods of t3-ast that started with a verb end with it, so a subject leads and related names sort together (#569). `GetCXAtomicOp` is `CXAtomicOpGet`, `AddInput` is `InputAdd`, `MakeFunction` is `FunctionMake`, `AllocCXArgument` is `CXArgumentAlloc`, `ResizeMemory` is `MemoryResize`, `AllocateStrAt` is `StrAllocateAt`, and so on for the 129 names of docs/NamingPlan.md
- The exported functions of the other packages that started with a verb end with it as well (#569): `RegisterPackage` is `PackageRegister`, `InitConstants` is `ConstantsInit`, `NewLexer` is `LexerNew`, `NewTestRunner` is `TestRunnerNew`, `StartCPUProfile` is `CPUProfileStart`, `MakeProgram` is `ProgramMake`, `DeclareImport` is `ImportDeclare`, `AddNativeInputToExpression` is `ExpressionNativeInputAdd`, `AddInitFunction` is `SysInitFunctionAdd`, and so on for the 110 names of docs/NamingPlan.md
- `cx build` without `-o` names the output as `go build` does, after the first source file without `.cx` or after the source directory, in the working directory and with `.exe` on Windows, so `cx build fib.cx` writes `fib` and `cx build --target=llvm fib.cx` writes `fib.ll` and `fib`. It was the error `no output file`, which now remains only for sources that give no single name, such as two directories
- Functions live in `CxIrState.CXFunctionMng` and are named by `CXFunctionGid` (#351, #352). `CXProgram.CXFunctions`, `CXProgram.AddFunctionInArray`, `CXProgram.GetFunctionFromArray` and the type `ast.CXFunctionIndex` are removed. `MakeFunction` allocates and returns the Gid and the function, `CXPackage.RemoveFunction` takes the program and releases the function, and `execute.Callback` and `execute.MakeCall` take a `CXFunctionGid`. `CXFunctionMng.CXFunctionGet` reads a function, `CxIrStateFunctionGetOrNil` the operator of an expression, which is `CXFunctionGidNil` for a declaration where it was -1. Serialized programs are format 9 and must be recompiled
- `t5-cipher` is `t4-cipher`, as it no longer imports `t4-opcodes`. `core.CoreLibrariesMissing` takes the libraries whose opcodes the binary links, by the build tags of the core. `cx-core` sets its own opcode switch from the libraries its tags link, with no registration (D119)
- The simple GL and GLFW opcodes, 88 of them such as `gl.Color3f`, `gl.Clear`, `gl.Uniform4f` and `glfw.SwapBuffers`, make their call inline in `CxfxOpcodeDispatch` with the operands read at their offsets, instead of calling a handler. A row of `constants.OpcodeDefinitionTable` outside the core may name a Go call as its handler, `cxfx.gl.Clear(uint32(%0))`, and t2-cx-core-opcodegen writes it into the case. The handlers and `cxgl` wrappers they replace are gone. `gl.DeleteProgram` deletes the program. It called `glDeleteShader` (#432)
- `ast.ProgramMemoryRelease` leaves the memory and the collector scratch of a program that is done to the next program `RunPrepare` prepares in the process (D106). Loading programs one after another no longer allocates and clears about 85 MB each time. The prepare step of `BenchmarkLoad` goes from about 7 ms to 0.5 ms and the REPL and the playground release each program before the next (#407)
- The heap has a fixed cap, set when the program is loaded from `-heap-max`, 64M by default, or from the built program, where `cx build` now stores it (D42). `RunPrepare` allocates program memory for the cap once, with the scratch and the root table of the collector, so a run that grows the heap or collects makes no Go allocation and test-gc.cx makes 0 instead of 90. The collector stays mark and compact, but finds objects and the references to update in bitmaps and keeps the type of a queued object in its header, instead of a sorted list and a map. An allocation past the cap is `CX_RUNTIME_HEAP_EXHAUSTED_ERROR` with `out of memory: N bytes requested with T taken do not fit the heap cap of C bytes, raise it with -heap-max`, where it printed `heap exhausted`. Near the cap the collection runs before the heap is full, so `-heap-max 110 issue-297b.cx` now fails where the program means to and not in its garbage loop. Serialized programs carry the cap, format 7 (#407)
- The opcode set is fixed when cx is compiled (D21). `constants.OpcodeDefinitionTable` gives every opcode its handler and signature, and `t2-cx-core-opcodegen` writes the signature tables and the opcode switches, `CoreOpcodeDispatch` and, with the cxfx tag, `CxfxOpcodeDispatch`, whose cases call the handlers directly. `ast.OpcodeHandlers`, `ast.Natives`, `ast.Operators`, the registration of opcode names and `RegisterOpcodes`, `RegisterOpCode`, `RegisterFunction` and `RegisterOperator` are gone. `make check-opcodes`, part of `make check`, fails when the generated files are out of date. As the tier rule requires, t5-regexp, t5-cxos-core and t5-cx-runtime-printer are now t4, t6-cx-runtime-core is t5-cx-runtime-core and t7-cx-core-run is t6-cx-core-run (#432)
- The system library functions are fixed when cx is compiled (D96). `constants.LibraryFunctionDefinitionTable` gives every live library function its handler, vm form and signature, and `t2-cx-core-opcodegen` writes the library signature tables and one switch per library in the package of its handlers and under its build tag, `LibraryFileDispatch` and `LibraryProcessDispatch` in t5-cxos, `LibraryNetDispatch` in t5-cxnet and `LibraryCipherDispatch` in t5-cipher, each with a `Vm` switch of its vm forms. `LibraryCall` runs the switch after the unchanged sandbox check (D16, D17). A binary links a library with `opcodes.LibraryLink`. `ast.LibraryFunctionHandlers`, `ast.LibraryFunctionNatives`, `vm.LibraryFunctionVmHandlers`, `RegisterLibraryFunction`, `RegisterLibraryFunctionVm` and `MakeNativeFunction` are gone, and the library handlers are exported (#432, #492)
- The old loop makes no Go allocation while it runs a program that does not print composite values, call a system library, grow the heap or collect it. `execute.RunPrepare`, which `ExecutorRun` calls before the run, sizes the heap, the operand buffers to the largest arity and the format buffer. `feedOSArgs` no longer builds an error when `os` is not imported, and the per run `rand.Seed`, a no-op since Go 1.24, is gone. `TestRunAllocs` requires 0 for every `BenchmarkRun` program and the shrunk `t0-benchmarks/compare` programs, test-gc.cx aside, `TestRunNoGC` requires no Go GC cycle during a run, and `docs/RunPathAllocations.md` lists the VMALLOC codes of what still allocates (#407)
- The str opcodes of the old loop read strs as views into program memory and write new strs straight into it, and print, printf, sprintf and the str conversions format into one buffer kept on the program. They no longer copy every str into a Go string, and test-gc.cx makes 98 Go allocations per run instead of 202,756. Output is unchanged. The str to f32 and f64 conversions parse the bytes with the new `t0-cx-core-floatparse`, a port of `strconv.ParseFloat` of Go 1.27.3 that gives the same bits and the same errors without a string (#412, D92)
- The compiler finds a built-in opcode, library function or constant by the name Gids of its package, receiver and name. `ast.OpNames`, `ast.OpCodes` and `actions.ConstCodes` are gone. `ast.NativeNames` and `actions.ConstNames` hold the names sorted, a program compares the bytes of a name with them once, and then finds it by its Gids in a cache in `CxIrState`. No lookup builds a `pkg + "." + name` key (#413)
- The names of a `global` statement are name Gids from the parser, and the compiler builds their text only for an error. `CxIrStateManagerLen` and the unused manager check of the serializer carry the type tag of the manager instead of its name (#413)
- The garbage collector runs as a tick of the old loop instead of inside the allocator. The allocator grows the heap and counts the bytes since the last collection, and after each step `GarbageCollectorCheck` compares them with the heap free after that collection and `GarbageCollectorTrigger` collects. The collector moved to `t2-cx-runtime-gc`, which sees only a root table and the heap, and only `t3-ast` imports it. Nothing moves inside an opcode any more, so `Heap.Pinned` is gone. A single step that needs more than the room left below `-heap-max` is a heap exhausted error, where the inline collection could have freed earlier garbage for it. Op counts are unchanged. `docs/GarbageCollector.md` describes the interface and compares reference counting by type with it (#485)
- `types.Allocator`, `types.Write_str`, `types.Write_obj`, `types.AllocWrite_obj_data` and `types.AllocWrite_str_data` are gone. They wrote through the memory slice from before the allocation, so `regexp.MustCompile` wrote past the end of memory when the allocation grew the heap. `-heap-initial 0 test-regexp.cx` failed with `index out of range`. It and the cxfx callers allocate through `t3-ast` (#485)
- The unused `ast.DisplaceReferences` is gone (#485)
- `cx build` links a system library into the core only when the program calls it and `--enable` enables it. `make build` builds a core per set of libraries, `cx-core` for none to `cx-core-file-net-process`, and `cx build` picks the one for the program, from `--core` when it names a directory. The core without a library is 5.0 MB, with file 6.0 MB, with net 5.9 MB and with all three 7.1 MB, where it was 8.6 MB with tcp and its `net/rpc` server. `cxos.InitConstants` takes the constant registrar, so `t5-cxos` no longer imports the compiler (#492)
- `t5-tcp` uses `net` only. The unused `net/rpc` server pulled `net/http`, `crypto/tls` and `html/template` into every binary with the tcp functions (#492)
- The serialized program is a snapshot of the arenas and managers with a 24 byte header, the magic `CXPROGRM`, format 2, the pointer size and the language version, and nine length prefixed sections. A deserialized program lowers and runs as the original, escaped locals and GC roots included. `Deserialize` returns an error instead of panicking on malformed bytes and refuses a program from another pointer size. `SerializeDebugInfo` reports the size of each section, and the skyencoder `SerializedCXProgram` is gone. See "Serialized form" in docs/IR.md (#12, #292, #293, #295)
- `serialize` returns `[]ui8` and `deserialize` takes `[]ui8`, so they round trip. `deserialize` of bytes that are not a program is the runtime error `CX_RUNTIME_INVALID_ARGUMENT`
- `CXCall.Call` fetches the atomic operator and its function once per step and passes them to `processBuiltInOperators`, `processNonAtomicOperators` and `wipeDeclarationMemory`, which fetched them again. The jmp, goto, break and continue handlers still fetch the atomic operator themselves (#417)
- `CXCall.Call` takes a pointer to the current expression instead of a copy, so a step no longer allocates. `CXValue.Expr` is cleared when the opcode handler returns. Allocations per run of a 100000 iteration loop fall from 1,000,019 to 11 and run time by about 30% (#407, #417)
- `CXTypeSignature_Struct` objects live in `CxIrState.CXTypeSignature_StructMng`. Struct signatures are allocated there, `CXProgram.TypeSignatureForStructs` and `AddCXTypeSignatureStructInArray` are gone, and `GetCXTypeSignatureStructFromArray` reads the manager by slot index (#299, #321)
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

- cx no longer panics when CXPATH is unset and the user lookup fails, as on js/wasm. It takes the home directory from $HOME, and without one it says so and runs on
- A program that uses a library opcode its run does not enable is refused by `execute.RunCompiled` and `lower.Program` as well as by `core.ExecutorRun`, through the new `CXProgram.CXProgramOperationRefused`, so an embedder with its own executor cannot run it (D119). `lower.Program` reports it as `ReasonOperationRefused` and lowers nothing
- `gl.UploadImageToTexture` of an `.hdr` file reads the `-Y height +X width` line the right way round. It took the height as the width, so a texture that was not square decoded wrong. The HDR header and the image format are read as bytes, which drops STRHOT-325 and leaves STRHOT-197 a failure branch only (#432)
- The old loop stops a program at the call depth limit with the stack overflow error of vm, exit code 6. It indexed one past the call stack and crashed. A runtime error in `*init` is a runtime error with exit code 5, as on vm, where it was a Go panic (#514)
- A method whose receiver is not a struct, `func (x i32) f()`, is compile error 7081, and a method of a struct that does not exist reports the missing struct. Both crashed the compiler. An assignment to an undeclared `v.x` after another declaration, `var d T` then `v.x = 1`, no longer crashes the compiler either (#515)
- `TestLowerCompare` and the other in-process compiles of the tests compile through `compileSourceStrings`, as the cx command does, so they report the compile errors of `*init` too. They accepted initialization cycles and global type errors (#516)
- The REPL reports the compile errors `AddInitFunction` records, such as an initialization cycle, and does not run the program. It ran it anyway (#519)
- A function whose input escapes kept a pointer into the function arena across the copies of its escaped inputs, which may grow the arena, so its parameters and size could be written to a stale copy (#492)
- A package imported by its path, such as `math/geometry`, is compiled once however many files and packages import it. Each import was looked up by its full path and the package is named by its last element, so a second import compiled it again and reported its methods as redeclared. A struct slice literal element of the wrong type, such as `U{a: 1}` in `[]T{U{a: 1}}`, is 7051 at the start of the element instead of at its closing brace. A field, method call or index may follow a struct literal, as in `T{a: 1}.m()` and `pkg.T{}.m()`, and `&pkg.T{...}` is a value in every position. A method of value receiver called on a pointer, `p.m()` or `(&t).m()`, gets the value p points to instead of `expected input argument of type 'T'; '*T' was provided`. `make generate-objmodel` writes the site counts of `docs/ObjectModel.md` from `docs/objectmodel-sites.csv`, which list 1302 sites, and `make check-objmodel` fails when they differ
- Every independent compile error is reported once, at `file:line:column`, sorted by file, line and column. A call of a missing function, native or package member, a built-in package used without its import, a loop condition without a value, an invalid indirection, a repeated struct field and a missing type in a declaration no longer stop the compiler, and the bodies are still checked after a missing type. A syntax error is a record like the others and names the unexpected and expected tokens, as in `syntax error: unexpected comma, expected ]`. `*x` of an `i32` x is `cannot dereference non-pointer 'x' of type 'i32'` (skycoin-cx#15). Nested slice literal elements have their full type, and a call with the wrong number of inputs no longer also reports its receivers. The serialized format is 4, with the column of `CXArgumentDebug` and `CXLine`
- A file without a package clause is `no package clause` at line 1:1 and is not compiled further. Its declarations went to the last package registered, `tcp`, and a type there panicked. A type, function, global or method declared twice in a package is `'T' redeclared in this block, other declaration at file:line:column` or `method T.m already declared at ...`, the uses take the first declaration, and a global redeclared with another type no longer panics. Every slice literal element of the wrong type, nested ones included, is `cannot use value of type 'U' as type 'T' in slice literal` (7051) at the element. A missing type in a field, parameter or variable is reported at the type name. A string with a newline is `newline in string` at its start and ends there, so the errors after it print. A package that cannot be loaded is `could not import` at the import, and its members report nothing more, nor do the fields of a value of no valid type. The serialized format is 5, with the position of `CXStruct` and the column of `CXFunction`
- An imported package with compile errors no longer ends the compile. Its errors print with the errors of the program, at the path of the imported file relative to the working directory, and its members report nothing more, like a package that cannot be loaded. `could not import` names the path relative to the working directory or as `$CXPATH/src/...` instead of the absolute path, and prints once per package whatever the spelling of the import path. A method call on a value of no valid type reports nothing more. `func (a T, b T) m()` is `method has multiple receivers` (7056) at the second receiver instead of a panic. A type and a function or global of the same name in a package are `'X' redeclared in this block, other declaration at file:line:column` at the later one. A call element of the wrong type in a struct slice literal, as `makeU()` in `[]T{makeU()}`, is 7051 at the element. The partial parser grammar drops the unused `identifier_list` and `int_value` rules
- Compile errors print in file and line order, an identical error prints once, and a recoverable error no longer stops the compiler. `y := noout()` still declares `y`, as a placeholder of the new type `types.INVALID`, so its uses are no `identifier 'y' does not exist`. A missing method, a method call on a primitive or undeclared receiver, a method without a result used as a value and a call with a wrong number of inputs or receivers are recorded, and the errors after them in the same function are reported without a cascade through the method result. `z = "s"` prints `cannot assign value of type 'str' to identifier 'z' of type 'i32'` once instead of with `trying to assign argument`, and names `z` without its scope number. An element of the wrong type in a slice literal is `cannot use value of type 'U' as type 'T' in slice literal`, `CompileErrorCodeSliceLiteralElementType` (7051). `operator 'f' expects 1 input/s` ends with `input arguments were provided` on one line
- `append` checks the full type of a call result or literal element, so `append(a, f())` with an `i32` or `[3]i32` result into a `[][2]i32` and `append(b, []i32{1})` into a `[][]i64` are compile errors naming both types, at the line of the call
- `x := v` typed `x` as the element type when `v` was a slice, an array, an element of a `[][]i32` or `[2][3]i32`, a slice or array field, or `&a` of an array. `x` takes the full type of `v`. `d = r[1]` on a `[2][3]i32` copied one element
- `append` that collected could lose the elements of the slice it grew, as `tests/test-gc.cx` did on a ptr64 build, which `make test-ptr64` now runs
- `cx -ast` prints functions, structs and imports in declaration order instead of map order
- An assignment from a function without outputs stopped the compiler, so the errors after it, such as a missing method, were not reported
- After a compile error in one function, every `var` in later functions was dropped, which reported false `identifier does not exist` errors and could panic the compiler in `ProcessShortDeclaration`. An assignment to an undeclared variable is a compile error, not a panic
- `operands are not of the same type` names the types, as in `i32 != ui64`, instead of type codes
- A slice field read through a global pointer to a heap struct, as in `gp.s[i]`, `(*gp).s[i]` or `len(gp.s)`, failed with `CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE`. `tests/test-gc.cx` passes and runs in `make test` (#13)
- The temporary of `a[0].p.y + a[1].n` took the size of the array element, so vm refused the function with `operand outside frame`
- `[3]*i32`, `[3]*Point` and `[3]pack.T` were syntax errors, and `[3]pack.T` without the import also printed a second error. An array of pointers or of a struct of another package can be declared in variables, fields, parameters and results, and an array of pointers is sized as pointers. Without the import the error is `package 'pack' not imported` (#42, #43, skycoin/cx#521, skycoin/cx#526)
- An element of an array or slice literal can leave out its type, as in Go: `[][]i32{{1, 2}}`, `[2][2]i32{{1, 2}, {3, 4}}` and `[]Point{{x: 1}}`. An elided element of a type that cannot take one is the compile error `invalid composite literal type 'i32'`, `CompileErrorCodeLiteralElementType` (7049), and an elided struct element without field names is `struct literal of type 'Point' must name its fields`, `CompileErrorCodeStructLiteralFieldNames` (7050). `x := [][]i32{...}` typed `x` as `[]i32`, and `f([][]i32{...})` and `f(g())` with a `[][]i32` result failed to compile or panicked the compiler (#50, skycoin/cx#37)
- The type checker stored the lengths of a parameter or result array innermost first and the grammar outermost first, so `m = f()` with `f` returning `[2][3]i32` was the false error `expected receiving variable of type '[2][3]i32'; '[3][2]i32' was provided`. Lengths are outermost first everywhere, and type errors print them in that order
- `read()` made a new reader on every call, so the first call buffered the rest of a piped stdin and the second call lost it, and the end of input was `CX_INTERNAL_ERROR` (exit 3). One reader set by `CXProgram.ProgramStdinInit` serves the whole run, and the end of input returns `""` for now (skycoin/cx#110)
- `x = append(t.s, u)` and `p.s = append(p.s, u)`, with `s` a slice field of a struct or of a struct pointer, copied only the slice header size of each element, so a struct element lost every field past the first 4 bytes and an `f64` element was 0. Through a pointer the append and `len(p.s)` were runtime errors
- `import "sys/gl"`, `"sys/glfw"`, `"sys/al"` or `"sys/gltext"` in a cx built without the cxfx tag is the compile error `package "sys/gl" needs a GUI build of cx, build it with make build-gui`, `CompileErrorCodeImportGuiBuildMissing` (7048), instead of `identifier 'gl.Init' does not exist` at the first call (skycoin/cx#432)
- `cx -h` names the real flags. `-l` is an alias for `-lexer`, `-hi` and `-hm` for `-heap-initial` and `-heap-max`, `-co` is renamed `-lexer-output` because it only redirects `-lexer`, and `-ast` works after other flags. `-debug-lexer` and `-Dl`, which did nothing, are removed (skycoin/cx#567)
- Struct, array and slice literals are values. `f([]i32{1, 2})`, `f(P{x: 1})`, `T{f: U{a: 1}}`, `[2]P{P{x: 1}}`, `return P{x: 1}`, `x := [2]i32{1, 2}` and `var d Box = Box{v: 3}` failed to compile, `P{x: i32.add(f(), 1)}` named a temporary as a field, `T{s: []U{U{a: 1}}}` panicked the compiler, and `t = T{}` did nothing. A struct literal now fills a new zeroed variable, so an unnamed field is zero and every field is computed before the target is written. An array literal longer than its array is the compile error `array literal of length 2 has 3 elements`, `CompileErrorCodeArrayLiteralLength` (7046), and a field named twice is `duplicate field name 'a' in struct literal`, `CompileErrorCodeStructLiteralFieldRepeated` (7047) (#40, #51)
- `p = &s` on a `*[]str` copied the slice instead of taking its address, so appends through `p` were lost
- `-` and `!` on a call, a parenthesized expression or a struct field failed to compile, so `!f(x)`, `!(a && b)` and `-foo()` were `invalid indirection` and `-t.x` was `identifier 'x' does not exist`. They apply to any operand now (#28, #56, #58)
- `i++` and `i--` on a type other than `i32` were `operands are not of the same type`, reported at the end of the function. The 1 takes the type of the operand, and an operand type mismatch is reported at its own line. `p.x++`, `s[i]++` and `p.x += 1` wrote to `p` or panicked, and work now (#45)
- `i++` as an argument or an operand gave a wrong value, so `sprintf("%d", i++)` printed `%!d(MISSING)`, and `++i` did nothing. As in Go, `i++` and `i--` are statements and both are syntax errors (#18)
- `a, b := f()` and `a, b := 1, 2` failed to compile, `a, b = x || y, x && y` panicked the compiler, and `f(), g()` as a statement panicked it too. `=` and `:=` take several targets with as many values or one call with as many results, every value is computed before any target is assigned, and a different count is the compile error `assignment mismatch: 2 variables but 3 values`, `CompileErrorCodeAssignmentCount` (7045). A comma list without an assignment is a syntax error (#23, #44)
- `b := Box{v: -1}` and other struct literals with a computed field value did not declare `b`. `x := y` with an `i64`, `f64` or struct `y` and `v := p.y` overwrote the next variable or panicked
- A method call used as an operand, an argument, an index, a condition, a struct literal field, the right side of `+=` or a receiver compiled to a read of a field named like the method, so `i32.print(p.sum())` printed garbage, and `p.show(s)` or `p.hit(&q)` with a first argument that is not a literal was `illegal method call or field access`. A method call now works anywhere an expression can appear, with any arguments, and `for c.next() {` runs its condition every time (#24)
- A pointer receiver method called on a value, `p.move(1)` with `func (p *Point) move(n i32)`, got the value as the pointer and wrote to the wrong address, and the same call on a `*Point` variable was a type error. The value is passed by reference and the pointer as it is
- A method with no output used as a value is the compile error `method 'Point.show' returns no value`, `CompileErrorCodeMethodResultMissing` (7044)
- `pkg.F` without parentheses called `F`. CX has no function values, so it is read like a bare `F`, which is `invalid expression` as a statement and a type error elsewhere (#48)
- `f() ()` called `f` once and ignored the second argument list. Calling the result of a call is the compile error `the result of a call is not a function`, `CompileErrorCodeCallResultCall` (7043) (#22)
- `main.test1()` in another package found `test1` in whichever package main imported last. `pkg.f` names only a function declared in pkg, and is `identifier 'main.test1' does not exist` otherwise, as for a global (skycoin/cx#9)
- A function named like a built-in, such as `add` or `len`, was never called, as the built-in won. A function declared in a package shadows the built-in of the same name in that package, and operators such as `+` still use the built-in (#294)
- An arithmetic or bitwise result was stored in a variable of another type without a check, so `var f f32 = -1` printed NaN and `var x i64 = -1` printed 4294967295. It is the compile error `cannot assign value of type 'i32' to identifier 'f' of type 'f32'`, as for a literal. Use `-1.0` or `-1L` (#27)
- `append`, `insert`, `resize`, `remove` and `copy` did not check the element type, so `append(s, b)` with a `fooB` into a `[]fooA` compiled and ran. A value or result of the wrong type is the compile error `function 'append' expected argument of type 'fooA'; 'fooB' was provided`, `CompileErrorCodeSliceElementType` (7042) (#55)
- A struct field slice with more than one dimension, `m [][]i32`, reported the type of `st.m` as `[]i32` and of `st.m[0]` as `i32`, and `st.m[i][j]` was `invalid indexing`. Its type has one `[]` per dimension and each index reads the next slice
- A global declared twice in one package compiled and the second declaration won. It is the compile error `'i' redeclared` (#57)
- `s[1,0]` acted as `s[1]` for reads and writes. A comma inside an index is a syntax error (#59)
- `goto` to a label that does not exist compiled and did nothing. It is the compile error `label 'name' is not defined`, `CompileErrorCodeGotoLabelMissing` (7041)
- `x.y` on a variable that is not a struct compiled and ran with exit 0. It is the compile error `illegal method call or field access on identifier 'x' of primitive type 'i32'` (#14)
- `[]T{T{x: 1}}` with a struct element type assigned the fields to the slice variable itself, so `len` printed garbage that changed with the string fields. Each element is now built in its own variable and appended, and an element of the wrong type is a compile error. `[]i32{}` and `[]T{}` give an empty slice instead of leaving the target unchanged (#47)
- `(*p)[i]` on a `*[]i32` read the pointer, and `len(*p)` stopped with "type is not known". Both index and measure the slice p points to, `t = *p` copies the whole slice header for 1 byte element types too, and `p[i]` or `*p[i]` is `invalid indexing` (#49, #267)
- `append(p, v)` on a `*[]T` compiled and did nothing. A pointer passed where append, resize, insert, remove or copy take a slice is the compile error `function 'append' expected a slice as input argument`, `CompileErrorCodeSliceArgumentType` (7040). `append(*p, v)` is the valid form (#52)
- `f()[0] + 1` panicked the compiler for an array or slice result, and `f()[i]` with a variable index on a slice result read element 0
- `number.A[1]` on a global array or slice of an imported package panicked the compiler, for reads and writes
- `resize` or `os.Read*Slice` past twice the capacity set the length past the allocation, so the next allocation overwrote the slice. The capacity now grows to at least the new length, and a slice too large for the heap is `CX_RUNTIME_HEAP_EXHAUSTED_ERROR` instead of a wrapped size. `os.Read*Slice` reads its ui64 count in full and returns false for a count past what the heap can hold
- `t = T{n: "s"}` crashed the compiler with a nil dereference instead of giving a compile error
- `&&` and `||` evaluated both operands, so `i < len(s) && s[i] > 0` read past the end of `s` and a call on the right always ran. A jmp on the left operand now skips the right one, as in Go, on the old loop and on vm. `bool.and` and `bool.or` stay ordinary calls that evaluate both arguments (#26)
- `break` and `continue` in an inner loop left every enclosing loop, because each enclosing loop retargeted them. A loop now turns each one it resolves into a jump (#39)
- `for true { ... }` never ran its body and `for b { ... }` on a bool variable was a syntax error. A literal or variable condition is read in place, and the for header no longer takes `b {` as a struct literal, so `for ...; i = j {` parses too. An empty condition, `for i := 0; ; i++`, is true instead of a host panic (#20, #21)
- `break` or `continue` outside a loop compiled and ran on. It is the compile error `break is not in a loop`, `CompileErrorCodeLoopJumpOutsideLoop` (7039) (#60)
- Two sibling `{ }` blocks that declared the same name reported a redeclaration (#16)
- A `goto` inside a labelled loop jumped to the loop's own label, since the label overwrote the goto's target
- Small invalid programs crashed the compiler with a Go panic, exit 2, instead of a compile error. An unknown type in an array, slice, pointer, global, field or parameter declaration, more indexes than an array has dimensions, indexing a scalar, `return` with more values than the function has outputs, an unknown function inside an operator or a condition, and `append(s)` without a value are compile errors now
- A struct that holds itself by value, directly or through another struct, sent the compiler into unbounded recursion. It is the compile error `invalid recursive type`, `CompileErrorCodeStructRecursive` (7038)
- An array, struct, frame or data segment past 4 GiB wrapped its size, so variables silently overlapped. It is the compile error `'name' is too large`, `CompileErrorCodeDeclarationTooLarge` (7037), in both pointer widths
- `lt(true, false)`, `true < false` and `bitand(6.0, 3.0)` compiled and then crashed at run time with exit 5. An operator on a type it has no version for is the compile error `invalid argument types`
- A function that returns an array, `func f() (arr [2]i32)`, panicked the compiler with "type is not known". Assigning, declaring, returning, passing and indexing such a result now work
- Assigning to a global of an imported package, `number.Num = 11`, panicked the compiler. It works, as the guide says globals are available to importing packages
- A call of a method a struct does not have is the compile error `type 'T' has no method 'M'`, `CompileErrorCodeMethodMissing` (7036)
- Errors on array element types, struct fields and return counts had no file or line
- A block whose first or last statement was another block or a method call lost a scope or the method call, so valid programs crashed the compiler or got a false redeclared error, and a one statement block had no scope. Scope starts and ends are now counted per expression in `ExpressionScopeOpenCount` and `ExpressionScopeCloseCount`
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
