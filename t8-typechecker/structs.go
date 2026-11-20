package type_checker

import (
	"github.com/skycoin/skycoin-cx/t1-plloader" // thin: only File; no depth gain
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t6-declextractor"
)

// Parse Structs
// - takes in structs from cx/cmd/declaration_extractor
// - adds structs to AST
func ParseStructs(files []*loader.File, structs []declaration_extractor.StructDeclaration) error {

	// Get Package
	for _, strct := range structs {

		pkg, err := actions.AST.GetPackage(strct.PackageID)

		// If package is not found
		if err != nil {

			// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
			// Now a constructor call, site S1522.
			newPkg := ast.MakePackage(strct.PackageID)
			// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
			// Now an arena insert, site S1523.
			pkgIdx := actions.AST.AddPackage(newPkg)
			// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
			// Now an arena getter, site S1524.
			newPkg, err = actions.AST.GetPackageFromArray(pkgIdx)

			if err != nil {
				return err
			}

			pkg = newPkg

		}

		// Select Package to Add to AST
		actions.AST.SelectPackage(strct.PackageID)

		// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
		// Now a constructor call, site S1525.
		structCX := ast.MakeStruct(strct.StructName)
		structCX.Package = ast.CXPackageIndex(pkg.Index)

		pkg = pkg.AddStruct(actions.AST, structCX)

		src, err := GetSourceBytes(files, strct.FileID)
		if err != nil {
			return err
		}

		var structFields []*ast.CXArgument

		for _, strctFieldDec := range strct.StructFields {

			structFieldLine := src[strctFieldDec.StartOffset : strctFieldDec.StartOffset+strctFieldDec.Length]

			structFieldSpecifier, err := ParseParameterDeclaration(structFieldLine, pkg, strct.FileID, strctFieldDec.LineNumber)
			if err != nil {
				return err
			}

			structFields = append(structFields, structFieldSpecifier)
		}

		actions.DeclareStruct(actions.AST, strct.StructName, structFields)

	}
	return nil
}
