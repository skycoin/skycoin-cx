# Wrapping a Go library for CX

A CX built-in package wraps Go code as a set of native functions. The steps
below use [t4-regexp](../t4-regexp) as the worked example. It is small and has
every part. A library that reaches the operating system or links a module
outside the standard library is a system library instead, see
[Conventions.md](Conventions.md) rule 16.

1. **Give every function an opcode row.** Add a row for each function to
   `constants.OpcodeLifecycleTable` in
   [t1-constants/opcodes.go](../t1-constants/opcodes.go), taking the next
   number after the highest, and mirror it in [opcodes.csv](opcodes.csv).
   t4-regexp has the `regexp.Compile` row and the rows after it. See
   [Conventions.md](Conventions.md) rules 7 and 8.

2. **Write the handlers.** A handler is an exported function with the
   signature
   `func(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue)`.
   It reads its inputs and writes its outputs with the `CXValue` getters
   and setters. `OpRegexpCompile` in [t4-regexp/op_regexp.go](../t4-regexp/op_regexp.go)
   reads a string with `inputs[0].Get_str(prgrm)` and writes its error
   with `outputs[1].Set_str(prgrm, ...)`.

3. **Define each function.** Add a row to `constants.OpcodeDefinitionTable`
   in [t1-constants/opcodes_definitions.go](../t1-constants/opcodes_definitions.go)
   with the number, name, handler, inputs and outputs, for example
   `{203, "regexp.Compile", "regexp.OpRegexpCompile", "str", "r:regexp.Regexp str", ""}`.
   A new handler package also needs its group in `constants.OpcodeGroup`, an
   entry in `handlerPackages` of t2-cx-core-opcodegen and, for a build tag,
   a pair of `opcodes_linked_*.go` files and a `dispatchGroups` entry. Run
   `make generate-opcodes`, which writes the signature tables and adds the
   handlers to the opcode switch (decision D21). There is no registration.

4. **Write `RegisterPackage(prgrm *ast.CXProgram)`** when the package has
   types or structs. In [t4-regexp/init.go](../t4-regexp/init.go) it calls
   `ast.MakePackage(prgrm.CXProgramNameIntern("regexp"))`, then
   `prgrm.AddPackage` and `AddStruct` for the `Regexp` struct. Call it from
   `RegisterPackages` in [t7-init/init.go](../t7-init/init.go), next to
   `regexp.RegisterPackage(prgrm)`. There is no `func init`.

5. **Make it importable.** Add the package name to `DefaultPackages` in
   [t0-packages/packages.go](../t0-packages/packages.go). CX code then
   imports it as `sys/<name>`.

6. **Respect the tiers.** The package is named `t<N>-<name>`, where N is one
   more than the highest tier it imports, and it must sit below
   t5-cx-runtime-core, which holds the opcode switch. t4-regexp imports
   t3-ast, so it is tier 4.

To make the package optional, give its files a build tag and the
`RegisterPackage` an empty stand in under the opposite tag, as t4-regexp does
with `noregexp`. Pass concrete Go types to the wrapped library, not
interfaces. `make check` runs the tier, init, opcode table and object model
checks.
