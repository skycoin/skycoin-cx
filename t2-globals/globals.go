package globals

import (
	"github.com/skycoin/skycoin-cx/t0-types" // thin: only Pointer; no depth gain
	"os"
)

// Var
var (
	// VIOLATION: state outside State. Owner: t2-globals; moves to CXProgramMng in phase 2.
	HeapOffset types.Pointer
)

// Path is only used by os module and only to get working directory
// Path and working directory should not be hard coded into program struct (etc, when serialized)
// Working directory is property of executable and can be retrieved with golang library
var CxProgramPath string = ""

// VIOLATION: init chaining. Init 1: CXPATH, BINPATH, PKGPATH from the env,
// called from Go package init. t11-cx GetCXPath sets them again.
// VIOLATION: state outside State. Owner: t2-globals; moves to ConfigMng in phase 2.
var CXPATH = os.Getenv("CXPATH") + "/"

// VIOLATION: state outside State. Owner: t2-globals; moves to ConfigMng in phase 2.
var BINPATH = CXPATH + "bin/" // TODO @evanlinjin: Not used.
// VIOLATION: state outside State. Owner: t2-globals; moves to ConfigMng in phase 2.
var PKGPATH = CXPATH + "pkg/" // TODO @evanlinjin: Not used.
// VIOLATION: state outside State. Owner: t2-globals; moves to ConfigMng in phase 2.
var SRCPATH = CXPATH + "src/"
