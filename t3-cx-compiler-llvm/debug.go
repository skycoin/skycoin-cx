package llvm

import (
	"fmt"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
)

// debugInfo builds the DWARF metadata of a module, decision D8: a
// subprogram per function, a location per source line from ImageLines, and a
// variable per frame variable of ImageLocals kept in an alloca, with its name
// and type. Metadata !0 is the branch weights, so ids start at 1.
type debugInfo struct {
	debugMeta       []string
	debugFiles      map[uint32]int
	debugLocations  map[[2]uint32]int
	debugTypes      map[string]int
	debugUnit       int
	debugRoutine    int
	debugSubprogram int
	debugCurrent    int
}

func debugInfoNew() *debugInfo {
	d := &debugInfo{debugFiles: map[uint32]int{}, debugLocations: map[[2]uint32]int{}, debugTypes: map[string]int{}}
	d.debugRoutine = d.add("!DISubroutineType(types: !{null})")
	return d
}

// add appends a metadata node and returns its id.
func (d *debugInfo) add(text string) int {
	d.debugMeta = append(d.debugMeta, text)
	return len(d.debugMeta)
}

func (d *debugInfo) file(image *ir.Image, index uint32) int {
	if id, found := d.debugFiles[index]; found {
		return id
	}
	name := ""
	if index < uint32(len(image.ImageFileNames)) {
		name = image.ImageFileNames[index]
	}
	id := d.add(fmt.Sprintf("!DIFile(filename: %s, directory: \"\")", debugString(name)))
	d.debugFiles[index] = id
	if d.debugUnit == 0 {
		d.debugUnit = d.add(fmt.Sprintf("distinct !DICompileUnit(language: DW_LANG_C, file: !%d, producer: \"cx build\", isOptimized: true, runtimeVersion: 0, emissionKind: FullDebug)", id))
	}
	return id
}

// location returns the location of line in the current subprogram.
func (d *debugInfo) location(line uint32) int {
	key := [2]uint32{uint32(d.debugSubprogram), line}
	if id, found := d.debugLocations[key]; found {
		return id
	}
	id := d.add(fmt.Sprintf("!DILocation(line: %d, scope: !%d)", line, d.debugSubprogram))
	d.debugLocations[key] = id
	return id
}

// typeOf returns the type of a variable, a basic type for a scalar and a
// byte array named by its type name for any other.
func (d *debugInfo) typeOf(code types.Code, name string, size uint32) int {
	key := fmt.Sprintf("%d %s %d", code, name, size)
	if id, found := d.debugTypes[key]; found {
		return id
	}
	var id int
	switch {
	case code == types.BOOL:
		id = d.add(fmt.Sprintf("!DIBasicType(name: \"bool\", size: 8, encoding: DW_ATE_boolean)"))
	case code == types.F32 || code == types.F64:
		id = d.add(fmt.Sprintf("!DIBasicType(name: %q, size: %d, encoding: DW_ATE_float)", code.Name(), 8*typeSize(code)))
	case typeSigned(code):
		id = d.add(fmt.Sprintf("!DIBasicType(name: %q, size: %d, encoding: DW_ATE_signed)", code.Name(), 8*typeSize(code)))
	case typeSize(code) > 0:
		id = d.add(fmt.Sprintf("!DIBasicType(name: %q, size: %d, encoding: DW_ATE_unsigned)", code.Name(), 8*typeSize(code)))
	default:
		element := d.typeOf(types.UI8, "", 1)
		subrange := d.add(fmt.Sprintf("!DISubrange(count: %d)", size))
		id = d.add(fmt.Sprintf("!DICompositeType(tag: DW_TAG_array_type, name: %s, baseType: !%d, size: %d, elements: !{!%d})", debugString(name), element, 8*size, subrange))
	}
	d.debugTypes[key] = id
	return id
}

// debugString quotes text as an LLVM metadata string.
func debugString(text string) string {
	return llvmBytes([]byte(text))[1:]
}

