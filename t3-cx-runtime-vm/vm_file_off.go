//go:build nocxos || (cxcore && !cxcorelibraryfile)

package vm

import (
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// fileOpcodeRun does nothing in a build without the file library, which
// refuses a program that uses it before it runs (decision D119).
func fileOpcodeRun(op ir.Opcode, memory []byte, offset uint32) {}
