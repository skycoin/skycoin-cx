// Package llvm turns an ir.Image into textual LLVM IR, the LLVM target of
// cx build, decisions D8, D9 and D10. It is pure Go and links no LLVM.
//
// A scalar opcode becomes its template from the generated opcode table, or a
// call of its runtime function when it has none. Prints, test and stops call
// the runtime too. A frame variable whose address is never taken becomes an
// alloca that LLVM keeps in registers, with its extent from ImageLocals. A
// variable whose address is taken stays in a shadow stack inside cx_mem at
// its vm offset, so every pointer keeps its vm value. A load of a literal of
// ImageReadOnly becomes a constant. Stack, depth, index, address and divide
// checks stop the run with the exit code and message of the vm.
package llvm

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// RuntimeC is the C runtime every emitted module links with. A native build
// compiles it with the module.
//
//go:embed runtime.c.txt
var RuntimeC []byte

// LibraryCallRefusal is the error of an image that calls a system library,
// whose functions are Go and have no native form.
const LibraryCallRefusal = "a system library call has no native form"

// RuntimeFileName is the name the runtime source is written under.
const RuntimeFileName = "cx_runtime.c"

const pointerSize = uint32(types.POINTER_SIZE)

// Stop is a run time check of the emitted code that ends the run.
type Stop uint8

const (
	StopDivideByZero Stop = iota
	StopStackOverflow
	StopIndexNegative
	StopAddressInvalid
	StopCount
)

// Options holds what the emitted code prints and returns, which the caller
// takes from the vm and its printer so a native run prints what vm prints.
type Options struct {
	// StopCodes is the exit code of each stop, as vm.StopExitCodes.
	StopCodes [StopCount]int32
	// StopLines is the first line vm prints for each stop, without the
	// error header, as printer.PrinterVmErrorPrint writes it.
	StopLines [StopCount]string
	// AssertCode is the exit code after a failed test.
	AssertCode int32
	// FallbackOnly calls the runtime function of every scalar opcode, also
	// of one with a template, to check the two against each other.
	FallbackOnly bool
	// Debug adds DWARF debug information: a subprogram per function, the
	// source line of each instruction, and the name and type of each
	// variable kept in an alloca.
	Debug bool
	// RefusalLine, when set, is the line main prints before it ends with
	// RefusalCode, without running the program: the loader of cx refuses a
	// program that uses a system library its run does not enable (D119).
	RefusalLine string
	RefusalCode int32
}

// Stats counts the instructions of a module by how they were emitted.
// StatsInline and StatsFallback count scalar opcodes, by opcode, StatsRuntime
// the other instructions that call the runtime. StatsAlloca and StatsShadow
// count the frame regions kept in registers and in cx_mem.
type Stats struct {
	StatsInline   map[ir.Opcode]int
	StatsFallback map[ir.Opcode]int
	StatsRuntime  int
	StatsAlloca   int
	StatsShadow   int
}

// region is a byte range of a frame or of the scratch that becomes one
// alloca, or stays in cx_mem when regionShadow is set.
type region struct {
	regionStart  uint32
	regionEnd    uint32
	regionShadow bool
	regionName   string
}

// access is one operand of an instruction, accessWhich 0 to 3 for A to D.
// An address access takes the address of a variable, which must stay in
// memory.
type access struct {
	accessWhich   int
	accessSize    uint32
	accessAddress bool
}

type pendingCall struct {
	pendingCallActive bool
	pendingCallFrame  uint32
	pendingCallArgs   map[uint32]string
	pendingCallSizes  map[uint32]uint32
}

type emitter struct {
	image    *ir.Image
	options  Options
	out      bytes.Buffer
	body     bytes.Buffer
	cold     bytes.Buffer
	decls    map[string]string
	strs     map[string]string
	stats    Stats
	dataEnd  uint32
	function uint32
	fn       *ir.Func
	tmp      int
	frame    []region
	scratch  []region
	call     pendingCall
	result   string
	resultOf uint32
	// resultBlock says that the last call was a library opcode, whose outputs
	// lie in the block at sp.
	resultBlock bool
	readOnly    []bool
	callResults []string
	debug       *debugInfo
}

// Emit returns the LLVM IR module of image, or an error for an instruction
// the target does not compile. Every function of image that has code is
// compiled, and main runs *init and then main as the vm does.
func Emit(image *ir.Image, options Options) ([]byte, Stats, error) {
	e := &emitter{
		image:   image,
		options: options,
		decls:   map[string]string{},
		strs:    map[string]string{},
		dataEnd: ir.ImageDataEnd(image),
		stats:   Stats{StatsInline: map[ir.Opcode]int{}, StatsFallback: map[ir.Opcode]int{}},
	}
	e.readOnlyMark()
	if options.Debug {
		e.debug = debugInfoNew()
	}
	for id := range image.ImageFuncs {
		fn := &image.ImageFuncs[id]
		if len(fn.FuncCode) == 0 {
			continue
		}
		if err := e.emitFunction(uint32(id), fn); err != nil {
			return nil, e.stats, fmt.Errorf("%s: %w", image.ImageFuncNames[id], err)
		}
	}
	if options.RefusalLine != "" {
		e.str(options.RefusalLine)
	}
	e.header()
	e.out.Write(e.body.Bytes())
	e.entry()
	names := make([]string, 0, len(e.decls))
	for name := range e.decls {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		e.out.WriteString(e.decls[name] + "\n")
	}
	e.out.WriteString("\n!0 = !{!\"branch_weights\", i32 1, i32 1048576}\n")
	e.debugModule()
	return e.out.Bytes(), e.stats, nil
}

