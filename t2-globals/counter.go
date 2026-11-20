package globals

import (
	"github.com/skycoin/skycoin-cx/t1-constants" // thin: only END_OF_NAMED_OPCODES; core, keep
)

var (
	// VIOLATION: state outside State. Owner: t2-globals; moves to CXNativeFunctionMng in phase 1.
	OpCodeSystemCounter int = constants.END_OF_NAMED_OPCODES
)
