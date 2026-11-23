# AGENTS.md

Rules for anyone changing this repository, people and coding agents alike.
The full list is [docs/Conventions.md](docs/Conventions.md). `make check`
enforces the tier, init, interface and recover rules.

## Naming

Every name describes itself. Names are hierarchical and lexicographically
prefixed, and they use Java type name style (UpperCamelCase).

- The most general part comes first, and each later part narrows it:
  `Struct`, `StructDefine`, `StructDefineFieldAdd`.
- Related names share a prefix, so they sort next to each other:
  `RegisterClear`, `RegisterFocusLeft`, `RegisterTypeSet`.
- A name that does not say what it is is invalid. That includes single
  letters, `Tmp`, `Data`, `Info`, `Handler`, `Do`, `Get`, and the method
  names `Error`, `Unwrap`, `Code`, `Is` and `As`.
- Booleans take no `is` or `has` prefix.
- Abbreviations are allowed only where they are the usual name, such as `ID`,
  `UDP` or `JSON`.

`make names-csv` lists every declared Go and CX name with its file, line and
kind, for review. Add `REPOS="../skycoin-cx-forth"` to scan more repositories.

## Changes

- `make check` passes before every push.
- One branch and one pull request per change, merged into `main`.
- No coding tool attribution in commits or pull requests.
