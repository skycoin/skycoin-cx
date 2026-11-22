package type_checker

import (
	"regexp"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-plloader" // thin: only File; no depth gain
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t6-declextractor"
)

// Parse Globals
// - takes in globals from cx/cmd/declaration_extractor
// - adds globals to AST
func ParseGlobals(files []*loader.File, globals []declaration_extractor.GlobalDeclaration) error {
	prgrm := actions.AST

	// Range over global declarations and parse
	for _, global := range globals {

		// Get Package
		pkgNameGid := actions.AST.CXProgramNameIntern(global.PackageID)
		pkg, err := actions.AST.GetPackage(pkgNameGid)

		// If package not in AST
		if err != nil || pkg == nil {

			// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
			// Now a constructor call, site S1510.
			newPkg := ast.MakePackage(pkgNameGid)
			// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
			// Now an arena insert, site S1511.
			pkgIdx := actions.AST.AddPackage(newPkg)
			// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
			// Now an arena getter, site S1512.
			newPkg, err := actions.AST.GetPackageFromArray(pkgIdx)

			if err != nil {
				return err
			}

			pkg = newPkg
		}

		actions.AST.SelectPackage(pkgNameGid)

		// Read File
		source, err := GetSourceBytes(files, global.FileID)
		if err != nil {
			return err
		}

		// Extract Declaration from file
		reGlobalDeclaration := regexp.MustCompile(`var\s+(\w*)\s+([\*\[\]\w\.]+)(?:\s*\=\s*[\s\S]+\S+){0,1}`)
		globalDeclaration := source[global.StartOffset : global.StartOffset+global.Length]
		globalTokens := reGlobalDeclaration.FindSubmatch(globalDeclaration)

		// add Global as CX Argument to CX Package
		_, globalArg := ast.MakeArgument(prgrm, prgrm.CXProgramNameIntern(global.GlobalVariableName), prgrm.CXProgramSourceFileIntern(global.FileID), global.LineNumber)
		globalArg.Offset = types.InvalidPointer
		globalArg.Package = ast.CXPackageIndex(pkg.Index)
		globalArgIdx := ast.CXArgumentIndex(globalArg.Index)

		pkg.AddGlobal(actions.AST, globalArgIdx)

		var declarationSpecifier *ast.CXArgument

		declarationSpecifier, err = ParseDeclarationSpecifier(prgrm, globalTokens[2], prgrm.CXProgramSourceFileIntern(global.FileID), global.LineNumber, declarationSpecifier)
		if err != nil {
			return err
		}

		// OBJMODEL: state.CXArgumentMng.CXArgumentGet(gid), phase 5 #327.
		// Now an arena getter, site S1515.
		actions.DeclareGlobalInPackage(actions.AST, nil, actions.AST.GetCXArgFromArray(globalArgIdx), declarationSpecifier, nil, false)

	}
	return nil
}
