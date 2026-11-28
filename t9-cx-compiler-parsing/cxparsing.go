package parsing

import (
	"bytes"
	"errors"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t2-cx-core-environment" // thin: only FoundCompileErrors, SRCPATH; no depth gain
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"

	"github.com/skycoin/skycoin-cx/t1-cx-runtime-profiling" // thin: only CleanupAndExit, StartProfile, StopProfile; no depth gain
	"github.com/skycoin/skycoin-cx/t4-cx-compiler-astbuild"
	"github.com/skycoin/skycoin-cx/t4-cx-runtime-printer"         // thin: only CompileErrorsPrint; no depth gain
	"github.com/skycoin/skycoin-cx/t5-cx-compiler-partialparsing" // thin: only CurrentFile, Parse, Program; no depth gain
)

// ErrCompilation is returned by SourceStringsParse when the source has
// parse or compile errors. The errors themselves are printed to stdout.
var ErrCompilation = errors.New("compilation error")

// SourceStringsParse parses CX source held in memory into actions.AST.
// It returns on errors instead of exiting, so the parser can be embedded in
// a long running process.
func SourceStringsParse(sourceCodeStrings []string, fileNames []string) error {
	// The compile errors print on return, and before a panic of the compiler.
	defer func() { printer.CompileErrorsPrint(astbuild.AST) }()
	return parseSourceStrings(sourceCodeStrings, fileNames)
}

// SourceStringsParseQuiet is SourceStringsParse without printing the errors,
// for the compiler library, which hands them to the CX program (D152).
func SourceStringsParseQuiet(sourceCodeStrings []string, fileNames []string) error {
	return parseSourceStrings(sourceCodeStrings, fileNames)
}

// parseSourceStrings is SourceStringsParse without printing the errors, which
// an imported package leaves to the program that imports it.
func parseSourceStrings(sourceCodeStrings []string, fileNames []string) error {
	partialparsing.Program = astbuild.AST
	astbuild.AST.CxIrState.CxIrStateManagerGrowBoundary()
	sourceCodeStrings, fileNames = sourcesWithPackageClause(astbuild.AST, sourceCodeStrings, fileNames)

	/*
		We need to traverse the elements by hierarchy first add all the
		packages and structs at the same time then add globals, as these
		can be of a custom type (and it could be imported) the signatures
		of functions and methods are added in the partialparsing.y pass
	*/
	// The globals of the imports, which Preliminarystage compiles, come first.
	deferredFrom := len(astbuild.AST.CxIrState.CxIrStateGlobalsDeferred)
	parseErrors := 0
	var packages []*ast.CXPackage
	if len(sourceCodeStrings) > 0 {
		parseErrors, packages = Preliminarystage(sourceCodeStrings, fileNames)
	}

	astbuild.AST = partialparsing.Program
	astbuild.AST.CxIrState.CxIrStateManagerGrowBoundary()

	// A syntax error leaves the declarations incomplete. After other errors the
	// bodies are compiled too, and report each error pass one found once.
	if parseErrors > 0 {
		astbuild.AST.CxIrState.CxIrStateGlobalsDeferred = astbuild.AST.CxIrState.CxIrStateGlobalsDeferred[:deferredFrom]
		return ErrCompilation
	}
	for _, pkg := range packages {
		astbuild.PackageRedeclarationCheck(astbuild.AST, pkg)
		astbuild.ImportScopedCheck(astbuild.AST, pkg)
	}
	/*
		Adding global variables `OsArgs` to the `os` (operating system)
		package.
	*/
	if osPkg, err := astbuild.AST.PackageGet(astbuild.AST.CxIrState.CxIrStateNameOs); err == nil {
		if _, err := osPkg.GlobalGet(astbuild.AST, astbuild.AST.CxIrState.CxIrStateNameOsArgs); err != nil {
			_, arg0 := ast.ArgumentMake(astbuild.AST, astbuild.AST.CxIrState.CxIrStateNameOsArgs, ast.SourceFileGidNil, -1)
			arg0.TypeSet(valuetypes.UNDEFINED)
			arg0.Package = ast.CXPackageIndex(osPkg.Index)

			_, arg1 := ast.ArgumentMake(astbuild.AST, astbuild.AST.CxIrState.CxIrStateNameOsArgs, ast.SourceFileGidNil, -1)
			arg1.TypeSet(valuetypes.STR)
			arg1 = astbuild.DeclarationSpecifiers(arg1, []valuetypes.Pointer{0}, opcodetable.DeclarationBasic)
			arg1 = astbuild.DeclarationSpecifiers(arg1, []valuetypes.Pointer{0}, opcodetable.DeclarationSlice)
			astbuild.GlobalInPackageDeclare(astbuild.AST, osPkg, arg0, arg1, nil, false)
		}
	}

	globalsDeferredInfer(astbuild.AST, sourceCodeStrings, fileNames, deferredFrom)
	// The body pass meets the errors of the values of deferred globals again.
	astbuild.AST.CxIrState.CompileErrorList.CompileErrorListPreliminary = len(astbuild.AST.CxIrState.CompileErrorList.CompileErrorListRecords)

	profiling.ProfileStart("4. passtwo")

	/*
	 The pass two of parsing that generates the actual output.
	*/

	for i, source := range sourceCodeStrings {
		astbuild.AST.CxIrState.CxIrStateManagerGrowBoundary()

		/*
			Because of an unknown reason, sometimes some CX programs
			throw an error related to a premature EOF (particularly in Windows).
			Adding a newline character solves this.
		*/

		source = source + "\n"

		// Each file selects its package with its package clause.
		astbuild.AST.CurrentPackage = -1
		astbuild.LineNo = 1

		b := bytes.NewBufferString(source)

		astbuild.CurrentFile = ast.SourceFileGidNil
		if len(fileNames) > 0 {
			astbuild.CurrentFile = astbuild.AST.CXProgramSourceFileIntern(fileNames[i])
		}

		profiling.ProfileStart(astbuild.AST.CXProgramSourceFileText(astbuild.CurrentFile))

		astbuild.ImportScopedDirectivesRead(astbuild.CurrentFile, source)
		parseErrors += Passtwo(b)

		profiling.ProfileStop(astbuild.AST.CXProgramSourceFileText(astbuild.CurrentFile))
	}

	profiling.ProfileStop("4. passtwo")
	astbuild.AffordanceLiteralsResolve(astbuild.AST)
	astbuild.AST.CxIrState.CxIrStateManagerGrowBoundary()

	if environment.FoundCompileErrors || parseErrors > 0 {
		return ErrCompilation
	}
	return nil
}

