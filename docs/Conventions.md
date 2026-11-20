# Conventions

Every rule of this repository in one list. Each item links to its detailed
document or check script.

1. **Tiers.** Every Go package sits one level below the root as
   `t<N>-<name>`, where N is 1 plus the highest tier it imports. `make
   check-tiers` runs [check-tiers.sh](../check-tiers.sh). See the layout
   section of the [README](../README.md#repository-layout).
2. **Explicit Init.** No `func init()` outside package main. Setup is an
   exported Init function that `t11-cx` main calls in order. `make
   check-inits` runs [check-inits.sh](../check-inits.sh) against
   [init-allowlist.txt](../init-allowlist.txt), which only shrinks.
3. **No interface embedding.** An interface lists its methods and never
   embeds another interface. `make check-ifaces` runs
   [check-ifaces.sh](../check-ifaces.sh) against
   [iface-allowlist.txt](../iface-allowlist.txt).
4. **No recover.** No code outside `vendor/` calls `recover`. `make
   check-recovers` runs [check-recovers.sh](../check-recovers.sh) against
   [recover-allowlist.txt](../recover-allowlist.txt).
5. **Two classes of error.** Path, file and network failures return their
   module's error struct, and every other failure is a crashing assert from
   `t0-cx-core-assert`. See [ErrorHandling.md](ErrorHandling.md).
6. **Naming.** Verbose, descriptive names, see
   [skycoin-cx#212](https://github.com/skycoin/skycoin-cx/issues/212) until
   it has its own document.
7. **Opcode numbers are fixed.** Every opcode has a fixed number in
   `constants.OpcodeLifecycleTable` in
   [t1-constants/opcodes.go](../t1-constants/opcodes.go), mirrored in
   [opcodes.csv](opcodes.csv). A new opcode takes the next number after the
   highest. A deleted opcode keeps its row as deprecated, gets no handler, and
   its number is never reused before the row's reassignable version.
8. **Opcode lifecycle.** Each row records the version the number was added,
   deprecated and becomes reassignable, 0 for the last two while live and 0
   added for opcodes older than versioning. Reassignable is the deprecated
   version plus one year (`OpcodeReassignmentIntervalYears`), same month and
   day, release 0001, so deprecated 2611200001 is reassignable from
   2711200001, and 29 February moves to 1 March. Every event is listed in
   order in [t1-constants/opcodes_history.go](../t1-constants/opcodes_history.go).
   `go test ./t1-constants` fails when the table, the CSV and the history
   disagree.
9. **Language version.** The integer `yymmddnnnn`, the release date and the
   release number that day, for example 2611200001 is 2026-11-20 release
   0001. `constants.LanguageVersionCurrent` is stored in every serialized
   program and `Deserialize` rejects any other version. See the Language
   versions section of [LanguageGuide.md](../LanguageGuide.md#language-versions).
10. **Branching.** `main` is the only long-lived branch. One branch and one
    pull request per change, merged into `main` after the `check` jobs pass.
    See [CONTRIBUTING.md](../CONTRIBUTING.md#git-branching-model).
11. **Changelog.** Every user visible change adds a line under `Unreleased`
    in [CHANGELOG.md](../CHANGELOG.md) with its issue or pull request number.
