package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/skycoin/skycoin-cx/t0-cxtestrunner"
	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "cxtest",
		Usage: "cx programs tester",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "cxpath",
				Usage:       "cx binary path",
				DefaultText: "../bin/cx",
			},
			&cli.StringFlag{
				Name:        "wdir",
				Usage:       "working directory with *.cx tests",
				DefaultText: "./tests",
			},
			&cli.StringFlag{
				Name:  "log",
				Usage: "Enable logMask set (all, success, stderr, fail, skip, time)",
			},
			&cli.StringFlag{
				Name:  "enable-tests",
				Usage: "Enable test set (all, stable, issue, gui, examples)",
			},
			&cli.StringFlag{
				Name:  "disable-tests",
				Usage: "Disable test set (all, stable, issue, gui, examples)",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Print debug information",
			},
		},
		Action: func(_ context.Context, c *cli.Command) error {
			return Execute(c)
		},
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func Execute(c *cli.Command) error {
	cxPath := c.String("cxpath")
	if cxPath == "" {
		cxPath = "../bin/cx"
	}

	workingDir := c.String("wdir")
	if workingDir == "" {
		workingDir = "./tests"
	}

	debug := c.Bool("debug")

	var parseBitMask = func(flagName string, bitsMap map[string]runner.Bits, defaultBit runner.Bits) runner.Bits {
		var mask runner.Bits = 0
		flags := strings.Split(c.String(flagName), ",")
		for _, flag := range flags {
			mask = runner.Set(mask, bitsMap[flag])
		}
		if debug {
			fmt.Printf("Parsed bit mask for flag %s%s: %06b\n", flagName, flags, mask)
		}
		return mask
	}

	logMask := parseBitMask("log", runner.LogBits, runner.LogNone)
	enableTestsMaks := parseBitMask("enable-tests", runner.TestBits, runner.TestNone)
	disabledTestsMaks := parseBitMask("disable-tests", runner.TestBits, runner.TestNone)

	if enableTestsMaks == runner.TestAll && disabledTestsMaks == runner.TestAll {
		return errors.New("invalid test flags combination")
	}

	var testsMask runner.Bits = runner.TestAll
	if debug {
		fmt.Printf("Initial test mask: %06b\n", testsMask)
	}
	// turn on only enabled tests
	if enableTestsMaks != runner.TestNone {
		testsMask = testsMask & enableTestsMaks
	}
	// turn off only disabled test
	testsMask = testsMask &^ disabledTestsMaks
	if debug {
		fmt.Printf("Resulting test mask: %06b\n", testsMask)
	}

	tester := runner.NewTestRunner(&runner.Config{
		CxPath:         cxPath,
		WorkingDir:     workingDir,
		TestsMask:      testsMask,
		LogMask:        logMask,
		DefaultTimeout: 300 * time.Second,
	})

	var start = time.Now().Unix()

	fmt.Printf("Running CX tests in dir: '%s'\n", workingDir)
	runTestCases(tester)
	end := time.Now().Unix()

	if runner.Has(logMask, runner.LogTime) {
		fmt.Printf("\nTests finished after %d milliseconds", end-start)
	}

	fmt.Printf("\nA total of %d tests were performed\n", tester.TestCount)
	fmt.Printf("%d were successful\n", tester.TestSuccess)
	fmt.Printf("%d failed\n", tester.TestCount-tester.TestSuccess)
	fmt.Printf("%d skipped\n", tester.TestSkipped)

	if tester.TestCount == 0 || (tester.TestSuccess != tester.TestCount) {
		return errors.New("not all test succeeded")
	}

	return nil
}

