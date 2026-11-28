package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-cx-core-digest"
	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t0-cx-runtime-output"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	llvm "github.com/skycoin/skycoin-cx/t3-cx-compiler-llvm"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-cx-compiler-astbuild"
	"github.com/skycoin/skycoin-cx/t4-cx-runtime-printer"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
)

// The --target values of cx build. core appends the program to a core
// binary, llvm compiles its image to LLVM IR, decisions D8 and D9.
const (
	buildTargetCore = "core"
	buildTargetLlvm = "llvm"
)

// buildLlvmStops maps each stop of the emitted code to the vm stop it
// copies.
var buildLlvmStops = [llvm.StopCount]vm.Stop{
	llvm.StopDivideByZero:    vm.StopDivideByZero,
	llvm.StopStackOverflow:   vm.StopStackOverflow,
	llvm.StopIndexNegative:   vm.StopIndexNegative,
	llvm.StopAddressInvalid:  vm.StopAddressInvalid,
	llvm.StopIndexOutOfRange: vm.StopIndexOutOfRange,
}

// buildLlvmOptions gives the emitted code the exit codes and messages of vm.
func buildLlvmOptions() llvm.Options {
	options := llvm.Options{AssertCode: valuetypes.CXAssert}
	for stop, vmStop := range buildLlvmStops {
		code := vm.StopExitCodes[vmStop]
		options.StopCodes[stop] = code
		options.StopLines[stop] = ast.ErrorString(int(code)) + ", " + printer.VmStopMessage(vmStop)
	}
	return options
}

// buildLlvm compiles the parsed program to LLVM IR, decision D9. An output
// ending in .ll gets the IR alone. Any other output is a native executable,
// which clang compiles from the IR, written beside it with .ll in place of
// .exe, and the C runtime. A program outside the vm subset is refused with
// the message vm gives.
func buildLlvm(options buildOptions) int {
	prgrm := astbuild.AST
	core.LibrariesEnabledResolve(prgrm)
	prgrm.CxIrState.CxIrStateManagerGrowBoundary()
	prgrm.CxIrState.StringPool.StringPoolFreeze()
	llvmOptions := buildLlvmOptions()
	llvmOptions.Debug = options.buildOptionsDebug
	linker, problem := buildLlvmLinkerChoose(options)
	if problem != "" {
		return buildFail(problem)
	}
	tinygoLink := linker == buildLlvmLinkerTinygo && !strings.HasSuffix(options.buildOptionsOutput, buildLlvmExtension)
	if tinygoLink {
		llvmOptions.EntryName = buildLlvmTinygoEntryName
	}
	image, _, outside := core.ExecutorLower(prgrm)
	if code := core.OperationRefused(prgrm); code != 0 {
		// The program is refused before it runs, as cx refuses it, so its
		// code needs no native form.
		llvmOptions.RefusalLine, llvmOptions.RefusalCode = core.OperationRefusalLine(prgrm, code), valuetypes.CXRuntimeLibraryDisabled
		image, outside = ir.Image{ImageEntryInit: ir.ImageFuncNone, ImageEntryMain: ir.ImageFuncNone}, ""
	}
	if outside != "" {
		fmt.Fprintf(os.Stderr, "cx build: cannot run on vm: %s\n", outside)
		return valuetypes.CXRuntimeNotImplemented
	}
	if tinygoLink && image.ImageOperationsEnabled.OpcodeSetLibraryAny(opcodetable.LibraryNumberNet) {
		return buildFail("the program uses the net library, which --llvm-linker=tinygo leaves out")
	}
	module, stats, err := llvm.Emit(&image, llvmOptions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cx build: cannot build for llvm: %v\n", err)
		return valuetypes.CXRuntimeNotImplemented
	}
	if options.buildOptionsStats {
		buildLlvmStatsPrint(stats)
	}
	output := options.buildOptionsOutput
	if strings.HasSuffix(output, buildLlvmExtension) {
		if err := os.WriteFile(output, module, 0o644); err != nil {
			return buildFail(err.Error())
		}
		return valuetypes.CXSuccess
	}
	modulePath := strings.TrimSuffix(output, ".exe") + buildLlvmExtension
	if err := os.WriteFile(modulePath, module, 0o644); err != nil {
		return buildFail(err.Error())
	}
	if tinygoLink {
		return buildLlvmTinygoNative(modulePath, output, options.buildOptionsLlvmTinygoTarget, options.buildOptionsDebug)
	}
	return buildLlvmNative(modulePath, output, options.buildOptionsDebug, buildLlvmOutputThread(image.ImageOutputMode))
}

// buildLlvmExtension ends the file name of LLVM IR.
const buildLlvmExtension = ".ll"

// buildLlvmClangEnv names the C compiler that compiles LLVM IR, clang when
// it is unset.
const buildLlvmClangEnv = "CX_CLANG"

// buildLlvmClang returns the clang cx build runs, or "" when there is none.
func buildLlvmClang() string {
	clang := os.Getenv(buildLlvmClangEnv)
	if clang == "" {
		clang = "clang"
	}
	path, err := exec.LookPath(clang)
	if err != nil {
		return ""
	}
	return path
}

