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
	runGuideSnippetTests(tester, workingDir)
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
	t.Run("test-str-compare.cx", types.CX_SUCCESS, "str ordering with string.OrderingByByteValue (skycoin-cx#87, #183)")
	t.Run("test-str-functions.cx", types.CX_SUCCESS, "str comparison functions (skycoin-cx#183)")
	t.RunWant("test-str-operators-removed.cx", types.CX_COMPILATION_ERROR, "every operator on str is a compile error (skycoin-cx#183)",
		"test-str-operators-removed.cx:15 str == str is not allowed, use string.EqualityByByteValue or string.EqualityByHeapPointer",
		"test-str-operators-removed.cx:16 str != str is not allowed, use string.EqualityByByteValue or string.EqualityByHeapPointer",
		"test-str-operators-removed.cx:17 str < str is not allowed, use string.OrderingByByteValue",
		"test-str-operators-removed.cx:18 str > str is not allowed, use string.OrderingByByteValue",
		"test-str-operators-removed.cx:19 str <= str is not allowed, use string.OrderingByByteValue",
		"test-str-operators-removed.cx:20 str >= str is not allowed, use string.OrderingByByteValue",
		"test-str-operators-removed.cx:21 str + str is not allowed, use string.Concatenation",
		"test-str-operators-removed.cx:22 operator - is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:23 operator * is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:24 operator / is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:25 operator % is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:26 operator & is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:27 operator | is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:28 operator ^ is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:29 operator &^ is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:30 operator << is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:31 operator >> is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:32 operator - is not allowed on str, strings have no arithmetic operators",
		"test-str-operators-removed.cx:33 str + str is not allowed, use string.Concatenation",
	)
	t.RunWant("test-pointer-ordering-removed.cx", types.CX_COMPILATION_ERROR, "pointer ordering is a compile error (skycoin-cx#183)",
		"test-pointer-ordering-removed.cx:14 pointer < pointer is not allowed, pointers support only == and !=",
		"test-pointer-ordering-removed.cx:15 pointer > pointer is not allowed, pointers support only == and !=",
		"test-pointer-ordering-removed.cx:16 pointer <= pointer is not allowed, pointers support only == and !=",
		"test-pointer-ordering-removed.cx:17 pointer >= pointer is not allowed, pointers support only == and !=",
	)
	t.Run("test-pointer-equality.cx", types.CX_SUCCESS, "pointer == and != compare addresses (skycoin-cx#183)")
	t.Run("test-os-getwd.cx", types.CX_SUCCESS, "os.GetWorkingDirectory (skycoin-cx#117)")
	t.Run("test-os-sandbox.cx", types.CX_SUCCESS, "program paths stay inside the sandbox (skycoin-cx#433)")
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
	t.RunWant("test-slice-pointer-index.cx", types.CX_SUCCESS, "(*p)[i] and len(*p) on a pointer to a slice (skycoin-cx#49, skycoin-cx#267)",
		"3\n2\n3\n3\nlen 2 sum 8\nbump [3, 15]\nvar 15 plus 4\nf64 1.5 2.5 2\nui8 [7] 7\nstruct 5 40 2\n")
	t.RunWant("test-slice-pointer-index-errors.cx", types.CX_COMPILATION_ERROR, "p[i] and *p[i] on a pointer to a slice are invalid indexing (skycoin-cx#49)",
		"test-slice-pointer-index-errors.cx:9 invalid indexing",
		"test-slice-pointer-index-errors.cx:10 invalid indexing")
	t.RunWant("test-append-slice-pointer.cx", types.CX_COMPILATION_ERROR, "append, resize and remove of a pointer to a slice are compile errors (skycoin-cx#52)",
		"test-append-slice-pointer.cx:6 function 'append' expected a slice as input argument; '*[]i32' was provided",
		"test-append-slice-pointer.cx:7 function 'resize' expected a slice as input argument; '*[]i32' was provided",
		"test-append-slice-pointer.cx:8 function 'remove' expected a slice as input argument; '*[]i32' was provided")
	t.RunWant("test-append-slice-pointer-deref.cx", types.CX_SUCCESS, "append(*p, v) appends to the slice p points to (skycoin-cx#52)",
		"3 [4, 5, 6] [4, 5, 6]\n")
	t.RunWant("test-slice-literal-struct.cx", types.CX_SUCCESS, "slice literals of structs and empty slice literals (skycoin-cx#47)",
		"len 1\nlen 1 hmmmm\n4 [{x: 3, y: 1.5}, {x: 9, y: 0}, {x: 7, y: 0}, {x: 0, y: 0}]\nloop 1 2\nmany 7 28\ndecl 1 8\nglobal 2 [{x: 1, y: 0}, {x: 2, y: 2.5}]\nempty 0 0\n")
	t.RunWant("test-slice-literal-struct-errors.cx", types.CX_COMPILATION_ERROR, "elements of the wrong type in a struct slice literal (skycoin-cx#47)",
		"test-slice-literal-struct-errors.cx:20 cannot assign value of type 'U'",
		"test-slice-literal-struct-errors.cx:21 trying to assign argument of type 'U'",
		"test-slice-literal-struct-errors.cx:22 trying to assign argument of type 'i32'",
		"test-slice-literal-struct-errors.cx:23 function 'makeU' expected receiving variable of type 'U'; 'T' was provided",
		"test-slice-literal-struct-errors.cx:24 field 'n' in struct literal of type 'T' expected argument of type 'i32'; 'str' was provided",
		"test-slice-literal-struct-errors.cx:25 field 'n' in struct literal of type 'T' expected argument of type 'i32'; 'str' was provided",
		"test-slice-literal-struct-errors.cx:26 function 'append' expected argument of type 'i32'; 'T' was provided")
	t.RunWant("test-call-index-operator.cx", types.CX_SUCCESS, "f()[i] of an array or slice result as an operator operand",
		"7\n9\n42\n2\n6 14 2.5\ncond\n")
	t.RunWant("test-import-global-array-number.cx test-import-global-array.cx", types.CX_SUCCESS, "pkg.A[i] reads and writes a global array or slice of another package",
		"4 7 14\n1.5 3\n20 2 12\n[0, 4, 7] [11, 20] 7\n")
	t.RunWant("test-slice-resize-grow.cx", types.CX_SUCCESS, "resize past twice the capacity allocates the whole slice (audit A-04)",
		"64 1111 2222 2\n[100, 101, 102, 103, 104, 105, 106, 107, 108, 109] [5, 5, 5, 5, 5, 5]\n100 2 3\n")
	t.RunWant("test-slice-resize-huge.cx", types.CX_RUNTIME_HEAP_EXHAUSTED_ERROR, "a slice too large for the heap is a heap exhausted error (audit A-05)",
		"test-slice-resize-huge.cx:5, CX_RUNTIME_HEAP_EXHAUSTED_ERROR")
	t.RunWant("test-os-read-slice-grow.cx", types.CX_SUCCESS, "os.ReadI32Slice allocates the whole slice and returns false for a count past the heap (audit B-01, B-12)",
		"read true len 10 same true\nhuge false len 0\n")
	t.RunWant("test-test-line.cx", types.CX_ASSERT, "a failing test() reports its own line (skycoin-cx#254)",
		"test-test-line.cx: 4: result was not equal to the expected value; first statement",
		"test-test-line.cx: 6: result was not equal to the expected value; after a declaration",
		"test-test-line.cx: 8: result was not equal to the expected value; after an assignment",
	)
	t.RunWant("test-vm-divide-callee.cx", types.CX_RUNTIME_ERROR, "a division by zero in a callee prints its line and the call stack (skycoin-cx#408)",
		"5\nerror: test-vm-divide-callee.cx:8, CX_RUNTIME_ERROR, runtime error: integer divide by zero\n===Callstack===\n>>> divide()",
		">>> middle()",
		">>> main()",
	)
	t.RunWant("test-vm-test-callee.cx", types.CX_ASSERT, "a failed test in a callee prints its message (skycoin-cx#408)",
		"1\ntest-vm-test-callee.cx: 8: result was not equal to the expected value; local message\n",
		"test-vm-test-callee.cx: 9: result was not equal to the expected value\n",
		"test-vm-test-callee.cx: 10: result was not equal to the expected value; literal message\n2\n",
	)
	t.RunWant("test-vm-stack-overflow.cx", types.CX_RUNTIME_STACK_OVERFLOW_ERROR, "large frames overflow the stack (skycoin-cx#408)",
		"error: test-vm-stack-overflow.cx:143, CX_RUNTIME_STACK_OVERFLOW_ERROR, stack overflow\n===Callstack===\n>>> recur()",
	)
	t.RunWant("test-vm-call-depth.cx", types.CX_RUNTIME_STACK_OVERFLOW_ERROR, "the call depth limit is a stack overflow (skycoin-cx#408)",
		"1\nerror: test-vm-call-depth.cx:7, CX_RUNTIME_STACK_OVERFLOW_ERROR, stack overflow\n===Callstack===\n>>> recur()",
	)
	t.RunWant("test-vm-init-divide.cx", types.CX_RUNTIME_ERROR, "a division by zero in a global initializer is a runtime error (skycoin-cx#408)",
		"error: test-vm-init-divide.cx:7, CX_RUNTIME_ERROR, runtime error: integer divide by zero\n===Callstack===\n>>> *init()",
	)
	t.RunWant("test-vm-array.cx", types.CX_SUCCESS, "arrays in the stack and data segment (skycoin-cx#408)",
		"16\n30\n23\n23\n7\n5\n",
	)
	t.RunWant("test-vm-struct.cx", types.CX_SUCCESS, "struct fields, arrays of structs and struct parameters (skycoin-cx#408)",
		"12\n9\n6\n11\n12\n5\n9\n",
	)
	t.RunWant("test-vm-pointer.cx", types.CX_SUCCESS, "pointers to scalars, fields and elements in the stack and data segment (skycoin-cx#408)",
		"42\n43\n2\n33\n7\n9\n8\n",
	)
	t.RunWant("test-vm-print.cx", types.CX_SUCCESS, "string.PrintLine and printf (skycoin-cx#408)",
		"hello\nfrom a global\nfrom a field\n\n",
		"-12 1234567890123 200\n1.5000000 2.2500000000000000\n-12 1.5 2.25 true from a field\n",
		"from a field has 3\ntrue false\n100% done, a \n escape\nmissing -12 and %!d(MISSING)\n",
		"-12 on a float is empty: []\nelement 77\nthrough a pointer 77 77\n",
	)
	t.RunWant("test-array-index-negative.cx", types.CX_RUNTIME_INVALID_CAST, "a negative array index is a runtime error and the call stack prints without a panic",
		"1\nerror: test-array-index-negative.cx:7, CX_RUNTIME_INVALID_CAST, 11\n===Callstack===\n>>> store()\n",
		"\ttest-array-index-negative.cx:7 : identity() : <index out of range>\n\n>>> main()\n",
	)
	t.RunWant("test-printf-extra.cx", types.CX_SUCCESS, "printf prints values past the last verb as %!(EXTRA ...)",
		"none\n%!(EXTRA i32=1, i32=2)5\n%!(EXTRA i64=1234567890123, bool=true, str=text)5 and\n%!(EXTRA i32=7, i32=9, f64=2.5)",
	)
	t.RunWant("test-printf-extra-values.cx", types.CX_SUCCESS, "printf prints arrays, structs and fields past the last verb as %!(EXTRA ...)",
		"1\n%!(EXTRA i32=[0, 7, 0], point={x: 0, y: 9}, i32=9)",
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
	t.Run("test-str-substr-out-of-range-a.cx", types.CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, "string.SubstringByByteRange with a negative start")
	t.Run("test-str-substr-out-of-range-b.cx", types.CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, "string.SubstringByByteRange with a negative end")
	t.Run("test-str-substr-out-of-range-c.cx", types.CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, "string.SubstringByByteRange with start > end")
	t.Run("test-str-substr-out-of-range-d.cx", types.CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, "string.SubstringByByteRange with end > len")
	t.Run("test-short-declarations.cx", types.CX_SUCCESS, "short declarations")
	t.Run("test-parse.cx", types.CX_SUCCESS, "parse")
	t.Run("test-collection-functions.cx", types.CX_SUCCESS, "collection functions")
	t.Run("test-scopes.cx", types.CX_SUCCESS, "ProgramError in scopes.")
	t.RunWant("test-scope-block-starts-with-block.cx", types.CX_SUCCESS, "a block whose first statement is a block", "2\n")
	t.RunWant("test-scope-shadow-after-block.cx", types.CX_SUCCESS, "shadowing after a nested block", "2\n1\n")
	t.RunWant("test-scope-one-statement-block.cx", types.CX_SUCCESS, "a one statement block has its own scope", "0\n")
	t.RunWant("test-scope-method-call-first.cx", types.CX_SUCCESS, "a method call as the first statement of a block", "7\n1\n2\n")
	t.RunWant("test-scope-nested-shadow.cx", types.CX_SUCCESS, "shadowing in nested if, for and else blocks", "3\n4\n2\n5\n1\n")
	t.RunWant("test-scope-sibling-blocks.cx", types.CX_SUCCESS, "sibling blocks declare the same names (skycoin-cx#16)", "1\n2\n")
	t.RunWant("test-loop-break-nested.cx", types.CX_SUCCESS, "break leaves only the innermost loop (skycoin-cx#39)",
		"j: 0\nout of j loop\nj: 0\nout of j loop\nout of i loop\nj: 0\nout of j loop\nj: 0\nout of j loop\nout of i loop\nout of h loop\n")
	t.RunWant("test-loop-continue-nested.cx", types.CX_SUCCESS, "continue goes on with the innermost loop (skycoin-cx#39)",
		"j: 0\nj: 1\nj: 2\nout of j loop\nj: 0\nj: 1\nj: 2\nout of j loop\nout of h loop\n")
	t.RunWant("test-loop-three-deep.cx", types.CX_SUCCESS, "break and continue three loops deep (skycoin-cx#39)",
		"break 0 0 0\nbreak 0 1 0\nbreak 1 0 0\nbreak 1 1 0\n"+
			"continue 0 0 0\ncontinue 0 0 2\ncontinue 0 1 0\ncontinue 0 1 2\nafter c 0 1\n"+
			"continue 1 0 0\ncontinue 1 0 2\ncontinue 1 1 0\ncontinue 1 1 2\nafter c 1 1\nafter b 1\n")
	t.RunWant("test-loop-if-else.cx", types.CX_SUCCESS, "break and continue inside if and else (skycoin-cx#39)", "odd 1\nodd 3\n3\n")
	t.RunWant("test-loop-goto.cx", types.CX_SUCCESS, "goto out of and back into a labelled loop with break and continue", "1 0 0\n2 2 0\n3 0 0\n4\n")
	t.RunWant("test-loop-for-true.cx", types.CX_SUCCESS, "for true runs its body (skycoin-cx#20)", "55\n4\n")
	t.RunWant("test-loop-for-bool.cx", types.CX_SUCCESS, "for on a bool variable or expression, an empty condition and an assignment post statement (skycoin-cx#21)", "3\n20\n6\n6\n6\n")
	t.RunWant("test-shortcircuit-assign.cx", types.CX_SUCCESS, "&& and || in assignments, nested, with ! and in a global initializer short-circuit (skycoin-cx#26)",
		"T0 T1 = true\nF3 T4 = true\nF5 = false\nT7 F8 = false\nF9 F10 T11 = true\nT13 T14 F15 = false\nT17 F18 T19 = true\nF20 = false\nF23 T25 T26 = true\nF27 = true\nT29 F30 T31 = false\nT32 = false\n= true\n= false\nF36 = true\n= true\n= false\nearly true\n26\n")
	t.RunWant("test-shortcircuit-control.cx", types.CX_SUCCESS, "&& and || in if and for conditions skip a division by zero and an out of range index (skycoin-cx#26)",
		"T1 then\nF3 else\nF5 T6 F7 then\nT0 T1 T2 n 3\nT10 T11 \nno divide\na[0] 5\n10\n")
	t.RunWant("test-shortcircuit-call.cx", types.CX_SUCCESS, "&& and || in arguments and return values short-circuit, bool.and and bool.or evaluate both (skycoin-cx#26)",
		"T1 F3 show true false\nF5 T100 show true true\nT200 show false true\nT301 pair true true\nT300 pair true false\nT6 F7 T8 F9 T10 F11 show true false\nF0 T0 show false true\nshow true false\nF2 T2 show false true\nT12 T13 = true\nF14 T15 = false\n")
	t.RunWant("test-shortcircuit-slice.cx", types.CX_SUCCESS, "&& and || guard slice indexes, including a bare indexed operand (skycoin-cx#26)",
		"s[0] 5\nin range check\nfalse\ntrue\n")
	t.RunWant("test-loop-break-outside.cx", types.CX_COMPILATION_ERROR, "break outside a loop (skycoin-cx#60)",
		`test-loop-break-outside.cx:5 break is not in a loop`)
	t.RunWant("test-loop-continue-outside.cx", types.CX_COMPILATION_ERROR, "continue outside a loop (skycoin-cx#60)",
		`test-loop-continue-outside.cx:6 continue is not in a loop`)
	t.RunWant("test-operator-result-type.cx", types.CX_COMPILATION_ERROR, "an arithmetic result of another type is not cast implicitly (skycoin-cx#27)",
		"test-operator-result-type.cx:4 cannot assign value of type 'i32' to identifier '1:f' of type 'f32'",
		"test-operator-result-type.cx:6 cannot assign value of type 'i32' to identifier '2:h' of type 'f32'",
		"test-operator-result-type.cx:8 cannot assign value of type 'i32' to identifier '4:y' of type 'f32'",
		"test-operator-result-type.cx:9 cannot assign value of type 'i32' to identifier '5:l' of type 'i64'",
		"test-operator-result-type.cx:11 cannot assign value of type 'i32' to identifier '6:a' of type 'f32'",
		"test-operator-result-type.cx:13 cannot assign value of type 'f32' to identifier '8:n' of type 'i32'")
	t.RunWant("test-operator-result-type-global.cx", types.CX_COMPILATION_ERROR, "a global initialized with an arithmetic result of another type (skycoin-cx#27)",
		"test-operator-result-type-global.cx:3 cannot assign value of type 'i32' to identifier 'g' of type 'f32'")
	t.RunWant("test-operator-result-type-valid.cx", types.CX_SUCCESS, "negative literals and arithmetic results of the receiver's type (skycoin-cx#27)",
		"-1 -1 -1 7 2 -1\n")
	t.Run("issue-520.cx", types.CX_COMPILATION_ERROR, "a negative i32 literal initializes an f32 (skycoin-cx#27)")
	t.RunWant("test-slice-builtin-element-type.cx", types.CX_COMPILATION_ERROR, "element and result types of append, insert, resize, remove and copy (skycoin-cx#55)",
		"test-slice-builtin-element-type.cx:19 function 'append' expected argument of type 'fooA'; 'fooB' was provided",
		"test-slice-builtin-element-type.cx:21 function 'append' expected argument of type 'i32'; 'str' was provided",
		"test-slice-builtin-element-type.cx:23 function 'append' expected argument of type '[]i32'; 'i32' was provided",
		"test-slice-builtin-element-type.cx:24 function 'insert' expected argument of type 'i32'; 'f32' was provided",
		"test-slice-builtin-element-type.cx:26 function 'copy' expected argument of type '[]i32'; '[]f32' was provided",
		"test-slice-builtin-element-type.cx:27 function 'append' expected argument of type '[]i32'; '[]f32' was provided",
		"test-slice-builtin-element-type.cx:28 function 'resize' expected argument of type '[]i32'; '[]f32' was provided",
		"test-slice-builtin-element-type.cx:29 function 'remove' expected argument of type '[]i32'; '[]f32' was provided",
		"test-slice-builtin-element-type.cx:31 function 'append' expected argument of type '[]i32'; 'i32' was provided",
		"test-slice-builtin-element-type.cx:32 function 'append' expected argument of type 'i32'; '[]i32' was provided")
	t.RunWant("test-slice-field-multidimensional.cx", types.CX_SUCCESS, "append, insert, remove, resize and indexing on multi-dimensional slice struct fields (skycoin-cx#55)",
		"3\n5\n6\n1.5\n8\n10\n12\n13\n")
	t.RunWant("test-global-redeclaration.cx", types.CX_COMPILATION_ERROR, "a global declared twice in one package (skycoin-cx#57)",
		"test-global-redeclaration.cx:4 'i' redeclared")
	t.RunWant("test-index-comma.cx", types.CX_COMPILATION_ERROR, "a comma inside an index is a syntax error (skycoin-cx#59)",
		"test-index-comma.cx:7: syntax error")
	t.Run("issue-22.cx", types.CX_COMPILATION_ERROR, "commas inside slice indexers of a struct field (skycoin-cx#59)")
	t.RunWant("test-goto-label-missing.cx", types.CX_COMPILATION_ERROR, "goto to a label that does not exist",
		"test-goto-label-missing.cx:4 label 'nowhere' is not defined")
	t.RunWant("test-field-access-primitive.cx", types.CX_COMPILATION_ERROR, "a field of a variable that is not a struct",
		"test-field-access-primitive.cx:5 illegal method call or field access on identifier 'x' of primitive type 'i32'",
		"test-field-access-primitive.cx:7 illegal method call or field access on identifier 'f' of primitive type 'f32'")
	t.RunWant("test-method-call-value.cx", types.CX_SUCCESS, "method calls in operands, arguments, return values, if and for conditions and chains (skycoin-cx#24)",
		"7\n8\n8\n77\n8\n13\n10\n18\np(3,4)\n7\n7\n5\n6\nd(3,4)\nif sum\nif positive\nboth positive\nloop 1\nloop 2\nloop 3\n14\n15\n4\n9\n10\n7\n7\n8\n14\n103\n8\n7 5\n7 2 13\n-7\nfalse\n7 x(3,4)\n")
	t.RunWant("test-method-pointer-receiver.cx", types.CX_SUCCESS, "a pointer receiver method on a value, a pointer and a struct field",
		"11 12\n111 112\n0 0 6 1\n")
	t.RunWant("test-method-result-missing.cx", types.CX_COMPILATION_ERROR, "a method with no output used as a value",
		"test-method-result-missing.cx:15 method 'Point.show' returns no value")
	t.RunWant("test-call-result-call.cx", types.CX_COMPILATION_ERROR, "calling the result of a call (skycoin-cx#22)",
		"test-call-result-call.cx:19 the result of a call is not a function",
		"test-call-result-call.cx:20 the result of a call is not a function",
		"test-call-result-call.cx:21 the result of a call is not a function",
		"test-call-result-call.cx:23 the result of a call is not a function")
	t.RunWant("test-package-function-qualified.cx", types.CX_COMPILATION_ERROR, "pkg.f does not find a function of an import of pkg (skycoin/cx#9)",
		"test-package-function-qualified.cx:25 identifier 'main.test1' does not exist")
	t.RunWant("test-package-function-value.cx", types.CX_COMPILATION_ERROR, "pkg.F without parentheses is not a call (skycoin-cx#48)",
		"test-package-function-value.cx:8 cannot assign value of type 'func()(i32)' to identifier '1:x' of type 'i32'",
		"test-package-function-value.cx:9 function 'i32.print' expected input argument of type 'i32'; 'func()(i32)' was provided")
	t.RunWant("issue-136.cx", types.CX_COMPILATION_ERROR, "pkg.F without parentheses is not a call (skycoin-cx#48)",
		"issue-136.cx:7 invalid expression")
	t.RunWant("test-function-shadows-builtin.cx", types.CX_SUCCESS, "a function shadows a built-in of the same name in its package (skycoin-cx#294)",
		"-1\n99\n7\n20\n7\n20\n")
	t.RunWant("issue-23.cx", types.CX_SUCCESS, "logical not on a call result (skycoin-cx#58)", "false\n")
	t.Run("issue-121.cx", types.CX_SUCCESS, "unary minus on a call result (skycoin-cx#28)")
	t.RunWant("issue-25.cx", types.CX_SUCCESS, "unary minus on struct fields (skycoin-cx#56)", "-12\n15\n")
	t.RunWant("test-unary-value.cx", types.CX_SUCCESS, "! and - on calls, method calls and parenthesized expressions (skycoin-cx#58)",
		"false\ntrue\ntrue\nside 1\ntrue\ntrue\n-4\n-8\n-7\n3\n-1.5\n-3\nfalse\n1\n-1\n")
	t.RunWant("test-unary-field.cx", types.CX_SUCCESS, "! and - on nested, pointer and indexed struct fields (skycoin-cx#56)",
		"-5\n-1.5\nfalse\n-9\n-5\n-5\n5\n")
	t.RunWant("issue-450.cx", types.CX_SUCCESS, "++ and -- on every numeric type (skycoin-cx#45)", "2\n0\n2\n2\n0\n2\n")
	t.RunWant("issue-450b.cx", types.CX_SUCCESS, "++ and -- on every numeric type (skycoin-cx#45)", "2\n0\n2\n2\n2\n")
	t.RunWant("test-incdec-operand.cx", types.CX_SUCCESS, "++, -- and compound assignment on unsigned types, fields, indexes and pointers (skycoin-cx#45)",
		"0\n65535\n42\n0\n5\n9\n15\n4\n6\n3.5\n2\n14\n8\n3\n3\n2\n1\n")
	t.RunWant("test-operator-operand-line.cx", types.CX_COMPILATION_ERROR, "an operand type mismatch is reported at its line (skycoin-cx#45)",
		"test-operator-operand-line.cx:8 operands are not of the same type")
	t.RunWant("issue-68.cx", types.CX_COMPILATION_ERROR, "i++ is a statement, not an argument (skycoin-cx#18)", "issue-68.cx:7 syntax error")
	t.RunWant("test-incdec-value.cx", types.CX_COMPILATION_ERROR, "i++ is a statement, not an operand (skycoin-cx#18)", "test-incdec-value.cx:5 syntax error")
	t.RunWant("test-incdec-prefix.cx", types.CX_COMPILATION_ERROR, "there is no prefix ++", "test-incdec-prefix.cx:5 syntax error")
	t.RunWant("issue-62.cx", types.CX_COMPILATION_ERROR, "a comma list is not a statement (skycoin-cx#23)", "issue-62.cx:16 syntax error")
	t.Run("issue-34.cx", types.CX_SUCCESS, "multiple results of a call in := (skycoin-cx#44)")
	t.RunWant("issue-512.cx", types.CX_SUCCESS, "multiple results of a call in := (skycoin-cx#44)", "2\n1\n")
	t.RunWant("issue-512b.cx", types.CX_SUCCESS, "several values in := (skycoin-cx#44)", "20\n10\n")
	t.RunWant("cassignfunc.cx", types.CX_SUCCESS, "multiple results of a call in := (skycoin-cx#44)", "20\n10\n")
	t.RunWant("cassigntwovarialbe.cx", types.CX_SUCCESS, "several values in := (skycoin-cx#44)", "20\n10\n")
	t.RunWant("multi-return-from-function.cx", types.CX_SUCCESS, "multiple results of a call in =", "20\n10\n")
	t.Run("multi-return-from-function-new-varaible.cx", types.CX_SUCCESS, "multiple results of a call in := (skycoin-cx#44)")
	t.RunWant("test-multi-assign.cx", types.CX_SUCCESS, "multiple assignment from values, calls and method calls, with swaps and short-circuit values (skycoin-cx#44)",
		"2 1\n7 seven\n7 seven\nside 1\nside 2\ntrue true\n9 2.5\n0 9\n2 1\n1 two 3.5\n5000000000\n0.25\ntrue\n3\n-1 2 -2\n2 5000000000\ni 0 j 5\ni 1 j 4\ni 2 j 3\n")
	t.RunWant("test-multi-assign-count.cx", types.CX_COMPILATION_ERROR, "a multiple assignment with the wrong number of values",
		"test-multi-assign-count.cx:16 assignment mismatch: 2 variables but 3 values",
		"test-multi-assign-count.cx:17 assignment mismatch: 2 variables but 1 value",
		"test-multi-assign-count.cx:18 assignment mismatch: 3 variables but 2 values",
		"test-multi-assign-count.cx:19 assignment mismatch: 2 variables but 1 value")
	t.RunWant("test-multi-assign-operator.cx", types.CX_COMPILATION_ERROR, "a multiple assignment takes only = and :=", "test-multi-assign-operator.cx:6 syntax error")
	t.RunWant("test-struct-literal-negative.cx", types.CX_SUCCESS, "negative and computed values in struct literal fields", "-1\n-2.5\n-4\n4\n1.5\n-3\n")
	t.RunWant("test-short-declaration-copy.cx", types.CX_SUCCESS, "a short declaration copies an i64, an f64, a field or a struct",
		"5000000000\n7\n0.125\n8\n6000000000\n9\n1\n2\n10\n6000000000\n")
	t.RunWant("issue-35.cx", types.CX_SUCCESS, "slice literal as an argument (skycoin-cx#51)", "3\n3\n")
	t.RunWant("issue-539.cx", types.CX_SUCCESS, "array literal of structs (skycoin-cx#40)", "1\n")
	t.RunWant("issue-526.cx", types.CX_SUCCESS, "[N]*T parameters and variables (skycoin-cx#42)", "1\n")
	t.RunWant("issue-521.cx", types.CX_SUCCESS, "[N]pkg.T variables (skycoin-cx#43)", "5\n")
	t.RunWant("issue-521b.cx", types.CX_COMPILATION_ERROR, "[N]pkg.T of a package that is not imported (skycoin-cx#43)",
		"issue-521b.cx:12 package 'pack' not imported")
	t.RunWant("issue-37.cx", types.CX_SUCCESS, "nested slice literal without inner types (skycoin-cx#50)", "3\n")
	t.RunWant("issue-37b.cx", types.CX_SUCCESS, "nested slice literal with inner types (skycoin-cx#50)", "3\n")
	t.RunWant("test-array-pointer-element.cx", types.CX_SUCCESS, "arrays of pointers as variables, fields, globals, arguments and results (skycoin-cx#42)",
		"4\n4\n6\n6\n9\n9\n6\n")
	t.RunWant("test-array-package-type.cx", types.CX_SUCCESS, "arrays of a struct of another package as variables, fields, globals, arguments and results (skycoin-cx#43)",
		"3\n7\n3\n5\n6\n8\n")
	t.RunWant("test-array-2d-result.cx", types.CX_SUCCESS, "2-D and 3-D array results and arguments", "7\n9\n9\n5\n")
	t.RunWant("test-array-2d-errors.cx", types.CX_COMPILATION_ERROR, "2-D array type errors name the dimensions outermost first",
		"test-array-2d-errors.cx:12 function 'mk' expected receiving variable of type '[3][2]i32'; '[2][3]i32' was provided",
		"function 'take' expected input argument of type '[2][3]i32'; '[3][2]i32' was provided")
	t.RunWant("test-literal-elided.cx", types.CX_SUCCESS, "array and slice literals with elided element types (skycoin-cx#50)",
		"5\n3\n4\n3\n3\n8\n9\n0\n6\n5\n6\n11\n10\n6\n6\n7\n2\n4\n8\n4\n")
	t.RunWant("test-literal-elided-errors.cx", types.CX_COMPILATION_ERROR, "elided literal elements of a type that cannot take one",
		"test-literal-elided-errors.cx:9 invalid composite literal type 'i32'",
		"test-literal-elided-errors.cx:10 struct literal of type 'Point' must name its fields",
		"test-literal-elided-errors.cx:11 invalid composite literal type '[]i32'",
		"test-literal-elided-errors.cx:12 invalid composite literal type 'i32'",
		"test-literal-elided-errors.cx:13 struct literal of type 'Point' must name its fields")
	t.Run("test-struct-literal-value.cx", types.CX_SUCCESS, "a struct literal is a value: var initializer, T{} zeroes, argument, result and a swap of fields")
	t.Run("test-struct-literal-nested.cx", types.CX_SUCCESS, "struct, slice and array literals nested in each other, with calls in the fields")
	t.Run("test-struct-literal-package.cx", types.CX_SUCCESS, "struct literals in global initializers and of another package")
	t.Run("test-array-literal-value.cx", types.CX_SUCCESS, "array literals of structs, as arguments, results and short declarations")
	t.RunWant("test-slice-literal-argument.cx", types.CX_SUCCESS, "slice literals as call arguments (skycoin-cx#51)", "2 1\n")
	t.Run("test-string-slice-pointer.cx", types.CX_SUCCESS, "p = &s on a *[]str takes the address of the slice")
	t.RunWant("test-composite-literal-errors.cx", types.CX_COMPILATION_ERROR, "an array literal longer than its array and a field named twice",
		"test-composite-literal-errors.cx:9 array literal of length 2 has 3 elements",
		"test-composite-literal-errors.cx:11 array literal of length 1 has 2 elements",
		"test-composite-literal-errors.cx:13 duplicate field name 'a' in struct literal")
	t.Run("issue-540a.cx", types.CX_SUCCESS, "continue inside three nested loops (skycoin-cx#39)")
	t.Run("issue-540b.cx", types.CX_SUCCESS, "break inside three nested loops (skycoin-cx#39)")
	t.Run("issue-537.cx", types.CX_SUCCESS, "Crash when a block starts with a for loop and declares a variable")
	t.Run("issue-31.cx", types.CX_SUCCESS, "len of a slice after append, read into a short declaration")
	t.RunWant("issue-157.cx", types.CX_SUCCESS, "a call result as a slice index in a bubble sort (skycoin-cx#29)", "[0, 1, 2, 3, 4, 5, 6, 8, 9, 10]\n")
	t.RunWantInput("issue-110.cx", "abc\ndef\n", types.CX_SUCCESS, "a second read() on a piped stdin (skycoin/cx#110)", "Hello, abc!\ndef\n")
	t.RunWantInput("test-read-eof.cx", "one\ntwo", types.CX_SUCCESS, "read() returns an unterminated last line, then \"\" at the end of input", "[one]\n[two]\n[]\n[]\n")
	t.RunWant("issue-432.cx", types.CX_COMPILATION_ERROR, "sys/gl in a cx built without cxfx (skycoin/cx#432)", "issue-432.cx:6 package \"sys/gl\" needs a GUI build of cx, build it with make build-gui")
	t.RunWant("-h", types.CX_SUCCESS, "cx -h names the real flags (skycoin/cx#567)", "alias for -lexer\n", "alias for -heap-initial\n", "alias for -heap-max\n", "-lexer-output string")
	t.RunWant("test-append-struct-field.cx", types.CX_SUCCESS, "append and len on a slice field of a struct and of a struct pointer",
		"1 7 9000000000 cee\n2 7 9000000000 cee\n2 2.5 3.5\none two\n4 5\n3 3 9000000000 cee\n2 9000000000 cee\n")
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
	t.Run("test-sys-import.cx", types.CX_SUCCESS, "built-in packages imported by their sys/ path (skycoin-cx#380)")
	t.RunWant("test-sys-import-unknown.cx", types.CX_COMPILATION_ERROR, "unknown sys/ import path (skycoin-cx#380)",
		`test-sys-import-unknown.cx:3 unknown built-in package "sys/nosuchpackage"`)
	t.RunWant("test-sys-import-missing.cx", types.CX_COMPILATION_ERROR, "built-in package used without its import (skycoin-cx#380)",
		`test-sys-import-missing.cx:4 built-in package os is used without importing it, add import "sys/os"`)
	t.RunWant("test-sys-import-bare.cx", types.CX_COMPILATION_ERROR, "built-in package imported without sys/ (skycoin-cx#380)",
		`test-sys-import-bare.cx:3 built-in package os is imported as "sys/os"`)
	t.Run("test-sys-string.cx", types.CX_SUCCESS, "sys/string functions (skycoin-cx#382)")
	t.RunWant("test-str-renamed.cx", types.CX_COMPILATION_ERROR, "a renamed str function names its new name (skycoin-cx#382)",
		`test-str-renamed.cx:4 str.concat was renamed to string.Concatenation in language version 2611200002, import "sys/string"`)
	t.RunWant("test-sys-string-not-imported.cx", types.CX_COMPILATION_ERROR, "sys/string used without its import (skycoin-cx#382)",
		`test-sys-string-not-imported.cx:4 built-in package string is used without importing it, add import "sys/string"`)
	t.Run("test-unknown-native-function.cx", types.CX_COMPILATION_ERROR, "unknown function in an imported native package")
	t.RunWant("test-unknown-type-declarations.cx", types.CX_COMPILATION_ERROR, "every unknown type in a declaration is reported, without a crash",
		`test-unknown-type-declarations.cx:4 struct 'st' not found in package 'main'`,
		`test-unknown-type-declarations.cx:7 struct 'i' not found in package 'main'`,
		`test-unknown-type-declarations.cx:9 struct 'foo' not found in package 'main'`,
		`test-unknown-type-declarations.cx:13 struct 'Nope' not found in package 'main'`,
		`test-unknown-type-declarations.cx:14 struct 'Nope' not found in package 'main'`,
		`test-unknown-type-declarations.cx:15 struct 'Nope' not found in package 'main'`)
	t.RunWant("test-unknown-type-imported-array.cx", types.CX_COMPILATION_ERROR, "an array of a type from a package that is not imported",
		`test-unknown-type-imported-array.cx:4 package 'other' not imported`)
	t.RunWant("test-index-too-many.cx", types.CX_COMPILATION_ERROR, "more indexes than dimensions, and indexing a scalar",
		`test-index-too-many.cx:5 invalid indexing`,
		`test-index-too-many.cx:8 invalid indexing`,
		`test-index-too-many.cx:10 invalid indexing`,
		`test-index-too-many.cx:12 invalid indexing`)
	t.RunWant("test-struct-recursive.cx", types.CX_COMPILATION_ERROR, "a struct that holds itself",
		`test-struct-recursive.cx:4 invalid recursive type 'T'`)
	t.RunWant("test-struct-recursive-mutual.cx", types.CX_COMPILATION_ERROR, "two structs that hold each other",
		`test-struct-recursive-mutual.cx:8 invalid recursive type 'B'`)
	t.RunWant("test-method-missing.cx", types.CX_COMPILATION_ERROR, "a call of a method the struct does not have",
		`test-method-missing.cx:9 type 'T' has no method 'Nope'`)
	t.RunWant("test-return-count.cx", types.CX_COMPILATION_ERROR, "return with the wrong number of values",
		`test-return-count.cx:4 function 'f' expects to return 2 arguments, but 1 output argument was provided`,
		`test-return-count.cx:8 function 'main' expects to return 0 argument, but 1 output argument was provided`)
	t.RunWant("test-array-too-large.cx", types.CX_COMPILATION_ERROR, "a struct and a global past the 32 bit size range",
		`test-array-too-large.cx:5 'b' is too large`,
		`test-array-too-large.cx:8 'g' is too large`)
	t.RunWant("test-array-too-large-local.cx", types.CX_COMPILATION_ERROR, "a local array whose size wraps",
		`test-array-too-large-local.cx:4 'l' is too large`)
	t.RunWant("test-array-too-large-frame.cx", types.CX_COMPILATION_ERROR, "locals whose frame passes the 32 bit size range",
		`test-array-too-large-frame.cx:5 'c' is too large`)
	t.RunWant("test-function-missing-in-expression.cx", types.CX_COMPILATION_ERROR, "an unknown native inside an operator",
		`test-function-missing-in-expression.cx:4 function 'i32.nope' does not exist`)
	t.RunWant("test-function-missing-in-condition.cx", types.CX_COMPILATION_ERROR, "an unknown function in a loop condition",
		`test-function-missing-in-condition.cx:4 function 'nope' not found in package 'main' or its imports`)
	t.RunWant("test-append-missing-value.cx", types.CX_COMPILATION_ERROR, "append without a value",
		`test-append-missing-value.cx:5 operator 'append' expects 2 input/s, but 1 input argument`)
	t.RunWant("test-operator-argument-types.cx", types.CX_COMPILATION_ERROR, "a type inferred operator on a type it does not support",
		`test-operator-argument-types.cx:4 invalid argument types for 'lt' operator`,
		`test-operator-argument-types.cx:5 invalid argument types for 'bitand' operator`,
		`test-operator-argument-types.cx:6 invalid argument types for 'lt' operator`)
	t.Run("test-function-return-array.cx", types.CX_SUCCESS, "functions that return arrays")
	t.Run("test-imported-global-assign.cx", types.CX_SUCCESS, "assignment to a global of an imported package")
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
	t.Run("issue-234.cx", types.CX_COMPILATION_ERROR, "Panic when trying to access an invalid field. (skycoin-cx#14)")
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
	t.RunWant("issue-251.cx", types.CX_SUCCESS, "Panic when a str var is shadowed by a struct var in another scope", "test\n")
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
	t.Run("issue-279a.cx", types.CX_SUCCESS, "False positive when detecting variable redeclaration.")
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
	t.RunEx("issue-67.cx", types.CX_COMPILATION_ERROR, "Panic when using void return value of a function in a for loop expression", runner.TestIssue, 0)
	t.Run("issue-66.cx", types.CX_SUCCESS, "Wrong sprintf behaviour when printing boolean values with %v")
	t.Run("issue-65.cx", types.CX_SUCCESS, "for true {} loop scope is not executed")
	t.Run("issue-64.cx", types.CX_SUCCESS, "for loop using boolean value is not compiling")
	t.Run("issue-303.cx", types.CX_SUCCESS, "Concatenation of str variables with + operator doesn't work")
	t.Run("issue-304.cx", types.CX_SUCCESS, "Short declaration doesn't compile with opcode return value")
	t.Run("issue-305.cx", types.CX_SUCCESS, "Compilation error when struct field is named 'input' or 'output'")
	t.RunWant("issue-63.cx", types.CX_COMPILATION_ERROR, "No compilation error when using empty argument list after function call (skycoin-cx#22)",
		"issue-63.cx:10 the result of a call is not a function")
	t.Run("issue-306.cx", types.CX_COMPILATION_ERROR, "No compilation error when using float value in place of boolean expression")
	t.Run("issue-308.cx", types.CX_COMPILATION_ERROR, "Panic when package contains duplicate function signature")
	t.Run("issue-61-a.cx", types.CX_SUCCESS, "Compilation error when using return value of a member method in a expression (skycoin-cx#24)")
	t.Run("issue-61-b.cx", types.CX_SUCCESS, "Compilation error when using return value of a member method in a expression (skycoin-cx#24)")
	t.Run("issue-309.cx", types.CX_SUCCESS, "Compilation error when left hand side of an assignment expression is a struct field")
	t.RunEx("--heap-initial=1M issue-60.cx", types.CX_SUCCESS, "Crash when INIT_HEAP_SIZE limit is reached ", runner.TestIssue, 0)
	t.RunEx("--heap-initial=1M issue-59-a.cx", types.CX_SUCCESS, "Crash in garbage collector when heap is resized", runner.TestIssue, 0)
	t.RunEx("--heap-initial=1M issue-59-b.cx", types.CX_SUCCESS, "Crash in garbage collector when heap is resized", runner.TestIssue, 0)
	t.RunEx("issue-53-a.cx", types.CX_SUCCESS, "Issues with slice of type T where sizeof T is different than 4 ", runner.TestStable, 0)
	t.RunEx("issue-53-b.cx", types.CX_SUCCESS, "Issues with slice of type T where sizeof T is different than 4 ", runner.TestStable, 0)
	t.RunEx("issue-53-c.cx", types.CX_SUCCESS, "Issues with slice of type T where sizeof T is different than 4 ", runner.TestStable, 0)
	t.RunEx("issue-51.cx", types.CX_COMPILATION_ERROR, "No compilation error when global variable is redeclared at local scope", runner.TestIssue, 0) // known failing, skycoin/skycoin-cx#25
	t.RunEx("issue-49.cx", types.CX_COMPILATION_ERROR, "No compilation error when assigning an literal which overflow the receiving type", runner.TestIssue, 0)
	t.Run("issue-48.cx", types.CX_SUCCESS, "Cx is not supporting short-circuit evaluation")
	t.Run("issue-39.cx", types.CX_COMPILATION_ERROR, "func defined with no arguments, called WITH arguments causes PANIC")
	t.Run("issue-2.cx", types.CX_SUCCESS, "multi-dimensional arrays are not working")
	t.Run("issue-1.cx", types.CX_SUCCESS, "multi-dimensional slices are not working")
	t.Run("issue-177/structofslicechar.cx", types.CX_SUCCESS, "len of a struct slice literal does not depend on its strings (skycoin-cx#47)")
	t.Run("issue-177/structofsliceint32.cx", types.CX_SUCCESS, "len of a struct slice literal (skycoin-cx#47)")
	t.Run("issue-177/structofslicestring.cx", types.CX_SUCCESS, "len of a struct slice literal with a str field (skycoin-cx#47)")
	t.Run("issue-177/structofslicestrings.cx", types.CX_SUCCESS, "len of a struct slice literal with another str field (skycoin-cx#47)")
	t.Run("issue-120-a.cx", types.CX_COMPILATION_ERROR, "Invalid implicit cast when assigning the result of a math operator to a variable. (skycoin-cx#27)")
	t.Run("issue-120-b.cx", types.CX_COMPILATION_ERROR, "Invalid implicit cast when assigning the result of a math operator to a variable. (skycoin-cx#27)")
	t.Run("issue-120-c.cx", types.CX_COMPILATION_ERROR, "Invalid implicit cast when assigning the result of a math operator to a variable. (skycoin-cx#27)")
	t.RunEx("issue-131.cx", types.CX_SUCCESS, "Panic when using arithmetic operations.", runner.TestIssue, 0)
	t.RunEx("test-ar-1.cx", types.CX_SUCCESS, "Panic when using string to pointer array.", runner.TestIssue, 0)
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
