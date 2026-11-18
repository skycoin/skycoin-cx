# Benchmarks

Imported from github.com/skycoin/cx-benchmark at a511da9ec17c.

`cx/` holds the CX programs. The Go files here hold the same algorithms in Go
for comparison. Run `make` here, or `make cx` and `make go`.

`bm-structs.cx` and `bm-struct-serialization.cx` use pre 2022 syntax and fail
to parse with the current compiler, the same as with skycoin/cx develop.
`../t0-benchmarks-structser` is the Go side of the struct serialization
benchmark.
