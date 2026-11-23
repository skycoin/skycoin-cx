package main

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

// OpcodeTableFile holds OpcodeLifecycleTable, the CX opcode names.
const OpcodeTableFile = "t1-constants/opcodes.go"

// OpcodeNames records every CX opcode name of OpcodeLifecycleTable. The
// package column is the part before the dot, or builtin. A row whose
// deprecated version is not 0 has the kind opcode_deprecated.
func OpcodeNames(repository, root string) ([]NameRow, error) {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, filepath.Join(root, filepath.FromSlash(OpcodeTableFile)), nil, 0)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rows []NameRow
	ast.Inspect(parsed, func(node ast.Node) bool {
		spec, ok := node.(*ast.ValueSpec)
		if !ok || len(spec.Names) != 1 || spec.Names[0].Name != "OpcodeLifecycleTable" || len(spec.Values) != 1 {
			return true
		}
		table, ok := spec.Values[0].(*ast.CompositeLit)
		if !ok {
			return false
		}
		for _, element := range table.Elts {
			entry, ok := element.(*ast.CompositeLit)
			if !ok || len(entry.Elts) < 4 {
				continue
			}
			literal, ok := entry.Elts[1].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				continue
			}
			name, err := strconv.Unquote(literal.Value)
			if err != nil {
				continue
			}
			namespace := "builtin"
			if dot := strings.LastIndex(name, "."); dot > 0 {
				namespace = name[:dot]
			}
			kind := "opcode"
			if deprecated, ok := entry.Elts[3].(*ast.BasicLit); !ok || deprecated.Value != "0" {
				kind = "opcode_deprecated"
			}
			position := fileSet.Position(literal.Pos())
			rows = append(rows, NameRow{Repository: repository, Language: "cx", Package: namespace, File: OpcodeTableFile, Line: position.Line, Column: position.Column, Name: name, Kind: kind, Exported: true})
		}
		return false
	})
	return rows, nil
}
