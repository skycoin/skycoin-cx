package main

import (
	"fmt"
	"os"
	"sort"

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

// buildLlvm compiles the parsed program to LLVM IR at output. A program
// outside the vm subset is refused with the message vm gives.
func buildLlvm(options buildOptions) int {
	prgrm := actions.AST
	core.CoreLibrariesEnabledResolve(prgrm)
	prgrm.CxIrState.CxIrStateManagerGrowBoundary()
	prgrm.CxIrState.StringPool.StringPoolFreeze()
	image, _, outside := core.ExecutorLower(prgrm)
	if outside != "" {
		fmt.Fprintf(os.Stderr, "cx build: cannot run on vm: %s\n", outside)
		return types.CX_RUNTIME_NOT_IMPLEMENTED
	}
	module, stats, err := llvm.Emit(&image, buildLlvmOptions())
	if err != nil {
		fmt.Fprintf(os.Stderr, "cx build: cannot build for llvm: %v\n", err)
		return types.CX_RUNTIME_NOT_IMPLEMENTED
	}
	if options.buildOptionsStats {
		buildLlvmStatsPrint(stats)
	}
	if err := os.WriteFile(options.buildOptionsOutput, module, 0o644); err != nil {
		return buildFail(err.Error())
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
