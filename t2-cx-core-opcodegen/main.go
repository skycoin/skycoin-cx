// Command opcodegen writes the opcode signature tables of t1-constants from
// constants.OpcodeDefinitionTable, decision D21, and the library signature
// tables and library switches from constants.LibraryFunctionDefinitionTable,
// decision D96. With -check it fails when a
// written file differs from what it would write. Run it with go generate
// ./t1-constants or make generate-opcodes.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
)

// handlerPackages maps the package of a Handler to its group.
var handlerPackages = map[string]constants.OpcodeGroup{
	"opcodes":  constants.OpcodeGroupCore,
	"printer":  constants.OpcodeGroupCore,
	"cxoscore": constants.OpcodeGroupOsCore,
	"regexp":   constants.OpcodeGroupRegexp,
	"cxfx":     constants.OpcodeGroupCxfx,
}

// generated is everything opcodegen derives from the definition table.
type generated struct {
	parameters []constants.OpcodeParameter
	structs    []constants.OpcodeParameterStruct
	signatures map[int]constants.OpcodeSignature
	operators  map[types.Code]map[int]int
	nameRows   map[int]int
	names      []constants.NativeName
	size       int
}

func main() {
	root := flag.String("root", "..", "the repository root")
	check := flag.Bool("check", false, "fail when a generated file is out of date")
	flag.Parse()
	gen, err := generate()
	if err != nil {
		fmt.Fprintln(os.Stderr, "opcodegen:", err)
		os.Exit(1)
	}
	libraries, err := librariesGenerate()
	if err != nil {
		fmt.Fprintln(os.Stderr, "opcodegen:", err)
		os.Exit(1)
	}
	files := map[string][]byte{
		"t1-constants/opcodes_signatures_gen.go":   gen.signaturesFile(),
		"t1-constants/libraries_signatures_gen.go": libraries.signaturesFile(),
	}
	for path, source := range gen.dispatchFiles() {
		files[path] = source
	}
	for path, source := range libraries.dispatchFiles() {
		files[path] = source
	}
	failed := false
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		source, err := format.Source(files[path])
		if err != nil {
			fmt.Fprintln(os.Stderr, "opcodegen:", path, err)
			os.Exit(1)
		}
		full := filepath.Join(*root, path)
		if *check {
			old, _ := os.ReadFile(full)
			if !bytes.Equal(old, source) {
				fmt.Fprintf(os.Stderr, "opcodegen: %s is out of date, run make generate-opcodes\n", path)
				failed = true
			}
			continue
		}
		if err := os.WriteFile(full, source, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "opcodegen:", err)
			os.Exit(1)
		}
	}
	if failed {
		os.Exit(1)
	}
}

// typeCodes maps a type name, as types.Code.Name spells it, to its code.
func typeCodes() map[string]types.Code {
	codes := map[string]types.Code{}
	for code := types.BOOL; code < types.COUNT; code++ {
		codes[code.Name()] = code
	}
	return codes
}

// generate checks the definition table against the lifecycle table and
// derives the tables.
func generate() (*generated, error) {
	gen := &generated{
		signatures: map[int]constants.OpcodeSignature{},
		operators:  map[types.Code]map[int]int{},
		nameRows:   map[int]int{},
	}
	codes := typeCodes()
	defined := map[int]bool{}
	previous := 0
	for _, row := range constants.OpcodeDefinitionTable {
		live, found := constants.OpcodeLifecycleLiveByNumber(row.Number)
		if !found || live.Name != row.Name {
			return nil, fmt.Errorf("definition %d %q is not a live row of OpcodeLifecycleTable", row.Number, row.Name)
		}
		if row.Number <= previous {
			return nil, fmt.Errorf("definition %d %q is out of number order", row.Number, row.Name)
		}
		previous = row.Number
		defined[row.Number] = true
		signature := constants.OpcodeSignature{OpcodeSignatureGroup: constants.OpcodeGroupCore}
		if row.Handler != "" {
			pkg, function, ok := strings.Cut(row.Handler, ".")
			group, known := handlerPackages[pkg]
			if !ok || !known || function == "" || function[0] < 'A' || function[0] > 'Z' {
				return nil, fmt.Errorf("definition %q: handler %q is not an exported function of a known package", row.Name, row.Handler)
			}
			signature.OpcodeSignatureGroup = group
			signature.OpcodeSignatureHandled = true
		}
		signature.OpcodeSignatureParameterFirst = uint16(len(gen.parameters))
		inputs, err := gen.parametersParse(codes, row.Inputs)
		if err != nil {
			return nil, fmt.Errorf("definition %q inputs: %w", row.Name, err)
		}
		outputs, err := gen.parametersParse(codes, row.Outputs)
		if err != nil {
			return nil, fmt.Errorf("definition %q outputs: %w", row.Name, err)
		}
		signature.OpcodeSignatureInputCount = uint8(inputs)
		signature.OpcodeSignatureOutputCount = uint8(outputs)
		gen.signatures[row.Number] = signature
		if row.Operator != "" {
			if err := gen.operatorAdd(codes, row); err != nil {
				return nil, err
			}
		}
		gen.size = row.Number + 1
	}
	for index, row := range constants.OpcodeLifecycleTable {
		if row.VersionDeprecated != 0 {
			continue
		}
		if !defined[row.Number] && row.Number != constants.OP_LIBRARY_CALL {
			return nil, fmt.Errorf("live opcode %d %q has no definition row", row.Number, row.Name)
		}
		gen.nameRows[row.Number] = index + 1
		if row.Number >= gen.size {
			gen.size = row.Number + 1
		}
		if defined[row.Number] {
			gen.names = append(gen.names, constants.NativeName{NativeNameText: row.Name, NativeNameCode: row.Number})
		}
	}
	for _, row := range constants.LibraryFunctionLifecycleTable {
		if row.VersionDeprecated == 0 {
			gen.names = append(gen.names, constants.NativeName{NativeNameText: row.Name, NativeNameCode: constants.LibraryFunctionKey(row.LibraryNumber, row.Number)})
		}
	}
	sort.Slice(gen.names, func(left int, right int) bool {
		return gen.names[left].NativeNameText < gen.names[right].NativeNameText
	})
	for i := 1; i < len(gen.names); i++ {
		if gen.names[i].NativeNameText == gen.names[i-1].NativeNameText {
			return nil, fmt.Errorf("built-in name %q is used twice", gen.names[i].NativeNameText)
		}
	}
	return gen, nil
}