func (e *emitter) header() {
	image := e.image
	e.out.WriteString("; CX program image as LLVM IR, written by cx build --target=llvm\n")
	size := e.dataEnd + image.ImageScratchSize
	fmt.Fprintf(&e.out, "@cx_mem = internal global [%d x i8] zeroinitializer, align 16\n", size)
	for i, run := range e.dataRuns() {
		data := image.ImageData[run[0]:run[1]]
		fmt.Fprintf(&e.out, "@cx_init.%d = private unnamed_addr constant [%d x i8] %s\n", i, len(data), llvmBytes(data))
	}
	fmt.Fprintf(&e.out, "@cx_text = internal constant [%d x i8] %s\n", len(image.ImageText), llvmBytes(image.ImageText))
	keys := make([]string, 0, len(e.strs))
	for text := range e.strs {
		keys = append(keys, text)
	}
	sort.Strings(keys)
	for _, text := range keys {
		data := append([]byte(text), 0)
		fmt.Fprintf(&e.out, "%s = private unnamed_addr constant [%d x i8] %s\n", e.strs[text], len(data), llvmBytes(data))
	}
	e.out.WriteString("\n")
}

func (e *emitter) entry() {
	image := e.image
	e.body.Reset()
	attachment := e.debugEntry()
	fmt.Fprintf(&e.body, "define i32 @main()%s {\nentry:\n", attachment)
	e.declare("cx_rt_start", "declare void @cx_rt_start(i32)")
	e.line("call void @cx_rt_start(i32 %d)", e.options.StopCodes[StopDivideByZero])
	if e.options.RefusalLine != "" {
		e.declare("cx_rt_stop", "declare void @cx_rt_stop(i32, ptr) noreturn cold nounwind")
		e.line("call void @cx_rt_stop(i32 %d, ptr %s)", e.options.RefusalCode, e.str(e.options.RefusalLine))
	}
	for i, run := range e.dataRuns() {
		e.line("call void @llvm.memcpy.p0.p0.i64(ptr getelementptr (i8, ptr @cx_mem, i64 %d), ptr @cx_init.%d, i64 %d, i1 false)", image.ImageDataStart+uint32(run[0]), i, run[1]-run[0])
	}
	e.declare("llvm.memcpy", "declare void @llvm.memcpy.p0.p0.i64(ptr, ptr, i64, i1)")
	fp := uint32(0)
	if image.ImageEntryInit != ir.ImageFuncNone {
		e.line("call %s %s(i32 0, i32 0)", e.retType(&image.ImageFuncs[image.ImageEntryInit]), e.funcName(image.ImageEntryInit))
		fp = image.ImageFuncs[image.ImageEntryInit].FuncFrameSize
	}
	if image.ImageEntryMain != ir.ImageFuncNone {
		e.line("call %s %s(i32 %d, i32 0)", e.retType(&image.ImageFuncs[image.ImageEntryMain]), e.funcName(image.ImageEntryMain), fp)
	}
	e.declare("cx_rt_finish", "declare i32 @cx_rt_finish(i32)")
	e.line("%%code = call i32 @cx_rt_finish(i32 %d)", e.options.AssertCode)
	e.line("ret i32 %%code")
	e.body.WriteString("}\n\n")
	e.out.Write(e.body.Bytes())
}

func llvmBytes(data []byte) string {
	var b strings.Builder
	b.WriteString("c\"")
	for _, c := range data {
		if c >= 0x20 && c < 0x7f && c != '"' && c != '\\' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "\\%02X", c)
		}
	}
	b.WriteString("\"")
	return b.String()
}

func (e *emitter) declare(name, text string) { e.decls[name] = text }

func (e *emitter) str(text string) string {
	if name, found := e.strs[text]; found {
		return name
	}
	name := "@cx_str." + strconv.Itoa(len(e.strs))
	e.strs[text] = name
	return name
}

// funcName is the LLVM name of function id, its id and CX name.
func (e *emitter) funcName(id uint32) string {
	return fmt.Sprintf("@\"cx.%d.%s\"", id, e.image.ImageFuncNames[id])
}

// valueType is the LLVM type that carries a slot of size bytes.
func valueType(size uint32) string {
	switch size {
	case 1, 2, 4, 8, 16:
		return "i" + strconv.Itoa(int(size*8))
	}
	return fmt.Sprintf("[%d x i8]", size)
}

func zeroValue(size uint32) string {
	switch size {
	case 1, 2, 4, 8, 16:
		return "0"
	}
	return "zeroinitializer"
}

// typeLlvm is the LLVM type of a scalar type code, i8 for a bool.
func typeLlvm(code types.Code) string {
	switch code {
	case types.F32:
		return "float"
	case types.F64:
		return "double"
	}
	return valueType(typeSize(code))
}

func typeSize(code types.Code) uint32 {
	switch code {
	case types.BOOL, types.I8, types.UI8:
		return 1
	case types.I16, types.UI16:
		return 2
	case types.I32, types.UI32, types.F32:
		return 4
	case types.I64, types.UI64, types.F64:
		return 8
	}
	return 0
}

func typeSigned(code types.Code) bool {
	return code == types.I8 || code == types.I16 || code == types.I32 || code == types.I64
}

func typeFloat(code types.Code) bool { return code == types.F32 || code == types.F64 }

