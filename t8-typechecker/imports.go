package type_checker

import (
	"github.com/skycoin/skycoin-cx/t0-packages" // thin: only IsBuiltinImport; no depth gain
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t6-declextractor"
)

func ParseImports(imports []declaration_extractor.ImportDeclaration) error {

	// Make and add import packages to AST
	for _, imprt := range imports {
		// DeclareImport adds the built-in packages under their own name.
		if packages.IsBuiltinImport(imprt.ImportName) {
			continue
		}

		// Get Package
		pkg, err := actions.AST.GetPackage(imprt.ImportName)

		// If package not in AST
		if err != nil || pkg == nil {

			// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
			// Now a constructor call, site S1516.
			newPkg := ast.MakePackage(imprt.ImportName)
			// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
			// Now an arena insert, site S1517.
			pkgIdx := actions.AST.AddPackage(newPkg)
			// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
			// Now an arena getter, site S1518.
			newPkg, err := actions.AST.GetPackageFromArray(pkgIdx)

			if err != nil {
				return err
			}

			pkg = newPkg
		}
	}

	// Declare import in the correct packages
	for _, imprt := range imports {

		// Get Package
		pkg, err := actions.AST.GetPackage(imprt.PackageID)

		// If package not in AST
		if err != nil || pkg == nil {

			// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
			// Now a constructor call, site S1519.
			newPkg := ast.MakePackage(imprt.PackageID)
			// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
			// Now an arena insert, site S1520.
			pkgIdx := actions.AST.AddPackage(newPkg)
			// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
			// Now an arena getter, site S1521.
			newPkg, err := actions.AST.GetPackageFromArray(pkgIdx)

			if err != nil {
				return err
			}

			pkg = newPkg
		}

		actions.AST.SelectPackage(imprt.PackageID)

		actions.DeclareImport(actions.AST, imprt.ImportName, imprt.FileID, imprt.LineNumber)

	}

	return nil

}
