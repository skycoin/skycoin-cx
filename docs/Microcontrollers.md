# TinyGo and microcontrollers

This page describes the TinyGo builds of cx. They are an alternative to the
Go builds, which stay the default everywhere.

## cx with TinyGo

`make build-tinygo` builds cx and cx-core with TinyGo, and
`make build-wasm-tinygo` builds cx for js/wasm. `TINYGO` names the tinygo
to run.

TinyGo has no `os.Root` and no `os.SameFile`, has no CPU profiler and cannot
link the amd64 assembly of `lukechampine.com/blake3`. A TinyGo build
therefore uses, by the `tinygo` build tag:

* a `sandbox.Root` that checks each path in Go before the call: it must be
  local and no part of it may be a symlink, except the last one for a
  remove. A symlink made between the check and the call is not caught,
  which `os.Root` catches;
* a portable BLAKE3 in `t0-cxhash`, which a Go test checks against
  `lukechampine.com/blake3`;
* a path comparison with symlinks resolved in place of `os.SameFile`;
* no CPU profile rate.

The Go builds keep `os.Root`, `os.SameFile` and `lukechampine.com/blake3`.

With TinyGo 0.45.0-dev of the 0magnet fork on x86-64 Linux, a TinyGo cx
passes every execution record in every form, run with
`CX_EXECUTION_RECORDS_CX` (see [ExecutionRecords.md](ExecutionRecords.md)).

The js/wasm build compiles but does not run in the playground: the js/wasm
runtime of upstream TinyGo passes no arguments, environment or file system
from JavaScript.