func (e *emitter) retType(fn *ir.Func) string {
	if len(fn.FuncOutputs) == 0 {
		return "void"
	}
	parts := make([]string, len(fn.FuncOutputs))
	for i, slot := range fn.FuncOutputs {
		parts[i] = valueType(slot.SlotSize)
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func (e *emitter) newTmp() string {
	e.tmp++
	return "%t" + strconv.Itoa(e.tmp)
}

func (e *emitter) newLabel(prefix string) string {
	e.tmp++
	return prefix + strconv.Itoa(e.tmp)
}

func (e *emitter) line(format string, args ...any) {
	e.body.WriteString("  ")
	fmt.Fprintf(&e.body, format, args...)
	if e.debug != nil && e.debug.debugCurrent != 0 {
		fmt.Fprintf(&e.body, ", !dbg !%d", e.debug.debugCurrent)
	}
	e.body.WriteByte('\n')
}

// accesses lists the operands of in with their sizes.
func accesses(in *ir.Instr) []access {
	n, m := in.InstrN, in.InstrM
	switch in.InstrOpcode {
	case ir.OpcodeJumpIf:
		return []access{{0, 1, false}}
	case ir.OpcodeZero:
		return []access{{0, n, false}}
	case ir.OpcodeMove:
		return []access{{0, n, false}, {2, n, false}}
	case ir.OpcodeCallArg:
		return []access{{0, n, false}}
	case ir.OpcodeCallResult:
		return []access{{2, n, false}}
	case ir.OpcodeTestEqual:
		return []access{{0, n, false}, {1, m, false}, {2, pointerSize, false}}
	case ir.OpcodeAddressIndexStatic:
		return []access{{0, 0, true}, {1, 4, false}, {2, 4, false}}
	case ir.OpcodeAddressIndex:
		return []access{{0, 4, false}, {1, 4, false}, {2, 4, false}}
	case ir.OpcodeAddressDerefStatic:
		return []access{{0, pointerSize, false}, {2, 4, false}}
	case ir.OpcodeAddressDeref:
		return []access{{0, 4, false}, {2, 4, false}}
	case ir.OpcodeAddressLoad:
		return []access{{0, 4, false}, {2, n, false}}
	case ir.OpcodeAddressStore:
		return []access{{0, n, false}, {2, 4, false}}
	case ir.OpcodeAddressOf:
		return []access{{0, 0, true}, {2, pointerSize, false}}
	case ir.OpcodeAddressPointer:
		return []access{{0, 4, false}, {2, pointerSize, false}}
	case ir.OpcodePrintValue:
		return []access{{0, typeSize(types.Code(m)), false}}
	case ir.OpcodePrintStr:
		return []access{{0, pointerSize, false}}
	}
	fact := &opcodeFacts[in.InstrOpcode]
	if fact.OpcodeFactName == "" {
		return nil
	}
	list := []access{{0, typeSize(fact.OpcodeFactType), false}}
	if fact.OpcodeFactInputs == 2 {
		list = append(list, access{1, typeSize(fact.OpcodeFactType), false})
	}
	if fact.OpcodeFactResult != types.UNDEFINED {
		list = append(list, access{2, typeSize(fact.OpcodeFactResult), false})
	}
	return list
}

func operand(in *ir.Instr, which int) (uint32, bool) {
	switch which {
	case 0:
		return in.InstrA, in.InstrMode&ir.InstrModeFrameA != 0
	case 1:
		return in.InstrB, in.InstrMode&ir.InstrModeFrameB != 0
	case 2:
		return in.InstrC, in.InstrMode&ir.InstrModeFrameC != 0
	}
	return in.InstrD, in.InstrMode&ir.InstrModeFrameD != 0
}

// partition merges overlapping byte ranges into regions.
func partition(list []region) []region {
	sort.Slice(list, func(i, j int) bool { return list[i].regionStart < list[j].regionStart })
	var out []region
	for _, r := range list {
		if r.regionEnd <= r.regionStart {
			continue
		}
		if n := len(out); n > 0 && r.regionStart < out[n-1].regionEnd {
			out[n-1].regionEnd = max(out[n-1].regionEnd, r.regionEnd)
			continue
		}
		out = append(out, region{regionStart: r.regionStart, regionEnd: r.regionEnd})
	}
	return out
}

func find(regions []region, offset uint32) int {
	i := sort.Search(len(regions), func(i int) bool { return regions[i].regionEnd > offset })
	if i < len(regions) && regions[i].regionStart <= offset {
		return i
	}
	return -1
}

// analyze splits the frame of fn into regions, one per variable of
// ImageLocals joined with the operands outside them, and the scratch into
// the ranges its instructions use. A region whose address is taken stays in
// cx_mem.
func (e *emitter) analyze(fn *ir.Func) error {
	var frame, scratch []region
	var addressed []uint32
	locals := e.image.ImageLocals[fn.FuncLocalFirst : fn.FuncLocalFirst+fn.FuncLocalCount]
	for _, local := range locals {
		frame = append(frame, region{regionStart: local.LocalOffset, regionEnd: local.LocalOffset + local.LocalSize})
	}
	for _, slot := range fn.FuncInputs {
		frame = append(frame, region{regionStart: slot.SlotOffset, regionEnd: slot.SlotOffset + slot.SlotSize})
	}
	for _, slot := range fn.FuncOutputs {
		frame = append(frame, region{regionStart: slot.SlotOffset, regionEnd: slot.SlotOffset + slot.SlotSize})
	}
	for pc := range fn.FuncCode {
		in := &fn.FuncCode[pc]
		for _, ac := range accesses(in) {
			offset, inFrame := operand(in, ac.accessWhich)
			switch {
			case inFrame && ac.accessAddress:
				addressed = append(addressed, offset)
			case inFrame:
				frame = append(frame, region{regionStart: offset, regionEnd: offset + ac.accessSize})
			case ac.accessAddress:
				if offset >= e.dataEnd {
					return fmt.Errorf("pc %d takes the address of scratch", pc)
				}
			case offset >= e.dataEnd:
				scratch = append(scratch, region{regionStart: offset, regionEnd: offset + ac.accessSize})
			}
		}
	}
	e.frame = partition(frame)
	e.scratch = partition(scratch)
	for _, offset := range addressed {
		i := find(e.frame, offset)
		if i < 0 {
			// The address of a byte no variable covers, so the whole frame
			// stays in memory.
			e.frame = []region{{regionStart: 0, regionEnd: fn.FuncFrameSize, regionShadow: true}}
			break
		}
		e.frame[i].regionShadow = true
	}
	for i := range e.frame {
		r := &e.frame[i]
		if r.regionShadow {
			e.stats.StatsShadow++
			continue
		}
		e.stats.StatsAlloca++
		r.regionName = fmt.Sprintf("%%f%d", r.regionStart)
		for _, local := range locals {
			if local.LocalOffset == r.regionStart && e.image.ImageLocalNames[local.LocalName] != "" {
				r.regionName = fmt.Sprintf("%%\"%s.f%d\"", e.image.ImageLocalNames[local.LocalName], r.regionStart)
				break
			}
		}
	}
	for i := range e.scratch {
		e.scratch[i].regionName = fmt.Sprintf("%%s%d", e.scratch[i].regionStart-e.dataEnd)
	}
	return nil
}

// ptr returns a pointer to operand which of in, size bytes.
func (e *emitter) ptr(in *ir.Instr, which int, size uint32) (string, error) {
	offset, inFrame := operand(in, which)
	return e.ptrAt(offset, inFrame, size)
}

func (e *emitter) ptrAt(offset uint32, inFrame bool, size uint32) (string, error) {
	regions := e.scratch
	if inFrame {
		regions = e.frame
	} else if offset < e.dataEnd {
		t := e.newTmp()
		e.line("%s = getelementptr inbounds i8, ptr @cx_mem, i64 %d", t, offset)
		return t, nil
	}
	i := find(regions, offset)
	if i < 0 || offset+size > regions[i].regionEnd {
		return "", fmt.Errorf("operand %d size %d outside its region", offset, size)
	}
	r := &regions[i]
	if r.regionShadow {
		t := e.newTmp()
		e.line("%s = getelementptr inbounds i8, ptr %%fpp, i64 %d", t, offset)
		return t, nil
	}
	if offset == r.regionStart {
		return r.regionName, nil
	}
	t := e.newTmp()
	e.line("%s = getelementptr inbounds i8, ptr %s, i64 %d", t, r.regionName, offset-r.regionStart)
	return t, nil
}

func (e *emitter) load(in *ir.Instr, which int, typ string, size uint32) (string, error) {
	offset, inFrame := operand(in, which)
	if v, found := e.constant(offset, inFrame, typ, size); found {
		return v, nil
	}
	p, err := e.ptr(in, which, size)
	if err != nil {
		return "", err
	}
	t := e.newTmp()
	e.line("%s = load %s, ptr %s, align 1", t, typ, p)
	return t, nil
}

func (e *emitter) store(in *ir.Instr, which int, typ string, size uint32, value string) error {
	p, err := e.ptr(in, which, size)
	if err != nil {
		return err
	}
	e.line("store %s %s, ptr %s, align 1", typ, value, p)
	return nil
}

// vmAddress returns the vm address of operand which, the frame pointer
// added for a frame operand.
func (e *emitter) vmAddress(in *ir.Instr, which int) string {
	offset, inFrame := operand(in, which)
	if !inFrame {
		return strconv.FormatUint(uint64(offset), 10)
	}
	t := e.newTmp()
	e.line("%s = add i32 %%fp, %d", t, offset)
	return t
}

// where is the file and line of pc in the current function, as
// ir.LineAt gives them.
func (e *emitter) where(pc uint32) (string, uint32) {
	if line, found := ir.LineAt(e.image, e.function, pc); found {
		return e.image.ImageFileNames[line.LineFile], line.LineNumber
	}
	return "", 0
}

// stopText is the first line vm prints for stop at pc.
func (e *emitter) stopText(pc uint32, stop Stop) string {
	file, number := e.where(pc)
	return fmt.Sprintf("error: %s:%d, %s", file, number, e.options.StopLines[stop])
}

// check branches to a cold block that stops the run when cond is true.
func (e *emitter) check(cond string, stop Stop, pc uint32) {
	bad, ok := e.newLabel("stop"), e.newLabel("ok")
	e.line("br i1 %s, label %%%s, label %%%s, !prof !0", cond, bad, ok)
	e.body.WriteString(ok + ":\n")
	text := e.str(e.stopText(pc, stop))
	e.declare("cx_rt_stop", "declare void @cx_rt_stop(i32, ptr) noreturn cold nounwind")
	fmt.Fprintf(&e.cold, "%s:\n  call void @cx_rt_stop(i32 %d, ptr %s)\n  unreachable\n", bad, e.options.StopCodes[stop], text)
}

func (e *emitter) emitFunction(id uint32, fn *ir.Func) error {
	e.function, e.fn, e.tmp = id, fn, 0
	e.cold.Reset()
	e.call = pendingCall{}
	if err := e.analyze(fn); err != nil {
		return err
	}
	params := []string{"i32 %fp", "i32 %depth"}
	for i, slot := range fn.FuncInputs {
		params = append(params, fmt.Sprintf("%s %%in%d", valueType(slot.SlotSize), i))
	}
	attachment := e.debugFunction(id, fn)
	fmt.Fprintf(&e.body, "define internal %s %s(%s) nounwind%s {\nentry:\n", e.retType(fn), e.funcName(id), strings.Join(params, ", "), attachment)
	e.line("%%fp64 = zext i32 %%fp to i64")
	e.line("%%fpp = getelementptr inbounds i8, ptr @cx_mem, i64 %%fp64")
	e.line("%%sp = add i32 %%fp, %d", fn.FuncFrameSize)
	e.declare("llvm.memset", "declare void @llvm.memset.p0.i64(ptr, i8, i64, i1)")
	e.declare("llvm.memcpy", "declare void @llvm.memcpy.p0.p0.i64(ptr, ptr, i64, i1)")
	for _, r := range e.frame {
		if r.regionShadow {
			t := e.newTmp()
			e.line("%s = getelementptr inbounds i8, ptr %%fpp, i64 %d", t, r.regionStart)
			e.line("call void @llvm.memset.p0.i64(ptr %s, i8 0, i64 %d, i1 false)", t, r.regionEnd-r.regionStart)
			continue
		}
		e.line("%s = alloca [%d x i8], align 8", r.regionName, r.regionEnd-r.regionStart)
		e.line("call void @llvm.memset.p0.i64(ptr %s, i8 0, i64 %d, i1 false)", r.regionName, r.regionEnd-r.regionStart)
	}
	for _, r := range e.scratch {
		e.line("%s = alloca [%d x i8], align 8", r.regionName, r.regionEnd-r.regionStart)
		e.line("call void @llvm.memset.p0.i64(ptr %s, i8 0, i64 %d, i1 false)", r.regionName, r.regionEnd-r.regionStart)
	}
	e.debugLocals(fn)
	for i, slot := range fn.FuncInputs {
		p, err := e.ptrAt(slot.SlotOffset, true, slot.SlotSize)
		if err != nil {
			return err
		}
		e.line("store %s %%in%d, ptr %s, align 1", valueType(slot.SlotSize), i, p)
	}
	e.line("br label %%pc0")

	leaders := map[uint32]bool{0: true}
	for pc := range fn.FuncCode {
		in := &fn.FuncCode[pc]
		switch in.InstrOpcode {
		case ir.OpcodeJump:
			leaders[in.InstrM] = true
			leaders[uint32(pc)+1] = true
		case ir.OpcodeJumpIf:
			leaders[in.InstrM] = true
			leaders[in.InstrN] = true
			leaders[uint32(pc)+1] = true
		case ir.OpcodeReturn:
			leaders[uint32(pc)+1] = true
		}
	}
	terminated := true
	for pc := range fn.FuncCode {
		upc := uint32(pc)
		if leaders[upc] {
			if !terminated {
				e.line("br label %%pc%d", pc)
			}
			fmt.Fprintf(&e.body, "pc%d:\n", pc)
		}
		e.debugPc(upc)
		in := &fn.FuncCode[pc]
		terminated = in.InstrOpcode == ir.OpcodeReturn || in.InstrOpcode == ir.OpcodeJump || in.InstrOpcode == ir.OpcodeJumpIf
		if err := e.instr(upc, in); err != nil {
			return fmt.Errorf("pc %d %s: %w", pc, OpcodeName(in.InstrOpcode), err)
		}
	}
	e.body.Write(e.cold.Bytes())
	e.body.WriteString("}\n\n")
	if e.debug != nil {
		e.debug.debugCurrent = 0
	}
	return nil
}

func (e *emitter) copyBytes(dst, src string, size uint32) {
	switch size {
	case 1, 2, 4, 8:
		t := e.newTmp()
		typ := valueType(size)
		e.line("%s = load %s, ptr %s, align 1", t, typ, src)
		e.line("store %s %s, ptr %s, align 1", typ, t, dst)
	default:
		e.line("call void @llvm.memcpy.p0.p0.i64(ptr %s, ptr %s, i64 %d, i1 false)", dst, src, size)
	}
}

// memAt returns a pointer into cx_mem at the vm address addr, an i32 value,
// plus offset, after checking that size bytes fit below the data end.
func (e *emitter) memAt(addr string, offset uint32, size uint32, pc uint32) string {
	a64 := e.newTmp()
	e.line("%s = zext i32 %s to i64", a64, addr)
	sum := e.newTmp()
	e.line("%s = add i64 %s, %d", sum, a64, offset)
	bad := e.newTmp()
	e.line("%s = icmp ugt i64 %s, %d", bad, sum, int64(e.dataEnd)-int64(size))
	e.check(bad, StopAddressInvalid, pc)
	p := e.newTmp()
	e.line("%s = getelementptr inbounds i8, ptr @cx_mem, i64 %s", p, sum)
	return p
}

// pointerTo32 narrows a pointer value to the i32 of an address slot.
func (e *emitter) pointerTo32(v string) string {
	if pointerSize == 4 {
		return v
	}
	t := e.newTmp()
	e.line("%s = trunc %s %s to i32", t, valueType(pointerSize), v)
	return t
}

func (e *emitter) instr(pc uint32, in *ir.Instr) error {
	if ir.OpcodeLibrary(in.InstrOpcode) {
		e.stats.StatsRuntime++
		return e.instrLibrary(in)
	}
	switch in.InstrOpcode {
	case ir.OpcodeReturn:
		return e.instrReturn()
	case ir.OpcodeJump:
		e.line("br label %%pc%d", in.InstrM)
		return nil
	case ir.OpcodeJumpIf:
		v, err := e.load(in, 0, "i8", 1)
		if err != nil {
			return err
		}
		c := e.newTmp()
		e.line("%s = icmp ne i8 %s, 0", c, v)
		e.line("br i1 %s, label %%pc%d, label %%pc%d", c, in.InstrM, in.InstrN)
		return nil
	case ir.OpcodeZero:
		p, err := e.ptr(in, 0, in.InstrN)
		if err != nil {
			return err
		}
		e.line("call void @llvm.memset.p0.i64(ptr %s, i8 0, i64 %d, i1 false)", p, in.InstrN)
		return nil
	case ir.OpcodeMove:
		if offset, inFrame := operand(in, 0); in.InstrN <= 8 {
			if v, found := e.constant(offset, inFrame, valueType(in.InstrN), in.InstrN); found {
				return e.store(in, 2, valueType(in.InstrN), in.InstrN, v)
			}
		}
		src, err := e.ptr(in, 0, in.InstrN)
		if err != nil {
			return err
		}
		dst, err := e.ptr(in, 2, in.InstrN)
		if err != nil {
			return err
		}
		e.copyBytes(dst, src, in.InstrN)
		return nil
	case ir.OpcodeCallFrame:
		return e.instrCallFrame(pc, in)
	case ir.OpcodeCallArg:
		if !e.call.pendingCallActive {
			return fmt.Errorf("argument outside a call")
		}
		v, err := e.load(in, 0, valueType(in.InstrN), in.InstrN)
		if err != nil {
			return err
		}
		e.call.pendingCallArgs[in.InstrC] = v
		e.call.pendingCallSizes[in.InstrC] = in.InstrN
		return nil
	case ir.OpcodeCall:
		return e.instrCall(in)
	case ir.OpcodeCallResult:
		return e.instrCallResult(in)
	case ir.OpcodeLibraryCall:
		return errors.New(LibraryCallRefusal)
	case ir.OpcodeAddressIndexStatic, ir.OpcodeAddressIndex,
		ir.OpcodeAddressDerefStatic, ir.OpcodeAddressDeref,
		ir.OpcodeAddressLoad, ir.OpcodeAddressStore,
		ir.OpcodeAddressOf, ir.OpcodeAddressPointer:
		return e.instrAddress(pc, in)
	case ir.OpcodePrintText, ir.OpcodePrintStr, ir.OpcodePrintValue, ir.OpcodeTestEqual:
		e.stats.StatsRuntime++
		return e.instrRuntime(pc, in)
	}
	fact := &opcodeFacts[in.InstrOpcode]
	if fact.OpcodeFactName == "" {
		return fmt.Errorf("opcode %d not compiled", in.InstrOpcode)
	}
	return e.instrScalar(pc, in, fact)
}

func (e *emitter) instrReturn() error {
	fn := e.fn
	if len(fn.FuncOutputs) == 0 {
		e.line("ret void")
		return nil
	}
	aggregate := "undef"
	rt := e.retType(fn)
	for i, slot := range fn.FuncOutputs {
		p, err := e.ptrAt(slot.SlotOffset, true, slot.SlotSize)
		if err != nil {
			return err
		}
		v := e.newTmp()
		e.line("%s = load %s, ptr %s, align 1", v, valueType(slot.SlotSize), p)
		next := e.newTmp()
		e.line("%s = insertvalue %s %s, %s %s, %d", next, rt, aggregate, valueType(slot.SlotSize), v, i)
		aggregate = next
	}
	e.line("ret %s %s", rt, aggregate)
	return nil
}

func (e *emitter) instrCallFrame(pc uint32, in *ir.Instr) error {
	if e.call.pendingCallActive {
		return fmt.Errorf("nested call")
	}
	e.call = pendingCall{pendingCallActive: true, pendingCallFrame: in.InstrN, pendingCallArgs: map[uint32]string{}, pendingCallSizes: map[uint32]uint32{}}
	d := e.newTmp()
	e.line("%s = icmp uge i32 %%depth, %d", d, ir.ImageCallDepthMax-1)
	e.check(d, StopStackOverflow, pc)
	end := e.newTmp()
	e.line("%s = add i64 %%fp64, %d", end, uint64(e.fn.FuncFrameSize)+uint64(in.InstrN))
	s := e.newTmp()
	e.line("%s = icmp ugt i64 %s, %d", s, end, e.image.ImageStackSize)
	e.check(s, StopStackOverflow, pc)
	return nil
}

func (e *emitter) instrCall(in *ir.Instr) error {
	callee := &e.image.ImageFuncs[in.InstrM]
	if len(callee.FuncCode) == 0 {
		return fmt.Errorf("call of %s, which has no code", e.image.ImageFuncNames[in.InstrM])
	}
	if callee.FuncFrameSize != e.call.pendingCallFrame {
		return fmt.Errorf("call frame %d, callee frame %d", e.call.pendingCallFrame, callee.FuncFrameSize)
	}
	d := e.newTmp()
	e.line("%s = add i32 %%depth, 1", d)
	args := []string{"i32 %sp", "i32 " + d}
	used := 0
	for _, slot := range callee.FuncInputs {
		typ := valueType(slot.SlotSize)
		if v, found := e.call.pendingCallArgs[slot.SlotOffset]; found {
			if e.call.pendingCallSizes[slot.SlotOffset] != slot.SlotSize {
				return fmt.Errorf("argument size %d for a slot of %d", e.call.pendingCallSizes[slot.SlotOffset], slot.SlotSize)
			}
			args = append(args, typ+" "+v)
			used++
			continue
		}
		args = append(args, typ+" "+zeroValue(slot.SlotSize))
	}
	if used != len(e.call.pendingCallArgs) {
		return fmt.Errorf("argument outside the callee inputs")
	}
	rt := e.retType(callee)
	if rt == "void" {
		e.line("call void %s(%s)", e.funcName(in.InstrM), strings.Join(args, ", "))
		e.result = ""
	} else {
		e.result = e.newTmp()
		e.line("%s = call %s %s(%s)", e.result, rt, e.funcName(in.InstrM), strings.Join(args, ", "))
	}
	e.resultOf = in.InstrM
	e.resultBlock = false
	e.call = pendingCall{}
	return nil
}

func (e *emitter) instrCallResult(in *ir.Instr) error {
	if e.resultBlock {
		return e.instrLibraryResult(in)
	}
	callee := &e.image.ImageFuncs[e.resultOf]
	for i, slot := range callee.FuncOutputs {
		if slot.SlotOffset != in.InstrA {
			continue
		}
		if slot.SlotSize != in.InstrN {
			return fmt.Errorf("result size %d of a slot of %d", in.InstrN, slot.SlotSize)
		}
		v := e.newTmp()
		e.line("%s = extractvalue %s %s, %d", v, e.retType(callee), e.result, i)
		return e.store(in, 2, valueType(slot.SlotSize), slot.SlotSize, v)
	}
	return fmt.Errorf("result %d is not a callee output", in.InstrA)
}

func (e *emitter) instrAddress(pc uint32, in *ir.Instr) error {
	op := in.InstrOpcode
	switch op {
	case ir.OpcodeAddressIndexStatic, ir.OpcodeAddressIndex:
		var base string
		if op == ir.OpcodeAddressIndexStatic {
			base = e.vmAddress(in, 0)
		} else {
			slot, err := e.load(in, 0, "i32", 4)
			if err != nil {
				return err
			}
			base = e.newTmp()
			e.line("%s = add i32 %s, %d", base, slot, in.InstrM)
		}
		index, err := e.load(in, 1, "i32", 4)
		if err != nil {
			return err
		}
		negative := e.newTmp()
		e.line("%s = icmp slt i32 %s, 0", negative, index)
		e.check(negative, StopIndexNegative, pc)
		product := e.newTmp()
		e.line("%s = mul i32 %s, %d", product, index, in.InstrN)
		sum := e.newTmp()
		e.line("%s = add i32 %s, %s", sum, base, product)
		return e.store(in, 2, "i32", 4, sum)
	case ir.OpcodeAddressDerefStatic, ir.OpcodeAddressDeref:
		var p string
		if op == ir.OpcodeAddressDerefStatic {
			var err error
			if p, err = e.ptr(in, 0, pointerSize); err != nil {
				return err
			}
		} else {
			slot, err := e.load(in, 0, "i32", 4)
			if err != nil {
				return err
			}
			p = e.memAt(slot, in.InstrM, pointerSize, pc)
		}
		v := e.newTmp()
		e.line("%s = load %s, ptr %s, align 1", v, valueType(pointerSize), p)
		bad := e.newTmp()
		e.line("%s = icmp uge %s %s, %d", bad, valueType(pointerSize), v, e.dataEnd)
		e.check(bad, StopAddressInvalid, pc)
		return e.store(in, 2, "i32", 4, e.pointerTo32(v))
	case ir.OpcodeAddressLoad:
		slot, err := e.load(in, 0, "i32", 4)
		if err != nil {
			return err
		}
		src := e.memAt(slot, in.InstrM, in.InstrN, pc)
		dst, err := e.ptr(in, 2, in.InstrN)
		if err != nil {
			return err
		}
		e.copyBytes(dst, src, in.InstrN)
		return nil
	case ir.OpcodeAddressStore:
		slot, err := e.load(in, 2, "i32", 4)
		if err != nil {
			return err
		}
		dst := e.memAt(slot, in.InstrM, in.InstrN, pc)
		src, err := e.ptr(in, 0, in.InstrN)
		if err != nil {
			return err
		}
		e.copyBytes(dst, src, in.InstrN)
		return nil
	}
	var v string
	if op == ir.OpcodeAddressOf {
		v = e.vmAddress(in, 0)
	} else {
		slot, err := e.load(in, 0, "i32", 4)
		if err != nil {
			return err
		}
		v = e.newTmp()
		e.line("%s = add i32 %s, %d", v, slot, in.InstrM)
	}
	if pointerSize != 4 {
		t := e.newTmp()
		e.line("%s = zext i32 %s to %s", t, v, valueType(pointerSize))
		v = t
	}
	return e.store(in, 2, valueType(pointerSize), pointerSize, v)
}

// instrRuntime emits the instructions that only the runtime runs: the
// prints and test.
func (e *emitter) instrRuntime(pc uint32, in *ir.Instr) error {
	switch in.InstrOpcode {
	case ir.OpcodePrintText:
		e.declare("cx_rt_print_text", "declare void @cx_rt_print_text(ptr, i32) nounwind")
		p := e.newTmp()
		e.line("%s = getelementptr inbounds i8, ptr @cx_text, i64 %d", p, in.InstrN)
		e.line("call void @cx_rt_print_text(ptr %s, i32 %d)", p, in.InstrM)
		return nil
	case ir.OpcodePrintStr:
		v, err := e.load(in, 0, valueType(pointerSize), pointerSize)
		if err != nil {
			return err
		}
		e.declare("cx_rt_print_str", "declare i32 @cx_rt_print_str(ptr, i32, i32, i32, i32) nounwind")
		ok := e.newTmp()
		e.line("%s = call i32 @cx_rt_print_str(ptr @cx_mem, i32 %d, i32 %d, i32 %s, i32 %d)", ok, e.dataEnd, pointerSize, e.pointerTo32(v), in.InstrN)
		bad := e.newTmp()
		e.line("%s = icmp eq i32 %s, 0", bad, ok)
		e.check(bad, StopAddressInvalid, pc)
		return nil
	case ir.OpcodePrintValue:
		code := types.Code(in.InstrM)
		size := typeSize(code)
		if size == 0 {
			return fmt.Errorf("printvalue of type code %d", code)
		}
		typ := typeLlvm(code)
		v, err := e.load(in, 0, typ, size)
		if err != nil {
			return err
		}
		name := "cx_rt_print_value_" + code.Name()
		param := typ + extension(code)
		e.declare(name, fmt.Sprintf("declare void @%s(%s, i32) nounwind", name, param))
		e.line("call void @%s(%s %s, i32 %d)", name, param, v, in.InstrN)
		return nil
	}
	a, err := e.ptr(in, 0, in.InstrN)
	if err != nil {
		return err
	}
	b, err := e.ptr(in, 1, in.InstrM)
	if err != nil {
		return err
	}
	message, err := e.load(in, 2, valueType(pointerSize), pointerSize)
	if err != nil {
		return err
	}
	file, number := e.where(pc)
	where := e.str(fmt.Sprintf("%s: %d", file, number))
	e.declare("cx_rt_test_equal", "declare void @cx_rt_test_equal(ptr, i32, ptr, i32, ptr, i32, i32, i32, ptr) nounwind")
	e.line("call void @cx_rt_test_equal(ptr %s, i32 %d, ptr %s, i32 %d, ptr @cx_mem, i32 %d, i32 %d, i32 %s, ptr %s)",
		a, in.InstrN, b, in.InstrM, e.dataEnd+e.image.ImageScratchSize, pointerSize, e.pointerTo32(message), where)
	return nil
}

// instrScalar emits a scalar opcode as its template, or as a call of its
// runtime function when it has none.
func (e *emitter) instrScalar(pc uint32, in *ir.Instr, fact *OpcodeFact) error {
	operandType := typeLlvm(fact.OpcodeFactType)
	size := typeSize(fact.OpcodeFactType)
	a, err := e.load(in, 0, operandType, size)
	if err != nil {
		return err
	}
	b := ""
	if fact.OpcodeFactInputs == 2 {
		if b, err = e.load(in, 1, operandType, size); err != nil {
			return err
		}
	}
	if fact.OpcodeFactTemplate != "" && !e.options.FallbackOnly {
		e.stats.StatsInline[in.InstrOpcode]++
		r := e.newTmp()
		e.expand(fact, a, b, r, pc)
		return e.store(in, 2, typeLlvm(fact.OpcodeFactResult), typeSize(fact.OpcodeFactResult), r)
	}
	e.stats.StatsFallback[in.InstrOpcode]++
	name := FallbackName(in.InstrOpcode)
	param := operandType + extension(fact.OpcodeFactType)
	params := []string{param}
	args := []string{param + " " + a}
	if b != "" {
		params = append(params, param)
		args = append(args, param+" "+b)
	}
	if fallbackStops(fact) {
		params = append(params, "ptr")
		args = append(args, "ptr "+e.str(e.stopText(pc, StopDivideByZero)))
	}
	if fact.OpcodeFactResult == types.UNDEFINED {
		e.declare(name, fmt.Sprintf("declare void @%s(%s) nounwind", name, strings.Join(params, ", ")))
		e.line("call void @%s(%s)", name, strings.Join(args, ", "))
		return nil
	}
	resultType := typeLlvm(fact.OpcodeFactResult)
	result := strings.TrimPrefix(extension(fact.OpcodeFactResult)+" ", " ") + resultType
	attributes := "nounwind willreturn memory(none)"
	if fallbackStops(fact) {
		attributes = "nounwind"
	}
	e.declare(name, fmt.Sprintf("declare %s @%s(%s) %s", result, name, strings.Join(params, ", "), attributes))
	r := e.newTmp()
	e.line("%s = call %s @%s(%s)", r, result, name, strings.Join(args, ", "))
	return e.store(in, 2, resultType, typeSize(fact.OpcodeFactResult), r)
}

// fallbackStops reports whether the runtime function of an opcode can stop
// the run, an integer division or modulo. It takes the stop text last.
func fallbackStops(fact *OpcodeFact) bool {
	return !typeFloat(fact.OpcodeFactType) && (fact.OpcodeFactOperation == "div" || fact.OpcodeFactOperation == "mod")
}

// FallbackName is the runtime function a scalar opcode calls when it has
// no template, cx_rt_ and its language name with _ for the dot.
func FallbackName(op ir.Opcode) string {
	return "cx_rt_" + strings.ReplaceAll(opcodeFacts[op].OpcodeFactName, ".", "_")
}

// OpcodeName names an image opcode, by its language name for a scalar
// opcode.
func OpcodeName(op ir.Opcode) string {
	if op < ir.OpcodeCount && opcodeFacts[op].OpcodeFactName != "" {
		return opcodeFacts[op].OpcodeFactName
	}
	return "opcode " + strconv.Itoa(int(op))
}

// dataRuns returns the runs of the data segment that are not zero, as
// [start, end) offsets into ImageData, joining runs less than 64 bytes apart.
func (e *emitter) dataRuns() [][2]int {
	data := e.image.ImageData
	var runs [][2]int
	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			continue
		}
		if n := len(runs); n > 0 && i-runs[n-1][1] < 64 {
			runs[n-1][1] = i + 1
			continue
		}
		runs = append(runs, [2]int{i, i + 1})
	}
	return runs
}

