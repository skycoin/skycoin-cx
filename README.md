![cx logo](https://user-images.githubusercontent.com/26845312/32426758-2a4bbb00-c282-11e7-858e-a1eaf3ea92f3.png)

# CX Programming Language
 
[![Build Status](https://travis-ci.com/skycoin/cx.svg?branch=develop)](https://travis-ci.com/skycoin/cx) [![Build status](https://ci.appveyor.com/api/projects/status/y04pofhhfmpw8vef/branch/master?svg=true)](https://ci.appveyor.com/project/skycoin/cx/branch/master)

CX is a general purpose, interpreted and compiled programming
language, with a very strict type system and a syntax
similar to Golang's. CX provides a new programming paradigm based on
the concept of affordances.

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


Table of Contents

   * [CX Programming Language](#cx-programming-language-1)
   * [Installation](#installation)
      * [Binary Releases](#binary-releases)  
      * [Compiling from Source](#compiling-from-source)
         * [Compiling CX on Linux](#compiling-on-linux)
         * [Compiling CX on MacOS](#compiling-on-macos)
         * [Compiling CX on Windows](#compiling-on-windows)
      * [Updating CX](#updating-cx)
   * [Resources and libraries](#resources-and-libraries)

# CX Programming Language

CX is a general purpose, interpreted and compiled programming
language, with a very strict type system and a syntax
similar to Golang's. CX provides a new programming paradigm based on
the concept of affordances, where the user can ask the programming
language at runtime what can be done with a CX object (functions,
expressions, packages, etc.), and interactively or automatically choose
one of the affordances to be applied. This paradigm has the main objective
of providing an additional security layer for decentralized,
blockchain-based applications, but can also be used for general
purpose programming. 

## Installation

CX requires Go 1.26 or newer. `make build` (or `go build -o bin/cx ./t11-cx`) builds the language with cipher, os and regexp support; `make build-gui` adds the cxfx GL package, which needs cgo and X11/GL headers.

### Binary Releases

You can find binary releases for most major systems on the [release page](https://github.com/skycoin/cx/releases). 

### Compiling on Linux

If you are using an `apt` compatible system, install the dependencies with:

```
sudo apt-get update

sudo apt-get install -y glade xvfb libxinerama-dev libxcursor-dev libxrandr-dev libgl1-mesa-dev libxi-dev libperl-dev libcairo2-dev libpango1.0-dev libglib2.0-dev libopenal-dev libxxf86vm-dev make
```

If you have not set up Go on your machine, follow this [guide](https://www.tecmint.com/install-go-in-ubuntu/) to install and set up Go.

Download CX's repository with git:

```
git clone https://github.com/skycoin/skycoin-cx
```

Get required Go dependencies with:

```
go get -u golang.org/x/mobile/cmd/gomobile
go get golang.org/x/mobile/gl 
```

Navigate to CX's repository.

Build CX's binary and install by running:

```
make build
make test
```

Add the CX binary path to your operating system's `$PATH`. For example, in Linux:

```
export PATH=$PATH:$HOME/skycoin-cx/bin
```

You should test your installation by running:

```
make test
```

### Compiling on MacOS

If you have not set up Go on your machine, follow this [guide](https://www.digitalocean.com/community/tutorials/how-to-install-go-and-set-up-a-local-programming-environment-on-macos) to install and set up Go.

If you do not have `git` installed, install it with:

```
brew install git
```

Download CX's repository with git:

```
git clone https://github.com/skycoin/skycoin-cx
```

Navigate to CX's repository.

Build CX's binary and install by running:

```
make build
```

Add the CX binary path to your operating system's `$PATH`. For example:

```
export PATH=$PATH:$HOME/skycoin-cx/bin
```

You should test your installation by running:

```
make test
```

### Compiling on Windows

Compiling CX on Windows requires a recent version of Git to be installed. 

Pacman is a utility which manages software packages.   

To install pacman, download [MSYS2](https://www.msys2.org) and run the installer.

When the installation is complete, click `Run MSYS2 now`.

If MSYS2 has already been installed, run it through the start menu.

You can run a full system upgrade and install required dependencies with: 

```
pacman -Syu git mingw-w64-x86_64-openal base-devel mingw-w64-x86_64-toolchain

if [ ! -a /mingw64/lib/libOpenAL32.a ]; then ln -s /mingw64/lib/libopenal.a /mingw64/lib/libOpenAL32.a; fi

if [ ! -a /mingw64/lib/libOpenAL32.dll.a ]; then ln -s /mingw64/lib/libopenal.dll.a /mingw64/lib/libOpenAL32.dll.a; fi
```

You can compile CX by running: 

```
cx-setup.bat
```

Test your installation by running:

```
cx lib/args.cx tests/main.cx ++wdir=tests ++disable-tests=issue
```

## Updating CX

You can update your CX installation by running, in the repository:

```
git pull
make build
```

Or on Windows:

```
cx-setup.bat
```

## Resources and libraries

If you are interested in learning more about CX, please refer to the [resources documentation](docs/cx-resources.md). 

If you want to get started with some basic example programs and tutorials check out the [tutorials section](docs/cx-tutorials.md). 

The docs also provide a high level [overview](docs/overview.md) of the language.

Guide for CX compiler development:
* [Compiler Development](docs/CompilerDevelopment.md)

How to write a CX library that wraps a Go library:
* [Wrapping Go libraries](docs/Wrappinggolanglibraries.md)
