// Command strhot finds the string operations reachable from the run loop of
// the old executor and checks that each one has a STRHOT code (#412). The
// run path is RunCxAst and every opcode handler, followed through static
// calls into this module. check-strhot.sh runs it, see docs/StringsHotPath.md.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const modulePath = "github.com/skycoin/skycoin-cx"

// runRoot is the run loop. Every function with the OpcodeHandler signature
// is a root as well.
const runRoot = modulePath + "/t4-execute.RunCxAst"

// listedPackage is the part of `go list -json` the tool reads.
type listedPackage struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	Export     string
	Module     *struct{ Path string }
}

// site is one string operation on one line.
type site struct {
	file     string
	line     int
	function string
	category string
	class    string
	code     string
	owner    string
	// commentClass is the class the VIOLATION comment names.
	commentClass string
}

// function is a declared function of the module with its calls and sites.
type function struct {
	name    string
	calls   []string
	sites   []site
	handler bool
	parent  string // the caller it was first reached from
}

var codePattern = regexp.MustCompile(`VIOLATION (STRHOT-[0-9]{3}): ([A-Z]?)`)

func main() {
	tags := flag.String("tags", "", "build tags")
	list := flag.Bool("l", false, "list every site with its code")
	inventory := flag.Bool("inventory", false, "list every site without a code, with a suggested class")
	codesPath := flag.String("codes", "strhot-codes.txt", "the STRHOT code table")
	why := flag.Bool("why", false, "with -inventory, add the call chain from a root to each function")
	flag.Parse()
	functions, err := load(*tags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	codes, err := readCodes(*codesPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	mode := reportCheck
	switch {
	case *inventory && *why:
		mode = reportInventoryWhy
	case *inventory:
		mode = reportInventory
	case *list:
		mode = reportList
	}
	os.Exit(report(os.Stdout, functions, codes, mode))
}

// codeEntry is one line of the code table.
type codeEntry struct {
	class   string
	removed bool
	cxfx    bool
}

// readCodes reads the code table, one "STRHOT-nnn CLASS STATUS file function"
// line per code. STATUS is open or removed. # starts a comment line.
func readCodes(path string) (map[string]codeEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	codes := map[string]codeEntry{}
	scanner := bufio.NewScanner(f)
	for n := 1; scanner.Scan(); n++ {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		if len(fields) < 5 || !strings.Contains("IOPEB", fields[1]) || len(fields[1]) != 1 || (fields[2] != "open" && fields[2] != "removed" && fields[2] != "cxfx") {
			return nil, fmt.Errorf("%s:%d: want STRHOT-nnn CLASS open|removed|cxfx file function", path, n)
		}
		if _, dup := codes[fields[0]]; dup {
			return nil, fmt.Errorf("%s:%d: %s listed twice", path, n, fields[0])
		}
		codes[fields[0]] = codeEntry{class: fields[1], removed: fields[2] == "removed", cxfx: fields[2] == "cxfx"}
	}
	return codes, scanner.Err()
}

// load type checks the module packages of the cx build and returns its
// functions by full name.
func load(tags string) (map[string]*function, error) {
	cmd := exec.Command("go", "list", "-deps", "-export", "-json", "-tags", tags, "./t11-cx")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}
	exports := map[string]string{}
	var own []listedPackage
	decoder := json.NewDecoder(bytes.NewReader(out))
	for {
		var p listedPackage
		if err := decoder.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		exports[p.ImportPath] = p.Export
		if p.Module != nil && p.Module.Path == modulePath && strings.HasPrefix(p.ImportPath, modulePath+"/") {
			own = append(own, p)
		}
	}
	fset := token.NewFileSet()
	imp := importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
		if exports[path] == "" {
			return nil, fmt.Errorf("no export data for %s", path)
		}
		return os.Open(exports[path])
	})
	wd, _ := os.Getwd()
	functions := map[string]*function{}
	for _, p := range own {
		var files []*ast.File
		for _, name := range p.GoFiles {
			f, err := parser.ParseFile(fset, filepath.Join(p.Dir, name), nil, parser.ParseComments)
			if err != nil {
				return nil, err
			}
			files = append(files, f)
		}
		info := &types.Info{
			Types:      map[ast.Expr]types.TypeAndValue{},
			Defs:       map[*ast.Ident]types.Object{},
			Uses:       map[*ast.Ident]types.Object{},
			Selections: map[*ast.SelectorExpr]*types.Selection{},
		}
		config := types.Config{Importer: imp}
		if _, err := config.Check(p.ImportPath, fset, files, info); err != nil {
			return nil, fmt.Errorf("type check %s: %w", p.ImportPath, err)
		}
		for _, f := range files {
			rel, _ := filepath.Rel(wd, fset.Position(f.Pos()).Filename)
			scanFile(fset, info, f, filepath.ToSlash(rel), functions)
		}
	}
	return functions, nil
}

