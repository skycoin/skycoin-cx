package llvm

import (
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// OpcodeFact is what the LLVM target knows of a scalar image opcode, taken
// from its row of constants.OpcodeDefinitionTable by t2-cx-core-opcodegen.
// OpcodeFactOperation is the part of the name after the dot, the target
// type of a conversion. OpcodeFactResult is types.UNDEFINED for no output.
type OpcodeFact struct {
	OpcodeFactName      string
	OpcodeFactOperation string
	OpcodeFactType      types.Code
	OpcodeFactInputs    uint8
	OpcodeFactResult    types.Code
}

// OpcodeFactOf returns the fact of op, with an empty name for an opcode that
// is not scalar.
func OpcodeFactOf(op ir.Opcode) OpcodeFact {
	if op >= ir.OpcodeCount {
		return OpcodeFact{}
	}
	return opcodeFacts[op]
}
