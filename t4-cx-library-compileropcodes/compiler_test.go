package compileropcodes

import (
	"runtime/debug"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-cx-runtime-output"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// compilerStub parses every program into one package with one global and
// lowers nothing, so the state is tested without the Go compiler.
type compilerStub struct {
	compilerStubErrors int
}

func (stub compilerStub) CompilerSourcesParse(names [][]byte, sources [][]byte, operations *opcodetable.OpcodeSet, mode output.Mode) CompilerParse {
	parse := CompilerParse{CompilerParseProgram: &ast.CXProgram{}}
	for i := 0; i < stub.compilerStubErrors; i++ {
		parse.CompilerParseErrors = append(parse.CompilerParseErrors, CompilerError{CompilerErrorLine: int32(i + 3), CompilerErrorText: []byte("error: stub")})
	}
	parse.CompilerParsePackages = []CompilerPackage{{CompilerPackageName: names[0], CompilerPackageGlobals: [][]byte{sources[0]}}}
	return parse
}

func (compilerStub) CompilerProgramLower(prgrm *ast.CXProgram) ([]byte, []byte) {
	return nil, []byte("error: stub lowers nothing")
}

// TestLibraryStateStub checks the sources, errors, AST handles and their
// release against a stub compiler, and that the reads allocate nothing.
func TestLibraryStateStub(t *testing.T) {
	store := ir.ImageStoreNew(ir.ImageStoreRoomDefault)
	LibraryCompilerSet(nil)
	LibraryPrepare(&opcodetable.OpcodeSet{}, output.ModeBatch, nil, store)
	if got := Library.SourceParse(store); got != -1 || Library.ErrorCountGet() != 1 {
		t.Errorf("parse without a compiler gives %d and %d errors, want -1 and 1", got, Library.ErrorCountGet())
	}
	LibraryCompilerSet(compilerStub{})
	name := store.ImageStoreStringPoolTextAppend([]byte("main.cx"))
	text := []byte("global")
	if got := Library.SourceFileAppend(store, name, text, int32(len(text))); got != 0 {
		t.Fatalf("append gives %d, want 0", got)
	}
	if got := Library.SourceFileAppend(store, name, text, 7); got != -1 {
		t.Errorf("append past the buffer gives %d, want -1", got)
	}
	if got := Library.SourceFileAppend(store, 0, text, 1); got != -1 {
		t.Errorf("append with no name gives %d, want -1", got)
	}
	first := Library.SourceParse(store)
	if first <= 0 || Library.ProgramTypeCheck(first) != 0 || Library.AstPackageCountGet(first) != 1 {
		t.Fatalf("parse gives %#x, %d errors and %d packages", first, Library.ProgramTypeCheck(first), Library.AstPackageCountGet(first))
	}
	buffer := make([]byte, 16)
	if size := store.ImageStoreStringPoolTextRead(Library.AstPackageGlobalNameGet(first, 0, 0), buffer); string(buffer[:size]) != "global" {
		t.Errorf("global name %q, want global", buffer[:size])
	}
	if Library.AstPackageNameGet(first, 0) != name || Library.AstPackageStructGet(nil, first, 0, 0) != 0 || Library.AstPackageFunctionCountGet(first, 1) != -1 {
		t.Errorf("package reads are wrong")
	}
	if got := Library.SourceParse(store); got != -1 {
		t.Errorf("parse of no source gives %d, want -1", got)
	}
	allocs := testing.AllocsPerRun(10, func() {
		Library.ProgramTypeCheck(first)
		Library.AstPackageCountGet(first)
		Library.AstPackageNameGet(first, 0)
		Library.AstPackageGlobalCountGet(first, 0)
		Library.AstPackageGlobalNameGet(first, 0, 0)
		Library.AstPackageFunctionCountGet(first, 0)
		Library.AstPackageStructCountGet(first, 0)
		Library.ErrorCountGet()
		Library.ErrorMessageRead(0, buffer)
		Library.ErrorSourceLineGet(0)
	})
	if allocs != 0 {
		t.Errorf("%.0f allocations per read, want 0", allocs)
	}
	LibraryCompilerSet(compilerStub{compilerStubErrors: 2})
	Library.SourceFileAppend(store, name, text, 1)
	second := Library.SourceParse(store)
	if Library.ProgramTypeCheck(second) != 2 || Library.ErrorCountGet() != 2 || Library.ErrorSourceLineGet(1) != 4 || Library.ProgramLower(store, second) != -1 {
		t.Errorf("a program with 2 errors checks %d, keeps %d", Library.ProgramTypeCheck(second), Library.ErrorCountGet())
	}
	if Library.AstRelease(first) != 0 || Library.AstRelease(first) != -1 || Library.AstPackageCountGet(first) != -1 {
		t.Errorf("a released AST still reads")
	}
	for i := 0; i < len(Library.libraryAsts); i++ {
		Library.SourceFileAppend(store, name, text, 1)
		Library.SourceParse(store)
	}
	if Library.SourceFileAppend(store, name, text, 1); Library.SourceParse(store) != -1 {
		t.Errorf("parse with every AST in use does not give -1")
	}
}

// TestStoreRelease checks that a released image frees its slot and its
// handle names no image after.
func TestStoreRelease(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	store := ir.ImageStoreNew(ir.ImageStoreRoomDefault)
	var images []int64
	for image := store.ImageStoreImageCreate(); image != 0; image = store.ImageStoreImageCreate() {
		images = append(images, image)
	}
	if len(images) != ir.ImageStoreRoomDefault.ImageStoreRoomImages {
		t.Fatalf("%d images fit, want %d", len(images), ir.ImageStoreRoomDefault.ImageStoreRoomImages)
	}
	if store.ImageStoreImageRelease(images[1]) != 0 || store.ImageStoreImageRelease(images[1]) != -1 || store.ImageStoreFunctionCountGet(nil, images[1]) != -1 {
		t.Errorf("a released image still reads")
	}
	again := store.ImageStoreImageCreate()
	if again == 0 || again == images[1] || store.ImageStoreFunctionCountGet(nil, again) != 0 {
		t.Errorf("the freed slot gives %#x after %#x", again, images[1])
	}
	if allocs := testing.AllocsPerRun(10, func() {
		store.ImageStoreImageRelease(again)
		again = store.ImageStoreImageCreate()
	}); allocs != 0 {
		t.Errorf("%.0f allocations per release and create, want 0", allocs)
	}
}
