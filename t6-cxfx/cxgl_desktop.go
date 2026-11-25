//go:build cxfx

package cxfx

import (
	"github.com/go-gl/gl/v3.2-compatibility/gl"
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t3-ast"
)

const (
	cxglCLAMP_TO_EDGE               = gl.CLAMP_TO_EDGE
	cxglNEAREST                     = gl.NEAREST
	cxglRGB                         = gl.RGB
	cxglRGBA                        = gl.RGBA
	cxglRGBA8                       = gl.RGBA8
	cxglRGB16F                      = gl.RGB16F
	cxglTEXTURE_2D                  = gl.TEXTURE_2D
	cxglTEXTURE_CUBE_MAP            = gl.TEXTURE_CUBE_MAP
	cxglTEXTURE_CUBE_MAP_POSITIVE_X = gl.TEXTURE_CUBE_MAP_POSITIVE_X
	cxglTEXTURE_MAG_FILTER          = gl.TEXTURE_MAG_FILTER
	cxglTEXTURE_MIN_FILTER          = gl.TEXTURE_MIN_FILTER
	cxglTEXTURE_WRAP_R              = gl.TEXTURE_WRAP_R
	cxglTEXTURE_WRAP_S              = gl.TEXTURE_WRAP_S
	cxglTEXTURE_WRAP_T              = gl.TEXTURE_WRAP_T
	cxglUNSIGNED_BYTE               = gl.UNSIGNED_BYTE
	cxglFLOAT                       = gl.FLOAT
)

// gogl
func OpGlInit(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	gl.Init()
}

// cxgl_1_0
func cxglTexParameteri(target uint32, pname uint32, param int32) {
	gl.TexParameteri(target, pname, param)
}

func cxglTexImage2D(target uint32, level int32, internalFormat int32, width int32, height int32, border int32, format uint32, gltype uint32, pixels []uint8) {
	gl.TexImage2D(target, level, internalFormat, width, height, border, format, gltype, Slice_ui8_ToPtr(pixels))
}

func cxglEnable(cap uint32) {
	gl.Enable(cap)
}

func cxglGetTexLevelParameteriv(target uint32, level int32, pname uint32, params *int32) {
	gl.GetTexLevelParameteriv(target, level, pname, params)
}

// cxgl_1_1
func cxglBindTexture(target uint32, texture uint32) {
	gl.BindTexture(target, texture)
}

func cxglDeleteTextures(n int32, textures *uint32) {
	if n > 1 {
		panic(types.CX_RUNTIME_NOT_IMPLEMENTED)
	}
	gl.DeleteTextures(n, textures)
}

func cxglGenTextures(n int32, textures *uint32) {
	if n > 1 {
		panic(types.CX_RUNTIME_NOT_IMPLEMENTED)
	}
	gl.GenTextures(n, textures)
}

// cxgl_1_5
func cxglDeleteBuffers(n int32, buffers *uint32) {
	if n > 1 {
		panic(types.CX_RUNTIME_NOT_IMPLEMENTED)
	}
	gl.DeleteBuffers(n, buffers)
}

func cxglGenBuffers(n int32, buffers *uint32) {
	if n > 1 {
		panic(types.CX_RUNTIME_NOT_IMPLEMENTED)
	}
	gl.GenBuffers(n, buffers)
}

// cxgl_2_0
func cxglBufferData(target uint32, size int, data []uint8, usage uint32) {
	gl.BufferData(target, size, Slice_ui8_ToPtr(data), usage)
}

func cxglBufferSubData(target uint32, offset int, size int, data []uint8) {
	gl.BufferSubData(target, offset, size, Slice_ui8_ToPtr(data))
}

func cxglDrawBuffers(size int32, bufs []uint8) {
	gl.DrawBuffers(size, (*uint32)(Slice_ui8_ToPtr(bufs)))
}

func cxglBindAttribLocation(program uint32, index uint32, name []byte) {
	gl.BindAttribLocation(program, index, cxfxTextNulTerminated(name))
}

