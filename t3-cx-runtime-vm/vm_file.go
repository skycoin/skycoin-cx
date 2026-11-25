//go:build !nocxos && (!cxcore || cxcorelibraryfile)

package vm

import (
	"github.com/skycoin/skycoin-cx/t0-cxfiles"
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// fileOpcodeRun runs the file opcode op on the block at offset, decision
// D119. A build with nocxos or a core without cxcorelibraryfile leaves this
// file out, see vm_file_off.go.
func fileOpcodeRun(op ir.Opcode, memory []byte, offset uint32) {
	switch op {
	case ir.OpcodeFileClose:
		types.Write_bool(memory, types.Pointer(offset+4), cxfiles.FileHandleClose(types.Read_i32(memory, types.Pointer(offset))))
	case ir.OpcodeFileSeek:
		position := cxfiles.FileHandleSeek(types.Read_i32(memory, types.Pointer(offset)), types.Read_i64(memory, types.Pointer(offset+4)), types.Read_i32(memory, types.Pointer(offset+12)))
		types.Write_i64(memory, types.Pointer(offset+16), position)
	case ir.OpcodeFileSync:
		types.Write_bool(memory, types.Pointer(offset+4), cxfiles.FileHandleSync(types.Read_i32(memory, types.Pointer(offset))))
	}
}