// parametersParse appends the parameters spelled in text and returns their
// count.
func (gen *generated) parametersParse(codes map[string]types.Code, text string) (int, error) {
	fields := strings.Fields(text)
	for _, field := range fields {
		parameter := constants.OpcodeParameter{OpcodeParameterForm: constants.OpcodeParameterFormAtomic}
		if name, structName, ok := strings.Cut(field, ":"); ok {
			pkg, typeName, ok := strings.Cut(structName, ".")
			if !ok || name == "" {
				return 0, fmt.Errorf("struct parameter %q is not name:pkg.Struct", field)
			}
			parameter = constants.OpcodeParameter{OpcodeParameterForm: constants.OpcodeParameterFormStruct, OpcodeParameterType: types.STRUCT, OpcodeParameterStruct: gen.structIndex(pkg, typeName, name)}
			gen.parameters = append(gen.parameters, parameter)
			continue
		}
		if element, ok := strings.CutPrefix(field, "[]"); ok {
			parameter.OpcodeParameterForm = constants.OpcodeParameterFormSlice
			field = element
		}
		code, known := codes[field]
		if !known {
			return 0, fmt.Errorf("unknown type %q", field)
		}
		parameter.OpcodeParameterType = code
		gen.parameters = append(gen.parameters, parameter)
	}
	return len(fields), nil
}

// structIndex returns the row of the struct parameter, adding it when new.
func (gen *generated) structIndex(pkg string, typeName string, name string) uint8 {
	parameter := constants.OpcodeParameterStruct{OpcodeParameterStructPackage: pkg, OpcodeParameterStructType: typeName, OpcodeParameterStructName: name}
	for index, known := range gen.structs {
		if known == parameter {
			return uint8(index)
		}
	}
	gen.structs = append(gen.structs, parameter)
	return uint8(len(gen.structs) - 1)
}

// operatorAdd records row as the typed form of its untyped operator.
func (gen *generated) operatorAdd(codes map[string]types.Code, row constants.OpcodeDefinition) error {
	typeName, operatorName, ok := strings.Cut(row.Operator, " ")
	code, known := codes[typeName]
	operator, found := constants.OpcodeLifecycleLiveByName(operatorName)
	if !ok || !known || !found || operator.Number <= constants.START_OF_OPERATORS || operator.Number >= constants.END_OF_OPERATORS {
		return fmt.Errorf("definition %q: operator %q is not a type and an untyped operator", row.Name, row.Operator)
	}
	if gen.operators[code] == nil {
		gen.operators[code] = map[int]int{}
	}
	if other, taken := gen.operators[code][operator.Number]; taken {
		return fmt.Errorf("definition %q: operator %q is also opcode %d", row.Name, row.Operator, other)
	}
	gen.operators[code][operator.Number] = row.Number
	return nil
}

// typeConstant returns the name of the types constant for code.
func typeConstant(code types.Code) string {
	switch code {
	case types.POINTER:
		return "types.POINTER"
	case types.UNDEFINED:
		return "types.UNDEFINED"
	}
	return "types." + strings.ToUpper(code.Name())
}

