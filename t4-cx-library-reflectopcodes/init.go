package reflectopcodes

import (
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// PackageRegister adds the reflect package to prgrm. The functions of
// sys/reflect are the opcodes of constants.OpcodeGroupReflect, which
// ReflectOpcodeDispatch runs on the old loop and the vm runs itself
// (decision D150).
func PackageRegister(prgrm *ast.CXProgram) {
	reflectPkg := ast.PackageMake(prgrm.CXProgramNameIntern("reflect"))
	prgrm.PackageAdd(reflectPkg)
}
