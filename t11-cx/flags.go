package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t1-netbatch"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core" // thin: only ExecutorAuto; no depth gain
	"github.com/skycoin/skycoin-cx/t5-cxnet"
)

type cxCmdFlags struct {
	baseOutput       bool
	lexerOutput      string
	replMode         bool
	printHelp        bool
	printVersion     bool
	printEnv         bool
	printAST         bool
	tokenizeMode     bool
	initialHeap      string
	maxHeap          string
	stackSize        string
	minHeapFreeRatio float64
	maxHeapFreeRatio float64
	cxpath           string
	sandbox          string
	executor         string
	enable           string
	netRecord        string
	netReplay        string

	// Debug flags for the CX developers
	debugFileAccess bool
	debugProfile    int
	debugGoStack    bool
}

func defaultCmdFlags() cxCmdFlags {
	return cxCmdFlags{
		baseOutput:      false,
		lexerOutput:     "",
		replMode:        false,
		printHelp:       false,
		printEnv:        false,
		printVersion:    false,
		debugFileAccess: false,
		debugProfile:    0,
		executor:        core.ExecutorAuto,
	}
}

// VIOLATION: state outside State. Owner: t11-cx; moves to ConfigMng in phase 2.
var commandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

func appendDash(args []string) {

	for k, v := range args {
		switch v {
		case "version":
			args[k] = "-version"
		case "ast":
			args[k] = "-ast"
		}
	}
}

func parseFlags(options *cxCmdFlags, args []string) {
	if len(args) <= 0 {
		options.replMode = true
	}

	appendDash(args)

	commandLine.BoolVar(&options.printVersion, "version", options.printVersion, "Print the CX version and exit")
	commandLine.BoolVar(&options.printVersion, "v", options.printVersion, "alias for -version")
	commandLine.BoolVar(&options.printEnv, "env", options.printEnv, "Print the CX environment (GOROOT, GOPATH, CX version, binary location) and exit")
	commandLine.BoolVar(&options.printAST, "ast", options.printAST, "Print the AST of the source files instead of running them")
	commandLine.BoolVar(&options.tokenizeMode, "lexer", options.tokenizeMode, "Print the lexer tokens of the first source file instead of running it")
	commandLine.BoolVar(&options.tokenizeMode, "l", options.tokenizeMode, "alias for -lexer")
	commandLine.StringVar(&options.lexerOutput, "lexer-output", options.lexerOutput, "File that -lexer writes the tokens to instead of stdout")

	commandLine.BoolVar(&options.replMode, "repl", options.replMode, "Compile the source files, then start a read-eval-print loop instead of running main")
	commandLine.BoolVar(&options.replMode, "r", options.replMode, "alias for -repl")
	commandLine.StringVar(&options.initialHeap, "heap-initial", options.initialHeap, "Initial heap size of the CX virtual machine in bytes. The suffixes G, M and K (or g, m and k) mean gigabytes, megabytes and kilobytes.")
	commandLine.StringVar(&options.initialHeap, "hi", options.initialHeap, "alias for -heap-initial")
	commandLine.StringVar(&options.maxHeap, "heap-max", options.maxHeap, "Maximum heap size of the CX virtual machine, in the units of -heap-initial. A -heap-max below -heap-initial lowers the initial heap to it.")
	commandLine.StringVar(&options.maxHeap, "hm", options.maxHeap, "alias for -heap-max")
	commandLine.StringVar(&options.stackSize, "stack-size", options.stackSize, "Stack size of the CX virtual machine, in the units of -heap-initial")
	commandLine.StringVar(&options.stackSize, "ss", options.stackSize, "alias for -stack-size")
	commandLine.Float64Var(&options.minHeapFreeRatio, "min-heap-free", options.minHeapFreeRatio, "Fraction of the heap, from 0.0 to 1.0, kept free after a garbage collection. The heap grows when less is free. 0 keeps the default of 0.4.")
	commandLine.Float64Var(&options.maxHeapFreeRatio, "max-heap-free", options.maxHeapFreeRatio, "Fraction of the heap, from 0.0 to 1.0, that may be free after a garbage collection. The heap shrinks when more is free. 0 keeps the default of 0.7.")
	commandLine.StringVar(&options.cxpath, "cxpath", options.cxpath, "Workspace directory, used instead of the CXPATH environment variable")
	commandLine.StringVar(&options.sandbox, "sandbox", options.sandbox, "Directory a running program's file opcodes are confined to. Program paths must be relative and stay inside it. Defaults to the working directory.")
	commandLine.StringVar(&options.executor, "executor", options.executor, "Loop that runs the program: auto runs it on the flat vm when every function lowers and on the old loop otherwise, old and vm force one loop. vm fails on a program that does not lower.")
	commandLine.StringVar(&options.enable, "enable", options.enable, "System libraries the program may call, a comma separated list of file, net, process and cipher. A call into a library that is not enabled is the runtime error CX_RUNTIME_LIBRARY_DISABLED. See System libraries in LanguageGuide.md.")
	commandLine.StringVar(&options.netRecord, "net-record", options.netRecord, "Write every batch of the net library to this replay record, see docs/Networking.md")
	commandLine.StringVar(&options.netReplay, "net-replay", options.netReplay, "Run the net library from this replay record and open no socket, see docs/Networking.md")

	// Debug flags
	commandLine.BoolVar(&options.debugFileAccess, "debug-file-access", options.debugFileAccess, "Print each file the compiler opens, creates or stats")
	commandLine.BoolVar(&options.debugFileAccess, "Df", options.debugFileAccess, "alias for -debug-file-access")
	commandLine.IntVar(&options.debugProfile, "debug-profile", options.debugProfile, "Print the time of each compile and run phase, and write CPU and heap profiles of parsing to <cx binary>_parse_cpu.pprof and _parse_mem.pprof at this many CPU samples per second. View them with go tool pprof.")
	commandLine.IntVar(&options.debugProfile, "Dp", options.debugProfile, "alias for -debug-profile")
	commandLine.BoolVar(&options.debugGoStack, "debug-go-stack", options.debugGoStack, "Print the Go stack of a CX runtime error, to debug cx itself")

	commandLine.Parse(args)

}

