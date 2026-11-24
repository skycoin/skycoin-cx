# Table of Contents

* [Syntax](#syntax)
  * [Comments](#comments)
  * [Semicolons](#semicolons)
  * [Declarations](#declarations)
    * [Allowed Names](#allowed-names)
    * [Strict Type System](#strict-type-system)
    * [Primitive Types](#primitive-types)
    * [Global variables](#global-variables)
    * [Local variables](#local-variables)
    * [Scopes and shadowing](#scopes-and-shadowing)
    * [Arrays](#arrays)
    * [Slices](#slices)
    * [Literals](#literals)
    * [Functions](#functions)
    * [Custom Types](#custom-types)
    * [Methods](#methods)
    * [Packages](#packages)
    * [Imports into scope](#imports-into-scope)
    * [System libraries](#system-libraries)
    * [The cipher library](#the-cipher-library)
  * [Statements](#statements)
    * [If and if/else](#if-and-ifelse)
    * [Else if](#else-if)
    * [For loop](#for-loop)
    * [Goto](#goto)
  * [Expressions](#expressions)
  * [Assignments and Initializations](#assignments-and-initializations)
  * [Affordances](#affordances)
* [Runtime](#runtime)
  * [Packages](#packages-1)
  * [Data Structures](#data-structures)
    * [Literals](#literals-1)
    * [Variables](#variables)
    * [Primitive types](#primitive-types-1)
    * [Arrays](#arrays-1)
    * [Slices](#slices-1)
    * [Structures](#structures)
    * [Pointers](#pointers)
    * [Escape Analysis](#escape-analysis)
  * [Control Flow](#control-flow)
    * [Functions](#functions-1)
    * [Methods](#methods-1)
    * [If and if/else](#if-and-ifelse-1)
    * [For loop](#for-loop-1)
    * [Go-to](#go-to)
  * [Affordances](#affordances-1)
* [Native Functions](#native-functions)
  * [Type-inferenced Functions](#type-inferenced-functions)
  * [Slice Functions](#slice-functions)
  * [Input/Output Functions](#inputoutput-functions)
  * [Parse Functions](#parse-functions)
  * [Unit Testing](#unit-testing)
  * [`bool` Type Functions](#bool-type-functions)
  * [`sys/string` Package Functions](#sysstring-package-functions)
  * [`i32` Type Functions](#i32-type-functions)
  * [`i64` Type Functions](#i64-type-functions)
  * [`f32` Type Functions](#f32-type-functions)
  * [`f64` Type Functions](#f64-type-functions)
  * [`time` Package Functions](#time-package-functions)
  * [`os` Package Functions](#os-package-functions)
  * [`gl` Package Functions](#gl-package-functions)
  * [`glfw` Package Functions](#glfw-package-functions)
  * [`gltext` Package Functions](#gltext-package-functions)
* [Building an executable](#building-an-executable)
* [Language versions](#language-versions)

# [Syntax](#table-of-contents)

In this section, we're going to have a look at how a CX program looks
like. Basically, the following sections are not going to discuss about
the logic behind the various CX constructs, i.e. how they
behave; we're only going to see how they look like.

## [Comments](#table-of-contents)

Some of the code snippets that follow have comments in them, i.e.,
blocks of text that are not actually "run" by the CX compiler or
interpreter. Just like in C, Golang and many other programming
languages, single line comments are created by placing double slashes
(//) before the text being commented. For example:

```cx
// Example of adding two 32 bit integers in CX

i32.print(i32.add(3, 4))       // This will be ignored

// End of the program
```

Mult-line comments are opened by writing slash-asterisk (/\*), and are
closed by writing asterisk-slash (\*/).

```cx
/* This code won't be executed
string.PrintLine("Hello world!")
*/
```

## [Semicolons](#table-of-contents)

As in Go, statements and struct fields are separated by semicolons, and
the lexer adds one at the end of a line that ends with a name, a
literal, `return`, `break`, `continue`, `++`, `--`, `)`, `]` or `}`. A
multi-line comment that holds a newline acts like a newline. The
semicolon before a closing `}` may be left out, so short declarations
fit on one line.

A list of elements, fields, arguments, parameters or results may end in a
comma. Go requires the comma when the closing brace or parenthesis is on
the next line, since the line would otherwise end with a semicolon.

```cx
package main

type Point struct { x i32; y i32 }

func add(a i32,
	b i32,
) i32 {
	return a + b
}

func main () {
	p := Point{
		x: 1,
		y: 2,
	}
	s := []i32{
		3,
		4,
	}
	i32.print(add(
		p.y,
		s[1],
	)) // prints 6
}
```

```cx
package main

type Point struct { x i32; y i32 }

func double (n i32) i32 { return n * 2 }

func sign (n i32) i32 { if n < 0 { return -1 } else { return 1 } }

func main () {
    i32.print(double(Point{x: 3}.x) + sign(-4)) // prints 5
}
```

## [Declarations](#table-of-contents)

A declaration refers to a *named* element in a program's
structure, which are described using other constructs, such as
expressions and other statements. For example: a function can be
referred by its name and it's constructed by expressions and local
variable declarations.

### [Allowed Names](#table-of-contents)

Any name that satisfies the PCRE regular expression
`[_a-zA-Z][_a-zA-Z0-9]*` is allowed as an identifier for a declared
element. In other words, an identifier can start with an underscore
(*_*) or any lowercase or uppercase letter, and can be followed by 0
or more underscores or lowercase or uppercase letters, and any number
from 0 to 9.

### [Strict Type System](#table-of-contents)

One of CX's goals is to provide a very strict type system. The purpose
of this is to reduce runtime errors as much as possible. In order to
achieve this goal, many of CX's native functions are constrained to a
single type signature. For example, if you want to add two 32-bit
integers, you'd need to use `i32.add`. In contrast, if you want to add
two 64-bit integers, you'd use `i64.add`. These functions can help the
programmer to ensure that a particular data type is being received or
sent during a process.

If the programmer doesn't want to use those type-specific functions,
CX still provides type inference in some cases. For example, instead
of writing `i32.add(5, 5)` you can just write `5 + 5`. In this case,
CX is going to see that you're using 32-bit integers, and the parser
is going to transform that expression to `i32.add(5, 5)`. However, if
you try to do `5 + 5L`, i.e. if you try to add a 32-bit integer to a
64-bit integer, CX will throw a compile-time error because you're
mixing types.

The proper way to handle types in CX is to explicitly parse
everything. This way one can be sure that you're always going to be
handling the desired type. So, retaking the previous example, you'd
need to parse one of them to match the other's type, either
`i32.i64(5) + 5L` or `5 + i64.i32(5L)`.

### [Primitive Types](#table-of-contents)

There are twelve primitive types in CX: *bool*, *str*, the signed
integers *i8*, *i16*, *i32* and *i64*, the unsigned integers *ui8*,
*ui16*, *ui32* and *ui64*, and *f32* and *f64*. Those represent
Booleans (*true* or *false*), character strings, integers of 8, 16, 32
and 64 bits, and single and double precision floating-point numbers.
There is no *byte* type any more. Use *ui8* or *i8*.

### [Global variables](#table-of-contents)

Global variables are different from local variables regarding
scope. Global variables are available to any function defined in a
package, and to any package that is importing the package that
contains that global declaration. A function names every global it
uses in a `global` statement at the top of its body, so a reader sees
which globals the function reaches. An example of some global
variables is shown below.

```cx
package main

var global1 i32
var global2 i64

func foo () {
    global global1, global2
    i32.print(global1)
    i64.print(global2)
}

func main () {
    global global1, global2
    global1 = 5
    global2 = 5L

    i32.print(global1)
    i64.print(global2)
}
```

In the example above we can see that both the `main` and `foo`
functions are printing the values of the two global variables
defined. They are going to print the same values, as they are
referring to the same variables.

The `global` statement lists names separated by commas. A global of the
function's own package is named alone, `global counter`, and a global of
an imported package as `package.name`, `global config.Limit`, which is
also how the body reaches it. The statements come first in the body,
before any other statement, and there may be several. The compiler
reports each of these as an error:

* a global the function uses without declaring it, `function 'f' uses
  global 'counter' without declaring it`, at the first use,
* a declared global the function does not use, `global 'counter'
  declared and not used in function 'f'`, as Go reports an unused
  import,
* a `global` statement after another statement or in an inner block,
* a name that is not a global of its package, a package that is not
  imported, and a global declared twice in one function.

The values of globals, `var total = count * 2`, are not functions and
declare nothing.

A package declares each global once. Declaring the same name twice is
the compile error `'name' redeclared`, also when the declarations are
in different files of the package.

Several globals can share one declaration, as in Go. `var a, b i32`
declares two, `var a, b i32 = 1, 2` gives each its value, and
`var ( ... )` groups declarations. A global without a type takes the
type of its value, which may be any expression, as in Go: `var count = 3`,
`var total = f() + count`, `var p = Point{x: 1}` or `var s = []i32{1, 2}`.
A value with no type, such as the call of a function without results, is
a compile error that asks for the type.

`var a, b = f()` and `var a, b T = f()` give each global one result of
`f`, as in Go.

Globals are initialized in the order they are declared, except that a
global comes after the globals its value refers to, directly or through
the functions and methods it calls, as in Go. A global whose value refers back to
itself is the compile error `initialization cycle`.

```cx
package main

var width, height i32 = 3, 4
var (
	name = "box"
	scale f32 = 2.5
)

// area comes after sides, which it refers to.
var area = width * height * sides
var sides = count()

func count() (r i32) {
	return 4
}

func main () {
	global width, height, name, area, scale
	var perimeter, corners = 2 * (width + height), 4
	printf("%s %d %d %d %v\n", name, area, perimeter, corners, scale)
}
```

### [Local variables](#table-of-contents)

In contrast to global variables, local variables are constrained to
the function where they are declared. This means that is not possible
for another function to call a variable defined in another function.

```cx compile-error
package main

func foo () {
	i32.print(local) // this expression will throw a compile-time error
}

func main () {
	var local i32
	local = 5

	i32.print(5)
	foo()
}
```

If you try to run the example above, CX will throw an error similar to
this: `error: examples/testing.cx:4 identifier 'local' does not
exist`, so CX will not even try to run that program. If we could
de-activate CX's compile-time type checking, and the program above
could make it to the runtime, CX would not print 5 when running
`foo()`, as that function is unaware of that variable.

Local variables take the same forms as globals, `var a, b i32`,
`var a, b i32 = 1, 2` and `var ( ... )`. A local without a type,
`var a, b = x, y`, is the short declaration `a, b := x, y`, so its
values may be any expressions, and `var a, b = f()` takes the two
results of `f`. A variable declared without a type, by `var c = v` or
`c := v`, takes the whole type of `v`, so `var c = p` of a `*Point` is a
`*Point` and `x := f()` of an `f` returning `[]*i32` or `*[]i32` has that
type.

A `:=` whose names and values differ in number is the compile error
`assignment mismatch`. Its new names are declared without a type, so
their uses report nothing more, and a name that the function declared
already is not redeclared, as `:=` assigns it in Go.

### [Scopes and shadowing](#table-of-contents)

A block, the body of an `if`, `else` or `for` and a `{ ... }` inside a
function, opens a scope. A variable declared in it is gone at the end
of the block, and a `for` declares its loop variable for the loop only.
Two blocks that do not nest may declare the same name.

A name may not hide another name, which is called shadowing and is a
compile error in CX. A local, a parameter, a result, a loop variable or
a name a `:=` declares may not have the name of

* a global of its package,
* a package the file imports,
* a function or a type of its package,
* a variable of a block around it.

The error names the declaration it would hide, for example `'x' shadows
variable declared at main.cx:4:2` or `'counter' shadows global declared
at main.cx:3:5`. A `:=` that assigns a name its own block declares hides
nothing. A name that an import brings into scope may not be masked
either, see [Imports into scope](#imports-into-scope).

```cx compile-error
package main

var counter i32

func main () {
	global counter
	counter = 1
	x := 1
	if x > 0 {
		x := 2 // 'x' shadows variable declared at line 8
		i32.print(x)
	}
}
```

### [Arrays](#table-of-contents)

Arrays (or vectors) and multi-dimensional arrays (or matrices) can be
declared using a syntax similar to C's.

```cx
package main

type Point struct {
    x i32
    y i32
}

func main () {
    var arr1 [5]i32
    var arr2 [5]Point
    var arr3 [2][2]f32

    arr1[0] = 10
    arr2[1] = Point{x: 20, y: 30}
    arr3[1][0] = 1.5
}
```

In the example above we see the declaration of an array of 5 elements
of type *i32*, followed by an array of the same cardinality but of
type *Point*, which is a custom type. Custom types are discussed in a
later section. Lastly, we see an example of a 2x2 matrix of type
*f32*.

Lastly, we can see how we can set array elements using the bracket
notation, e.g. `arr1[0] = 10`.
Each index takes its own brackets, as in `arr3[1][0]`. A comma inside
the brackets, `arr3[1, 0]`, is a syntax error.

An array can hold pointers and structs of another package, as in Go,
in variables, fields, parameters and results. `[3]*i32` is an array of
three pointers, `[2][2]*Point` a matrix of pointers and `[4]pack.T` an
array of the struct `T` of the imported package `pack`. Slices and
pointers of such types, `[]pack.T` and `*pack.T`, work the same way.
An array can also hold slices. `[2][]i32` is two slices of `i32`, and
`a[1] = append(a[1], 5)` grows the second one, in a variable, a global
or a struct field. An array of pointers to arrays and an array of
functions, such as `[2]*[3]i32` and `[2]func(i32)()`, are not supported
yet and are syntax errors.

### [Slices](#table-of-contents)

Golang-like slices exist in CX (dynamic arrays). Slices are declared
similarly to arrays, with the only difference that the size is
omitted.

```cx
package main

type Point struct {
    x i32
    y i32
}

func main () {
    var slc1 []i32
    var slc2 []Point
    var slc3 [][]f32
}
```

Slices, unlike arrays, cannot be directly initialized using the
bracket notation (unless you use the native function `make`
first). You can use the bracket notation to reassign values to a
slice, once an element associated to the index that you want to use
already exists, as shown in the example below.

```cx
package main

func main () {
    var slc []i32

    slc = append(slc, 1)
    slc = append(slc, 2)

    slc[0] = 10
    slc[2] = 30 // This is not allowed, as len(slc) == 2, not 3
}
```

As this behavior is more related to the logic behind slices, it is
further explained in the *Runtime->Data Structures->Slices* section.

### [Literals](#table-of-contents)

A literal is any data structure that is not being referenced by any
variable yet. For example: `1`, `true`, `[]i32{1, 2, 3}`, `Point{x:
10, y: 20}`.

Particularly, it is worth noting the cases of array, slice and struct
literals.

```cx
package main

type Point struct {
    x i32
    y i32
}

func main () {
    var a Point
    var b [5]i32
    var c []i32

    a = Point{x: 10, y: 20}
    b = [5]i32{1, 2, 3, 4, 5}
    c = []i32{100, 200, 300}
}
```

In the example above we can see examples of struct (`Point{x: 10,
y: 20}`), array (`[5]i32{1, 2, 3, 4, 5}`), and slice (`[]i32{100, 200, 300}`)
literals, in that order. These literals exist to simplify the creation
of such data structures.

A string literal in double quotes takes the escapes of Go: `\a`, `\b`,
`\f`, `\n`, `\r`, `\t`, `\v`, `\\` and `\"`, a byte as `\x41` or as
the octal `\101`, and a Unicode code point as `\u00e9` or `\U0001F600`,
written in UTF-8. Any other escape is the compile error `unknown escape`.
A raw string literal in back quotes takes no escapes, and its carriage
returns are dropped. CX has no rune literals.

A literal is a value, so it can be used wherever a value can: as the
initializer of a `var`, as a function argument or result, as a field of
a struct literal and as an element of an array or slice literal. Fields
and elements can be any expressions, including calls, and literals can
nest to any depth.

```cx
package main

type Point struct {
    x i32
    y i32
}

type Line struct {
    a Point
    b Point
    tags []i32
}

func length2 (l Line) (n i32) {
    n = (l.b.x - l.a.x) * (l.b.x - l.a.x) + (l.b.y - l.a.y) * (l.b.y - l.a.y)
}

func main () {
    var l Line = Line{a: Point{x: 1}, b: Point{x: 4, y: 4}, tags: []i32{7}}
    i32.print(length2(l)) // prints 25
    i32.print(length2(Line{b: Point{x: 3}})) // prints 9

    var ps [2]Point = [2]Point{Point{x: 1}, Point{y: 2}}
    i32.print(ps[1].y) // prints 2
}
```

As in Go, a field or element that the literal does not name is zero, so
`p = Point{}` sets every field of `p` to zero and `p = Point{x: 1}` also
sets `p.y` to zero. Every field is computed before `p` is written, so
`p = Point{x: p.y, y: p.x}` swaps the two fields. An array literal with
more elements than its length and a struct literal that names a field
twice are compile errors.

As in Go, an element of an array or slice literal can leave out its
type when it is itself a literal. `[][]i32{{1, 2}, {3}}` is
`[][]i32{[]i32{1, 2}, []i32{3}}`, `[2][2]i32{{1, 2}, {3, 4}}` is
`[2][2]i32{[2]i32{1, 2}, [2]i32{3, 4}}` and `[]Point{{x: 1}}` is
`[]Point{Point{x: 1}}`. Leaving out the type of an element that is not
an array, a slice or a struct, as in `[]i32{{1}}`, is a compile error.

The element type may be a pointer or mix array and slice dimensions, as
in `[]*i32{&x}`, `[2]*Point{&a, &b}`, `[]*[2]i32{&a}`,
`[2][]i32{{1}, {2, 3}}` and `[][2][]i32{{{1}, {2}}}`. An element of a
`[]*Point` may leave out `&Point` too, so `[]*Point{{x: 1}}` is
`[]*Point{&Point{x: 1}}`.

Some forms of Go are not supported yet. A struct literal names its
fields, so `Point{1, 2}` is a syntax error and `[]Point{{1, 2}}` is a
compile error. A field of a call result, as in `mk().x`, works as in
Go, and so do a field, a method call or an index after a struct
literal, as in `Point{x: 1}.x` and `Point{x: 1}.norm()`. A struct
literal is an operand like any other, so `Point{x: 1}.norm() + 1` and
`f(Point{x: 2}.x * 2)` work. As in Go, a struct literal in the header
of an `if` or `for` must be in parentheses, `if (Point{x: 1}.x == 1) {`,
since `if p {` opens the body. Without them the error says so,
`composite literal in condition must be parenthesized`.

### [Functions](#table-of-contents)

Functions in CX are similar in syntax to functions in Go. A function
with one output may give only its type, as in `func double (n i32) i32`.
Outputs in parentheses are all named, as in `func pair () (a i32, b i32)`,
or all given by type only, as in `func pair () (i32, str)`, which
`return 1, "a"` fills. An output may be of any type, an array such as
`[2]i32` or `*[2]i32` included.

```cx
package main

func foo () {

}

func main () {
	foo()
}
```

The example above doesn't do anything, but it illustrates the anatomy
of a function. In the case of `foo`, we have an empty function
declaration, and then we have `main`, which is defined by a single
call to `foo`. Functions can also receive inputs and return outputs,
as in the example below.

```cx
package main

func foo (in i32) {
    i32.print(in) // this will print 5
}

func bar () (out i32){
    out = 10
}

func main () {
    foo(5)

    var local i32
    local = bar()
    i32.print(local) // this will print 10
}
```

In this case, `foo` is declared to receive one input parameter, and
`bar` is declared to return one output parameter.

CX has no function values, so a function is used only by calling it.
`foo` or `pkg.foo` without parentheses is not a call. As a statement it
is the compile error `invalid expression`, and as a value it is a type
error. The result of a call cannot be called either, so `foo() ()` is
the compile error `the result of a call is not a function`.

`pkg.foo` names a function declared in the package `pkg`, not one of
the packages `pkg` imports.

A function may have the name of a built-in function, such as `add` or
`len`. Inside its package a call by that name calls the function, and
other packages still get the built-in. Operators such as `+` always use
the built-in.

```cx
package main

func add (a i32, b i32) (c i32) {
	c = a - b
}

func main () {
	i32.print(add(3, 4)) // prints -1
	i32.print(3 + 4) // prints 7
}
```

### [Custom Types](#table-of-contents)

If primitive types are not enough, you can define your own custom
types by combining the primitive types and other constructs like
slices, arrays, and even other custom types.

```cx
package main

type Point struct {
    x i32
    y i32
}

func main () {
    var p Point
    p.x = 10
    p.y = 20

    printf("Point coordinates: (%d, %d)\n", p.x, p.y)
}
```

In the example above, we can see a custom type that defines a
*Point* as the combination of two 32-bit integers (*i32*). After
declaring the custom type, you can start declaring variables of that
type anywhere in the package where it was declared in. The code in
`foo` shows how you can create and use an instance of that structure.
An instance can also be written as a struct literal, such as
`Point{x: 10, y: 20}`, which the Literals section describes.

### [Methods](#table-of-contents)

A variation of functions that are associated to custom types are
*methods*.

```cx
package main

type Point struct {
	x i32
	y i32
}

type Line struct {
	a Point
	b Point
}

func (p Point) print () {
	printf("Point coordinates: (%d, %d)\n", p.x, p.y)
}

func (l Line) print () {
	printf("Line point A: (%d, %d), Line point B: (%d, %d)\n", l.a.x, l.a.y, l.b.x, l.b.y)
}

func main () {
	var l Line
	var p1 Point
	var p2 Point

	p1.x = 10
	p1.y = 20
	p2.x = 11
	p2.y = 21

	l.a = p1
	l.b = p2

	p1.print()
	p2.print()
	l.print()
}
```

In the example above, we define two custom types: *Point* and
*Line*. The type line is defined by two fields of type
*Point*, and the type *Point* is defined as coordinate defined by two
fields of type *i32*.

As a simple example, we create two methods called `print`, one for the
type *Point* and another for the type *Line*. In the case of
`Point.print`, we just print the two coordinates, and in
the case of `Line.print` we print the coordinates of the two points
that define the *Line* instance.

A method call is an expression like a function call. Its result can be
an operand, an argument, an index, a condition or the receiver of
another call, and its arguments can be any expressions.

A method with a pointer receiver can be called on a value or on a
pointer. Called on a value, it gets the address of the value, so it can
change it. As in Go, the value must be addressable, so
`Point{x: 1}.move(2)` and `mk().move(2)` are the compile error
`cannot call pointer method move on Point`, and `(&Point{x: 1}).move(2)`
works. A method with a value receiver called on a pointer gets a
copy of the value the pointer points to, as in Go.

```cx
package main

type Point struct {
	x i32
	y i32
}

func (p Point) sum () (r i32) {
	r = p.x + p.y
}

func (p Point) shifted (n i32) (q Point) {
	q.x = p.x + n
	q.y = p.y + n
}

func (p *Point) move (n i32) {
	p.x = p.x + n
}

func main () {
	var p Point
	p.x = 1
	p.y = 2
	var n i32 = 10

	i32.print(p.sum() * 2) // prints 6
	i32.print(p.shifted(n).sum()) // prints 23
	if p.sum() == 3 {
		p.move(n)
	}
	i32.print(p.x) // prints 11
}
```

### [Packages](#table-of-contents)

In the previous examples we have always been using a single package:
`main`. If your program grows too large it's convenient to divide your
code into different packages.


```cx
package foo

func fn (in i32) {
    i32.print(in)
}

package bar

func fn () (out i32) {
    out = 5
}

package main
import "foo"
import "bar"

func main () {
    foo.fn(10) // prints 10

    var num i32
    num = bar.fn()

    i32.print(num) // prints 5
}
```

In the example above, we can see how two functions with the same name
(`fn`) are declared, each in a separate package. Both of these
functions have different signatures, as `foo.fn` accepts a single
input parameter and `bar.fn` doesn't accept any inputs but returns a
single output parameter.

We can then see how the `main` package `import`s both the `foo` and
`bar` packages, to later call each of these functions.

A global of an imported package is reached the same way, as
`foo.Count`, and a function that uses it declares it as
`global foo.Count`, see [Global variables](#global-variables). A member
of an imported package is never found by its name alone, unless an
import brings it into scope, see [Imports into scope](#imports-into-scope).

Built-in packages are implemented inside the compiler and runtime. Each
lives under `sys/` and is marked as built in where it is documented. A
program still imports it explicitly, by its `sys/` path, and calls it by
the last element of the path.

```cx
package main

import "sys/string"

func main () {
    string.PrintLine(string.Concatenation("Hello, ", "World!"))
}
```

Using a built-in package without its import is a compile error, and so
is importing one without `sys/` (`import "string"`). The built-in
packages are `sys/string`, `sys/os`, `sys/time`, `sys/tcp`,
`sys/regexp`, `sys/cipher` and `sys/cx`, and in a cxfx build `sys/gl`,
`sys/glfw`, `sys/al` and `sys/gltext`. Their functions have verbose
camel case names, and none of them is an operator. See
[docs/Conventions.md](docs/Conventions.md) rule 14.

### [Imports into scope](#table-of-contents)

`import "geometry"` lets a file use the members of `geometry` only as
`geometry.Point`, `geometry.Distance(a, b)` or `geometry.Count`. An
import into scope also lets it use a member by its name alone, as a
shorthand. The compiler still reads `Point` as `geometry.Point`, and the
qualified name keeps working. There are two scopes, each with its own
form.

| Import | Brings in | Where |
|---|---|---|
| `import "geometry"` | nothing, members are `geometry.name` | |
| `import . "geometry"` | every global, function and type | the package |
| `import file . "geometry"` | every global, function and type | this file |
| `import (Point, Distance) "geometry"` | the names listed | the package |
| `import file (Point, Distance) "geometry"` | the names listed | this file |

`import . "pkg"` is the form of Go. The package scope covers every file
of the package, and the file scope only the file that holds the import,
so another file of the package reaches the names as `geometry.Point`. A
listed name may be a type, a function or a global, and the names of a
built-in package work too, `import (PrintLine) "sys/string"`. A function
that uses a global brought in this way declares it by its short name,
`global Count`, or by its qualified one.

```cx
package geometry

type Point struct {
	X i32
	Y i32
}

func Distance(a Point, b Point) i32 {
	return b.X - a.X + b.Y - a.Y
}

package main

import (Point, Distance) "geometry"
import (PrintLine) "sys/string"

func main() {
	var a Point
	b := Point{X: 3, Y: 4}
	i32.print(Distance(a, b)) // prints 7
	PrintLine("done")
}
```

A name brought into scope may not mask another name. Masking is a
compile error, `'Distance' masks the name imported at main.cx:3:16`,
when the name is also

* a local, a parameter or a result of a function in that scope,
* a global, a function or a type of the package,
* a package the package imports,
* a name another import brings into the same scope from another
  package.

The error is at the other name and gives the position of the import. A
name a scoped import lists that the package does not have, a name listed
twice and a word other than `file` before the import are compile errors
as well.

A comment `//cx:masking warning` on the line before a function, written
like a Go compiler directive, makes masking by its locals, parameters and
results a warning. The warning prints and the program still builds. In
that function the local hides the imported name from its declaration to
the end of the function. Masking by a declaration of the package is
always an error.

```cx
package geometry

var Count i32 = 7

package main

import (Count) "geometry"

//cx:masking warning
func main() {
	Count := 1 // warning: 'Count' masks the name imported at main.cx:7:9
	i32.print(Count) // prints 1
}
```

### [System libraries](#table-of-contents)

A system library is a group of built-in functions that reach the
operating system. A program calls them as any other built-in function,
`os.Open("data.txt")` after `import "sys/os"`, but a call only runs when
the library is enabled for the run. A library is enabled with `--enable`.

```
cx --enable=file,net program.cx
```

| library | functions |
|---|---|
| `file` | the file functions of `sys/os` (`os.Open`, `os.Create`, `os.Close`, `os.Seek`, `os.Read*`, `os.Write*`, `os.ReadAllText`, `os.GetWorkingDirectory`, `os.LogFile`), `StartCPUProfile` and `StopCPUProfile` |
| `net` | every function of `sys/tcp` |
| `process` | `os.Run` |
| `cipher` | every function of `sys/cipher`, see [The cipher library](#the-cipher-library) |

`os.Exit`, `sys/time`, `sys/regexp` and `sys/string` are not system
libraries. They end the program, read the clock or compute, and always
run.

A call into a library that is not enabled is the runtime error
`CX_RUNTIME_LIBRARY_DISABLED` (12), on the old loop and on vm, and the
message names the library.

```
error: main.cx:7, CX_RUNTIME_LIBRARY_DISABLED, library file is not enabled, enable it with --enable=file
```

Libraries are disabled unless `--enable` names them. The setting
`constants.LibraryPolicyEnabledWhenImported` in
`t1-constants/libraries.go` holds this default. Set to `true`, it
enables every library the program calls. See
[skycoin-cx#492](https://github.com/skycoin/skycoin-cx/issues/492),
decision D17.

The compiler turns every call of a library function into one opcode,
`LibraryCall` (795), which carries the library number and the function
number. The numbers are in `constants.LibraryTable` and
`constants.LibraryFunctionLifecycleTable` and follow the rules of opcode
numbers, see [Language versions](#language-versions).

### [The cipher library](#table-of-contents)

`sys/cipher` hashes bytes and makes and checks secp256k1 signatures. It
is the system library `cipher`, so a run enables it with
`--enable=cipher`. Bytes, digests, keys and signatures are `[]ui8`.

| function | returns |
|---|---|
| `cipher.HashOfBytes(data []ui8)` | the 32 byte digest of `data` |
| `cipher.HashStateNew()` | an `i32` state for data that arrives in parts |
| `cipher.HashStateWrite(state i32, data []ui8)` | nothing, adds `data` to the state |
| `cipher.HashStateSum(state i32)` | the digest of what was written, the state keeps it |
| `cipher.HashStateRelease(state i32)` | nothing, frees the state |
| `cipher.SignatureVerify(hash, signature, pubKey []ui8)` | `bool`, whether `pubKey` signed `hash` |
| `cipher.SignatureRecoverPubKey(hash, signature []ui8)` | the public key that signed, and a `bool` |
| `cipher.SignatureSign(hash, secKey []ui8)` | the signature, and a `bool` |
| `cipher.KeyPairGenerate()` | a new public key and secret key |

The hash is SHA-256 until decision D23 names the BLAKE hash that replaces
it. Signatures are those of the `cipher` package of Skycoin (decision
D22): a public key is 33 bytes, a secret key 32 and a signature 65, and an
input of another length does not verify, recover or sign.

```cx
package main

import "sys/cipher"
import "sys/string"

func main() {
	var pubKey []ui8
	var secKey []ui8
	pubKey, secKey = cipher.KeyPairGenerate()
	var digest []ui8 = cipher.HashOfBytes(string.BytesOfString("Hello World!"))
	var signature []ui8
	var signed bool
	signature, signed = cipher.SignatureSign(digest, secKey)
	printf("%v %v\n", signed, cipher.SignatureVerify(digest, signature, pubKey)) // true true
}
```

At most 64 hash states are held at once, and a state that was released or
never returned is an assert. The function numbers, allocations and vm
support are in [docs/CipherLibrary.md](docs/CipherLibrary.md).

## [Statements](#table-of-contents)

Statements are different to declarations, as they don't create any
named elements in a program. They are used to control the flow of a
program.

### [If and if/else](#table-of-contents)

The most basic statement is the *if* statement, which is going to
execute a block of code only if a condition is true.

```cx
package main

import "sys/string"

func main () {
    if false {
        string.PrintLine("This will never print")
    }
}
```

The example above won't do anything, as the condition for the *if*
statement is always going to evaluate to *false*.

```cx
package main

import "sys/string"

func main () {
    if true {
        string.PrintLine("This will always print")
    }
}
```

In contrast, the example above will always print.

```cx
package main

import "sys/string"

func main () {
    if true {
        string.PrintLine("This will always print")
    } else {
        string.PrintLine("This will never print")
    }
}
```

Lastly, the example above shows how to write an *if/else* statement in CX.

As a note about its syntax, the predicates or conditions don't need to
be enclosed in parentheses, just like in Go.

### [Else if](#table-of-contents)

Instead of simply adding one alternative path, you can string together a series
of *else if* blocks, which check for as many different conditions as you like.
Giving you similar functionality as Go's *switch*/*select* blocks (containing
various conditions/cases).

```cx
package main

import "sys/string"

func main () {
   var i i32
   i = 0

   if i == 0 {
     string.PrintLine("i is 0")
   } else if i == 1 {
     string.PrintLine("i is 1")
   } else if i == 2 {
     string.PrintLine("i is 2")
   } else {
     string.PrintLine("i is NOT 0, 1 or 2")
   }
}
```

### [For loop](#table-of-contents)

CX's only looping statement is the *for* loop. Similar to Go, the
*for* loop in CX can be used as the *while* statement in other
programming languages, and as a traditional *for* statement.

```cx
package main

import "sys/string"

func main () {
	for true {
		string.PrintLine("forever")
	}
}
```

As the simplest example of a loop, we have the infinite loop shown in
the example above. In this case, the loop will print the character
string "forever" indefinitely. If you try this code, remember that you
can cancel the program's execution by hitting `Ctrl-C`.

A bare boolean variable works as the condition too, `for running { ... }`.
A struct literal in the condition or the post statement needs parentheses.
`break` leaves the innermost loop and `continue` starts its next iteration.
Either one outside a loop is a compile error.

```cx
package main

import "sys/string"

func main () {
	for string.EqualityByByteValue("hi", "hi") {
		string.PrintLine("hi")
	}
}
```

The code above shows another example, one where we use an expression
as its predicate, rather than a literal `true` or `false`. It is worth
mentioning that strings have no `==` operator, see [Language versions](#language-versions).


```cx
package main

func main () {
	var c i32
	for c = 0 ; c < 10; c++ {
		i32.print(c)
	}
}
```

The traditional *for* loop shown in the example above. In languages
like C, you need to first declare your counter variable, and then you
have the option to initialize or reassign the counter in the first
part of the *for* loop. The second part of the *for* loop is reserved
for the predicate, and the last part is usually used to increment the
counter. Nevertheless, just like in C, there's nothing stopping you
from doing whatever you want in the first and last parts. However, the
predicate part needs to be an expression that evaluates to a Boolean.

```cx
package main

func main () {
	for c := 0; c < 10; c++ {
		i32.print(c)
	}
}
```

A more Go-ish way of declaring and initializing the counter is to use
an inline declaration, as seen in the example above.

```cx
package main

func main () {
	var c i32
	c = 0
	for ; c < 10; c++ {
		i32.print(c)
	}
}
```

Lastly, the for loop can also completely omit the initialization part,
as seen above.

### [Goto](#table-of-contents)

`goto` can be used to immediately jump the execution of a program to
the corresponding labeled expression.

```cx
package main

import "sys/string"

func main () (out i32) {
	goto label2
label1:
	string.PrintLine("this should never be reached")
label2:
	string.PrintLine("this should be printed")
}
```

In the example above, we see how a `goto` statement forces CX to
ignore executing the expression labeled as `label1`, and instead jumps
to the `label2` expression.

The label must be in the same function. A `goto` to a label that does
not exist is the compile error `label 'name' is not defined`.

## [Expressions](#table-of-contents)

Expressions are basically function calls. But the term expression also
takes into consideration the variables that are receiving the
function's output arguments, the input arguments, and any dereference
operations.

```cx
package main

func foo () (arr [2]i32) {
    arr = [2]i32{10, 20}
}

func main () {
    i32.print(foo()[0])
}
```

For example, the expression `i32.print(foo()[0])` in the code above
consists of two function calls, and the array returned by the call
to `foo` is "dereferenced" to its *0th* element.

`&&` and `||` short-circuit as in Go. The right operand runs only when
the left one does not decide the result, so `i < len(s) && s[i] > 0`
never reads past the end of `s`. The functions `bool.and` and `bool.or`
are ordinary calls and evaluate both arguments.

The unary operators `-` and `!` apply to any operand, such as a call
`-f(x)`, a method call `!p.done()`, a struct field `-p.x` or a
parenthesized expression `!(a && b)`.

Binary operators bind as in Go
([Operator precedence](https://go.dev/ref/spec#Operator_precedence)).
There are five levels, from the tightest to the loosest. Operators of
one level group from the left, so `a - b + c` is `(a - b) + c`.

| Level | Operators |
|-------|-----------|
| 5 | `*` `/` `%` `<<` `>>` `&` `&^` |
| 4 | `+` `-` `\|` `^` |
| 3 | `==` `!=` `<` `<=` `>` `>=` |
| 2 | `&&` |
| 1 | `\|\|` |

So `d & 1 == 1` is `(d & 1) == 1`, and `1 + 2 << 3` is `1 + (2 << 3)`.
Unary operators bind tighter than any binary operator.

## [Assignments and Initializations](#table-of-contents)

As in many other C-like languages, assignments are done using the
equal (`=`) sign.

```cx
package main

func main () {
    var foo i32
    foo = 5
    foo = 50
}
```

In the case of the code above, the variable `foo` is declared and then
initialized to `5` using the `=` sign. Then we reassign the `foo`
variable to the value `50`.

```cx
package main

func main () {
    foo := 5
    foo = 50
}
```

As in other programming languages, *short variable declarations* exist
in CX. The `:=` token can be used to tell CX to infer a variable's
type. This way, CX declares and initializes at the same time, as seen
in the example above.

`i++` and `i--` add or subtract 1 of the type of `i`, which can be any
integer or floating point type, a struct field, an element or `*p`. As
in Go they are statements, so `f(i++)` and `x = i++` are syntax errors,
and there is no prefix `++i`.

`=` and `:=` take several targets and as many values, or several
targets and one call with as many results. As in Go, every value is
computed before any target is assigned, so `a, b = b, a` swaps.

```cx
package main

func pair () (n i32, s str) {
	n = 7
	s = "seven"
}

func main () {
	a, b := 1, 2
	a, b = b, a
	i32.print(a) // prints 2
	n, s := pair()
	printf("%d %s\n", n, s) // prints 7 seven
}
```

A different number of targets and values is the compile error
`assignment mismatch`. As in Go, `:=` with several names declares the
names its block does not have yet and assigns the others, so
`a, b := 2, 3` after `a := 1` assigns `a`. A name of an enclosing block
is declared again in the inner one. A `:=` that declares no new name is
the compile error `no new variables on left side of :=`. A call with
several results used as one of several values, `a, b := two(), 1`, is
the compile error `multiple-value two() (value of type (a i32, b i32))
in single-value context`, and a call without results, `var y i32 =
none()`, is `function 'none' returns no value`. One difference from Go
remains. An index of a target is read when the target is assigned, not
before the values are computed, so after `i, s[i] = 1, 7` it is `s[1]`
that holds 7.

## [Affordances](#table-of-contents)

The affordance system lets a program ask, at run time, what can be
done with one of its own objects, such as an expression, a function or
a package, and filter the answers with predicate functions. A block
written `#{ ... }` holds the target and the filters, and functions of
the `aff` package, such as `aff.query`, run the query.

Affordances cannot be used right now, an open bug tracked in
[#500](https://github.com/skycoin/skycoin-cx/issues/500). Their
opcodes are still in the compiler, but `aff` is not in the list of
built-in packages, so `import "aff"` fails and `import "sys/aff"` is an
unknown built-in package. The programs in `tests/examples/affordances`
fail to load for the same reason.

# [Runtime](#table-of-contents)

The previous section presents the language features from a syntax
perspective. In this section we'll cover what's the logic behind these
features: how they interact with other elements in your program, and
what are the intrinsic capabilities of each of these features.

## [Packages](#table-of-contents)

Packages are CX's mechanism for better organizing your code. Although
it is theoretically possible to store a big project in a single
package, the code will most likely become very hard to understand. In
CX the programmer is encouraged to place the files that define the
code of a package in separate directory. Any subdirectory in a
package's directory should also contain only source code files that
define elements of the same package. Nevertheless, CX will not
throw any error if you don't follow this way of laying out your source
files. In fact, you can declare different packages in a single source
code file.

## [Data Structures](#table-of-contents)

Data structures are particular arrangements of bytes that the language
interprets and stores in special ways. The most basic data structures
represent basic data, such as numbers and character strings, but these
basic types can be used to construct more complex data types.

### [Literals](#table-of-contents)

A literal is any data structure that is not being referenced by any
variable yet. For example: `1`, `true`, `[]i32{1, 2, 3}`, `Point{x:
10, y: 20}`.

It's important to make a distinction, particularly with *arrays*,
*slices* and *struct instances*.

```cx
package main

type Point struct {
	x i32
	y i32
}

func main () {
	var p1 Point
	p1.x = 10
	p1.y = 20

	p2 := Point{x: 11, y: 21}

	i32.print(p2.x)
	i32.print(p2.y)
}
```

In the example above we are creating two instances of the `Point`
type. The first method we use does not involve struct literals, as a
variable of that type is first created and then initialized.

In the second case (`p2`), the full struct instance is first
created. CX creates an anonymous struct instance as soon as it
encounters `Points{x: 11, y: 21}`, and then it proceeds to assign that
literal to the `p2` variable, using *short variable declarations*.

```cx
package main

func main () {
	var arr1 [3]i32
	arr1[0] = 1
	arr1[1] = 2
	arr1[2] = 3

	var arr2 [3]i32
	arr2 = [3]i32{10, 20, 30}
}
```

```cx
package main

func main () {
	var slc1 []i32
	slc1 = append(slc1, 1)
	slc1 = append(slc1, 2)
	slc1 = append(slc1, 3)

	var slc2 []i32
	slc2 = []i32{10, 20, 30}
}
```
Similarly, in the two examples above we can see how we can declare
array and slice variables and then we initialize them. In the case of
arrays, we use the bracket notation, and for slices we have to use
`append`, as `slc1` starts with a size and capacity of 0. In the cases
of `arr2` and `slc2`, we use literals to initialize them more
conveniently. A short variable declaration works with a struct, array
or slice literal, as `p2` shows above and as in `arr3 := [3]i32{10, 20, 30}`.

Regarding numbers, you need to be aware that implicit casting does not
exist in CX. This means that the number `34` cannot be assigned to a
variable of type *i64*. In order to assign it, you need to either
parse it using the native function `i32.i64` or you can create a
64-bit integer literal. An integer literal without a suffix is an
*i32* and a floating-point literal without a suffix is an *f32*. For
other types, use these suffixes: `B` for *i8*, `H` for *i16*, `L` for
*i64*, `UB` for *ui8*, `UH` for *ui16*, `U` for *ui32*, `UL` for
*ui64* and `D` for *f64*. So, assuming `foo` is of type *i64*, you can
do this assignment: `foo = 34L`.

The same holds for the result of an operator. `-1` is the negation of
the *i32* literal `1`, so `var f f32 = -1` and `var x i64 = 1 + 2` are
compile errors. Write `-1.0` for an *f32* and `-1L` for an *i64*.

```cx compile-error
package main

func main () {
	var f f32 = -1
}
```

### [Variables](#table-of-contents)

When CX compiles a program, it knows how many bytes need to be
reserved in the stack for each of the functions. CX can know this
thanks to variable declarations.


```cx
package main

type Point struct {
    x i32
    y i32
}

func foo (inp Point) {
    var test1 i64
    var test2 bool
}

func main () {
    var test3 i32
    var test4 f32
}
```

The two functions declared in the example above are going to reserve 17
and 8 bytes in the stack, respectively. In the case of the first
function, `foo` needs to reserve space for an input parameter of type
`Point`, which requires 8 bytes (because of the two *i32* fields), and
two local variables: one 64-bit integer that requires 8 bytes and a
Boolean that requires a single byte. In the case of `main`, CX needs
to reserve bytes for two local variables: a 32-bit integer and a
single-precision floating point number, where each of them require 4
bytes.

```cx
package main

var global1 i32

func main () {
    var local i32
}
```

Local variables are different than global variables. In order for
globals to have a global scope they need to be allocated in a
different memory segment than local variables. This different memory
segment does not shrink or get bigger like the stack. This means that
any global variable is going to be kept "alive" as long as the program
keeps being executed.

A global scope means that variables of this type are accessible to any
function declared in the same package where the variable is declared,
and to any function of other packages that are importing this package.

```cx
package main

func main () {
    var foo i32
    i32.print(foo) // prints 0
}
```

In CX every variable is going to initially point to a *nil*
value. This *nil* value is basically a series of one or more zeroes,
depending on the size of the data type of a given variable. For
example, in the code above we see that we have declared a variable of
type *i32* and we immediately print its value without initializing
it. This CX program will print 0, as the value of `foo` is `[0 0 0 0]`
in the stack (4 zeroes, as a 32-bit integer is represented by 4
bytes).

In the case of data types that point to variable-sized
structures, such as slices or character strings, these are initialized
to a nil pointer, which is represented by 4 zeroed bytes. This nil
pointer is located in the heap memory segment, instead of the stack.

### [Primitive types](#table-of-contents)

There are twelve primitive types in CX: *bool*, *str*, *i8*, *i16*,
*i32*, *i64*, *ui8*, *ui16*, *ui32*, *ui64*, *f32* and *f64*. These
types can be used to construct other more complex types, as will be
seen in the next sections.

*bool*, *i8* and *ui8* require a single byte to represent their
 values. In the case of *bool*, there are only two possible values:
 `true` or `false`. A *ui8* holds 256 values, from 0 to 255. *i16* and
 *ui16* take 2 bytes, *i32*, *ui32* and *f32* take 4 bytes, and
 *i64*, *ui64* and *f64* take 8 bytes each.

Now, strings are special as they are static and dynamic sized at the
same time. If you have a look at how a variable of type *str* reserves
memory in the stack, you'll see that it requires 4 bytes, regardless
of what text it's pointing to. The explanation behind this is that any
*str* in CX actually behaves like a pointer behind the scenes, and the
actual string gets stored in the heap memory segment.

```cx
package main

import "sys/string"

func main () {
	var foo str

	foo = string.Concatenation("Hello, ", "World!")
	foo = "Hi"
}
```

When CX compiles the example above, three strings are first stored in
the data memory segment (just like global variables, as these strings
are constants, memory-wise): `"Hello, "`, `"World"` and `"Hi"`. When
the program is executed, `string.Concatenation` is called, which creates a new
string by concatenating `"Hello, "` and `"World!"`, and this new
character string is allocated in the heap memory segment. Then `foo`
is assigned only the address of this new character string. Then we
immediately re-assign `foo` with the address of `"Hi"`. This means
that `foo` was first assigned a memory address located in the data
memory segment, and then it was assigned an address located in the
heap.

### [Arrays](#table-of-contents)

Arrays, as in other programming languages, are used to create
collections of data structures. These data structures can be primitive
types, custom types or even arrays or slices.


```cx
package main

type Point struct {
    x i32
    y i32
}

func main () {
    var nums [5]i32
    var points [5]Point
}
```

In the example above, we're creating two arrays, one of a primitive
type and the other one of a custom type. CX reserves memory for these
arrays in the stack as soon as the function that contains them is
called. In this case, 60 bytes are going to be reserved for `main` as
soon as the program starts its execution, as `main` acts as the
program's entry point. You need to be careful with arrays, as those
can easily fill up your memory, especially with multi-dimensional
arrays (or matrices).

Also, another point to consider is performance. While accessing arrays
is almost as fast as accessing an atomic variable, arrays can be
troublesome when being sent/received as to/from functions. The reason
behind this is that an array needs to be copied whenever it is sent to
another function. If you're working with arrays of millions of
elements and you need to be sending that arrays millions of times to
another function, it's going to impact your program's performance a
lot. A way to avoid this is to either use pointers to arrays or slices.

### [Slices](#table-of-contents)

Dynamic arrays don't exist in CX. This means that the following
code is not a valid CX program:

```cx compile-error
package main

func main () {
    var size i32
    size = 13
    var arr [size] // this is not valid
}
```

If you need an array that can grow in size as required, you need to
use slices. Behind the scenes, slices are just arrays with some extra
features. First of all, any slice in CX goes directly to the heap, as
it's a data structure that is going to be changing in size. In
contrast, an array is stored in the stack frame unless its address
escapes the function, as [Escape Analysis](#escape-analysis) describes.

The second characteristic of slices in CX is how they change their
size. Any slice, when it's first declared, starts with a size and
capacity of 0. The size represents how many elements are in a given
slice, while the capacity represents how many elements can be
allocated in that slice without having to be relocated in the heap.

```cx
package main

func main () {
	var slc []i32

	slc = append(slc, 1)
	slc = append(slc, 2)
	slc = append(slc, 3)
	slc = append(slc, 4)
}
```

In the code above we can see how we declare a slice and then we
initialize it using the `append`function. After all the `append`s,
we'll end up with a slice of size 4 and capacity 4, and this
`append`ing process will create the following objects in the heap:

```
[0 0 0 0 0 12 0 0 0 1 0 0 0 1 0 0 0 1 0 0 0 0 0 0 0 0 16 0 0 0 2 0 0 0 2 0 0 0 1 0 0 0 2 0 0 0 0 0 0 0 0 24 0 0 0 4 0 0 0 4 0 0 0 1 0 0 0 2 0 0 0 3 0 0 0 4 0 0 0]
```

First, the slice `slc` starts with 0 objects in it; it is pointing to
*nil*. Then, after the first `append`, the object
`[0 0 0 0 0 12 0 0 0 1 0 0 0 1 0 0 0 1 0 0 0]` is allocated to the
heap. The first five bytes are used by CX's garbage collector. The
next 4 bytes indicate the size of the object, and the remaining bytes
are the actual slice `slc`. The first four bytes of `slc` tell us its
current size, while the next four tell us its capacity. The remaining
bytes of this object are the elements of the slice.

The following object,
`[0 0 0 0 0 16 0 0 0 2 0 0 0 2 0 0 0 1 0 0 0 2 0 0 0]`, shows now a
size of 2 and a capacity of 2, with the 32-bit integers `1` and `2` as
its elements. The last object, `0 0 0 0 0 24 0 0 0 4 0 0 0 4 0 0 0 1 0
0 0 2 0 0 0 3 0 0 0 4 0 0 0`, needs careful attention. We can see that
our objects jumped from size 1 to 2 and finally 4. The same happened
to its capacity, and the containing elements are now `1`, `2`, `3` and
`4`. What happened to the slice of size 3 and capacity 3? First of
all, capacities are increased by getting doubled each time the size of
an object is greater than its capacity, so we would never get a slice
of capacity 4 by following this method. Next, we need to think on what
is capacity used for.

Slices are just arrays, which means that they can't be resized. The
dynamic nature of slices is emulated by copying the *full* slice to
somewhere else in memory, but with a greater capacity. However, this
will only happen if adding a new element to the existing slice would
overflow it. This is why slices keep track of two metrics: *size* and
*capacity*, i.e. how many actual elements are in the slice, and how
many elements the currently allocated slice can hold, respectively.

```cx
package main

func main () {
    var arr1 [1]i32
    arr1[0] = 1 // add the first value

    var arr2 [2]i32 // double the size
    arr2[0] = arr1[0] // copy previous array
    arr2[1] = 2 // add the second value

    var arr3 [4]i32 // double the size
    arr3[0] = arr2[0] // copy previous array
    arr3[1] = arr2[1] // copy previous array

    arr3[2] = 3 // add the third value
    arr3[3] = 4 // add the fourth value
}
```

The example above shows the behavior of the slice in the previous
example, but using arrays. When `resize` asks for more than twice the
capacity, the new capacity is the new size, and a slice too large for
the heap is a `CX_RUNTIME_HEAP_EXHAUSTED_ERROR`.

A slice literal can hold structs, and `[]i32{}` is an empty slice. A
pointer to a slice is dereferenced before it is indexed, measured or
passed to `append`, as in Go.

```cx
package main

type Point struct {
	x i32
}

func push(p *[]Point, x i32) {
	var q Point
	q.x = x
	*p = append(*p, q)
}

func main () {
	var points []Point
	points = []Point{Point{x: 1}, Point{x: 2}}
	push(&points, 3)

	var p *[]Point
	p = &points
	printf("%d %d\n", len(*p), (*p)[2].x) // prints 3 3
}
```

`append(p, q)`, `p[0]` and `*p[0]` on a pointer to a slice are compile
errors.

```cx compile-error
package main

func main () {
	var s []i32
	var p *[]i32
	p = &s
	p = append(p, 1)
}
```

The value passed to `append` or `insert` has the element type of the
slice, `copy` takes two slices of the same type, and the result of
`append`, `insert`, `resize` and `remove` is assigned to a slice of the
same type. Each mismatch is a compile error. This holds for slice struct
fields with more than one dimension too, such as `st.m[0]` on a
`m [][]i32` field.

```cx compile-error
package main

type A struct {
	f f32
}

type B struct {
	i i32
}

func main () {
	var s []A
	var b B
	s = append(s, b)
}
```

### [Structures](#table-of-contents)

Structures are CX's mechanism for creating custom types, as in many other
C-like languages. Structures are basically a grouping of other
primitive or custom types (called *fields*) that together create
another type of data structure. For example, a *point* can be defined
by its coordinates in a two-dimensional space. In order to create a
type `Point`, you can use a structure that contains two fields of type
*i32*, one for `x` and another for `y`, as in the example below.


```cx
package main

type Point struct {
	x i32
	y i32
}

func main () {
	var p Point
	p.x = 10
	p.y = 20
}
```

Only a struct has fields. `x.y` on a variable of any other type is a
compile error.

Whenever an instance of a structure is created by either declaring a
variable of that type or by creating a literal of that type, CX
reserves memory to hold space for all the fields defined in the
structure declaration. Like in C, the bytes are reserved depending on
the order of the fields in the structure declaration.

```cx
package main

type struct1 struct {
	field1 bool
	field2 i32
	field3 i64
}

type struct2 struct {
	field1 i64
	field2 bool
	field3 i32
}

func main () {
	var s1 struct1
	var s2 struct2
}
```

For example, in the code above a call to `main` will reserve a total
of 26 bytes in the stack. In the case of the first struct instance,
the first byte is going to represent `field1` of type *bool*, the next
four bytes are going to represent `field2` of type *i32*, and the
final 8 bytes are going to represent `field3` of type *i64*. In the
case of the next struct instance, the first eight bytes represent an
*i64* field so, although both struct instances contain the same number
of fields and of the same type, the byte layout changes.

A struct literal, such as `struct1{field2: 5}`, also reserves the bytes of
one instance. Its fields start at zero and the named ones are written,
then the instance is copied to where the literal is used, such as a
variable, an argument or a field of another struct.

### [Pointers](#table-of-contents)

Sometimes it's useful to pass variables to functions by reference
instead of by value.

```cx
package main
import "sys/time"

func foo (nums [100][100]i32) {
	// do something with nums
}

func main () {
	var start i64
	var end i64
	var nums [100][100]i32

	start = time.UnixMilli()

	for c := 0; c < 10000; c++ {
		foo(nums)
	}

	end = time.UnixMilli()

	printf("elapsed time: \t%d milliseconds\n", end - start)
}
```

The example above is very inefficient, as CX is going to be sending a
10,000 element matrix to `foo` 10,000 times. Every time `foo` is
called, every byte of that matrix needs to be copied for `foo`. In my
computer the example above takes around 638 milliseconds to run.

```cx
package main
import "sys/time"

func foo (nums *[100][100]i32) {
	// do something with nums
}

func main () {
	var start i64
	var end i64
	var nums [100][100]i32

	start = time.UnixMilli()

	for c := 0; c < 10000; c++ {
		foo(&nums)
	}

	end = time.UnixMilli()

	printf("elapsed time: \t%d milliseconds\n", end - start)
}
```

A new version of the last program is shown above. In contrast to the
last program, the code above sends a pointer to the matrix to
`foo`. A pointer in CX uses 4 bytes, or 8 bytes in a `cx` built with
`make build TAGS=ptr64`, so instead of copying 40,000 bytes, we only
copy 4 bytes to `foo` every time we call it. This version of the program takes only 3 milliseconds
to run in my computer.

```cx
package main

func foo (inp i32) {
	inp = 10
}

func main () {
	var num i32
	num = 15
	i32.print(num) // prints 15
	foo(num)
	i32.print(num) // prints 15
}
```

In the example above, we send`num` to `foo`, and then we re-assign
the input's value to `10`. If we print the value of `num` before and
after calling `foo`, we can see that in both instances `15` will be
printed to the console.

```cx
package main

func foo (num *i32) {
	*num = 10
}

func main () {
	var num i32
	num = 15
	i32.print(num) // prints 15
	foo(&num)
	i32.print(num) // prints 10
}
```

The code above is a pointer-version of the previous example. In this
case, instead of sending `num` by value, we send it by reference,
using the `&` operator. `foo` also changed, and it now accepts a
pointer to a 32-bit integer, i.e. `*i32`. After running the example,
you'll notice that, this time, `foo` is now changing `num`'s value.

Assigning a pointer copies the pointer, so both point to the same value.
A field after an index reads through the pointer, whether it is an
array of pointers or a pointer to an array.

```cx
package main

type Point struct {
	x i32
}

func main () {
	var arr [2]Point
	arr[1].x = 5

	var p *Point
	p = &arr[1]
	var q *Point
	q = p
	q.x = 7
	i32.print(arr[1].x) // prints 7

	var pa [2]*Point
	pa[1] = q
	var parr *[2]Point
	parr = &arr
	i32.print(pa[1].x + parr[1].x) // prints 14
}
```

Indexing a pointer to an array reads through the pointer, as in Go, so
`parr[1]` is the same element as `(*parr)[1]` whatever the element type.
A field of a pointer field reads through the pointer too. `b.p.x` is the
field `x` of the struct that `b.p` points to, and `b.p` alone is the
pointer.

### [Escape Analysis](#table-of-contents)

Consider the following example:

```cx
package main

func foo () (pNum *i32) {
	var num i32
	num = 5

	pNum = &num // num escapes, so it lives on the heap
}

func stackDestroyer () {
	var arr [5]i32
}

func main () {
	var pNum *i32
	pNum = foo()

	stackDestroyer()

	i32.print(*pNum) // prints 5
}
```

If `num` lived in `foo`'s stack frame, `stackDestroyer` would reuse that
memory for `arr` and the program would print `0`. The compiler sees that
the address of `num` outlives the call, so `num` lives in an object on
the heap instead, and `pNum` keeps pointing at the `5`.

The compiler decides this for every local, input and `&T{}` literal of a
function before it compiles the function, following Go's rule. The
address of a local escapes when it is

* returned, or stored in an output,
* stored in a global,
* stored through a pointer, or in a slice element, unless the pointer
  only ever points to locals of the same function,
* passed to `append`, `insert` or `copy`,
* passed to a function that keeps it, or that returns it to a variable
  that escapes,
* stored in a variable that lives longer than one pass of the loop that
  declares the local, so each pass gets its own object.

A function compiled earlier tells the compiler whether it keeps each
input. A call to a function compiled later, or to the function itself,
counts as keeping the pointer.

Not every value being pointed to is moved to the heap. A local whose
address does not escape stays in the stack frame, so no allocation is
added for it. For example, let's re-examine one of the examples
presented in the *Pointers* section:

```cx
package main

func foo (num *i32) {
	*num = 10
}

func main () {
	var num i32
	num = 15
	i32.print(num) // prints 15
	foo(&num)
	i32.print(num) // prints 10
}
```

`foo` does not keep the pointer it receives, so `num` stays in `main`'s
frame and `foo` changes it there.

An escaped local is allocated when its declaration runs. Its frame slot
holds the address of its heap object, and every use of the local reads
through that slot. The garbage collector keeps the object while a
pointer to it is reachable. A program with an escaped local runs on the
old loop, because the flat VM does not use the heap. An output whose
address is taken, as `p = &r` for an output `r`, stays in the frame.

## [Control Flow](#table-of-contents)

Once we have the appropriate data structures for our program, we'll
now need to process them. In order to do so, we need to have access to
some control flow structures.

### [Functions](#table-of-contents)

Functions are used to encapsulate routines that we plan to be
frequently calling. In addition to encapsulating a series of
expressions and statements, we can also receive input parameters and
return output parameters, just like mathematical functions.

```cx
package main

import "sys/string"

func main () {
	var players []str
	players = []str{"Richard", "Mario", "Edward"}

	string.PrintLine("=======================")
	string.PrintLine(string.Concatenation("Name: \t", players[0]))
	string.PrintLine("=======================")

	string.PrintLine("=======================")
	string.PrintLine(string.Concatenation("Name: \t", players[1]))
	string.PrintLine("=======================")

	string.PrintLine("=======================")
	string.PrintLine(string.Concatenation("Name: \t", players[2]))
	string.PrintLine("=======================")
}
```

For example, if we see the code above we'll notice that it seems
repetitive. We can fix this by creating a function, as seen in the
example below.


```cx
package main

import "sys/string"

func drawBox (player str) {
	string.PrintLine("=======================")
	string.PrintLine(string.Concatenation("Name: \t", player))
	string.PrintLine("=======================")
}

func main () {
	var players []str
	players = []str{"Richard", "Mario", "Edward"}

	drawBox(players[0])
	drawBox(players[1])
	drawBox(players[2])
}
```

### [Methods](#table-of-contents)

Methods are useful when we want to associate a particular function to
a particular custom type (associating functions to primitive types is
not allowed). This allows us to create more readable code.

```cx
package main

import "sys/string"

type Player struct {
	Name str
	HP i32
	Mana i32
	Lives i32
}

type Monster struct {
	Name str
	HP i32
	Mana i32
}

func (player Player) draw () {
	string.PrintLine(sprintf("\n\tName: \t%s\n\tHP: \t%d\n\tMana: \t%d\n\tLives: \t%d\n\n%s",
		player.Name,
		player.HP,
		player.Mana,
		player.Lives,
		`
─▄████▄▄░
▄▀█▀▐└─┐░░
█▄▐▌▄█▄┘██
└▄▄▄▄▄┘███
██▒█▒███▀`))
}

func (monster Monster) draw () {
	string.PrintLine(sprintf("\n\tName: \t%s\n\tHP: \t%d\n\tMana: \t%d\n\n%s",
		monster.Name,
		monster.HP,
		monster.Mana,
		`
╲╲╭━━━━╮╲╲
╭╮┃▆┈┈▆┃╭╮
┃╰┫▽▽▽▽┣╯┃
╰━┫△△△△┣━╯
╲╲┃┈┈┈┈┃╲╲
╲╲┃┈┏┓┈┃╲╲
▔▔╰━╯╰━╯▔▔`))
}

func main () {
	var player Player
	player.Name = "Mario"
	player.HP = 10
	player.Mana = 10
	player.Lives = 3

	player.draw()

	var monster Monster
	monster.Name = "Domo-kun"
	monster.HP = 7
	monster.Mana = 4

	monster.draw()
}
```

The example above shows us how we can create two versions of the
function `draw`, and the behavior of each depends on the custom type
that we're using to call it.

### [If and if/else](#table-of-contents)

*if* and *if/else* statements are used to execute a block of
 instructions only if certain condition is true or false. Behind the
 scenes, *if* and *if/else* statements are parsed to a series of
 `jmp` instructions internally. For example, in the case of an *if*
 statement, we will jump *0* instructions if certain predicate is
 true, and it will jump *n* instructions if the predicate is false,
 where *n* is the number of instructions in the *if* block of
 instructions.

```cx
package main

import "sys/string"

func main () {
	if true {
		string.PrintLine("hi")
	}
	string.PrintLine("bye")
}
```

```
0.- Package: main
	Imports
		0.- Import: string
	Functions
		0.- Function: main () ()
			0.- Line: 8: }
			1.- Expression: jmp(*lit bool)
			2.- Line: 7: string.PrintLine("hi")
			3.- Expression: string.PrintLine(*lit str)
			4.- Line: 8: }
			5.- Expression: jmp(*lit bool)
			6.- Line: 9: string.PrintLine("bye")
			7.- Expression: string.PrintLine(*lit str)
```

In the two code snippets above, the second printed by `cx -ast`, we
can see how an if statement is
translated by the parser to a set of two `jmp` instructions. These
`jmp` instructions have some meta data in them that is not shown in
the second snippet: how many lines to jump if its predicate is true
and how many lines to jump if the predicate is false. `jmp` is
not meant to be used by CX programmers (it's only part of the CX base
language), so you don't need to worry about it.

```cx
package main

import "sys/string"

type Player struct {
	Name str
	HP i32
	Mana i32
	Lives i32
}

type Monster struct {
	Name str
	HP i32
	Mana i32
}

func main () {
	var player Player
	player.Name = "Mario"
	player.HP = 10
	player.Mana = 10
	player.Lives = 3

	var monster Monster
	monster.Name = "Domo-kun"
	monster.HP = 7
	monster.Mana = 4

	if player.HP < 5 {
		string.PrintLine("===DANGER!===")
	} else {
		string.PrintLine("===YOU CAN DO IT!===")
	}

	if monster.HP < 10 {
		string.PrintLine(sprintf("===%s is bleeding!===", monster.Name))
	}

	if monster.HP < 5 {
		string.PrintLine(sprintf("===%s is dying!===", monster.Name))
	}

	if monster.HP == 0 {
		string.PrintLine(sprintf("===%s is dead!===", monster.Name))
	}
}
```

Continuing with the example from the previous section (to some
extent), let's use *if* and *if/else* statements to determine what
messages are going to be displayed to the user. These messages
represent the state of the player or the monster, depending on their
hit points (HP).

### [For loop](#table-of-contents)

The *for* loop is the only looping mechanism in CX. Just like *if* and
*if/else* statements are constructed using `jmp` statements, *for*
loop statements are also constructed the same way.

```cx
package main

func main () {
	for c := 0; c < 10; c++ {
		i32.print(c)
	}
}
```

```
0.- Package: main
	Functions
		0.- Function: main () ()
			0.- Line: 4: for c := 0;
			1.- Declaration: 1:c i32
			2.- Expression: 1:c i32 = identity(*lit i32)
			3.- Line: 4: for c := 0; c < 10;
			4.- Expression: *tmp_0 bool = i32.lt(1:c i32, *lit i32)
			5.- Line: 6: }
			6.- Expression: jmp(*tmp_0 bool)
			7.- Line: 5: i32.print
			8.- Expression: i32.print(1:c i32)
			9.- Line: 4: for c := 0; c < 10; c++
			10.- Expression: 1:c i32 = i32.add(1:c i32, *lit i32)
			11.- Line: 6: }
			12.- Expression: jmp(*lit bool)
```

The code snippets above, the second printed by `cx -ast`, illustrate
how a *for* loop that counts from 0
to 9 is translated to a set of of `jmp` instructions.

```cx
package main

import "sys/string"

type Player struct {
	Name str
	HP i32
	Mana i32
	Lives i32
}

type Monster struct {
	Name str
	HP i32
	Mana i32
}

func (player Player) draw () {
	string.PrintLine(sprintf("\n\tName: \t%s\n\tHP: \t%d\n\tMana: \t%d\n\tLives: \t%d\n\n%s",
		player.Name,
		player.HP,
		player.Mana,
		player.Lives,
		`
─▄████▄▄░
▄▀█▀▐└─┐░░
█▄▐▌▄█▄┘██
└▄▄▄▄▄┘███
██▒█▒███▀`))
}

func (monster Monster) draw () {
	string.PrintLine(sprintf("\n\tName: \t%s\n\tHP: \t%d\n\tMana: \t%d\n\n%s",
		monster.Name,
		monster.HP,
		monster.Mana,
		`
╲╲╭━━━━╮╲╲
╭╮┃▆┈┈▆┃╭╮
┃╰┫▽▽▽▽┣╯┃
╰━┫△△△△┣━╯
╲╲┃┈┈┈┈┃╲╲
╲╲┃┈┏┓┈┃╲╲
▔▔╰━╯╰━╯▔▔`))
}

func playerAttack (player Player, cmd str, monster *Monster) {
	if bool.or(string.EqualityByByteValue(cmd, "M"), string.EqualityByByteValue(cmd, "m")) {
		var dmg i32
		dmg = i32.rand(1, 4)
		(*monster).HP = (*monster).HP - dmg
		printf("'%s' suffered a magic attack. Lost %d HP. New HP is %d\n", (*monster).Name, dmg, (*monster).HP)
	} else {
		var dmg i32
		dmg = i32.rand(1, 2)
		(*monster).HP = (*monster).HP - dmg
		printf("'%s' suffered a physical attack. Lost %d HP. New HP is %d\n", (*monster).Name, dmg, (*monster).HP)
	}
}

func monsterAttack (monster Monster, cmd str, player *Player) {
	var dmg i32
	dmg = i32.rand(1, 5)
	(*player).HP = (*player).HP - dmg

	printf("'%s' suffered a physical attack. Lost %d HP. New HP is %d\n", (*player).Name, dmg, (*player).HP)
}

func battleStatus (player Player, monster Monster) {
	if player.HP < 5 {
		string.PrintLine("===DANGER!===")
	} else {
		string.PrintLine("===YOU CAN DO IT!===")
	}

	if player.HP == 0 {
		string.PrintLine("===YOU DIED===")
	}

	if monster.HP < 10 && monster.HP >= 5 {
		string.PrintLine(sprintf("===%s is bleeding!===", monster.Name))
	}

	if monster.HP < 5 && monster.HP > 0 {
		string.PrintLine(sprintf("===%s is dying!===", monster.Name))
	}

	if monster.HP <= 0 {
		string.PrintLine(sprintf("===%s is dead!===", monster.Name))
	}
}

func main () {
	var player Player
	player.Name = "Mario"
	player.HP = 10
	player.Mana = 10
	player.Lives = 3

	var monster Monster
	monster.Name = "Domo-kun"
	monster.HP = 7
	monster.Mana = 4

	player.draw()
	monster.draw()

	for player.HP > 0 && monster.HP > 0 {
		printf("Command? (M)agic; (P)hysical; (E)xit\t")
		var cmd str
		cmd = read()

		if string.EqualityByByteValue(cmd, "E") || string.EqualityByByteValue(cmd, "e") {
			return
		}

		player.draw()
		monster.draw()

		playerAttack(player, cmd, &monster)
		monsterAttack(monster, cmd, &player)
		battleStatus(player, monster)
	}
}
```

Lastly, we can see how we use a *for* loop to create something similar
to a REPL for the program that we have been building in the last few sections.

The attacks could also be methods, such as
`func (player Player) attack (cmd str, monster *Monster)` called as
`player.attack(cmd, &monster)`.

### [Go-to](#table-of-contents)

The last control flow mechanism is *go-to*, which is achieved through
the `goto` statement.


```cx
package main

import "sys/string"

func main () (out i32) {
beginning:
	printf("What animal do you like the most: (C)at; (D)og; (P)igeon\n")

	var cmd str
	cmd = read()

	if string.EqualityByByteValue(cmd, "C") || string.EqualityByByteValue(cmd, "c") {
		goto cat
	}

	if string.EqualityByByteValue(cmd, "D") || string.EqualityByByteValue(cmd, "d") {
		goto dog
	}

	if string.EqualityByByteValue(cmd, "P") || string.EqualityByByteValue(cmd, "p") {
		goto pigeon
	}

cat:
	string.PrintLine("meow")
	goto beginning
dog:
	string.PrintLine("woof")
	goto beginning
pigeon:
	string.PrintLine("tweet")
	goto beginning
}
```

The program above creates an infinite loop by using `goto`s. The loop
will keep asking the user to input commands, and will jump to certain
expression depending on the command.

## [Affordances](#table-of-contents)

![CX Affordances](docs/images/affordances.png)

The diagram shows the design. See [Affordances](#affordances) above for
why they cannot be used yet.

# [Native Functions](#table-of-contents)

## [Type-inferenced Functions](#table-of-contents)

CX has a small set of functions that are not associated to a single
type signature. For example, instead of using `i32.add` to add two
32-bit integers, you can use the generalized `add`
function. Furthermore, whenever you use arithmetic operators, such as
`+`, `-` or `%`, these are translated to their corresponding
"type-inferenced" function, e.g. `num = 5 + 5` is translated to `num =
i32.add(5, 5)`. These native functions still follow CX's philosophy of
having a strict typing system, as the types of the arguments sent to
these native functions must be the same.

Note that after listing a group of similar "type-inferenced" functions
below, we list the compatible types for the corresponding functions.

### `eq`
### `uneq`

Note: the preceding functions only work with arguments of type
*bool*, any integer type, *f32*, *f64* or a pointer. Two pointers
are equal when they hold the same address. Strings are compared with
`string.EqualityByByteValue` or `string.EqualityByHeapPointer`, see
[Language versions](#language-versions).

#### Example

```cx
package main

import "sys/string"

func main () {
	bool.print(eq(5, 5))
	bool.print(5 == 5) // alternative

	bool.print(uneq(3L, 4L))
	bool.print(3L != 4L) // alternative

	bool.print(string.EqualityByByteValue("hihi", "byebye"))
}
```

### `lt`
### `gt`
### `lteq`
### `gteq`

Note: the preceding functions only work with arguments of an integer
type, *f32* or *f64*. Booleans and pointers cannot be ordered.
Strings are ordered with `string.OrderingByByteValue`, which returns -1, 0
or 1, see [Language versions](#language-versions).

#### Example

```cx
package main

import "sys/string"

func main () {
	bool.print(lt(3B, 4B))
	bool.print(3B < 4B) // alternative

	bool.print(string.OrderingByByteValue("hello", "hi!") > 0)

	bool.print(lteq(5.3D, 5.3D))
	bool.print(5.3D <= 5.3D) // alternative

	bool.print(gteq(10L, 3L))
	bool.print(10L >= 3L) // alternative
}
```

### `bitand`
### `bitor`
### `bitxor`
### `bitclear`
### `bitshl`
### `bitshr`

Note: the preceding functions only work with arguments of an integer
type: *i8*, *i16*, *i32*, *i64*, *ui8*, *ui16*, *ui32* or *ui64*.

#### Example

```cx
package main

func main () {
	i32.print(bitand(5, 1))
	i32.print(5 & 1) // alternative

	i64.print(bitor(3L, 2L))
	i64.print(3L | 2L) // alternative

	i32.print(bitxor(10, 2))
	i32.print(10 ^ 2) // alternative

	i64.print(bitclear(5L, 2L))
	i64.print(5L &^ 2L) // alternative

	i32.print(bitshl(2, 3))
	i32.print(2 << 3) // alternative

	i32.print(bitshr(16, 3))
	i32.print(16 >> 3) // alternative
}
```

### `add`
### `sub`
### `mul`
### `div`

Note: the preceding functions only work with arguments of type
an integer type, *f32* or *f64*.

#### Example

```cx
package main

func main () {
	i8.print(add(5B, 10B))
	i8.print(5B + 10B) // alternative

	i32.print(sub(3, 7))
	i32.print(3 - 7) // alternative

	i64.print(mul(4L, 5L))
	i64.print(4L * 5L) // alternative

	f32.print(div(4.3, 2.1))
	f32.print(4.3 / 2.1) // alternative
}
```

### `mod`

Note: the preceding function works with arguments of an integer type,
*f32* or *f64*.

#### Example

```cx
package main

func main () {
	i8.print(mod(5B, 3B))
	i8.print(5B % 3B) // alternative
}
```

### `len`

Note: the preceding function only works with arguments of type *str*,
*arrays* or *slices*.

#### Example

```cx
package main

func main () {
	var string str
	var array [5]i32
	var slice []i32

	string = "this should print 20"
	array = [5]i32{1, 2, 3, 4, 5}
	slice = []i32{10, 20, 30}

	i32.print(len(string)) // prints 20
	i32.print(len(array)) // prints 5
	i32.print(len(slice)) // prints 3
}
```

### `printf`

Note: the preceding function requires a format *str* as its first
argument, followed by any number of arguments of type *str*, *i32*,
*i64*, *f32* or *f64*. The format string recognizes the following
directives: `%s` for strings, `%d` for integers and `%f` for floating
point numbers.

`%s` and `%v` print the bytes of a str as they are, and `%%` prints
`%`. Escapes are read only in string literals, so a `\n` that a str
holds as two bytes prints as `\n`.

`%v` prints any value. A pointer prints as Go prints it, in hexadecimal
with `0x`, a nil pointer as `<nil>` and a pointer to a struct as
`&{x: 1, y: 2}`. Structs print their field names, so `%+v` prints as
`%v`. Slices and arrays print as `[1, 2]`, and a nil slice as `[]`.

#### Example

```cx
package main

func main () {
	var name str
	var age i32
	var wrongPI f32
	var error f64

	name = "Richard"
	age = 14
	wrongPI = 3.16
	error = 0.0000000000000001D

	printf("Hello, %s. My name is %s. I see that you calculated the value of PI wrong (%f). I think this is not so bad, considering your young age of %d. When I was %d years old, I remember I miscalculated it, too (I got %f as a result, using a numerical method). If you are using a numerical method, please consider reaching an error lower than %f to get an acceptable result, and not a ridiculous value such as %f. \n\nBest regards!\n", name, "Edward", wrongPI, age, 25, 3.1417, error, 0.1)
}
```

### `sprintf`

Note: the preceding function requires a format *str* as its first
argument, followed by any number of arguments of type *str*, *i32*,
*i64*, *f32* or *f64*. The format string recognizes the following
directives: `%s` for strings, `%d` for integers and `%f` for floating
point numbers.

```cx
package main

import "sys/string"

func main () {
	var reply str
	var name str
	var title str

	name = "Edward"
	title = "Richard 8 PI"

	reply = sprintf("Thank you for contacting our technical support, %s. We see that you are having trouble with our video game titled '%s', targetted to kids under age %d. If you provide us with your parents e-mail address, we'll be glad to help you!", name, title, 14)

	string.PrintLine(reply)
}
```

## [Slice Functions](#table-of-contents)

### `append`

#### Example

```cx
package main

func main () {
	var slc1 []i32
	slc1 = append(slc1, 1)
	slc1 = append(slc1, 2)

	var slc2 []i32
	slc2 = append(slc1, 3)
	slc2 = append(slc2, 4)

	i32.print(len(slc1)) // prints 2
	i32.print(len(slc2)) // prints 4
}
```

## [Input/Output Functions](#table-of-contents)

The following functions are used to handle input from the user and to
print output to a terminal.

### `read`

`read` reads one line from standard input and returns it without its line
ending. One reader serves the whole run, so lines piped in at once are
read in order. At the end of input `read` returns `""`.

The end of input result is provisional. A second `ok bool` output that is
false at the end of input is under discussion, so a program cannot yet
tell an empty line from the end of input.

#### Example

```cx
package main

import "sys/string"

func main () {
	var password str

	printf("What's the password, kid? ")
	password = read()

	for bool.not(string.EqualityByByteValue(password, "123")) {
		string.PrintLine("Wrong, but you'll get another chance.")
		printf("What's the password, kid? ")
		password = read()
	}

	string.PrintLine("Welcome back.")
}
```

### `i8.print`
### `ui8.print`
### `bool.print`
### `string.PrintLine`
### `i32.print`
### `i64.print`
### `f32.print`
### `f64.print`
### `printf`

Every integer type has its own print function, from `i8.print` to
`ui64.print`.

#### Example

```cx
package main

import "sys/string"

func main () {
    i8.print(5B)
    ui8.print(5UB)
    bool.print(true)
    string.PrintLine("Hello!")
    i32.print(5)
    i64.print(5L)
    f32.print(5.0)
    f64.print(5.0D)
    printf("For a better example, check section Type-inferenced Functions\n")
}
```

## [Parse Functions](#table-of-contents)

All parse functions follow the same pattern: `XXX.YYY` where *XXX* is
the receiving type and *YYY* is the target type. You can read these
functions as "parse XXX to YYY". Every numeric type can be parsed to
every other numeric type and to *str*, so besides the ones listed
below there are, for example, `i16.f64` and `ui32.i8`. Strings are
parsed with the `string.Parse...` functions of `sys/string`.

### `ui8.str`
### `ui8.i32`
### `ui8.i64`
### `ui8.f32`
### `ui8.f64`

#### Example

```cx
package main

import "sys/string"

func main () {
	var b ui8
	b = 30UB

	string.PrintLine(string.Concatenation("Hello, ", ui8.str(b)))
	i32.print(5 + ui8.i32(b))
	i64.print(10L + ui8.i64(b))
	f32.print(33.3 + ui8.f32(b))
	f64.print(50.111D + ui8.f64(b))
}
```

### `i32.ui8`
### `i32.str`
### `i32.i64`
### `i32.f32`
### `i32.f64`

#### Example

```cx
package main

import "sys/string"

func main () {
	var num i32
	num = 43

	string.PrintLine(string.Concatenation("Hello, ", i32.str(num)))
	ui8.print(5UB + i32.ui8(num))
	i64.print(10L + i32.i64(num))
	f32.print(33.3 + i32.f32(num))
	f64.print(50.111D + i32.f64(num))
}
```

### `i64.ui8`
### `i64.str`
### `i64.i32`
### `i64.f32`
### `i64.f64`

#### Example

```cx
package main

import "sys/string"

func main () {
	var num i64
	num = 43L

	string.PrintLine(string.Concatenation("Hello, ", i64.str(num)))
	ui8.print(5UB + i64.ui8(num))
	i32.print(10 + i64.i32(num))
	f32.print(33.3 + i64.f32(num))
	f64.print(50.111D + i64.f64(num))
}
```

### `f32.ui8`
### `f32.str`
### `f32.i32`
### `f32.i64`
### `f32.f64`

#### Example

```cx
package main

import "sys/string"

func main () {
	var num f32
	num = 43.33

	string.PrintLine(string.Concatenation("Hello, ", f32.str(num)))
	ui8.print(5UB + f32.ui8(num))
	i32.print(33 + f32.i32(num))
	i64.print(10L + f32.i64(num))
	f64.print(50.111D + f32.f64(num))
}
```

### `f64.ui8`
### `f64.str`
### `f64.i32`
### `f64.i64`
### `f64.f32`

#### Example

```cx
package main

import "sys/string"

func main () {
	var num f64
	num = 43.33D

	string.PrintLine(string.Concatenation("Hello, ", f64.str(num)))
	ui8.print(5UB + f64.ui8(num))
	i32.print(33 + f64.i32(num))
	i64.print(10L + f64.i64(num))
	f32.print(50.111 + f64.f32(num))
}
```

### `string.ParseDecimalUI8`
### `string.ParseDecimalI32`
### `string.ParseDecimalI64`
### `string.ParseFloatF32`
### `string.ParseFloatF64`

#### Example

```cx
package main

import "sys/string"

func main () {
	var num str
	num = "33"

	ui8.print(5UB + string.ParseDecimalUI8(num))
	i32.print(33 + string.ParseDecimalI32(num))
	i64.print(10L + string.ParseDecimalI64(num))
	f32.print(50.111 + string.ParseFloatF32(num))
	f64.print(50.111D + string.ParseFloatF64(num))
}
```

## [Unit Testing](#table-of-contents)

The `assert` function is used to test the value of an expression
against another value. This function is useful to test that a
package is working as intended.

### assert

#### Example

```cx
package main

func foo() (res str) {
    res = "Working well"
}

func main () {
    var results []bool

    results = append(results, assert(5 + 5, 10, "Something went wrong with 5 + 5"))
    results = append(results, assert(foo(), "Working well", "Something went wrong with foo()"))

    var successfulTests i32
	for c := 0; c < len(results); c++ {
		if results[c] {
			successfulTests = successfulTests + 1
		}
	}

    printf("%d tests were performed\n", len(results))
    printf("%d were successful\n", successfulTests)
    printf("%d failed\n", len(results) - successfulTests)
}
```

## [`bool` Type Functions](#table-of-contents)

### `bool.print`
### `bool.eq`
### `bool.uneq`
### `bool.not`
### `bool.or`
### `bool.and`

#### Example

```cx
package main

func main () {
	bool.print(bool.eq(true, true))
	bool.print(bool.uneq(false, true))
	bool.print(bool.not(false))
	bool.print(bool.or(true, false))
	bool.print(bool.and(true, true))
}
```

## [`sys/string` Package Functions](#table-of-contents)

Built in, import with `import "sys/string"`. Using it without the import
is a compile error. Positions and lengths count bytes, not characters.

### `string.ByteLength (s str) (i32)`
### `string.Concatenation (a str, b str) (str)`
### `string.SubstringByByteRange (s str, start i32, end i32) (str)`
### `string.ByteIndexOfFirstSubstring (s str, sub str) (i32)`
### `string.ByteIndexOfLastSubstring (s str, sub str) (i32)`
### `string.TrimSurroundingWhitespace (s str) (str)`
### `string.ParseDecimalI8`, `I16`, `I32`, `I64`, `UI8`, `UI16`, `UI32`, `UI64` `(s str)`
### `string.ParseFloatF32`, `string.ParseFloatF64` `(s str)`
### `string.PrintLine (s str) ()`
### `string.BytesOfString (s str) ([]ui8)`
### `string.StringOfBytes (b []ui8) (str)`

`BytesOfString` returns a new `[]ui8` of the bytes of `s` and
`StringOfBytes` a new `str` of the bytes of `b`, as `[]byte(s)` and
`string(b)` do in Go. `SubstringByByteRange` returns bytes `start` up to but not including
`end`. The index functions return -1 when `sub` is absent. The parse
functions raise a runtime error when `s` does not parse.

#### Example

```cx
package main

import "sys/string"

func main () {
	string.PrintLine(string.Concatenation("Hello, ", "World!"))
}
```

### `string.EqualityByByteValue (str, str) (bool)`
### `string.EqualityByHeapPointer (str, str) (bool)`
### `string.OrderingByByteValue (str, str) (i32)`

`string.EqualityByByteValue` reports whether two strings hold the same bytes.
`string.EqualityByHeapPointer` reports whether both values refer to the same
string object, which is true after an assignment such as `y = x` but
false for two strings built separately. `string.OrderingByByteValue` returns
-1, 0 or 1 by byte order. These are the only way to compare strings,
see [Language versions](#language-versions).

#### Example

```cx
package main

import "sys/string"

func main () {
	var x str
	var y str
	x = "apple"
	y = x
	bool.print(string.EqualityByByteValue(x, string.Concatenation("app", "le"))) // true
	bool.print(string.EqualityByHeapPointer(x, y))                      // true
	i32.print(string.OrderingByByteValue(x, "banana"))                // -1
}
```

## [`i32` Type Functions](#table-of-contents)

The following functions are of general purpose and are restricted to
work with data structures of type *i32* where it makes sense. There are
no `log`, `pow` or `sqrt` functions for integer types. Parse to *f32*
or *f64* and use theirs.

### `i32.print`
### `i32.add`
### `i32.sub`
### `i32.mul`
### `i32.div`
### `i32.mod`
### `i32.abs`

#### Example

```cx
package main

func main () {
	i32.print(i32.add(5, 7))
	i32.print(i32.sub(6, 3))
	i32.print(i32.mul(4, 8))
	i32.print(i32.div(15, 3))
	i32.print(i32.mod(5, 3))
	i32.print(i32.abs(-13))
}
```

### `i32.gt`
### `i32.gteq`
### `i32.lt`
### `i32.lteq`
### `i32.eq`
### `i32.uneq`

#### Example

```cx
package main

func main () {
	bool.print(i32.gt(5, 3))
	bool.print(i32.gteq(3, 8))
	bool.print(i32.lt(4, 3))
	bool.print(i32.lteq(8, 6))
	bool.print(i32.eq(-9, -9))
	bool.print(i32.uneq(3, 3))
}
```

### `i32.bitand`
### `i32.bitor`
### `i32.bitxor`
### `i32.bitclear`
### `i32.bitshl`
### `i32.bitshr`

#### Example

```cx
package main

func main () {
	i32.print(i32.bitand(2, 5))
	i32.print(i32.bitor(8, 3))
	i32.print(i32.bitxor(3, 9))
	i32.print(i32.bitclear(4, 4))
	i32.print(i32.bitshl(5, 9))
	i32.print(i32.bitshr(1, 6))
}
```

### `i32.max`
### `i32.min`

#### Example

```cx
package main

func main () {
	i32.print(i32.max(2, 5))
	i32.print(i32.min(10, 3))
}
```

### `i32.rand`

#### Example

```cx
package main

func main () {
	i32.print(i32.rand(0, 100))
}
```

## [`i64` Type Functions](#table-of-contents)

The following functions are of general purpose and are restricted to
work with data structures of type *i64* where it makes sense. There are
no `log`, `pow` or `sqrt` functions for integer types. Parse to *f32*
or *f64* and use theirs.

### `i64.print`
### `i64.add`
### `i64.sub`
### `i64.mul`
### `i64.div`
### `i64.mod`
### `i64.abs`

#### Example

```cx
package main

func main () {
	i64.print(i64.add(5L, 7L))
	i64.print(i64.sub(6L, 3L))
	i64.print(i64.mul(4L, 8L))
	i64.print(i64.div(15L, 3L))
	i64.print(i64.mod(5L, 3L))
	i64.print(i64.abs(-13L))
}
```

### `i64.gt`
### `i64.gteq`
### `i64.lt`
### `i64.lteq`
### `i64.eq`
### `i64.uneq`

#### Example

```cx
package main

func main () {
	bool.print(i64.gt(5L, 3L))
	bool.print(i64.gteq(3L, 8L))
	bool.print(i64.lt(4L, 3L))
	bool.print(i64.lteq(8L, 6L))
	bool.print(i64.eq(-9L, -9L))
	bool.print(i64.uneq(3L, 3L))
}
```

### `i64.bitand`
### `i64.bitor`
### `i64.bitxor`
### `i64.bitclear`
### `i64.bitshl`
### `i64.bitshr`

#### Example

```cx
package main

func main () {
	i64.print(i64.bitand(2L, 5L))
	i64.print(i64.bitor(8L, 3L))
	i64.print(i64.bitxor(3L, 9L))
	i64.print(i64.bitclear(4L, 4L))
	i64.print(i64.bitshl(5L, 9L))
	i64.print(i64.bitshr(1L, 6L))
}
```

### `i64.max`
### `i64.min`

#### Example

```cx
package main

func main () {
	i64.print(i64.max(2L, 5L))
	i64.print(i64.min(10L, 3L))
}
```

### `i64.rand`

#### Example

```cx
package main

func main () {
	i64.print(i64.rand(0L, 100L))
}
```

## [`f32` Type Functions](#table-of-contents)

The following functions are of general purpose and are restricted to
work with data structures of type *f32* where it makes sense.

### `f32.print`
### `f32.add`
### `f32.sub`
### `f32.mul`
### `f32.div`
### `f32.abs`

#### Example

```cx
package main

func main () {
	f32.print(f32.add(5.3, 10.5))
	f32.print(f32.sub(3.2, 6.7))
	f32.print(f32.mul(-7.9, -7.1))
	f32.print(f32.div(10.3, 2.4))
	f32.print(f32.abs(-3.14159))
}
```

### `f32.log`
### `f32.log2`
### `f32.log10`
### `f32.pow`
### `f32.sqrt`

#### Example

```cx
package main

func main () {
	f32.print(f32.log(2.3))
	f32.print(f32.log2(3.4))
	f32.print(f32.log10(3.0))
	f32.print(f32.pow(-5.3, 2.0))
	f32.print(f32.sqrt(4.0))
}
```

### `f32.sin`
### `f32.cos`

#### Example

```cx
package main

func main () {
	f32.print(f32.sin(1.0))
	f32.print(f32.cos(2.0))
}
```

### `f32.gt`
### `f32.gteq`
### `f32.lt`
### `f32.lteq`
### `f32.eq`
### `f32.uneq`

#### Example

```cx
package main

func main () {
	bool.print(f32.gt(5.3, 3.1))
	bool.print(f32.gteq(3.7, 1.9))
	bool.print(f32.lt(2.4, 5.5))
	bool.print(f32.lteq(8.4, 3.2))
	bool.print(f32.eq(10.3, 10.3))
	bool.print(f32.uneq(8.9, 3.3))
}
```

### `f32.max`
### `f32.min`

#### Example

```cx
package main

func main () {
	f32.print(f32.max(3.3, 4.2))
	f32.print(f32.min(5.8, 9.9))
}
```

## [`f64` Type Functions](#table-of-contents)

The following functions are of general purpose and are restricted to
work with data structures of type *f64* where it makes sense.

### `f64.print`
### `f64.add`
### `f64.sub`
### `f64.mul`
### `f64.div`
### `f64.abs`

#### Example

```cx
package main

func main () {
	f64.print(f64.add(5.3D, 10.5D))
	f64.print(f64.sub(3.2D, 6.7D))
	f64.print(f64.mul(-7.9D, -7.1D))
	f64.print(f64.div(10.3D, 2.4D))
	f64.print(f64.abs(-3.14159D))
}
```

### `f64.log`
### `f64.log2`
### `f64.log10`
### `f64.pow`
### `f64.sqrt`

#### Example

```cx
package main

func main () {
	f64.print(f64.log(2.3D))
	f64.print(f64.log2(3.4D))
	f64.print(f64.log10(3.0D))
	f64.print(f64.pow(-5.3D, 2.0D))
	f64.print(f64.sqrt(4.0D))
}
```

### `f64.sin`
### `f64.cos`

#### Example

```cx
package main

func main () {
	f64.print(f64.sin(1.0D))
	f64.print(f64.cos(2.0D))
}
```

### `f64.gt`
### `f64.gteq`
### `f64.lt`
### `f64.lteq`
### `f64.eq`
### `f64.uneq`

#### Example

```cx
package main

func main () {
	bool.print(f64.gt(5.3D, 3.1D))
	bool.print(f64.gteq(3.7D, 1.9D))
	bool.print(f64.lt(2.4D, 5.5D))
	bool.print(f64.lteq(8.4D, 3.2D))
	bool.print(f64.eq(10.3D, 10.3D))
	bool.print(f64.uneq(8.9D, 3.3D))
}
```

### `f64.max`
### `f64.min`

#### Example

```cx
package main

func main () {
	f64.print(f64.max(3.3D, 4.2D))
	f64.print(f64.min(5.8D, 9.9D))
}
```

## [`time` Package Functions](#table-of-contents)

Built in, import with `import "sys/time"`.

The functions in the `time` package deal with real-time in your
programs. They are used to measure and stop time. Using them without
the import is a compile error.

### `time.Sleep`
### `time.UnixMilli`
### `time.UnixNano`

#### Example

```cx
package main
import "sys/time"

func main () {
	var start i64
	var end i64

	start = time.UnixMilli()
	time.Sleep(1000)
	end = time.UnixMilli()

	printf("elapsed time in milliseconds: \t%d\n", end - start)

	start = time.UnixNano()
	time.Sleep(1000)
	end = time.UnixNano()

	printf("elapsed time in nanoseconds: \t%d\n", end - start)
}
```

## [`os` Package Functions](#table-of-contents)

Built in, import with `import "sys/os"`.

The `os` package provides functions that serve as an interface to CX's
underlaying operating system.

The file functions, `os.GetWorkingDirectory`, `os.LogFile` and the
CPU profile belong to the system library `file` and `os.Run` to `process`,
so a program calls them with `cx --enable=file` or `--enable=process`. See
[System libraries](#system-libraries).

File paths passed to `os`, `gl`, `al` and `StartCPUProfile` are
confined to the sandbox directory, the working directory unless `cx
--sandbox <dir>` names another. A path must be relative, use `/` as the
separator, contain no NUL byte, be at most 4096 bytes and not leave the
sandbox through `..` or a symlink. A rejected path fails like a missing
file, `os.Open` returns -1 (#433).

### `os.GetWorkingDirectory`
### `os.Open`
### `os.Close`

`os.Open` returns a handle, or -1 when the file cannot be opened.
`os.Close` returns `true` when it closed the handle.

#### Example

```cx
package main
import "sys/os"

func main () {
	var handle i32
	handle = os.Open("testing.cx")

	var closed bool
	closed = os.Close(handle)
	bool.print(closed)
}
```

## [`gl` Package Functions](#table-of-contents)

Built in, import with `import "sys/gl"`. The OpenAL functions are
built in as `sys/al`.

"OpenGL is the premier environment for developing portable,
interactive 2D and 3D graphics applications. Since its introduction in
1992, OpenGL has become the industry's most widely used and supported
2D and 3D graphics application programming interface (API), bringing
thousands of applications to a wide variety of computer
platforms. OpenGL fosters innovation and speeds application
development by incorporating a broad set of rendering, texture
mapping, special effects, and other powerful visualization
functions. Developers can leverage the power of OpenGL across all
popular desktop and workstation platforms, ensuring wide application
deployment." This description was extracted from OpenGL's website
(https://www.opengl.org/).


### `gl.ActiveTexture`
### `gl.AttachShader`
### `gl.Begin`
### `gl.BindAttribLocation`
### `gl.BindBuffer`
### `gl.BindFramebuffer`
### `gl.BindTexture`
### `gl.BindVertexArray`
### `gl.BlendFunc`
### `gl.BufferData`
### `gl.BufferSubData`
### `gl.CheckFramebufferStatus`
### `gl.ClearColor`
### `gl.ClearDepth`
### `gl.Clear`
### `gl.Color3f`
### `gl.Color4f`
### `gl.CompileShader`
### `gl.CreateProgram`
### `gl.CreateShader`
### `gl.CullFace`
### `gl.DeleteBuffers`
### `gl.DeleteFramebuffers`
### `gl.DeleteProgram`
### `gl.DeleteShader`
### `gl.DeleteTextures`
### `gl.DeleteVertexArrays`
### `gl.DepthFunc`
### `gl.DepthMask`
### `gl.DetachShader`
### `gl.Disable`
### `gl.DrawArrays`
### `gl.EnableClientState`
### `gl.EnableVertexAttribArray`
### `gl.Enable`
### `gl.End`
### `gl.FramebufferTexture2D`
### `gl.Frustum`
### `gl.GenBuffers`
### `gl.GenFramebuffers`
### `gl.GenTextures`
### `gl.GenVertexArrays`
### `gl.GetAttribLocation`
### `gl.GetError`
### `gl.GetShaderiv`
### `gl.GetTexLevelParameteriv`
### `gl.Hint`
### `gl.Init`
### `gl.Lightfv`
### `gl.LinkProgram`
### `gl.LoadIdentity`
### `gl.MatrixMode`
### `gl.NewTexture`
### `gl.Normal3f`
### `gl.Ortho`
### `gl.PopMatrix`
### `gl.PushMatrix`
### `gl.Rotatef`
### `gl.Scalef`
### `gl.ShaderSource`
### `gl.TexCoord2d`
### `gl.TexCoord2f`
### `gl.TexEnvi`
### `gl.TexImage2D`
### `gl.TexParameteri`
### `gl.Translatef`
### `gl.Uniform1f`
### `gl.Uniform1i`
### `gl.UseProgram`
### `gl.Vertex2f`
### `gl.Vertex3f`
### `gl.VertexAttribPointerI32`
### `gl.VertexAttribPointer`
### `gl.Viewport`
### `gl.getUniformLocation`

## [`glfw` Package Functions](#table-of-contents)

Built in, import with `import "sys/glfw"`.

"GLFW is an Open Source, multi-platform library for OpenGL, OpenGL ES
and Vulkan development on the desktop. It provides a simple API for
creating windows, contexts and surfaces, receiving input and events."
This description was extracted from GLFW's website (https://www.glfw.org/).

A window is an `i32` handle. `glfw.CreateWindow(width, height, title)`
returns it and every other window function takes it. A callback is the
name of a function, written without quotes, and the compiler resolves it,
so `glfw.SetKeyboardCallback(window, onKey)` calls `onKey` with the
window handle as its first input. A name that is not a function is a
compile error. A handle that was never returned is an assert at run time.

### `glfw.CreateWindow`
### `glfw.GetCursorPos`
### `glfw.GetFramebufferSize`
### `glfw.GetTime`
### `glfw.Init`
### `glfw.MakeContextCurrent`
### `glfw.PollEvents`
### `glfw.SetCursorPosCallback`
### `glfw.SetInputMode`
### `glfw.SetKeyCallback`
### `glfw.SetMouseButtonCallback`
### `glfw.SetShouldClose`
### `glfw.SetWindowPos`
### `glfw.ShouldClose`
### `glfw.SwapBuffers`
### `glfw.SwapInterval`
### `glfw.WindowHint`

## [`gltext` Package Functions](#table-of-contents)

Built in, import with `import "sys/gltext"`.

"The gltext package offers a simple set of text rendering utilities
for OpenGL programs. It deals with TrueType and Bitmap (raster)
fonts. Text can be rendered in various directions (Left-to-right,
right-to-left and top-to-bottom). This allows for correct display of
text for various languages." This description was extracted from
gltext's website (https://github.com/go-gl/gltext).


The `gltext` functions can be used to display character strings on
windows. Different fonts can be used by loading font files using `gltext.LoadTrueType`.

A font is an `i32` handle that `gltext.LoadTrueType` returns, -1 when the
file does not load.

### `gltext.GlyphBounds (i32) (i32, i32)`
### `gltext.LoadTrueType (i32, i32, i32, i32, i32) (i32)`
### `gltext.Metrics (i32, str) (i32, i32)`
### `gltext.NextGlyph (i32, str, i32) (i32, i32, i32, i32, i32, i32, i32)`
### `gltext.Printf`
### `gltext.Texture`
# [Building an executable](#table-of-contents)

`cx build` compiles a program into one executable file that runs without
cx and without the source.

```
cx build hello.cx -o hello
./hello first second
```

Several files and the compile flags of cx are given as they are to cx,
`cx build --cxpath ws main.cx util.cx -o app`. `-o` names the output, on
Windows with `.exe`, and `--core path` the core binary or a directory of
cores, by default the directory of cx, where `make build` builds them.

The output is the core binary with the serialized program and a trailer
appended. The core is a minimum interpreter with no parser. At start up it
reads the program from the end of its own file and runs it, on vm when the
program lowers and on the old loop otherwise. Its command line arguments
reach the program unchanged as `os.Args`, so `./hello ++a` sees what
`cx hello.cx ++a` sees. The run time flags of cx, such as `--executor` and
`-heap-max`, are not recorded and take their defaults.

The core includes the built-in functions, `sys/string`, `sys/regexp`,
`sys/time`, `os.Exit` of `sys/os` and the `LibraryCall` opcode. A
[system library](#system-libraries) is linked only when the program calls
it and `--enable` enables it, so there is one core per set of libraries,
`cx-core` for none, `cx-core-file`, `cx-core-file-net` and so on, and
`cx build` picks the one for the program.

```
cx build --enable=file,net client.cx -o client
```

The built program records the libraries it was built with and runs with
exactly those. A library that is not enabled stays out of it, and a call
into it is the runtime error `CX_RUNTIME_LIBRARY_DISABLED`, as under cx.
The file library resolves paths in the working directory, as `cx` does
without `--sandbox`. A program that calls `gl` or another opcode
outside the core is refused.
Every `cx build` failure exits with `CX_COMPILATION_ERROR` (1). A core with
no program appended, or one whose trailer is damaged, prints a message and
exits with the same code.

The core uses no module outside the Go standard library, except that a
core with the cipher library links the `cipher` package of Skycoin, see
decision D75 in [docs/CipherLibrary.md](docs/CipherLibrary.md#cx-build).

# [Language versions](#table-of-contents)

A language version is a change to what CX source may say or to the opcode
numbers a compiled program uses. A version is the integer `yymmddnnnn`, the
release date and the release number on that day, so `2611200001` is
2026-11-20 release 0001. The current version is
`constants.LanguageVersionCurrent`. It is stored in every serialized
program, and a program serialized under another version is rejected when
it is loaded, so it must be compiled again from source.

## Opcode numbers

Every opcode has a fixed number, listed in number order in
`t1-constants/opcodes.go` and in `docs/opcodes.csv`. The number never
depends on registration order or build tags. A new opcode takes the next
number after the highest in the table.

Each row records three versions. Added is the version the number was
assigned, 0 for opcodes older than language versions. Deprecated is the
version the opcode was removed, 0 while it is live. Reassignable is the
first version in which the number may be given to a new opcode, 0 while
it is live. A deprecated opcode keeps its row and leaves a gap, nothing
registers a handler on its number, and the number is not reused before
its reassignable version.

Reassignable is the deprecated version plus one year, same month and day,
release 0001. Deprecated `2611200001` is reassignable from `2711200001`. A
29 February deprecation moves to 1 March. `t1-constants/opcodes_history.go`
lists every lifecycle event in version order. See
[docs/Conventions.md](docs/Conventions.md).

A renamed opcode keeps its number and its row, which takes the new name.
The old name and the version of the rename are kept in
`OpcodeRenameTable` and in the `former_names` column of the CSV, the
old name can no longer be called, and using it is a compile error that
names the new one. An old name is never given to another opcode.

## Version 2611200001, 2026-11-20

This was called revision 1 before versions were integers.

Strings have no built-in operators. `==`, `!=`, `<`, `>`, `<=`, `>=`,
`+` and every other arithmetic or bitwise operator on `str` operands is
a compile error that names the function to use instead.

* `str.equal_by_value(a, b)` compares bytes.
* `str.equal_by_pointer(a, b)` reports whether both are the same string
  object.
* `str.compare_by_value(a, b)` returns -1, 0 or 1 and replaces the
  ordering operators.
* `str.concat(a, b)` replaces `+`.

The opcodes `str.eq` (559), `str.uneq` (560), `str.lt` (582), `str.lteq`
(583), `str.gt` (584) and `str.gteq` (585) are deprecated, so they cannot
be called by name either. Their numbers stay empty and are reassignable
from version `2711200001`. Every other opcode keeps the number it had
before, `ptr.eq` and `ptr.uneq` are new as 589 and 590, and the cxfx
package opcodes moved to fixed numbers 591 to 791.

Pointers keep `==` and `!=`, which compare addresses, but `<`, `>`, `<=`
and `>=` on pointers are compile errors.

Serialized programs from before this version must be recompiled. See
[skycoin-cx#183](https://github.com/skycoin/skycoin-cx/issues/183).

## Version 2611200002, 2026-11-20

Every built-in library lives under `sys/` and needs an explicit import,
see [Packages](#packages). `import "os"` becomes `import "sys/os"`, and
the same for `time`, `json`, `tcp`, `regexp`, `cipher`, `cx`, `gl`,
`glfw`, `al` and `gltext`. A bare import of a built-in package, an
unknown `sys/` path and a built-in package used without its import are
compile errors.

The `str` functions moved to `sys/string` with verbose camel case names.
Each keeps its opcode number.

| opcode | before | from 2611200002 |
|---|---|---|
| 561 | `str.concat` | `string.Concatenation` |
| 562 to 569 | `str.i8` to `str.ui64` | `string.ParseDecimalI8` to `string.ParseDecimalUI64` |
| 570, 571 | `str.f32`, `str.f64` | `string.ParseFloatF32`, `string.ParseFloatF64` |
| 572 | `str.print` | `string.PrintLine` |
| 573 | `str.substr` | `string.SubstringByByteRange` |
| 574 | `str.index` | `string.ByteIndexOfFirstSubstring` |
| 575 | `str.lastindex` | `string.ByteIndexOfLastSubstring` |
| 576 | `str.trimspace` | `string.TrimSurroundingWhitespace` |
| 586 | `str.equal_by_value` | `string.EqualityByByteValue` |
| 587 | `str.equal_by_pointer` | `string.EqualityByHeapPointer` |
| 588 | `str.compare_by_value` | `string.OrderingByByteValue` |
| 756 | `glfw.func_i32_i32` | `glfw.RegisterCallbackI32I32` |
| 757 | `glfw.call_i32_i32` | `glfw.InvokeCallbackI32I32` |

`string.ByteLength` is new as opcode 792. The generic `len` still works
on strings. Calling an old name is a compile error that names the new
one. Serialized programs from before this version must be recompiled.
See [skycoin-cx#380](https://github.com/skycoin/skycoin-cx/issues/380)
and [skycoin-cx#382](https://github.com/skycoin/skycoin-cx/issues/382).

## Version 2611230001, 2026-11-23

File access, networking and `os.Run` are
[system libraries](#system-libraries), called through the one opcode
`LibraryCall` (795) instead of an opcode each. The source does not change,
`import "sys/os"` and `os.Open(...)` work as before, but the library must
be enabled with `--enable`, for example `cx --enable=file program.cx`.

The opcodes `os.GetWorkingDirectory` to `os.Run` (138 to 189), `json.Open`
to `json.Str` (191 to 200), `StartCPUProfile` and `StopCPUProfile` (201,
202) and `tcp.Dial` to `tcp.Close` (206 to 209) are deprecated and
reassignable from version `2711230001`. `os.Exit` keeps opcode 190.
Serialized programs from before this version must be recompiled. See
[skycoin-cx#492](https://github.com/skycoin/skycoin-cx/issues/492).

## Version 2611230002, 2026-11-23

A function declares the globals it uses with a `global` statement at the
top of its body, `global counter, config.Limit`, including the globals of
its own package. A global of another package is always reached as
`package.name`. A use of a global the function does not declare, a
declared global it does not use and a `global` statement after another
statement are compile errors, see [Global variables](#global-variables).

Shadowing is a compile error. A local, a parameter, a result, a loop
variable or a `:=` name may not have the name of a global, an imported
package, a function or a type of its package, or a variable of an
enclosing block, see [Scopes and shadowing](#scopes-and-shadowing).
`global` is a keyword.

Programs from before this version add the `global` statements and rename
the variables that shadow another name. No opcode changes. Serialized
programs from before this version must be recompiled. See
[skycoin-cx#486](https://github.com/skycoin/skycoin-cx/issues/486).

## Version 2611230003, 2026-11-23

An import brings names into scope with `import . "pkg"`,
`import file . "pkg"`, `import (a, b) "pkg"` and `import file (a, b)
"pkg"`, see [Imports into scope](#imports-into-scope). A name brought into
scope that has the name of a local, a parameter, a global, a function, a
type, an imported package or a name of another import is a compile error,
and a warning in a function under `//cx:masking warning`.

`import "pkg"` alone no longer lets a file use a function or a type of
`pkg` by its name alone. It did so without a rule, taking the first
package with the name. Such a use is now `function 'f' not found in
package 'main'` or `struct 'T' not found in package 'main'`, and is
written `pkg.f` or brought in with an import into scope. Two programs of
the repository used it. No opcode changes. Serialized programs from
before this version must be recompiled. See
[skycoin-cx#487](https://github.com/skycoin/skycoin-cx/issues/487).

## Version 2611240001, 2026-11-24

The cxfx packages `sys/glfw`, `sys/gltext` and `sys/gl` name windows,
fonts, gifs and texture pixel copies by `i32` handles instead of `str`
names, and take callbacks as function names the compiler resolves.

| before | from 2611240001 |
|---|---|
| `glfw.CreateWindow("w", 800, 600, "Title")` | `window = glfw.CreateWindow(800, 600, "Title")` |
| `glfw.SwapBuffers("w")` and every other window function | `glfw.SwapBuffers(window)` |
| `glfw.SetKeyboardCallback("w", "onKey", "main")` | `glfw.SetKeyboardCallback(window, onKey)` |
| `glfw.SetKeyCallback("w", "onKey")` | `glfw.SetKeyCallback(window, onKey)` |
| `glfw.RegisterCallbackI32I32("main", "f")` | `glfw.RegisterCallbackI32I32(f)` |
| `gltext.LoadTrueType(file, "name", size, low, high, direction)` | `font = gltext.LoadTrueType(file, size, low, high, direction)` |
| `gltext.Printf("name", x, y, text)` and the other font functions | `gltext.Printf(font, x, y, text)` |
| `gltext.GlyphBounds()` | `gltext.GlyphBounds(font)` |
| `frames, loops, w, h = gl.NewGIF(path)` | `gif, frames, loops, w, h = gl.NewGIF(path)` |
| `gl.FreeGIF(path)`, `gl.GIFFrameToTexture(path, frame, texture)` | `gl.FreeGIF(gif)`, `gl.GIFFrameToTexture(gif, frame, texture)` |
| `gl.UploadImageToTexture(path, target, level, copy)` | `pixels = gl.UploadImageToTexture(path, target, level, copy)` |
| `gl.CxTextureGetPixel(path, x, y)`, `gl.CxReleaseTexture(path)` | `gl.CxTextureGetPixel(pixels, x, y)`, `gl.CxReleaseTexture(pixels)` |
| `gl.Strs(key, source)`, `gl.ShaderSource(shader, 1, key)`, `gl.Free(key)` | `gl.ShaderSource(shader, 1, source)` |

A callback receives the window handle, an `i32`, where it received the
window name. A callback argument that is not a function name is the
compile error 7080, and a handle that was never returned, was freed or is
out of range is an assert. `glfw.GetWindowContentScale` returns two `f32`
as its handler always wrote. `gl.GetShaderInfoLog` and
`gl.GetProgramInfoLog` return the log without the NUL padding up to the
requested length.

The opcodes `gl.Destroy` (592), `gl.Strs` (593) and `gl.Free` (594) are
deprecated and reassignable from version `2711240001`. The other opcodes
keep their numbers. Serialized programs from before this version must be
recompiled. See
[skycoin-cx#432](https://github.com/skycoin/skycoin-cx/issues/432).

## Version 2611240002, 2026-11-24

JSON is not part of the core (decision D46). The package `sys/json` and
its functions `json.Open`, `json.Close`, `json.More`, `json.Next`,
`json.Type`, `json.Delim`, `json.Bool`, `json.Float64`, `json.Int64` and
`json.Str` are removed, with the constants `json.TOKEN_*` and
`json.DELIM_*`. `import "sys/json"` is a compile error. They were
functions 52 to 61 of the system library `file`, which are deprecated and
reassignable from version `2711240001`. The other functions keep their
numbers. Serialized programs from before this version must be recompiled.
See [skycoin-cx#509](https://github.com/skycoin/skycoin-cx/issues/509).

## Version 2611240003, 2026-11-24

`sys/cipher` is the system library `cipher`, number 4, and a run enables
it with `--enable=cipher`, see [The cipher library](#the-cipher-library).
It hashes bytes, in one call or in parts through a state, and verifies,
recovers and makes secp256k1 signatures of Skycoin.

| before | from 2611240003 |
|---|---|
| `var pubKey cipher.PubKey`, `var secKey cipher.SecKey` | `var pubKey []ui8`, `var secKey []ui8` |
| `pubKey, secKey = cipher.GenerateKeyPair()` | `pubKey, secKey = cipher.KeyPairGenerate()` |
| `pubKey.PubKey[i]`, `secKey.SecKey[i]` | `pubKey[i]`, `secKey[i]` |

The opcode `cipher.GenerateKeyPair` (134) is deprecated and reassignable
from version `2711240001`, and the structs `cipher.PubKey` and
`cipher.SecKey` are removed. The new functions are numbers 1 to 9 of the
cipher library. Serialized programs from before this version must be
recompiled. See
[skycoin-cx#510](https://github.com/skycoin/skycoin-cx/issues/510).
