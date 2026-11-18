package type_checker

import (
	"github.com/skycoin/skycoin-cx/t1-plloader"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t6-declextractor"
	cxinit "github.com/skycoin/skycoin-cx/t7-init"
)

func ParseAllDeclarations(files []*loader.File, imports []declaration_extractor.ImportDeclaration, globals []declaration_extractor.GlobalDeclaration, structs []declaration_extractor.StructDeclaration, funcs []declaration_extractor.FuncDeclaration) error {

	// Make AST if not made yet
	if actions.AST == nil {
		actions.AST = cxinit.MakeProgram()
	}

	err := ParseImports(imports)
	if err != nil {
		return err
	}

	err = ParseStructs(files, structs)
	if err != nil {
		return err
	}

	err = ParseGlobals(files, globals)
	if err != nil {
		return err
	}

	err = ParseFuncHeaders(files, funcs)
	if err != nil {
		return err
	}

	return nil
}
