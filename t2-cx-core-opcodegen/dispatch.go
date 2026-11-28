package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
)

// dispatchGroup is one switch opcodegen writes, with the build tag of its
// file and of the file that stands in for it without the tag.
type dispatchGroup struct {
	group      opcodetable.OpcodeGroup
	path       string
	pkg        string
	function   string
	tag        string
	tagOff     string
	importPath string
	importName string
}

var dispatchGroups = []dispatchGroup{
	{opcodetable.OpcodeGroupOsCore, "t5-cx-runtime-core/core_dispatch_oscore", "core", "coreOpcodeDispatchOsCore", "!nocxos", "nocxos", "github.com/skycoin/skycoin-cx/t4-cx-library-osopcodes", "osopcodes"},
	{opcodetable.OpcodeGroupRegexp, "t5-cx-runtime-core/core_dispatch_regexp", "core", "coreOpcodeDispatchRegexp", "!noregexp", "noregexp", "github.com/skycoin/skycoin-cx/t4-cx-library-regexp", "regexp"},
	{opcodetable.OpcodeGroupTerminal, "t5-cx-runtime-core/core_dispatch_terminal", "core", "coreOpcodeDispatchTerminal", "", "", "github.com/skycoin/skycoin-cx/t4-cx-library-terminalopcodes", "terminalopcodes"},
	{opcodetable.OpcodeGroupCxfx, "t6-cx-library-graphics/cxfx_dispatch", "graphics", "CxfxOpcodeDispatch", "cxfx", "!cxfx", "", ""},
	{opcodetable.OpcodeGroupCipher, "t4-cx-library-cipher/cipher_dispatch", "cipher", "CipherOpcodeDispatch", "!nocipher", "nocipher", "", ""},
	{opcodetable.OpcodeGroupNet, "t4-cx-library-netopcodes/net_dispatch", "netopcodes", "NetOpcodeDispatch", "", "", "", ""},
	{opcodetable.OpcodeGroupFile, "t5-cx-library-fileopcodes/file_dispatch", "fileopcodes", "FileOpcodeDispatch", "!nocxos", "nocxos", "", ""},
	{opcodetable.OpcodeGroupReflect, "t4-cx-library-reflectopcodes/reflect_dispatch", "reflectopcodes", "ReflectOpcodeDispatch", "", "", "", ""},
}

// coreImports are the handler packages of OpcodeGroupCore.
var coreImports = map[string]string{
	"opcodes": "github.com/skycoin/skycoin-cx/t4-cx-runtime-opcodes",
	"printer": "github.com/skycoin/skycoin-cx/t4-cx-runtime-printer",
}

const astImport = "\t// GCREACH: t3-cx-core-ast owns the heap. The switch hands the operands to the\n\t// handlers and allocates nothing, see docs/GarbageCollector.md.\n\t\"github.com/skycoin/skycoin-cx/t3-cx-core-ast\"\n"

const dispatchParameters = "(prgrm *ast.CXProgram, code int, inputs []ast.CXValue, outputs []ast.CXValue) bool"

// dispatchFiles returns the switch files: CoreOpcodeDispatch for the core
// group, with the opcodes of the tagged core groups in their own files, and
// CxfxOpcodeDispatch in t6-cx-library-graphics.
func (gen *generated) dispatchFiles() map[string][]byte {
	files := map[string][]byte{}
	var b bytes.Buffer
	b.WriteString(header)
	b.WriteString("package core\n\nimport (\n" + astImport)
	for _, name := range []string{"opcodes", "printer"} {
		fmt.Fprintf(&b, "\t%s %q\n", name, coreImports[name])
	}
	b.WriteString(")\n\n// CoreOpcodeDispatch runs opcode code of the core on its operands, the\n// handler of its row of opcodetable.OpcodeDefinitionTable, and reports false for\n// an opcode the core does not link (decision D21).\n")
	b.WriteString("func CoreOpcodeDispatch" + dispatchParameters + " {\n\tswitch code {\n")
	for _, row := range opcodetable.OpcodeDefinitionTable {
		pkg, function, _ := strings.Cut(row.GoFunction, ".")
		if goFunctionPackages[pkg] == opcodetable.OpcodeGroupCore && row.GoFunction != "" {
			fmt.Fprintf(&b, "\tcase %d:\n\t\t%s.%s(prgrm, inputs, outputs)\n", row.Number, pkg, function)
		}
	}
	for _, group := range dispatchGroups[:3] {
		numbers := gen.groupNumbers(group.group)
		fmt.Fprintf(&b, "\tcase %s:\n\t\treturn %s(prgrm, code, inputs, outputs)\n", strings.Join(numbers, ", "), group.function)
	}
	b.WriteString("\tdefault:\n\t\treturn false\n\t}\n\treturn true\n}\n")
	files["t5-cx-runtime-core/core_dispatch_gen.go"] = b.Bytes()
	for _, group := range dispatchGroups {
		files[group.path+"_gen.go"] = gen.dispatchGroupFile(group)
		if group.tagOff != "" {
			files[group.path+"_off_gen.go"] = dispatchGroupOffFile(group)
		}
	}
	return files
}

