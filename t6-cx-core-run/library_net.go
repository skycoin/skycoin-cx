//go:build cxcorelibrarynet

package main

import (
	"os"

	"github.com/skycoin/skycoin-cx/t1-netbatch"
	"github.com/skycoin/skycoin-cx/t5-cxnet"
)

// coreNetReplayEnv names the replay record a built program runs its net
// library from, since a built program takes no cx flags. See
// docs/Networking.md.
const coreNetReplayEnv = "CX_NET_REPLAY"

// The net library is linked into this core, see coreLibraryRegisters.
func init() {
	coreLibraryRegisters = append(coreLibraryRegisters, cxnet.RegisterPackage)
	if path := os.Getenv(coreNetReplayEnv); path != "" {
		cxnet.LibraryNetMode, cxnet.LibraryNetRecordPath = netbatch.NetModeReplay, path
	}
}
