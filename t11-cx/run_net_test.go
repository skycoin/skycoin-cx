package main

import (
	"io"
	"net"
	"runtime/debug"
	"strconv"
	"testing"
	"time"

	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t1-netbatch"
	"github.com/skycoin/skycoin-cx/t3-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-execute"
	core "github.com/skycoin/skycoin-cx/t6-cx-runtime-core"
)

// runNetSource updates, reads every data entry and writes one byte back, on
// the sockets the test opened before the run.
const runNetSource = `package main

import "sys/net"

var buffer [64]ui8

func main() { global buffer
	var count i32
	var handle i32
	var trigger i32
	var message i32
	var detail i32
	var length i32
	var queued i32
	var clock i64
	var i i32
	var update i32
	for update = 0; update < 10; update++ {
		count = net.Update(0)
		for i = 0; i < count; i++ {
			handle, trigger, message, detail = net.ReadyNext()
			if trigger == net.TriggerData {
				length = net.Read(handle, message, buffer, 0)
				queued = net.Write(handle, buffer, 1)
			}
		}
		count = net.ReadyCount()
		clock = net.Clock()
	}
}
`

// runNetPeers starts a live run with count connections from the program
// side to plain Go peers on loopback and returns the peers.
func runNetPeers(t *testing.T, count int) []net.Conn {
	t.Helper()
	if err := netbatch.NetRunStart(netbatch.NetModeLive, ""); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(netbatch.NetRunStop)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	address := listener.Addr().(*net.TCPAddr)
	for i := 0; i < count; i++ {
		netbatch.NetTCPDial(net.JoinHostPort("127.0.0.1", strconv.Itoa(address.Port)))
	}
	var peers []net.Conn
	for len(peers) < count {
		peer, err := listener.Accept()
		if err != nil {
			t.Fatal(err)
		}
		peers = append(peers, peer)
	}
	connected := 0
	deadline := time.Now().Add(5 * time.Second)
	for connected < count && time.Now().Before(deadline) {
		netbatch.NetUpdate(20)
		for {
			_, trigger, _, _ := netbatch.NetReadyNext()
			if trigger == netbatch.NetTriggerNone {
				break
			}
			if trigger == netbatch.NetTriggerConnected {
				connected++
			}
		}
	}
	if connected != count {
		t.Fatalf("%d of %d connected", connected, count)
	}
	return peers
}

// runNetSend makes every peer send a few bytes and waits until they wait for
// the next update.
func runNetSend(peers []net.Conn, payload []byte) {
	for _, peer := range peers {
		peer.Write(payload)
	}
	deadline := time.Now().Add(5 * time.Second)
	for netbatch.NetArrivalsPending() < len(peers) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Microsecond)
	}
}

// runNetEchoed fails unless every peer received the bytes the program wrote
// back.
func runNetEchoed(t *testing.T, peers []net.Conn) {
	t.Helper()
	for _, peer := range peers {
		peer.SetReadDeadline(time.Now().Add(5 * time.Second))
		var echoed [1]byte
		if _, err := io.ReadFull(peer, echoed[:]); err != nil {
			t.Fatalf("the program wrote nothing back: %v", err)
		}
	}
}

// TestRunNetAllocs fails when an update, a ready read, a read or a write of
// the net library allocates Go memory on the old loop (#407, #508).
func TestRunNetAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	peers := runNetPeers(t, 4)
	const runs = 3
	var compiled []*ast.CXProgram
	for i := 0; i <= runs; i++ {
		compileRunProgram(t, runBenchProgram{name: "net", source: runNetSource})
		actions.AST.ProgramLibrariesEnabled[constants.LibraryNumberNet] = true
		execute.RunPrepare(actions.AST)
		compiled = append(compiled, actions.AST)
	}
	payload := []byte("ab")
	next := 0
	allocs := testing.AllocsPerRun(runs, func() {
		runNetSend(peers, payload)
		if err := execute.RunCompiled(compiled[next], 0, nil); err != nil {
			t.Fatal(err)
		}
		next++
	})
	runNetEchoed(t, peers)
	if allocs != 0 {
		t.Errorf("%.0f allocations per run, want 0", allocs)
	}
}

// TestRunVmNetAllocs is TestRunNetAllocs on vm, which must lower the program.
func TestRunVmNetAllocs(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	initProgram()
	peers := runNetPeers(t, 4)
	compileRunProgram(t, runBenchProgram{name: "net", source: runNetSource})
	actions.AST.ProgramLibrariesEnabled[constants.LibraryNumberNet] = true
	image, _, outside := core.ExecutorLower(actions.AST)
	if outside != "" {
		t.Fatalf("does not lower: %s", outside)
	}
	machine := vm.MachineNew(&image, core.ExecutorOutputSize)
	payload := []byte("ab")
	allocs := testing.AllocsPerRun(3, func() {
		runNetSend(peers, payload)
		machine.Reset()
		if stop, _ := machine.Run(image.ImageEntryMain, 0); stop != vm.StopReturn {
			t.Fatalf("stop %d", stop)
		}
	})
	runNetEchoed(t, peers)
	if allocs != 0 {
		t.Errorf("%.0f allocations per run, want 0", allocs)
	}
}
