![cx logo](https://user-images.githubusercontent.com/26845312/32426758-2a4bbb00-c282-11e7-858e-a1eaf3ea92f3.png)

# CX Programming Language
 
[![check](https://github.com/skycoin/skycoin-cx/actions/workflows/check.yml/badge.svg?branch=main)](https://github.com/skycoin/skycoin-cx/actions/workflows/check.yml)

CX is a general purpose, interpreted and compiled programming
language, with a very strict type system and a syntax
similar to Golang's. CX provides a new programming paradigm based on
the concept of affordances.

The rules of this repository are listed in
[docs/Conventions.md](docs/Conventions.md).

## Repository layout

Every Go package, binaries included, sits one level below the root as
`t<N>-<name>`, so no Go file is deeper than `<dir>/<file>.go`. N is the
package's tier in the import graph. A tier 0 package imports no other package
of this module, and a tier N package imports only packages of lower tiers,
tests included. `t11-cx` is the cx binary, `t1-cxtest` the CX program test
runner. `tests/` holds the CX test programs, `lib/` CX libraries and
`t0-benchmarks/` the CX and Go benchmarks. `CLEANUP.md` lists every file of
the former CX repositories with a keep, merge or delete recommendation.

Init lives in one place. No package has a Go `func init()`, and no setup is
hidden in a constructor or in a package level variable. A constructor only
returns a new object. Setup that touches shared state is an explicit,
exported Init function, and `t11-cx` main calls each one in a numbered order
before the run loop. `make check-inits` fails on `func init()` outside package
main. The files that still break the rule are listed in `init-allowlist.txt`,
which only shrinks, and each one is marked `// VIOLATION: init chaining`.

No interface chaining. An interface does not embed another interface, so
`type ReadCloser interface { io.Reader; Close() error }` is not allowed. List
the methods instead. `make check-ifaces` runs `t0-ifacecheck` and fails on an
embedded interface. The ones that still break the rule are listed in
`iface-allowlist.txt`, which only shrinks.

Two classes of error, no recovery. A failure of a path, a file, the network
or anything else outside the program returns its module's error struct
pointer, nil on success, never the Go `error` interface. Every other failure,
a precondition, invariant or postcondition violation, a mathematical
violation or an impossible state, is a strict assert from `t0-cx-core-assert`
that crashes the process. No code calls `recover`. `make check-recovers`
fails on a `recover(` call outside `vendor/`, and the sites that still break
the rule are listed in `recover-allowlist.txt`, which only shrinks. See
[docs/ErrorHandling.md](docs/ErrorHandling.md).

State lives in one place. One State object holds every manager, and no
state lives anywhere else, so there are no mutable package level variables.
Each object type has an `<Obj>Def` file, a 64 bit generational index
`<Obj>Gid` and an `<Obj>Mng` manager that holds the array of objects.
Objects are created and returned only through their manager. A pointer kept
at rest, in a struct field or a global, is a Gid instead. A pointer local to
a function may stay. Work goes through the packages by tier, lowest first.
Code that still breaks the rule is marked `// VIOLATION: state outside
State`, `// VIOLATION: pointer at rest` or `// VIOLATION: created outside
manager`, and the marks only go away. See skycoin/skycoin-cx#9.


## Install

CX needs Go 1.27 or newer and `make`. On Windows use Git Bash or MSYS2, so
`make` and the check scripts have a shell.

```
git clone https://github.com/skycoin/skycoin-cx
cd skycoin-cx
make build
```

This puts the `cx` binary in `bin/`. Add that directory to your `PATH` if you
want to run `cx` from anywhere. To update, `git pull` and `make build` again.

## Build

`make build` builds cx with the cipher, os and regexp packages. Leave any of
them out with `TAGS=nocipher`, `TAGS=nocxos` or `TAGS=noregexp`, and
`TAGS=ptr64` switches the VM to 64 bit pointers.

`make build-gui` adds the `gl`, `glfw` and `al` packages (cxfx). It needs cgo
and the GL, X11 and OpenAL headers. On Debian or Ubuntu:

```
sudo apt-get install libgl1-mesa-dev xorg-dev libopenal-dev
make build-gui
```

## Hello world

```
package main

func main() {
	str.print("Hello World!")
}
```

Save it as `hello.cx` and run `./bin/cx hello.cx`. More programs are in
`tests/examples`, and the OpenGL ones in `tests/examples/opengl` need
`make build-gui`.

## Tests

- `make check` runs the tier, init, interface and recover checks, `go vet`, the Go
  tests and the stable CX program tests. CI runs it on Linux, macOS and Windows.
- `make test` runs only the stable CX program tests.
- `make test-all` also runs the tests for known issues. These are expected to
  fail, and each one names its issue in `t1-cxtest/main.go`.
- `make vet-gui` vets the cxfx build.

## Status

Works:

- `make check` passes on Linux, macOS and Windows.
- 175 stable CX program tests pass. `make test-all` passes 182 and fails the
  20 known issue tests.
- The cxfx build compiles and `tests/examples/opengl/triangle.cx` draws its
  triangle under Xvfb on Linux. Other GUI programs have not been checked.

Known broken:

- 20 CX program tests fail, among them `for true {}`
  ([#20](https://github.com/skycoin/skycoin-cx/issues/20)), no short circuit
  evaluation ([#26](https://github.com/skycoin/skycoin-cx/issues/26)) and
  `-heap-max` not enforced
  ([#17](https://github.com/skycoin/skycoin-cx/issues/17)). Issues #13 to #29
  cover them all.
- The GUI tests `issue-208`, `issue-216` and `issue-252` only work against a
  cxfx build and are commented out
  ([#30](https://github.com/skycoin/skycoin-cx/issues/30),
  [#31](https://github.com/skycoin/skycoin-cx/issues/31),
  [#32](https://github.com/skycoin/skycoin-cx/issues/32)).
- `t7-cxmobile` does not compile
  ([#35](https://github.com/skycoin/skycoin-cx/issues/35)).
- Older language bugs carried over from skycoin/cx are tracked from
  [#38](https://github.com/skycoin/skycoin-cx/issues/38) on, for example
  `break` leaving every loop
  ([#39](https://github.com/skycoin/skycoin-cx/issues/39)).

See the [open issues](https://github.com/skycoin/skycoin-cx/issues) for the
full list.

## History

This repository merges [skycoin/cx](https://github.com/skycoin/cx), which saw
its last activity in 2023, with the CX playground repositories, in November
2026. Development continues here. Open issues from skycoin/cx that still apply
were filed again here with a link to the original.

## Resources and libraries

If you are interested in learning more about CX, please refer to the [resources documentation](docs/cx-resources.md). 

If you want to get started with some basic example programs and tutorials check out the [tutorials section](docs/cx-tutorials.md). 

The docs also provide a high level [overview](docs/overview.md) of the language.

Guide for CX compiler development:
* [Compiler Development](docs/CompilerDevelopment.md)
* [The compiler IR and serialized form](docs/IR.md)

How to write a CX library that wraps a Go library:
* [Wrapping Go libraries](docs/Wrappinggolanglibraries.md)
