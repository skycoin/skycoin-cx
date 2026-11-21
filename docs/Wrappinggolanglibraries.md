# Wrapping a Go library for CX

A CX built-in package wraps Go code as a set of native functions. The steps
below use [t5-tcp](../t5-tcp) as the worked example. It is small and has
every part.

1. **Give every function an opcode row.** Add a row for each function to
   `constants.OpcodeLifecycleTable` in
   [t1-constants/opcodes.go](../t1-constants/opcodes.go), taking the next
   number after the highest, and mirror it in [opcodes.csv](opcodes.csv).
   t5-tcp has `{206, "tcp.Dial", 0, 0, 0}` and the rows after it.
   `opcodes.RegisterOpCode` asserts that the row is live and has the same
   name. See [Conventions.md](Conventions.md) rules 7 and 8. This step may
   change once the maintainer settles the static opcode table.

2. **Write the handlers.** A handler has the signature
   `func(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue)`.
   It reads its inputs and writes its outputs with the `CXValue` getters
   and setters. `opTCPDial` in [t5-tcp/op_tcp.go](../t5-tcp/op_tcp.go)
   reads two strings with `inputs[0].Get_str(prgrm)` and writes its result
   with `outputs[0].Set_str(prgrm, ...)`.

3. **Write `RegisterPackage(prgrm *ast.CXProgram)`.** In
   [t5-tcp/init.go](../t5-tcp/init.go) it calls
   `ast.MakePackage(prgrm.CXProgramNameIntern("tcp"))`, then
   `prgrm.AddPackage` and `AddStruct` for the `Dialer` struct, then
   `opcodes.RegisterFunction(prgrm, "tcp.Dial", opTCPDial, opcodes.In(...), opcodes.Out(...))`
   for each function. There is no `func init`.

4. **Call it from `RegisterPackages`** in
   [t7-init/init.go](../t7-init/init.go), next to `tcp.RegisterPackage(prgrm)`.

5. **Make it importable.** Add the package name to `DefaultPackages` in
   [t0-packages/packages.go](../t0-packages/packages.go). CX code then
   imports it as `sys/<name>`.

6. **Respect the tiers.** The package is named `t<N>-<name>`, where N is one
   more than the highest tier it imports, and it must sit below t7 so
   t7-init can import it. t5-tcp imports t3-ast and t4-opcodes, so it is
   tier 5.

To make the package optional, add a build tag and a `noinit.go` file with an
empty `RegisterPackage`, as t5-regexp does with `noregexp`. Pass concrete Go
types to the wrapped library, not interfaces. `make check` runs the tier,
init and object model checks.