var parameterForms = map[constants.OpcodeParameterForm]string{
	constants.OpcodeParameterFormAtomic: "OpcodeParameterFormAtomic",
	constants.OpcodeParameterFormSlice:  "OpcodeParameterFormSlice",
	constants.OpcodeParameterFormStruct: "OpcodeParameterFormStruct",
}

var groupNames = map[constants.OpcodeGroup]string{
	constants.OpcodeGroupCore:   "OpcodeGroupCore",
	constants.OpcodeGroupOsCore: "OpcodeGroupOsCore",
	constants.OpcodeGroupRegexp: "OpcodeGroupRegexp",
	constants.OpcodeGroupCxfx:   "OpcodeGroupCxfx",
}

const header = "// Code generated by t2-cx-core-opcodegen from OpcodeDefinitionTable. DO NOT EDIT.\n\n"

func (gen *generated) signaturesFile() []byte {
	var b bytes.Buffer
	b.WriteString(header)
	b.WriteString("package constants\n\nimport (\n\t\"github.com/skycoin/skycoin-cx/t0-types\" // thin: only Code constants; core types, keep\n)\n\n")
	fmt.Fprintf(&b, "// OpcodeSignatureTableSize is one past the highest live opcode number.\nconst OpcodeSignatureTableSize = %d\n\n", gen.size)
	b.WriteString("// OpcodeParameterTable holds the parameters of every opcode signature. Read only.\nvar OpcodeParameterTable = [...]OpcodeParameter{\n")
	for _, parameter := range gen.parameters {
		fmt.Fprintf(&b, "\t{%s, %s, %d},\n", parameterForms[parameter.OpcodeParameterForm], typeConstant(parameter.OpcodeParameterType), parameter.OpcodeParameterStruct)
	}
	b.WriteString("}\n\n// OpcodeParameterStructTable holds the struct types of struct parameters. Read only.\nvar OpcodeParameterStructTable = [...]OpcodeParameterStruct{\n")
	for _, parameter := range gen.structs {
		fmt.Fprintf(&b, "\t{%q, %q, %q},\n", parameter.OpcodeParameterStructPackage, parameter.OpcodeParameterStructType, parameter.OpcodeParameterStructName)
	}
	b.WriteString("}\n\n// OpcodeSignatureTable is the signature of each opcode by number. Read only.\nvar OpcodeSignatureTable = [OpcodeSignatureTableSize]OpcodeSignature{\n")
	for number := 0; number < gen.size; number++ {
		signature, found := gen.signatures[number]
		if !found {
			continue
		}
		fmt.Fprintf(&b, "\t%d: {%s, %t, %d, %d, %d},\n", number, groupNames[signature.OpcodeSignatureGroup], signature.OpcodeSignatureHandled, signature.OpcodeSignatureParameterFirst, signature.OpcodeSignatureInputCount, signature.OpcodeSignatureOutputCount)
	}
	b.WriteString("}\n\n// OpcodeOperatorTable is the typed form of each untyped operator by operand\n// type, indexed by the operator number less START_OF_OPERATORS plus one. Read only.\n")
	b.WriteString("var OpcodeOperatorTable = [types.COUNT][END_OF_OPERATORS - START_OF_OPERATORS]uint16{\n")
	for code := types.BOOL; code < types.COUNT; code++ {
		operators := gen.operators[code]
		if len(operators) == 0 {
			continue
		}
		numbers := make([]int, 0, len(operators))
		for operator := range operators {
			numbers = append(numbers, operator)
		}
		sort.Ints(numbers)
		fmt.Fprintf(&b, "\t%s: {\n", typeConstant(code))
		for _, operator := range numbers {
			fmt.Fprintf(&b, "\t\t%d - START_OF_OPERATORS - 1: %d,\n", operator, operators[operator])
		}
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n\n// OpcodeNameRowTable is one plus the row of OpcodeLifecycleTable that names\n// each live opcode number, 0 for none. Read only.\n")
	b.WriteString("var OpcodeNameRowTable = [OpcodeSignatureTableSize]uint16{\n")
	for number := 0; number < gen.size; number++ {
		if row, found := gen.nameRows[number]; found {
			fmt.Fprintf(&b, "\t%d: %d,\n", number, row)
		}
	}
	b.WriteString("}\n\n// NativeNameTable is every name a program calls a built-in by, the opcodes of\n// OpcodeDefinitionTable and the live library functions, sorted by name. Read only.\n")
	b.WriteString("var NativeNameTable = [...]NativeName{\n")
	for _, name := range gen.names {
		fmt.Fprintf(&b, "\t{%q, %d},\n", name.NativeNameText, name.NativeNameCode)
	}
	b.WriteString("}\n")
	return b.Bytes()
}
