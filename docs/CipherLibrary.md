# The cipher library

`sys/cipher` gives CX programs a hash and secp256k1 signatures. It is the
system library `cipher`, number 4, so a program imports it and the run
enables it.

```
cx --enable=cipher program.cx
```

See [skycoin-cx#510](https://github.com/skycoin/skycoin-cx/issues/510),
the System libraries section of [LanguageGuide.md](../LanguageGuide.md#system-libraries)
and rule 16 of [Conventions.md](Conventions.md).

## Functions

Every function is called through `LibraryCall` (795) with library 4 and its
function number. The numbers are fixed for life in
`constants.LibraryFunctionLifecycleTable`.

| Number | Function | Does |
| --- | --- | --- |
| 1 | `cipher.HashOfBytes(data []ui8) (digest []ui8)` | the 32 byte digest of `data` |
| 2 | `cipher.HashStateNew() (state i32)` | a new incremental hash |
| 3 | `cipher.HashStateWrite(state i32, data []ui8)` | adds `data` to the state |
| 4 | `cipher.HashStateSum(state i32) (digest []ui8)` | the digest of what was written. The state keeps it and takes more writes |
| 5 | `cipher.HashStateRelease(state i32)` | frees the state |
| 6 | `cipher.SignatureVerify(hash []ui8, signature []ui8, pubKey []ui8) (valid bool)` | whether `signature` is a signature of `hash` by `pubKey` |
| 7 | `cipher.SignatureRecoverPubKey(hash []ui8, signature []ui8) (pubKey []ui8, recovered bool)` | the public key that made `signature` |
| 8 | `cipher.SignatureSign(hash []ui8, secKey []ui8) (signature []ui8, signed bool)` | the signature of `hash` by `secKey` |
| 9 | `cipher.KeyPairGenerate() (pubKey []ui8, secKey []ui8)` | a new random key pair |

A hash is 32 bytes, a public key 33 (compressed), a secret key 32 and a
signature 65, 64 bytes of signature and the recovery byte, as Skycoin
stores them. An input of another length does not verify, recover or sign,
and the function returns false and the nil slice. `SignatureSign` also
returns false for a secret key that is not valid and a hash of zeros.

```cx
package main

import "sys/cipher"
import "sys/string"

func main() {
	var pubKey []ui8
	var secKey []ui8
	pubKey, secKey = cipher.KeyPairGenerate()
	var digest []ui8 = cipher.HashOfBytes(string.BytesOfString("Hello World!"))
	var signature []ui8
	var signed bool
	signature, signed = cipher.SignatureSign(digest, secKey)
	printf("%v %v\n", signed, cipher.SignatureVerify(digest, signature, pubKey)) // true true
}
```

## The hash

The hash is BLAKE3 with a 256 bit output, BLAKE3-256 (decision D23). It
replaced SHA-256, and a digest stays 32 bytes. It is one function,
`cxhash.CxHashOfBytes` in `t0-cxhash`, with `cxhash.CxHashNew` for the
incremental form and `cxhash.CxHashName`. The execution records hash
through the same functions, so a change of hash changes the cipher library
and the records at once. The implementation is `lukechampine.com/blake3`,
pure Go with assembly for amd64 and arm64, and the published BLAKE3 vectors
are checked in `t0-cxhash/cxhash_test.go` and `tests/test-cipher-library.cx`.
Signatures still sign a 32 byte digest, which is now a BLAKE3-256 digest.

An incremental state is an `i32` handle into a fixed table of 64 states,
`CipherHashStateCountMax` in `t5-cipher`. The table is made before the run
and a state is reset when it is taken again, so a run allocates nothing for
it. A handle out of range, released or never returned, and a 65th state
held at once, are asserts. The table belongs to the process, as the cxfx
tables do (decision D45).

## Signatures

Signatures are secp256k1 through the `cipher` package of Skycoin (decision
D22), so a signature Skycoin makes verifies here and the other way round.
`SignatureVerify` is `cipher.VerifyPubKeySignedHash` of Skycoin, which
recovers the key from the signature, compares it with `pubKey` and then
checks the signature. `SignatureRecoverPubKey` is `cipher.PubKeyFromSig`,
`SignatureSign` is `cipher.SignHash`, which picks a random nonce, and
`KeyPairGenerate` is `cipher.GenerateKeyPair`. The hash a program signs is
the 32 bytes it passes, whatever hash made them.

## Allocations

The byte slices are read as views of program memory and copied into fixed
arrays on the Go stack, and every digest, key and signature a function
returns is a new `[]ui8` in program memory. Program memory is not Go memory
(see [RunPathAllocations.md](RunPathAllocations.md)), so the hash functions
make no Go allocation. `TestRunCipherAllocs` in `t11-cx` requires 0 per run
of 100 calls.

`SignatureVerify`, `SignatureRecoverPubKey`, `SignatureSign` and
`KeyPairGenerate` allocate inside Skycoin, whose `secp256k1-go2` computes
with `math/big`, VMALLOC-024. A verify makes 108 allocations there, and
`TestRunCipherAllocs` requires that the library adds none. Removing them
means fixed size field arithmetic in Skycoin's secp256k1, see decision D76.

Measured on an Intel i7-3770S. The run is `BenchmarkRunCipher` in
`t11-cx`, a run of the old loop divided by its 100 calls, which includes
about 2000 ns per call of run start and loop, its `LoopWithoutCalls`. The
Go call is `BenchmarkCxHashOfBytes` in `t0-cxhash` and
`BenchmarkCipherSignatureVerify` in `t5-cipher`.

| Call | Run, ns/call | Go call, ns/op | Go allocations per call |
| --- | --- | --- | --- |
| `HashOfBytes`, 64 bytes | 3800 | 980 | 0 |
| `HashOfBytes`, 1024 bytes | 14400 | 7400 | 0 |
| `HashStateWrite`, 1024 bytes | 14900 | | 0 |
| `SignatureVerify` | 585000 | 486000 | 108 |

## The vm

The vm has no heap. A slice lives in the heap, the lowering rejects a slice
operand (`ReasonOperandHeap`, `ReasonOperandNotScalar`), and a vm library
function reads only the scalars of its block. So `HashStateNew` and
`HashStateRelease`, which take and return only an `i32`, have vm forms, and
the other seven run on the old loop only. A program that calls one of them
does not lower and runs on the old loop under `--executor=auto`. The vm
runs them once it has a heap or can pass a slice as a view of the heap of
the old loop.

## cx build

A core links the cipher library only when the program calls it and
`--enable` enables it, so `make build` builds sixteen cores, eight of them
with cipher, `cx-core-cipher`, `cx-core-file-cipher` and so on. A core with
cipher links the `cipher` package of Skycoin, the one module outside the Go
standard library a core links, see decision D75.

## Before 2611240003

`cipher.GenerateKeyPair`, opcode 134, returned the structs `cipher.PubKey`
and `cipher.SecKey`. It is deprecated, reassignable from `2711240001`, and
the structs are gone. `cipher.KeyPairGenerate` returns the keys as `[]ui8`.
