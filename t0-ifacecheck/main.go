// Command ifacecheck prints one line per interface that embeds another
// interface, as "<file> <interface> <embedded>". check-ifaces.sh compares the
// output with iface-allowlist.txt.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	var out []string
	err := filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (p == "vendor" || p == ".git" || p == ".claude") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		lines, err := embeds(filepath.ToSlash(p), src)
		out = append(out, lines...)
		return err
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sort.Strings(out)
	for _, l := range out {
		fmt.Println(l)
	}
}

// embeds lists the embedded interfaces of every interface type in src. Type
// set terms such as ~int or int | string are not interfaces and are skipped.
func embeds(name string, src []byte) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var out []string
	owner := "anonymous"
	ast.Inspect(f, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.TypeSpec:
			owner = t.Name.Name
		case *ast.FuncDecl:
			owner = "anonymous"
		case *ast.InterfaceType:
			for _, m := range t.Methods.List {
				if len(m.Names) > 0 || !named(m.Type) {
					continue
				}
				var b strings.Builder
				printer.Fprint(&b, fset, m.Type)
				out = append(out, name+" "+owner+" "+b.String())
			}
		}
		return true
	})
	return out, nil
}

func named(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.Ident:
		return !basic[t.Name]
	case *ast.SelectorExpr, *ast.InterfaceType:
		return true
	case *ast.IndexExpr:
		return named(t.X)
	case *ast.IndexListExpr:
		return named(t.X)
	}
	return false
}

var basic = map[string]bool{
	"bool": true, "byte": true, "rune": true, "string": true, "uintptr": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true, "complex64": true, "complex128": true,
}
