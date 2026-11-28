package metaopcodes

import (
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// PackageRegister adds the meta package to prgrm. The functions of sys/meta
// are the opcodes of opcodetable.OpcodeGroupMeta, which MetaOpcodeDispatch
// runs on the old loop and the vm runs itself (decision D151).
func PackageRegister(prgrm *ast.CXProgram) {
	metaPkg := ast.PackageMake(prgrm.CXProgramNameIntern("meta"))
	prgrm.PackageAdd(metaPkg)
}
