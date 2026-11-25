package main

import (
	"testing"

	types "github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-execute"
)

// nativeHostCode is an opcode outside the opcode table, as the playground
// adds app.Rect and the other app functions.
const nativeHostCode = 4096

// A program that embeds CX adds an opcode through ast.NativeFunctionHost and
// a wrapper of ast.OpcodeDispatch, and a CX program calls it by name.
func TestNativeFunctionHost(t *testing.T) {
	ast.NativeNames.NativeNameIndexAdd(nativeHostCode, "os.HostTwice")
	ast.NativeFunctionHost = func(prgrm *ast.CXProgram, code int) *ast.CXNativeFunction {
		if code != nativeHostCode {
			return nil
		}
		return &ast.CXNativeFunction{AtomicOPCode: code, Inputs: []*ast.CXArgument{ast.Param(types.I32)}, Outputs: []*ast.CXArgument{ast.Param(types.I32)}}
	}
	defer func() { ast.NativeFunctionHost = nil }()
	if !ast.NativeLinked(nativeHostCode) || ast.NativeLinked(nativeHostCode+1) {
		t.Fatal("NativeLinked does not ask NativeFunctionHost")
	}

	compileRunSources(t, []string{"host.cx"}, []string{`package main

import "sys/os"

func main() {
	test(os.HostTwice(21), 42, "a host opcode")
}
`})
	calls := 0
	core := ast.OpcodeDispatch
	ast.OpcodeDispatch = func(prgrm *ast.CXProgram, code int, inputs []ast.CXValue, outputs []ast.CXValue) bool {
		if code != nativeHostCode {
			return core(prgrm, code, inputs, outputs)
		}
		calls++
		outputs[0].Set_i32(prgrm, 2*inputs[0].Get_i32(prgrm))
		return true
	}
	defer func() { ast.OpcodeDispatch = core }()
	if err := execute.RunCompiled(actions.AST, 0, nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("the host opcode ran %d times, want 1", calls)
	}
}
