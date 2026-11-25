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

	"github.com/skycoin/skycoin-cx/t0-cxhash"
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	"github.com/skycoin/skycoin-cx/t3-ast"
	llvm "github.com/skycoin/skycoin-cx/t3-cx-compiler-llvm"
	vm "github.com/skycoin/skycoin-cx/t3-cx-runtime-vm"
	"github.com/skycoin/skycoin-cx/t4-actions"
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
	llvm.StopDivideByZero:   vm.StopDivideByZero,
	llvm.StopStackOverflow:  vm.StopStackOverflow,
	llvm.StopIndexNegative:  vm.StopIndexNegative,
	llvm.StopAddressInvalid: vm.StopAddressInvalid,
}

// buildLlvmOptions gives the emitted code the exit codes and messages of vm.
func buildLlvmOptions() llvm.Options {
	options := llvm.Options{AssertCode: types.CX_ASSERT}
	for stop, vmStop := range buildLlvmStops {
		code := vm.StopExitCodes[vmStop]
		options.StopCodes[stop] = code
		options.StopLines[stop] = ast.ErrorString(int(code)) + ", " + printer.PrinterVmStopMessage(vmStop)
	}
	return options
}

// buildLlvm compiles the parsed program to LLVM IR, decision D9. An output
// ending in .ll gets the IR alone. Any other output is a native executable,
// which clang compiles from the IR, written beside it with .ll in place of
// .exe, and the C runtime. A program outside the vm subset is refused with
// the message vm gives.
func buildLlvm(options buildOptions) int {
	prgrm := actions.AST
	core.CoreLibrariesEnabledResolve(prgrm)
	prgrm.CxIrState.CxIrStateManagerGrowBoundary()
	prgrm.CxIrState.StringPool.StringPoolFreeze()
	llvmOptions := buildLlvmOptions()
	llvmOptions.Debug = options.buildOptionsDebug
	image, _, outside := core.ExecutorLower(prgrm)
	if code := core.CoreOperationRefused(prgrm); code != 0 {
		// The program is refused before it runs, as cx refuses it, so its
		// code needs no native form.
		llvmOptions.RefusalLine, llvmOptions.RefusalCode = core.CoreOperationRefusalLine(prgrm, code), types.CX_RUNTIME_LIBRARY_DISABLED
		image, outside = ir.Image{ImageEntryInit: ir.ImageFuncNone, ImageEntryMain: ir.ImageFuncNone}, ""
	}
	if outside != "" {
		fmt.Fprintf(os.Stderr, "cx build: cannot run on vm: %s\n", outside)
		return types.CX_RUNTIME_NOT_IMPLEMENTED
	}
	module, stats, err := llvm.Emit(&image, llvmOptions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cx build: cannot build for llvm: %v\n", err)
		return types.CX_RUNTIME_NOT_IMPLEMENTED
	}
	if options.buildOptionsStats {
		buildLlvmStatsPrint(stats)
	}
	output := options.buildOptionsOutput
	if strings.HasSuffix(output, buildLlvmExtension) {
		if err := os.WriteFile(output, module, 0o644); err != nil {
			return buildFail(err.Error())
		}
		return types.CX_SUCCESS
	}
	modulePath := strings.TrimSuffix(output, ".exe") + buildLlvmExtension
	if err := os.WriteFile(modulePath, module, 0o644); err != nil {
		return buildFail(err.Error())
	}
	return buildLlvmNative(modulePath, output, options.buildOptionsDebug)
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
func buildLlvmNative(modulePath string, output string, debug bool) int {
	clang := buildLlvmClang()
	if clang == "" {
		return buildFail(fmt.Sprintf("wrote %s but found no clang to compile it into %s, install clang, set %s or give an output ending in .ll", modulePath, output, buildLlvmClangEnv))
	}
	directory, err := os.MkdirTemp("", "cx-llvm-")
	if err != nil {
		return buildFail(err.Error())
	}
	defer os.RemoveAll(directory)
	runtimePath, err := buildLlvmRuntime(clang, directory)
	if err != nil {
		return buildFail(err.Error())
	}
	args := []string{"-O2", "-w", "-o", output, modulePath, runtimePath}
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
	return types.CX_SUCCESS
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
func buildLlvmRuntime(clang string, directory string) (string, error) {
	source := filepath.Join(directory, llvm.RuntimeFileName)
	if err := os.WriteFile(source, llvm.RuntimeC, 0o644); err != nil {
		return "", err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return source, nil
	}
	hash := cxhash.CxHashOfBytes(append(append([]byte(clang), 0), llvm.RuntimeC...))
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
	command := exec.Command(clang, "-O2", "-w", "-c", "-o", partial, source)
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