func cgxlDeleteProgram(program uint32) {
	gl.DeleteProgram(program)
}

func cxglGetAttribLocation(program uint32, name []byte) int32 {
	return gl.GetAttribLocation(program, cxfxTextNulTerminated(name))
}

func cxglGetProgramiv(program uint32, pname uint32) int32 {
	var params int32
	gl.GetProgramiv(program, pname, &params)
	return params
}

// cxglGetProgramInfoLog returns the info log in the text scratch, valid until
// the next GL name or log is read.
func cxglGetProgramInfoLog(program uint32, maxLen int32) []byte {
	buffer := cxfxTextBufferEnsure(int(max(maxLen, 0)) + 1)
	var length int32
	gl.GetProgramInfoLog(program, max(maxLen, 0), &length, &buffer[0])
	return buffer[:length]
}

func cxglGetShaderiv(shader uint32, pname uint32) int32 {
	var params int32
	gl.GetShaderiv(shader, pname, &params)
	return params
}

// cxglGetShaderInfoLog returns the info log in the text scratch, valid until
// the next GL name or log is read.
func cxglGetShaderInfoLog(shader uint32, maxLen int32) []byte {
	buffer := cxfxTextBufferEnsure(int(max(maxLen, 0)) + 1)
	var length int32
	gl.GetShaderInfoLog(shader, max(maxLen, 0), &length, &buffer[0])
	return buffer[:length]
}

func cxglGetUniformLocation(program uint32, name []byte) int32 {
	return gl.GetUniformLocation(program, cxfxTextNulTerminated(name))
}

// cxglShaderSource hands GL the source where it is in program memory with its
// length, so it is not copied. The memory stays pinned while GL reads it.
// GCREACH: allocates the argument holder once, see docs/GarbageCollector.md.
func cxglShaderSource(shader uint32, count int32, source []byte) {
	if count > 1 {
		panic(types.CX_RUNTIME_NOT_IMPLEMENTED)
	}
	if cxfxState.CxfxShaderSource == nil {
		cxfxState.CxfxShaderSource = &CxfxShaderSourceArgument{}
	}
	argument := cxfxState.CxfxShaderSource
	if len(source) == 0 {
		argument.CxfxShaderSourcePointer = cxfxTextNulTerminated(source)
	} else {
		argument.CxfxShaderSourcePointer = &source[0]
	}
	argument.CxfxShaderSourceLength = int32(len(source))
	cxfxState.CxfxShaderSourcePinner.Pin(argument.CxfxShaderSourcePointer)
	gl.ShaderSource(shader, count, &argument.CxfxShaderSourcePointer, &argument.CxfxShaderSourceLength)
	cxfxState.CxfxShaderSourcePinner.Unpin()
	argument.CxfxShaderSourcePointer = nil
}

func cxglUniform1fv(location int32, count int32, value []uint8) {
	gl.Uniform1fv(location, count, (*float32)(Slice_ui8_ToPtr(value)))
}

func cxglUniform2fv(location int32, count int32, value []uint8) {
	gl.Uniform2fv(location, count, (*float32)(Slice_ui8_ToPtr(value)))
}

func cxglUniform3fv(location int32, count int32, value []uint8) {
	gl.Uniform3fv(location, count, (*float32)(Slice_ui8_ToPtr(value)))
}

func cxglUniform4fv(location int32, count int32, value []uint8) {
	gl.Uniform4fv(location, count, (*float32)(Slice_ui8_ToPtr(value)))
}

func cxglUniform1iv(location int32, count int32, value []uint8) {
	gl.Uniform1iv(location, count, (*int32)(Slice_ui8_ToPtr(value)))
}

func cxglUniform2iv(location int32, count int32, value []uint8) {
	gl.Uniform2iv(location, count, (*int32)(Slice_ui8_ToPtr(value)))
}

