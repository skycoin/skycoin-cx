// Command cx-core is the minimum interpreter core of cx build. It runs the
// serialized program appended to its own file and passes its command line
// arguments to the program as os.Args. It has no parser and loads no source.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t0-util"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t5-cx-runtime-printer" // thin: only PrinterCompileErrorsRegister; no depth gain
	core "github.com/skycoin/skycoin-cx/t6-cx-runtime-core"
)

// coreLibraryRegisters register the system libraries linked into this core.
// The library_*.go files add them by build tag, cxcorelibraryfile,
// cxcorelibrarynet, cxcorelibraryprocess and cxcorelibrarycipher.
var coreLibraryRegisters []func(prgrm *ast.CXProgram)

func main() {
	os.Exit(int(coreRun(os.Args[1:])))
}

// coreRun loads the appended program and runs it with args as os.Args. It
// returns the exit code.
func coreRun(args []string) int32 {
	runtime.LockOSThread()
	runtime.GOMAXPROCS(2)
	core.ExecutorMemoryLimitsClamp()

	path, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cx-core: cannot find this executable: %s\n", err)
		return types.CX_COMPILATION_ERROR
	}
	program, fileError := core.CoreTrailerRead(path)
	if fileError != nil {
		fmt.Fprintf(os.Stderr, "cx-core: %s: %s\n", fileError.FileErrorPath, fileError.FileErrorMessage)
		return types.CX_COMPILATION_ERROR
	}
	if program == nil {
		fmt.Fprintln(os.Stderr, "cx-core: no CX program is appended to this file, make one with cx build")
		return types.CX_COMPILATION_ERROR
	}

	core.CoreRegister(core.CoreProgramMake(), coreLibraryRegisters...)
	prgrm, err := ast.Deserialize(program, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cx-core: %s\n", err)
		return types.CX_COMPILATION_ERROR
	}
	if missing := core.CoreOpcodesMissing(prgrm); len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "cx-core: the program uses opcodes this core does not include: %s\n", strings.Join(missing, ", "))
		return types.CX_COMPILATION_ERROR
	}
	core.CoreLibrariesEnabledResolve(prgrm)
	if missing := core.CoreLibrariesMissing(prgrm); len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "cx-core: the program enables system libraries this core does not link: %s\n", strings.Join(missing, ", "))
		return types.CX_COMPILATION_ERROR
	}
	if prgrm.ProgramLibrariesEnabled[constants.LibraryNumberFile] {
		// The file library resolves paths in the working directory, as cx
		// does without --sandbox. The core has no flags to name another.
		root, fileError := util.ProgramSandboxRootOpen("")
		if fileError != nil {
			fmt.Fprintf(os.Stderr, "cx-core: cannot open the sandbox: %s\n", fileError.FileErrorMessage)
			return types.CX_COMPILATION_ERROR
		}
		defer root.Close()
		prgrm.ProgramSandboxRoot = root
	}
	printer.PrinterCompileErrorsRegister(prgrm)
	prgrm.ProgramStdinInit(os.Stdin)
	return core.ExecutorRun(prgrm, core.ExecutorAuto, args)
}
