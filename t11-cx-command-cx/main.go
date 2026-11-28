package main

import (
	"fmt"
	"os"
	"runtime"

	sourceargs "github.com/skycoin/skycoin-cx/t0-cx-compiler-sourceargs"
	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t0-cx-library-terminal"
	filelog "github.com/skycoin/skycoin-cx/t0-cx-runtime-filelog"
	"github.com/skycoin/skycoin-cx/t0-cx-runtime-output"
	sandbox "github.com/skycoin/skycoin-cx/t1-cx-runtime-sandbox"

	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t1-cx-runtime-profiling"
	"github.com/skycoin/skycoin-cx/t10-cx-tools-repl"      // thin: only Repl, TargetFunction; only user of cxrepl, could merge it here, max tier 11 to 10
	"github.com/skycoin/skycoin-cx/t2-cx-core-environment" // thin: only DebugGolangStackTrace; no depth gain
	"github.com/skycoin/skycoin-cx/t4-cx-compiler-astbuild"
	"github.com/skycoin/skycoin-cx/t4-cx-library-compileropcodes"
	"github.com/skycoin/skycoin-cx/t4-cx-library-netopcodes"  // thin: only ConstantsInit; no depth gain
	"github.com/skycoin/skycoin-cx/t5-cx-library-fileopcodes" // thin: only ConstantsInit; no depth gain
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"   // thin: only ExecutorRun, ExecutorMemoryLimitsClamp and the executor names; no depth gain
	"github.com/skycoin/skycoin-cx/t6-cx-library-graphics"    // thin: only ConstantsInit; no depth gain
	"github.com/skycoin/skycoin-cx/t7-cx-runtime-libraries"   // thin: only MakeProgram, RegisterPackages; no depth gain
)

func main() {
	Run(os.Args[1:])
}

func Run(args []string) {
	defer operationCountPrint()
	compileropcodes.LibraryCompilerSet(compilerLibrary{})
	if len(args) > 0 && args[0] == buildCommand {
		os.Exit(buildRun(args[1:]))
	}
	options, cxArgs, sourceCode, fileNames, start := runStart(args)
	if !start {
		return
	}

	if run := parseProgram(options, fileNames, sourceCode); run {

		if options.printAST || checkAST(args) {
			printProgramAST(options, cxArgs, sourceCode)
			return
		}

		if options.tokenizeMode {
			printTokenize(options, fileNames)
			return
		}

		//if strings.Contains(args[0], "tokenize") {
		if checktokenizeMode(args) {
			printTokenize(options, fileNames)
			return
		}

		runProgram(options, cxArgs, sourceCode)
	}
}

func runProgram(options cxCmdFlags, cxArgs []string, sourceCode []*os.File) {
	root, fileError := sandbox.RootOpen(options.sandbox)
	if fileError != nil {
		fmt.Fprintf(os.Stderr, "cannot open the sandbox: %s\n", fileError.FileErrorMessage)
		os.Exit(valuetypes.CXCompilationError)
	}
	defer root.Close()
	astbuild.AST.ProgramSandboxRoot = root
	astbuild.AST.ProgramStdinInit(os.Stdin)
	terminal.ArgsSet(cxArgs)

	profiling.ProfileStart("run")

	defer profiling.ProfileStop("run")

	if options.executor != core.ExecutorAuto && options.executor != core.ExecutorOld && options.executor != core.ExecutorVm {
		fmt.Fprintf(os.Stderr, "unknown --executor %q, want auto, old or vm\n", options.executor)
		os.Exit(valuetypes.CXCompilationError)
	}

	if options.replMode || len(sourceCode) == 0 {
		repl.Repl()
		return
	}

	if code := core.ExecutorRun(astbuild.AST, options.executor, cxArgs); code != valuetypes.CXSuccess {
		os.Exit(int(code))
	}
}