func cxglUniform3iv(location int32, count int32, value []uint8) {
	gl.Uniform3iv(location, count, (*int32)(Slice_ui8_ToPtr(value)))
}

func cxglUniform4iv(location int32, count int32, value []uint8) {
	gl.Uniform4iv(location, count, (*int32)(Slice_ui8_ToPtr(value)))
}

func cxglUniformMatrix2fv(location int32, count int32, transpose bool, value []uint8) {
	gl.UniformMatrix2fv(location, count, transpose, (*float32)(Slice_ui8_ToPtr(value)))
}

func cxglUniformMatrix3fv(location int32, count int32, transpose bool, value []uint8) {
	gl.UniformMatrix3fv(location, count, transpose, (*float32)(Slice_ui8_ToPtr(value)))
}

func cxglUniformMatrix4fv(location int32, count int32, transpose bool, value []uint8) {
	gl.UniformMatrix4fv(location, count, transpose, (*float32)(Slice_ui8_ToPtr(value)))
}

// cxgl_3_0
func cxglClearBufferiv(buffer uint32, drawBuffer int32, value []int32) {
	gl.ClearBufferiv(buffer, drawBuffer, (*int32)(Slice_i32_ToPtr(value)))
}

func cxglClearBufferuiv(buffer uint32, drawBuffer int32, value []uint32) {
	gl.ClearBufferuiv(buffer, drawBuffer, (*uint32)(Slice_ui32_ToPtr(value)))
}

func cxglClearBufferfv(buffer uint32, drawBuffer int32, value []float32) {
	gl.ClearBufferfv(buffer, drawBuffer, (*float32)(Slice_f32_ToPtr(value)))
}

func cxglDeleteRenderbuffers(n int32, renderbuffers *uint32) {
	if n > 1 {
		panic(types.CX_RUNTIME_NOT_IMPLEMENTED)
	}
	gl.DeleteRenderbuffers(n, renderbuffers)
}

func cxglGenRenderbuffers(n int32, renderbuffers *uint32) {
	if n > 1 {
		panic(types.CX_RUNTIME_NOT_IMPLEMENTED)
	}
	gl.GenRenderbuffers(n, renderbuffers)
}

func cxglDeleteFramebuffers(n int32, framebuffers *uint32) {
	if n > 1 {
		panic(types.CX_RUNTIME_NOT_IMPLEMENTED)
	}
	gl.DeleteFramebuffers(n, framebuffers)
}

func cxglGenFramebuffers(n int32, framebuffers *uint32) {
	if n > 1 {
		panic(types.CX_RUNTIME_NOT_IMPLEMENTED)
	}
	gl.GenFramebuffers(n, framebuffers)
}

func cxglBindVertexArrayAPPLE(array uint32) {
	gl.BindVertexArrayAPPLE(array)
}

func cxglBindVertexArray(array uint32) {
	gl.BindVertexArray(array)
}

func cxglDeleteVertexArraysAPPLE(n int32, arrays *uint32) {
	if n > 1 {
		panic(types.CX_RUNTIME_NOT_IMPLEMENTED)
	}
	gl.DeleteVertexArraysAPPLE(n, arrays)
}

func cxglDeleteVertexArrays(n int32, arrays *uint32) {
	if n > 1 {
		panic(types.CX_RUNTIME_NOT_IMPLEMENTED)
	}
	gl.DeleteVertexArrays(n, arrays)
}

func cxglGenVertexArraysAPPLE(n int32, arrays *uint32) {
	if n > 1 {
		panic(types.CX_RUNTIME_NOT_IMPLEMENTED)
	}
	gl.GenVertexArraysAPPLE(n, arrays)
}

func cxglGenVertexArrays(n int32, arrays *uint32) {
	if n > 1 {
		panic(types.CX_RUNTIME_NOT_IMPLEMENTED)
	}
	gl.GenVertexArrays(n, arrays)
}

// gl_0_0
func OpGlLightfv(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	// pointers
	panic("gl.Lightfv")
}
