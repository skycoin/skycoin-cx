//go:build cxfx

package cxfx

import (
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-opcodes" // thin: only In, Out, RegisterFunction; registration API, keep
)

func RegisterPackage(prgrm *ast.CXProgram) {
	// gogl
	opcodes.RegisterFunction(prgrm, "gl.Init", OpGlInit, nil, nil)
	opcodes.RegisterFunction(prgrm, "gl.NewTexture", OpGlNewTexture, opcodes.In(ast.ConstCxArg_STR), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.NewTextureCube", OpGlNewTextureCube, opcodes.In(ast.ConstCxArg_STR, ast.ConstCxArg_STR), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.NewGIF", OpGlNewGIF, opcodes.In(ast.ConstCxArg_STR), opcodes.Out(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.FreeGIF", OpGlFreeGIF, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.GIFFrameToTexture", OpGlGIFFrameToTexture, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32, ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.UploadImageToTexture", OpGlUploadImageToTexture, opcodes.In(ast.ConstCxArg_STR, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_BOOL), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.CxReleaseTexture", OpCxReleaseTexture, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.CxTextureGetPixel", OpCxTextureGetPixel, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32))

	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1431.
	opcodes.RegisterFunction(prgrm, "gl.AppendF32", OpGlAppend, opcodes.In(ast.Slice(types.UI8), ast.ConstCxArg_F32), opcodes.Out(ast.Slice(types.UI8)))
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1432.
	opcodes.RegisterFunction(prgrm, "gl.AppendUI16", OpGlAppend, opcodes.In(ast.Slice(types.UI8), ast.ConstCxArg_UI16), opcodes.Out(ast.Slice(types.UI8)))
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1433.
	opcodes.RegisterFunction(prgrm, "gl.AppendUI32", OpGlAppend, opcodes.In(ast.Slice(types.UI8), ast.ConstCxArg_UI32), opcodes.Out(ast.Slice(types.UI8)))

	// gl_0.0
	opcodes.RegisterFunction(prgrm, "gl.MatrixMode", OpGlMatrixMode, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Rotatef", OpGlRotatef, opcodes.In(ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Translatef", OpGlTranslatef, opcodes.In(ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.LoadIdentity", OpGlLoadIdentity, nil, nil)
	opcodes.RegisterFunction(prgrm, "gl.PushMatrix", OpGlPushMatrix, nil, nil)
	opcodes.RegisterFunction(prgrm, "gl.PopMatrix", OpGlPopMatrix, nil, nil)
	opcodes.RegisterFunction(prgrm, "gl.EnableClientState", OpGlEnableClientState, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Color3f", OpGlColor3f, opcodes.In(ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Color4f", OpGlColor4f, opcodes.In(ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Begin", OpGlBegin, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.End", OpGlEnd, nil, nil)
	opcodes.RegisterFunction(prgrm, "gl.Normal3f", OpGlNormal3f, opcodes.In(ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Vertex2f", OpGlVertex2f, opcodes.In(ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Vertex3f", OpGlVertex3f, opcodes.In(ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Lightfv", OpGlLightfv, opcodes.In(ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Frustum", OpGlFrustum, opcodes.In(ast.ConstCxArg_F64, ast.ConstCxArg_F64, ast.ConstCxArg_F64, ast.ConstCxArg_F64, ast.ConstCxArg_F64, ast.ConstCxArg_F64), nil)
	opcodes.RegisterFunction(prgrm, "gl.TexEnvi", OpGlTexEnvi, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Ortho", OpGlOrtho, opcodes.In(ast.ConstCxArg_F64, ast.ConstCxArg_F64, ast.ConstCxArg_F64, ast.ConstCxArg_F64, ast.ConstCxArg_F64, ast.ConstCxArg_F64), nil)
	opcodes.RegisterFunction(prgrm, "gl.Scalef", OpGlScalef, opcodes.In(ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.TexCoord2d", OpGlTexCoord2d, opcodes.In(ast.ConstCxArg_F64, ast.ConstCxArg_F64), nil)
	opcodes.RegisterFunction(prgrm, "gl.TexCoord2f", OpGlTexCoord2f, opcodes.In(ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)

	// gl_1_0
	opcodes.RegisterFunction(prgrm, "gl.CullFace", OpGlCullFace, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.FrontFace", OpGlFrontFace, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Hint", OpGlHint, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Scissor", OpGlScissor, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.TexParameteri", OpGlTexParameteri, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1434.
	opcodes.RegisterFunction(prgrm, "gl.TexImage2D", OpGlTexImage2D, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.Slice(types.UI32)), nil)
	opcodes.RegisterFunction(prgrm, "gl.Clear", OpGlClear, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.ClearColor", OpGlClearColor, opcodes.In(ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.ClearStencil", OpGlClearStencil, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.ClearDepth", OpGlClearDepth, opcodes.In(ast.ConstCxArg_F64), nil)
	opcodes.RegisterFunction(prgrm, "gl.StencilMask", OpGlStencilMask, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.ColorMask", OpGlColorMask, opcodes.In(ast.ConstCxArg_BOOL, ast.ConstCxArg_BOOL, ast.ConstCxArg_BOOL, ast.ConstCxArg_BOOL), nil)
	opcodes.RegisterFunction(prgrm, "gl.DepthMask", OpGlDepthMask, opcodes.In(ast.ConstCxArg_BOOL), nil)
	opcodes.RegisterFunction(prgrm, "gl.Disable", OpGlDisable, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Enable", OpGlEnable, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.BlendFunc", OpGlBlendFunc, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.StencilFunc", OpGlStencilFunc, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.StencilOp", OpGlStencilOp, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.DepthFunc", OpGlDepthFunc, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.GetError", OpGlGetError, nil, opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.GetTexLevelParameteriv", OpGlGetTexLevelParameteriv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.DepthRange", OpGlDepthRange, opcodes.In(ast.ConstCxArg_F64, ast.ConstCxArg_F64), nil)
	opcodes.RegisterFunction(prgrm, "gl.Viewport", OpGlViewport, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)

	// gl_1_1
	opcodes.RegisterFunction(prgrm, "gl.DrawArrays", OpGlDrawArrays, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.DrawElements", OpGlDrawElements, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.BindTexture", OpGlBindTexture, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.DeleteTextures", OpGlDeleteTextures, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.GenTextures", OpGlGenTextures, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))

	// gl_1_3
	opcodes.RegisterFunction(prgrm, "gl.ActiveTexture", OpGlActiveTexture, opcodes.In(ast.ConstCxArg_I32), nil)

	// gl_1_4
	opcodes.RegisterFunction(prgrm, "gl.BlendFuncSeparate", OpGlBlendFuncSeparate, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)

	// gl_1_5
	opcodes.RegisterFunction(prgrm, "gl.BindBuffer", OpGlBindBuffer, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.DeleteBuffers", OpGlDeleteBuffers, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.GenBuffers", OpGlGenBuffers, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1435.
	opcodes.RegisterFunction(prgrm, "gl.BufferData", OpGlBufferData, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.Slice(types.UI8), ast.ConstCxArg_I32), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1436.
	opcodes.RegisterFunction(prgrm, "gl.BufferSubData", OpGlBufferSubData, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.Slice(types.UI8)), nil)

	//gl_2_0
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1437.
	opcodes.RegisterFunction(prgrm, "gl.DrawBuffers", OpGlDrawBuffers, opcodes.In(ast.ConstCxArg_I32, ast.Slice(types.UI32)), nil)
	opcodes.RegisterFunction(prgrm, "gl.StencilOpSeparate", OpGlStencilOpSeparate, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.StencilFuncSeparate", OpGlStencilFuncSeparate, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.StencilMaskSeparate", OpGlStencilMaskSeparate, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.AttachShader", OpGlAttachShader, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.BindAttribLocation", OpGlBindAttribLocation, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_STR), nil)
	opcodes.RegisterFunction(prgrm, "gl.CompileShader", OpGlCompileShader, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.CreateProgram", OpGlCreateProgram, nil, opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.CreateShader", OpGlCreateShader, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.DeleteProgram", OpGlDeleteProgram, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.DeleteShader", OpGlDeleteShader, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.DetachShader", OpGlDetachShader, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.EnableVertexAttribArray", OpGlEnableVertexAttribArray, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.GetAttribLocation", OpGlGetAttribLocation, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_STR), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.GetProgramiv", OpGlGetProgramiv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.GetProgramInfoLog", OpGlGetProgramInfoLog, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_STR))
	opcodes.RegisterFunction(prgrm, "gl.GetShaderiv", OpGlGetShaderiv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.GetShaderInfoLog", OpGlGetShaderInfoLog, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_STR))
	opcodes.RegisterFunction(prgrm, "gl.GetUniformLocation", OpGlGetUniformLocation, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_STR), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.LinkProgram", OpGlLinkProgram, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.ShaderSource", OpGlShaderSource, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_STR), nil)
	opcodes.RegisterFunction(prgrm, "gl.UseProgram", OpGlUseProgram, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Uniform1f", OpGlUniform1f, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Uniform2f", OpGlUniform2f, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Uniform3f", OpGlUniform3f, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Uniform4f", OpGlUniform4f, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Uniform1i", OpGlUniform1i, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Uniform2i", OpGlUniform2i, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Uniform3i", OpGlUniform3i, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.Uniform4i", OpGlUniform4i, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1438.
	opcodes.RegisterFunction(prgrm, "gl.Uniform1fv", OpGlUniform1fv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_F32, ast.Slice(types.F32)), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1439.
	opcodes.RegisterFunction(prgrm, "gl.Uniform2fv", OpGlUniform2fv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_F32, ast.Slice(types.F32)), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1440.
	opcodes.RegisterFunction(prgrm, "gl.Uniform3fv", OpGlUniform3fv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.Slice(types.F32)), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1441.
	opcodes.RegisterFunction(prgrm, "gl.Uniform4fv", OpGlUniform4fv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.Slice(types.F32)), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1442.
	opcodes.RegisterFunction(prgrm, "gl.Uniform1iv", OpGlUniform1iv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.Slice(types.I32)), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1443.
	opcodes.RegisterFunction(prgrm, "gl.Uniform2iv", OpGlUniform2iv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.Slice(types.I32)), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1444.
	opcodes.RegisterFunction(prgrm, "gl.Uniform3iv", OpGlUniform3iv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.Slice(types.I32)), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1445.
	opcodes.RegisterFunction(prgrm, "gl.Uniform4iv", OpGlUniform4iv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.Slice(types.I32)), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1446.
	opcodes.RegisterFunction(prgrm, "gl.UniformMatrix2fv", OpGlUniformMatrix2fv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_BOOL, ast.Slice(types.F32)), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1447.
	opcodes.RegisterFunction(prgrm, "gl.UniformMatrix3fv", OpGlUniformMatrix3fv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_BOOL, ast.Slice(types.F32)), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1448.
	opcodes.RegisterFunction(prgrm, "gl.UniformMatrix4fv", OpGlUniformMatrix4fv, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_BOOL, ast.Slice(types.F32)), nil)
	opcodes.RegisterFunction(prgrm, "gl.UniformV4F", OpGlUniformV4F, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_UND_TYPE), nil)
	opcodes.RegisterFunction(prgrm, "gl.UniformM44F", OpGlUniformM44F, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_BOOL, ast.ConstCxArg_UND_TYPE), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1449.
	opcodes.RegisterFunction(prgrm, "gl.UniformM44FV", OpGlUniformM44FV, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_BOOL, ast.Slice(types.UNDEFINED)), nil)
	opcodes.RegisterFunction(prgrm, "gl.VertexAttribPointer", OpGlVertexAttribPointer, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_BOOL, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.VertexAttribPointerI32", OpGlVertexAttribPointerI32, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_BOOL, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)

	// gl_3_0
	opcodes.RegisterFunction(prgrm, "gl.ClearBufferI", OpGlClearBufferI, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.ClearBufferUI", OpGlClearBufferUI, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_UI32, ast.ConstCxArg_UI32, ast.ConstCxArg_UI32, ast.ConstCxArg_UI32), nil)
	opcodes.RegisterFunction(prgrm, "gl.ClearBufferF", OpGlClearBufferF, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_F32), nil)
	opcodes.RegisterFunction(prgrm, "gl.BindRenderbuffer", OpGlBindRenderbuffer, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.DeleteRenderbuffers", OpGlDeleteRenderbuffers, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.GenRenderbuffers", OpGlGenRenderbuffers, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.RenderbufferStorage", OpGlRenderbufferStorage, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.BindFramebuffer", OpGlBindFramebuffer, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.DeleteFramebuffers", OpGlDeleteFramebuffers, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.GenFramebuffers", OpGlGenFramebuffers, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.CheckFramebufferStatus", OpGlCheckFramebufferStatus, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.FramebufferTexture2D", OpGlFramebufferTexture2D, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.FramebufferRenderbuffer", OpGlFramebufferRenderbuffer, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.GenerateMipmap", OpGlGenerateMipmap, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.BindVertexArray", OpGlBindVertexArray, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.BindVertexArrayCore", OpGlBindVertexArrayCore, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.DeleteVertexArrays", OpGlDeleteVertexArrays, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.DeleteVertexArraysCore", OpGlDeleteVertexArraysCore, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "gl.GenVertexArrays", OpGlGenVertexArrays, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gl.GenVertexArraysCore", OpGlGenVertexArraysCore, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))

	// goglfw
	opcodes.RegisterFunction(prgrm, "glfw.Fullscreen", OpGlfwFullscreen, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_BOOL, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)

	// glfw
	opcodes.RegisterFunction(prgrm, "glfw.Init", OpGlfwInit, nil, nil)
	opcodes.RegisterFunction(prgrm, "glfw.WindowHint", OpGlfwWindowHint, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "glfw.CreateWindow", OpGlfwCreateWindow, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_STR), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "glfw.MakeContextCurrent", OpGlfwMakeContextCurrent, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "glfw.ShouldClose", OpGlfwShouldClose, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_BOOL))
	opcodes.RegisterFunction(prgrm, "glfw.SetShouldClose", OpGlfwSetShouldClose, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_BOOL), nil)
	opcodes.RegisterFunction(prgrm, "glfw.PollEvents", OpGlfwPollEvents, nil, nil)
	opcodes.RegisterFunction(prgrm, "glfw.SwapBuffers", OpGlfwSwapBuffers, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "glfw.GetFramebufferSize", OpGlfwGetFramebufferSize, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32, ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "glfw.GetWindowPos", OpGlfwGetWindowPos, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32, ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "glfw.GetWindowSize", OpGlfwGetWindowSize, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32, ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "glfw.GetTime", OpGlfwGetTime, nil, opcodes.Out(ast.ConstCxArg_F64))
	opcodes.RegisterFunction(prgrm, "glfw.SwapInterval", OpGlfwSwapInterval, opcodes.In(ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "glfw.SetStartCallback", OpGlfwSetStartCallback, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_FUNC), nil)
	opcodes.RegisterFunction(prgrm, "glfw.SetStopCallback", OpGlfwSetStopCallback, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_FUNC), nil)
	opcodes.RegisterFunction(prgrm, "glfw.SetKeyboardCallback", OpGlfwSetKeyboardCallback, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_FUNC), nil)
	opcodes.RegisterFunction(prgrm, "glfw.SetMouseCallback", OpGlfwSetMouseCallback, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_FUNC), nil)
	opcodes.RegisterFunction(prgrm, "glfw.SetFramebufferSizeCallback", OpGlfwSetFramebufferSizeCallback, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_FUNC), nil)
	opcodes.RegisterFunction(prgrm, "glfw.SetWindowPosCallback", OpGlfwSetWindowPosCallback, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_FUNC), nil)
	opcodes.RegisterFunction(prgrm, "glfw.SetWindowSizeCallback", OpGlfwSetWindowSizeCallback, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_FUNC), nil)
	opcodes.RegisterFunction(prgrm, "glfw.SetKeyCallback", OpGlfwSetKeyCallback, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_FUNC), nil)                 // TODO : to deprecate
	opcodes.RegisterFunction(prgrm, "glfw.SetMouseButtonCallback", OpGlfwSetMouseButtonCallback, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_FUNC), nil) // TODO : to deprecate
	opcodes.RegisterFunction(prgrm, "glfw.SetCursorPosCallback", OpGlfwSetCursorPosCallback, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_FUNC), nil)     // TODO : to deprecate
	opcodes.RegisterFunction(prgrm, "glfw.GetCursorPos", OpGlfwGetCursorPos, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_F64, ast.ConstCxArg_F64))
	opcodes.RegisterFunction(prgrm, "glfw.SetInputMode", OpGlfwSetInputMode, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "glfw.SetWindowPos", OpGlfwSetWindowPos, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "glfw.GetKey", OpGlfwGetKey, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "glfw.RegisterCallbackI32I32", OpGlfwFuncI32I32, opcodes.In(ast.ConstCxArg_FUNC), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "glfw.InvokeCallbackI32I32", OpGlfwCallI32I32, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), nil)
	opcodes.RegisterFunction(prgrm, "glfw.GetWindowContentScale", OpGlfwGetWindowContentScale, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_F32, ast.ConstCxArg_F32))
	opcodes.RegisterFunction(prgrm, "glfw.GetMonitorContentScale", OpGlfwGetMonitorContentScale, nil, opcodes.Out(ast.ConstCxArg_F32, ast.ConstCxArg_F32))

	// gltext
	opcodes.RegisterFunction(prgrm, "gltext.LoadTrueType", OpGltextLoadTrueType, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gltext.LoadTrueTypeCore", OpGltextLoadTrueTypeCore, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gltext.Printf", OpGltextPrintf, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_F32, ast.ConstCxArg_F32, ast.ConstCxArg_STR), nil)
	opcodes.RegisterFunction(prgrm, "gltext.Metrics", OpGltextMetrics, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_STR), opcodes.Out(ast.ConstCxArg_I32, ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gltext.Texture", OpGltextTexture, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gltext.NextGlyph", OpGltextNextGlyph, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_STR, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gltext.GlyphBounds", OpGltextGlyphBounds, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32, ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gltext.GlyphMetrics", OpGltextGlyphMetrics, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32, ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "gltext.GlyphInfo", OpGltextGlyphInfo, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32))

	// goal
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1450.
	opcodes.RegisterFunction(prgrm, "al.LoadWav", OpAlLoadWav, opcodes.In(ast.ConstCxArg_STR), opcodes.Out(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_I64, ast.Slice(types.UI8)))

	// openal
	opcodes.RegisterFunction(prgrm, "al.CloseDevice", OpAlCloseDevice, nil, nil)
	opcodes.RegisterFunction(prgrm, "al.DeleteBuffers", OpAlDeleteBuffers, opcodes.In(ast.ConstCxArg_I32), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1451.
	opcodes.RegisterFunction(prgrm, "al.DeleteSources", OpAlDeleteSources, opcodes.In(ast.Slice(types.I32)), nil)
	opcodes.RegisterFunction(prgrm, "al.DeviceError", OpAlDeviceError, nil, opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "al.Error", OpAlError, nil, opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "al.Extensions", OpAlExtensions, nil, opcodes.Out(ast.ConstCxArg_STR))
	opcodes.RegisterFunction(prgrm, "al.OpenDevice", OpAlOpenDevice, nil, nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1452.
	opcodes.RegisterFunction(prgrm, "al.PauseSources", OpAlPauseSources, opcodes.In(ast.Slice(types.I32)), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1453.
	opcodes.RegisterFunction(prgrm, "al.PlaySources", OpAlPlaySources, opcodes.In(ast.Slice(types.I32)), nil)
	opcodes.RegisterFunction(prgrm, "al.Renderer", OpAlRenderer, nil, opcodes.Out(ast.ConstCxArg_STR))
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1454.
	opcodes.RegisterFunction(prgrm, "al.RewindSources", OpAlRewindSources, opcodes.In(ast.Slice(types.I32)), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1455.
	opcodes.RegisterFunction(prgrm, "al.StopSources", OpAlStopSources, opcodes.In(ast.Slice(types.I32)), nil)
	opcodes.RegisterFunction(prgrm, "al.Vendor", OpAlVendor, nil, opcodes.Out(ast.ConstCxArg_STR))
	opcodes.RegisterFunction(prgrm, "al.Version", OpAlVersion, nil, opcodes.Out(ast.ConstCxArg_STR))
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1456.
	opcodes.RegisterFunction(prgrm, "al.GenBuffers", OpAlGenBuffers, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.Slice(types.I32)))
	opcodes.RegisterFunction(prgrm, "al.BufferData", OpAlBufferData, opcodes.In(ast.ConstCxArg_I32, ast.ConstCxArg_I32, ast.ConstCxArg_UND_TYPE, ast.ConstCxArg_I32), nil)
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1457.
	opcodes.RegisterFunction(prgrm, "al.GenSources", OpAlGenSources, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.Slice(types.I32)))
	opcodes.RegisterFunction(prgrm, "al.SourceBuffersProcessed", OpAlSourceBuffersProcessed, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	opcodes.RegisterFunction(prgrm, "al.SourceBuffersQueued", OpAlSourceBuffersQueued, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1458.
	opcodes.RegisterFunction(prgrm, "al.SourceQueueBuffers", OpAlSourceQueueBuffers, opcodes.In(ast.ConstCxArg_I32, ast.Slice(types.I32)), nil)
	opcodes.RegisterFunction(prgrm, "al.SourceState", OpAlSourceState, opcodes.In(ast.ConstCxArg_I32), opcodes.Out(ast.ConstCxArg_I32))
	// OBJMODEL: state.CXArgumentMng.CXArgumentAlloc(), phase 4 #326.
	// Now a constructor call, site S1459.
	opcodes.RegisterFunction(prgrm, "al.SourceUnqueueBuffers", OpAlSourceUnqueueBuffers, opcodes.In(ast.ConstCxArg_I32, ast.Slice(types.I32)), nil)
}
