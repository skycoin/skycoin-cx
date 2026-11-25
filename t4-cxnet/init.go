package cxnet

import (
	"github.com/skycoin/skycoin-cx/t1-netbatch"
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// RegisterPackage adds the net package to prgrm. The functions of sys/net
// are the opcodes of constants.OpcodeGroupNet, which NetOpcodeDispatch runs
// on the old loop and the vm runs itself, see docs/Networking.md.
func RegisterPackage(prgrm *ast.CXProgram) {
	netbatch.NetPrepare()

	// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
	// Now a constructor call, site S1427.
	netPkg := ast.MakePackage(prgrm.CXProgramNameIntern("net"))
	// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
	// Now an arena insert, site S1428.
	prgrm.AddPackage(netPkg)
}

// The constant codes of the net.Trigger* and net.Error* constants.
const (
	constNetTriggerFirst = iota + 0x2FFFF
	constNetErrorFirst   = constNetTriggerFirst + int(netbatch.NetTriggerCount)
)

// InitConstants registers the net.Trigger* and net.Error* constants through
// constantAdd, actions.AddConstI32 in cx, which keeps the compiler out of
// the cx build core.
func InitConstants(constantAdd func(code int, name string, value int32)) {
	triggers := [netbatch.NetTriggerCount]string{"None", "Data", "Accepted", "Connected", "Listening", "Closed", "Error", "Timeout", "Dropped"}
	for value, name := range triggers {
		constantAdd(constNetTriggerFirst+value, "net.Trigger"+name, int32(value))
	}
	errors := [netbatch.NetErrorCount]string{"None", "Other", "Refused", "Reset", "Timeout", "Address", "AddressInUse", "SocketTableFull"}
	for value, name := range errors {
		constantAdd(constNetErrorFirst+value, "net.Error"+name, int32(value))
	}
}