// readOnlyMark marks the bytes of ImageReadOnly.
func (e *emitter) readOnlyMark() {
	start := e.image.ImageDataStart
	e.readOnly = make([]bool, len(e.image.ImageData))
	for _, span := range e.image.ImageReadOnly {
		for address := max(span.SpanStart, start); address < span.SpanEnd && address < e.dataEnd; address++ {
			e.readOnly[address-start] = true
		}
	}
}

// constant returns the value of size bytes at the absolute address offset
// as an LLVM constant of typ, false when the bytes are not a literal.
func (e *emitter) constant(offset uint32, inFrame bool, typ string, size uint32) (string, bool) {
	if inFrame || size > 8 || offset < e.image.ImageDataStart || offset+size > e.dataEnd {
		return "", false
	}
	for address := offset; address < offset+size; address++ {
		if !e.readOnly[address-e.image.ImageDataStart] {
			return "", false
		}
	}
	var v uint64
	for i := uint32(0); i < size; i++ {
		v |= uint64(e.image.ImageData[offset-e.image.ImageDataStart+i]) << (8 * i)
	}
	switch typ {
	case "float":
		t := e.newTmp()
		e.line("%s = bitcast i32 %d to float", t, int32(uint32(v)))
		return t, true
	case "double":
		t := e.newTmp()
		e.line("%s = bitcast i64 %d to double", t, int64(v))
		return t, true
	}
	switch size {
	case 1:
		return strconv.Itoa(int(int8(v))), true
	case 2:
		return strconv.Itoa(int(int16(v))), true
	case 4:
		return strconv.Itoa(int(int32(v))), true
	case 8:
		return strconv.FormatInt(int64(v), 10), true
	}
	return "", false
}

