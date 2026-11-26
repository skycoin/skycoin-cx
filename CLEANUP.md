# CX repositories cleanup

Every tracked path of the eight CX repositories, grouped by area. A row is a
directory, collapsed to two levels (three under cx/packages, cxparser/util and
the package loaders), or a single file at a repository root. Files counts git
tracked files under the path. Every file under a row shares its action.

Actions: **keep** stays as it is (moved where Now in says), **merge** folds into
another place or dedupes, **delete** removes it. `(own repo)` and `(games repo)`
mean keep it out of skycoin-cx. Sources: skycoin/cx develop 0c08a443, cx-chains
5d5b2e41, cx-tracker aa3055f, cx-playground 1165f4e, cx-gui-helper 5592a3b,
cx-benchmark a511da9, cx-games d33f1c2, cx-game b3a09ef.

## Summary

| Repo | keep | merge | delete | Files |
|---|---:|---:|---:|---:|
| cx | 742 | 43 | 1127 | 1912 |
| cx-chains | 477 | 2 | 1293 | 1772 |
| cx-tracker | 9 | 8 | 1270 | 1287 |
| cx-playground | 18 | 176 | 1 | 195 |
| cx-gui-helper | 0 | 0 | 5 | 5 |
| cx-benchmark | 0 | 19 | 9 | 28 |
| cx-games | 741 | 4 | 0 | 745 |
| cx-game | 755 | 0 | 862 | 1617 |

Counts are files, not rows.

