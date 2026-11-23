// Package core is the minimum interpreter core that cx build puts in front of
// a serialized program: the opcode set it registers, the trailer that marks
// the appended program, and the executors. It uses no module outside the Go
// standard library and has no parser.
package core

import (
	"sort"

	"github.com/skycoin/skycoin-cx/t0-cx-core-assert"
	"github.com/skycoin/skycoin-cx/t1-constants"
	stringpool "github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
	// GCREACH: t3-ast owns the heap and the GC roots. core builds the program and its
	// opcode table and allocates nothing, see docs/GarbageCollector.md.
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-opcodes"
	"github.com/skycoin/skycoin-cx/t5-cx-runtime-printer"
	cxoscore "github.com/skycoin/skycoin-cx/t5-cxos-core"
	"github.com/skycoin/skycoin-cx/t5-regexp"
)

// CoreRegister adds the native packages of the core to prgrm and their
// opcodes to the global tables, and the system libraries that
// libraryRegisters register. It leaves out every other library (t5-cxos,
// t5-tcp) and every opcode that needs a module outside the standard library
// (t5-cipher, t6-cxfx). The built-in opcodes come last, as in cx.
func CoreRegister(prgrm *ast.CXProgram, libraryRegisters ...func(prgrm *ast.CXProgram)) {
	cxoscore.RegisterPackage(prgrm)
	printer.PrinterOpcodesRegister(prgrm)
	printer.PrinterCompileErrorsRegister(prgrm)
	regexp.RegisterPackage(prgrm)
	for _, register := range libraryRegisters {
		register(prgrm)
	}
	opcodes.RegisterOpcodes(prgrm)
}

// CoreProgramMake returns an empty program to register the opcodes in. A
// program to run comes from ast.Deserialize instead.
func CoreProgramMake() *ast.CXProgram {
	state := &ast.CxIrState{}
	state.CxIrStateInit()
	return &ast.CXProgram{
		Packages:       make(map[stringpool.StringGid]ast.CXPackageIndex),
		CurrentPackage: -1,
		Version:        constants.LanguageVersionCurrent,
		CxIrState:      state,
	}
}

// CoreOpcodeSet returns the numbers of the opcodes CoreRegister registers. It
// registers them into the empty global tables and empties them again, so it
// runs before anything else registers an opcode.
func CoreOpcodeSet() map[int]bool {
	assert.AssertPreconditionTrue(len(ast.OpcodeHandlers) == 0 && len(ast.OpNames) == 0, "CoreOpcodeSet after opcodes were registered")
	CoreRegister(CoreProgramMake())
	set := map[int]bool{}
	for code, handler := range ast.OpcodeHandlers {
		if handler != nil {
			set[code] = true
		}
	}
	ast.OpcodeHandlers = nil
	ast.OpNames = map[int]string{}
	ast.OpCodes = map[string]int{}
	ast.Natives = map[int]*ast.CXNativeFunction{}
	ast.Operators = nil
	return set
}

// CoreOpcodesUsed returns the sorted numbers of the opcodes prgrm can call,
// those of its built-in functions.
func CoreOpcodesUsed(prgrm *ast.CXProgram) []int {
	seen := map[int]bool{}
	var used []int
	for index := range prgrm.CXFunctions {
		code := prgrm.CXFunctions[index].AtomicOPCode
		if code != 0 && !seen[code] {
			seen[code] = true
			used = append(used, code)
		}
	}
	sort.Ints(used)
	return used
}

// CoreOpcodesMissing returns the names of the opcodes prgrm uses that the
// global tables have no handler for, sorted by number.
func CoreOpcodesMissing(prgrm *ast.CXProgram) []string {
	var missing []string
	for _, code := range CoreOpcodesUsed(prgrm) {
		if code >= len(ast.OpcodeHandlers) || ast.OpcodeHandlers[code] == nil {
			missing = append(missing, CoreOpcodeName(code))
		}
	}
	return missing
}

// CoreOpcodeName returns the name of opcode number code in
// constants.OpcodeLifecycleTable.
func CoreOpcodeName(code int) string {
	row, found := constants.OpcodeLifecycleLiveByNumber(code)
	assert.AssertPreconditionTrue(found, "opcode number without a live row")
	return row.Name
}

// CoreLibrariesEnabledResolve applies constants.LibraryPolicyEnabledWhenImported
// to prgrm: when it is true, every library the program calls is enabled.
func CoreLibrariesEnabledResolve(prgrm *ast.CXProgram) {
	if !constants.LibraryPolicyEnabledWhenImported {
		return
	}
	for library, used := range prgrm.ProgramLibrariesUsed {
		if used {
			prgrm.ProgramLibrariesEnabled[library] = true
		}
	}
}

// CoreLibrariesMissing returns the names of the libraries enabled for prgrm
// that this binary does not link, in number order.
func CoreLibrariesMissing(prgrm *ast.CXProgram) []string {
	var missing [constants.LibraryCount]bool
	for library, enabled := range prgrm.ProgramLibrariesEnabled {
		missing[library] = enabled && !ast.LibraryLinked(library)
	}
	return CoreLibraryNames(missing)
}

// CoreLibraryNames returns the names of the libraries libraries marks, in
// number order.
func CoreLibraryNames(libraries [constants.LibraryCount]bool) []string {
	var names []string
	for library, marked := range libraries {
		if marked {
			names = append(names, constants.LibraryNameOf(library))
		}
	}
	return names
}
