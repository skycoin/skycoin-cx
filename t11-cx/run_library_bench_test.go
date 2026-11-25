package main

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-execute"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
)

// runLibraryCallLoopSource makes 400,000 system library calls with scalar
// operands, in three libraries.
const runLibraryCallLoopSource = `package main

import "sys/os"
import "sys/net"
import "sys/cipher"

func main() {
	var i i32
	var closed bool
	var state i32
	for i = 0; i < 100000; i++ {
		closed = os.Close(-1)
		closed = net.Close(-1)
		state = cipher.HashStateNew()
		cipher.HashStateRelease(state)
	}
}
`

// runLibraryCallLoopCompile compiles runLibraryCallLoopSource with its
// libraries enabled.
func runLibraryCallLoopCompile(tb testing.TB) {
	compileRunProgram(tb, runBenchProgram{name: "library-loop", source: runLibraryCallLoopSource})
	for _, library := range []int{constants.LibraryNumberFile, constants.LibraryNumberNet, constants.LibraryNumberCipher} {
		actions.AST.ProgramLibrariesEnabled[library] = true
	}
}

// BenchmarkRunLibraryCall times a loop of system library calls on the old
// loop and on vm (decision D96).
func BenchmarkRunLibraryCall(b *testing.B) {
	initProgram()
	b.Run("old", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			runLibraryCallLoopCompile(b)
			execute.RunPrepare(actions.AST)
			b.StartTimer()
			if err := execute.RunCompiled(actions.AST, 0, nil); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("vm", func(b *testing.B) {
		runLibraryCallLoopCompile(b)
		image, _, outside := core.ExecutorLower(actions.AST)
		if outside != "" {
			b.Fatalf("does not lower: %s", outside)
		}
		machine := vm.MachineNew(&image, core.ExecutorOutputSize)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if code := core.ExecutorVmRunMachine(actions.AST, machine); code != types.CX_SUCCESS {
				b.Fatalf("code %d", code)
			}
		}
	})
}
