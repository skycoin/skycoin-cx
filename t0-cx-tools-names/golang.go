package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// GoCollector gathers the names of the Go files of one directory.
type GoCollector struct {
	Repository string
	FileSet    *token.FileSet
	File       string
	Package    string
	TestFile   bool
	Rows       []NameRow
	Seen       map[*ast.Ident]bool
	// EnumTypes holds the local types that typed constants are declared with.
	EnumTypes map[string]bool
}

// GoDirectoryNames parses the Go files of one directory. A directory's
// package row is emitted once, at the package clause of its first file.
func GoDirectoryNames(repository, root string, files []string) ([]NameRow, error) {
	sort.Strings(files)
	collector := &GoCollector{Repository: repository, FileSet: token.NewFileSet(), Seen: map[*ast.Ident]bool{}, EnumTypes: map[string]bool{}}
	packages := map[string]bool{}
	for _, file := range files {
		parsed, err := parser.ParseFile(collector.FileSet, filepath.Join(root, filepath.FromSlash(file)), nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		if ast.IsGenerated(parsed) {
			continue
		}
		collector.File = file
		collector.Package = parsed.Name.Name
		collector.TestFile = strings.HasSuffix(file, "_test.go")
		if !packages[parsed.Name.Name] {
			packages[parsed.Name.Name] = true
			collector.Add(parsed.Name, "package", "", false)
		}
		collector.FileNames(parsed)
	}
	for i := range collector.Rows {
		row := &collector.Rows[i]
		if row.Kind == "type" && row.Parent == "" && collector.EnumTypes[row.Name] {
			row.Kind = "enum"
		}
	}
	return collector.Rows, nil
}

// Add records one identifier, once. The blank identifier is skipped.
func (c *GoCollector) Add(ident *ast.Ident, kind, parent string, exported bool) {
	if ident == nil || ident.Name == "_" || c.Seen[ident] {
		return
	}
	c.Seen[ident] = true
	c.AddAt(ident.Pos(), ident.Name, kind, parent, exported && ast.IsExported(ident.Name))
}

// AddAt records a name that is not an identifier, such as an import path.
func (c *GoCollector) AddAt(position token.Pos, name, kind, parent string, exported bool) {
	resolved := c.FileSet.Position(position)
	c.Rows = append(c.Rows, NameRow{Repository: c.Repository, Language: "go", Package: c.Package, File: c.File, Line: resolved.Line, Column: resolved.Column, Name: name, Kind: kind, Parent: parent, Exported: exported, TestFile: c.TestFile})
}

// FileNames records the imports and declarations of one file.
func (c *GoCollector) FileNames(file *ast.File) {
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			path = spec.Path.Value
		}
		c.AddAt(spec.Path.Pos(), path, "import", "", false)
		if spec.Name != nil && spec.Name.Name != "." {
			c.Add(spec.Name, "import_alias", "", false)
		}
	}
	for _, declaration := range file.Decls {
		switch typed := declaration.(type) {
		case *ast.GenDecl:
			c.GenDeclNames(typed, "")
			for _, spec := range typed.Specs {
				if value, ok := spec.(*ast.ValueSpec); ok {
					for _, expression := range value.Values {
						c.BodyNames(expression, value.Names[0].Name)
					}
				}
			}
		case *ast.FuncDecl:
			c.FuncDeclNames(typed)
		}
	}
}

