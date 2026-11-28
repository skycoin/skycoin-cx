package llvm

import (
	"fmt"
	"sort"

	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// libraryRuntimeFunctions is the C function of runtime.c.txt that is the
// native form of each system library opcode, decision D119. It takes the
// memory, its size, the pointer size and the block at sp, the inputs and
// then the outputs as the vm form reads them.
var libraryRuntimeFunctions = map[ir.Opcode]string{
	ir.OpcodeCipherHashStateNew:                   "cx_rt_cipher_hash_state_new",
	ir.OpcodeCipherHashStateRelease:               "cx_rt_cipher_hash_state_release",
	ir.OpcodeNetUpdate:                            "cx_rt_net_update",
	ir.OpcodeNetReadyCount:                        "cx_rt_net_ready_count",
	ir.OpcodeNetReadyNext:                         "cx_rt_net_ready_next",
	ir.OpcodeNetClock:                             "cx_rt_net_clock",
	ir.OpcodeNetRead:                              "cx_rt_net_read",
	ir.OpcodeNetWrite:                             "cx_rt_net_write",
	ir.OpcodeNetClose:                             "cx_rt_net_close",
	ir.OpcodeNetTimeoutSet:                        "cx_rt_net_timeout_set",
	ir.OpcodeNetTCPListen:                         "cx_rt_net_tcp_listen",
	ir.OpcodeNetTCPDial:                           "cx_rt_net_tcp_dial",
	ir.OpcodeNetUDPListen:                         "cx_rt_net_udp_listen",
	ir.OpcodeNetUDPDial:                           "cx_rt_net_udp_dial",
	ir.OpcodeNetUDPReadFrom:                       "cx_rt_net_udp_read_from",
	ir.OpcodeNetUDPWriteTo:                        "cx_rt_net_udp_write_to",
	ir.OpcodeNetUDPPeerResolve:                    "cx_rt_net_udp_peer_resolve",
	ir.OpcodeFileClose:                            "cx_rt_file_close",
	ir.OpcodeFileSeek:                             "cx_rt_file_seek",
	ir.OpcodeFileSync:                             "cx_rt_file_sync",
	ir.OpcodeTerminalArgCount:                     "cx_rt_terminal_arg_count",
	ir.OpcodeTerminalArgRead:                      "cx_rt_terminal_arg_read",
	ir.OpcodeTerminalInputLineRead:                "cx_rt_terminal_input_line_read",
	ir.OpcodeReflectCxTypeOfValueGet:              "cx_rt_reflect_cx_type_of_value_get",
	ir.OpcodeReflectCxTypeKindGet:                 "cx_rt_reflect_cx_type_kind_get",
	ir.OpcodeReflectCxTypeNameGet:                 "cx_rt_reflect_cx_type_name_get",
	ir.OpcodeReflectCxTypeSizeInBytesGet:          "cx_rt_reflect_cx_type_size_in_bytes_get",
	ir.OpcodeReflectCxTypeAlignmentInBytesGet:     "cx_rt_reflect_cx_type_alignment_in_bytes_get",
	ir.OpcodeReflectCxStructFieldCountGet:         "cx_rt_reflect_cx_struct_field_count_get",
	ir.OpcodeReflectCxStructFieldNameGet:          "cx_rt_reflect_cx_struct_field_name_get",
	ir.OpcodeReflectCxStructFieldTypeGet:          "cx_rt_reflect_cx_struct_field_type_get",
	ir.OpcodeReflectCxStructFieldOffsetInBytesGet: "cx_rt_reflect_cx_struct_field_offset_in_bytes_get",
	ir.OpcodeReflectCxStructFieldOrdinalByNameGet: "cx_rt_reflect_cx_struct_field_ordinal_by_name_get",
	ir.OpcodeReflectCxStructFieldValueRead:        "cx_rt_reflect_cx_struct_field_value_read",
	ir.OpcodeReflectCxStructFieldValueWrite:       "cx_rt_reflect_cx_struct_field_value_write",
	ir.OpcodeReflectCxStructMethodCountGet:        "cx_rt_reflect_cx_struct_method_count_get",
	ir.OpcodeReflectCxStructMethodNameGet:         "cx_rt_reflect_cx_struct_method_name_get",
	ir.OpcodeReflectCxStructMethodFunctionGet:     "cx_rt_reflect_cx_struct_method_function_get",
	ir.OpcodeReflectCxArrayLengthGet:              "cx_rt_reflect_cx_array_length_get",
	ir.OpcodeReflectCxArrayElementTypeGet:         "cx_rt_reflect_cx_array_element_type_get",
	ir.OpcodeReflectCxSliceElementTypeGet:         "cx_rt_reflect_cx_slice_element_type_get",
	ir.OpcodeReflectCxPointerElementTypeGet:       "cx_rt_reflect_cx_pointer_element_type_get",
	ir.OpcodeReflectCxFunctionNameGet:             "cx_rt_reflect_cx_function_name_get",
	ir.OpcodeReflectCxFunctionInputCountGet:       "cx_rt_reflect_cx_function_input_count_get",
	ir.OpcodeReflectCxFunctionInputTypeGet:        "cx_rt_reflect_cx_function_input_type_get",
	ir.OpcodeReflectCxFunctionOutputCountGet:      "cx_rt_reflect_cx_function_output_count_get",
	ir.OpcodeReflectCxFunctionOutputTypeGet:       "cx_rt_reflect_cx_function_output_type_get",
	ir.OpcodeReflectCxStringPoolTextRead:          "cx_rt_reflect_cx_string_pool_text_read",
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
		p := e.localNew()
		e.line("%s = getelementptr inbounds i8, ptr %s, i64 %d", p, block, offset)
		e.line("store %s %s, ptr %s, align 1", valueType(e.call.pendingCallSizes[offset]), e.call.pendingCallArgs[offset], p)
	}
	memorySize := uint64(e.dataEnd) + uint64(e.image.ImageScratchSize)
	if ir.OpcodeReflect(in.InstrOpcode) {
		// A reflect function also takes the address of the type table and the
		// type number of CxTypeOfValueGet (decision D150).
		e.declare(function, fmt.Sprintf("declare void @%s(ptr, i64, i32, ptr, i32, i32) nounwind", function))
		e.line("call void @%s(ptr @cx_mem, i64 %d, i32 %d, ptr %s, i32 %d, i32 %d)", function, memorySize, pointerSize, block, in.InstrAux, in.InstrN)
	} else {
		e.declare(function, fmt.Sprintf("declare void @%s(ptr, i64, i32, ptr) nounwind", function))
		e.line("call void @%s(ptr @cx_mem, i64 %d, i32 %d, ptr %s)", function, memorySize, pointerSize, block)
	}
	e.result = ""
	e.resultBlock = true
	e.call = pendingCall{}
	return nil
}

// instrLibraryResult copies an output of the last library opcode from the
// block at sp to C.
func (e *emitter) instrLibraryResult(in *ir.Instr) error {
	block := e.libraryBlock()
	p := e.localNew()
	e.line("%s = getelementptr inbounds i8, ptr %s, i64 %d", p, block, in.InstrA)
	v := e.localNew()
	e.line("%s = load %s, ptr %s, align 1", v, valueType(in.InstrN), p)
	return e.store(in, 2, valueType(in.InstrN), in.InstrN, v)
}

// libraryBlock returns a pointer to the block at sp in cx_mem.
func (e *emitter) libraryBlock() string {
	sp := e.localNew()
	e.line("%s = zext i32 %%sp to i64", sp)
	block := e.localNew()
	e.line("%s = getelementptr inbounds i8, ptr @cx_mem, i64 %s", block, sp)
	return block
}