// scanFile adds the functions of f with their calls and string sites.
func scanFile(fset *token.FileSet, info *types.Info, f *ast.File, rel string, functions map[string]*function) {
	comments := map[int]string{}
	for _, group := range f.Comments {
		comments[fset.Position(group.End()).Line] = group.Text()
	}
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		obj, ok := info.Defs[fd.Name].(*types.Func)
		if !ok {
			continue
		}
		fn := &function{name: obj.FullName(), handler: handlerSignature(obj.Type().(*types.Signature))}
		functions[fn.name] = fn
		funcCode, funcClass := "", ""
		if fd.Doc != nil {
			funcCode, funcClass = firstCode(fd.Doc.Text())
		}
		short := shortName(obj)
		var stack []ast.Node
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			stack = append(stack, n)
			if call, ok := n.(*ast.CallExpr); ok {
				if callee := calleeOf(info, call); callee != nil {
					fn.calls = append(fn.calls, callee.FullName())
				}
			}
			category := stringOperation(info, n)
			if category == "" {
				return true
			}
			line := fset.Position(n.Pos()).Line
			stmtLine := line
			for i := len(stack) - 1; i >= 0; i-- {
				if _, ok := stack[i].(ast.Stmt); ok {
					stmtLine = fset.Position(stack[i].Pos()).Line
					break
				}
			}
			code, class := firstCode(comments[stmtLine-1])
			if code == "" {
				code, class = firstCode(comments[line-1])
			}
			if code == "" {
				code, class = funcCode, funcClass
			}
			fn.sites = append(fn.sites, site{owner: fn.name, file: rel, line: line, function: short, category: category, class: suggestClass(info, stack), code: code, commentClass: class})
			return true
		})
	}
}

// firstCode returns the first STRHOT code in text and the class written after it.
func firstCode(text string) (code string, class string) {
	if m := codePattern.FindStringSubmatch(text); m != nil {
		return m[1], m[2]
	}
	return "", ""
}

// shortName is the function name as the code table writes it.
func shortName(obj *types.Func) string {
	recv := obj.Type().(*types.Signature).Recv()
	if recv == nil {
		return obj.Name()
	}
	return types.TypeString(recv.Type(), func(*types.Package) string { return "" }) + "." + obj.Name()
}

// handlerSignature reports func(*CXProgram, []CXValue, []CXValue), the
// OpcodeHandler type.
func handlerSignature(sig *types.Signature) bool {
	if sig.Recv() != nil || sig.Results().Len() != 0 || sig.Params().Len() != 3 {
		return false
	}
	want := []string{"*" + modulePath + "/t3-ast.CXProgram", "[]" + modulePath + "/t3-ast.CXValue", "[]" + modulePath + "/t3-ast.CXValue"}
	for i := range want {
		if sig.Params().At(i).Type().String() != want[i] {
			return false
		}
	}
	return true
}

// calleeOf returns the function a call statically calls, or nil.
func calleeOf(info *types.Info, call *ast.CallExpr) *types.Func {
	fun := ast.Unparen(call.Fun)
	if index, ok := fun.(*ast.IndexExpr); ok {
		fun = index.X
	}
	var obj types.Object
	switch f := fun.(type) {
	case *ast.Ident:
		obj = info.Uses[f]
	case *ast.SelectorExpr:
		if sel, ok := info.Selections[f]; ok {
			if sel.Kind() != types.MethodVal || types.IsInterface(sel.Recv()) {
				return nil
			}
			obj = sel.Obj()
		} else {
			obj = info.Uses[f.Sel]
		}
	}
	if fn, ok := obj.(*types.Func); ok {
		return fn.Origin()
	}
	return nil
}

