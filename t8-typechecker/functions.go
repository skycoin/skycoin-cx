package type_checker

import (
	"bytes"
	"regexp"

	"github.com/skycoin/skycoin-cx/t1-plloader" // thin: only File; no depth gain
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t6-declextractor"
)

// Parse Function Headers
// - takes in funcs from cx/cmd/declaration_extractor
// - adds func headers to AST
func ParseFuncHeaders(files []*loader.File, funcs []declaration_extractor.FuncDeclaration) error {
	prgrm := actions.AST

	for _, fun := range funcs {

		// Get Package
		pkgNameGid := actions.AST.CXProgramNameIntern(fun.PackageID)
		pkg, err := actions.AST.GetPackage(pkgNameGid)

		// If package not in AST
		if err != nil || pkg == nil {

			// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
			// Now a constructor call, site S1505.
			newPkg := ast.MakePackage(pkgNameGid)
			// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
			// Now an arena insert, site S1506.
			pkgIdx := actions.AST.AddPackage(newPkg)
			// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
			// Now an arena getter, site S1507.
			newPkg, err := actions.AST.GetPackageFromArray(pkgIdx)

			if err != nil {
				return err
			}

			pkg = newPkg
		}

		actions.AST.SelectPackage(pkgNameGid)

		source, err := GetSourceBytes(files, fun.FileID)
		if err != nil {
			return err
		}

		funcDeclarationLine := source[fun.StartOffset : fun.StartOffset+fun.Length]

		reFuncMethod := regexp.MustCompile(`func\s*\(\s*(.+)\s*\)\s*\S+\s*\(([\s\w,]*)\)(?:\s*\(([\s\w,]*)\))*`)
		funcMethod := reFuncMethod.FindSubmatch(funcDeclarationLine)

		if funcMethod != nil {
			receiverArg, err := ParseParameterDeclaration(prgrm, funcMethod[1], pkg, prgrm.CXProgramSourceFileIntern(fun.FileID), fun.LineNumber)
			if err != nil {
				return err
			}

			// OBJMODEL: state.CXArgumentMng (manager storage) or []CXArgumentGid, phase 4 #326.
			// Now a slice of objects, site S1508.
			fnIdx := actions.FunctionHeader(actions.AST, fun.FuncName, []*ast.CXArgument{receiverArg}, true, 0, 0)

			var inputs []*ast.CXArgument
			var outputs []*ast.CXArgument

			if funcMethod[2] != nil && len(funcMethod[2]) != 0 {
				inputs, err = ParseFuncParameters(prgrm, funcMethod[2], pkg, prgrm.CXProgramSourceFileIntern(fun.FileID), fun.LineNumber)
				if err != nil {
					return err
				}
			}

			if funcMethod[3] != nil && len(funcMethod[3]) != 0 {
				outputs, err = ParseFuncParameters(prgrm, funcMethod[3], pkg, prgrm.CXProgramSourceFileIntern(fun.FileID), fun.LineNumber)
				if err != nil {
					return err
				}
			}

			PreFunctionDeclaration(fnIdx, inputs, outputs)

		} else {

			reFuncRegular := regexp.MustCompile(`func\s*\S+\s*\(([\s\w,]*)\)(?:\s*\(([\s\w,]*)\))*`)
			funcRegular := reFuncRegular.FindSubmatch(funcDeclarationLine)

			fnIdx := actions.FunctionHeader(actions.AST, fun.FuncName, nil, false, 0, 0)

			var inputs []*ast.CXArgument
			var outputs []*ast.CXArgument

			if funcRegular[1] != nil && len(funcRegular[1]) != 0 {
				inputs, err = ParseFuncParameters(prgrm, funcRegular[1], pkg, prgrm.CXProgramSourceFileIntern(fun.FileID), fun.LineNumber)
				if err != nil {
					return err
				}
			}

			if funcRegular[2] != nil && len(funcRegular[2]) != 0 {
				outputs, err = ParseFuncParameters(prgrm, funcRegular[2], pkg, prgrm.CXProgramSourceFileIntern(fun.FileID), fun.LineNumber)
				if err != nil {
					return err
				}
			}

			PreFunctionDeclaration(fnIdx, inputs, outputs)
		}

	}
	return nil
}

func ParseFuncParameters(prgrm *ast.CXProgram, paramBytes []byte, pkg *ast.CXPackage, fileName ast.SourceFileGid, lineno int) ([]*ast.CXArgument, error) {
	var parameterList []*ast.CXArgument

	tokens := bytes.Split(paramBytes, []byte(","))
	for _, token := range tokens {
		parameter, err := ParseParameterDeclaration(prgrm, token, pkg, fileName, lineno)
		if err != nil {
			return nil, err
		}
		parameterList = append(parameterList, parameter)
	}

	return parameterList, nil
}

func PreFunctionDeclaration(fnIdx ast.CXFunctionGid, inputs []*ast.CXArgument, outputs []*ast.CXArgument) {
	fn := actions.AST.CxIrState.CXFunctionMng.CXFunctionGet(fnIdx)
	// adding inputs, outputs
	for _, inp := range inputs {
		fn.AddInput(actions.AST, inp)
	}
	for _, out := range outputs {
		fn.AddOutput(actions.AST, out)
	}
}
