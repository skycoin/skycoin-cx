package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// NameRowsText renders rows as "kind parent name exported" lines.
func NameRowsText(rows []NameRow) string {
	var builder strings.Builder
	for _, row := range rows {
		builder.WriteString(strings.Join([]string{row.Kind, row.Parent, row.Name, YesNo(row.Exported)}, " "))
		builder.WriteString("\n")
	}
	return builder.String()
}

func TestCXFileNames(test *testing.T) {
	source := `package demo

import "os"

type Point struct {
	x, y i32
	label str
}

var origin Point
const Limit i32 = 3

func (p *Point) Move(dx i32, dy i32) (moved Point) {
	total := 0
	for i := 0; i < 3; i++ {
		total = total + i
	}
	q := Point{x: 1,
		y: 2}
	again:
	var callback func(i32)
	run := func(step i32) { }
}

func Sum(a, b i32) i32 { var s i32; return a + b }
`
	want := `package  demo no
import  os no
struct  Point yes
field Point x yes
field Point y yes
field Point label yes
global_variable  origin yes
constant  Limit yes
method Point Move yes
receiver Point.Move p no
parameter Point.Move dx no
parameter Point.Move dy no
result Point.Move moved no
local_variable Point.Move total no
loop_variable Point.Move i no
local_variable Point.Move q no
label Point.Move again no
function_variable Point.Move callback no
function_variable Point.Move run no
parameter Point.Move step no
function  Sum yes
parameter Sum a no
parameter Sum b no
local_variable Sum s no
`
	got := NameRowsText(CXFileNames("repo", "demo.cx", []byte(source)))
	if got != want {
		test.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestGoFileNames(test *testing.T) {
	source := `package demo

import alias "strings"

type Color int

const (
	ColorRed Color = iota
	ColorBlue
)

type Shape interface{ Area() float64 }

type Box struct {
	Shape
	Width  float64
	render func()
}

var handler = func(input int) { doubled := input * 2; _ = doubled }

func (b Box) Area() (area float64) {
	for index, value := range []int{1} {
		_, _ = index, value
	}
	return b.Width
}
`
	collector := &GoCollector{Repository: "repo", FileSet: token.NewFileSet(), Seen: map[*ast.Ident]bool{}, EnumTypes: map[string]bool{}, File: "demo.go", Package: "demo"}
	file, err := parser.ParseFile(collector.FileSet, "demo.go", source, parser.ParseComments)
	if err != nil {
		test.Fatal(err)
	}
	collector.FileNames(file)
	want := `import  strings no
import_alias  alias no
type  Color yes
enum_value  ColorRed yes
enum_value  ColorBlue yes
interface  Shape yes
interface_method Shape Area yes
struct  Box yes
embedded_field Box Shape yes
field Box Width yes
function_field Box render no
function_variable  handler no
parameter handler input no
local_variable handler doubled no
method Box Area yes
receiver Box.Area b no
result Box.Area area no
loop_variable Box.Area index no
loop_variable Box.Area value no
`
	if got := NameRowsText(collector.Rows); got != want {
		test.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if !collector.EnumTypes["Color"] {
		test.Error("Color is not recorded as an enumeration type")
	}
}

func TestNameRowsWrite(test *testing.T) {
	var output bytes.Buffer
	rows := []NameRow{{Repository: "repo", Language: "go", Package: "demo", File: "a,b.go", Line: 1, Column: 2, Name: "x", Kind: "field", Parent: `say "hi"`}}
	if err := NameRowsWrite(&output, rows); err != nil {
		test.Fatal(err)
	}
	want := "repository,language,module_or_package,file,line,column,name,kind,parent,exported,test_file\r\nrepo,go,demo,\"a,b.go\",1,2,x,field,\"say \"\"hi\"\"\",no,no\r\n"
	if output.String() != want {
		test.Errorf("got %q, want %q", output.String(), want)
	}
}
