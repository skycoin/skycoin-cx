package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
)

// llvmFile is the generated opcode table of the LLVM target, decision D10.
const llvmFile = "t3-cx-compiler-llvm/llvm_opcodes_gen.go"

// llvmScalarIdents spells each scalar type as the ir.Opcode names do.
var llvmScalarIdents = map[types.Code]string{
	types.BOOL: "Bool", types.I8: "I8", types.I16: "I16", types.I32: "I32", types.I64: "I64",
	types.UI8: "UI8", types.UI16: "UI16", types.UI32: "UI32", types.UI64: "UI64", types.F32: "F32", types.F64: "F64",
}

// llvmOperationIdents spells each operation of the scalar opcode families of
// the image as the ir.Opcode names do. A row whose operation is not here and
// is not a conversion has no image opcode.
var llvmOperationIdents = map[string]string{
	"add": "Add", "sub": "Sub", "mul": "Mul", "div": "Div", "mod": "Mod", "neg": "Neg", "abs": "Abs",
	"max": "Max", "min": "Min", "eq": "Eq", "uneq": "Uneq", "lt": "Lt", "lteq": "Lteq", "gt": "Gt",
	"gteq": "Gteq", "bitand": "BitAnd", "bitor": "BitOr", "bitxor": "BitXor", "bitclear": "BitClear",
	"bitshl": "BitShl", "bitshr": "BitShr", "print": "Print", "not": "Not", "and": "And", "or": "Or",
}

// llvmRow is one scalar image opcode and the definition row it lowers from.
type llvmRow struct {
	ident     string
	row       constants.OpcodeDefinition
	operation string
	operand   types.Code
	result    types.Code
}

// llvmRows finds the definition rows named type.operation or type.type
// whose image opcode the LLVM target compiles, in table order.
func llvmRows() ([]llvmRow, error) {
	codes := typeCodes()
	var rows []llvmRow
	for _, row := range constants.OpcodeDefinitionTable {
		typeName, operation, found := strings.Cut(row.Name, ".")
		operand := codes[typeName]
		typeIdent, scalar := llvmScalarIdents[operand]
		if !found || !scalar {
			continue
		}
		ident := ""
		if operationIdent, known := llvmOperationIdents[operation]; known {
			ident = "Opcode" + typeIdent + operationIdent
		} else if target, conversion := llvmScalarIdents[codes[operation]]; conversion && operand != types.BOOL && codes[operation] != types.BOOL && codes[operation] != operand {
			ident = "Opcode" + typeIdent + "To" + target
		} else {
			continue
		}
		inputs := strings.Fields(row.Inputs)
		outputs := strings.Fields(row.Outputs)
		if len(inputs) < 1 || len(inputs) > 2 || len(outputs) > 1 || inputs[0] != typeName {
			return nil, fmt.Errorf("definition %q: a scalar image opcode takes one or two %s inputs and gives at most one output", row.Name, typeName)
		}
		result := types.UNDEFINED
		if len(outputs) == 1 {
			result = codes[outputs[0]]
			if _, scalar := llvmScalarIdents[result]; !scalar {
				return nil, fmt.Errorf("definition %q: output %q is not a scalar type", row.Name, outputs[0])
			}
		}
		rows = append(rows, llvmRow{ident: ident, row: row, operation: operation, operand: operand, result: result})
	}
	return rows, nil
}

// llvmFileSource writes the opcode table of the LLVM target.
func llvmFileSource(rows []llvmRow) []byte {
	var b bytes.Buffer
	b.WriteString(header)
	b.WriteString("package llvm\n\nimport (\n\t\"github.com/skycoin/skycoin-cx/t0-types\"\n\t\"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir\"\n)\n\n")
	b.WriteString("// opcodeFacts describes each scalar image opcode by the row of\n// OpcodeDefinitionTable it lowers from. Read only.\n")
	b.WriteString("var opcodeFacts = [ir.OpcodeCount]OpcodeFact{\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "\tir.%s: {%q, %q, %s, %d, %s},\n", row.ident, row.row.Name, row.operation, typeConstant(row.operand),
			len(strings.Fields(row.row.Inputs)), typeConstant(row.result))
	}
	b.WriteString("}\n")
	return b.Bytes()
}
