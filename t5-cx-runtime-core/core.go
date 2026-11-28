// Package core is the minimum interpreter core that cx build puts in front of
// a serialized program: the opcode set it links, the trailer that marks the
// appended program, and the executors. It uses no module outside the Go
// standard library and has no parser. A core that links the cipher library
// links the cipher package of Skycoin with it.
package core

import (
	"sort"

	"github.com/skycoin/skycoin-cx/t0-cx-core-assert"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	stringpool "github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
	// GCREACH: t3-cx-core-ast owns the heap and the GC roots. core builds the program and
	// allocates nothing, see docs/GarbageCollector.md.
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
	"github.com/skycoin/skycoin-cx/t4-cx-library-reflectopcodes"
	"github.com/skycoin/skycoin-cx/t4-cx-library-regexp"
	"github.com/skycoin/skycoin-cx/t4-cx-runtime-printer"
)

// Register adds the native packages of the core to prgrm, registers the
// system libraries that libraryRegisters register and sets the opcode switch
// to CoreOpcodeDispatch. It leaves out every other library (t5-cx-library-fileopcodes,
// t4-cx-library-netopcodes, t4-cx-library-cipher) and the opcodes that need a module outside the
// standard library (t6-cx-library-graphics).
func Register(prgrm *ast.CXProgram, libraryRegisters ...func(prgrm *ast.CXProgram)) {
	ast.OpcodeDispatch = CoreOpcodeDispatch
	printer.CompileErrorsRegister(prgrm)
	regexp.PackageRegister(prgrm)
	reflectopcodes.PackageRegister(prgrm)
	for _, register := range libraryRegisters {
		register(prgrm)
	}
}

// ProgramMake returns an empty program to register the libraries in. A
// program to run comes from ast.Deserialize instead.
func ProgramMake() *ast.CXProgram {
	state := &ast.CxIrState{}
	state.CxIrStateInit()
	return &ast.CXProgram{
		Packages:       make(map[stringpool.StringGid]ast.CXPackageIndex),
		CurrentPackage: -1,
		Version:        opcodetable.LanguageVersionCurrent,
		CxIrState:      state,
	}
}

// OpcodeLinked reports whether the core runs opcode code: LibraryCall,
// which the loop runs, or an opcode with a handler outside t6-cx-library-graphics that the
// build tags link. The core is built with the tags of cx, so cx can ask.
func OpcodeLinked(code int) bool {
	if code == opcodetable.OpLibraryCall {
		return true
	}
	return opcodetable.OpcodeHandled(code) && opcodetable.OpcodeLinked(code) && opcodetable.OpcodeSignatureOf(code).OpcodeSignatureGroup != opcodetable.OpcodeGroupCxfx
}

// OpcodesUsed returns the sorted numbers of the opcodes prgrm can call,
// those of its built-in functions.
func OpcodesUsed(prgrm *ast.CXProgram) []int {
	seen := map[int]bool{}
	var used []int
	for index := 0; index < prgrm.CXProgramFunctionSlotCount(); index++ {
		fn := prgrm.CXProgramFunctionSlotLive(index)
		if fn == nil {
			continue
		}
		if code := fn.AtomicOPCode; code != 0 && !seen[code] {
			seen[code] = true
			used = append(used, code)
		}
	}
	sort.Ints(used)
	return used
}

// OpcodesMissing returns the names of the opcodes prgrm uses that the
// core does not link, sorted by number.
func OpcodesMissing(prgrm *ast.CXProgram) []string {
	var missing []string
	for _, code := range OpcodesUsed(prgrm) {
		if !OpcodeLinked(code) {
			missing = append(missing, OpcodeName(code))
		}
	}
	return missing
}

// OpcodeName returns the name of opcode number code in
// constants.OpcodeLifecycleTable.
func OpcodeName(code int) string {
	row, found := opcodetable.OpcodeLifecycleLiveByNumber(code)
	assert.PreconditionTrue(found, "opcode number without a live row")
	return row.Name
}

// LibrariesEnabledResolve applies constants.LibraryPolicyEnabledWhenImported
// to prgrm: when it is true, every library the program calls is enabled.
// Then every opcode of an enabled library joins
// ProgramOperationsEnabled, the set the loader checks (decision D119).
func LibrariesEnabledResolve(prgrm *ast.CXProgram) {
	for library, used := range prgrm.ProgramLibrariesUsed {
		if used && opcodetable.LibraryPolicyEnabledWhenImported {
			prgrm.ProgramLibrariesEnabled[library] = true
		}
		if prgrm.ProgramLibrariesEnabled[library] {
			prgrm.ProgramOperationsEnabled.OpcodeSetLibraryAdd(library)
		}
	}
}

// LibrariesActive returns the libraries prgrm may call: those enabled
// whole and those with an enabled operation.
func LibrariesActive(prgrm *ast.CXProgram) [opcodetable.LibraryCount]bool {
	var active [opcodetable.LibraryCount]bool
	for library := range active {
		active[library] = prgrm.ProgramLibrariesEnabled[library] || prgrm.ProgramOperationsEnabled.OpcodeSetLibraryAny(library)
	}
	return active
}

// LibrariesMissing returns the names of the libraries active for prgrm
// that this binary does not link, in number order. opcodesLinked marks the
// libraries whose functions are opcodes that the binary links, by its build
// tags (decision D119), and the other libraries are linked when registered.
func LibrariesMissing(prgrm *ast.CXProgram, opcodesLinked [opcodetable.LibraryCount]bool) []string {
	var missing [opcodetable.LibraryCount]bool
	for library, active := range LibrariesActive(prgrm) {
		missing[library] = active && !ast.LibraryLinked(library) && !opcodesLinked[library]
	}
	return LibraryNames(missing)
}

// OperationRefused returns the first library opcode, in number order,
// that prgrm uses and its run does not enable, and 0 when there is none.
// The loader refuses such a program before it runs, so the run loop never
// checks a library or an operation (decision D119).
func OperationRefused(prgrm *ast.CXProgram) int {
	return prgrm.CXProgramOperationRefused()
}

// OperationRefusalLine is the line a program refused for opcode code
// prints, the runtime error CXRuntimeLibraryDisabled without a source
// position. It names the library when the run enables nothing of it, and
// the operation otherwise.
func OperationRefusalLine(prgrm *ast.CXProgram, code int) string {
	return prgrm.CXProgramOperationRefusalLine(code)
}

// LibraryNames returns the names of the libraries libraries marks, in
// number order.
func LibraryNames(libraries [opcodetable.LibraryCount]bool) []string {
	var names []string
	for library, marked := range libraries {
		if marked {
			names = append(names, opcodetable.LibraryNameOf(library))
		}
	}
	return names
}
