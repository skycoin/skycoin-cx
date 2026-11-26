package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	filelog "github.com/skycoin/skycoin-cx/t0-cx-runtime-filelog"
	types "github.com/skycoin/skycoin-cx/t0-types"
	sandbox "github.com/skycoin/skycoin-cx/t1-cx-runtime-sandbox"
	ast "github.com/skycoin/skycoin-cx/t3-ast"
	actions "github.com/skycoin/skycoin-cx/t4-actions"
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
	buildOptionsDebug   bool
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
	if options.buildOptionsOutput == "" {
		output, problem := buildOutputDefault(commandLine.Args())
		if problem != "" {
			return buildFail("no output file, " + problem + ", give one with -o, see cx build --help")
		}
		if options.buildOptionsTarget == buildTargetImage {
			output = strings.TrimSuffix(output, ".exe") + buildImageExtension
		}
		options.buildOptionsOutput = output
	}
	if !parseProgram(compileOptions, fileNames, sourceCode) {
		return types.CX_COMPILATION_ERROR
	}
	if options.buildOptionsTarget == buildTargetImage {
		return buildImage(options)
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
	active := core.CoreLibrariesActive(actions.AST)
	for library, used := range actions.AST.ProgramLibrariesUsed {
		active[library] = active[library] && used
	}
	libraries := core.CoreLibraryNames(active)
	corePath, problem := buildCorePath(options.buildOptionsCore, libraries)
	if problem != "" {
		return buildFail(problem)
	}
	if buildPathSame(corePath, options.buildOptionsOutput) {
		return buildFail(fmt.Sprintf("the output %s is the core binary", options.buildOptionsOutput))
	}
	coreBinary, err := os.ReadFile(corePath)
	if err != nil {
		return buildFileFail(sandbox.FileErrorWrap(sandbox.FileOperationRead, corePath, err))
	}
	if len(coreBinary) >= core.CoreTrailerSize {
		if _, found, _ := core.CoreTrailerDecode(coreBinary[len(coreBinary)-core.CoreTrailerSize:], uint64(len(coreBinary))); found {
			return buildFail(fmt.Sprintf("%s is a built program, not a core binary", corePath))
		}
	}

	output := options.buildOptionsOutput
	if err := os.WriteFile(output, core.CoreTrailerAppend(coreBinary, program), 0o755); err != nil {
		return buildFileFail(sandbox.FileErrorWrap(sandbox.FileOperationWrite, output, err))
	}
	// WriteFile keeps the mode of a file that exists and applies the umask.
	if err := os.Chmod(output, 0o755); err != nil {
		return buildFileFail(sandbox.FileErrorWrap(sandbox.FileOperationWrite, output, err))
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
		if !actions.AST.ProgramLibrariesUsed[library] {
			actions.AST.ProgramOperationsEnabled.OpcodeSetLibraryRemove(library)
		}
	}
	return ast.CXProgramSerialize(actions.AST, true, nil)
}

// buildOptionsParse splits args into the flags of cx build and the rest. It
// returns a problem with the arguments, or "".
func buildOptionsParse(args []string) (options buildOptions, problem string) {
	for index := 0; index < len(args); index++ {
		arg := args[index]
		name, value, valueGiven := strings.Cut(strings.TrimLeft(arg, "-"), "=")
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
		case "llvm-debug":
			options.buildOptionsDebug = true
			continue
		default:
			options.buildOptionsCompile = append(options.buildOptionsCompile, arg)
			continue
		}
		if !valueGiven {
			if index+1 == len(args) {
				return options, fmt.Sprintf("flag %s needs a value", arg)
			}
			index++
			value = args[index]
		}
		*target = value
	}
	switch options.buildOptionsTarget {
	case "":
		options.buildOptionsTarget = buildTargetCore
	case buildTargetCore, buildTargetLlvm, buildTargetImage:
	default:
		return options, fmt.Sprintf("unknown --target %q, want %s, %s or %s", options.buildOptionsTarget, buildTargetCore, buildTargetLlvm, buildTargetImage)
	}
	return options, ""
}