// runStart reads the flags and sets up the program state for args, the part
// of Run before parsing. It returns false when there is nothing to run.
func runStart(args []string) (options cxCmdFlags, cxArgs []string, sourceCode []*os.File, fileNames []string, start bool) {
	runtime.LockOSThread()

	runtime.GOMAXPROCS(2)

	options = defaultCmdFlags()

	parseFlags(&options, args)

	/*
		Checking if CXPATH is set, either by setting an environment variable
		 or by setting the `--cxpath` flag.
	*/
	if err := CXPathSet(options); err != nil {
		fmt.Fprintln(os.Stderr, "cx:", err)
	}

	/*
		checkHelp checks  for command line argument string "help"
		if exits print help
		$cx help
	*/
	if checkHelp(args) {
		commandLine.PrintDefaults()
		return options, nil, nil, nil, false
	}

	/*
		options.printHelp checks for flags string "help"
		if exits print help
		$cx --help
		$cx -help
	*/
	if options.printHelp {
		printHelp()
		return options, nil, nil, nil, false
	}

	/*
		options.printVersion checks for flags string "version"
		if exits print cx version
		$cx --version
		$cx -v
	*/
	if options.printVersion {
		printVersion()
		return options, nil, nil, nil, false
	}

	/*
		checkversion checks  for command line argument string "version"
		if exits print version
		$cx version
	*/
	if checkversion(args) {
		printVersion()
		return options, nil, nil, nil, false
	}

	/*
		options.printEnv checks for flags string "env"
		if exits print cx env
		$cx --env
	*/
	if options.printEnv {
		printEnv()
		return options, nil, nil, nil, false
	}

	/*
		checkenv checks  for command line argument string "env"
		if exits print env
		$cx env
	*/
	if checkenv(args) {
		printEnv()
		return options, nil, nil, nil, false
	}

	/*
		options.debugFileAccess prints open/close file events"
		$cx --debug-file-access
		$cx -Df
	*/
	environment.DebugGolangStackTrace = options.debugGoStack
	if options.debugFileAccess {
		filelog.Enable(true)
	}

	/*
		options.initialHeap checks for flags string "heap-initial"
		$cx --heap-initial
	*/
	if options.initialHeap != "" {
		opcodetable.HeapSizeInit = parseMemoryString(options.initialHeap)
	}

	/*
		options.maxHeap checks for flags string "maxHeap"
		$cx --maxHeap
	*/
	if options.maxHeap != "" {
		opcodetable.MaxHeapSize = parseMemoryString(options.maxHeap)
		if opcodetable.MaxHeapSize < opcodetable.HeapSizeInit {
			// Then MaxHeapSize overrides HeapSizeInit's value.
			opcodetable.HeapSizeInit = opcodetable.MaxHeapSize
		}
	}

	if options.stackSize != "" {
		opcodetable.StackSize = parseMemoryString(options.stackSize)
	}
	// Memory stays below constants.EscapeOffsetFlag.
	core.ExecutorMemoryLimitsClamp()

	/*
		options.minHeapFreeRatio checks for flags string "--min-heap-free"
		$cx --min-heap-free
	*/
	if options.minHeapFreeRatio != float64(0) {
		opcodetable.MinHeapFreeRatio = float32(options.minHeapFreeRatio)
	}

	/*
		options.maxHeapFreeRatio checks for flags string "--max-heap-free"
		$cx --max-heap-free
	*/
	if options.maxHeapFreeRatio != float64(0) {
		opcodetable.MaxHeapFreeRatio = float32(options.maxHeapFreeRatio)
	}

	// options, file pointers, filenames
	cxArgs, sourceCode, fileNames, err := sourceargs.Parse(commandLine.Args(), true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(valuetypes.CXCompilationError)
	}

	// Propagate some options out to other packages.
	profiling.DebugProfileRate = options.debugProfile

	profiling.DebugProfile = profiling.DebugProfileRate > 0

	// Init 4-9 run here, after the flags, so the heap flags size the program.
	// Init 1-3 are package vars (see the VIOLATION comments).
	// Init 4: cx.* constants in actions.ConstCodes
	astbuild.ConstantsInit()
	// Init 5: os.* constants (empty with nocxos)
	fileopcodes.ConstantsInit(astbuild.ConstI32Add)
	netopcodes.ConstantsInit(astbuild.ConstI32Add)
	// Init 6: gl, glfw and al constants (empty without cxfx)
	graphics.ConstantsInit()
	// Init 7: actions.AST, the program state
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Also state.CXProgramMng.CXProgramRelease(gid), phase 4 #371. Site S0062 S0063.
	astbuild.AST = libraries.ProgramMake()
	// Init 8: native packages in the program and the global opcode tables
	libraries.PackagesRegister(astbuild.AST)
	// Init 9: built-in opcode and operator tables

	enabled, operations, problem := librariesEnabledParse(options.enable)
	if problem == "" {
		problem = librariesDisabledApply(options.disable, &enabled, &operations)
	}
	if problem != "" {
		fmt.Fprintln(os.Stderr, problem)
		os.Exit(valuetypes.CXCompilationError)
	}
	astbuild.AST.ProgramLibrariesEnabled = enabled
	astbuild.AST.ProgramOperationsEnabled = operations
	mode, found := output.ModeByName(options.outputMode)
	if !found {
		fmt.Fprintf(os.Stderr, "unknown --output-mode %q, want batch, instant or thread\n", options.outputMode)
		os.Exit(valuetypes.CXCompilationError)
	}
	astbuild.AST.ProgramOutputMode = mode
	if problem := netModeSet(options); problem != "" {
		fmt.Fprintln(os.Stderr, problem)
		os.Exit(valuetypes.CXCompilationError)
	}

	return options, cxArgs, sourceCode, fileNames, true
}
