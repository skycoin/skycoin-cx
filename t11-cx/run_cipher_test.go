package main

import (
	"fmt"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-cxhash"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t3-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-execute"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
	"github.com/skycoin/skycoin/src/cipher"
)

// runCipherCallCount is the number of library calls each cipher program makes.
const runCipherCallCount = 100

// The key, signature and digest of tests/test-cipher-library.cx, a signature
// made by cipher.SignHash of Skycoin over the digest of "abc".
const (
	runCipherPubKeyHex    = "02f382589c67fbac7065b5eff66f35c30bc73e385ffeab46326ec1a930b10d1c53"
	runCipherSignatureHex = "b70e75dd813b465a5e7047b63f442dd96f48f58ceff7b3dd27425e79686398070ec2a9d61c512d103d52ebc4937e1e726a657adefd82734a0ae11f310465eed801"
)

// runCipherBytesLiteral returns a CX []ui8 literal of data.
func runCipherBytesLiteral(data []byte) string {
	elements := make([]string, len(data))
	for i, element := range data {
		elements[i] = fmt.Sprintf("%dUB", element)
	}
	return "[]ui8{" + strings.Join(elements, ", ") + "}"
}

// runCipherHashSource hashes size bytes runCipherCallCount times.
func runCipherHashSource(size int) string {
	return fmt.Sprintf(`package main

import "sys/cipher"
import "sys/string"

func main() {
	var data []ui8 = string.BytesOfString("%s")
	var digest []ui8
	var i i32
	for i = 0; i < %d; i++ {
		digest = cipher.HashOfBytes(data)
	}
	test(len(digest) > 0, true, "a digest")
}
`, strings.Repeat("a", size), runCipherCallCount)
}

// runCipherHashStateSource writes size bytes to one state runCipherCallCount
// times and sums it once.
func runCipherHashStateSource(size int) string {
	return fmt.Sprintf(`package main

import "sys/cipher"
import "sys/string"

func main() {
	var data []ui8 = string.BytesOfString("%s")
	var i i32
	var state i32 = cipher.HashStateNew()
	for i = 0; i < %d; i++ {
		cipher.HashStateWrite(state, data)
	}
	var digest []ui8 = cipher.HashStateSum(state)
	cipher.HashStateRelease(state)
	test(len(digest) > 0, true, "a digest")
}
`, strings.Repeat("a", size), runCipherCallCount)
}

// runCipherVerifySource verifies the Skycoin signature runCipherCallCount
// times.
func runCipherVerifySource(tb testing.TB) string {
	pubKey := cipher.MustPubKeyFromHex(runCipherPubKeyHex)
	signature := cipher.MustSigFromHex(runCipherSignatureHex)
	return fmt.Sprintf(`package main

import "sys/cipher"
import "sys/string"

func main() {
	var pubKey []ui8 = %s
	var signature []ui8 = %s
	var digest []ui8 = cipher.HashOfBytes(string.BytesOfString("abc"))
	var valid bool
	var i i32
	for i = 0; i < %d; i++ {
		valid = cipher.SignatureVerify(digest, signature, pubKey)
	}
	test(valid, true, "the signature verifies")
}
`, runCipherBytesLiteral(pubKey[:]), runCipherBytesLiteral(signature[:]), runCipherCallCount)
}

// runCipherPrograms are the programs of TestRunCipherAllocs and
// BenchmarkRunCipher. LoopWithoutCalls is the loop of HashOfBytes-64 with
// the call left out, the time per call that is not the library.
func runCipherPrograms(tb testing.TB) []runBenchProgram {
	return []runBenchProgram{
		{"LoopWithoutCalls", strings.Replace(runCipherHashSource(64), "digest = cipher.HashOfBytes(data)", "digest = data", 1), 0},
		{"HashOfBytes-64", runCipherHashSource(64), 0},
		{"HashOfBytes-1024", runCipherHashSource(1024), 0},
		{"HashStateWrite-1024", runCipherHashStateSource(1024), 0},
		{"SignatureVerify", runCipherVerifySource(tb), 0},
	}
}