// groupNumbers returns the numbers of the handled opcodes of group.
func (gen *generated) groupNumbers(group opcodetable.OpcodeGroup) []string {
	var numbers []string
	for _, row := range opcodetable.OpcodeDefinitionTable {
		pkg, _, _ := strings.Cut(row.GoFunction, ".")
		if row.GoFunction != "" && goFunctionPackages[pkg] == group {
			numbers = append(numbers, fmt.Sprint(row.Number))
		}
	}
	return numbers
}

func (gen *generated) dispatchGroupFile(group dispatchGroup) []byte {
	var cases bytes.Buffer
	used := map[string]bool{}
	for _, row := range opcodetable.OpcodeDefinitionTable {
		pkg, function, _ := strings.Cut(row.GoFunction, ".")
		if row.GoFunction == "" || goFunctionPackages[pkg] != group.group {
			continue
		}
		if goFunctionInline(function) {
			body, _ := inlineCase(row, function)
			inlinePackages(function, used)
			used["types"] = true
			fmt.Fprintf(&cases, "\tcase %d:\n\t\t%s\n", row.Number, body)
			continue
		}
		call := function
		if group.importName != "" {
			call = group.importName + "." + function
		}
		fmt.Fprintf(&cases, "\tcase %d:\n\t\t%s(prgrm, inputs, outputs)\n", row.Number, call)
	}
	var b bytes.Buffer
	if group.tag != "" {
		fmt.Fprintf(&b, "//go:build %s\n\n", group.tag)
	}
	fmt.Fprintf(&b, "%spackage %s\n\nimport (\n", header, group.pkg)
	names := make([]string, 0, len(used))
	for name := range used {
		if name != "types" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(&b, "\t%q\n", inlineImports[name])
	}
	if len(names) > 0 {
		b.WriteString("\n")
	}
	if used["types"] {
		b.WriteString("\t\"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes\"\n")
	}
	if group.pkg == "core" {
		b.WriteString(astImport)
	} else {
		b.WriteString("\t\"github.com/skycoin/skycoin-cx/t3-cx-core-ast\"\n")
	}
	if group.importPath != "" {
		fmt.Fprintf(&b, "\t%s %q\n", group.importName, group.importPath)
	}
	b.WriteString(")\n\n")
	fmt.Fprintf(&b, "// %s runs the opcode code of opcodetable.%s on its operands\n// and reports false for any other opcode.\n", group.function, groupNames[group.group])
	fmt.Fprintf(&b, "func %s%s {\n\tswitch code {\n", group.function, dispatchParameters)
	b.Write(cases.Bytes())
	b.WriteString("\tdefault:\n\t\treturn false\n\t}\n\treturn true\n}\n")
	return b.Bytes()
}

func dispatchGroupOffFile(group dispatchGroup) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "//go:build %s\n\n%spackage %s\n\nimport (\n", group.tagOff, header, group.pkg)
	if group.pkg == "core" {
		b.WriteString(astImport)
	} else {
		b.WriteString("\t\"github.com/skycoin/skycoin-cx/t3-cx-core-ast\"\n")
	}
	b.WriteString(")\n\n")
	fmt.Fprintf(&b, "// %s reports false, the build tags leave out opcodetable.%s.\n", group.function, groupNames[group.group])
	fmt.Fprintf(&b, "func %s%s {\n\treturn false\n}\n", group.function, dispatchParameters)
	return b.Bytes()
}