// sourcesWithPackageClause records an error at line 1 of each file that does
// not start with a package clause and returns the other files. Go does not
// parse such a file further, and its declarations would have no package.
func sourcesWithPackageClause(prgrm *ast.CXProgram, sources []string, names []string) ([]string, []string) {
	var keptSources, keptNames []string
	for i, source := range sources {
		if sourceHasPackageClause(source) {
			if keptSources != nil {
				keptSources = append(keptSources, source)
				if len(names) > i {
					keptNames = append(keptNames, names[i])
				}
			}
			continue
		}
		if keptSources == nil {
			keptSources = append(make([]string, 0, len(sources)), sources[:i]...)
			keptNames = append(make([]string, 0, len(names)), names[:min(i, len(names))]...)
		}
		file := ast.SourceFileGidNil
		if len(names) > i {
			file = prgrm.CXProgramSourceFileIntern(names[i])
		}
		prgrm.CXProgramCompileErrorAddPosition(ast.CompileErrorCodeNoPackageDefined, file, 1, 1, ast.CompileErrorOperands{})
	}
	if keptSources == nil {
		return sources, names
	}
	return keptSources, keptNames
}

// sourceHasPackageClause reports whether the first token of source after
// white space and comments is the keyword package.
func sourceHasPackageClause(source string) bool {
	for {
		trimmed := strings.TrimLeft(source, " \t\r\n\ufeff")
		switch {
		case strings.HasPrefix(trimmed, "//"):
			end := strings.IndexByte(trimmed, '\n')
			if end < 0 {
				return false
			}
			source = trimmed[end:]
		case strings.HasPrefix(trimmed, "/*"):
			end := strings.Index(trimmed[2:], "*/")
			if end < 0 {
				return false
			}
			source = trimmed[end+4:]
		default:
			rest, found := strings.CutPrefix(trimmed, "package")
			return found && rest != "" && !sourceIdentByte(rest[0])
		}
	}
}

func sourceIdentByte(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}
