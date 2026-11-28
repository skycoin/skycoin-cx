package serializeopcodes

import (
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// PackageRegister adds the serialize package to prgrm. The functions of
// sys/serialize are the opcodes of opcodetable.OpcodeGroupSerialize, which
// SerializeOpcodeDispatch runs on the old loop and the vm runs itself
// (decision D150).
func PackageRegister(prgrm *ast.CXProgram) {
	serializePkg := ast.PackageMake(prgrm.CXProgramNameIntern("serialize"))
	prgrm.PackageAdd(serializePkg)
}
