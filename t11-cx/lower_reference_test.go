package main

import (
	"encoding/binary"
	"math"
	"strconv"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// referenceFrame is a caller saved by OpcodeCall.
type referenceFrame struct {
	frameFunction uint32
	framePc       uint32
	framePointer  uint32
}

// referenceMachine runs an ir.Image with one switch loop. It is the
// reference for the lowering test of step 4 of #408, and the start of the vm
// of step 5. Run allocates nothing. Errors stop the run with a code.
type referenceMachine struct {
	image        *ir.Image
	memory       []byte
	frames       [ir.ImageCallDepthMax]referenceFrame
	output       []byte
	testFailures uint32
	stopCode     int32
	stopFunction uint32
	stopPc       uint32
}

// referenceMachineNew sizes the memory and output of a machine for image.
func referenceMachineNew(image *ir.Image) *referenceMachine {
	machine := &referenceMachine{image: image}
	machine.memory = make([]byte, int(image.ImageDataStart)+len(image.ImageData))
	machine.output = make([]byte, 0, 1<<16)
	return machine
}

// referenceReset clears the stack and output and reloads the data segment.
func referenceReset(machine *referenceMachine) {
	clear(machine.memory[:machine.image.ImageDataStart])
	copy(machine.memory[machine.image.ImageDataStart:], machine.image.ImageData)
	machine.output = machine.output[:0]
	machine.testFailures = 0
	machine.stopCode = types.CX_SUCCESS
}

// referenceRun runs the init function, then main in the frame after it, as
// execute.RunCompiled does.
func referenceRun(machine *referenceMachine) int32 {
	image := machine.image
	stackPointer := uint32(0)
	if image.ImageEntryInit != ir.ImageFuncNone {
		stackPointer = image.ImageFuncs[image.ImageEntryInit].FuncFrameSize
		if code := referenceInvoke(machine, image.ImageEntryInit, 0, stackPointer); code != types.CX_SUCCESS {
			return code
		}
	}
	if image.ImageEntryMain == ir.ImageFuncNone {
		return types.CX_SUCCESS
	}
	return referenceInvoke(machine, image.ImageEntryMain, stackPointer, stackPointer+image.ImageFuncs[image.ImageEntryMain].FuncFrameSize)
}

// referenceStop ends a run with code at pc of fn.
func referenceStop(machine *referenceMachine, code int32, fn uint32, pc uint32) int32 {
	machine.stopCode = code
	machine.stopFunction = fn
	machine.stopPc = pc - 1
	return code
}

// referenceInvoke runs entry with its frame at fp and the next free stack
// byte at sp until entry returns.
func referenceInvoke(machine *referenceMachine, entry uint32, fp uint32, sp uint32) int32 {
	funcs := machine.image.ImageFuncs
	stackSize := machine.image.ImageStackSize
	mem := machine.memory
	fn := entry
	code := funcs[fn].FuncCode
	pc := uint32(0)
	depth := uint32(0)
	for {
		in := &code[pc]
		pc++
		mode := uint32(in.InstrMode)
		a := in.InstrA + fp&-(mode&1)
		b := in.InstrB + fp&-(mode>>1&1)
		c := in.InstrC + fp&-(mode>>2&1)
		switch in.InstrOpcode {
		case ir.OpcodeReturn:
			if depth == 0 {
				return types.CX_SUCCESS
			}
			depth--
			frame := &machine.frames[depth]
			sp = fp
			fn, pc, fp = frame.frameFunction, frame.framePc, frame.framePointer
			code = funcs[fn].FuncCode
		case ir.OpcodeJump:
			pc = in.InstrM
		case ir.OpcodeJumpIf:
			if mem[a] != 0 {
				pc = in.InstrM
			} else {
				pc = in.InstrN
			}
		case ir.OpcodeZero:
			clear(mem[a : a+in.InstrN])
		case ir.OpcodeMove:
			copy(mem[c:c+in.InstrN], mem[a:a+in.InstrN])
		case ir.OpcodeCallFrame:
			if sp+in.InstrN > stackSize {
				return referenceStop(machine, types.CX_RUNTIME_STACK_OVERFLOW_ERROR, fn, pc)
			}
			clear(mem[sp : sp+in.InstrN])
		case ir.OpcodeCallArg:
			copy(mem[sp+in.InstrC:sp+in.InstrC+in.InstrN], mem[a:a+in.InstrN])
		case ir.OpcodeCall:
			if depth+1 >= ir.ImageCallDepthMax {
				return referenceStop(machine, types.CX_RUNTIME_STACK_OVERFLOW_ERROR, fn, pc)
			}
			machine.frames[depth] = referenceFrame{frameFunction: fn, framePc: pc, framePointer: fp}
			depth++
			fn = in.InstrM
			fp = sp
			sp += funcs[fn].FuncFrameSize
			code = funcs[fn].FuncCode
			pc = 0
		case ir.OpcodeCallResult:
			copy(mem[c:c+in.InstrN], mem[sp+in.InstrA:sp+in.InstrA+in.InstrN])
		case ir.OpcodeTestEqual:
			if in.InstrN != in.InstrM || string(mem[a:a+in.InstrN]) != string(mem[b:b+in.InstrM]) {
				machine.testFailures++
			}
		case ir.OpcodeBoolNot:
			wBool(mem, c, !rBool(mem, a))
		case ir.OpcodeBoolAnd:
			wBool(mem, c, rBool(mem, a) && rBool(mem, b))
		case ir.OpcodeBoolOr:
			wBool(mem, c, rBool(mem, a) || rBool(mem, b))
		case ir.OpcodeBoolEq:
			wBool(mem, c, rBool(mem, a) == rBool(mem, b))
		case ir.OpcodeBoolUneq:
			wBool(mem, c, rBool(mem, a) != rBool(mem, b))
		case ir.OpcodeBoolPrint:
			machine.output = strconv.AppendBool(machine.output, rBool(mem, a))
			machine.output = append(machine.output, '\n')
		case ir.OpcodeI8Add:
			wI8(mem, c, rI8(mem, a)+rI8(mem, b))
		case ir.OpcodeI8Sub:
			wI8(mem, c, rI8(mem, a)-rI8(mem, b))
		case ir.OpcodeI8Mul:
			wI8(mem, c, rI8(mem, a)*rI8(mem, b))
		case ir.OpcodeI8Div:
			d := rI8(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wI8(mem, c, rI8(mem, a)/d)
		case ir.OpcodeI8Mod:
			d := rI8(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wI8(mem, c, rI8(mem, a)%d)
		case ir.OpcodeI8Neg:
			wI8(mem, c, -rI8(mem, a))
		case ir.OpcodeI8Abs:
			v := rI8(mem, a)
			sign := v >> 7
			wI8(mem, c, (v^sign)-sign)
		case ir.OpcodeI8Max:
			v0, v1 := rI8(mem, a), rI8(mem, b)
			if v1 > v0 {
				v0 = v1
			}
			wI8(mem, c, v0)
		case ir.OpcodeI8Min:
			v0, v1 := rI8(mem, a), rI8(mem, b)
			if v1 < v0 {
				v0 = v1
			}
			wI8(mem, c, v0)
		case ir.OpcodeI8Eq:
			wBool(mem, c, rI8(mem, a) == rI8(mem, b))
		case ir.OpcodeI8Uneq:
			wBool(mem, c, rI8(mem, a) != rI8(mem, b))
		case ir.OpcodeI8Lt:
			wBool(mem, c, rI8(mem, a) < rI8(mem, b))
		case ir.OpcodeI8Lteq:
			wBool(mem, c, rI8(mem, a) <= rI8(mem, b))
		case ir.OpcodeI8Gt:
			wBool(mem, c, rI8(mem, a) > rI8(mem, b))
		case ir.OpcodeI8Gteq:
			wBool(mem, c, rI8(mem, a) >= rI8(mem, b))
		case ir.OpcodeI8BitAnd:
			wI8(mem, c, rI8(mem, a)&rI8(mem, b))
		case ir.OpcodeI8BitOr:
			wI8(mem, c, rI8(mem, a)|rI8(mem, b))
		case ir.OpcodeI8BitXor:
			wI8(mem, c, rI8(mem, a)^rI8(mem, b))
		case ir.OpcodeI8BitClear:
			wI8(mem, c, rI8(mem, a)&^rI8(mem, b))
		case ir.OpcodeI8BitShl:
			wI8(mem, c, rI8(mem, a)<<uint8(rI8(mem, b)))
		case ir.OpcodeI8BitShr:
			wI8(mem, c, rI8(mem, a)>>uint8(rI8(mem, b)))
		case ir.OpcodeI8Print:
			machine.output = strconv.AppendInt(machine.output, int64(rI8(mem, a)), 10)
			machine.output = append(machine.output, '\n')
		case ir.OpcodeI16Add:
			wI16(mem, c, rI16(mem, a)+rI16(mem, b))
		case ir.OpcodeI16Sub:
			wI16(mem, c, rI16(mem, a)-rI16(mem, b))
		case ir.OpcodeI16Mul:
			wI16(mem, c, rI16(mem, a)*rI16(mem, b))
		case ir.OpcodeI16Div:
			d := rI16(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wI16(mem, c, rI16(mem, a)/d)
		case ir.OpcodeI16Mod:
			d := rI16(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wI16(mem, c, rI16(mem, a)%d)
		case ir.OpcodeI16Neg:
			wI16(mem, c, -rI16(mem, a))
		case ir.OpcodeI16Abs:
			v := rI16(mem, a)
			sign := v >> 15
			wI16(mem, c, (v^sign)-sign)
		case ir.OpcodeI16Max:
			v0, v1 := rI16(mem, a), rI16(mem, b)
			if v1 > v0 {
				v0 = v1
			}
			wI16(mem, c, v0)
		case ir.OpcodeI16Min:
			v0, v1 := rI16(mem, a), rI16(mem, b)
			if v1 < v0 {
				v0 = v1
			}
			wI16(mem, c, v0)
		case ir.OpcodeI16Eq:
			wBool(mem, c, rI16(mem, a) == rI16(mem, b))
		case ir.OpcodeI16Uneq:
			wBool(mem, c, rI16(mem, a) != rI16(mem, b))
		case ir.OpcodeI16Lt:
			wBool(mem, c, rI16(mem, a) < rI16(mem, b))
		case ir.OpcodeI16Lteq:
			wBool(mem, c, rI16(mem, a) <= rI16(mem, b))
		case ir.OpcodeI16Gt:
			wBool(mem, c, rI16(mem, a) > rI16(mem, b))
		case ir.OpcodeI16Gteq:
			wBool(mem, c, rI16(mem, a) >= rI16(mem, b))
		case ir.OpcodeI16BitAnd:
			wI16(mem, c, rI16(mem, a)&rI16(mem, b))
		case ir.OpcodeI16BitOr:
			wI16(mem, c, rI16(mem, a)|rI16(mem, b))
		case ir.OpcodeI16BitXor:
			wI16(mem, c, rI16(mem, a)^rI16(mem, b))
		case ir.OpcodeI16BitClear:
			wI16(mem, c, rI16(mem, a)&^rI16(mem, b))
		case ir.OpcodeI16BitShl:
			wI16(mem, c, rI16(mem, a)<<uint16(rI16(mem, b)))
		case ir.OpcodeI16BitShr:
			wI16(mem, c, rI16(mem, a)>>uint16(rI16(mem, b)))
		case ir.OpcodeI16Print:
			machine.output = strconv.AppendInt(machine.output, int64(rI16(mem, a)), 10)
			machine.output = append(machine.output, '\n')
		case ir.OpcodeI32Add:
			wI32(mem, c, rI32(mem, a)+rI32(mem, b))
		case ir.OpcodeI32Sub:
			wI32(mem, c, rI32(mem, a)-rI32(mem, b))
		case ir.OpcodeI32Mul:
			wI32(mem, c, rI32(mem, a)*rI32(mem, b))
		case ir.OpcodeI32Div:
			d := rI32(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wI32(mem, c, rI32(mem, a)/d)
		case ir.OpcodeI32Mod:
			d := rI32(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wI32(mem, c, rI32(mem, a)%d)
		case ir.OpcodeI32Neg:
			wI32(mem, c, -rI32(mem, a))
		case ir.OpcodeI32Abs:
			v := rI32(mem, a)
			sign := v >> 31
			wI32(mem, c, (v^sign)-sign)
		case ir.OpcodeI32Max:
			v0, v1 := rI32(mem, a), rI32(mem, b)
			if v1 > v0 {
				v0 = v1
			}
			wI32(mem, c, v0)
		case ir.OpcodeI32Min:
			v0, v1 := rI32(mem, a), rI32(mem, b)
			if v1 < v0 {
				v0 = v1
			}
			wI32(mem, c, v0)
		case ir.OpcodeI32Eq:
			wBool(mem, c, rI32(mem, a) == rI32(mem, b))
		case ir.OpcodeI32Uneq:
			wBool(mem, c, rI32(mem, a) != rI32(mem, b))
		case ir.OpcodeI32Lt:
			wBool(mem, c, rI32(mem, a) < rI32(mem, b))
		case ir.OpcodeI32Lteq:
			wBool(mem, c, rI32(mem, a) <= rI32(mem, b))
		case ir.OpcodeI32Gt:
			wBool(mem, c, rI32(mem, a) > rI32(mem, b))
		case ir.OpcodeI32Gteq:
			wBool(mem, c, rI32(mem, a) >= rI32(mem, b))
		case ir.OpcodeI32BitAnd:
			wI32(mem, c, rI32(mem, a)&rI32(mem, b))
		case ir.OpcodeI32BitOr:
			wI32(mem, c, rI32(mem, a)|rI32(mem, b))
		case ir.OpcodeI32BitXor:
			wI32(mem, c, rI32(mem, a)^rI32(mem, b))
		case ir.OpcodeI32BitClear:
			wI32(mem, c, rI32(mem, a)&^rI32(mem, b))
		case ir.OpcodeI32BitShl:
			wI32(mem, c, rI32(mem, a)<<uint32(rI32(mem, b)))
		case ir.OpcodeI32BitShr:
			wI32(mem, c, rI32(mem, a)>>uint32(rI32(mem, b)))
		case ir.OpcodeI32Print:
			machine.output = strconv.AppendInt(machine.output, int64(rI32(mem, a)), 10)
			machine.output = append(machine.output, '\n')
		case ir.OpcodeI64Add:
			wI64(mem, c, rI64(mem, a)+rI64(mem, b))
		case ir.OpcodeI64Sub:
			wI64(mem, c, rI64(mem, a)-rI64(mem, b))
		case ir.OpcodeI64Mul:
			wI64(mem, c, rI64(mem, a)*rI64(mem, b))
		case ir.OpcodeI64Div:
			d := rI64(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wI64(mem, c, rI64(mem, a)/d)
		case ir.OpcodeI64Mod:
			d := rI64(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wI64(mem, c, rI64(mem, a)%d)
		case ir.OpcodeI64Neg:
			wI64(mem, c, -rI64(mem, a))
		case ir.OpcodeI64Abs:
			v := rI64(mem, a)
			sign := v >> 63
			wI64(mem, c, (v^sign)-sign)
		case ir.OpcodeI64Max:
			v0, v1 := rI64(mem, a), rI64(mem, b)
			if v1 > v0 {
				v0 = v1
			}
			wI64(mem, c, v0)
		case ir.OpcodeI64Min:
			v0, v1 := rI64(mem, a), rI64(mem, b)
			if v1 < v0 {
				v0 = v1
			}
			wI64(mem, c, v0)
		case ir.OpcodeI64Eq:
			wBool(mem, c, rI64(mem, a) == rI64(mem, b))
		case ir.OpcodeI64Uneq:
			wBool(mem, c, rI64(mem, a) != rI64(mem, b))
		case ir.OpcodeI64Lt:
			wBool(mem, c, rI64(mem, a) < rI64(mem, b))
		case ir.OpcodeI64Lteq:
			wBool(mem, c, rI64(mem, a) <= rI64(mem, b))
		case ir.OpcodeI64Gt:
			wBool(mem, c, rI64(mem, a) > rI64(mem, b))
		case ir.OpcodeI64Gteq:
			wBool(mem, c, rI64(mem, a) >= rI64(mem, b))
		case ir.OpcodeI64BitAnd:
			wI64(mem, c, rI64(mem, a)&rI64(mem, b))
		case ir.OpcodeI64BitOr:
			wI64(mem, c, rI64(mem, a)|rI64(mem, b))
		case ir.OpcodeI64BitXor:
			wI64(mem, c, rI64(mem, a)^rI64(mem, b))
		case ir.OpcodeI64BitClear:
			wI64(mem, c, rI64(mem, a)&^rI64(mem, b))
		case ir.OpcodeI64BitShl:
			wI64(mem, c, rI64(mem, a)<<uint64(rI64(mem, b)))
		case ir.OpcodeI64BitShr:
			wI64(mem, c, rI64(mem, a)>>uint64(rI64(mem, b)))
		case ir.OpcodeI64Print:
			machine.output = strconv.AppendInt(machine.output, int64(rI64(mem, a)), 10)
			machine.output = append(machine.output, '\n')
		case ir.OpcodeUI8Add:
			wUI8(mem, c, rUI8(mem, a)+rUI8(mem, b))
		case ir.OpcodeUI8Sub:
			wUI8(mem, c, rUI8(mem, a)-rUI8(mem, b))
		case ir.OpcodeUI8Mul:
			wUI8(mem, c, rUI8(mem, a)*rUI8(mem, b))
		case ir.OpcodeUI8Div:
			d := rUI8(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wUI8(mem, c, rUI8(mem, a)/d)
		case ir.OpcodeUI8Mod:
			d := rUI8(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wUI8(mem, c, rUI8(mem, a)%d)
		case ir.OpcodeUI8Max:
			v0, v1 := rUI8(mem, a), rUI8(mem, b)
			if v1 > v0 {
				v0 = v1
			}
			wUI8(mem, c, v0)
		case ir.OpcodeUI8Min:
			v0, v1 := rUI8(mem, a), rUI8(mem, b)
			if v1 < v0 {
				v0 = v1
			}
			wUI8(mem, c, v0)
		case ir.OpcodeUI8Eq:
			wBool(mem, c, rUI8(mem, a) == rUI8(mem, b))
		case ir.OpcodeUI8Uneq:
			wBool(mem, c, rUI8(mem, a) != rUI8(mem, b))
		case ir.OpcodeUI8Lt:
			wBool(mem, c, rUI8(mem, a) < rUI8(mem, b))
		case ir.OpcodeUI8Lteq:
			wBool(mem, c, rUI8(mem, a) <= rUI8(mem, b))
		case ir.OpcodeUI8Gt:
			wBool(mem, c, rUI8(mem, a) > rUI8(mem, b))
		case ir.OpcodeUI8Gteq:
			wBool(mem, c, rUI8(mem, a) >= rUI8(mem, b))
		case ir.OpcodeUI8BitAnd:
			wUI8(mem, c, rUI8(mem, a)&rUI8(mem, b))
		case ir.OpcodeUI8BitOr:
			wUI8(mem, c, rUI8(mem, a)|rUI8(mem, b))
		case ir.OpcodeUI8BitXor:
			wUI8(mem, c, rUI8(mem, a)^rUI8(mem, b))
		case ir.OpcodeUI8BitClear:
			wUI8(mem, c, rUI8(mem, a)&^rUI8(mem, b))
		case ir.OpcodeUI8BitShl:
			wUI8(mem, c, rUI8(mem, a)<<uint8(rUI8(mem, b)))
		case ir.OpcodeUI8BitShr:
			wUI8(mem, c, rUI8(mem, a)>>uint8(rUI8(mem, b)))
		case ir.OpcodeUI8Print:
			machine.output = strconv.AppendUint(machine.output, uint64(rUI8(mem, a)), 10)
			machine.output = append(machine.output, '\n')
		case ir.OpcodeUI16Add:
			wUI16(mem, c, rUI16(mem, a)+rUI16(mem, b))
		case ir.OpcodeUI16Sub:
			wUI16(mem, c, rUI16(mem, a)-rUI16(mem, b))
		case ir.OpcodeUI16Mul:
			wUI16(mem, c, rUI16(mem, a)*rUI16(mem, b))
		case ir.OpcodeUI16Div:
			d := rUI16(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wUI16(mem, c, rUI16(mem, a)/d)
		case ir.OpcodeUI16Mod:
			d := rUI16(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wUI16(mem, c, rUI16(mem, a)%d)
		case ir.OpcodeUI16Max:
			v0, v1 := rUI16(mem, a), rUI16(mem, b)
			if v1 > v0 {
				v0 = v1
			}
			wUI16(mem, c, v0)
		case ir.OpcodeUI16Min:
			v0, v1 := rUI16(mem, a), rUI16(mem, b)
			if v1 < v0 {
				v0 = v1
			}
			wUI16(mem, c, v0)
		case ir.OpcodeUI16Eq:
			wBool(mem, c, rUI16(mem, a) == rUI16(mem, b))
		case ir.OpcodeUI16Uneq:
			wBool(mem, c, rUI16(mem, a) != rUI16(mem, b))
		case ir.OpcodeUI16Lt:
			wBool(mem, c, rUI16(mem, a) < rUI16(mem, b))
		case ir.OpcodeUI16Lteq:
			wBool(mem, c, rUI16(mem, a) <= rUI16(mem, b))
		case ir.OpcodeUI16Gt:
			wBool(mem, c, rUI16(mem, a) > rUI16(mem, b))
		case ir.OpcodeUI16Gteq:
			wBool(mem, c, rUI16(mem, a) >= rUI16(mem, b))
		case ir.OpcodeUI16BitAnd:
			wUI16(mem, c, rUI16(mem, a)&rUI16(mem, b))
		case ir.OpcodeUI16BitOr:
			wUI16(mem, c, rUI16(mem, a)|rUI16(mem, b))
		case ir.OpcodeUI16BitXor:
			wUI16(mem, c, rUI16(mem, a)^rUI16(mem, b))
		case ir.OpcodeUI16BitClear:
			wUI16(mem, c, rUI16(mem, a)&^rUI16(mem, b))
		case ir.OpcodeUI16BitShl:
			wUI16(mem, c, rUI16(mem, a)<<uint16(rUI16(mem, b)))
		case ir.OpcodeUI16BitShr:
			wUI16(mem, c, rUI16(mem, a)>>uint16(rUI16(mem, b)))
		case ir.OpcodeUI16Print:
			machine.output = strconv.AppendUint(machine.output, uint64(rUI16(mem, a)), 10)
			machine.output = append(machine.output, '\n')
		case ir.OpcodeUI32Add:
			wUI32(mem, c, rUI32(mem, a)+rUI32(mem, b))
		case ir.OpcodeUI32Sub:
			wUI32(mem, c, rUI32(mem, a)-rUI32(mem, b))
		case ir.OpcodeUI32Mul:
			wUI32(mem, c, rUI32(mem, a)*rUI32(mem, b))
		case ir.OpcodeUI32Div:
			d := rUI32(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wUI32(mem, c, rUI32(mem, a)/d)
		case ir.OpcodeUI32Mod:
			d := rUI32(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wUI32(mem, c, rUI32(mem, a)%d)
		case ir.OpcodeUI32Max:
			v0, v1 := rUI32(mem, a), rUI32(mem, b)
			if v1 > v0 {
				v0 = v1
			}
			wUI32(mem, c, v0)
		case ir.OpcodeUI32Min:
			v0, v1 := rUI32(mem, a), rUI32(mem, b)
			if v1 < v0 {
				v0 = v1
			}
			wUI32(mem, c, v0)
		case ir.OpcodeUI32Eq:
			wBool(mem, c, rUI32(mem, a) == rUI32(mem, b))
		case ir.OpcodeUI32Uneq:
			wBool(mem, c, rUI32(mem, a) != rUI32(mem, b))
		case ir.OpcodeUI32Lt:
			wBool(mem, c, rUI32(mem, a) < rUI32(mem, b))
		case ir.OpcodeUI32Lteq:
			wBool(mem, c, rUI32(mem, a) <= rUI32(mem, b))
		case ir.OpcodeUI32Gt:
			wBool(mem, c, rUI32(mem, a) > rUI32(mem, b))
		case ir.OpcodeUI32Gteq:
			wBool(mem, c, rUI32(mem, a) >= rUI32(mem, b))
		case ir.OpcodeUI32BitAnd:
			wUI32(mem, c, rUI32(mem, a)&rUI32(mem, b))
		case ir.OpcodeUI32BitOr:
			wUI32(mem, c, rUI32(mem, a)|rUI32(mem, b))
		case ir.OpcodeUI32BitXor:
			wUI32(mem, c, rUI32(mem, a)^rUI32(mem, b))
		case ir.OpcodeUI32BitClear:
			wUI32(mem, c, rUI32(mem, a)&^rUI32(mem, b))
		case ir.OpcodeUI32BitShl:
			wUI32(mem, c, rUI32(mem, a)<<uint32(rUI32(mem, b)))
		case ir.OpcodeUI32BitShr:
			wUI32(mem, c, rUI32(mem, a)>>uint32(rUI32(mem, b)))
		case ir.OpcodeUI32Print:
			machine.output = strconv.AppendUint(machine.output, uint64(rUI32(mem, a)), 10)
			machine.output = append(machine.output, '\n')
		case ir.OpcodeUI64Add:
			wUI64(mem, c, rUI64(mem, a)+rUI64(mem, b))
		case ir.OpcodeUI64Sub:
			wUI64(mem, c, rUI64(mem, a)-rUI64(mem, b))
		case ir.OpcodeUI64Mul:
			wUI64(mem, c, rUI64(mem, a)*rUI64(mem, b))
		case ir.OpcodeUI64Div:
			d := rUI64(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wUI64(mem, c, rUI64(mem, a)/d)
		case ir.OpcodeUI64Mod:
			d := rUI64(mem, b)
			if d == 0 {
				return referenceStop(machine, types.CX_RUNTIME_ERROR, fn, pc)
			}
			wUI64(mem, c, rUI64(mem, a)%d)
		case ir.OpcodeUI64Max:
			v0, v1 := rUI64(mem, a), rUI64(mem, b)
			if v1 > v0 {
				v0 = v1
			}
			wUI64(mem, c, v0)
		case ir.OpcodeUI64Min:
			v0, v1 := rUI64(mem, a), rUI64(mem, b)
			if v1 < v0 {
				v0 = v1
			}
			wUI64(mem, c, v0)
		case ir.OpcodeUI64Eq:
			wBool(mem, c, rUI64(mem, a) == rUI64(mem, b))
		case ir.OpcodeUI64Uneq:
			wBool(mem, c, rUI64(mem, a) != rUI64(mem, b))
		case ir.OpcodeUI64Lt:
			wBool(mem, c, rUI64(mem, a) < rUI64(mem, b))
		case ir.OpcodeUI64Lteq:
			wBool(mem, c, rUI64(mem, a) <= rUI64(mem, b))
		case ir.OpcodeUI64Gt:
			wBool(mem, c, rUI64(mem, a) > rUI64(mem, b))
		case ir.OpcodeUI64Gteq:
			wBool(mem, c, rUI64(mem, a) >= rUI64(mem, b))
		case ir.OpcodeUI64BitAnd:
			wUI64(mem, c, rUI64(mem, a)&rUI64(mem, b))
		case ir.OpcodeUI64BitOr:
			wUI64(mem, c, rUI64(mem, a)|rUI64(mem, b))
		case ir.OpcodeUI64BitXor:
			wUI64(mem, c, rUI64(mem, a)^rUI64(mem, b))
		case ir.OpcodeUI64BitClear:
			wUI64(mem, c, rUI64(mem, a)&^rUI64(mem, b))
		case ir.OpcodeUI64BitShl:
			wUI64(mem, c, rUI64(mem, a)<<uint64(rUI64(mem, b)))
		case ir.OpcodeUI64BitShr:
			wUI64(mem, c, rUI64(mem, a)>>uint64(rUI64(mem, b)))
		case ir.OpcodeUI64Print:
			machine.output = strconv.AppendUint(machine.output, uint64(rUI64(mem, a)), 10)
			machine.output = append(machine.output, '\n')
		case ir.OpcodeF32Add:
			wF32(mem, c, rF32(mem, a)+rF32(mem, b))
		case ir.OpcodeF32Sub:
			wF32(mem, c, rF32(mem, a)-rF32(mem, b))
		case ir.OpcodeF32Mul:
			wF32(mem, c, rF32(mem, a)*rF32(mem, b))
		case ir.OpcodeF32Div:
			wF32(mem, c, rF32(mem, a)/rF32(mem, b))
		case ir.OpcodeF32Mod:
			wF32(mem, c, float32(math.Mod(float64(rF32(mem, a)), float64(rF32(mem, b)))))
		case ir.OpcodeF32Neg:
			wF32(mem, c, -rF32(mem, a))
		case ir.OpcodeF32Abs:
			wF32(mem, c, float32(math.Abs(float64(rF32(mem, a)))))
		case ir.OpcodeF32Max:
			wF32(mem, c, float32(math.Max(float64(rF32(mem, a)), float64(rF32(mem, b)))))
		case ir.OpcodeF32Min:
			wF32(mem, c, float32(math.Min(float64(rF32(mem, a)), float64(rF32(mem, b)))))
		case ir.OpcodeF32Eq:
			wBool(mem, c, rF32(mem, a) == rF32(mem, b))
		case ir.OpcodeF32Uneq:
			wBool(mem, c, rF32(mem, a) != rF32(mem, b))
		case ir.OpcodeF32Lt:
			wBool(mem, c, rF32(mem, a) < rF32(mem, b))
		case ir.OpcodeF32Lteq:
			wBool(mem, c, rF32(mem, a) <= rF32(mem, b))
		case ir.OpcodeF32Gt:
			wBool(mem, c, rF32(mem, a) > rF32(mem, b))
		case ir.OpcodeF32Gteq:
			wBool(mem, c, rF32(mem, a) >= rF32(mem, b))
		case ir.OpcodeF32Print:
			machine.output = strconv.AppendFloat(machine.output, float64(rF32(mem, a)), 'g', -1, 32)
			machine.output = append(machine.output, '\n')
		case ir.OpcodeF64Add:
			wF64(mem, c, rF64(mem, a)+rF64(mem, b))
		case ir.OpcodeF64Sub:
			wF64(mem, c, rF64(mem, a)-rF64(mem, b))
		case ir.OpcodeF64Mul:
			wF64(mem, c, rF64(mem, a)*rF64(mem, b))
		case ir.OpcodeF64Div:
			wF64(mem, c, rF64(mem, a)/rF64(mem, b))
		case ir.OpcodeF64Mod:
			wF64(mem, c, math.Mod(rF64(mem, a), rF64(mem, b)))
		case ir.OpcodeF64Neg:
			wF64(mem, c, -rF64(mem, a))
		case ir.OpcodeF64Abs:
			wF64(mem, c, math.Abs(rF64(mem, a)))
		case ir.OpcodeF64Max:
			wF64(mem, c, math.Max(rF64(mem, a), rF64(mem, b)))
		case ir.OpcodeF64Min:
			wF64(mem, c, math.Min(rF64(mem, a), rF64(mem, b)))
		case ir.OpcodeF64Eq:
			wBool(mem, c, rF64(mem, a) == rF64(mem, b))
		case ir.OpcodeF64Uneq:
			wBool(mem, c, rF64(mem, a) != rF64(mem, b))
		case ir.OpcodeF64Lt:
			wBool(mem, c, rF64(mem, a) < rF64(mem, b))
		case ir.OpcodeF64Lteq:
			wBool(mem, c, rF64(mem, a) <= rF64(mem, b))
		case ir.OpcodeF64Gt:
			wBool(mem, c, rF64(mem, a) > rF64(mem, b))
		case ir.OpcodeF64Gteq:
			wBool(mem, c, rF64(mem, a) >= rF64(mem, b))
		case ir.OpcodeF64Print:
			machine.output = strconv.AppendFloat(machine.output, float64(rF64(mem, a)), 'g', -1, 64)
			machine.output = append(machine.output, '\n')
		case ir.OpcodeI8ToI16:
			wI16(mem, c, int16(rI8(mem, a)))
		case ir.OpcodeI8ToI32:
			wI32(mem, c, int32(rI8(mem, a)))
		case ir.OpcodeI8ToI64:
			wI64(mem, c, int64(rI8(mem, a)))
		case ir.OpcodeI8ToUI8:
			wUI8(mem, c, uint8(rI8(mem, a)))
		case ir.OpcodeI8ToUI16:
			wUI16(mem, c, uint16(rI8(mem, a)))
		case ir.OpcodeI8ToUI32:
			wUI32(mem, c, uint32(rI8(mem, a)))
		case ir.OpcodeI8ToUI64:
			wUI64(mem, c, uint64(rI8(mem, a)))
		case ir.OpcodeI8ToF32:
			wF32(mem, c, float32(rI8(mem, a)))
		case ir.OpcodeI8ToF64:
			wF64(mem, c, float64(rI8(mem, a)))
		case ir.OpcodeI16ToI8:
			wI8(mem, c, int8(rI16(mem, a)))
		case ir.OpcodeI16ToI32:
			wI32(mem, c, int32(rI16(mem, a)))
		case ir.OpcodeI16ToI64:
			wI64(mem, c, int64(rI16(mem, a)))
		case ir.OpcodeI16ToUI8:
			wUI8(mem, c, uint8(rI16(mem, a)))
		case ir.OpcodeI16ToUI16:
			wUI16(mem, c, uint16(rI16(mem, a)))
		case ir.OpcodeI16ToUI32:
			wUI32(mem, c, uint32(rI16(mem, a)))
		case ir.OpcodeI16ToUI64:
			wUI64(mem, c, uint64(rI16(mem, a)))
		case ir.OpcodeI16ToF32:
			wF32(mem, c, float32(rI16(mem, a)))
		case ir.OpcodeI16ToF64:
			wF64(mem, c, float64(rI16(mem, a)))
		case ir.OpcodeI32ToI8:
			wI8(mem, c, int8(rI32(mem, a)))
		case ir.OpcodeI32ToI16:
			wI16(mem, c, int16(rI32(mem, a)))
		case ir.OpcodeI32ToI64:
			wI64(mem, c, int64(rI32(mem, a)))
		case ir.OpcodeI32ToUI8:
			wUI8(mem, c, uint8(rI32(mem, a)))
		case ir.OpcodeI32ToUI16:
			wUI16(mem, c, uint16(rI32(mem, a)))
		case ir.OpcodeI32ToUI32:
			wUI32(mem, c, uint32(rI32(mem, a)))
		case ir.OpcodeI32ToUI64:
			wUI64(mem, c, uint64(rI32(mem, a)))
		case ir.OpcodeI32ToF32:
			wF32(mem, c, float32(rI32(mem, a)))
		case ir.OpcodeI32ToF64:
			wF64(mem, c, float64(rI32(mem, a)))
		case ir.OpcodeI64ToI8:
			wI8(mem, c, int8(rI64(mem, a)))
		case ir.OpcodeI64ToI16:
			wI16(mem, c, int16(rI64(mem, a)))
		case ir.OpcodeI64ToI32:
			wI32(mem, c, int32(rI64(mem, a)))
		case ir.OpcodeI64ToUI8:
			wUI8(mem, c, uint8(rI64(mem, a)))
		case ir.OpcodeI64ToUI16:
			wUI16(mem, c, uint16(rI64(mem, a)))
		case ir.OpcodeI64ToUI32:
			wUI32(mem, c, uint32(rI64(mem, a)))
		case ir.OpcodeI64ToUI64:
			wUI64(mem, c, uint64(rI64(mem, a)))
		case ir.OpcodeI64ToF32:
			wF32(mem, c, float32(rI64(mem, a)))
		case ir.OpcodeI64ToF64:
			wF64(mem, c, float64(rI64(mem, a)))
		case ir.OpcodeUI8ToI8:
			wI8(mem, c, int8(rUI8(mem, a)))
		case ir.OpcodeUI8ToI16:
			wI16(mem, c, int16(rUI8(mem, a)))
		case ir.OpcodeUI8ToI32:
			wI32(mem, c, int32(rUI8(mem, a)))
		case ir.OpcodeUI8ToI64:
			wI64(mem, c, int64(rUI8(mem, a)))
		case ir.OpcodeUI8ToUI16:
			wUI16(mem, c, uint16(rUI8(mem, a)))
		case ir.OpcodeUI8ToUI32:
			wUI32(mem, c, uint32(rUI8(mem, a)))
		case ir.OpcodeUI8ToUI64:
			wUI64(mem, c, uint64(rUI8(mem, a)))
		case ir.OpcodeUI8ToF32:
			wF32(mem, c, float32(rUI8(mem, a)))
		case ir.OpcodeUI8ToF64:
			wF64(mem, c, float64(rUI8(mem, a)))
		case ir.OpcodeUI16ToI8:
			wI8(mem, c, int8(rUI16(mem, a)))
		case ir.OpcodeUI16ToI16:
			wI16(mem, c, int16(rUI16(mem, a)))
		case ir.OpcodeUI16ToI32:
			wI32(mem, c, int32(rUI16(mem, a)))
		case ir.OpcodeUI16ToI64:
			wI64(mem, c, int64(rUI16(mem, a)))
		case ir.OpcodeUI16ToUI8:
			wUI8(mem, c, uint8(rUI16(mem, a)))
		case ir.OpcodeUI16ToUI32:
			wUI32(mem, c, uint32(rUI16(mem, a)))
		case ir.OpcodeUI16ToUI64:
			wUI64(mem, c, uint64(rUI16(mem, a)))
		case ir.OpcodeUI16ToF32:
			wF32(mem, c, float32(rUI16(mem, a)))
		case ir.OpcodeUI16ToF64:
			wF64(mem, c, float64(rUI16(mem, a)))
		case ir.OpcodeUI32ToI8:
			wI8(mem, c, int8(rUI32(mem, a)))
		case ir.OpcodeUI32ToI16:
			wI16(mem, c, int16(rUI32(mem, a)))
		case ir.OpcodeUI32ToI32:
			wI32(mem, c, int32(rUI32(mem, a)))
		case ir.OpcodeUI32ToI64:
			wI64(mem, c, int64(rUI32(mem, a)))
		case ir.OpcodeUI32ToUI8:
			wUI8(mem, c, uint8(rUI32(mem, a)))
		case ir.OpcodeUI32ToUI16:
			wUI16(mem, c, uint16(rUI32(mem, a)))
		case ir.OpcodeUI32ToUI64:
			wUI64(mem, c, uint64(rUI32(mem, a)))
		case ir.OpcodeUI32ToF32:
			wF32(mem, c, float32(rUI32(mem, a)))
		case ir.OpcodeUI32ToF64:
			wF64(mem, c, float64(rUI32(mem, a)))
		case ir.OpcodeUI64ToI8:
			wI8(mem, c, int8(rUI64(mem, a)))
		case ir.OpcodeUI64ToI16:
			wI16(mem, c, int16(rUI64(mem, a)))
		case ir.OpcodeUI64ToI32:
			wI32(mem, c, int32(rUI64(mem, a)))
		case ir.OpcodeUI64ToI64:
			wI64(mem, c, int64(rUI64(mem, a)))
		case ir.OpcodeUI64ToUI8:
			wUI8(mem, c, uint8(rUI64(mem, a)))
		case ir.OpcodeUI64ToUI16:
			wUI16(mem, c, uint16(rUI64(mem, a)))
		case ir.OpcodeUI64ToUI32:
			wUI32(mem, c, uint32(rUI64(mem, a)))
		case ir.OpcodeUI64ToF32:
			wF32(mem, c, float32(rUI64(mem, a)))
		case ir.OpcodeUI64ToF64:
			wF64(mem, c, float64(rUI64(mem, a)))
		case ir.OpcodeF32ToI8:
			wI8(mem, c, int8(rF32(mem, a)))
		case ir.OpcodeF32ToI16:
			wI16(mem, c, int16(rF32(mem, a)))
		case ir.OpcodeF32ToI32:
			wI32(mem, c, int32(rF32(mem, a)))
		case ir.OpcodeF32ToI64:
			wI64(mem, c, int64(rF32(mem, a)))
		case ir.OpcodeF32ToUI8:
			wUI8(mem, c, uint8(rF32(mem, a)))
		case ir.OpcodeF32ToUI16:
			wUI16(mem, c, uint16(rF32(mem, a)))
		case ir.OpcodeF32ToUI32:
			wUI32(mem, c, uint32(rF32(mem, a)))
		case ir.OpcodeF32ToUI64:
			wUI64(mem, c, uint64(rF32(mem, a)))
		case ir.OpcodeF32ToF64:
			wF64(mem, c, float64(rF32(mem, a)))
		case ir.OpcodeF64ToI8:
			wI8(mem, c, int8(rF64(mem, a)))
		case ir.OpcodeF64ToI16:
			wI16(mem, c, int16(rF64(mem, a)))
		case ir.OpcodeF64ToI32:
			wI32(mem, c, int32(rF64(mem, a)))
		case ir.OpcodeF64ToI64:
			wI64(mem, c, int64(rF64(mem, a)))
		case ir.OpcodeF64ToUI8:
			wUI8(mem, c, uint8(rF64(mem, a)))
		case ir.OpcodeF64ToUI16:
			wUI16(mem, c, uint16(rF64(mem, a)))
		case ir.OpcodeF64ToUI32:
			wUI32(mem, c, uint32(rF64(mem, a)))
		case ir.OpcodeF64ToUI64:
			wUI64(mem, c, uint64(rF64(mem, a)))
		case ir.OpcodeF64ToF32:
			wF32(mem, c, float32(rF64(mem, a)))
		default:
			return referenceStop(machine, types.CX_RUNTIME_NOT_IMPLEMENTED, fn, pc)
		}
	}
}

func rBool(mem []byte, offset uint32) bool { return mem[offset] != 0 }
func wBool(mem []byte, offset uint32, value bool) {
	if value {
		mem[offset] = 1
	} else {
		mem[offset] = 0
	}
}
func rI8(mem []byte, offset uint32) int8      { return int8(mem[offset]) }
func wI8(mem []byte, offset uint32, v int8)   { mem[offset] = byte(v) }
func rUI8(mem []byte, offset uint32) uint8    { return mem[offset] }
func wUI8(mem []byte, offset uint32, v uint8) { mem[offset] = v }
func rI16(mem []byte, offset uint32) int16 {
	return int16(binary.LittleEndian.Uint16(mem[offset:]))
}
func wI16(mem []byte, offset uint32, v int16) {
	binary.LittleEndian.PutUint16(mem[offset:], uint16(v))
}
func rUI16(mem []byte, offset uint32) uint16 { return binary.LittleEndian.Uint16(mem[offset:]) }
func wUI16(mem []byte, offset uint32, v uint16) {
	binary.LittleEndian.PutUint16(mem[offset:], v)
}
func rI32(mem []byte, offset uint32) int32 {
	return int32(binary.LittleEndian.Uint32(mem[offset:]))
}
func wI32(mem []byte, offset uint32, v int32) {
	binary.LittleEndian.PutUint32(mem[offset:], uint32(v))
}
func rUI32(mem []byte, offset uint32) uint32 { return binary.LittleEndian.Uint32(mem[offset:]) }
func wUI32(mem []byte, offset uint32, v uint32) {
	binary.LittleEndian.PutUint32(mem[offset:], v)
}
func rI64(mem []byte, offset uint32) int64 {
	return int64(binary.LittleEndian.Uint64(mem[offset:]))
}
func wI64(mem []byte, offset uint32, v int64) {
	binary.LittleEndian.PutUint64(mem[offset:], uint64(v))
}
func rUI64(mem []byte, offset uint32) uint64 { return binary.LittleEndian.Uint64(mem[offset:]) }
func wUI64(mem []byte, offset uint32, v uint64) {
	binary.LittleEndian.PutUint64(mem[offset:], v)
}
func rF32(mem []byte, offset uint32) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(mem[offset:]))
}
func wF32(mem []byte, offset uint32, v float32) {
	binary.LittleEndian.PutUint32(mem[offset:], math.Float32bits(v))
}
func rF64(mem []byte, offset uint32) float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(mem[offset:]))
}
func wF64(mem []byte, offset uint32, v float64) {
	binary.LittleEndian.PutUint64(mem[offset:], math.Float64bits(v))
}
