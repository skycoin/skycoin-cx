# Overview Over the CX Codebase

This file contains an overview of the CX codebase.  It is recommended reading
if you want to get involved in developing CX.

## Directory structure

Every Go package sits one level below the root as `t<N>-<name>`, where N is
its tier in the import graph. A package imports only packages of lower tiers.
See the README for the rule.

### CX Source Code

* **t3-ast/** the data structures representing a CX program, the memory
model and the serializer. The root of the tree is the `CXProgram` struct.
* **t4-opcodes/**, **t4-execute/** the runtime part of CX.
* **t5-cxpartialparsing/**, **t8-cxparsingcompletor/**, **t9-cxparsing/**
the parser. It analyzes the source code and builds the tree above.
* **t11-cx/** the cx binary.
* **vendor/** 3rd party code needed to build CX. It is not part of the CX
sources themselves.

### Tests and benchmarks

* **tests/** product tests and regression tests for the CX parser and
interpreter, run by `t1-cxtest` (`make test`). See the file
`CONTRIBUTING.md` for more details about the tests.
* **t0-benchmarks/** benchmarks of CX against the same algorithms in Go.

## Parser Workflow

The CX parser analyzes the source code of a CX program and
 works in two passes:

1. The first pass (in `t5-cxpartialparsing`) reads all the types, global variables and
function names and stores them.
2. The second pass (in `t8-cxparsingcompletor`) reads the full source code of the program and
creates the object tree representing the program. This tree will be passed to
the interpreter run.

Each pass of the parser contains a lexer and a parser proper.  If you are not
familiar with these terms, we suggest that you read an introductory text about
computer compilers.

## Interpreter

The interpreter interprets the internal representation of the program. This
representation is created by a tree of the data structures in the following
subsections.


