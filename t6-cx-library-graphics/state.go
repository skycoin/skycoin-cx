//go:build cxfx

package graphics

import (
	"fmt"
	"image/gif"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/skycoin/gltext"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// The sizes of the handle tables. A program that creates more than a table
// holds fails with an assert, see #432.
const (
	WindowCountMax         = 16
	FontCountMax           = 64
	TexturePixelsCountMax  = 256
	GifCountMax            = 64
	CallbackI32I32CountMax = 256
	// CallbackInputCountMax is the most inputs an event callback takes,
	// the 7 of the mouse callback.
	CallbackInputCountMax = 7
	// TextBufferSizeInitial is the first size of the scratch a GL name is
	// copied into to end it with a NUL.
	TextBufferSizeInitial = 256
)

// CxfxHandleKind names the table a handle indexes, for the assert text.
type CxfxHandleKind int32

const (
	CxfxHandleKindWindow CxfxHandleKind = iota
	CxfxHandleKindFont
	CxfxHandleKindTexturePixels
	CxfxHandleKindGif
	CxfxHandleKindCallbackI32I32
)

// TexturePixels is the CPU copy of a texture that gl.UploadImageToTexture
// keeps when asked, read by gl.CxTextureGetPixel. A nil pixels is a free slot.
type TexturePixels struct {
	TexturePixelsWidth  int32
	TexturePixelsHeight int32
	TexturePixelsLevel  uint32
	TexturePixelsValues []float32
}

// ShaderSourceArgument is the source pointer and length glShaderSource
// reads.
type ShaderSourceArgument struct {
	CxfxShaderSourcePointer *uint8
	CxfxShaderSourceLength  int32
}

// Callback is the CX function an event calls and the window it passes.
type Callback struct {
	CallbackWindow int32
	// CallbackFunction is the function, resolved at compile time.
	CallbackFunction ast.CXFunctionGid
	CallbackSet      bool
}

// State holds every resource a cxfx program creates, in fixed tables an
// i32 handle indexes. GLFW and GL objects belong to the process, so there is
// one state, see #432.
type State struct {
	CxfxWindowTable        [WindowCountMax]*glfw.Window
	CxfxFontTable          [FontCountMax]*gltext.Font
	CxfxTexturePixelsTable [TexturePixelsCountMax]TexturePixels
	CxfxGifTable           [GifCountMax]*gif.GIF

	// CxfxCallbackI32I32Table holds the functions of the i32 callbacks, resolved
	// at compile time.
	CxfxCallbackI32I32Table [CallbackI32I32CountMax]ast.CXFunctionGid
	CxfxCallbackI32I32Count int32

	// CxfxAppCallbackTable is the callback of each event type.
	CxfxAppCallbackTable [AppEventCount]Callback

	// CxfxCallbackInputs are views of CxfxCallbackInputBytes that a callback
	// passes to execute.Callback, so an event allocates nothing.
	CxfxCallbackInputBytes [CallbackInputCountMax][8]byte
	CxfxCallbackInputs     [CallbackInputCountMax][]byte

	// CxfxTextBuffer is the scratch a GL name is copied into with a NUL.
	CxfxTextBuffer []byte

	// CxfxShaderSource holds the arguments of glShaderSource. It is on the
	// heap, since cgo rejects a pointer to a global that holds a pointer, and
	// CxfxShaderSourcePinner pins program memory while GL reads the source.
	CxfxShaderSource       *ShaderSourceArgument
	CxfxShaderSourcePinner runtime.Pinner

	// CxfxGlfwInitialized is set once glfw.Init has run.
	CxfxGlfwInitialized bool
}

// VIOLATION: state outside State. Owner: t6-cx-library-graphics; GLFW and GL are per process, see D45.
var cxfxState State

// cxfxHandleAssertFailure ends the program for a handle out of its table's
// range or naming a freed or never created slot.
// STRHOT REPORTER: builds the assert text of a bad handle from the integers, #412.
func cxfxHandleAssertFailure(kind CxfxHandleKind, handle int32) {
	var names = [...]string{"window", "font", "texture pixels", "gif", "callback"}
	assert.PreconditionTrue(false, fmt.Sprintf("cxfx %s handle %d is out of range, freed or never created", names[kind], handle))
}

// cxfxTableFullAssertFailure ends the program when a table has no free slot.
// STRHOT REPORTER: builds the assert text of a full table from the integers, #412.
func cxfxTableFullAssertFailure(kind CxfxHandleKind, count int) {
	var names = [...]string{"window", "font", "texture pixels", "gif", "callback"}
	assert.PreconditionTrue(false, fmt.Sprintf("cxfx %s table is full, it holds %d", names[kind], count))
}

func cxfxWindowGet(handle int32) *glfw.Window {
	if uint32(handle) >= WindowCountMax || cxfxState.CxfxWindowTable[handle] == nil {
		cxfxHandleAssertFailure(CxfxHandleKindWindow, handle)
	}
	return cxfxState.CxfxWindowTable[handle]
}

func cxfxWindowAdd(window *glfw.Window) int32 {
	for handle := range cxfxState.CxfxWindowTable {
		if cxfxState.CxfxWindowTable[handle] == nil {
			cxfxState.CxfxWindowTable[handle] = window
			return int32(handle)
		}
	}
	cxfxTableFullAssertFailure(CxfxHandleKindWindow, WindowCountMax)
	return -1
}

func cxfxFontGet(handle int32) *gltext.Font {
	if uint32(handle) >= FontCountMax || cxfxState.CxfxFontTable[handle] == nil {
		cxfxHandleAssertFailure(CxfxHandleKindFont, handle)
	}
	return cxfxState.CxfxFontTable[handle]
}

func cxfxFontAdd(font *gltext.Font) int32 {
	for handle := range cxfxState.CxfxFontTable {
		if cxfxState.CxfxFontTable[handle] == nil {
			cxfxState.CxfxFontTable[handle] = font
			return int32(handle)
		}
	}
	cxfxTableFullAssertFailure(CxfxHandleKindFont, FontCountMax)
	return -1
}

func cxfxTexturePixelsGet(handle int32) *TexturePixels {
	if uint32(handle) >= TexturePixelsCountMax || cxfxState.CxfxTexturePixelsTable[handle].TexturePixelsValues == nil {
		cxfxHandleAssertFailure(CxfxHandleKindTexturePixels, handle)
	}
	return &cxfxState.CxfxTexturePixelsTable[handle]
}

func cxfxTexturePixelsAdd(pixels TexturePixels) int32 {
	for handle := range cxfxState.CxfxTexturePixelsTable {
		if cxfxState.CxfxTexturePixelsTable[handle].TexturePixelsValues == nil {
			cxfxState.CxfxTexturePixelsTable[handle] = pixels
			return int32(handle)
		}
	}
	cxfxTableFullAssertFailure(CxfxHandleKindTexturePixels, TexturePixelsCountMax)
	return -1
}

func cxfxGifGet(handle int32) *gif.GIF {
	if uint32(handle) >= GifCountMax || cxfxState.CxfxGifTable[handle] == nil {
		cxfxHandleAssertFailure(CxfxHandleKindGif, handle)
	}
	return cxfxState.CxfxGifTable[handle]
}

func cxfxGifAdd(animation *gif.GIF) int32 {
	for handle := range cxfxState.CxfxGifTable {
		if cxfxState.CxfxGifTable[handle] == nil {
			cxfxState.CxfxGifTable[handle] = animation
			return int32(handle)
		}
	}
	cxfxTableFullAssertFailure(CxfxHandleKindGif, GifCountMax)
	return -1
}

func cxfxCallbackI32I32Get(handle int32) ast.CXFunctionGid {
	if uint32(handle) >= uint32(cxfxState.CxfxCallbackI32I32Count) {
		cxfxHandleAssertFailure(CxfxHandleKindCallbackI32I32, handle)
	}
	return cxfxState.CxfxCallbackI32I32Table[handle]
}

func cxfxCallbackI32I32Add(function ast.CXFunctionGid) int32 {
	handle := cxfxState.CxfxCallbackI32I32Count
	if handle >= CallbackI32I32CountMax {
		cxfxTableFullAssertFailure(CxfxHandleKindCallbackI32I32, CallbackI32I32CountMax)
	}
	cxfxState.CxfxCallbackI32I32Table[handle] = function
	cxfxState.CxfxCallbackI32I32Count++
	return handle
}

// cxfxTextBufferEnsure returns the text scratch with at least size bytes. It
// grows only for a longer text than any before.
// GCREACH: grows the scratch on a text longer than any before, see
// docs/GarbageCollector.md.
func cxfxTextBufferEnsure(size int) []byte {
	if size > len(cxfxState.CxfxTextBuffer) {
		capacity := TextBufferSizeInitial
		for capacity < size {
			capacity *= 2
		}
		cxfxState.CxfxTextBuffer = make([]byte, capacity)
	}
	return cxfxState.CxfxTextBuffer
}

// cxfxTextNulTerminated copies a GL name into the scratch with a NUL after it
// and returns the C string.
func cxfxTextNulTerminated(text []byte) *uint8 {
	buffer := cxfxTextBufferEnsure(len(text) + 1)
	copy(buffer, text)
	buffer[len(text)] = 0
	return &buffer[0]
}

// cxfxCallbackInputsI32 writes value as input index of an event callback.
func cxfxCallbackInputsI32(index int, value int32) {
	valuetypes.I32Write(cxfxState.CxfxCallbackInputBytes[index][:], 0, value)
	cxfxState.CxfxCallbackInputs[index] = cxfxState.CxfxCallbackInputBytes[index][:4]
}

// cxfxCallbackInputsI64 writes value as input index of an event callback.
func cxfxCallbackInputsI64(index int, value int64) {
	valuetypes.I64Write(cxfxState.CxfxCallbackInputBytes[index][:], 0, value)
	cxfxState.CxfxCallbackInputs[index] = cxfxState.CxfxCallbackInputBytes[index][:8]
}

// cxfxCallbackInputsF64 writes value as input index of an event callback.
func cxfxCallbackInputsF64(index int, value float64) {
	valuetypes.F64Write(cxfxState.CxfxCallbackInputBytes[index][:], 0, value)
	cxfxState.CxfxCallbackInputs[index] = cxfxState.CxfxCallbackInputBytes[index][:8]
}
