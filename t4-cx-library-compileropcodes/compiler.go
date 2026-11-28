// Package compileropcodes holds the compiler library, the opcodes of
// opcodetable.OpcodeGroupCompiler (decision D152). A CX program hands it
// source text, and it runs the Go compiler of cx on it: parse, check and
// lower. A lowered program is an image in the ir.ImageStore of sys/cxir, and
// the structs and functions of a parsed program are sys/reflect handles.
// The Go compiler is not linked here: cx gives the library a Compiler. The
// old loop runs the Op functions and vm runs VmOpcodeRun.
package compileropcodes

import (
	"github.com/skycoin/skycoin-cx/t0-cx-runtime-output"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	reflecttable "github.com/skycoin/skycoin-cx/t2-cx-library-reflecttable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	// GCREACH: t3-cx-core-ast owns the heap. The library holds the programs
	// the Go compiler builds, which allocate while it parses (D152).
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// AstTag is the tag of an AST handle, in the reflect, meta, cxir and
// compiler range 0x40 to 0x5F of gid-variants D154.
const AstTag uint8 = 0x44

// The Gid layout of t1-cx-core-objmodel, bits 0 to 31 the index, 32 to 55
// the generation and 56 to 63 the tag.
const (
	astGidGenerationShift = 32
	astGidTagShift        = 56
	astGenerationMax      = 0xFFFFFF
)

// CompilerError is one compile error, its line and the text cx prints for it
// without the newline.
type CompilerError struct {
	CompilerErrorLine int32
	CompilerErrorText []byte
}

// CompilerPackage is one package of a parsed program, its functions and
// structs as record numbers of CompilerParse.CompilerParseTypes, in name
// order, and its globals in declaration order.
type CompilerPackage struct {
	CompilerPackageName      []byte
	CompilerPackageFunctions []int32
	CompilerPackageStructs   []int32
	CompilerPackageGlobals   [][]byte
}

// CompilerParse is what the Go compiler gives back for the source files of
// one program. CompilerParseProgram is nil after a syntax error, and
// CompilerParseTypes is the type table of its declarations, nil when it has
// errors.
type CompilerParse struct {
	CompilerParseProgram  *ast.CXProgram
	CompilerParseErrors   []CompilerError
	CompilerParsePackages []CompilerPackage
	CompilerParseTypes    []byte
}

// Compiler is the Go compiler of cx. CompilerSourcesParse parses, checks and
// adds *init, compiled with the library operations operations and the output
// mode mode. CompilerProgramLower lowers a program without errors to an image
// file, or returns why it does not lower. Both allocate.
type Compiler interface {
	CompilerSourcesParse(names [][]byte, sources [][]byte, operations *opcodetable.OpcodeSet, mode output.Mode) CompilerParse
	CompilerProgramLower(prgrm *ast.CXProgram) (file []byte, problem []byte)
}

// LibraryRoom bounds the library, fixed before the run so no call but the
// ones that run the Go compiler allocates (D152).
type LibraryRoom struct {
	LibraryRoomSourceFiles int
	LibraryRoomSourceBytes int
	LibraryRoomAsts        int
	LibraryRoomErrors      int
	LibraryRoomErrorBytes  int
}

// LibraryRoomDefault is the room of a program that calls the library, about
// 272 KiB.
var LibraryRoomDefault = LibraryRoom{
	LibraryRoomSourceFiles: 16,
	LibraryRoomSourceBytes: 256 << 10,
	LibraryRoomAsts:        4,
	LibraryRoomErrors:      64,
	LibraryRoomErrorBytes:  16 << 10,
}

// astSlot is one parsed program. astTypeMap and astFunctionMap get the
// number in the type table of the run of each record of astTypes when a
// struct or function is first read.
type astSlot struct {
	astUsed          bool
	astGeneration    uint32
	astProgram       *ast.CXProgram
	astErrorCount    int32
	astPackages      []astPackage
	astTypes         []byte
	astTypeMap       []int32
	astFunctionMap   []int32
	astTypesAppended bool
	astTypesFit      bool
}

// astPackage is a package of an astSlot with its names in the store.
type astPackage struct {
	astPackageName      int64
	astPackageFunctions []int32
	astPackageStructs   []int32
	astPackageGlobals   []int64
}

// LibraryState is the state of the compiler library, its one global, see
// D139 on #412. LibraryCompiler is nil in a build that does not link the Go
// compiler, where every parse fails with an error.
type LibraryState struct {
	LibraryCompiler      Compiler
	LibraryOperations    opcodetable.OpcodeSet
	LibraryOutputMode    output.Mode
	LibraryTypeTable     []byte
	LibraryImageStore    *ir.ImageStore
	librarySourceNames   [][]byte
	librarySourceTexts   [][]byte
	librarySourceBytes   []byte
	librarySourceUsed    int
	libraryAsts          []astSlot
	libraryErrorLines    []int32
	libraryErrorTexts    [][]byte
	libraryErrorBytes    []byte
	libraryErrorUsed     int
	libraryErrorCount    int
	libraryNamesScratch  [][]byte
	librarySourceScratch [][]byte
}

// Library is the state of the compiler library.
var Library LibraryState

// The texts of the errors the library adds itself.
var (
	errorTextUnlinked   = []byte("error: this build of cx does not link the Go compiler, run the program with cx")
	errorTextNoSource   = []byte("error: no source files, append one with compiler.CxCompilerSourceFileAppend")
	errorTextStoreFull  = []byte("error: the image does not fit the image store")
	errorTextAstsFull   = []byte("error: every AST of the compiler library is in use, release one")
	errorTextAstInvalid = []byte("error: no such AST")
)

// LibraryPrepare makes the room of a run that enables operations and writes
// in output mode mode. table is the type table of reflect and store the
// images of the old loop, nil on vm, which passes its own to every call. It
// keeps LibraryCompiler.
func LibraryPrepare(operations *opcodetable.OpcodeSet, mode output.Mode, table []byte, store *ir.ImageStore) {
	room := LibraryRoomDefault
	Library = LibraryState{
		LibraryCompiler:      Library.LibraryCompiler,
		LibraryOperations:    *operations,
		LibraryOutputMode:    mode,
		LibraryTypeTable:     table,
		LibraryImageStore:    store,
		librarySourceNames:   make([][]byte, 0, room.LibraryRoomSourceFiles),
		librarySourceTexts:   make([][]byte, 0, room.LibraryRoomSourceFiles),
		librarySourceBytes:   make([]byte, room.LibraryRoomSourceBytes),
		libraryAsts:          make([]astSlot, room.LibraryRoomAsts),
		libraryErrorLines:    make([]int32, room.LibraryRoomErrors),
		libraryErrorTexts:    make([][]byte, room.LibraryRoomErrors),
		libraryErrorBytes:    make([]byte, room.LibraryRoomErrorBytes),
		libraryNamesScratch:  make([][]byte, room.LibraryRoomSourceFiles),
		librarySourceScratch: make([][]byte, room.LibraryRoomSourceFiles),
	}
	for i := range Library.libraryAsts {
		Library.libraryAsts[i].astGeneration = 1
	}
}

// LibraryCompilerSet gives the library the Go compiler, once at the start
// of cx.
func LibraryCompilerSet(compiler Compiler) {
	Library.LibraryCompiler = compiler
}

// errorsClear forgets the errors of the last parse, lower or compile.
func (state *LibraryState) errorsClear() {
	state.libraryErrorCount, state.libraryErrorUsed = 0, 0
}

// errorAdd keeps an error in the room, its text cut to the bytes left, and
// counts it. An error past the room is counted by the AST only.
func (state *LibraryState) errorAdd(line int32, text []byte) {
	if state.libraryErrorCount >= len(state.libraryErrorLines) {
		return
	}
	size := copy(state.libraryErrorBytes[state.libraryErrorUsed:], text)
	state.libraryErrorLines[state.libraryErrorCount] = line
	state.libraryErrorTexts[state.libraryErrorCount] = state.libraryErrorBytes[state.libraryErrorUsed : state.libraryErrorUsed+size]
	state.libraryErrorUsed += size
	state.libraryErrorCount++
}

// SourceFileAppend keeps a copy of the first length bytes of text as a
// source file named by the cxir name name, for the next parse or compile.
// It returns the file ordinal, -1 for no name, a bad length or no room.
func (state *LibraryState) SourceFileAppend(store *ir.ImageStore, name int64, text []byte, length int32) int32 {
	if store == nil || length < 0 || int(length) > len(text) || len(state.librarySourceNames) == cap(state.librarySourceNames) {
		return -1
	}
	room := state.librarySourceBytes[state.librarySourceUsed:]
	nameSize := store.ImageStoreStringPoolTextRead(name, room)
	if nameSize <= 0 || int(nameSize) >= len(room) || int(nameSize)+int(length) > len(room) {
		return -1
	}
	nameBytes := room[:nameSize]
	textBytes := room[nameSize : int(nameSize)+copy(room[nameSize:], text[:length])]
	state.librarySourceUsed += int(nameSize) + int(length)
	state.librarySourceNames = append(state.librarySourceNames, nameBytes)
	state.librarySourceTexts = append(state.librarySourceTexts, textBytes)
	return int32(len(state.librarySourceNames) - 1)
}

// sourcesParse runs the Go compiler on the source files appended since the
// last parse, which it forgets, and keeps the errors. ok is false when there
// is no compiler, no source or a syntax error.
func (state *LibraryState) sourcesParse() (parse CompilerParse, ok bool) {
	state.errorsClear()
	names := state.libraryNamesScratch[:copy(state.libraryNamesScratch, state.librarySourceNames)]
	sources := state.librarySourceScratch[:copy(state.librarySourceScratch, state.librarySourceTexts)]
	state.librarySourceNames, state.librarySourceTexts, state.librarySourceUsed = state.librarySourceNames[:0], state.librarySourceTexts[:0], 0
	switch {
	case state.LibraryCompiler == nil:
		state.errorAdd(0, errorTextUnlinked)
		return parse, false
	case len(names) == 0:
		state.errorAdd(0, errorTextNoSource)
		return parse, false
	}
	parse = state.LibraryCompiler.CompilerSourcesParse(names, sources, &state.LibraryOperations, state.LibraryOutputMode)
	for _, compileError := range parse.CompilerParseErrors {
		state.errorAdd(compileError.CompilerErrorLine, compileError.CompilerErrorText)
	}
	return parse, parse.CompilerParseProgram != nil
}

// astGidCompose returns the handle of AST slot index.
func astGidCompose(index int, generation uint32) int64 {
	return int64(uint64(AstTag)<<astGidTagShift | uint64(generation)<<astGidGenerationShift | uint64(uint32(index)))
}

// astOf returns the AST of handle, nil for no AST or a released one.
func (state *LibraryState) astOf(astGid int64) *astSlot {
	if astGid <= 0 || uint8(uint64(astGid)>>astGidTagShift) != AstTag {
		return nil
	}
	index := uint64(uint32(astGid))
	if index >= uint64(len(state.libraryAsts)) {
		return nil
	}
	slot := &state.libraryAsts[index]
	if !slot.astUsed || uint32(uint64(astGid)>>astGidGenerationShift)&astGenerationMax != slot.astGeneration {
		return nil
	}
	return slot
}

// SourceParse parses the appended source files into an AST and returns its
// handle, -1 after a syntax error or when every AST is in use. A program
// with type errors gets a handle, and ProgramTypeCheck counts them.
func (state *LibraryState) SourceParse(store *ir.ImageStore) int64 {
	free := -1
	for i := range state.libraryAsts {
		if !state.libraryAsts[i].astUsed {
			free = i
			break
		}
	}
	if free < 0 {
		state.librarySourceNames, state.librarySourceTexts, state.librarySourceUsed = state.librarySourceNames[:0], state.librarySourceTexts[:0], 0
		state.errorsClear()
		state.errorAdd(0, errorTextAstsFull)
		return -1
	}
	parse, ok := state.sourcesParse()
	if !ok || store == nil {
		return -1
	}
	slot := &state.libraryAsts[free]
	generation := slot.astGeneration
	*slot = astSlot{astUsed: true, astGeneration: generation, astProgram: parse.CompilerParseProgram, astErrorCount: int32(len(parse.CompilerParseErrors)), astTypes: parse.CompilerParseTypes}
	slot.astPackages = make([]astPackage, len(parse.CompilerParsePackages))
	for i, pkg := range parse.CompilerParsePackages {
		globals := make([]int64, len(pkg.CompilerPackageGlobals))
		for j, global := range pkg.CompilerPackageGlobals {
			globals[j] = store.ImageStoreStringPoolTextAppend(global)
		}
		slot.astPackages[i] = astPackage{astPackageName: store.ImageStoreStringPoolTextAppend(pkg.CompilerPackageName), astPackageFunctions: pkg.CompilerPackageFunctions, astPackageStructs: pkg.CompilerPackageStructs, astPackageGlobals: globals}
	}
	if slot.astTypes != nil {
		slot.astTypeMap = make([]int32, max(0, wordAt(slot.astTypes, reflecttable.HeaderTypeCount)))
		slot.astFunctionMap = make([]int32, max(0, wordAt(slot.astTypes, reflecttable.HeaderFunctionCount)))
	}
	return astGidCompose(free, generation)
}

// wordAt returns header word index of a type table.
func wordAt(table []byte, index int) int32 {
	if 4*index+4 > len(table) {
		return 0
	}
	return int32(uint32(table[4*index]) | uint32(table[4*index+1])<<8 | uint32(table[4*index+2])<<16 | uint32(table[4*index+3])<<24)
}

// AstRelease frees an AST for the next parse and returns 0, -1 for no AST.
// Its handle names no AST after.
func (state *LibraryState) AstRelease(astGid int64) int32 {
	slot := state.astOf(astGid)
	if slot == nil {
		return -1
	}
	generation := slot.astGeneration + 1
	if generation > astGenerationMax {
		generation = 1
	}
	*slot = astSlot{astGeneration: generation}
	return 0
}

// ProgramTypeCheck returns the errors of the parse of an AST, 0 when it
// has none, -1 for no AST. The Go compiler checks types while it parses.
func (state *LibraryState) ProgramTypeCheck(astGid int64) int32 {
	slot := state.astOf(astGid)
	if slot == nil {
		return -1
	}
	return slot.astErrorCount
}

// programImage lowers prgrm and decodes its image file into store, -1 with
// an error when it does not lower or fit.
func (state *LibraryState) programImage(store *ir.ImageStore, prgrm *ast.CXProgram) int64 {
	if store == nil {
		state.errorAdd(0, errorTextStoreFull)
		return -1
	}
	file, problem := state.LibraryCompiler.CompilerProgramLower(prgrm)
	if len(problem) > 0 {
		state.errorAdd(0, problem)
		return -1
	}
	image := store.ImageStoreFileDecode(file)
	if image < 0 {
		state.errorAdd(0, errorTextStoreFull)
	}
	return image
}

// ProgramLower lowers an AST without errors into a new image of store and
// returns its handle, -1 for no AST, one with errors, one that does not
// lower or a full store.
func (state *LibraryState) ProgramLower(store *ir.ImageStore, astGid int64) int64 {
	state.errorsClear()
	slot := state.astOf(astGid)
	switch {
	case slot == nil:
		state.errorAdd(0, errorTextAstInvalid)
		return -1
	case slot.astErrorCount > 0:
		return -1
	}
	return state.programImage(store, slot.astProgram)
}

// ProgramCompile parses, checks and lowers the appended source files into a
// new image of store and returns its handle, -1 on any error. It keeps no
// AST.
func (state *LibraryState) ProgramCompile(store *ir.ImageStore) int64 {
	parse, ok := state.sourcesParse()
	if !ok || len(parse.CompilerParseErrors) > 0 {
		return -1
	}
	return state.programImage(store, parse.CompilerParseProgram)
}

// ErrorCountGet returns the errors the library keeps of the last parse,
// lower or compile, at most LibraryRoomErrors.
func (state *LibraryState) ErrorCountGet() int32 {
	return int32(state.libraryErrorCount)
}

// ErrorMessageRead copies the text of error ordinal into buffer and returns
// the bytes copied, -1 for no error. A longer text is cut at the buffer.
func (state *LibraryState) ErrorMessageRead(ordinal int32, buffer []byte) int32 {
	if ordinal < 0 || int(ordinal) >= state.libraryErrorCount {
		return -1
	}
	return int32(copy(buffer, state.libraryErrorTexts[ordinal]))
}

// ErrorSourceLineGet returns the line of error ordinal, 0 for an error of
// no line, -1 for no error.
func (state *LibraryState) ErrorSourceLineGet(ordinal int32) int32 {
	if ordinal < 0 || int(ordinal) >= state.libraryErrorCount {
		return -1
	}
	return state.libraryErrorLines[ordinal]
}

// packageOf returns package ordinal pkg of an AST, nil for none.
func (state *LibraryState) packageOf(astGid int64, pkg int32) *astPackage {
	slot := state.astOf(astGid)
	if slot == nil || pkg < 0 || int(pkg) >= len(slot.astPackages) {
		return nil
	}
	return &slot.astPackages[pkg]
}

// AstPackageCountGet returns the packages of an AST, -1 for no AST.
func (state *LibraryState) AstPackageCountGet(astGid int64) int32 {
	slot := state.astOf(astGid)
	if slot == nil {
		return -1
	}
	return int32(len(slot.astPackages))
}

// AstPackageNameGet returns the cxir name of package pkg, 0 for none.
func (state *LibraryState) AstPackageNameGet(astGid int64, pkg int32) int64 {
	if p := state.packageOf(astGid, pkg); p != nil {
		return p.astPackageName
	}
	return 0
}

// AstPackageFunctionCountGet returns the functions of package pkg, -1 for
// none.
func (state *LibraryState) AstPackageFunctionCountGet(astGid int64, pkg int32) int32 {
	if p := state.packageOf(astGid, pkg); p != nil {
		return int32(len(p.astPackageFunctions))
	}
	return -1
}

// AstPackageStructCountGet returns the structs of package pkg, -1 for none.
func (state *LibraryState) AstPackageStructCountGet(astGid int64, pkg int32) int32 {
	if p := state.packageOf(astGid, pkg); p != nil {
		return int32(len(p.astPackageStructs))
	}
	return -1
}

// AstPackageGlobalCountGet returns the globals of package pkg, -1 for none.
func (state *LibraryState) AstPackageGlobalCountGet(astGid int64, pkg int32) int32 {
	if p := state.packageOf(astGid, pkg); p != nil {
		return int32(len(p.astPackageGlobals))
	}
	return -1
}

// AstPackageGlobalNameGet returns the cxir name of global ordinal of
// package pkg, 0 for none.
func (state *LibraryState) AstPackageGlobalNameGet(astGid int64, pkg int32, ordinal int32) int64 {
	p := state.packageOf(astGid, pkg)
	if p == nil || ordinal < 0 || int(ordinal) >= len(p.astPackageGlobals) {
		return 0
	}
	return p.astPackageGlobals[ordinal]
}

// astTypesAppend adds the types of an AST to table at its first struct or
// function read, and reports whether they are there. A table without the
// room keeps none, and every read of the AST returns 0.
func (slot *astSlot) astTypesAppend(table []byte) bool {
	if !slot.astTypesAppended {
		slot.astTypesAppended = true
		slot.astTypesFit = slot.astTypes != nil && reflecttable.TableAppend(table, slot.astTypes, slot.astTypeMap, slot.astFunctionMap)
	}
	return slot.astTypesFit
}

// AstPackageFunctionGet returns the reflect function handle of function
// ordinal of package pkg in table, 0 for none or no room in table.
func (state *LibraryState) AstPackageFunctionGet(table []byte, astGid int64, pkg int32, ordinal int32) int64 {
	p := state.packageOf(astGid, pkg)
	if p == nil || ordinal < 0 || int(ordinal) >= len(p.astPackageFunctions) {
		return 0
	}
	slot := state.astOf(astGid)
	record := p.astPackageFunctions[ordinal]
	if !slot.astTypesAppend(table) || record < 0 || int(record) >= len(slot.astFunctionMap) {
		return 0
	}
	return reflecttable.FunctionOfIndex(table, slot.astFunctionMap[record])
}

// AstPackageStructGet returns the reflect type handle of struct ordinal of
// package pkg in table, 0 for none or no room in table.
func (state *LibraryState) AstPackageStructGet(table []byte, astGid int64, pkg int32, ordinal int32) int64 {
	p := state.packageOf(astGid, pkg)
	if p == nil || ordinal < 0 || int(ordinal) >= len(p.astPackageStructs) {
		return 0
	}
	slot := state.astOf(astGid)
	record := p.astPackageStructs[ordinal]
	if !slot.astTypesAppend(table) || record < 0 || int(record) >= len(slot.astTypeMap) {
		return 0
	}
	return reflecttable.TypeOfIndex(table, slot.astTypeMap[record])
}