// debugFunction starts the subprogram of function id and returns the
// attachment of its define.
func (e *emitter) debugFunction(id uint32, fn *ir.Func) string {
	d := e.debug
	if d == nil {
		return ""
	}
	file, line := uint32(0), uint32(0)
	if fn.FuncLineCount > 0 {
		first := e.image.ImageLines[fn.FuncLineFirst]
		file, line = first.LineFile, first.LineNumber
	}
	fileId := d.file(e.image, file)
	d.debugSubprogram = d.add(fmt.Sprintf("distinct !DISubprogram(name: %s, scope: !%d, file: !%d, line: %d, type: !%d, scopeLine: %d, spFlags: DISPFlagDefinition | DISPFlagOptimized, unit: !%d)",
		debugString(e.image.ImageFuncNames[id]), fileId, fileId, line, d.debugRoutine, line, d.debugUnit))
	d.debugCurrent = d.location(line)
	return fmt.Sprintf(" !dbg !%d", d.debugSubprogram)
}

// debugPc moves the current location to the line of pc.
func (e *emitter) debugPc(pc uint32) {
	if e.debug == nil {
		return
	}
	if line, found := ir.LineAt(e.image, e.function, pc); found {
		e.debug.debugCurrent = e.debug.location(line.LineNumber)
	}
}

// debugLocals declares the variables of fn kept in allocas.
func (e *emitter) debugLocals(fn *ir.Func) {
	d := e.debug
	if d == nil {
		return
	}
	locals := e.image.ImageLocals[fn.FuncLocalFirst : fn.FuncLocalFirst+fn.FuncLocalCount]
	file := uint32(0)
	if fn.FuncLineCount > 0 {
		file = e.image.ImageLines[fn.FuncLineFirst].LineFile
	}
	for _, local := range locals {
		name := e.image.ImageLocalNames[local.LocalName]
		i := find(e.frame, local.LocalOffset)
		if name == "" || i < 0 || e.frame[i].regionShadow {
			continue
		}
		typeId := d.typeOf(local.LocalType, e.image.ImageLocalNames[local.LocalTypeName], local.LocalSize)
		variable := d.add(fmt.Sprintf("!DILocalVariable(name: %s, scope: !%d, file: !%d, type: !%d)", debugString(name), d.debugSubprogram, d.file(e.image, file), typeId))
		expression := "!DIExpression()"
		if offset := local.LocalOffset - e.frame[i].regionStart; offset > 0 {
			expression = fmt.Sprintf("!DIExpression(DW_OP_plus_uconst, %d)", offset)
		}
		e.declare("llvm.dbg.declare", "declare void @llvm.dbg.declare(metadata, metadata, metadata)")
		e.line("call void @llvm.dbg.declare(metadata ptr %s, metadata !%d, metadata %s)", e.frame[i].regionName, variable, expression)
	}
}

// debugModule writes the metadata and the module flags.
func (e *emitter) debugModule() {
	d := e.debug
	if d == nil || d.debugUnit == 0 {
		return
	}
	var b strings.Builder
	for i, text := range d.debugMeta {
		fmt.Fprintf(&b, "!%d = %s\n", i+1, text)
	}
	flags := len(d.debugMeta) + 1
	fmt.Fprintf(&b, "!llvm.dbg.cu = !{!%d}\n!llvm.module.flags = !{!%d, !%d}\n", d.debugUnit, flags, flags+1)
	fmt.Fprintf(&b, "!%d = !{i32 7, !\"Dwarf Version\", i32 4}\n!%d = !{i32 2, !\"Debug Info Version\", i32 3}\n", flags, flags+1)
	e.out.WriteString(b.String())
}

// debugEntry starts the subprogram of the entry function main, which calls
// *init and main, so inlined functions keep their debug information.
func (e *emitter) debugEntry() string {
	if e.debug == nil {
		return ""
	}
	if e.image.ImageEntryMain == ir.ImageFuncNone {
		e.debug.debugCurrent = 0
		return ""
	}
	return e.debugFunction(e.image.ImageEntryMain, &e.image.ImageFuncs[e.image.ImageEntryMain])
}
