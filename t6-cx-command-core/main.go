// Command cx-core is the minimum interpreter core of cx build. It runs the
// serialized program appended to its own file and passes its command line
// arguments to the program as os.Args. A core with no program appended, as on
// js/wasm, runs cx-core --program FILE args. It has no parser and loads no
// source.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t0-cx-library-terminal"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	sandbox "github.com/skycoin/skycoin-cx/t1-cx-runtime-sandbox"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
	"github.com/skycoin/skycoin-cx/t4-cx-library-reflectopcodes"
	printer "github.com/skycoin/skycoin-cx/t4-cx-runtime-printer" // thin: only CompileErrorsRegister; no depth gain
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
)

// coreLibraryRegisters register the system libraries linked into this core.
// The library_*.go files add them by build tag, cxcorelibraryfile,
// cxcorelibrarynet, cxcorelibraryprocess and cxcorelibrarycipher.
var coreLibraryRegisters []func(prgrm *ast.CXProgram)

// coreLibraryOpcodesLinked marks the libraries whose functions are opcodes
// that this core links, by the same build tags (decision D119).
var coreLibraryOpcodesLinked = [opcodetable.LibraryCount]bool{
	opcodetable.LibraryNumberCipher:   coreLibraryCipherLinked,
	opcodetable.LibraryNumberNet:      coreLibraryNetLinked,
	opcodetable.LibraryNumberFile:     coreLibraryFileLinked,
	opcodetable.LibraryNumberTerminal: true,
	opcodetable.LibraryNumberReflect:  true,
}

// coreOpcodeDispatch is the opcode switch of this core, the switch of
// t5-cx-runtime-core and then the switches of the libraries it links.
func coreOpcodeDispatch(prgrm *ast.CXProgram, code int, inputs []ast.CXValue, outputs []ast.CXValue) bool {
	return core.CoreOpcodeDispatch(prgrm, code, inputs, outputs) || coreOpcodeDispatchCipher(prgrm, code, inputs, outputs) || coreOpcodeDispatchNet(prgrm, code, inputs, outputs) || coreOpcodeDispatchFile(prgrm, code, inputs, outputs) || reflectopcodes.ReflectOpcodeDispatch(prgrm, code, inputs, outputs)
}

func main() {
	os.Exit(int(coreRun(os.Args[1:])))
}

// coreRun loads the appended program and runs it with args as os.Args. It
// returns the exit code.
func coreRun(args []string) int32 {
	runtime.LockOSThread()
	runtime.GOMAXPROCS(2)
	core.ExecutorMemoryLimitsClamp()

	program, args, problem := coreProgramLoad(args)
	if problem != "" {
		fmt.Fprintln(os.Stderr, "cx-core:", problem)
		return valuetypes.CXCompilationError
	}

	core.Register(core.ProgramMake(), coreLibraryRegisters...)
	ast.OpcodeDispatch = coreOpcodeDispatch
	prgrm, err := ast.Deserialize(program, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cx-core: %s\n", err)
		return valuetypes.CXCompilationError
	}
	if missing := core.OpcodesMissing(prgrm); len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "cx-core: the program uses opcodes this core does not include: %s\n", strings.Join(missing, ", "))
		return valuetypes.CXCompilationError
	}
	core.LibrariesEnabledResolve(prgrm)
	if missing := core.LibrariesMissing(prgrm, coreLibraryOpcodesLinked); len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "cx-core: the program enables system libraries this core does not link: %s\n", strings.Join(missing, ", "))
		return valuetypes.CXCompilationError
	}
	if core.LibrariesActive(prgrm)[opcodetable.LibraryNumberFile] {
		// The file library resolves paths in the working directory, as cx
		// does without --sandbox. The core has no flags to name another.
		root, fileError := sandbox.RootOpen("")
		if fileError != nil {
			fmt.Fprintf(os.Stderr, "cx-core: cannot open the sandbox: %s\n", fileError.FileErrorMessage)
			return valuetypes.CXCompilationError
		}
		defer root.Close()
		prgrm.ProgramSandboxRoot = root
	}
	printer.CompileErrorsRegister(prgrm)
	prgrm.ProgramStdinInit(os.Stdin)
	terminal.ArgsSet(args)
	return core.ExecutorRun(prgrm, core.ExecutorAuto, args)
}

// coreProgramFileFlag names a program file for a core with no program
// appended, cx-core --program FILE args, for hosts without os.Executable
// such as js/wasm. FILE is a program made with cx build.
const coreProgramFileFlag = "--program"

// coreProgramLoad returns the serialized program and the arguments of the
// program. The program appended to this executable comes first and gets
// every argument. Without one, --program FILE names it. problem says why
// there is no program.
func coreProgramLoad(args []string) (program []byte, programArgs []string, problem string) {
	path, err := os.Executable()
	if err == nil {
		appended, fileError := core.TrailerRead(path)
		if fileError == nil && appended != nil {
			return appended, args, ""
		}
		if fileError != nil && (len(args) < 2 || args[0] != coreProgramFileFlag) {
			return nil, nil, fileError.FileErrorPath + ": " + fileError.FileErrorMessage
		}
	}
	if len(args) < 2 || args[0] != coreProgramFileFlag {
		if err != nil {
			return nil, nil, "cannot find this executable: " + err.Error() + ", name the program with " + coreProgramFileFlag + " FILE"
		}
		return nil, nil, "no CX program is appended to this file, make one with cx build or name one with " + coreProgramFileFlag + " FILE"
	}
	built, fileError := core.TrailerRead(args[1])
	if fileError != nil {
		return nil, nil, fileError.FileErrorPath + ": " + fileError.FileErrorMessage
	}
	if built == nil {
		return nil, nil, args[1] + " holds no CX program, make one with cx build"
	}
	return built, args[2:], ""
}
