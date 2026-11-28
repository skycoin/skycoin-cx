package main

import (
	"sort"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t0-cx-runtime-output"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t2-cx-core-environment"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
	"github.com/skycoin/skycoin-cx/t4-cx-compiler-astbuild"
	"github.com/skycoin/skycoin-cx/t4-cx-library-compileropcodes"
	printer "github.com/skycoin/skycoin-cx/t4-cx-runtime-printer"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
	"github.com/skycoin/skycoin-cx/t7-cx-runtime-libraries"
	"github.com/skycoin/skycoin-cx/t9-cx-compiler-parsing"
)

// compilerLibrary is the Go compiler cx gives sys/compiler (decision D152).
// It compiles a program as cx does, into a fresh program of its own, and
// keeps the compiler state of the program that runs.
type compilerLibrary struct{}

// compilerState is the package state the Go compiler reads and writes,
// saved around a compile of the compiler library.
type compilerState struct {
	compilerStateProgram     *ast.CXProgram
	compilerStateFile        ast.SourceFileGid
	compilerStateLine        int
	compilerStateLineText    string
	compilerStateErrorsSeen  bool
	compilerStateHeapOffset  valuetypes.Pointer
	compilerStateTemporaries int
}

func compilerStateSave() compilerState {
	return compilerState{astbuild.AST, astbuild.CurrentFile, astbuild.LineNo, astbuild.LineStr, environment.FoundCompileErrors, environment.HeapOffset, astbuild.TemporaryVariableCounter}
}

func (state compilerState) compilerStateRestore() {
	astbuild.AST, astbuild.CurrentFile, astbuild.LineNo, astbuild.LineStr = state.compilerStateProgram, state.compilerStateFile, state.compilerStateLine, state.compilerStateLineText
	environment.FoundCompileErrors = state.compilerStateErrorsSeen
	environment.HeapOffset = state.compilerStateHeapOffset
	astbuild.TemporaryVariableCounter = state.compilerStateTemporaries
}

// CompilerSourcesParse compiles sources named by names as cx compiles the
// files of its command line, with the operations and output mode of the
// program that runs.
func (compilerLibrary) CompilerSourcesParse(names [][]byte, sources [][]byte, operations *opcodetable.OpcodeSet, mode output.Mode) compileropcodes.CompilerParse {
	saved := compilerStateSave()
	defer saved.compilerStateRestore()
	prgrm := libraries.ProgramMake()
	libraries.PackagesRegister(prgrm)
	builtinPackages := len(prgrm.CXPackages)
	prgrm.ProgramOperationsEnabled = *operations
	prgrm.ProgramOutputMode = mode
	astbuild.AST = prgrm
	environment.FoundCompileErrors = false
	environment.HeapOffset = 0
	astbuild.TemporaryVariableCounter = 0
	fileNames := make([]string, len(names))
	texts := make([]string, len(sources))
	for i := range names {
		fileNames[i], texts[i] = string(names[i]), string(sources[i])
	}
	err := parsing.SourceStringsParseQuiet(texts, fileNames)
	prgrm = astbuild.AST
	syntax := err != nil && len(prgrm.CxIrState.CompileErrorList.CompileErrorListRecords) == 0
	for _, record := range prgrm.CxIrState.CompileErrorList.CompileErrorListRecords {
		syntax = syntax || record.CompileErrorRecordCode == ast.CompileErrorCodeSyntax || record.CompileErrorRecordCode == ast.CompileErrorCodeNoPackageDefined
	}
	var parse compileropcodes.CompilerParse
	if !syntax {
		if _, missing := prgrm.FunctionGet(prgrm.CxIrState.CxIrStateNameMain, prgrm.CxIrState.CxIrStateNameMain); missing != nil {
			initMainPkg(prgrm)
		}
		if err := parsing.SysInitFunctionAdd(prgrm); err != nil {
			parse.CompilerParseErrors = append(parse.CompilerParseErrors, compileropcodes.CompilerError{CompilerErrorText: []byte("error: initializing *init: " + err.Error())})
		}
		astbuild.LineNo = 0
	}
	lines, errorTexts := printer.CompileErrorsCollect(prgrm)
	for i, text := range errorTexts {
		parse.CompilerParseErrors = append(parse.CompilerParseErrors, compileropcodes.CompilerError{CompilerErrorLine: int32(lines[i]), CompilerErrorText: text})
	}
	if syntax {
		if len(parse.CompilerParseErrors) == 0 {
			parse.CompilerParseErrors = append(parse.CompilerParseErrors, compileropcodes.CompilerError{CompilerErrorText: []byte("error: " + err.Error())})
		}
		return parse
	}
	if environment.FoundCompileErrors && len(parse.CompilerParseErrors) == 0 {
		parse.CompilerParseErrors = append(parse.CompilerParseErrors, compileropcodes.CompilerError{CompilerErrorText: []byte("error: " + parsing.ErrCompilation.Error())})
	}
	parse.CompilerParseProgram = prgrm
	compilerPackagesRead(prgrm, builtinPackages, &parse)
	return parse
}