// involvesString reports a string, or a slice, array, map or pointer of one.
func involvesString(t types.Type) bool {
	switch u := t.Underlying().(type) {
	case *types.Basic:
		return u.Info()&types.IsString != 0
	case *types.Slice:
		return involvesString(u.Elem())
	case *types.Array:
		return involvesString(u.Elem())
	case *types.Pointer:
		return involvesString(u.Elem())
	case *types.Map:
		return involvesString(u.Key()) || involvesString(u.Elem())
	}
	return false
}

func isString(t types.Type) bool {
	if t == nil {
		return false
	}
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&types.IsString != 0
}

func constant(info *types.Info, e ast.Expr) bool {
	return info.Types[e].Value != nil
}

// stringOperation returns the category of the string operation n is, or "".
func stringOperation(info *types.Info, n ast.Node) string {
	switch e := n.(type) {
	case *ast.BinaryExpr:
		if constant(info, e) || !isString(info.TypeOf(e.X)) {
			return ""
		}
		if e.Op == token.ADD {
			return "concat"
		}
		return "compare"
	case *ast.SwitchStmt:
		if e.Tag != nil && isString(info.TypeOf(e.Tag)) {
			return "compare"
		}
	case *ast.IndexExpr:
		if m, ok := info.TypeOf(e.X).Underlying().(*types.Map); ok && involvesString(m.Key()) {
			return "maplookup"
		}
		if isString(info.TypeOf(e.X)) {
			return "index"
		}
	case *ast.SliceExpr:
		if isString(info.TypeOf(e.X)) {
			return "slice"
		}
	case *ast.RangeStmt:
		if isString(info.TypeOf(e.X)) {
			return "range"
		}
	case *ast.CallExpr:
		if constant(info, e) {
			return ""
		}
		if tv := info.Types[e.Fun]; tv.IsType() {
			if len(e.Args) == 1 && (isString(tv.Type) || isString(info.TypeOf(e.Args[0]))) && !constant(info, e.Args[0]) {
				return "conv"
			}
			return ""
		}
		if callee := calleeOf(info, e); callee != nil && callee.Pkg() != nil {
			switch callee.Pkg().Path() {
			case "fmt", "strings", "errors", "log":
				return callee.Pkg().Path()
			}
		}
		sig, ok := info.TypeOf(e.Fun).Underlying().(*types.Signature)
		if !ok {
			return ""
		}
		for i := 0; i < sig.Results().Len(); i++ {
			if involvesString(sig.Results().At(i).Type()) {
				return "strresult"
			}
		}
		_, builtin := info.Uses[identOf(e.Fun)].(*types.Builtin)
		for i, arg := range e.Args {
			if builtin && constant(info, arg) {
				continue
			}
			if involvesString(info.TypeOf(arg)) {
				return "strarg"
			}
			if i < sig.Params().Len() && !sig.Variadic() && involvesString(sig.Params().At(i).Type()) {
				return "strarg"
			}
		}
	}
	return ""
}

// suggestClass guesses E for a site inside a panic, assert or error
// argument, or in a block that panics, and O otherwise.
func suggestClass(info *types.Info, stack []ast.Node) string {
	for i := len(stack) - 1; i >= 0; i-- {
		switch n := stack[i].(type) {
		case *ast.CallExpr:
			name := ""
			switch f := ast.Unparen(n.Fun).(type) {
			case *ast.Ident:
				name = f.Name
			case *ast.SelectorExpr:
				name = f.Sel.Name
			}
			if name == "panic" || name == "Errorf" || name == "New" || strings.HasPrefix(name, "Assert") || name == "Exit" {
				return "E"
			}
		case *ast.BlockStmt:
			for _, stmt := range n.List {
				if failing(info, stmt) {
					return "E"
				}
			}
			return "O"
		case *ast.CaseClause:
			for _, stmt := range n.Body {
				if failing(info, stmt) {
					return "E"
				}
			}
			return "O"
		}
	}
	return "O"
}

// failing reports a panic, os.Exit or a return of a non-nil error.
func failing(info *types.Info, stmt ast.Stmt) bool {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		if call, ok := s.X.(*ast.CallExpr); ok {
			switch f := ast.Unparen(call.Fun).(type) {
			case *ast.Ident:
				return f.Name == "panic"
			case *ast.SelectorExpr:
				return f.Sel.Name == "Exit"
			}
		}
	case *ast.ReturnStmt:
		for _, r := range s.Results {
			if t := info.TypeOf(r); t != nil && t.String() == "error" {
				if id, ok := r.(*ast.Ident); !ok || id.Name != "nil" {
					return true
				}
			}
		}
	}
	return false
}