// GenDeclNames records a type, const or var declaration. parent is the
// enclosing function, empty at package level.
func (c *GoCollector) GenDeclNames(declaration *ast.GenDecl, parent string) {
	topLevel := parent == ""
	switch declaration.Tok {
	case token.TYPE:
		for _, spec := range declaration.Specs {
			c.TypeSpecNames(spec.(*ast.TypeSpec), parent)
		}
	case token.CONST:
		var inheritedType ast.Expr
		inheritedIota := false
		for _, spec := range declaration.Specs {
			value := spec.(*ast.ValueSpec)
			if value.Type != nil || len(value.Values) > 0 {
				inheritedType = value.Type
				inheritedIota = ExprUsesIota(value.Values)
			}
			kind := "constant"
			if typeIdent, local := inheritedType.(*ast.Ident); local && types.Universe.Lookup(typeIdent.Name) == nil {
				c.EnumTypes[typeIdent.Name] = true
				kind = "enum_value"
			} else if inheritedIota {
				kind = "enum_value"
			}
			if !topLevel {
				kind = "local_" + kind
			}
			for _, name := range value.Names {
				c.Add(name, kind, parent, topLevel)
			}
		}
	case token.VAR:
		for _, spec := range declaration.Specs {
			value := spec.(*ast.ValueSpec)
			for i, name := range value.Names {
				kind := "local_variable"
				if topLevel {
					kind = "global_variable"
				}
				if FuncTyped(value.Type) || (i < len(value.Values) && FuncTyped(value.Values[i])) {
					kind = "function_variable"
				}
				c.Add(name, kind, parent, topLevel)
			}
		}
	}
}

// TypeSpecNames records a type, its type parameters and its members.
func (c *GoCollector) TypeSpecNames(spec *ast.TypeSpec, parent string) {
	topLevel := parent == ""
	kind := "type"
	switch spec.Type.(type) {
	case *ast.StructType:
		kind = "struct"
	case *ast.InterfaceType:
		kind = "interface"
	case *ast.FuncType:
		kind = "function_type"
	}
	if spec.Assign.IsValid() {
		kind = "type_alias"
	}
	c.Add(spec.Name, kind, parent, topLevel)
	owner := spec.Name.Name
	if parent != "" {
		owner = parent + "." + owner
	}
	c.FieldListNames(spec.TypeParams, "type_parameter", owner)
	c.TypeExprNames(spec.Type, owner, topLevel)
}

// TypeExprNames records the fields and methods of struct and interface
// types, including anonymous ones nested in other types.
func (c *GoCollector) TypeExprNames(expression ast.Expr, owner string, exported bool) {
	ast.Inspect(expression, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.StructType:
			for _, field := range typed.Fields.List {
				kind := "field"
				if FuncTyped(field.Type) {
					kind = "function_field"
				}
				if len(field.Names) == 0 {
					c.Add(EmbeddedIdent(field.Type), "embedded_field", owner, exported)
				}
				for _, name := range field.Names {
					c.Add(name, kind, owner, exported)
					c.TypeExprNames(field.Type, owner+"."+name.Name, exported)
				}
			}
			return false
		case *ast.InterfaceType:
			for _, method := range typed.Methods.List {
				if len(method.Names) == 0 {
					c.Add(EmbeddedIdent(method.Type), "embedded_interface", owner, exported)
				}
				for _, name := range method.Names {
					c.Add(name, "interface_method", owner, exported)
				}
			}
			return false
		case *ast.FuncType:
			return false
		}
		return true
	})
}

// FuncDeclNames records a function or method and everything declared in it.
func (c *GoCollector) FuncDeclNames(declaration *ast.FuncDecl) {
	owner := declaration.Name.Name
	if declaration.Recv != nil && len(declaration.Recv.List) > 0 {
		receiverType := ReceiverTypeName(declaration.Recv.List[0].Type)
		c.Add(declaration.Name, "method", receiverType, true)
		owner = receiverType + "." + owner
		c.FieldListNames(declaration.Recv, "receiver", owner)
	} else {
		c.Add(declaration.Name, "function", "", true)
	}
	c.FuncTypeNames(declaration.Type, owner)
	if declaration.Body != nil {
		c.BodyNames(declaration.Body, owner)
	}
}

// FuncTypeNames records type parameters, parameters and named results.
func (c *GoCollector) FuncTypeNames(funcType *ast.FuncType, owner string) {
	c.FieldListNames(funcType.TypeParams, "type_parameter", owner)
	c.FieldListNames(funcType.Params, "parameter", owner)
	c.FieldListNames(funcType.Results, "result", owner)
}

