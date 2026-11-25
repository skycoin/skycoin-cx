//go:build cxcorelibrarynet

package main

import (
	"os"

	"github.com/skycoin/skycoin-cx/t1-netbatch"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-cxnet"
)

// coreNetReplayEnv names the replay record a built program runs its net
// library from, since a built program takes no cx flags. See
// docs/Networking.md.
const coreNetReplayEnv = "CX_NET_REPLAY"

// coreLibraryNetLinked says that the net library is linked into this core.
const coreLibraryNetLinked = true

// The net library is linked into this core, see coreLibraryRegisters.
func init() {
	coreLibraryRegisters = append(coreLibraryRegisters, cxnet.RegisterPackage)
	if path := os.Getenv(coreNetReplayEnv); path != "" {
		netbatch.NetRunMode, netbatch.NetRunRecordPath = netbatch.NetModeReplay, path
	}
}

// coreOpcodeDispatchNet runs an opcode of the net library.
func coreOpcodeDispatchNet(prgrm *ast.CXProgram, code int, inputs []ast.CXValue, outputs []ast.CXValue) bool {
	return cxnet.NetOpcodeDispatch(prgrm, code, inputs, outputs)
}
