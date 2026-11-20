package astapi

import (
	"errors"

	cxast "github.com/skycoin/skycoin-cx/t3-ast"
)

func FindPackage(cxprogram *cxast.CXProgram, packageName string) (*cxast.CXPackage, error) {
	for _, pkgIdx := range cxprogram.Packages {
		// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
		// Now an arena getter, site S1615.
		pkg, err := cxprogram.GetPackageFromArray(pkgIdx)
		if err != nil {
			panic(err)
		}

		if pkg.Name == packageName {
			return pkg, nil
		}
	}
	return nil, errors.New("package not found")
}

func FindFunction(cxprogram *cxast.CXProgram, functionName string) (*cxast.CXFunction, error) {
	for _, pkgIdx := range cxprogram.Packages {
		// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
		// Now an arena getter, site S1616.
		pkg, err := cxprogram.GetPackageFromArray(pkgIdx)
		if err != nil {
			panic(err)
		}

		for _, fnIdx := range pkg.Functions {
			// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
			// Now an arena getter, site S1617.
			fn := cxprogram.GetFunctionFromArray(fnIdx)

			if fn.Name == functionName {
				return fn, nil
			}
		}
	}
	return nil, errors.New("function not found")
}
