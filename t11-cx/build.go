package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t0-util"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	core "github.com/skycoin/skycoin-cx/t6-cx-runtime-core"
)

// buildCommand is the first argument of cx that builds an executable, see
// "cx build" in README.md.
const buildCommand = "build"

// buildCoreName is the file name of the core binary that cx build looks for
// next to cx when --core is not given.
const buildCoreName = "cx-core"

// buildOptions holds the flags of cx build. buildOptionsCompile are the
// arguments left for the compiler, the source files and the cx flags.
type buildOptions struct {
	buildOptionsOutput  string
	buildOptionsCore    string
	buildOptionsHelp    bool
	buildOptionsCompile []string
}

// buildRun runs cx build with args, the arguments after "build", and returns
// the exit code. Every failure is CX_COMPILATION_ERROR with a message on
// stderr.
func buildRun(args []string) int {
	options, problem := buildOptionsParse(args)
	if options.buildOptionsHelp {
		buildUsagePrint()
		return types.CX_SUCCESS
	}
	if problem != "" {
		return buildFail(problem + ", see cx build --help")
	}
	corePath, problem := buildCorePath(options.buildOptionsCore)
	if problem != "" {
		return buildFail(problem)
	}
	if buildPathSame(corePath, options.buildOptionsOutput) {
		return buildFail(fmt.Sprintf("the output %s is the core binary", options.buildOptionsOutput))
	}
	coreBinary, err := os.ReadFile(corePath)
	if err != nil {
		return buildFileFail(util.FileErrorWrap(util.FileOperationRead, corePath, err))
	}
	if len(coreBinary) >= core.CoreTrailerSize {
		if _, found, _ := core.CoreTrailerDecode(coreBinary[len(coreBinary)-core.CoreTrailerSize:], uint64(len(coreBinary))); found {
			return buildFail(fmt.Sprintf("%s is a built program, not a core binary", corePath))
		}
	}

	coreOpcodes := core.CoreOpcodeSet()
	compileOptions, cxArgs, sourceCode, fileNames, start := runStart(options.buildOptionsCompile)
	if !start {
		return types.CX_SUCCESS
	}
	if len(cxArgs) > 0 {
		return buildFail(fmt.Sprintf("program arguments %s are given when the built program runs, not to cx build", strings.Join(cxArgs, " ")))
	}
	if len(sourceCode) == 0 {
		return buildFail("no source files, see cx build --help")
	}
	if !parseProgram(compileOptions, fileNames, sourceCode) {
		return types.CX_COMPILATION_ERROR
	}

	var missing []string
	for _, code := range core.CoreOpcodesUsed(actions.AST) {
		if !coreOpcodes[code] {
			missing = append(missing, core.CoreOpcodeName(code))
		}
	}
	if len(missing) > 0 {
		return buildFail(fmt.Sprintf("the core has no module outside the Go standard library, and the program uses opcodes it does not include: %s", strings.Join(missing, ", ")))
	}
	core.CoreLibrariesEnabledResolve(actions.AST)
	for library, enabled := range actions.AST.ProgramLibrariesEnabled {
		actions.AST.ProgramLibrariesEnabled[library] = enabled && actions.AST.ProgramLibrariesUsed[library]
	}
	if enabled := core.CoreLibraryNames(actions.AST.ProgramLibrariesEnabled); len(enabled) > 0 {
		return buildFail(fmt.Sprintf("the core links no system library, and the program calls the enabled libraries: %s", strings.Join(enabled, ", ")))
	}

	program, err := ast.SerializeCXProgram(actions.AST, true, nil)
	if err != nil {
		return buildFail(err.Error())
	}
	output := options.buildOptionsOutput
	if err := os.WriteFile(output, core.CoreTrailerAppend(coreBinary, program), 0o755); err != nil {
		return buildFileFail(util.FileErrorWrap(util.FileOperationWrite, output, err))
	}
	// WriteFile keeps the mode of a file that exists and applies the umask.
	if err := os.Chmod(output, 0o755); err != nil {
		return buildFileFail(util.FileErrorWrap(util.FileOperationWrite, output, err))
	}
	return types.CX_SUCCESS
}

// buildOptionsParse splits args into the flags of cx build and the rest. It
// returns a problem with the arguments, or "".
func buildOptionsParse(args []string) (options buildOptions, problem string) {
	for index := 0; index < len(args); index++ {
		arg := args[index]
		name, value, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if !strings.HasPrefix(arg, "-") {
			if arg == "help" {
				options.buildOptionsHelp = true
				continue
			}
			options.buildOptionsCompile = append(options.buildOptionsCompile, arg)
			continue
		}
		var target *string
		switch name {
		case "h", "help":
			options.buildOptionsHelp = true
			continue
		case "o", "output":
			target = &options.buildOptionsOutput
		case "core":
			target = &options.buildOptionsCore
		default:
			options.buildOptionsCompile = append(options.buildOptionsCompile, arg)
			continue
		}
		if !hasValue {
			if index+1 == len(args) {
				return options, fmt.Sprintf("flag %s needs a value", arg)
			}
			index++
			value = args[index]
		}
		*target = value
	}
	if options.buildOptionsOutput == "" {
		return options, "no output file, give one with -o"
	}
	return options, ""
}

// buildCorePath returns the core binary to use: path, or cx-core next to the
// running cx when path is empty.
func buildCorePath(path string) (string, string) {
	if path != "" {
		return path, ""
	}
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Sprintf("cannot find the cx executable to look for %s next to it, give the core with --core: %s", buildCoreName, err)
	}
	name := buildCoreName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(filepath.Dir(executable), name), ""
}

// buildPathSame reports whether first and second name the same file.
func buildPathSame(first, second string) bool {
	firstInfo, firstErr := os.Stat(first)
	secondInfo, secondErr := os.Stat(second)
	return firstErr == nil && secondErr == nil && os.SameFile(firstInfo, secondInfo)
}

func buildFail(message string) int {
	fmt.Fprintf(os.Stderr, "cx build: %s\n", message)
	return types.CX_COMPILATION_ERROR
}

func buildFileFail(fileError *util.FileError) int {
	return buildFail(fmt.Sprintf("%s: %s", fileError.FileErrorPath, fileError.FileErrorMessage))
}

func buildUsagePrint() {
	fmt.Printf(`Usage: cx build [cx flags] source-files -o output [--core path]

Compiles the program and writes output, one executable file that holds the
core binary, the serialized program and a trailer. The core runs the program
and passes its own command line arguments to it as os.Args.

-o, --output path   The executable file to write.
--core path         The core binary, by default %s next to cx.

The core has no parser, no system library and no module outside the Go
standard library. A program that uses an opcode the core does not include is
refused with the list of those opcodes, and so is a program that enables a
system library it calls, see --enable. A library that is not enabled stays
out of the built program, and a call into it is a runtime error.
`, buildCoreName)
}
