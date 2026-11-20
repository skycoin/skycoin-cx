package cxparsering

import (
	"bufio"
	"path/filepath"
	"regexp"
	"strings"

	cxpackages "github.com/skycoin/skycoin-cx/t0-packages" // thin: only IsDefaultPackage; no depth gain
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t0-util"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t1-profiling" // thin: only CleanupAndExit, StartProfile, StopProfile; no depth gain
	"github.com/skycoin/skycoin-cx/t2-globals"   // thin: only FoundCompileErrors, SRCPATH; no depth gain
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	cxpartialparsing "github.com/skycoin/skycoin-cx/t5-cxpartialparsing" // thin: only CurrentFileName, Parse, Program; no depth gain
)

// preliminarystage performs a first pass for the CX cxgo. Globals, packages and
// custom types are added to `cxpartialparsing.Program`.
func Preliminarystage(srcStrs, srcNames []string) int {
	var prePkg *ast.CXPackage
	parseErrors := 0

	reMultiCommentOpen := regexp.MustCompile(`/\*`)
	reMultiCommentClose := regexp.MustCompile(`\*/`)
	reComment := regexp.MustCompile("//")

	rePkg := regexp.MustCompile("package")
	rePkgName := regexp.MustCompile(`(^|[\s])package\s+([_a-zA-Z][_a-zA-Z0-9]*)`)
	reStrct := regexp.MustCompile("type")
	reStrctName := regexp.MustCompile(`(^|[\s])type\s+([_a-zA-Z][_a-zA-Z0-9]*)?\s`)

	reGlbl := regexp.MustCompile("var")
	reGlblName := regexp.MustCompile(`(^|[\s])var\s([_a-zA-Z][_a-zA-Z0-9]*)`)

	reBodyOpen := regexp.MustCompile("{")
	reBodyClose := regexp.MustCompile("}")

	reImp := regexp.MustCompile("import")
	reImpName := regexp.MustCompile(`(^|[\s])import\s+"([_a-zA-Z][_a-zA-Z0-9/-]*)"`)

	profiling.StartProfile("1. packages/structs")
	// 1. Identify all the packages and structs
	for srcI, srcStr := range srcStrs {
		srcName := srcNames[srcI]
		profiling.StartProfile(srcName)

		reader := strings.NewReader(srcStr)
		scanner := bufio.NewScanner(reader)
		var commentedCode bool
		var lineno = 0
		for scanner.Scan() {
			line := scanner.Bytes()
			lineno++

			// Identify whether we are in a comment or not.
			commentLoc := reComment.FindIndex(line)
			multiCommentOpenLoc := reMultiCommentOpen.FindIndex(line)
			multiCommentCloseLoc := reMultiCommentClose.FindIndex(line)
			if commentedCode && multiCommentCloseLoc != nil {
				commentedCode = false
			}
			if commentedCode {
				continue
			}
			if multiCommentOpenLoc != nil && !commentedCode && multiCommentCloseLoc == nil {
				commentedCode = true
				continue
			}

			// At this point we know that we are *not* in a comment
			// 1-a. Identify all the packages
			if loc := rePkg.FindIndex(line); loc != nil {
				if (commentLoc != nil && commentLoc[0] < loc[0]) ||
					(multiCommentOpenLoc != nil && multiCommentOpenLoc[0] < loc[0]) ||
					(multiCommentCloseLoc != nil && multiCommentCloseLoc[0] > loc[0]) {
					// then it's commented out
					continue
				}

				if match := rePkgName.FindStringSubmatch(string(line)); match != nil {
					if pkg, err := cxpartialparsing.Program.GetPackage(match[len(match)-1]); err != nil {
						// then it hasn't been added
						// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
						// Now a constructor call, site S1620.
						newPkg := ast.MakePackage(match[len(match)-1])
						// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
						// Now an arena insert, site S1621.
						pkgIdx := cxpartialparsing.Program.AddPackage(newPkg)
						// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
						// Now an arena getter, site S1622.
						newPkg, err = cxpartialparsing.Program.GetPackageFromArray(pkgIdx)
						if err != nil {
							panic(err)
						}
						prePkg = newPkg
					} else {
						prePkg = pkg
					}
				}
			}

			// 1-b. Identify all the structs
			if loc := reStrct.FindIndex(line); loc != nil {
				if (commentLoc != nil && commentLoc[0] < loc[0]) ||
					(multiCommentOpenLoc != nil && multiCommentOpenLoc[0] < loc[0]) ||
					(multiCommentCloseLoc != nil && multiCommentCloseLoc[0] > loc[0]) {
					// then it's commented out
					continue
				}

				if match := reStrctName.FindStringSubmatch(string(line)); match != nil {
					if prePkg == nil {
						println(ast.CompilationError(srcName, lineno),
							"No package defined")
					} else if _, err := cxpartialparsing.Program.GetStruct(match[len(match)-1], prePkg.Name); err != nil {
						// then it hasn't been added
						// OBJMODEL: state.CXStructMng.CXStructAlloc(), phase 4 #336.
						// Now a constructor call, site S1623.
						strct := ast.MakeStruct(match[len(match)-1])
						prePkg.AddStruct(cxpartialparsing.Program, strct)
					}
				}
			}
		}
		profiling.StopProfile(srcName)
	} // for range srcStrs
	profiling.StopProfile("1. packages/structs")

	profiling.StartProfile("2. globals")
	// 2. Identify all global variables
	//    We also identify packages again, so we know to what
	//    package we're going to add the variable declaration to.
	for i, source := range srcStrs {
		profiling.StartProfile(srcNames[i])
		// inBlock needs to be 0 to guarantee that we're in the global scope
		var inBlock int
		var commentedCode bool

		scanner := bufio.NewScanner(strings.NewReader(source))
		for scanner.Scan() {
			line := scanner.Bytes()

			// we need to ignore function bodies
			// it'll also ignore struct declaration's bodies, but this doesn't matter
			commentLoc := reComment.FindIndex(line)

			multiCommentOpenLoc := reMultiCommentOpen.FindIndex(line)
			multiCommentCloseLoc := reMultiCommentClose.FindIndex(line)

			if commentedCode && multiCommentCloseLoc != nil {
				commentedCode = false
			}

			if commentedCode {
				continue
			}

			if multiCommentOpenLoc != nil && !commentedCode && multiCommentCloseLoc == nil {
				commentedCode = true
				// continue
			}

			// Identify all the package imports.
			if loc := reImp.FindIndex(line); loc != nil {
				if (commentLoc != nil && commentLoc[0] < loc[0]) ||
					(multiCommentOpenLoc != nil && multiCommentOpenLoc[0] < loc[0]) ||
					(multiCommentCloseLoc != nil && multiCommentCloseLoc[0] > loc[0]) {
					// then it's commented out
					continue
				}

				if match := reImpName.FindStringSubmatch(string(line)); match != nil {
					pkgName := match[len(match)-1]
					// Checking if `pkgName` already exists and if it's not a standard library package.
					if _, err := cxpartialparsing.Program.GetPackage(pkgName); err != nil && !cxpackages.IsDefaultPackage(pkgName) {
						_, sourceCode, fileNames, err := util.ParseArgsForCX([]string{filepath.Join(globals.SRCPATH, pkgName)}, false)
						if err != nil {
							println(err.Error())
							return parseErrors + 1
						}
						ParseSourceCode(sourceCode, fileNames)
					}
				}
			}

			// we search for packages at the same time, so we can know to what package to add the global
			if loc := rePkg.FindIndex(line); loc != nil {
				if (commentLoc != nil && commentLoc[0] < loc[0]) ||
					(multiCommentOpenLoc != nil && multiCommentOpenLoc[0] < loc[0]) ||
					(multiCommentCloseLoc != nil && multiCommentCloseLoc[0] > loc[0]) {
					// then it's commented out
					continue
				}

				if match := rePkgName.FindStringSubmatch(string(line)); match != nil {
					if pkg, err := cxpartialparsing.Program.GetPackage(match[len(match)-1]); err != nil {
						// then it hasn't been added
						// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
						// Now a constructor call, site S1624.
						prePkg = ast.MakePackage(match[len(match)-1])
						// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
						// Now an arena insert, site S1625.
						pkgIdx := cxpartialparsing.Program.AddPackage(prePkg)
						// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
						// Now an arena getter, site S1626.
						prePkg, err = cxpartialparsing.Program.GetPackageFromArray(pkgIdx)
						if err != nil {
							panic(err)
						}
					} else {
						prePkg = pkg
					}
				}
			}

			if locs := reBodyOpen.FindAllIndex(line, -1); locs != nil {
				for _, loc := range locs {
					if !(multiCommentCloseLoc != nil && multiCommentCloseLoc[0] > loc[0]) {
						// then it's outside of a */, e.g. `*/ }`
						if (commentLoc == nil && multiCommentOpenLoc == nil && multiCommentCloseLoc == nil) ||
							(commentLoc != nil && commentLoc[0] > loc[0]) ||
							(multiCommentOpenLoc != nil && multiCommentOpenLoc[0] > loc[0]) ||
							(multiCommentCloseLoc != nil && multiCommentCloseLoc[0] < loc[0]) {
							// then we have an uncommented opening bracket
							inBlock++
						}
					}
				}
			}
			if locs := reBodyClose.FindAllIndex(line, -1); locs != nil {
				for _, loc := range locs {
					if !(multiCommentCloseLoc != nil && multiCommentCloseLoc[0] > loc[0]) {
						if (commentLoc == nil && multiCommentOpenLoc == nil && multiCommentCloseLoc == nil) ||
							(commentLoc != nil && commentLoc[0] > loc[0]) ||
							(multiCommentOpenLoc != nil && multiCommentOpenLoc[0] > loc[0]) ||
							(multiCommentCloseLoc != nil && multiCommentCloseLoc[0] < loc[0]) {
							// then we have an uncommented closing bracket
							inBlock--
						}
					}
				}
			}

			// we could have this situation: {var local i32}
			// but we don't care about this, as the later passes will throw an error as it's invalid syntax

			if loc := rePkg.FindIndex(line); loc != nil {
				if (commentLoc != nil && commentLoc[0] < loc[0]) ||
					(multiCommentOpenLoc != nil && multiCommentOpenLoc[0] < loc[0]) ||
					(multiCommentCloseLoc != nil && multiCommentCloseLoc[0] > loc[0]) {
					// then it's commented out
					continue
				}

				if match := rePkgName.FindStringSubmatch(string(line)); match != nil {
					if pkg, err := cxpartialparsing.Program.GetPackage(match[len(match)-1]); err != nil {
						// it should be already present
						panic(err)
					} else {
						prePkg = pkg
					}
				}
			}

			// finally, if we read a "var" and we're in global scope, we add the global without any type
			// the type will be determined later on
			if loc := reGlbl.FindIndex(line); loc != nil {
				if (commentLoc != nil && commentLoc[0] < loc[0]) ||
					(multiCommentOpenLoc != nil && multiCommentOpenLoc[0] < loc[0]) ||
					(multiCommentCloseLoc != nil && multiCommentCloseLoc[0] > loc[0]) || inBlock != 0 {
					// then it's commented out or inside a block
					continue
				}
				if match := reGlblName.FindStringSubmatch(string(line)); match != nil {
					if _, err := prePkg.GetGlobal(actions.AST, match[len(match)-1]); err != nil {
						// then it hasn't been added
						// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
						// Now a constructor call, site S1627.
						arg := ast.MakeArgument(match[len(match)-1], "", 0)
						arg.Offset = types.InvalidPointer
						arg.Package = ast.CXPackageIndex(prePkg.Index)
						// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
						// Now an arena insert, site S1628.
						argIdx := actions.AST.AddCXArgInArray(arg)

						prePkg.AddGlobal(actions.AST, argIdx)
					}
				}
			}
		}
		profiling.StopProfile(srcNames[i])
	}
	profiling.StopProfile("2. globals")

	profiling.StartProfile("3. cxpartialparsing")

	for i, source := range srcStrs {
		profiling.StartProfile(srcNames[i])
		source = source + "\n"
		if len(srcNames) > 0 {
			cxpartialparsing.CurrentFileName = srcNames[i]
		}
		/*
			passone
		*/
		parseErrors += Passone(source)
		profiling.StopProfile(srcNames[i])
	}

	profiling.StopProfile("3. cxpartialparsing")
	return parseErrors
}

func AddInitFunction(prgrm *ast.CXProgram) error {
	mainPkg, err := prgrm.GetPackage(constants.MAIN_PKG)
	if err != nil {
		return err
	}

	// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
	// Now a constructor call, site S1629.
	initFn := ast.MakeFunction(constants.SYS_INIT_FUNC, actions.CurrentFile, actions.LineNo)
	_, fnIdx := mainPkg.AddFunction(prgrm, initFn)

	//Init Expressions
	actions.FunctionDeclaration(prgrm, fnIdx, nil, nil, prgrm.SysInitExprs)

	if _, err := mainPkg.SelectFunction(prgrm, constants.MAIN_FUNC); err != nil {
		return err
	}
	return nil
}
