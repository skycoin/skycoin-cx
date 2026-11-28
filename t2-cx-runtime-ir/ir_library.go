package ir

import (
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
)

// opcodeLibraryOperations is the language opcode of each library opcode
// that has no range of its own, from OpcodeLibraryFirst on.
var opcodeLibraryOperations = [...]int{
	OpcodeCipherHashStateNew - OpcodeLibraryFirst:     opcodetable.OpcodeCipherHashStateNew,
	OpcodeCipherHashStateRelease - OpcodeLibraryFirst: opcodetable.OpcodeCipherHashStateRelease,
	OpcodeNetUpdate - OpcodeLibraryFirst:              opcodetable.OpcodeNetUpdate,
	OpcodeNetReadyCount - OpcodeLibraryFirst:          opcodetable.OpcodeNetReadyCount,
	OpcodeNetReadyNext - OpcodeLibraryFirst:           opcodetable.OpcodeNetReadyNext,
	OpcodeNetClock - OpcodeLibraryFirst:               opcodetable.OpcodeNetClock,
	OpcodeNetRead - OpcodeLibraryFirst:                opcodetable.OpcodeNetRead,
	OpcodeNetWrite - OpcodeLibraryFirst:               opcodetable.OpcodeNetWrite,
	OpcodeNetClose - OpcodeLibraryFirst:               opcodetable.OpcodeNetClose,
	OpcodeNetTimeoutSet - OpcodeLibraryFirst:          opcodetable.OpcodeNetTimeoutSet,
	OpcodeNetTCPListen - OpcodeLibraryFirst:           opcodetable.OpcodeNetTCPListen,
	OpcodeNetTCPDial - OpcodeLibraryFirst:             opcodetable.OpcodeNetTCPDial,
	OpcodeNetUDPListen - OpcodeLibraryFirst:           opcodetable.OpcodeNetUDPListen,
	OpcodeNetUDPDial - OpcodeLibraryFirst:             opcodetable.OpcodeNetUDPDial,
	OpcodeNetUDPReadFrom - OpcodeLibraryFirst:         opcodetable.OpcodeNetUDPReadFrom,
	OpcodeNetUDPWriteTo - OpcodeLibraryFirst:          opcodetable.OpcodeNetUDPWriteTo,
	OpcodeNetUDPPeerResolve - OpcodeLibraryFirst:      opcodetable.OpcodeNetUDPPeerResolve,
	OpcodeFileClose - OpcodeLibraryFirst:              opcodetable.OpcodeFileClose,
	OpcodeFileSeek - OpcodeLibraryFirst:               opcodetable.OpcodeFileSeek,
	OpcodeFileSync - OpcodeLibraryFirst:               opcodetable.OpcodeFileSync,
	OpcodeTerminalArgCount - OpcodeLibraryFirst:       opcodetable.OpcodeTerminalArgCount,
	OpcodeTerminalArgRead - OpcodeLibraryFirst:        opcodetable.OpcodeTerminalArgRead,
	OpcodeTerminalInputLineRead - OpcodeLibraryFirst:  opcodetable.OpcodeTerminalInputLineRead,
}

// opcodeLibraryRanges are the libraries whose image opcodes follow the
// order of their language opcodes.
var opcodeLibraryRanges = [...]struct {
	opcodeLibraryRangeFirst     Opcode
	opcodeLibraryRangeLast      Opcode
	opcodeLibraryRangeOperation int
}{
	{OpcodeReflectFirst, OpcodeReflectLast, opcodetable.OpcodeReflectCxTypeOfValueGet},
	{OpcodeSerializeFirst, OpcodeSerializeLast, opcodetable.OpcodeSerializeCxSerializeEncodedSizeInBytesGet},
	{OpcodeMetaFirst, OpcodeMetaLast, opcodetable.OpcodeMetaCxStructDefinitionBegin},
	{OpcodeCxirFirst, OpcodeCxirLast, opcodetable.OpcodeCxirCxProgramImageCurrentGet},
}

// OpcodeLibraryOperation returns the language opcode of the library opcode
// op, the number ImageOperationsEnabled holds, 0 for any other opcode.
func OpcodeLibraryOperation(op Opcode) int {
	for i := range opcodeLibraryRanges {
		libraryRange := &opcodeLibraryRanges[i]
		if op >= libraryRange.opcodeLibraryRangeFirst && op <= libraryRange.opcodeLibraryRangeLast {
			return libraryRange.opcodeLibraryRangeOperation + int(op-libraryRange.opcodeLibraryRangeFirst)
		}
	}
	if op >= OpcodeLibraryFirst && int(op-OpcodeLibraryFirst) < len(opcodeLibraryOperations) {
		return opcodeLibraryOperations[op-OpcodeLibraryFirst]
	}
	return 0
}

// OpcodeOfLibraryOperation returns the image opcode of the language opcode
// of a system library function that runs on vm, OpcodeInvalid for any other
// opcode (decision D119).
func OpcodeOfLibraryOperation(operation int) Opcode {
	if operation <= 0 {
		return OpcodeInvalid
	}
	for op := OpcodeLibraryFirst; op <= OpcodeLibraryLast; op++ {
		if OpcodeLibraryOperation(op) == operation {
			return op
		}
	}
	return OpcodeInvalid
}

// ImageCxirEnabled reports whether image enables an operation of sys/cxir.
// It reads no opcode table, so cx-core-image links none.
func ImageCxirEnabled(image *Image) bool {
	for op := OpcodeCxirFirst; op <= OpcodeCxirLast; op++ {
		if image.ImageOperationsEnabled.OpcodeSetContains(OpcodeLibraryOperation(op)) {
			return true
		}
	}
	return false
}
