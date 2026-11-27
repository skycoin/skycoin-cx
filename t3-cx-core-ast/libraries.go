package ast

import (
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
)

// LibraryDispatchFunction runs function of one system library on its
// operands and reports false for a number the library does not define. The
// switches are generated from constants.LibraryFunctionDefinitionTable
// (decision D96).
type LibraryDispatchFunction func(prgrm *CXProgram, function int, inputs []CXValue, outputs []CXValue) bool

// LibraryDispatches holds the switch of each system library this binary
// links, by library number, nil for a library it leaves out. The
// registration of the binary sets it, see opcodes.LibraryLink.
// VIOLATION: state outside State. Owner: t3-cx-core-ast; moves to CXNativeFunctionMng in phase 1.
var LibraryDispatches [opcodetable.LibraryCount]LibraryDispatchFunction

// LibraryFunctionKey returns the key of a library function in NativeNames
// and NativeFunctionOf. It is negative, so it never meets an opcode number.
func LibraryFunctionKey(library int, function int) int {
	return opcodetable.LibraryFunctionKey(library, function)
}

// LibraryLinked reports whether this binary links library.
func LibraryLinked(library int) bool {
	return library > 0 && library < opcodetable.LibraryCount && LibraryDispatches[library] != nil
}

// LibraryFunctionLinked reports whether this binary links library and the
// library defines function.
func LibraryFunctionLinked(library int, function int) bool {
	return LibraryLinked(library) && opcodetable.LibraryFunctionSignatureOf(library, function).LibraryFunctionSignatureDefined
}

// BuiltInName returns the name of the built-in fn calls, the library function
// for constants.OpLibraryCall and the opcode otherwise.
func (fn *CXFunction) BuiltInName() string {
	if fn.AtomicOPCode == opcodetable.OpLibraryCall {
		// A binary that does not link the library, such as a cx build core,
		// has no NativeNames entry for it.
		row, _ := opcodetable.LibraryFunctionLiveByNumber(fn.LibraryNumber, fn.LibraryFunctionNumber)
		return row.Name
	}
	return opcodetable.OpcodeNameOf(fn.AtomicOPCode)
}

// BuiltInNameBytes returns the bytes of BuiltInName without building a
// string, for printing on the run path. The caller must not write to them.
func (fn *CXFunction) BuiltInNameBytes() []byte {
	if fn.AtomicOPCode == opcodetable.OpLibraryCall {
		return opcodetable.LibraryFunctionLiveNameBytes(fn.LibraryNumber, fn.LibraryFunctionNumber)
	}
	return opcodetable.OpcodeNameBytesOf(fn.AtomicOPCode)
}

// RuntimeLibraryDisabled is the panic value of a call into a system library
// the run did not enable, the library number. Error builds the message the
// runtime error prints, after the run (#412).
type RuntimeLibraryDisabled struct {
	RuntimeLibraryDisabledLibrary int
}

func (disabled RuntimeLibraryDisabled) Error() string {
	return RuntimeLibraryDisabledMessage(disabled.RuntimeLibraryDisabledLibrary)
}

// RuntimeLibraryDisabledMessage is the message of a call into the disabled
// library number.
func RuntimeLibraryDisabledMessage(library int) string {
	name := opcodetable.LibraryNameOf(library)
	return "library " + name + " is not enabled, enable it with --enable=" + name
}

// libraryCall runs library function fn through the switch of its library
// after the sandbox check, one indexed load and compare.
func libraryCall(prgrm *CXProgram, fn *CXFunction, inputs []CXValue, outputs []CXValue) {
	if !prgrm.ProgramLibrariesEnabled[fn.LibraryNumber] {
		panic(RuntimeLibraryDisabled{fn.LibraryNumber})
	}
	dispatch := LibraryDispatches[fn.LibraryNumber]
	if dispatch == nil || !dispatch(prgrm, fn.LibraryFunctionNumber, inputs, outputs) {
		panic(RuntimeOpcodeUnlinked(opcodetable.OpLibraryCall))
	}
}
