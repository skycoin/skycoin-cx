//go:build cxfx

package cxfx

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t0-util"
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// textTestProgram returns a program whose memory holds a pointer at offset 0
// to a str object with text, and the value that reads it.
func textTestProgram(text string) (*ast.CXProgram, ast.CXValue) {
	memory := make([]byte, 64+len(text))
	heapOffset := types.Pointer(32)
	types.Write_obj_data(memory, heapOffset, types.Compute_obj_size([]byte(text)), []byte(text))
	types.Write_ptr(memory, 0, heapOffset)
	return &ast.CXProgram{Memory: memory}, ast.CXValue{Offset: 0}
}

func TestCxfxTextNulTerminated(t *testing.T) {
	prgrm, value := textTestProgram("projection")
	name := cxfxTextNulTerminated(ast.ProgramStringBytesRead(prgrm, &value))
	if got := string(cxfxState.CxfxTextBuffer[:len("projection")+1]); got != "projection\x00" || name != &cxfxState.CxfxTextBuffer[0] {
		t.Fatalf("the C string is %q", got)
	}
	long := make([]byte, 3*CxfxTextBufferSizeInitial)
	cxfxTextNulTerminated(long)
	if len(cxfxState.CxfxTextBuffer) < len(long)+1 || cxfxState.CxfxTextBuffer[len(long)] != 0 {
		t.Fatal("a long name did not grow the scratch")
	}
}

func TestCxfxTextNulTerminatedAllocations(t *testing.T) {
	prgrm, value := textTestProgram("projection")
	cxfxTextNulTerminated(ast.ProgramStringBytesRead(prgrm, &value))
	allocations := testing.AllocsPerRun(100, func() {
		cxfxTextNulTerminated(ast.ProgramStringBytesRead(prgrm, &value))
	})
	if allocations != 0 {
		t.Fatalf("a GL name allocates %v times", allocations)
	}
}

func TestCxfxTextViewAllocations(t *testing.T) {
	prgrm, value := textTestProgram("Score: 1234")
	var sink int
	allocations := testing.AllocsPerRun(100, func() {
		sink += len(util.ProgramTextStringView(ast.ProgramStringBytesRead(prgrm, &value)))
	})
	if allocations != 0 {
		t.Fatalf("reading text allocates %v times", allocations)
	}
	if view := util.ProgramTextStringView(ast.ProgramStringBytesRead(prgrm, &value)); view != "Score: 1234" {
		t.Fatalf("text is %q", view)
	}
}

func TestCxfxHandleTables(t *testing.T) {
	saved := cxfxState
	defer func() { cxfxState = saved }()
	cxfxState = CxfxState{}

	first := cxfxTexturePixelsAdd(CxfxTexturePixels{CxfxTexturePixelsData: []float32{1}})
	second := cxfxTexturePixelsAdd(CxfxTexturePixels{CxfxTexturePixelsData: []float32{2}})
	if first != 0 || second != 1 || cxfxTexturePixelsGet(second).CxfxTexturePixelsData[0] != 2 {
		t.Fatalf("handles %d and %d", first, second)
	}
	*cxfxTexturePixelsGet(first) = CxfxTexturePixels{}
	if again := cxfxTexturePixelsAdd(CxfxTexturePixels{CxfxTexturePixelsData: []float32{3}}); again != first {
		t.Fatalf("a freed slot was not reused, got %d", again)
	}
	if handle := cxfxCallbackI32I32Add(7); handle != 0 || cxfxCallbackI32I32Get(0) != 7 {
		t.Fatalf("callback handle %d", handle)
	}
}

func TestCxfxCallbackInputsAllocations(t *testing.T) {
	allocations := testing.AllocsPerRun(100, func() {
		cxfxCallbackInputsI32(0, 3)
		cxfxCallbackInputsI64(1, -1)
		cxfxCallbackInputsF64(2, 0.5)
	})
	if allocations != 0 {
		t.Fatalf("writing callback inputs allocates %v times", allocations)
	}
	if types.Read_i32(cxfxState.CxfxCallbackInputs[0], 0) != 3 || types.Read_i64(cxfxState.CxfxCallbackInputs[1], 0) != -1 || types.Read_f64(cxfxState.CxfxCallbackInputs[2], 0) != 0.5 {
		t.Fatal("callback inputs read back wrong")
	}
}

func BenchmarkCxfxTextUniformName(b *testing.B) {
	prgrm, value := textTestProgram("projection")
	b.ReportAllocs()
	for b.Loop() {
		cxfxTextNulTerminated(ast.ProgramStringBytesRead(prgrm, &value))
	}
}