// buildLlvmNative compiles the module at modulePath and links it with the
// C runtime into the executable output.
func buildLlvmNative(modulePath string, output string, debug bool, thread bool) int {
	clang := buildLlvmClang()
	if clang == "" {
		return buildFail(fmt.Sprintf("wrote %s but found no clang to compile it into %s, install clang or TinyGo, set %s or give an output ending in .ll", modulePath, output, buildLlvmClangEnv))
	}
	directory, err := os.MkdirTemp("", "cx-llvm-")
	if err != nil {
		return buildFail(err.Error())
	}
	defer os.RemoveAll(directory)
	var defines []string
	if thread && runtime.GOOS != "windows" {
		// The writing thread of thread mode, see runtime.c.txt. Windows
		// writes as instant mode.
		defines = append(defines, "-DCX_RT_OUTPUT_THREAD", "-pthread")
	}
	runtimePath, err := buildLlvmRuntime(clang, directory, defines)
	if err != nil {
		return buildFail(err.Error())
	}
	args := append([]string{"-O2", "-w", "-o", output, modulePath, runtimePath}, defines...)
	if debug {
		args = append(args, "-g")
	}
	if runtime.GOOS != "windows" {
		// The MSVC C library holds the math functions, elsewhere libm does.
		args = append(args, "-lm")
	} else {
		// Winsock holds the sockets of the net library.
		args = append(args, "-lws2_32")
	}
	command := exec.Command(clang, args...)
	command.Stdout, command.Stderr = os.Stderr, os.Stderr
	if err := command.Run(); err != nil {
		return buildFail(fmt.Sprintf("%s: %v", clang, err))
	}
	return valuetypes.CXSuccess
}

// buildLlvmStatsPrint lists each scalar opcode of the module by how it was
// emitted, inline or as a runtime call, then the totals.
func buildLlvmStatsPrint(stats llvm.Stats) {
	seen := map[ir.Opcode]bool{}
	for op := range stats.StatsInline {
		seen[op] = true
	}
	for op := range stats.StatsFallback {
		seen[op] = true
	}
	ops := make([]int, 0, len(seen))
	for op := range seen {
		ops = append(ops, int(op))
	}
	sort.Ints(ops)
	inline, fallback := 0, 0
	for _, op := range ops {
		o := ir.Opcode(op)
		inline += stats.StatsInline[o]
		fallback += stats.StatsFallback[o]
		fmt.Fprintf(os.Stderr, "llvm %s inline %d runtime %d\n", llvm.OpcodeName(o), stats.StatsInline[o], stats.StatsFallback[o])
	}
	fmt.Fprintf(os.Stderr, "llvm scalar inline %d runtime %d, other runtime calls %d, frame regions alloca %d memory %d\n",
		inline, fallback, stats.StatsRuntime, stats.StatsAlloca, stats.StatsShadow)
}

// buildLlvmRuntime returns the C runtime compiled by clang, an object kept in
// the user cache under the hash of the runtime and the path of clang, so
// only the first native build compiles it. Without a cache it returns the C
// source written to directory, which clang then compiles with the module.
func buildLlvmRuntime(clang string, directory string, defines []string) (string, error) {
	source := filepath.Join(directory, llvm.RuntimeFileName)
	if err := os.WriteFile(source, llvm.RuntimeC, 0o644); err != nil {
		return "", err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return source, nil
	}
	key := append([]byte(clang), 0)
	for _, define := range defines {
		key = append(append(key, define...), 0)
	}
	hash := digest.CxHashOfBytes(append(key, llvm.RuntimeC...))
	object := filepath.Join(cache, "cx", "llvm-runtime-"+hex.EncodeToString(hash[:8])+".o")
	if _, err := os.Stat(object); err == nil {
		return object, nil
	}
	if err := os.MkdirAll(filepath.Dir(object), 0o755); err != nil {
		return source, nil
	}
	// Each build compiles to its own file and renames it into place, so
	// parallel builds never read a half written object.
	partialFile, err := os.CreateTemp(filepath.Dir(object), "llvm-runtime-*.o")
	if err != nil {
		return source, nil
	}
	partial := partialFile.Name()
	partialFile.Close()
	command := exec.Command(clang, append(append([]string{"-O2", "-w", "-c"}, defines...), "-o", partial, source)...)
	command.Stdout, command.Stderr = os.Stderr, os.Stderr
	if err := command.Run(); err != nil {
		os.Remove(partial)
		return "", fmt.Errorf("%s: compiling the runtime: %v", clang, err)
	}
	if err := os.Rename(partial, object); err != nil {
		os.Remove(partial)
		return source, nil
	}
	return object, nil
}

// buildLlvmOutputThread reports whether mode writes the output from a
// second thread, which a native program starts (decision D142).
func buildLlvmOutputThread(mode output.Mode) bool {
	return mode == output.ModeThread
}