// reachable walks the static calls from the run loop and the handlers.
func reachable(functions map[string]*function) []*function {
	seen := map[string]bool{}
	var queue []string
	from := ""
	add := func(name string) {
		if !seen[name] && functions[name] != nil {
			seen[name] = true
			functions[name].parent = from
			queue = append(queue, name)
		}
	}
	add(runRoot)
	for name, fn := range functions {
		if fn.handler {
			add(name)
		}
	}
	var out []*function
	for len(queue) > 0 {
		fn := functions[queue[0]]
		queue = queue[1:]
		out = append(out, fn)
		from = fn.name
		for _, callee := range fn.calls {
			add(callee)
		}
	}
	return out
}

// report prints the sites without a code and the code table mismatches, and
// returns the exit status.
// reportMode is what report prints.
type reportMode int

const (
	reportCheck reportMode = iota
	reportList
	reportInventory
	reportInventoryWhy
)

func report(w io.Writer, functions map[string]*function, codes map[string]codeEntry, mode reportMode) int {
	var sites []site
	for _, fn := range reachable(functions) {
		sites = append(sites, fn.sites...)
	}
	sort.Slice(sites, func(i, j int) bool {
		if sites[i].file != sites[j].file {
			return sites[i].file < sites[j].file
		}
		return sites[i].line < sites[j].line
	})
	rc := 0
	used := map[string]bool{}
	lines := map[string]bool{}
	for _, s := range sites {
		key := fmt.Sprintf("%s:%d", s.file, s.line)
		if lines[key] {
			continue
		}
		lines[key] = true
		switch {
		case mode >= reportInventory && s.code == "":
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.class, key, s.function, s.category)
			if mode == reportInventoryWhy {
				fmt.Fprintf(w, "\t\t%s\n", chain(functions, s.owner))
			}
		case mode >= reportInventory:
		case mode == reportList:
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.code, key, s.function, s.category)
		case s.code == "":
			fmt.Fprintf(w, "%s: %s: %s on the run path without a STRHOT code\n", key, s.function, s.category)
			rc = 1
		case codes[s.code].class == "":
			fmt.Fprintf(w, "%s: %s is not in strhot-codes.txt\n", key, s.code)
			rc = 1
		case codes[s.code].class != s.commentClass:
			fmt.Fprintf(w, "%s: %s says class %q, strhot-codes.txt %s\n", key, s.code, s.commentClass, codes[s.code].class)
			rc = 1
		case codes[s.code].removed:
			fmt.Fprintf(w, "%s: %s is marked removed in strhot-codes.txt\n", key, s.code)
			rc = 1
		}
		used[s.code] = true
	}
	if mode != reportCheck {
		return 0
	}
	var stale []string
	for code, entry := range codes {
		if !entry.removed && !entry.cxfx && !used[code] {
			stale = append(stale, code)
		}
	}
	sort.Strings(stale)
	for _, code := range stale {
		fmt.Fprintf(w, "%s has no site on the run path, mark it removed in strhot-codes.txt\n", code)
		rc = 1
	}
	counts := map[string]int{}
	for _, entry := range codes {
		if !entry.removed && !entry.cxfx {
			counts[entry.class]++
		}
	}
	fmt.Fprintf(w, "%d string operation lines on the run path, open codes I %d, O %d, P %d, E %d, B %d\n",
		len(lines), counts["I"], counts["O"], counts["P"], counts["E"], counts["B"])
	return rc
}

// chain is the call chain from a root to name.
func chain(functions map[string]*function, name string) string {
	var names []string
	for name != "" && len(names) < 30 {
		names = append(names, strings.TrimPrefix(strings.ReplaceAll(name, modulePath+"/", ""), "("))
		name = functions[name].parent
	}
	return strings.Join(names, " < ")
}

// identOf returns the identifier a call expression names, or nil.
func identOf(fun ast.Expr) *ast.Ident {
	id, _ := ast.Unparen(fun).(*ast.Ident)
	return id
}