func printHelp() {
	fmt.Printf(`Usage: cx [options] [source-files]

CX options:
-h, --help                        Prints this message.
-n, --new                         Creates a new project located at $CXPATH/src
-r, --repl                        Loads source files into memory and starts a read-eval-print loop.
-w, --web                         Start CX as a web service.

Notes:
* Option --web makes every other flag to be ignored.
`)
}

func printVersion() {
	fmt.Printf("CX version %v %v/%v\n", VERSION, runtime.GOOS, runtime.GOARCH)
}

func checkHelp(args []string) bool {
	if len(args) > 0 && command(args[0], "help") {
		return true
	}
	return false
}

func checkversion(args []string) bool {
	if len(args) > 0 && command(args[0], "version") {
		return true
	}
	return false
}

func checkenv(args []string) bool {
	if len(args) > 0 && command(args[0], "env") {
		return true
	}
	return false
}

func checkAST(args []string) bool {
	if len(args) > 0 && command(args[0], "ast") {
		return true
	}
	return false
}

func checktokenizeMode(args []string) bool {
	if len(args) > 0 && command(args[0], "tokenize") {
		return true
	}
	return false
}

func printEnv() {
	ex, _ := os.Executable()

	fmt.Println("GOROOT: ", runtime.GOROOT())
	fmt.Println("GOPATH: ", build.Default.GOPATH)
	fmt.Println("GOBIN: ", os.Getenv("GOBIN"))
	fmt.Println("GO version: ", runtime.Version())
	fmt.Println("Operating system: ", runtime.GOOS)
	fmt.Println("CX version: ", VERSION)
	fmt.Println("CX binary location: ", filepath.Dir(ex))
}

// command reports whether arg names the subcommand, with or without leading
// dashes, so a file path containing the name is not taken for it.
func command(arg, name string) bool {
	return strings.TrimLeft(arg, "-") == name
}

// librariesEnabledParse returns the set of system libraries the --enable
// value list names, or the problem with it.
func librariesEnabledParse(list string) (enabled [constants.LibraryCount]bool, problem string) {
	for _, name := range strings.Split(list, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		library, found := constants.LibraryByName(name)
		if !found {
			var names []string
			for _, known := range constants.LibraryTable {
				names = append(names, known.LibraryName)
			}
			return enabled, fmt.Sprintf("unknown library %q in --enable, the libraries are %s", name, strings.Join(names, ", "))
		}
		enabled[library.LibraryNumber] = true
	}
	return enabled, ""
}

// netModeSet sets where the net library takes its batches from, by
// --net-record and --net-replay, and returns a problem when both are given.
func netModeSet(options cxCmdFlags) string {
	cxnet.LibraryNetMode, cxnet.LibraryNetRecordPath = netbatch.NetModeLive, ""
	switch {
	case options.netRecord != "" && options.netReplay != "":
		return "--net-record and --net-replay cannot both be given"
	case options.netRecord != "":
		cxnet.LibraryNetMode, cxnet.LibraryNetRecordPath = netbatch.NetModeRecord, options.netRecord
	case options.netReplay != "":
		cxnet.LibraryNetMode, cxnet.LibraryNetRecordPath = netbatch.NetModeReplay, options.netReplay
	}
	return ""
}
