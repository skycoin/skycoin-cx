package netopcodes

import (
	"fmt"
	"net"
	"strconv"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t1-cx-library-networkbatch"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// netUpdate is net.Update. A replay that runs out of updates ends the
// program, since the program then differs from the recorded one.
func netUpdate(waitMilliseconds int32) int32 {
	count := networkbatch.NetUpdate(waitMilliseconds)
	if count < 0 {
		netReplayEndAssertFailure()
	}
	return count
}

// netReplayEndAssertFailure ends a program that asks for an update the
// replay record does not hold.
// STRHOT REPORTER: builds the assert text of a replay past its record, #412.
func netReplayEndAssertFailure() {
	assert.PreconditionTrue(false, fmt.Sprintf("the net replay record has no update %d, or one of its entries names a socket the program did not open", networkbatch.NetUpdateNumber()))
}

// netBufferView returns the bytes of the ui8 array value, in place in
// program memory.
func netBufferView(prgrm *ast.CXProgram, value *ast.CXValue) []byte {
	signature := value.TypeSignature
	if signature.Type != ast.TypeArrayAtomic || signature.TypeGet(prgrm) != valuetypes.UI8 {
		netBufferAssertFailure()
	}
	return prgrm.Memory[value.Offset : value.Offset+signature.ArraySizeGet(prgrm)]
}

// netBufferAssertFailure ends a program that passes a buffer that is not a
// ui8 array.
// STRHOT REPORTER: holds the assert text of a buffer that is not a ui8 array, #412.
func netBufferAssertFailure() {
	assert.PreconditionTrue(false, "a net buffer must be a ui8 array, such as var buffer [1024]ui8")
}

// netBufferLength clamps a length the program passes to the buffer.
func netBufferLength(buffer []byte, length int32) []byte {
	return buffer[:max(0, min(int(length), len(buffer)))]
}

func OpNetUpdate(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, netUpdate(inputs[0].I32Get(prgrm)))
}

func OpNetReadyCount(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, networkbatch.NetReadyCount())
}

func OpNetReadyNext(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	handle, trigger, messageID, detail := networkbatch.NetReadyNext()
	outputs[0].I32Set(prgrm, handle)
	outputs[1].I32Set(prgrm, int32(trigger))
	outputs[2].I32Set(prgrm, messageID)
	outputs[3].I32Set(prgrm, detail)
}

func OpNetClock(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, networkbatch.NetClock())
}

func OpNetRead(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	buffer := netBufferView(prgrm, &inputs[2])
	outputs[0].I32Set(prgrm, networkbatch.NetRead(inputs[0].I32Get(prgrm), inputs[1].I32Get(prgrm), buffer, inputs[3].I32Get(prgrm)))
}

func OpNetWrite(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	buffer := netBufferLength(netBufferView(prgrm, &inputs[1]), inputs[2].I32Get(prgrm))
	outputs[0].I32Set(prgrm, networkbatch.NetWrite(inputs[0].I32Get(prgrm), buffer))
}

func OpNetClose(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].BoolSet(prgrm, networkbatch.NetClose(inputs[0].I32Get(prgrm)))
}

func OpNetTimeoutSet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].BoolSet(prgrm, networkbatch.NetTimeoutSet(inputs[0].I32Get(prgrm), inputs[1].I32Get(prgrm)))
}

// VIOLATION STRHOT-329: B, hands an address to Go and the OS, #508.
func OpNetTCPListen(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, networkbatch.NetTCPListen(netAddress(inputs[0].StrGet(prgrm), inputs[1].I32Get(prgrm))))
}

// VIOLATION STRHOT-330: B, hands an address to Go and the OS, #508.
func OpNetTCPDial(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, networkbatch.NetTCPDial(netAddress(inputs[0].StrGet(prgrm), inputs[1].I32Get(prgrm))))
}

// VIOLATION STRHOT-331: B, hands an address to Go and the OS, #508.
func OpNetUDPListen(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, networkbatch.NetUDPListen(netAddress(inputs[0].StrGet(prgrm), inputs[1].I32Get(prgrm))))
}

// VIOLATION STRHOT-332: B, hands an address to Go and the OS, #508.
func OpNetUDPDial(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, networkbatch.NetUDPDial(netAddress(inputs[0].StrGet(prgrm), inputs[1].I32Get(prgrm))))
}

func OpNetUDPReadFrom(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	buffer := netBufferView(prgrm, &inputs[2])
	length, peer := networkbatch.NetUDPReadFrom(inputs[0].I32Get(prgrm), inputs[1].I32Get(prgrm), buffer, inputs[3].I32Get(prgrm))
	outputs[0].I32Set(prgrm, length)
	outputs[1].I32Set(prgrm, peer)
}

func OpNetUDPWriteTo(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	buffer := netBufferLength(netBufferView(prgrm, &inputs[2]), inputs[3].I32Get(prgrm))
	outputs[0].I32Set(prgrm, networkbatch.NetUDPWriteTo(inputs[0].I32Get(prgrm), inputs[1].I32Get(prgrm), buffer))
}

// VIOLATION STRHOT-333: B, hands an address to Go, #508.
func OpNetUDPPeerResolve(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, networkbatch.NetUDPPeerResolve(netAddress(inputs[0].StrGet(prgrm), inputs[1].I32Get(prgrm))))
}

// netAddress joins host and port as Go's net package takes them.
// VIOLATION STRHOT-337: B, builds the address it hands to Go and the OS, #508.
func netAddress(host string, port int32) string {
	return net.JoinHostPort(host, strconv.Itoa(int(port)))
}
