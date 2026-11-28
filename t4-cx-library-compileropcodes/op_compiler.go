package compileropcodes

import (
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// PackageRegister adds the compiler package to prgrm. The functions of
// sys/compiler are the opcodes of opcodetable.OpcodeGroupCompiler, which
// CompilerOpcodeDispatch runs on the old loop and VmOpcodeRun on vm
// (decision D152).
func PackageRegister(prgrm *ast.CXProgram) {
	compilerPkg := ast.PackageMake(prgrm.CXProgramNameIntern("compiler"))
	prgrm.PackageAdd(compilerPkg)
}

// valueBytes returns the memory of a buffer input, nil when it lies outside
// memory.
func valueBytes(prgrm *ast.CXProgram, value *ast.CXValue) []byte {
	end := uint64(value.Offset) + uint64(value.Size)
	if end > uint64(len(prgrm.Memory)) {
		return nil
	}
	return prgrm.Memory[value.Offset:end]
}

func OpCxCompilerSourceFileAppend(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, Library.SourceFileAppend(Library.LibraryImageStore, inputs[0].I64Get(prgrm), valueBytes(prgrm, &inputs[1]), inputs[2].I32Get(prgrm)))
}

func OpCxCompilerSourceParse(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, Library.SourceParse(Library.LibraryImageStore))
}

func OpCxCompilerProgramTypeCheck(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, Library.ProgramTypeCheck(inputs[0].I64Get(prgrm)))
}

func OpCxCompilerProgramLower(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, Library.ProgramLower(Library.LibraryImageStore, inputs[0].I64Get(prgrm)))
}

func OpCxCompilerProgramCompile(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, Library.ProgramCompile(Library.LibraryImageStore))
}

func OpCxCompilerErrorCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, Library.ErrorCountGet())
}

func OpCxCompilerErrorMessageRead(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, Library.ErrorMessageRead(inputs[0].I32Get(prgrm), valueBytes(prgrm, &inputs[1])))
}

func OpCxCompilerErrorSourceLineGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, Library.ErrorSourceLineGet(inputs[0].I32Get(prgrm)))
}

func OpCxAstPackageCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, Library.AstPackageCountGet(inputs[0].I64Get(prgrm)))
}

func OpCxAstPackageNameGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, Library.AstPackageNameGet(inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxAstPackageFunctionCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, Library.AstPackageFunctionCountGet(inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxAstPackageFunctionGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, Library.AstPackageFunctionGet(Library.LibraryTypeTable, inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm), inputs[2].I32Get(prgrm)))
}

func OpCxAstPackageStructCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, Library.AstPackageStructCountGet(inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxAstPackageStructGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, Library.AstPackageStructGet(Library.LibraryTypeTable, inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm), inputs[2].I32Get(prgrm)))
}

func OpCxAstPackageGlobalCountGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, Library.AstPackageGlobalCountGet(inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm)))
}

func OpCxAstPackageGlobalNameGet(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I64Set(prgrm, Library.AstPackageGlobalNameGet(inputs[0].I64Get(prgrm), inputs[1].I32Get(prgrm), inputs[2].I32Get(prgrm)))
}

func OpCxCompilerAstRelease(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	outputs[0].I32Set(prgrm, Library.AstRelease(inputs[0].I64Get(prgrm)))
}