// buildOutputDefault names the output of cx build without -o in the working
// directory as go build does, after the first source file without .cx or
// after the one source directory, with .exe on Windows. sources are the
// arguments that sourceargs.Parse reads. It returns a problem when they
// give no single name.
func buildOutputDefault(sources []string) (string, string) {
	var files, directories []string
	skip := false
	for _, arg := range sources {
		if skip {
			skip = false
			continue
		}
		// sourceargs.Parse skips flags and the value of a flag without =.
		if len(arg) > 1 && arg[0] == '-' {
			skip = !strings.Contains(arg, "=")
			continue
		}
		if strings.HasPrefix(arg, "++") {
			continue
		}
		if pathStat, err := filelog.Stat(filepath.Clean(arg)); err == nil && pathStat.IsDir() {
			directories = append(directories, arg)
		} else {
			files = append(files, arg)
		}
	}
	var name string
	switch {
	case len(directories) == 0 && len(files) > 0:
		name = filepath.Base(files[0])
		if filepath.Ext(name) != ".cx" {
			return "", fmt.Sprintf("the source %s does not end in .cx", files[0])
		}
		name = strings.TrimSuffix(name, ".cx")
	case len(directories) == 1 && len(files) == 0:
		directory, err := filepath.Abs(directories[0])
		if err != nil {
			return "", err.Error()
		}
		name = filepath.Base(directory)
	default:
		return "", "the sources are several directories or directories and files"
	}
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "", "the sources give no name"
	}
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if sourceStat, err := os.Stat(name); err == nil && sourceStat.IsDir() {
		return "", fmt.Sprintf("%s is a directory", name)
	}
	return name, ""
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
	if pathStat, err := os.Stat(path); path != "" && (err != nil || !pathStat.IsDir()) {
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

func buildFail(message string) int {
	fmt.Fprintf(os.Stderr, "cx build: %s\n", message)
	return types.CX_COMPILATION_ERROR
}

func buildFileFail(fileError *sandbox.FileError) int {
	return buildFail(fmt.Sprintf("%s: %s", fileError.FileErrorPath, fileError.FileErrorMessage))
}

func buildUsagePrint() {
	fmt.Printf(`Usage: cx build [cx flags] source-files [-o output] [--core path] [--target core|llvm|image]

Compiles the program and writes output, one executable file that holds the
core binary, the serialized program and a trailer. The core runs the program
and passes its own command line arguments to it as os.Args.

-o, --output path   The executable file to write. Without -o, as go build,
                    it is named after the first source file without .cx,
                    or after the source directory, in the working
                    directory, with .exe on Windows: cx build fib.cx
                    writes fib.
--core path         The core binary, or a directory of cores. By default the
                    core next to cx that links the libraries the program
                    calls and --enable enables, %s for none,
                    %s-file-net for file and net.
--target name       core, the default, appends the program to a core. llvm
                    compiles the program to LLVM IR, output.ll, and with
                    clang (or CX_CLANG) to a native executable, output.
                    An output ending in .ll gets the IR alone. It needs a
                    program that runs on vm and calls no system library.
                    image writes the image vm runs, output.cximage, for
                    cx-core-image, the vm runner for microcontrollers. Give
                    -stack-size to fit the board, see
                    docs/Microcontrollers.md.
--llvm-stats        With --target=llvm, print how each scalar opcode was
                    emitted, inline or as a call into the runtime.
--llvm-debug        With --target=llvm, add DWARF debug information: the
                    source line of each instruction and the name and type
                    of each variable kept in a register.

The core has no parser. It links a system library only when the program
calls it and --enable enables it, and only the cipher library brings a
module outside the Go standard library, the cipher package of Skycoin.
The built program records the libraries it was built with, and a call
into a library that was not enabled is a runtime error. A program that uses
an opcode the core does not include is refused with the list of those
opcodes.
`, buildCoreName, buildCoreName)
}
