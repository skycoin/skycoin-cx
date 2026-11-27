package ast

import (
	"strconv"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
)

// CXProgramLookupErrorCode names the program table or the lookup that
// failed, see the code ranges of docs/ErrorHandling.md.
type CXProgramLookupErrorCode uint32

const (
	CXProgramLookupErrorCodePackageIndex CXProgramLookupErrorCode = 8001 + iota
	CXProgramLookupErrorCodeStructIndex
	CXProgramLookupErrorCodeLineIndex
	CXProgramLookupErrorCodeArgumentIndex
	CXProgramLookupErrorCodeAtomicOperatorIndex
	CXProgramLookupErrorCodeTypeSignatureIndex
	CXProgramLookupErrorCodeTypeSignatureArrayIndex
	CXProgramLookupErrorCodeTypeSignatureStructIndex
	CXProgramLookupErrorCodeOperationTypeMissing
	CXProgramLookupErrorCodeLineMissing
)

// CXProgramLookupError is a failed lookup in a table of the program as
// data. Error builds the text only when it is printed, after the run (#412).
type CXProgramLookupError struct {
	CXProgramLookupErrorCode  CXProgramLookupErrorCode
	CXProgramLookupErrorIndex int
}

func (lookupError CXProgramLookupError) Error() string {
	index := strconv.Itoa(lookupError.CXProgramLookupErrorIndex)
	switch lookupError.CXProgramLookupErrorCode {
	case CXProgramLookupErrorCodePackageIndex:
		return "error: CXPackages[" + index + "]: index out of bounds"
	case CXProgramLookupErrorCodeStructIndex:
		return "error: CXStructs[" + index + "]: index out of bounds"
	case CXProgramLookupErrorCodeLineIndex:
		return "error: CXLines[" + index + "]: index out of bounds"
	case CXProgramLookupErrorCodeArgumentIndex:
		return "error: CXArgs[" + index + "]: index out of bounds"
	case CXProgramLookupErrorCodeAtomicOperatorIndex:
		return "error: CXAtomicOps[" + index + "]: index out of bounds"
	case CXProgramLookupErrorCodeTypeSignatureIndex:
		return "error: CXTypeSignature[" + index + "]: index out of bounds"
	case CXProgramLookupErrorCodeTypeSignatureArrayIndex:
		return "error: TypeSignatureForArrays[" + index + "]: index out of bounds"
	case CXProgramLookupErrorCodeTypeSignatureStructIndex:
		return "error: TypeSignatureForStructs[" + index + "]: index out of bounds"
	case CXProgramLookupErrorCodeOperationTypeMissing:
		return "operation type is not found."
	case CXProgramLookupErrorCodeLineMissing:
		return "CXLine not found."
	}
	assert.Unreachable("CXProgramLookupError with code " + strconv.FormatUint(uint64(lookupError.CXProgramLookupErrorCode), 10))
	return ""
}
