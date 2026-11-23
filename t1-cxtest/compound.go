package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-cxtestrunner"
	"github.com/skycoin/skycoin-cx/t0-types"
)

// compoundType is a type made of pointers, slices, arrays and structs, a
// body that sets the result r of a function to a value of it, and a
// statement that prints the value v.
type compoundType struct {
	name  string
	value string
	show  string
	want  string
}

var compoundTypes = []compoundType{
	{"i32", "r = 7", `printf("%d\n", v)`, "7"},
	{"str", `r = "s"`, `printf("%s\n", v)`, "s"},
	{"*i32", "r = &ti", `printf("%d\n", *v)`, "7"},
	{"*str", "r = &ts", `printf("%s\n", *v)`, "s"},
	{"**i32", "r = &tpi", `printf("%d\n", **v)`, "7"},
	{"[]i32", "r = append(r, 7)", `printf("%d %d\n", len(v), v[0])`, "1 7"},
	{"[]str", `r = append(r, "s")`, `printf("%s\n", v[0])`, "s"},
	{"[]*i32", "r = append(r, &ti)", `printf("%d\n", *v[0])`, "7"},
	{"[2]i32", "r[1] = 7", `printf("%d %d\n", v[0], v[1])`, "0 7"},
	{"[2]*i32", "r[0] = &ti\n\tr[1] = &ti", `printf("%d\n", *v[1])`, "7"},
	{"*[]i32", "r = &tsl", `printf("%d\n", (*v)[0])`, "7"},
	{"*[]str", "r = &tssl", `printf("%s\n", (*v)[0])`, "s"},
	{"*[2]i32", "r = &ta", `printf("%d\n", (*v)[1])`, "7"},
	{"[2][]i32", "r[1] = append(r[1], 7)", `printf("%d\n", v[1][0])`, "7"},
	{"[][2]i32", "var e [2]i32\n\te[1] = 7\n\tr = append(r, e)", `printf("%d\n", v[0][1])`, "7"},
	{"[][]i32", "r = append(r, []i32{7})", `printf("%d\n", v[0][0])`, "7"},
	{"[2][3]i32", "r[1][2] = 7", `printf("%d\n", v[1][2])`, "7"},
	{"[2][]*i32", "r[1] = append(r[1], &ti)", `printf("%d\n", *v[1][0])`, "7"},
	{"Point", "r.x = 7", `printf("%d\n", v.x)`, "7"},
	{"*Point", "r = &tp", `printf("%d\n", v.x)`, "7"},
	{"[]Point", "r = append(r, tp)", `printf("%d\n", v[0].x)`, "7"},
	{"[]*Point", "r = append(r, &tp)", `printf("%d\n", v[0].x)`, "7"},
	{"[2]Point", "r[1].x = 7", `printf("%d\n", v[1].x)`, "7"},
	{"[2]*Point", "r[0] = &tp\n\tr[1] = &tp", `printf("%d\n", v[1].x)`, "7"},
	{"*[]Point", "r = &tps", `printf("%d\n", (*v)[0].x)`, "7"},
	{"*[2]Point", "r = &tpa", `printf("%d\n", (*v)[1].x)`, "7"},
	{"**Point", "r = &tpp", `printf("%d\n", (**v).x)`, "7"},
	{"[2]*[2]i32", "r[1] = &ta", `printf("%d\n", (*v[1])[1])`, "7"},
	{"[]*[2]i32", "r = append(r, &ta)", `printf("%d\n", (*v[0])[1])`, "7"},
	{"[2]*[]i32", "r[1] = &tsl", `printf("%d\n", (*v[1])[0])`, "7"},
	{"[]*[]i32", "r = append(r, &tsl)", `printf("%d\n", (*v[0])[0])`, "7"},
	{"*[]*i32", "if len(tspi) == 0 {\n\t\ttspi = append(tspi, &ti)\n\t}\n\tr = &tspi", `printf("%d\n", *(*v)[0])`, "7"},
	{"[][]Point", "r = append(r, tps)", `printf("%d\n", v[0][0].x)`, "7"},
	{"[2][]Point", "r[1] = tps", `printf("%d\n", v[1][0].x)`, "7"},
	{"*[2][]i32", "tas[1] = tsl\n\tr = &tas", `printf("%d\n", (*v)[1][0])`, "7"},
	{"[]str", `r = append(r, "s")`, `printf("%s\n", v[0])`, "s"},
}