// extension is the attribute that widens a value of code passed to or
// returned from C, as the C calling conventions of the targets want a
// small integer widened by the caller.
func extension(code types.Code) string {
	switch code {
	case types.I8, types.I16:
		return " signext"
	case types.BOOL, types.UI8, types.UI16:
		return " zeroext"
	}
	return ""
}

// llvmIntrinsics are the intrinsics a template may call, each declared for
// the operand type when a template uses it.
var llvmIntrinsics = []string{"llvm.abs", "llvm.smax", "llvm.smin", "llvm.umax", "llvm.umin", "llvm.fabs"}

// expand writes the template of fact with its placeholders filled, a and b
// the operands and r the result.
func (e *emitter) expand(fact *OpcodeFact, a, b, r string, pc uint32) {
	operandType := typeLlvm(fact.OpcodeFactType)
	bits := 8 * typeSize(fact.OpcodeFactType)
	floatSuffix := map[types.Code]string{types.F32: "f32", types.F64: "f64"}[fact.OpcodeFactType]
	pairs := []string{"{a}", a, "{b}", b, "{r}", r, "{T}", operandType, "{R}", typeLlvm(fact.OpcodeFactResult), "{F}", floatSuffix,
		"{W1}", strconv.Itoa(int(bits) - 1), "{W}", strconv.Itoa(int(bits))}
	for i := 1; i <= 9; i++ {
		key := "{t" + strconv.Itoa(i) + "}"
		if strings.Contains(fact.OpcodeFactTemplate, key) {
			pairs = append(pairs, key, e.newTmp())
		}
	}
	text := strings.NewReplacer(pairs...).Replace(fact.OpcodeFactTemplate)
	for _, line := range strings.Split(text, "\n") {
		if check, found := strings.CutPrefix(line, "check "); found {
			condition, stop, _ := strings.Cut(check, " ")
			assert.AssertInvariantTrue(stop == "DivideByZero", "llvm: template stop "+stop)
			e.check(condition, StopDivideByZero, pc)
			continue
		}
		e.line("%s", line)
	}
	for _, intrinsic := range llvmIntrinsics {
		if !strings.Contains(text, "@"+intrinsic+".") {
			continue
		}
		switch intrinsic {
		case "llvm.fabs":
			e.declare(intrinsic+"."+floatSuffix, fmt.Sprintf("declare %s @%s.%s(%s)", operandType, intrinsic, floatSuffix, operandType))
		case "llvm.abs":
			e.declare(intrinsic+"."+operandType, fmt.Sprintf("declare %s @%s.%s(%s, i1)", operandType, intrinsic, operandType, operandType))
		default:
			e.declare(intrinsic+"."+operandType, fmt.Sprintf("declare %s @%s.%s(%s, %s)", operandType, intrinsic, operandType, operandType, operandType))
		}
	}
}
