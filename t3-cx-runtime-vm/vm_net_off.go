//go:build cxcore && !cxcorelibrarynet

package vm

import (
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// netOpcodeRun does nothing in a core built without cxcorelibrarynet. The
// core refuses a program that uses the net library before it runs, so no
// net opcode reaches the vm (decision D119).
func netOpcodeRun(op ir.Opcode, memory []byte, offset uint32) {}
