package astapi

import (
	"errors"

	cxast "github.com/skycoin/skycoin-cx/t3-ast"
)

// FindPackage returns the package named packageName. It does not intern
// the name, since a name that was never interned names no package.
func FindPackage(cxprogram *cxast.CXProgram, packageName string) (*cxast.CXPackage, error) {
	packageNameGid, found := cxprogram.CXProgramNameFind(packageName)
	if !found {
		return nil, errors.New("package not found")
	}
	if _, ok := cxprogram.Packages[packageNameGid]; !ok {
		return nil, errors.New("package not found")
	}
	// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
	// Now an arena getter, site S1615.
	return cxprogram.GetPackage(packageNameGid)
}

// FindFunction returns the function named functionName in any package.
func FindFunction(cxprogram *cxast.CXProgram, functionName string) (*cxast.CXFunction, error) {
	functionNameGid, found := cxprogram.CXProgramNameFind(functionName)
	if !found {
		return nil, errors.New("function not found")
	}
	for _, pkgIdx := range cxprogram.Packages {
		// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
		// Now an arena getter, site S1616.
		pkg, err := cxprogram.GetPackageFromArray(pkgIdx)
		if err != nil {
			panic(err)
		}

		if fnIdx, ok := pkg.Functions[functionNameGid]; ok {
			return cxprogram.CxIrState.CXFunctionMng.CXFunctionGet(fnIdx), nil
		}
	}
	return nil, errors.New("function not found")
}
