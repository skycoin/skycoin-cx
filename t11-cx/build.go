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
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
)

// buildCommand is the first argument of cx that builds an executable, see
// "cx build" in README.md.
const buildCommand = "build"

// buildCoreName starts the file name of every core binary, see
// buildCoreFileName.
const buildCoreName = "cx-core"

// buildOptions holds the flags of cx build. buildOptionsCompile are the
// arguments left for the compiler, the source files and the cx flags.
type buildOptions struct {
	buildOptionsOutput  string
	buildOptionsCore    string
	buildOptionsHelp    bool
	buildOptionsTarget  string
	buildOptionsStats   bool
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
	if options.buildOptionsTarget == buildTargetLlvm {
		return buildLlvm(options)
	}

	var missing []string
	for _, code := range core.CoreOpcodesUsed(actions.AST) {
		if !core.CoreOpcodeLinked(code) {
			missing = append(missing, core.CoreOpcodeName(code))
		}
	}
	if len(missing) > 0 {
		return buildFail(fmt.Sprintf("the core has no module outside the Go standard library, and the program uses opcodes it does not include: %s", strings.Join(missing, ", ")))
	}
	program, err := buildProgramImage()
	if err != nil {
		return buildFail(err.Error())
	}
	libraries := core.CoreLibraryNames(actions.AST.ProgramLibrariesEnabled)
	corePath, problem := buildCorePath(options.buildOptionsCore, libraries)
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

// buildProgramImage keeps the enabled libraries the compiled program calls
// and returns the program as cx build appends it to the core. Its hash is
// the program hash of an execution record, see docs/ExecutionRecords.md.
func buildProgramImage() ([]byte, error) {
	core.CoreLibrariesEnabledResolve(actions.AST)
	for library, enabled := range actions.AST.ProgramLibrariesEnabled {
		actions.AST.ProgramLibrariesEnabled[library] = enabled && actions.AST.ProgramLibrariesUsed[library]
	}
	return ast.SerializeCXProgram(actions.AST, true, nil)
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
		case "target":
			target = &options.buildOptionsTarget
		case "llvm-stats":
			options.buildOptionsStats = true
			continue
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
	switch options.buildOptionsTarget {
	case "":
		options.buildOptionsTarget = buildTargetCore
	case buildTargetCore, buildTargetLlvm:
	default:
		return options, fmt.Sprintf("unknown --target %q, want %s or %s", options.buildOptionsTarget, buildTargetCore, buildTargetLlvm)
	}
	return options, ""
}

// buildCoreFileName returns the file name of the core that links exactly
// libraries, in number order: cx-core, then -<library> for each, so
// cx-core-file-net links file and net. make build builds every one.
func buildCoreFileName(libraries []string) string {
	name := strings.Join(append([]string{buildCoreName}, libraries...), "-")
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// buildCorePath returns the core binary to use for a program that enables
// libraries. A path that names a file is the core. A directory, or the
// directory of the running cx when path is empty, holds the cores by
// buildCoreFileName.
func buildCorePath(path string, libraries []string) (string, string) {
	if info, err := os.Stat(path); path != "" && (err != nil || !info.IsDir()) {
		return path, ""
	}
	directory := path
	if directory == "" {
		executable, err := os.Executable()
		if err != nil {
			return "", fmt.Sprintf("cannot find the cx executable to look for %s next to it, give the core with --core: %s", buildCoreFileName(libraries), err)
		}
		directory = filepath.Dir(executable)
	}
	corePath := filepath.Join(directory, buildCoreFileName(libraries))
	if _, err := os.Stat(corePath); err != nil {
		return "", fmt.Sprintf("no core %s, which links the enabled libraries the program calls, build the cores with make build or give one with --core", corePath)
	}
	return corePath, ""
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
	fmt.Printf(`Usage: cx build [cx flags] source-files -o output [--core path] [--target core|llvm]

Compiles the program and writes output, one executable file that holds the
core binary, the serialized program and a trailer. The core runs the program
and passes its own command line arguments to it as os.Args.

-o, --output path   The executable file to write.
--core path         The core binary, or a directory of cores. By default the
                    core next to cx that links the libraries the program
                    calls and --enable enables, %s for none,
                    %s-file-net for file and net.
--target name       core, the default, appends the program to a core. llvm
                    compiles the program to LLVM IR, output.ll, and with
                    clang (or CX_CLANG) to a native executable, output.
                    An output ending in .ll gets the IR alone. It needs a
                    program that runs on vm and calls no system library.
--llvm-stats        With --target=llvm, print how each scalar opcode was
                    emitted, inline or as a call into the runtime.

The core has no parser. It links a system library only when the program
calls it and --enable enables it, and only the cipher library brings a
module outside the Go standard library, the cipher package of Skycoin.
The built program records the libraries it was built with, and a call
into a library that was not enabled is a runtime error. A program that uses
an opcode the core does not include is refused with the list of those
opcodes.
`, buildCoreName, buildCoreName)
}
