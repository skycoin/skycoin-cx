package cxiropcodes

import (
	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// structField is one field of a struct sys/cxir declares.
type structField struct {
	structFieldName string
	structFieldType valuetypes.Code
}

// structDeclarations are the structs of sys/cxir, with the fields and
// sizes of ir.Instr, ir.Slot, ir.Local and ir.Line, packed as CX packs a
// struct, so a program fills them directly (decision D153).
var structDeclarations = [...]struct {
	structDeclarationName   string
	structDeclarationFields []structField
}{
	{"CxInstruction", []structField{
		{"Opcode", valuetypes.UI16}, {"Mode", valuetypes.UI16}, {"A", valuetypes.UI32}, {"B", valuetypes.UI32}, {"C", valuetypes.UI32},
		{"D", valuetypes.UI32}, {"N", valuetypes.UI32}, {"M", valuetypes.UI32}, {"Aux", valuetypes.UI32},
	}},
	{"CxFrameSlot", []structField{{"Offset", valuetypes.UI32}, {"Size", valuetypes.UI32}}},
	{"CxFrameLocal", []structField{
		{"Offset", valuetypes.UI32}, {"Size", valuetypes.UI32}, {"Name", valuetypes.UI32}, {"TypeName", valuetypes.UI32}, {"Type", valuetypes.I32},
	}},
	{"CxSourceLine", []structField{{"ProgramCounter", valuetypes.UI32}, {"File", valuetypes.UI32}, {"Number", valuetypes.UI32}}},
}

// PackageRegister adds the cxir package and its structs to prgrm. The
// functions of sys/cxir are the opcodes of opcodetable.OpcodeGroupCxir,
// which CxirOpcodeDispatch runs on the old loop and the vm runs itself
// (decision D153).
func PackageRegister(prgrm *ast.CXProgram) {
	cxirPkg := ast.PackageMake(prgrm.CXProgramNameIntern("cxir"))
	pkgIdx := prgrm.PackageAdd(cxirPkg)
	cxirPkg, err := prgrm.PackageFromArrayGet(pkgIdx)
	if err != nil {
		panic(err)
	}
	for _, declaration := range structDeclarations {
		strct := ast.StructMake(prgrm.CXProgramNameIntern(declaration.structDeclarationName))
		for _, field := range declaration.structDeclarationFields {
			_, arg := ast.ArgumentMake(prgrm, prgrm.CXProgramNameIntern(field.structFieldName), ast.SourceFileGidNil, 0)
			arg.TypeSet(field.structFieldType).PackageSet(cxirPkg)
			strct.FieldAdd(prgrm, arg)
		}
		cxirPkg.StructAdd(prgrm, strct)
	}
}