// compoundProgram is a program that passes, returns, declares, assigns,
// stores and prints values of typ.
func compoundProgram(typ compoundType, cases []string) string {
	return strings.NewReplacer("TYPE", typ.name, "VALUE", typ.value, "SHOW", typ.show).Replace(`package main

type Point struct {
	x i32
	y i32
}

type Holder struct {
	f TYPE
}

var ti i32 = 7
var ts str = "s"
var tsl []i32 = []i32{7}
var tssl []str = []str{"s"}
var ta [2]i32 = [2]i32{0, 7}
var tp Point = Point{x: 7}
var tps []Point = []Point{Point{x: 7}}
var tpa [2]Point = [2]Point{Point{x: 1}, Point{x: 7}}
var tpi *i32 = &ti
var tpp *Point = &tp
var tspi []*i32
var tas [2][]i32

func value() (r TYPE) {
	VALUE
	return r
}

func identity(p TYPE) (r TYPE) {
	r = p
	return r
}

func show(v TYPE) {
	SHOW
}

var global TYPE
var globalTyped TYPE = value()
var globalInferred = value()
var globalCopy = globalTyped

func main() {
` + strings.Join(cases, "\n") + `}
`)
}

var compoundCases = []string{
	"\tshow(value())\n",
	"\tvar typed TYPE = value()\n\tshow(typed)\n",
	"\tshort := value()\n\tshow(short)\n",
	"\tvar inferred = value()\n\tshow(inferred)\n",
	"\tvar assigned TYPE\n\tassigned = identity(value())\n\tshow(assigned)\n",
	"\tvar declared TYPE = value()\n\tvar fromVar = declared\n\tshow(fromVar)\n",
	"\tvar source TYPE = value()\n\tcopied := source\n\tshow(copied)\n",
	"\tglobal = value()\n\tshow(global)\n",
	"\tshow(globalTyped)\n",
	"\tshow(globalInferred)\n",
	"\tshow(globalCopy)\n",
	"\tvar h Holder\n\th.f = value()\n\tshow(h.f)\n",
	"\tvar elements [2]TYPE\n\telements[1] = value()\n\tshow(elements[1])\n",
	"\tvar items []TYPE\n\titems = append(items, value())\n\tshow(items[0])\n",
}

// runCompoundTypeTests runs the program of each compound type on the loop
// cx picks and on the old loop.
func runCompoundTypeTests(t *runner.TestRunner, workingDir string) {
	dir, err := os.MkdirTemp(workingDir, "compound-types-")
	if err != nil {
		fmt.Printf("compound types: %v\n", err)
		t.TestCount++
		return
	}
	defer os.RemoveAll(dir)

	for index, typ := range compoundTypes {
		name := fmt.Sprintf("compound-%02d.cx", index)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(compoundProgram(typ, compoundCases)), 0o644); err != nil {
			fmt.Printf("compound types: %v\n", err)
			t.TestCount++
			continue
		}
		want := strings.Repeat(typ.want+"\n", len(compoundCases))
		path := filepath.Base(dir) + "/" + name
		for _, executor := range []string{"auto", "old"} {
			t.RunWant("--executor="+executor+" "+path, types.CX_SUCCESS, fmt.Sprintf("values of %s as parameters, results, locals, globals, fields and elements, on %s", typ.name, executor), want)
		}
	}
}
