package constants

import "strings"

// OpcodeSetWordCount is the number of 64 bit words of an OpcodeSet.
const OpcodeSetWordCount = (OpcodeSignatureTableSize + 63) / 64

// OpcodeSet is a set of opcode numbers, one bit per number of the opcode
// table. It has a fixed size and allocates nothing.
type OpcodeSet [OpcodeSetWordCount]uint64

// OpcodeSetAdd adds opcode code to set.
func (set *OpcodeSet) OpcodeSetAdd(code int) {
	if code > 0 && code < OpcodeSignatureTableSize {
		set[code>>6] |= 1 << (uint(code) & 63)
	}
}

// OpcodeSetContains reports whether set holds opcode code.
func (set *OpcodeSet) OpcodeSetContains(code int) bool {
	return code > 0 && code < OpcodeSignatureTableSize && set[code>>6]&(1<<(uint(code)&63)) != 0
}

// OpcodeSetLibraryAdd adds every opcode of library to set.
func (set *OpcodeSet) OpcodeSetLibraryAdd(library int) {
	for _, row := range OpcodeDefinitionTable {
		if OpcodeLibraryOf(row.Number) == library {
			set.OpcodeSetAdd(row.Number)
		}
	}
}

// OpcodeSetLibraryAny reports whether set holds an opcode of library.
func (set *OpcodeSet) OpcodeSetLibraryAny(library int) bool {
	for _, row := range OpcodeDefinitionTable {
		if OpcodeLibraryOf(row.Number) == library && set.OpcodeSetContains(row.Number) {
			return true
		}
	}
	return false
}

// LibraryOperationName returns the name --enable takes for the library
// opcode code, the library name and the CX function name after its
// package, cipher.HashStateNew for cipher.HashStateNew and file.Open for
// os.Open. It returns "" for an opcode of no library.
func LibraryOperationName(code int) string {
	library := OpcodeLibraryOf(code)
	if library == 0 {
		return ""
	}
	name := OpcodeNameOf(code)
	if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
		name = name[dot+1:]
	}
	return LibraryNameOf(library) + "." + name
}

// LibraryOperationByName returns the library opcode whose
// LibraryOperationName is name.
func LibraryOperationByName(name string) (int, bool) {
	for _, row := range OpcodeDefinitionTable {
		if OpcodeLibraryOf(row.Number) != 0 && LibraryOperationName(row.Number) == name {
			return row.Number, true
		}
	}
	return 0, false
}