// compilerPackagesRead gives parse the packages prgrm declares after its
// first builtinPackages, their functions and structs in name order without
// methods and *init, and the type table of their declarations when the
// program has no errors.
func compilerPackagesRead(prgrm *ast.CXProgram, builtinPackages int, parse *compileropcodes.CompilerParse) {
	var structs []*ast.CXStruct
	var functions []*ast.CXFunction
	for index := builtinPackages; index < len(prgrm.CXPackages); index++ {
		pkg := &prgrm.CXPackages[index]
		read := compileropcodes.CompilerPackage{CompilerPackageName: []byte(prgrm.CXProgramNameText(pkg.Name))}
		functionNames := make([]string, 0, len(pkg.Functions))
		functionsByName := map[string]*ast.CXFunction{}
		for name, gid := range pkg.Functions {
			text := prgrm.CXProgramNameText(name)
			fn := prgrm.CxIrState.CxIrStateFunctionGetOrNil(gid)
			if fn == nil || strings.Contains(text, ".") || strings.HasPrefix(text, "*") {
				continue
			}
			functionNames = append(functionNames, text)
			functionsByName[text] = fn
		}
		sort.Strings(functionNames)
		for _, name := range functionNames {
			read.CompilerPackageFunctions = append(read.CompilerPackageFunctions, int32(len(functions)))
			functions = append(functions, functionsByName[name])
		}
		structNames := make([]string, 0, len(pkg.Structs))
		structsByName := map[string]*ast.CXStruct{}
		for name, structIndex := range pkg.Structs {
			text := prgrm.CXProgramNameText(name)
			structNames = append(structNames, text)
			structsByName[text] = &prgrm.CXStructs[structIndex]
		}
		sort.Strings(structNames)
		for _, name := range structNames {
			read.CompilerPackageStructs = append(read.CompilerPackageStructs, int32(len(structs)))
			structs = append(structs, structsByName[name])
		}
		if pkg.Globals != nil {
			for _, field := range pkg.Globals.Fields {
				read.CompilerPackageGlobals = append(read.CompilerPackageGlobals, []byte(prgrm.CXProgramNameText(prgrm.CXTypeSignatureFromArrayGet(field).Name)))
			}
		}
		parse.CompilerParsePackages = append(parse.CompilerParsePackages, read)
	}
	if len(parse.CompilerParseErrors) > 0 {
		return
	}
	table, structTypes, functionRecords := ast.ReflectTableDeclarationsBuild(prgrm, structs, functions)
	for i := range parse.CompilerParsePackages {
		read := &parse.CompilerParsePackages[i]
		for j, ordinal := range read.CompilerPackageStructs {
			read.CompilerPackageStructs[j] = structTypes[ordinal]
		}
		for j, ordinal := range read.CompilerPackageFunctions {
			read.CompilerPackageFunctions[j] = functionRecords[ordinal]
		}
	}
	parse.CompilerParseTypes = table
}

// CompilerProgramLower lowers prgrm to the image cx build --target=image
// writes, byte for byte.
func (compilerLibrary) CompilerProgramLower(prgrm *ast.CXProgram) ([]byte, []byte) {
	saved := compilerStateSave()
	defer saved.compilerStateRestore()
	image, problem, _ := core.ExecutorImageBuild(prgrm)
	if problem != "" {
		return nil, []byte("error: " + problem)
	}
	return ir.ImageFileEncode(&image), nil
}