## Language core

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx | `cx/assert/` | 1 | removed (#107) | `t4-assert` | Nothing imports it |
| cx | `cx/ast/` | 32 | keep | `t3-cx-core-ast` | AST, memory model, serializer and GC |
| cx | `cx/constants/` | 2 | keep | `t1-cx-core-opcodetable` | Opcode, type and exit code constants |
| cx | `cx/globals/` | 3 | keep | `t2-cx-core-environment` | Compiler wide flags and counters, a source of the shared process state |
| cx | `cx/init/` | 1 | keep | `t7-cx-runtime-libraries` | Builds an empty program and registers the stdlib |
| cx | `cx/types/` | 5 | keep | `t0-cx-core-valuetypes` | Leaf type codes and pointer size, imported by everything |
| cx | `cx/util/` | 1 | keep | `t1-cx-runtime-sandbox`, `t0-cx-compiler-sourceargs`, `t0-cx-runtime-filelog` | Split by subject |

## Compiler

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx | `cx/astapi/` | 10 | keep | `t9-cx-core-astapi` | Programmatic AST construction, used by the generator and tests |
| cx | `cx/generator/` | 2 | keep | `t10-cx-tools-programgenerator` | Random program generator for the serializer tests |
| cx | `cxparser/actions/` | 16 | keep | `t4-cx-compiler-astbuild` | Semantic actions called from the grammars |
| cx | `cxparser/cxparsing/` | 3 | keep | `t9-cx-compiler-parsing` | Driver for the parse passes, gained SourceStringsParse |
| cx | `cxparser/cxparsingcompletor/` | 7 | keep | `t8-cx-compiler-fullparsing` | Pass two grammar (.y) and its generated parser |
| cx | `cxparser/cxpartialparsing/` | 5 | keep | `t5-cx-compiler-partialparsing` | Pass one grammar (.y) and its generated parser |

## VM and runtime

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx | `cx/execute/` | 2 | keep | `t4-cx-runtime-interpreter` | The interpreter loop |
| cx | `cx/opcodes/` | 21 | keep | `t4-cx-runtime-opcodes` | Native opcode implementations |

## Standard library

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx | `cx/packages/` | 1 | keep | `t0-cx-core-syspackages` | Registry of built in package names |
| cx | `cx/packages/cipher/` | 3 | keep | `t4-cx-library-cipher` | CX cipher package, on by default now |
| cx | `cx/packages/cxfx/` | 15 | keep | `t6-cx-library-graphics` | GL, GLFW, OpenAL and gltext bindings behind the cxfx tag (cgo) |
| cx | `cx/packages/cxos/` | 7 | keep | `t5-cx-library-fileopcodes` | CX os, json, time and profile packages, on by default now |
| cx | `cx/packages/regexp/` | 3 | keep | `t4-cx-library-regexp` | CX regexp package, on by default now |
| cx | `cx/packages/tcp/` | 2 | keep | `t4-cx-library-netopcodes` | CX net library, batched since skycoin-cx 508 |
| cx-games | `libs/utest/` | 4 | merge | `skycoin-cx/lib/utest` | CX unit test library, runs on the current compiler |

## New compiler (WIP)

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx | `cmd/declaration_extractor/` | 70 | keep | `t6-cx-compiler-declarationextractor` | Declaration extraction for the new type checker, tests pass |
| cx | `cmd/packageloader/` | 3 | keep | `t3-cx-tools-packagecli` | CLI main and makefile for the v1 loader |
| cx | `cmd/packageloader2/` | 3 | merge | `t3-packageloader2` | CLI for the v2 loader, merge into t3-cx-tools-packagecli |
| cx | `cmd/packageloader2/bolt/` | 1 | delete | `(shared t0-cx-tools-packagestore)` | Byte identical copy of packageloader/bolt |
| cx | `cmd/packageloader2/encoder/` | 2 | delete | `(shared t2-cx-tools-packageencoder)` | Byte identical copy of packageloader/encoder |
| cx | `cmd/packageloader2/file_output/` | 10 | delete | `(shared t5-cx-tools-packagefiles)` | Byte identical copy of packageloader/file_output |
| cx | `cmd/packageloader2/graph/` | 2 | merge | `t2-cx-tools-packagegraph` | Same code as v1, only its newer graph_test.go was kept |
| cx | `cmd/packageloader2/loader/` | 26 | merge | `t1-plloader2` | Newer loader, merge into t1-cx-tools-packageloader once the type checker moves to it |
| cx | `cmd/packageloader2/redis/` | 1 | delete | `(shared t0-plredis)` | Byte identical copy of packageloader/redis |
| cx | `cmd/packageloader2/tree/` | 9 | merge | `t2-pltree2` | Newer tree, merge with t2-cx-tools-packagetree together with the loaders |
| cx | `cmd/packageloader/bolt/` | 1 | keep | `t0-cx-tools-packagestore` | BoltDB store for the package loader |
| cx | `cmd/packageloader/encoder/` | 2 | keep | `t2-cx-tools-packageencoder` | Package encoder |
| cx | `cmd/packageloader/file_output/` | 10 | keep | `t5-cx-tools-packagefiles` | Loads files back from the store |
| cx | `cmd/packageloader/graph/` | 2 | keep | `t2-cx-tools-packagegraph` | Import graph output |
| cx | `cmd/packageloader/loader/` | 17 | keep | `t1-cx-tools-packageloader` | v1 loader, used by the type checker |
| cx | `cmd/packageloader/redis/` | 1 | removed (#114) | `t0-plredis` | Redis store, superseded by bolt or a minimalist key-value store |
| cx | `cmd/packageloader/tree/` | 7 | keep | `t2-cx-tools-packagetree` | Import tree output |
| cx | `cmd/type_checker/` | 24 | keep | `t8-cx-compiler-typechecker` | Work in progress type checker, its tests panic upstream too |

## Tooling

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx | `cmd/cx/` | 5 | keep | `t11-cx-command-cx` | The cx compiler and interpreter binary |
| cx | `cmd/cxmobile/` | 2 | delete | `t7-cx-command-mobile` | Android only (cxfx && android), unbuilt since 2021, cannot be tested here |
| cx | `cmd/cxrepl/` | 1 | keep | `t10-cx-tools-repl` | REPL used by t11-cx-command-cx |
| cx | `cmd/goyacc/` | 187 | delete | `go tool goyacc` | modernc goyacc copy, did not build and did not generate the parsers. golang.org/x/tools goyacc does |
| cx | `cmd/token-fuzzer/` | 1 | keep | `t9-cx-tools-tokenfuzzer` | Lexer fuzzer, small |
| cx | `cxparser/util/cxprof/` | 1 | merge | `t1-cxprof` | Nothing imports it, fold into t1-cx-runtime-profiling or delete |
| cx | `cxparser/util/profiling/` | 1 | keep | `t1-cx-runtime-profiling` | Used by t11-cx-command-cx and the parser |

## Tests

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx | `cmd/cxtest/` | 4 | keep | `t1-cx-tools-programtest, t0-cx-tools-testrunner` | Runner for the tests/ CX programs |
| cx | `tests/` | 230 | keep | `tests` | Stable and issue CX test programs run by cxtest |
| cx | `tests/deprecated/` | 4 | delete | `tests/deprecated` | Excluded from the runner |
| cx | `tests/issue-177/` | 4 | keep | `tests/issue-177` | Multi file issue test |
| cx | `tests/testdata/` | 1 | keep | `tests/testdata` | JSON fixture for the os/json tests |
| cx | `tests/testlib/` | 2 | keep | `tests/testlib` | CX helper libraries used by tests |
| cx | `tests/test-workspace/` | 3 | keep | `tests/test-workspace` | Workspace layout fixture |

## Examples

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx | `tests/examples/` | 176 | keep | `tests/examples` | Canonical example corpus, the playground copy duplicates 150 of these |
| cx-playground | `examples/` | 16 | merge | `skycoin-cx/tests/examples` | 150 of 160 files are byte identical to cx tests/examples, serve that copy instead |
| cx-playground | `examples/affordances/` | 31 | delete | | aff.query and the other query functions were dropped by decision D5 (#500), skycoin-cx removed its copy |
| cx-playground | `examples/blockchain/` | 9 | merge | `skycoin-cx/tests/examples` | 150 of 160 files are byte identical to cx tests/examples, serve that copy instead |
| cx-playground | `examples/book/` | 38 | merge | `skycoin-cx/tests/examples` | 150 of 160 files are byte identical to cx tests/examples, serve that copy instead |
| cx-playground | `examples/buggy-examples/` | 22 | merge | `skycoin-cx/tests/examples` | 150 of 160 files are byte identical to cx tests/examples, serve that copy instead |
| cx-playground | `examples/control-flow/` | 4 | merge | `skycoin-cx/tests/examples` | 150 of 160 files are byte identical to cx tests/examples, serve that copy instead |
| cx-playground | `examples/data-structures/` | 7 | merge | `skycoin-cx/tests/examples` | 150 of 160 files are byte identical to cx tests/examples, serve that copy instead |
| cx-playground | `examples/net/` | 5 | merge | `skycoin-cx/tests/examples` | 150 of 160 files are byte identical to cx tests/examples, serve that copy instead |
| cx-playground | `examples/opengl/` | 29 | merge | `skycoin-cx/tests/examples` | 150 of 160 files are byte identical to cx tests/examples, serve that copy instead |
| cx-playground | `examples/project-example/` | 3 | merge | `skycoin-cx/tests/examples` | 150 of 160 files are byte identical to cx tests/examples, serve that copy instead |
| cx-playground | `examples/serialization/` | 1 | merge | `skycoin-cx/tests/examples` | 150 of 160 files are byte identical to cx tests/examples, serve that copy instead |
| cx-playground | `examples/syntax/` | 1 | merge | `skycoin-cx/tests/examples` | 150 of 160 files are byte identical to cx tests/examples, serve that copy instead |
| cx-playground | `examples/to_be_added_examples/` | 10 | merge | `skycoin-cx/tests/examples` | The 10 files not in cx, review and move into the language repo |

## Benchmarks

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx-benchmark | `benchmarks/` | 4 | merge | `skycoin-cx/t0-cx-tools-benchmarks, t0-cx-tools-benchmarkserialize` | CX struct benchmarks and their Go side (structser) |
| cx-benchmark | `benchmarks/cx-benchmarks/` | 5 | merge | `skycoin-cx/t0-cx-tools-benchmarks/compare/cx` | CX versions of the algorithms |
| cx-benchmark | `benchmarks/golang-benchmarks/` | 9 | merge | `skycoin-cx/t0-cx-tools-benchmarks` | Go versions for comparison, one file each |
| cx-benchmark | `benchmarks/python-benchmarks/` | 6 | delete | - | Python comparison, not part of a Go repo |
| cx-benchmark | `README.md` | 1 | merge | `skycoin-cx/t0-cx-tools-benchmarks/README.md` | Rewritten |

## Docs

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx | `CHANGELOG.md` | 1 | keep | `CHANGELOG.md` | History up to 0.8.0 |
| cx | `CONTRIBUTING.md` | 1 | keep | `CONTRIBUTING.md` | Update the issue link when published |
| cx | `cxparser/` | 1 | merge | `docs/cxparser-readme.md` | Parser readme moved into docs |
| cx | `docs/` | 22 | keep | `docs` | Language and compiler design notes, prune the [WIP] files that went nowhere |
| cx | `docs/images/` | 3 | keep | `docs/images` | Images referenced by docs |
| cx | `LanguageGuide.md` | 1 | keep | `LanguageGuide.md` | The language reference |
| cx | `LICENSE` | 1 | keep | `LICENSE` | License |
| cx | `README.md` | 1 | keep | `README.md` | Updated for the new layout and Go 1.26 |
| cx | `TODO.md` | 1 | merge | - | 2018 era list, move live items to issues and delete |

## Build

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx | `cx-setup.bat` | 1 | delete | `cx-setup.bat` | Windows installer script for the old GOPATH layout |
| cx | `.gitignore` | 1 | keep | `.gitignore` | Plus program_list.db from the loader tests |
| cx | `go.mod` | 1 | keep | `go.mod` | New module github.com/skycoin/skycoin-cx, go 1.26, goyacc as a tool |
| cx | `go.sum` | 1 | keep | `go.sum` | Regenerated |
| cx | `Makefile` | 1 | keep | `Makefile` | Rewritten without vendor, with build-gui, vet and build-parser |
| cx | `vendor/` | 913 | delete | - | go modules fetch dependencies, vendor pinned 2021 versions |
| cx | `.vscode/` | 3 | delete | `.vscode` | Editor settings |
| cx-benchmark | `.gitignore` | 1 | delete | - | Covered by skycoin-cx |
| cx-benchmark | `go.mod` | 1 | delete | - | Covered by the skycoin-cx module |
| cx-benchmark | `go.sum` | 1 | delete | - | Covered by the skycoin-cx module |
| cx-chains | `.gitignore` | 1 | keep | `(own repo)` | Ignore rules |
| cx-chains | `go.mod` | 1 | keep | `(own repo)` | Must import skycoin-cx instead of a vendored cx when revived |
| cx-chains | `go.sum` | 1 | keep | `(own repo)` | Chain module sums |
| cx-chains | `vendor/` | 1288 | delete | - | Vendors a Dec 2020 cx, the reason its tests panic |
| cx-game | `.gitignore` | 1 | keep | `(own repo)` | Ignore rules |
| cx-game | `go.mod` | 1 | keep | `(own repo)` | Module |
| cx-game | `go.mod232` | 1 | delete | - | Stray backup of go.mod |
| cx-game | `go.sum` | 1 | keep | `(own repo)` | Module sums |
| cx-games | `.gitignore` | 1 | keep | `(games repo)` | Ignore rules |
| cx-gui-helper | `.gitignore` | 1 | delete | - | Goes with the repo |
| cx-playground | `.gitignore` | 1 | keep | `skycoin-cx-playground/.gitignore` | Plus /bin |
| cx-playground | `go.mod` | 1 | keep | `skycoin-cx-playground/go.mod` | Requires skycoin-cx, go.work points at the local checkout |
| cx-playground | `go.sum` | 1 | keep | `skycoin-cx-playground/go.sum` | Regenerated |
| cx-tracker | `.gitignore` | 1 | merge | `cx-chains` | Ignore rules |
| cx-tracker | `go.mod` | 1 | merge | `cx-chains` | Tracker module |
| cx-tracker | `go.sum` | 1 | merge | `cx-chains` | Tracker module sums |
| cx-tracker | `.idea/` | 4 | delete | - | Editor settings |
| cx-tracker | `vendor/` | 1265 | delete | - | go modules fetch dependencies |

## CI

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx | `.appveyor.yml` | 1 | delete | `.appveyor.yml` | Dead CI service config |
| cx | `.github/` | 1 | keep | `.github` | PR template |
| cx | `.golangci.yml` | 1 | keep | `.golangci.yml` | Linter config, review against current golangci-lint |
| cx | `.travis.yml` | 1 | delete | `.travis.yml` | Dead CI service config, replace with a GitHub Actions make check |
| cx-chains | `.appveyor.yml` | 1 | delete | - | Dead CI service config |
| cx-chains | `ci-scripts/` | 2 | delete | - | Dead CI helper scripts |
| cx-chains | `ci-scripts/certs/` | 1 | delete | - | Dead CI helper scripts |
| cx-chains | `.github/` | 1 | keep | `(own repo)` | Templates |
| cx-chains | `.golangci.yml` | 1 | keep | `(own repo)` | Linter config |
| cx-chains | `.travis.yml` | 1 | delete | - | Dead CI service config |
| cx-gui-helper | `.travis.yml` | 1 | delete | - | Dead CI service config |
| cx-playground | `.travis.yml` | 1 | delete | - | Pushed a docker image to a personal hub account |
| cx-tracker | `ci_scripts/` | 1 | delete | - | Dead CI helper |

## Playground

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx-playground | `dist/` | 1 | keep | `skycoin-cx-playground/dist` | Web front end |
| cx-playground | `dist/js/` | 1 | keep | `skycoin-cx-playground/dist` | Web front end |
| cx-playground | `dist/style/` | 1 | keep | `skycoin-cx-playground/dist` | Web front end |
| cx-playground | `Dockerfile` | 1 | keep | `skycoin-cx-playground/Dockerfile` | Untested, needs skycoin-cx published or in the build context |
| cx-playground | `main.go` | 1 | keep | `skycoin-cx-playground/main.go` | Server entry, adds the child eval mode |
| cx-playground | `Makefile` | 1 | keep | `skycoin-cx-playground/Makefile` | Rewritten with build, run, test |
| cx-playground | `playground/` | 2 | keep | `skycoin-cx-playground/n002_playground` | Eval, AST and examples handlers, eval now runs in a child process |
| cx-playground | `playground/examples/` | 3 | keep | `skycoin-cx-playground/n002_playground` | Eval, AST and examples handlers, eval now runs in a child process |
| cx-playground | `README.md` | 1 | keep | `skycoin-cx-playground/README.md` | Updated |
| cx-playground | `webapi/` | 3 | keep | `skycoin-cx-playground/n001_webapi` | Program metadata API |

## Chain

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx-chains | `CHANGELOG.md` | 1 | keep | `(own repo)` | Chain history |
| cx-chains | `cmd/cipher-testdata/` | 1 | keep | `(own repo)` | cxchain, cxchain-cli and peer monitor binaries |
| cx-chains | `cmd/cli/` | 4 | keep | `(own repo)` | cxchain, cxchain-cli and peer monitor binaries |
| cx-chains | `cmd/cxchain/` | 1 | keep | `(own repo)` | cxchain, cxchain-cli and peer monitor binaries |
| cx-chains | `cmd/cxchain-cli/` | 15 | keep | `(own repo)` | cxchain, cxchain-cli and peer monitor binaries |
| cx-chains | `cmd/monitor-peers/` | 2 | keep | `(own repo)` | cxchain, cxchain-cli and peer monitor binaries |
| cx-chains | `cx/examples/` | 2 | merge | `tests/examples/blockchain` | Two chain example programs, compare with tests/examples/blockchain |
| cx-chains | `doc/` | 2 | keep | `(own repo)` | Chain docs |
| cx-chains | `INSTALLATION.md` | 1 | keep | `(own repo)` | Chain docs |
| cx-chains | `integration/` | 1 | keep | `(own repo)` | Chain integration test config |
| cx-chains | `Makefile` | 1 | keep | `(own repo)` | Chain build |
| cx-chains | `README.md` | 1 | keep | `(own repo)` | Chain docs |
| cx-chains | `run-cxchain.sh` | 1 | keep | `(own repo)` | Chain launcher |
| cx-chains | `src/api/` | 138 | keep | `(own repo)` | CX chain node built on a 2020 Skycoin fork, not language code |
| cx-chains | `src/cli/` | 59 | keep | `(own repo)` | CX chain node built on a 2020 Skycoin fork, not language code |
| cx-chains | `src/coin/` | 20 | keep | `(own repo)` | CX chain node built on a 2020 Skycoin fork, not language code |
| cx-chains | `src/cx/` | 15 | keep | `(own repo)` | Chain spec, tracker client and dmsg glue, not language code |
| cx-chains | `src/daemon/` | 61 | keep | `(own repo)` | CX chain node built on a 2020 Skycoin fork, not language code |
| cx-chains | `src/kvstorage/` | 11 | keep | `(own repo)` | CX chain node built on a 2020 Skycoin fork, not language code |
| cx-chains | `src/params/` | 7 | keep | `(own repo)` | CX chain node built on a 2020 Skycoin fork, not language code |
| cx-chains | `src/readable/` | 10 | keep | `(own repo)` | CX chain node built on a 2020 Skycoin fork, not language code |
| cx-chains | `src/skycoin/` | 6 | keep | `(own repo)` | CX chain node built on a 2020 Skycoin fork, not language code |
| cx-chains | `src/transaction/` | 8 | keep | `(own repo)` | CX chain node built on a 2020 Skycoin fork, not language code |
| cx-chains | `src/util/` | 3 | keep | `(own repo)` | CX chain node built on a 2020 Skycoin fork, not language code |
| cx-chains | `src/visor/` | 76 | keep | `(own repo)` | CX chain node built on a 2020 Skycoin fork, not language code |
| cx-chains | `src/wallet/` | 27 | keep | `(own repo)` | CX chain node built on a 2020 Skycoin fork, not language code |
| cx-tracker | `cmd/cx-tracker/` | 1 | merge | `cx-chains/cmd` | Tracker binary, belongs next to cxchain |
| cx-tracker | `doc/` | 1 | merge | `cx-chains/doc` | Tracker docs |
| cx-tracker | `doc/assets/` | 1 | merge | `cx-chains/doc` | Tracker docs |
| cx-tracker | `Makefile` | 1 | merge | `cx-chains` | Tracker build |
| cx-tracker | `pkg/api/` | 5 | keep | `(own repo or into cx-chains)` | Chain spec tracker service, pairs with cx-chains not the language |
| cx-tracker | `pkg/store/` | 4 | keep | `(own repo or into cx-chains)` | Chain spec tracker service, pairs with cx-chains not the language |
| cx-tracker | `README.md` | 1 | merge | `cx-chains` | Tracker docs |

## GUI

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx-gui-helper | `ide.glade` | 1 | delete | - | 2019 gotk3 Glade IDE stub, one commit, unrelated to the language |
| cx-gui-helper | `ide_glade.go` | 1 | delete | - | Generated Glade string for the stub |
| cx-gui-helper | `README.md` | 1 | delete | - | gotk3 install notes |

## Games

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx-games | `2DFWork/` | 6 | keep | `(games repo)` | 2D game framework in CX |
| cx-games | `2DFWork/draw/` | 63 | keep | `(games repo)` | 2D game framework in CX |
| cx-games | `2DFWork/entities/` | 2 | keep | `(games repo)` | 2D game framework in CX |
| cx-games | `2DFWork/geom/` | 2 | keep | `(games repo)` | 2D game framework in CX |
| cx-games | `2DFWork/inputs/` | 4 | keep | `(games repo)` | 2D game framework in CX |
| cx-games | `2DFWork/menu/` | 2 | keep | `(games repo)` | 2D game framework in CX |
| cx-games | `2DFWork/mesh/` | 12 | keep | `(games repo)` | 2D game framework in CX |
| cx-games | `aground-cx/aground-game/` | 190 | keep | `(games repo)` | Game and its test assets |
| cx-games | `aground-cx/aground-test/` | 194 | keep | `(games repo)` | Game and its test assets |
| cx-games | `brick-breaker/` | 1 | keep | `(games repo)` | Game |
| cx-games | `brick-breaker/assets/` | 8 | keep | `(games repo)` | Game |
| cx-games | `brick-breaker/internal/` | 7 | keep | `(games repo)` | Game |
| cx-games | `brick-breaker/_pkg/` | 9 | keep | `(games repo)` | Game |
| cx-games | `brick-breaker/run_main/` | 1 | keep | `(games repo)` | Game |
| cx-games | `contest-elim/` | 1 | keep | `(games repo)` | Game |
| cx-games | `contest-elim/cmd/` | 1 | keep | `(games repo)` | Game |
| cx-games | `CXtris/` | 1 | keep | `(games repo)` | Game |
| cx-games | `CXtris/assets/` | 4 | keep | `(games repo)` | Game |
| cx-games | `CXtris/draw/` | 3 | keep | `(games repo)` | Game |
| cx-games | `CXtris/internal_shape/` | 3 | keep | `(games repo)` | Game |
| cx-games | `CXtris/main/` | 5 | keep | `(games repo)` | Game |
| cx-games | `flappy-cat/` | 2 | keep | `(games repo)` | Game |
| cx-games | `flappy-cat/assets/` | 5 | keep | `(games repo)` | Game |
| cx-games | `flappy-cat/cmd/` | 1 | keep | `(games repo)` | Game |
| cx-games | `flappy-cat/internal/` | 5 | keep | `(games repo)` | Game |
| cx-games | `flappy-cat/pkg/` | 9 | keep | `(games repo)` | Game |
| cx-games | `flappy-cat/textures/` | 1 | keep | `(games repo)` | Game |
| cx-games | `libs/graphical3d/` | 11 | keep | `(games repo)` | 3D geometry helpers for GL games |
| cx-games | `pacman/` | 1 | keep | `(games repo)` | Game |
| cx-games | `pacman-3d/` | 2 | keep | `(games repo)` | Game |
| cx-games | `pacman-3d/assets/` | 60 | keep | `(games repo)` | Game |
| cx-games | `pacman-3d/cmd/` | 1 | keep | `(games repo)` | Game |
| cx-games | `pacman-3d/internal/` | 15 | keep | `(games repo)` | Game |
| cx-games | `pacman-3d/pkg/` | 51 | keep | `(games repo)` | Game |
| cx-games | `pacman/assets/` | 1 | keep | `(games repo)` | Game |
| cx-games | `pacman/cmd/` | 4 | keep | `(games repo)` | Game |
| cx-games | `pacman/game_pkg/` | 3 | keep | `(games repo)` | Game |
| cx-games | `pacman/internal/` | 3 | keep | `(games repo)` | Game |
| cx-games | `README.md` | 1 | keep | `(games repo)` | Games index |
| cx-games | `SkyBOOM/` | 1 | keep | `(games repo)` | Game |
| cx-games | `SkyBOOM/assets/` | 11 | keep | `(games repo)` | Game |
| cx-games | `SkyBOOM/cmd/` | 5 | keep | `(games repo)` | Game |
| cx-games | `SkyBOOM/entities/` | 3 | keep | `(games repo)` | Game |
| cx-games | `Sky-Taxi/` | 1 | keep | `(games repo)` | Game |
| cx-games | `Sky-Taxi/assets/` | 4 | keep | `(games repo)` | Game |
| cx-games | `Sky-Taxi/cmd/` | 5 | keep | `(games repo)` | Game |
| cx-games | `snake/` | 1 | keep | `(games repo)` | Game |
| cx-games | `snake/assets/` | 12 | keep | `(games repo)` | Game |
| cx-games | `snake/cmd/` | 1 | keep | `(games repo)` | Game |
| cx-games | `snake/internal/` | 1 | keep | `(games repo)` | Game |

## Game engine

| Repo | Path | Files | Action | Now in | Reason |
|---|---|---:|---|---|---|
| cx-game | `assets/` | 15 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/Boumsee/` | 4 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/cursors/` | 1 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/enemies/` | 1 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/font/` | 2 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/hud/` | 11 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/item/` | 7 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/luis/` | 509 | delete | - | 509 files of one artist pack, check what is referenced and drop the rest |
| cx-game | `assets/particles/` | 3 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/player/` | 3 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/projectile/` | 7 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/shader/` | 47 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/sound/` | 4 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/sprite/` | 251 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/starfield/` | 19 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `assets/tile/` | 3 | keep | `(own repo)` | Art, shaders and sound for the Go game |
| cx-game | `cmd/dynamicBatchShaderExample/` | 36 | keep | `(own repo)` | Shader and sprite tool examples |
| cx-game | `cmd/shaderexample/` | 4 | keep | `(own repo)` | Shader and sprite tool examples |
| cx-game | `cmd/spritetool/` | 1 | keep | `(own repo)` | Shader and sprite tool examples |
| cx-game | `components/` | 3 | keep | `(own repo)` | ECS components |
| cx-game | `components/agents/` | 37 | keep | `(own repo)` | ECS components |
| cx-game | `components/datastructs/` | 1 | keep | `(own repo)` | ECS components |
| cx-game | `components/particles/` | 29 | keep | `(own repo)` | ECS components |
| cx-game | `components/types/` | 1 | keep | `(own repo)` | ECS components |
| cx-game | `constants/` | 12 | keep | `(own repo)` | Game constants |
| cx-game | `cxmath/` | 15 | keep | `(own repo)` | Math helpers, pure Go despite the name |
| cx-game | `cxmath/math32/` | 1 | keep | `(own repo)` | Math helpers, pure Go despite the name |
| cx-game | `cxmath/math32i/` | 1 | keep | `(own repo)` | Math helpers, pure Go despite the name |
| cx-game | `cxmath/mathi/` | 1 | keep | `(own repo)` | Math helpers, pure Go despite the name |
| cx-game | `cxmath/spineMath/` | 4 | keep | `(own repo)` | Math helpers, pure Go despite the name |
| cx-game | `docs/` | 4 | keep | `(own repo)` | Game docs |
| cx-game | `effects/` | 2 | keep | `(own repo)` | Effects |
| cx-game | `engine/` | 2 | keep | `(own repo)` | Rendering, input, sound and UI |
| cx-game | `engine/bone/` | 1 | keep | `(own repo)` | Rendering, input, sound and UI |
| cx-game | `engine/camera/` | 3 | keep | `(own repo)` | Rendering, input, sound and UI |
| cx-game | `engine/cursors/` | 1 | keep | `(own repo)` | Rendering, input, sound and UI |
| cx-game | `engine/input/` | 4 | keep | `(own repo)` | Rendering, input, sound and UI |
| cx-game | `engine/sound/` | 2 | keep | `(own repo)` | Rendering, input, sound and UI |
| cx-game | `engine/spine/` | 20 | keep | `(own repo)` | Rendering, input, sound and UI |
| cx-game | `engine/spineold/` | 12 | delete | - | Superseded by engine/spine |
| cx-game | `engine/spriteloader/` | 15 | keep | `(own repo)` | Rendering, input, sound and UI |
| cx-game | `engine/ui/` | 18 | keep | `(own repo)` | Rendering, input, sound and UI |
| cx-game | `game/` | 10 | keep | `(own repo)` | Game loop |
| cx-game | `.github/images/` | 1 | keep | `(own repo)` | README images |
| cx-game | `item/` | 15 | keep | `(own repo)` | Items |
| cx-game | `lib/openal/` | 21 | keep | `(own repo)` | OpenAL bindings |
| cx-game | `main.go` | 1 | keep | `(own repo)` | Entry point |
| cx-game | `particles/` | 6 | keep | `(own repo)` | Particles |
| cx-game | `physics/` | 5 | keep | `(own repo)` | Physics |
| cx-game | `physics/timer/` | 1 | keep | `(own repo)` | Physics |
| cx-game | `physics/verlet/` | 1 | keep | `(own repo)` | Physics |
| cx-game | `procgen/` | 1 | keep | `(own repo)` | Procedural generation |
| cx-game | `procgen/mapgen/` | 1 | keep | `(own repo)` | Procedural generation |
| cx-game | `procgen/perlin/` | 2 | keep | `(own repo)` | Procedural generation |
| cx-game | `README.md` | 1 | keep | `(own repo)` | Docs |
| cx-game | `render/` | 19 | keep | `(own repo)` | Renderer |
| cx-game | `render/worldctx/` | 1 | keep | `(own repo)` | Renderer |
| cx-game | `starfield/` | 4 | keep | `(own repo)` | Starfield |
| cx-game | `stats/` | 2 | keep | `(own repo)` | Stats |
| cx-game | `test/` | 11 | keep | `(own repo)` | Game tests and benchmarks |
| cx-game | `test/benchmarks/` | 8 | keep | `(own repo)` | Game tests and benchmarks |
| cx-game | `test/sound/` | 4 | keep | `(own repo)` | Game tests and benchmarks |
| cx-game | `test/spine-animation/` | 340 | delete | - | 340 file spine sample set, move to a release asset if needed |
| cx-game | `world/` | 22 | keep | `(own repo)` | World model |
| cx-game | `world/import/` | 7 | keep | `(own repo)` | World model |
| cx-game | `world/pipesim/` | 2 | keep | `(own repo)` | World model |
| cx-game | `world/tiling/` | 8 | keep | `(own repo)` | World model |
| cx-game | `world/worldcollider/` | 1 | keep | `(own repo)` | World model |
| cx-game | `world/worldgen/` | 1 | keep | `(own repo)` | World model |
