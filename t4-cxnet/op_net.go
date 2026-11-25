package cxnet

import (
	"fmt"
	"net"
	"strconv"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-netbatch"
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// netUpdate is net.Update. A replay that runs out of updates ends the
// program, since the program then differs from the recorded one.
func netUpdate(waitMilliseconds int32) int32 {
	count := netbatch.NetUpdate(waitMilliseconds)
	if count < 0 {
		netReplayEndAssertFailure()
	}
	return count
}

// netReplayEndAssertFailure ends a program that asks for an update the
// replay record does not hold.
// VIOLATION STRHOT-328: E, the assert text of a replay past its record, built only when it fails, #508.
func netReplayEndAssertFailure() {
	assert.AssertPreconditionTrue(false, fmt.Sprintf("the net replay record has no update %d, or one of its entries names a socket the program did not open", netbatch.NetUpdateNumber()))
}

// netBufferView returns the bytes of the ui8 array value, in place in
// program memory.
func netBufferView(prgrm *ast.CXProgram, value *ast.CXValue) []byte {
	signature := value.TypeSignature
	if signature.Type != ast.TYPE_ARRAY_ATOMIC || signature.GetType(prgrm) != types.UI8 {
		netBufferAssertFailure()
	}
	return prgrm.Memory[value.Offset : value.Offset+signature.GetArraySize(prgrm)]
}

// netBufferAssertFailure ends a program that passes a buffer that is not a
// ui8 array.
// VIOLATION STRHOT-336: E, the assert text of a buffer that is not a ui8 array, built only when it fails, #508.
func netBufferAssertFailure() {
	assert.AssertPreconditionTrue(false, "a net buffer must be a ui8 array, such as var buffer [1024]ui8")
}

// netBufferLength clamps a length the program passes to the buffer.
func netBufferLength(buffer []byte, length int32) []byte {
	return buffer[:max(0, min(int(length), len(buffer)))]
}

func OpNetUpdate(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].Set_i32(prgrm, netUpdate(inputs[0].Get_i32(prgrm)))
}

func OpNetReadyCount(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].Set_i32(prgrm, netbatch.NetReadyCount())
}

func OpNetReadyNext(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	handle, trigger, messageID, detail := netbatch.NetReadyNext()
	outputs[0].Set_i32(prgrm, handle)
	outputs[1].Set_i32(prgrm, int32(trigger))
	outputs[2].Set_i32(prgrm, messageID)
	outputs[3].Set_i32(prgrm, detail)
}

func OpNetClock(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].Set_i64(prgrm, netbatch.NetClock())
}

func OpNetRead(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	buffer := netBufferView(prgrm, &inputs[2])
	outputs[0].Set_i32(prgrm, netbatch.NetRead(inputs[0].Get_i32(prgrm), inputs[1].Get_i32(prgrm), buffer, inputs[3].Get_i32(prgrm)))
}

func OpNetWrite(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	buffer := netBufferLength(netBufferView(prgrm, &inputs[1]), inputs[2].Get_i32(prgrm))
	outputs[0].Set_i32(prgrm, netbatch.NetWrite(inputs[0].Get_i32(prgrm), buffer))
}

func OpNetClose(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].Set_bool(prgrm, netbatch.NetClose(inputs[0].Get_i32(prgrm)))
}

func OpNetTimeoutSet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].Set_bool(prgrm, netbatch.NetTimeoutSet(inputs[0].Get_i32(prgrm), inputs[1].Get_i32(prgrm)))
}

// VIOLATION STRHOT-329: B, hands an address to Go and the OS, #508.
func OpNetTCPListen(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].Set_i32(prgrm, netbatch.NetTCPListen(netAddress(inputs[0].Get_str(prgrm), inputs[1].Get_i32(prgrm))))
}

// VIOLATION STRHOT-330: B, hands an address to Go and the OS, #508.
func OpNetTCPDial(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].Set_i32(prgrm, netbatch.NetTCPDial(netAddress(inputs[0].Get_str(prgrm), inputs[1].Get_i32(prgrm))))
}

// VIOLATION STRHOT-331: B, hands an address to Go and the OS, #508.
func OpNetUDPListen(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].Set_i32(prgrm, netbatch.NetUDPListen(netAddress(inputs[0].Get_str(prgrm), inputs[1].Get_i32(prgrm))))
}

// VIOLATION STRHOT-332: B, hands an address to Go and the OS, #508.
func OpNetUDPDial(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].Set_i32(prgrm, netbatch.NetUDPDial(netAddress(inputs[0].Get_str(prgrm), inputs[1].Get_i32(prgrm))))
}

func OpNetUDPReadFrom(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	buffer := netBufferView(prgrm, &inputs[2])
	length, peer := netbatch.NetUDPReadFrom(inputs[0].Get_i32(prgrm), inputs[1].Get_i32(prgrm), buffer, inputs[3].Get_i32(prgrm))
	outputs[0].Set_i32(prgrm, length)
	outputs[1].Set_i32(prgrm, peer)
}

func OpNetUDPWriteTo(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	buffer := netBufferLength(netBufferView(prgrm, &inputs[2]), inputs[3].Get_i32(prgrm))
	outputs[0].Set_i32(prgrm, netbatch.NetUDPWriteTo(inputs[0].Get_i32(prgrm), inputs[1].Get_i32(prgrm), buffer))
}

// VIOLATION STRHOT-333: B, hands an address to Go, #508.
func OpNetUDPPeerResolve(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].Set_i32(prgrm, netbatch.NetUDPPeerResolve(netAddress(inputs[0].Get_str(prgrm), inputs[1].Get_i32(prgrm))))
}

// netAddress joins host and port as Go's net package takes them.
// VIOLATION STRHOT-337: B, builds the address it hands to Go and the OS, #508.
func netAddress(host string, port int32) string {
	return net.JoinHostPort(host, strconv.Itoa(int(port)))
}
