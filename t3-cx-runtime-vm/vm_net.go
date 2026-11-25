//go:build !cxcore || cxcorelibrarynet

package vm

import (
	"fmt"
	"net"
	"strconv"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-netbatch"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// The opcodes of the net library on vm, decision D119. A core built
// without cxcorelibrarynet leaves this file out, so it links no socket code,
// see vm_net_off.go.

// netBlock reads the inputs and writes the outputs of a net opcode in
// order, as the lowering packs them in the block at sp.
type netBlock struct {
	blockMemory []byte
	blockOffset uint32
}

func (block *netBlock) blockI32() int32 {
	value := types.Read_i32(block.blockMemory, types.Pointer(block.blockOffset))
	block.blockOffset += 4
	return value
}

func (block *netBlock) blockI32Write(value int32) {
	types.Write_i32(block.blockMemory, types.Pointer(block.blockOffset), value)
	block.blockOffset += 4
}

func (block *netBlock) blockBoolWrite(value bool) {
	types.Write_bool(block.blockMemory, types.Pointer(block.blockOffset), value)
	block.blockOffset++
}

// blockBuffer returns the ui8 array at the address and size the lowering
// wrote, empty when it lies outside memory.
func (block *netBlock) blockBuffer() []byte {
	address := uint64(types.Read_ptr(block.blockMemory, types.Pointer(block.blockOffset)))
	size := uint64(types.Read_ptr(block.blockMemory, types.Pointer(block.blockOffset+machinePointerSize)))
	block.blockOffset += 2 * machinePointerSize
	if address+size > uint64(len(block.blockMemory)) {
		return nil
	}
	return block.blockMemory[address : address+size]
}

// blockAddress returns the host of a str input and the i32 port after it,
// joined as Go's net package takes them.
func (block *netBlock) blockAddress() string {
	text, _ := LibraryStrView(block.blockMemory, block.blockOffset)
	block.blockOffset += machinePointerSize
	return net.JoinHostPort(string(text), strconv.Itoa(int(block.blockI32())))
}

// netBufferLength clamps a length the program passes to the buffer.
func netBufferLength(buffer []byte, length int32) []byte {
	return buffer[:max(0, min(int(length), len(buffer)))]
}

// netReplayEndAssertFailure ends a program that asks for an update the
// replay record does not hold.
func netReplayEndAssertFailure() {
	assert.AssertPreconditionTrue(false, fmt.Sprintf("the net replay record has no update %d, or one of its entries names a socket the program did not open", netbatch.NetUpdateNumber()))
}

// netOpcodeRun runs the net opcode op on the block at offset.
// VIOLATION VMALLOC-018: a system library call allocates in Go and the OS, #407.
func netOpcodeRun(op ir.Opcode, memory []byte, offset uint32) {
	block := netBlock{memory, offset}
	switch op {
	case ir.OpcodeNetUpdate:
		count := netbatch.NetUpdate(block.blockI32())
		if count < 0 {
			netReplayEndAssertFailure()
		}
		block.blockI32Write(count)
	case ir.OpcodeNetReadyCount:
		block.blockI32Write(netbatch.NetReadyCount())
	case ir.OpcodeNetReadyNext:
		handle, trigger, messageID, detail := netbatch.NetReadyNext()
		block.blockI32Write(handle)
		block.blockI32Write(int32(trigger))
		block.blockI32Write(messageID)
		block.blockI32Write(detail)
	case ir.OpcodeNetClock:
		types.Write_i64(memory, types.Pointer(offset), netbatch.NetClock())
	case ir.OpcodeNetRead:
		handle, messageID, buffer, payloadOffset := block.blockI32(), block.blockI32(), block.blockBuffer(), block.blockI32()
		block.blockI32Write(netbatch.NetRead(handle, messageID, buffer, payloadOffset))
	case ir.OpcodeNetWrite:
		handle, buffer := block.blockI32(), block.blockBuffer()
		buffer = netBufferLength(buffer, block.blockI32())
		block.blockI32Write(netbatch.NetWrite(handle, buffer))
	case ir.OpcodeNetClose:
		block.blockBoolWrite(netbatch.NetClose(block.blockI32()))
	case ir.OpcodeNetTimeoutSet:
		handle, milliseconds := block.blockI32(), block.blockI32()
		block.blockBoolWrite(netbatch.NetTimeoutSet(handle, milliseconds))
	case ir.OpcodeNetTCPListen:
		block.blockI32Write(netbatch.NetTCPListen(block.blockAddress()))
	case ir.OpcodeNetTCPDial:
		block.blockI32Write(netbatch.NetTCPDial(block.blockAddress()))
	case ir.OpcodeNetUDPListen:
		block.blockI32Write(netbatch.NetUDPListen(block.blockAddress()))
	case ir.OpcodeNetUDPDial:
		block.blockI32Write(netbatch.NetUDPDial(block.blockAddress()))
	case ir.OpcodeNetUDPReadFrom:
		handle, messageID, buffer, payloadOffset := block.blockI32(), block.blockI32(), block.blockBuffer(), block.blockI32()
		length, peer := netbatch.NetUDPReadFrom(handle, messageID, buffer, payloadOffset)
		block.blockI32Write(length)
		block.blockI32Write(peer)
	case ir.OpcodeNetUDPWriteTo:
		handle, peer, buffer := block.blockI32(), block.blockI32(), block.blockBuffer()
		buffer = netBufferLength(buffer, block.blockI32())
		block.blockI32Write(netbatch.NetUDPWriteTo(handle, peer, buffer))
	case ir.OpcodeNetUDPPeerResolve:
		block.blockI32Write(netbatch.NetUDPPeerResolve(block.blockAddress()))
	}
}
