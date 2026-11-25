package llvm

import (
	"fmt"
	"sort"

	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// libraryRuntimeFunctions is the C function of runtime.c.txt that is the
// native form of each system library opcode, decision D119. It takes the
// block at sp, the inputs and then the outputs as the vm form reads them.
var libraryRuntimeFunctions = map[ir.Opcode]string{
	ir.OpcodeCipherHashStateNew:     "cx_rt_cipher_hash_state_new",
	ir.OpcodeCipherHashStateRelease: "cx_rt_cipher_hash_state_release",
}

// instrLibrary emits a system library opcode: the arguments of the pending
// call go to the block at sp in cx_mem, zeroed first as the vm frame is,
// and the runtime function runs on the block. instrLibraryResult reads the
// outputs back.
func (e *emitter) instrLibrary(in *ir.Instr) error {
	function, found := libraryRuntimeFunctions[in.InstrOpcode]
	if !found {
		return fmt.Errorf("library opcode %d has no runtime function", in.InstrOpcode)
	}
	if !e.call.pendingCallActive {
		return fmt.Errorf("library opcode outside a call")
	}
	block := e.libraryBlock()
	e.line("call void @llvm.memset.p0.i64(ptr %s, i8 0, i64 %d, i1 false)", block, e.call.pendingCallFrame)
	offsets := make([]uint32, 0, len(e.call.pendingCallArgs))
	for offset := range e.call.pendingCallArgs {
		offsets = append(offsets, offset)
	}
	sort.Slice(offsets, func(i, j int) bool { return offsets[i] < offsets[j] })
	for _, offset := range offsets {
		p := e.newTmp()
		e.line("%s = getelementptr inbounds i8, ptr %s, i64 %d", p, block, offset)
		e.line("store %s %s, ptr %s, align 1", valueType(e.call.pendingCallSizes[offset]), e.call.pendingCallArgs[offset], p)
	}
	e.declare(function, fmt.Sprintf("declare void @%s(ptr) nounwind", function))
	e.line("call void @%s(ptr %s)", function, block)
	e.result = ""
	e.resultBlock = true
	e.call = pendingCall{}
	return nil
}

// instrLibraryResult copies an output of the last library opcode from the
// block at sp to C.
func (e *emitter) instrLibraryResult(in *ir.Instr) error {
	block := e.libraryBlock()
	p := e.newTmp()
	e.line("%s = getelementptr inbounds i8, ptr %s, i64 %d", p, block, in.InstrA)
	v := e.newTmp()
	e.line("%s = load %s, ptr %s, align 1", v, valueType(in.InstrN), p)
	return e.store(in, 2, valueType(in.InstrN), in.InstrN, v)
}

// libraryBlock returns a pointer to the block at sp in cx_mem.
func (e *emitter) libraryBlock() string {
	sp := e.newTmp()
	e.line("%s = zext i32 %%sp to i64", sp)
	block := e.newTmp()
	e.line("%s = getelementptr inbounds i8, ptr @cx_mem, i64 %s", block, sp)
	return block
}