// FieldListNames records every name of a field list as kind.
func (c *GoCollector) FieldListNames(list *ast.FieldList, kind, owner string) {
	if list == nil {
		return
	}
	for _, field := range list.List {
		for _, name := range field.Names {
			c.Add(name, kind, owner, false)
		}
	}
}

// BodyNames records the locals, labels, closures and local types of a body.
func (c *GoCollector) BodyNames(body ast.Node, owner string) {
	ast.Inspect(body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			if typed.Tok != token.DEFINE {
				return true
			}
			for i, expression := range typed.Lhs {
				ident, ok := expression.(*ast.Ident)
				if !ok || ident.Obj == nil || ident.Obj.Decl != typed {
					continue
				}
				kind := "local_variable"
				if i < len(typed.Rhs) && len(typed.Lhs) == len(typed.Rhs) && FuncTyped(typed.Rhs[i]) {
					kind = "function_variable"
				}
				c.Add(ident, kind, owner, false)
			}
		case *ast.ForStmt:
			if assign, ok := typed.Init.(*ast.AssignStmt); ok && assign.Tok == token.DEFINE {
				for _, expression := range assign.Lhs {
					if ident, ok := expression.(*ast.Ident); ok {
						c.Add(ident, "loop_variable", owner, false)
					}
				}
			}
		case *ast.RangeStmt:
			if typed.Tok == token.DEFINE {
				for _, expression := range []ast.Expr{typed.Key, typed.Value} {
					if ident, ok := expression.(*ast.Ident); ok {
						c.Add(ident, "loop_variable", owner, false)
					}
				}
			}
		case *ast.TypeSwitchStmt:
			if assign, ok := typed.Assign.(*ast.AssignStmt); ok && len(assign.Lhs) == 1 {
				if ident, ok := assign.Lhs[0].(*ast.Ident); ok {
					c.Add(ident, "local_variable", owner, false)
				}
			}
		case *ast.DeclStmt:
			if declaration, ok := typed.Decl.(*ast.GenDecl); ok {
				c.GenDeclNames(declaration, owner)
			}
			return true
		case *ast.LabeledStmt:
			c.Add(typed.Label, "label", owner, false)
		case *ast.FuncLit:
			c.FuncTypeNames(typed.Type, owner)
		}
		return true
	})
}

// ReceiverTypeName returns the base type name of a method receiver.
func ReceiverTypeName(expression ast.Expr) string {
	for {
		switch typed := expression.(type) {
		case *ast.StarExpr:
			expression = typed.X
		case *ast.IndexExpr:
			expression = typed.X
		case *ast.IndexListExpr:
			expression = typed.X
		case *ast.ParenExpr:
			expression = typed.X
		case *ast.Ident:
			return typed.Name
		default:
			return ""
		}
	}
}

// EmbeddedIdent returns the type name identifier of an embedded field.
func EmbeddedIdent(expression ast.Expr) *ast.Ident {
	for {
		switch typed := expression.(type) {
		case *ast.StarExpr:
			expression = typed.X
		case *ast.SelectorExpr:
			return typed.Sel
		case *ast.IndexExpr:
			expression = typed.X
		case *ast.IndexListExpr:
			expression = typed.X
		case *ast.Ident:
			return typed
		default:
			return nil
		}
	}
}

// FuncTyped reports whether an expression is a func type or a func literal.
func FuncTyped(expression ast.Expr) bool {
	switch expression.(type) {
	case *ast.FuncType, *ast.FuncLit:
		return true
	}
	return false
}

// ExprUsesIota reports whether any of the expressions mentions iota.
func ExprUsesIota(expressions []ast.Expr) bool {
	found := false
	for _, expression := range expressions {
		ast.Inspect(expression, func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok && ident.Name == "iota" {
				found = true
			}
			return !found
		})
	}
	return found
}
