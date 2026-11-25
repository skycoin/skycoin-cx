//go:build !cxcore || cxcorelibrarynet

package core

import (
	"github.com/skycoin/skycoin-cx/t1-netbatch"
)

// executorNetRunStart closes what an earlier run left open and starts the
// net library in netbatch.NetRunMode.
func executorNetRunStart() error {
	return netbatch.NetRunStart(netbatch.NetRunMode, netbatch.NetRunRecordPath)
}
