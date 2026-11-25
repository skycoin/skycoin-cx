//go:build !nocipher

package cipher

import (
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// RegisterPackage adds the cipher package to prgrm and makes the hash
// states. The functions of sys/cipher are the opcodes of
// constants.OpcodeGroupCipher, which CipherOpcodeDispatch runs, see
// docs/CipherLibrary.md.
func RegisterPackage(prgrm *ast.CXProgram) {
	cipherHashTablePrepare()

	// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
	// Now a constructor call, site S1359.
	cipherPkg := ast.MakePackage(prgrm.CXProgramNameIntern("cipher"))
	// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
	// Now an arena insert, site S1362.
	prgrm.AddPackage(cipherPkg)
}