func runTestCases(t *runner.TestRunner) {
	// tests
	t.Run("test-i8.cx", types.CX_SUCCESS, "i32")
	t.Run("test-i16.cx", types.CX_SUCCESS, "i32")
	t.Run("test-i32.cx", types.CX_SUCCESS, "i32")
	t.Run("test-i64.cx", types.CX_SUCCESS, "i64")
	t.Run("test-ui8.cx", types.CX_SUCCESS, "i32")
	t.Run("test-ui16.cx", types.CX_SUCCESS, "i32")
	t.Run("test-ui32.cx", types.CX_SUCCESS, "i32")
	t.Run("test-ui64.cx", types.CX_SUCCESS, "i64")
	t.Run("test-f32.cx", types.CX_SUCCESS, "f32")
	t.Run("test-f64.cx", types.CX_SUCCESS, "f64")
	t.Run("test-bool.cx", types.CX_SUCCESS, "bool")
	t.Run("test-array.cx", types.CX_SUCCESS, "array")
	t.Run("test-function.cx", types.CX_SUCCESS, "function")
	t.Run("test-control-flow.cx", types.CX_SUCCESS, "control floow")
	t.Run("test-utils.cx test-struct.cx", types.CX_SUCCESS, "struct")
	t.Run("test-str.cx", types.CX_SUCCESS, "str")
	t.Run("test-str-compare.cx", types.CX_SUCCESS, "str ordering with str.compare_by_value (skycoin-cx#87, #183)")
	t.Run("test-str-functions.cx", types.CX_SUCCESS, "str comparison functions (skycoin-cx#183)")
	t.RunWant("test-str-operators-removed.cx", types.CX_COMPILATION_ERROR, "every operator on str is a compile error (skycoin-cx#183)",
		"test-str-operators-removed.cx:13 str == str is not allowed, use str.equal_by_value or str.equal_by_pointer",
		"test-str-operators-removed.cx:14 str != str is not allowed, use str.equal_by_value or str.equal_by_pointer",
		"test-str-operators-removed.cx:15 str < str is not allowed, use str.compare_by_value",
		"test-str-operators-removed.cx:16 str > str is not allowed, use str.compare_by_value",
		"test-str-operators-removed.cx:17 str <= str is not allowed, use str.compare_by_value",
		"test-str-operators-removed.cx:18 str >= str is not allowed, use str.compare_by_value",
		"test-str-operators-removed.cx:19 str + str is not allowed, use str.concat",
		"test-str-operators-removed.cx:20 operator - is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:21 operator * is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:22 operator / is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:23 operator % is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:24 operator & is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:25 operator | is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:26 operator ^ is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:27 operator &^ is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:28 operator << is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:29 operator >> is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:30 operator - is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:31 str + str is not allowed, use str.concat",
	)
	t.RunWant("test-pointer-ordering-removed.cx", types.CX_COMPILATION_ERROR, "pointer ordering is a compile error (skycoin-cx#183)",
		"test-pointer-ordering-removed.cx:14 pointer < pointer is not allowed, pointers support only == and !=",
		"test-pointer-ordering-removed.cx:15 pointer > pointer is not allowed, pointers support only == and !=",
		"test-pointer-ordering-removed.cx:16 pointer <= pointer is not allowed, pointers support only == and !=",
		"test-pointer-ordering-removed.cx:17 pointer >= pointer is not allowed, pointers support only == and !=",
	)
	t.Run("test-pointer-equality.cx", types.CX_SUCCESS, "pointer == and != compare addresses (skycoin-cx#183)")
	t.Run("test-os-getwd.cx", types.CX_SUCCESS, "os.GetWorkingDirectory (skycoin-cx#117)")
	t.Run("test-os-run-empty.cx", types.CX_SUCCESS, "os.Run with an empty command returns OS_RUN_EMPTY_CMD")
	t.Run("test-cx-constants.cx", types.CX_SUCCESS, "cx package constants (skycoin-cx#84)")
	t.Run("test-tcp.cx", types.CX_SUCCESS, "tcp package (skycoin-cx#115)")
	t.Run("test-utils.cx test-pointers.cx", types.CX_SUCCESS, "pointers")
	t.Run("test-slices.cx", types.CX_SUCCESS, "slices")
	t.Run("test-slices-append-multi.cx", types.CX_SUCCESS, "append with several values")
	t.Run("test-slices-of-slices-literal.cx", types.CX_SUCCESS, "slice of slices literal with two inner literals")
	t.RunWant("test-printf-v.cx", types.CX_SUCCESS, "printf %v of structs, arrays and slices of structs and array pointers (skycoin-cx#256)",
		"pa [1, 2, 3]\n",
		"p {x: 5, y: 6}\n",
		"ps [{x: 0, y: 0}, {x: 0, y: 7}]\n",
		"sp [{x: 5, y: 6}]\n",
		"g [0, 4, 0]\n",
		"l {a: {x: 1, y: 0}, b: {x: 0, y: 2}, n: hi, v: [0, 3], s: [4]}\n",
		"l.b {x: 0, y: 2}\n",
		"ss [[8] [8]]\n",
		"a[1] 2\n",
	)
	t.RunWant("test-struct-pointer-deref.cx", types.CX_SUCCESS, "*p on a struct pointer is the struct value (skycoin-cx#261)",
		"deref {x: 1, y: 2}\n",
		"local {x: 3, y: 4}\n",
	)
	t.RunWant("test-slice-pointer.cx", types.CX_SUCCESS, "taking, passing and dereferencing the address of a slice (skycoin-cx#262)",
		"t [4]\n",
		"deref [4]\n",
		"grown [4, 5]\n",
		"assigned [9]\n",
		"filled [9, 7, 7]\n",
		"set [4, 5]\n",
		"global [1]\n",
	)
	t.RunWant("test-test-line.cx", types.CX_ASSERT, "a failing test() reports its own line (skycoin-cx#254)",
		"test-test-line.cx: 4: result was not equal to the expected value; first statement",
		"test-test-line.cx: 6: result was not equal to the expected value; after a declaration",
		"test-test-line.cx: 8: result was not equal to the expected value; after an assignment",
	)
	t.RunWant("test-test-type-line.cx", types.CX_COMPILATION_ERROR, "a test(), panic or operator type error names the line of the call (skycoin-cx#259)",
		"test-test-type-line.cx:8 first and second input arguments' types are not equal in 'test' call",
		"test-test-type-line.cx:11 first and second input arguments' types are not equal in 'panic' call",
		"test-test-type-line.cx:15 invalid argument types for 'add' operator",
	)

	t.Run("--cxpath test-workspace test-workspace-a.cx", types.CX_SUCCESS, "Testing if CX can set a workspace and then import a library, taking that workspace as the new relative path.")
	t.Run("--cxpath test-workspace test-workspace-b.cx", types.CX_SUCCESS, "Testing if CX can set a workspace and then import a nested library, taking that workspace as the new relative path.")
	t.Run("--cxpath test-workspace test-workspace-c.cx test-workspace-d.cx", types.CX_SUCCESS, "Testing if files supplied to the CLI override libraries in the workspace.")
	t.Run("test-slices-index-out-of-range-a.cx", types.CX_RUNTIME_INVALID_CAST, "Test index < 0")
	t.Run("test-slices-index-out-of-range-b.cx", types.CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, "Test index >= len")
	t.Run("test-slices-resize-out-of-range-a.cx", types.CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, "Test out of range after resize")
	t.Run("test-slices-resize-out-of-range-b.cx", types.CX_RUNTIME_INVALID_CAST, "Test resize with count < 0")
	t.Run("test-slices-insert-out-of-range-a.cx", types.CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, "Test insert with index > len")
	t.Run("test-slices-insert-out-of-range-b.cx", types.CX_RUNTIME_INVALID_CAST, "Test insert with index < 0")
	t.Run("test-slices-remove-out-of-range-a.cx", types.CX_RUNTIME_INVALID_CAST, "Test remove with index < 0")
	t.Run("test-slices-remove-out-of-range-b.cx", types.CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, "Test remove with index >= len")
	t.Run("test-slices-remove-out-of-range-c.cx", types.CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, "Test remove with index == 0 && len == 0")
	t.Run("test-str-substr-out-of-range-a.cx", types.CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, "str.substr with a negative start")
	t.Run("test-str-substr-out-of-range-b.cx", types.CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, "str.substr with a negative end")
	t.Run("test-str-substr-out-of-range-c.cx", types.CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, "str.substr with start > end")
	t.Run("test-str-substr-out-of-range-d.cx", types.CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, "str.substr with end > len")
	t.Run("test-short-declarations.cx", types.CX_SUCCESS, "short declarations")
	t.Run("test-parse.cx", types.CX_SUCCESS, "parse")
	t.Run("test-collection-functions.cx", types.CX_SUCCESS, "collection functions")
	t.Run("test-scopes.cx", types.CX_SUCCESS, "ProgramError in scopes.")
	t.RunEx("-heap-initial 0 test-gc.cx", types.CX_SUCCESS, "Stress-testing the garbage collector", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#13
	t.Run("testlib/json.cx test-json.cx", types.CX_SUCCESS, "ProgramError in json lib.")
	t.Run("testlib/args.cx test-args.cx", types.CX_SUCCESS, "ProgramError in args lib.")
	t.Run("test-regexp-must-compile-fail.cx", types.CX_RUNTIME_ERROR, "ProgramError in regexp lib - MustCompile should have thrown an error.")
	t.Run("test-regexp-compile-fail.cx", types.CX_SUCCESS, "ProgramError in regexp lib - error thrown by regexp.Compile does not matches expected error.")
	t.Run("test-regexp.cx", types.CX_SUCCESS, "ProgramError in regexp lib.")
	t.Run("test-regexp-not-imported.cx", types.CX_COMPILATION_ERROR, "regexp used without importing it")
	t.Run("test-regexp-find-failed-compile.cx", types.CX_RUNTIME_ERROR, "Find on a Regexp that did not compile")
	t.Run("test-regexp-find-loop.cx", types.CX_SUCCESS, "Find in a loop")
	t.Run("test-unknown-package-function.cx", types.CX_COMPILATION_ERROR, "unknown function in a built in package")
	t.Run("test-unknown-native-function.cx", types.CX_COMPILATION_ERROR, "unknown function in an imported native package")
	// t.Run("test-cipher.cx", types.CX_SUCCESS, "ProgramError in cipher lib.")

	// issues
	t.Run("issue-207.cx", types.CX_COMPILATION_ERROR, "Type casting error not reported.")
	// FAILING: panics, gl and glfw only exist with -tags cxfx; see #30
	// t.RunEx("issue-208.cx", types.CX_COMPILATION_ERROR, "Panic if return value is not used.", runner.TestGui|runner.TestStable, 0)
	t.Run("issue-214.cx", types.CX_SUCCESS, "String not working across packages")
	t.Run("issue-215.cx issue-215a.cx", types.CX_SUCCESS, "Order of files matters for structs")
	t.Run("issue-215a.cx issue-215.cx", types.CX_SUCCESS, "Order of files matters for structs")
	// FAILING: panics, gl and glfw only exist with -tags cxfx; see #31
	// t.RunEx("issue-216.cx", types.CX_COMPILATION_ERROR, "Panic when calling gl.BindBuffer with only one argument.", runner.TestGui|runner.TestStable, 0)
	t.RunEx("issue-217.cx", types.CX_SUCCESS, "Panic when giving []f32 argument to gl.BufferData", runner.TestGui, 0)
	t.Run("issue-218.cx", types.CX_SUCCESS, "Struct field crushed")
	t.Run("issue-219.cx", types.CX_SUCCESS, "Failed to modify value in an array")
	t.Run("issue-220.cx", types.CX_SUCCESS, "Panic when trying to index (using a var) an array, member of a struct passed as a function argument")
	t.Run("issue-27.cx", types.CX_SUCCESS, "Failed to use shorthand operator-assign (+=, etc.) for arithmetic statements")
	t.Run("issue-221.cx", types.CX_SUCCESS, "Can't call method from package")
	t.Run("issue-222.cx", types.CX_SUCCESS, "Can't call method if it has a parameter")
	t.Run("issue-223.cx", types.CX_SUCCESS, "Panic when using arithmetic to index an array field of a struct")
	t.Run("issue-224.cx", types.CX_SUCCESS, "Panic if return value is used in an expression")
	t.Run("issue-225.cx", types.CX_SUCCESS, "Using a variable to store the return boolean value of a function doesnt work with an if statement")
	t.Run("issue-226.cx", types.CX_SUCCESS, "Panic when accessing property of struct array passed in as argument to func")
	t.Run("issue-227.cx", types.CX_SUCCESS, "Unexpected results when accessing arrays of structs in a struct")
	t.Run("issue-230.cx", types.CX_SUCCESS, "Inline initializations and arrays")
	t.Run("issue-231.cx", types.CX_SUCCESS, "Slice keeps growing though it's cleared inside the loop")
	t.Run("issue-232.cx", types.CX_SUCCESS, "Scope not working in loops")
	t.Run("issue-233.cx", types.CX_SUCCESS, "Interdependant Structs")
	t.RunEx("issue-234.cx", types.CX_COMPILATION_ERROR, "Panic when trying to access an invalid field.", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#14
	t.Run("issue-235.cx", types.CX_COMPILATION_ERROR, "No compilation error when using an using an invalid identifier")
	t.Run("issue-236a.cx issue-236.cx", types.CX_SUCCESS, "Silent name clash between packages")
	t.Run("issue-236.cx issue-236a.cx", types.CX_SUCCESS, "Silent name clash between packages")
	t.Run("issue-237.cx", types.CX_COMPILATION_ERROR, "Invalid implicit cast.")
	t.Run("issue-238.cx", types.CX_COMPILATION_ERROR, "Panic when using +* in an expression")
	t.Run("issue-239.cx", types.CX_COMPILATION_ERROR, "No compilation error when defining a struct with duplicate fields.")
	t.Run("issue-240.cx", types.CX_SUCCESS, "Can't define struct with a single character identifier.")
	t.Run("issue-241.cx", types.CX_SUCCESS, "Panic when variable used in if statement without parenthesis.")
	t.Run("issue-242.cx", types.CX_SUCCESS, "Struct field stomped")
	t.Run("issue-243.cx", types.CX_COMPILATION_ERROR, "No compilation error when indexing an array with a non integral var.")
	t.Run("issue-244a.cx", types.CX_SUCCESS, "Panic when a field of a struct returned by a function is used in an expression")
	t.Run("issue-244b.cx", types.CX_SUCCESS, "Panic when a field of a struct returned by a function is used in an expression")
	t.Run("issue-245a.cx issue-245.cx", types.CX_COMPILATION_ERROR, "No compilation error when using var without package qualification.")
	t.Run("issue-246.cx", types.CX_SUCCESS, "No compilation error when passing *i32 as an i32 arg and conversely")
	t.Run("issue-246a.cx", types.CX_COMPILATION_ERROR, "No compilation error when passing *i32 as an i32 arg and conversely")
	t.RunEx("issue-247.cx", types.CX_COMPILATION_ERROR, "No compilation error when dereferencing an i32 var.", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#15
	t.Run("issue-248.cx", types.CX_SUCCESS, "Wrong pointer behaviour.")
	t.Run("issue-249.cx", types.CX_SUCCESS, "Return from a function doesnt work")
	t.Run("issue-249b.cx", types.CX_COMPILATION_ERROR, "Mismatched number of returning arguments is not throwing an error")
	t.Run("issue-250.cx", types.CX_COMPILATION_ERROR, "No compilation error when var is accessed outside of its declaring scope")
	t.Run("issue-251.cx", types.CX_COMPILATION_ERROR, "Panic when a str var is shadowed by a struct var in another scope")
	// FAILING: fails, glfw only exists with -tags cxfx; see #32
	// t.RunEx("issue-252.cx", types.CX_SUCCESS, "glfw.GetCursorPos() throws error", runner.TestGui|runner.TestStable, 0)
	t.Run("issue-253.cx", types.CX_SUCCESS, "Inline field and index 'dereferences' to function calls' outputs")
	t.Run("issue-254.cx", types.CX_COMPILATION_ERROR, "No compilation error when redeclaring a variable")
	t.Run("issue-255.cx", types.CX_SUCCESS, "Multi-dimensional slices don't work")
	t.Run("issue-256.cx", types.CX_SUCCESS, "can't prefix a (f32) variable with minus to flip it's signedness")
	t.Run("issue-257.cx", types.CX_COMPILATION_ERROR, "Using int literal 0 where 0.0 was needed gave no error")
	t.Run("issue-258.cx", types.CX_SUCCESS, "error with sending references of structs to functions")
	t.Run("issue-258b.cx", types.CX_SUCCESS, "error with references to struct literals")
	t.Run("issue-259.cx", types.CX_COMPILATION_ERROR, "struct identifier (when initializing fields) can be with or without a '&' prefix, with no CX error")
	t.Run("issue-260.cx", types.CX_COMPILATION_ERROR, "can assign to previously undeclared vars with just '='")
	t.Run("issue-261.cx", types.CX_SUCCESS, "empty code blocks (even if they contain commented-out lines) crash like this")
	t.Run("issue-262.cx", types.CX_SUCCESS, "increment operator ++ does not work")
	t.Run("issue-263.cx", types.CX_SUCCESS, "Method does not work")
	t.Run("issue-264.cx", types.CX_SUCCESS, "Cannot use bool variable in if expression")
	t.Run("issue-265.cx", types.CX_SUCCESS, "CX Parser does not recognize method")
	t.Run("issue-266.cx", types.CX_SUCCESS, "Goto not working on windows")
	t.Run("issue-267.cx", types.CX_SUCCESS, "Methods with pointer receivers don't work")
	t.RunEx("issue-268.cx", types.CX_SUCCESS, "when using 2 f32 out parameters, only the value of the 2nd gets through", runner.TestGui, 0)
	t.Run("issue-269.cx", types.CX_COMPILATION_ERROR, "Variable redeclaration should not be allowed")
	t.Run("issue-270.cx", types.CX_SUCCESS, "Short variable declarations are not working with calls to methods or functions")
	t.Run("issue-271.cx", types.CX_COMPILATION_ERROR, "Panic when using equality operator between a bool and an i32")
	t.Run("issue-272.cx", types.CX_SUCCESS, "String concatenation using the + operator doesn't work")
	t.Run("issue-273.cx", types.CX_SUCCESS, "Argument list is not parsed correctly")
	t.Run("issue-274.cx", types.CX_SUCCESS, "Dubious error message when indexing an array with a substraction expression")
	t.Run("issue-275.cx", types.CX_SUCCESS, "Dubious error message when inline initializing a slice")
	t.Run("issue-276a.cx issue-276.cx", types.CX_SUCCESS, "Troubles when accessing a global var from another package")
	t.Run("issue-277.cx", types.CX_SUCCESS, "same func names (but in different packages) collide")
	t.Run("issue-278.cx", types.CX_COMPILATION_ERROR, "can use vars from other packages without a 'packageName.' prefix")
	t.Run("issue-279.cx", types.CX_SUCCESS, "False positive when detecting variable redeclaration.")
	t.RunEx("issue-279a.cx", types.CX_SUCCESS, "False positive when detecting variable redeclaration.", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#16
	t.Run("issue-280.cx", types.CX_SUCCESS, "Problem with struct literals in short variable declarations")
	t.Run("issue-281.cx", types.CX_SUCCESS, "Panic when using the return value of a function in a short declaration")
	t.Run("issue-282a.cx", types.CX_COMPILATION_ERROR, "Panic when inserting a new line in a string literal")
	t.Run("issue-282b.cx", types.CX_SUCCESS, "Panic when inserting a new line in a string literal")
	t.Run("issue-283.cx", types.CX_COMPILATION_ERROR, "Panic when declaring a variable of an unknown type")
	t.Run("issue-284.cx", types.CX_COMPILATION_ERROR, "No compilation error when using arithmetic operators on struct instances")
	t.RunEx("issue-285.cx", types.CX_SUCCESS, "Parser gets confused with `2 -2`", runner.TestStable, 0)
	t.Run("issue-286.cx", types.CX_SUCCESS, "Panic in when assigning an empty initializer list to a []i32 variable")
	t.Run("issue-287.cx", types.CX_SUCCESS, "Cx stack overflow when appending to a slice passed by address")
	t.Run("issue-288.cx", types.CX_COMPILATION_ERROR, "Panic when trying to assign return value of a function returning void")
	t.Run("issue-289.cx", types.CX_COMPILATION_ERROR, "Panic when using a function declared in another package without importing the package")
	t.Run("issue-290.cx", types.CX_SUCCESS, "Cx memory stomped")
	t.Run("issue-291.cx", types.CX_SUCCESS, "Invalid offset calculation of non literal strings when appended to a slice")
	t.Run("issue-292.cx", types.CX_COMPILATION_ERROR, "Panic when calling a function from another package where the package name alias a local variable name")
	t.Run("issue-293.cx", types.CX_SUCCESS, "Garbage memory when passing the address of slice element to a function")
	t.Run("issue-294.cx", types.CX_SUCCESS, "Type deduction of struct field fails")
	t.Run("issue-295.cx", types.CX_COMPILATION_ERROR, "No compilation error when assigning a i32 value to a []i32 variable")
	t.Run("issue-296.cx", types.CX_COMPILATION_ERROR, "No compilation error when comparing value of different types")
	t.Run("-stack-size 30 issue-297a.cx", types.CX_RUNTIME_STACK_OVERFLOW_ERROR, "No stack overflow error")
	t.RunEx("-heap-initial 100 -heap-max 110 issue-297b.cx", types.CX_RUNTIME_HEAP_EXHAUSTED_ERROR, "No heap exhausted error", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#17
	t.Run("issue-298.cx", types.CX_SUCCESS, "Argument type deduction failed when passing address of an i32 struct field to a function accepting *i32 argument.")
	t.Run("issue-299.cx", types.CX_COMPILATION_ERROR, "Type checking is not working with receiving variables of unexpected types")
	t.Run("issue-300.cx", types.CX_SUCCESS, "Crash when using a constant expression in a slice literal expression")
	t.Run("issue-301-a.cx", types.CX_COMPILATION_ERROR, "Can redeclare variables if they are inline initialized")
	t.Run("issue-301-b.cx", types.CX_COMPILATION_ERROR, "Can redeclare variables if they are inline initialized")
	t.Run("issue-302.cx", types.CX_SUCCESS, "Trying to determine the length of a slice of struct instances throws an error.")
	t.RunEx("issue-68.cx", types.CX_SUCCESS, "Wrong sprintf behaviour when passing increment expression as argument", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#18
	t.RunEx("issue-67.cx", types.CX_COMPILATION_ERROR, "Panic when using void return value of a function in a for loop expression", runner.TestIssue, 0)
	t.Run("issue-66.cx", types.CX_SUCCESS, "Wrong sprintf behaviour when printing boolean values with %v")
	t.RunEx("issue-65.cx", types.CX_SUCCESS, "for true {} loop scope is not executed", runner.TestIssue, 0)        // known failing, skycoin/skycoin-cx#20
	t.RunEx("issue-64.cx", types.CX_SUCCESS, "for loop using boolean value is not compiling", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#21
	t.Run("issue-303.cx", types.CX_SUCCESS, "Concatenation of str variables with + operator doesn't work")
	t.Run("issue-304.cx", types.CX_SUCCESS, "Short declaration doesn't compile with opcode return value")
	t.Run("issue-305.cx", types.CX_SUCCESS, "Compilation error when struct field is named 'input' or 'output'")
	t.RunEx("issue-63.cx", types.CX_COMPILATION_ERROR, "No compilation error when using empty argument list after function call", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#22
	t.Run("issue-306.cx", types.CX_COMPILATION_ERROR, "No compilation error when using float value in place of boolean expression")
	t.Run("issue-308.cx", types.CX_COMPILATION_ERROR, "Panic when package contains duplicate function signature")
	t.RunEx("issue-62.cx", types.CX_COMPILATION_ERROR, "Left hand side of , is not compiled", runner.TestIssue, 0)                                  // known failing, skycoin/skycoin-cx#23
	t.RunEx("issue-61-a.cx", types.CX_SUCCESS, "Compilation error when using return value of a member method in a expression", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#24
	t.RunEx("issue-61-b.cx", types.CX_SUCCESS, "Compilation error when using return value of a member method in a expression", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#24
	t.Run("issue-309.cx", types.CX_SUCCESS, "Compilation error when left hand side of an assignment expression is a struct field")
	t.RunEx("--heap-initial=1M issue-60.cx", types.CX_SUCCESS, "Crash when INIT_HEAP_SIZE limit is reached ", runner.TestIssue, 0)
	t.RunEx("--heap-initial=1M issue-59-a.cx", types.CX_SUCCESS, "Crash in garbage collector when heap is resized", runner.TestIssue, 0)
	t.RunEx("--heap-initial=1M issue-59-b.cx", types.CX_SUCCESS, "Crash in garbage collector when heap is resized", runner.TestIssue, 0)
	t.RunEx("issue-53-a.cx", types.CX_SUCCESS, "Issues with slice of type T where sizeof T is different than 4 ", runner.TestStable, 0)
	t.RunEx("issue-53-b.cx", types.CX_SUCCESS, "Issues with slice of type T where sizeof T is different than 4 ", runner.TestStable, 0)
	t.RunEx("issue-53-c.cx", types.CX_SUCCESS, "Issues with slice of type T where sizeof T is different than 4 ", runner.TestStable, 0)
	t.RunEx("issue-51.cx", types.CX_COMPILATION_ERROR, "No compilation error when global variable is redeclared at local scope", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#25
	t.RunEx("issue-49.cx", types.CX_COMPILATION_ERROR, "No compilation error when assigning an literal which overflow the receiving type", runner.TestIssue, 0)
	t.RunEx("issue-48.cx", types.CX_SUCCESS, "Cx is not supporting short-circuit evaluation", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#26
	t.Run("issue-39.cx", types.CX_COMPILATION_ERROR, "func defined with no arguments, called WITH arguments causes PANIC")
	t.Run("issue-2.cx", types.CX_SUCCESS, "multi-dimensional arrays are not working")
	t.Run("issue-1.cx", types.CX_SUCCESS, "multi-dimensional slices are not working")
	t.RunEx("issue-120-a.cx", types.CX_COMPILATION_ERROR, "Invalid implicit cast when assigning the result of a math operator to a variable.", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#27
	t.RunEx("issue-120-b.cx", types.CX_COMPILATION_ERROR, "Invalid implicit cast when assigning the result of a math operator to a variable.", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#27
	t.RunEx("issue-120-c.cx", types.CX_COMPILATION_ERROR, "Invalid implicit cast when assigning the result of a math operator to a variable.", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#27
	t.RunEx("issue-121.cx", types.CX_SUCCESS, "Compilation error when using unary negative operator on a function call", runner.TestIssue, 0)                       // known failing, skycoin/skycoin-cx#28
	t.RunEx("issue-131.cx", types.CX_SUCCESS, "Panic when using arithmetic operations.", runner.TestIssue, 0)
	t.RunEx("test-ar-1.cx", types.CX_SUCCESS, "Panic when using string to pointer array.", runner.TestIssue, 0)
	t.RunEx("issue-157.cx", types.CX_SUCCESS, "expected either 'i32' or 'i64', got 'ident'", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#29
	t.Run("issue-650.cx", types.CX_SUCCESS, "test should not fail")

	// We need to fix serialization and deserialization as user-callable functions
	// t.RunEx("issue-309.cx", types.CX_SUCCESS, "Serialization is not taking into account non-default stack sizes.", TestIssue, 0)
	// t.RunEx("issue-310.cx", types.CX_SUCCESS, "Splitting a serialized program into its blockchain and transaction parts.", TestIssue, 0)
	// t.RunEx("issue-311.cx", types.CX_SUCCESS, "`CurrentFunction` and `CurrentStruct` are causing errors in programs with more than 1 package.", TestIssue, 0)
	// t.RunEx("issue-312.cx", types.CX_SUCCESS, "Deserialization is not setting correctly the sizes for the CallStack, HeapStartsAt and StackSize fields of the CXProgram structure.", TestIssue, 0)

	t.Run("issue-28-f32.cx", types.CX_SUCCESS, "f32")
	t.Run("issue-28-f64.cx", types.CX_SUCCESS, "f64")
	t.Run("issue-28-i8.cx", types.CX_SUCCESS, "i8")
	t.Run("issue-28-i16.cx", types.CX_SUCCESS, "i16")
	t.Run("issue-28-i32.cx", types.CX_SUCCESS, "i32")
	t.Run("issue-28-i64.cx", types.CX_SUCCESS, "i64")
	t.Run("issue-28-ui8.cx", types.CX_SUCCESS, "ui8")
	t.Run("issue-28-ui16.cx", types.CX_SUCCESS, "ui16")
	t.Run("issue-28-ui32.cx", types.CX_SUCCESS, "ui32")
	t.Run("issue-28-ui64.cx", types.CX_SUCCESS, "ui64")

	// // New tests added starting 05/10/2022
	t.Run("issue-struct-a.cx", types.CX_SUCCESS, "struct printf")
	t.Run("issue-struct-b.cx", types.CX_SUCCESS, "struct printf uninitialized")

	// Examples
	t.RunEx("./examples/bubble-sort.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/cipher.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/concatenating-strings.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/conversions.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/convert-time.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/digital-root.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/errors.cx", types.CX_RUNTIME_ERROR, "", runner.TestExamples, 0)
	t.RunEx("./examples/even-or-odd.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/expressions.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/factorial.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/fibonacci-recursive.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/function-to-function-call.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/hello-world.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/nth-pow.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/nth-recursive-pow.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/packages.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	// t.RunEx("./examples/program.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	// t.RunEx("./examples/read-string.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/regexp.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/return.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/sum-digits-integer.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/temp-converter.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/testing-global-1.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/testing-globals.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/testing-packages.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
	t.RunEx("./examples/testing-pointers-to-arrays-of-structs.cx", types.CX_SUCCESS, "", runner.TestExamples, 0)
}