// runCipherCompile compiles p with the cipher library enabled and prepares
// it to run.
func runCipherCompile(tb testing.TB, p runBenchProgram) *ast.CXProgram {
	compileRunProgram(tb, p)
	actions.AST.ProgramLibrariesEnabled[constants.LibraryNumberCipher] = true
	execute.RunPrepare(actions.AST)
	return actions.AST
}

// runCipherAllocs returns the allocations of one run of p, each run on its
// own compile as in TestRunAllocs.
func runCipherAllocs(t *testing.T, p runBenchProgram) float64 {
	const runs = 3
	var compiled []*ast.CXProgram
	for i := 0; i <= runs; i++ {
		compiled = append(compiled, runCipherCompile(t, p))
	}
	next := 0
	return testing.AllocsPerRun(runs, func() {
		if err := execute.RunCompiled(compiled[next], 0, nil); err != nil {
			t.Fatal(err)
		}
		next++
	})
}

// TestRunCipherAllocs fails when a hash of the cipher library allocates Go
// memory, or a verify allocates more than the Skycoin call it makes (#407,
// #510). The byte slices are views of program memory and the digests go to
// program memory.
func TestRunCipherAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	discardStdout(t)

	pubKey := cipher.MustPubKeyFromHex(runCipherPubKeyHex)
	signature := cipher.MustSigFromHex(runCipherSignatureHex)
	digest := cipher.SHA256(cxhash.CxHashOfBytes([]byte("abc")))
	skycoinVerifyAllocs := testing.AllocsPerRun(10, func() {
		if cipher.VerifyPubKeySignedHash(pubKey, signature, digest) != nil {
			t.Fatal("the Skycoin signature does not verify")
		}
	})

	for _, p := range runCipherPrograms(t) {
		allocs := runCipherAllocs(t, p)
		want := 0.0
		if p.name == "SignatureVerify" {
			// VMALLOC-024: secp256k1-go2 of Skycoin computes with math/big.
			want = skycoinVerifyAllocs * runCipherCallCount
		}
		t.Logf("%s: %.0f allocations per run of %d calls, Skycoin verify %.0f per call", p.name, allocs, runCipherCallCount, skycoinVerifyAllocs)
		if allocs > want {
			t.Errorf("%s: %.0f allocations per run of %d calls, want at most %.0f", p.name, allocs, runCipherCallCount, want)
		}
	}
}

// BenchmarkRunCipher times the cipher library calls in a run of the old
// loop. ns/call and allocs/call divide by the calls of a run, and include
// the loop that makes them.
func BenchmarkRunCipher(b *testing.B) {
	initProgram()
	for _, p := range runCipherPrograms(b) {
		b.Run(p.name, func(b *testing.B) {
			discardStdout(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				prgrm := runCipherCompile(b, p)
				b.StartTimer()
				if err := execute.RunCompiled(prgrm, 0, nil); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*runCipherCallCount), "ns/call")
		})
	}
}

const runCipherHashStateVmSource = `package main

import "sys/cipher"

func main() {
	var state i32
	var i i32
	for i = 0; i < 1000; i++ {
		state = cipher.HashStateNew()
		cipher.HashStateRelease(state)
	}
}
`

// TestRunVmCipherAllocs fails when the cipher functions with a vm form do not
// lower or allocate on vm (#407, #510).
func TestRunVmCipherAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	compileRunProgram(t, runBenchProgram{name: "cipher", source: runCipherHashStateVmSource})
	actions.AST.ProgramLibrariesEnabled[constants.LibraryNumberCipher] = true
	image, _, outside := core.ExecutorLower(actions.AST)
	if outside != "" {
		t.Fatalf("does not lower: %s", outside)
	}
	machine := vm.MachineNew(&image, core.ExecutorOutputSize)
	allocs := testing.AllocsPerRun(3, func() {
		machine.Reset()
		if stop, _ := machine.Run(image.ImageEntryMain, 0); stop != vm.StopReturn {
			t.Fatalf("stop %d", stop)
		}
	})
	if allocs != 0 {
		t.Errorf("%.0f allocations per run, want 0", allocs)
	}
}
