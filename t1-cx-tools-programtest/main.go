package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t0-cx-tools-testrunner"
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

	var parseBitMask = func(flagName string, bitsMap map[string]testrunner.Bits, defaultBit testrunner.Bits) testrunner.Bits {
		var mask testrunner.Bits = 0
		flags := strings.Split(c.String(flagName), ",")
		for _, flag := range flags {
			mask = testrunner.Set(mask, bitsMap[flag])
		}
		if debug {
			fmt.Printf("Parsed bit mask for flag %s%s: %06b\n", flagName, flags, mask)
		}
		return mask
	}

	logMask := parseBitMask("log", testrunner.LogBits, testrunner.LogNone)
	enableTestsMaks := parseBitMask("enable-tests", testrunner.TestBits, testrunner.TestNone)
	disabledTestsMaks := parseBitMask("disable-tests", testrunner.TestBits, testrunner.TestNone)

	if enableTestsMaks == testrunner.TestAll && disabledTestsMaks == testrunner.TestAll {
		return errors.New("invalid test flags combination")
	}

	var testsMask testrunner.Bits = testrunner.TestAll
	if debug {
		fmt.Printf("Initial test mask: %06b\n", testsMask)
	}
	// turn on only enabled tests
	if enableTestsMaks != testrunner.TestNone {
		testsMask = testsMask & enableTestsMaks
	}
	// turn off only disabled test
	testsMask = testsMask &^ disabledTestsMaks
	if debug {
		fmt.Printf("Resulting test mask: %06b\n", testsMask)
	}

	tester := testrunner.TestRunnerNew(&testrunner.Config{
		CxPath:         cxPath,
		WorkingDir:     workingDir,
		TestsMask:      testsMask,
		LogMask:        logMask,
		DefaultTimeout: 300 * time.Second,
	})

	var start = time.Now().Unix()

	fmt.Printf("Running CX tests in dir: '%s'\n", workingDir)
	runTestCases(tester)
	runFileWriteTests(tester, workingDir)
	runGuideSnippetTests(tester, workingDir)
	runCompoundTypeTests(tester, workingDir)
	end := time.Now().Unix()

	if testrunner.Has(logMask, testrunner.LogTime) {
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

func runTestCases(t *testrunner.TestRunner) {
	// tests
	t.Run("test-i8.cx", valuetypes.CXSuccess, "i32")
	t.Run("test-i16.cx", valuetypes.CXSuccess, "i32")
	t.Run("test-i32.cx", valuetypes.CXSuccess, "i32")
	t.Run("test-i64.cx", valuetypes.CXSuccess, "i64")
	t.Run("test-ui8.cx", valuetypes.CXSuccess, "i32")
	t.Run("test-ui16.cx", valuetypes.CXSuccess, "i32")
	t.Run("test-ui32.cx", valuetypes.CXSuccess, "i32")
	t.Run("test-ui64.cx", valuetypes.CXSuccess, "i64")
	t.Run("test-f32.cx", valuetypes.CXSuccess, "f32")
	t.Run("test-f64.cx", valuetypes.CXSuccess, "f64")
	t.Run("test-bool.cx", valuetypes.CXSuccess, "bool")
	t.Run("test-array.cx", valuetypes.CXSuccess, "array")
	t.Run("test-function.cx", valuetypes.CXSuccess, "function")
	t.Run("test-control-flow.cx", valuetypes.CXSuccess, "control floow")
	t.Run("test-utils.cx test-struct.cx", valuetypes.CXSuccess, "struct")
	t.Run("test-str.cx", valuetypes.CXSuccess, "str")
	t.Run("test-str-compare.cx", valuetypes.CXSuccess, "str ordering with string.OrderingByByteValue (skycoin-cx#87, #183)")
	t.Run("test-str-functions.cx", valuetypes.CXSuccess, "str comparison functions (skycoin-cx#183)")
	t.RunWant("test-str-operators-removed.cx", valuetypes.CXCompilationError, "every operator on str is a compile error (skycoin-cx#183)",
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
	t.RunWant("test-pointer-ordering-removed.cx", valuetypes.CXCompilationError, "pointer ordering is a compile error (skycoin-cx#183)",
		"test-pointer-ordering-removed.cx:14 pointer < pointer is not allowed, pointers support only == and !=",
		"test-pointer-ordering-removed.cx:15 pointer > pointer is not allowed, pointers support only == and !=",
		"test-pointer-ordering-removed.cx:16 pointer <= pointer is not allowed, pointers support only == and !=",
		"test-pointer-ordering-removed.cx:17 pointer >= pointer is not allowed, pointers support only == and !=",
	)
	t.Run("test-pointer-equality.cx", valuetypes.CXSuccess, "pointer == and != compare addresses (skycoin-cx#183)")
	t.Run("--enable=file test-os-getwd.cx", valuetypes.CXSuccess, "os.GetWorkingDirectory (skycoin-cx#117)")
	t.Run("--enable=file test-os-sandbox.cx", valuetypes.CXSuccess, "program paths stay inside the sandbox (skycoin-cx#433)")
	t.Run("--enable=process test-os-run-empty.cx", valuetypes.CXSuccess, "os.Run with an empty command returns OS_RUN_EMPTY_CMD")
	t.Run("test-cx-constants.cx", valuetypes.CXSuccess, "cx package constants (skycoin-cx#84)")
	t.RunWant("--enable=net --executor=old test-net-tcp.cx", valuetypes.CXSuccess, "TCP through the batched net library on the old loop (skycoin-cx#508)", "server read 2 bytes, first 104\nclient read 3 bytes, last 33\nclosed\n")
	t.RunWant("--enable=net --executor=vm test-net-tcp.cx", valuetypes.CXSuccess, "TCP through the batched net library on vm (skycoin-cx#508)", "server read 2 bytes, first 104\nclient read 3 bytes, last 33\nclosed\n")
	t.RunWant("--enable=net --executor=old test-net-udp.cx", valuetypes.CXSuccess, "UDP through the batched net library on the old loop (skycoin-cx#508)", "4\nserver got 4 bytes from peer 1, first 112\n1\nclient got 1 bytes, first 113\n")
	t.RunWant("--enable=net --executor=vm test-net-udp.cx", valuetypes.CXSuccess, "UDP through the batched net library on vm (skycoin-cx#508)", "4\nserver got 4 bytes from peer 1, first 112\n1\nclient got 1 bytes, first 113\n")
	t.RunWant("--enable=file,net --executor=vm test-library-call.cx", valuetypes.CXSuccess, "system library calls with scalar operands run on vm (skycoin-cx#492)", "7\n")
	t.RunWant("--enable=file,net --executor=old test-library-call.cx", valuetypes.CXSuccess, "system library calls run on the old loop (skycoin-cx#492)", "7\n")
	t.RunWant("--executor=vm test-library-disabled.cx", valuetypes.CXRuntimeLibraryDisabled, "a program that uses a library that is not enabled is refused before vm runs it (D119)", "error: CX_RUNTIME_LIBRARY_DISABLED, library file is not enabled, enable it with --enable=file")
	t.RunWant("--executor=old test-library-disabled.cx", valuetypes.CXRuntimeLibraryDisabled, "a program that uses a library that is not enabled is refused before the old loop runs it (D119)", "error: CX_RUNTIME_LIBRARY_DISABLED, library file is not enabled, enable it with --enable=file")
	t.RunWant("--enable=net,process test-library-disabled.cx", valuetypes.CXRuntimeLibraryDisabled, "enabling other libraries leaves file disabled (skycoin-cx#492)", "library file is not enabled")
	t.RunWant("--enable=file test-library-disabled.cx", valuetypes.CXSuccess, "an enabled library runs (skycoin-cx#492)", "1\n2\n")
	t.RunWant("--enable=cipher test-cipher-hash-state-vm.cx", valuetypes.CXSuccess, "a library entry of --enable enables every operation of the library (D119)", "0 1 0\n")
	t.RunWant("--enable=cipher.HashStateNew,cipher.HashStateRelease test-cipher-hash-state-vm.cx", valuetypes.CXSuccess, "operation entries of --enable enable those operations (D119)", "0 1 0\n")
	t.RunWant("--enable=file,cipher.HashStateNew,cipher.HashStateRelease test-cipher-hash-state-vm.cx", valuetypes.CXSuccess, "--enable mixes library and operation entries (D119)", "0 1 0\n")
	t.RunWant("--enable=cipher.HashStateNew test-cipher-hash-state-vm.cx", valuetypes.CXRuntimeLibraryDisabled, "a program is refused for one operation that is not enabled while others of its library are (D119)", "operation cipher.HashStateRelease is not enabled, enable it with --enable=cipher.HashStateRelease")
	t.RunWant("--disable=terminal test-terminal-read.cx", valuetypes.CXRuntimeLibraryDisabled, "a program that reads the terminal is refused when --disable names the library (D142)", "error: CX_RUNTIME_LIBRARY_DISABLED, library terminal is not enabled, enable it with --enable=terminal")
	t.RunWant("--disable=terminal.InputLineRead test-terminal-read.cx", valuetypes.CXRuntimeLibraryDisabled, "a program is refused for the one terminal operation --disable names (D142)", "operation terminal.InputLineRead is not enabled, enable it with --enable=terminal.InputLineRead")
	t.RunWant("--output-mode=instant test-output-flush.cx", valuetypes.CXSuccess, "instant output writes every print and prints what batch prints (D142)", "line 0 text\n0\n1.5\nfalse\nline 1 text\n1\n1.5\ntrue\nline 2 text\n2\n1.5\nfalse\ndone\n")
	t.RunWant("--output-mode=thread test-output-flush.cx", valuetypes.CXSuccess, "thread output queues every print for a second thread and prints what batch prints (D142)", "line 0 text\n0\n1.5\nfalse\nline 1 text\n1\n1.5\ntrue\nline 2 text\n2\n1.5\nfalse\ndone\n")
	t.RunWant("--output-mode=fast test-output-flush.cx", valuetypes.CXCompilationError, "an unknown output mode is an error (D142)", `unknown --output-mode "fast"`)
	t.RunWant("--enable=cipher.NoSuch test-cipher-hash-state-vm.cx", valuetypes.CXCompilationError, "an unknown operation in --enable is an error (D119)", "unknown library or operation \"cipher.NoSuch\" in --enable")
	t.RunWant("test-cipher-hash-state-vm.cx", valuetypes.CXRuntimeLibraryDisabled, "a program that uses a library it does not enable is refused before it runs (D119)", "error: CX_RUNTIME_LIBRARY_DISABLED, library cipher is not enabled, enable it with --enable=cipher")
	t.RunWant("test-os-getwd.cx", valuetypes.CXRuntimeLibraryDisabled, "a str library call that is not enabled stops the old loop (skycoin-cx#492)", "library file is not enabled, enable it with --enable=file")
	t.RunWant("test-net-tcp.cx", valuetypes.CXRuntimeLibraryDisabled, "sys/net is the net library (skycoin-cx#492)", "library net is not enabled, enable it with --enable=net")
	t.RunWant("--net-record=a.netreplay --net-replay=b.netreplay test-net-tcp.cx", valuetypes.CXCompilationError, "a run records or replays the net library, not both (skycoin-cx#508)", "--net-record and --net-replay cannot both be given")
	t.RunWant("test-os-run-empty.cx", valuetypes.CXRuntimeLibraryDisabled, "os.Run is the process library (skycoin-cx#492)", "library process is not enabled, enable it with --enable=process")
	t.RunWant("--enable=cipher test-cipher-library.cx", valuetypes.CXSuccess, "the cipher library hashes with the published vectors and verifies, recovers and makes Skycoin signatures (skycoin-cx#510)",
		"af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262\n2d3adedff11b61f14c886e35afa036736dcd87a74d27b5c1510225d0f592e213\n4eed7141ea4a5cd4b788606bd23f46e212af9cacebacdc7d1f4c6dc7f2511b98\nde1e5fa0be70df6d2be8fffd0e99ceaa8eb6e8c93a63f2d8d1c30ecb6b263dee\n42214739f095a406f3fc83deb889744ac00df831c10daa55189b5d121c855af7\nd00278ae47eb27b34faecf67b4fe263f82d5412916c1ffd97c8cb7fb814b8444\n5f4d72f40d7a5f82b15ca2b2e44b1de3c2ef86c426c95c1af0b6879522563030\n"+
			"42214739f095a406f3fc83deb889744ac00df831c10daa55189b5d121c855af7\nd00278ae47eb27b34faecf67b4fe263f82d5412916c1ffd97c8cb7fb814b8444\n"+
			"verify true\nverify other hash false\nverify changed signature false\nverify short key false\n"+
			"recover true 02f382589c67fbac7065b5eff66f35c30bc73e385ffeab46326ec1a930b10d1c53\nrecover from a short signature false 0\n"+
			"sign true 65\nverify signed true\nverify signed with the other hash false\nsign with a public key false 0\n")
	t.RunWant("--enable=cipher test-cipher-keypair.cx", valuetypes.CXSuccess, "a generated key pair signs and verifies (skycoin-cx#510)", "33 32 true 65 true\ntrue true\n")
	t.RunWant("--enable=cipher --executor=vm test-cipher-hash-state-vm.cx", valuetypes.CXSuccess, "cipher functions with i32 operands run on vm (skycoin-cx#510)", "0 1 0\n")
	t.RunWant("--enable=cipher --executor=old test-cipher-hash-state-vm.cx", valuetypes.CXSuccess, "cipher hash states on the old loop (skycoin-cx#510)", "0 1 0\n")
	t.RunWant("test-cipher-library.cx", valuetypes.CXRuntimeLibraryDisabled, "sys/cipher is the cipher library (skycoin-cx#510)", "library cipher is not enabled, enable it with --enable=cipher")
	t.RunWant("--enable=file,nosuch test-library-call.cx", valuetypes.CXCompilationError, "an unknown library in --enable (skycoin-cx#492)", `unknown library or operation "nosuch" in --enable, the libraries are file, net, process, cipher`)
	t.Run("test-utils.cx test-pointers.cx", valuetypes.CXSuccess, "pointers")
	t.Run("test-slices.cx", valuetypes.CXSuccess, "slices")
	t.Run("test-slices-append-multi.cx", valuetypes.CXSuccess, "append with several values")
	t.Run("test-slices-of-slices-literal.cx", valuetypes.CXSuccess, "slice of slices literal with two inner literals")
	t.RunWant("test-slice-nested-call.cx", valuetypes.CXSuccess, "len, indexing and slice built-ins of a nested opcode or call result",
		"3\n4\n5\n3\n4\n4\n3\n4\n6\n65\n66\n68\n195\n3\n2\n2\n3\n6\n66\n3\n2\n4\n2\n2\n2\n5\n4\nround trip\n7\n")
	t.RunWant("--enable=cipher test-slice-nested-library-call.cx", valuetypes.CXSuccess, "len and indexing of a nested library call result",
		"32\n32\n32\n33\n32\n")
	t.RunWant("test-printf-v.cx", valuetypes.CXSuccess, "printf %v of structs, arrays and slices of structs and array pointers (skycoin-cx#256)",
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
	t.RunWant("test-struct-pointer-deref.cx", valuetypes.CXSuccess, "*p on a struct pointer is the struct value (skycoin-cx#261)",
		"deref {x: 1, y: 2}\n",
		"local {x: 3, y: 4}\n",
	)
	t.RunWant("test-slice-pointer.cx", valuetypes.CXSuccess, "taking, passing and dereferencing the address of a slice (skycoin-cx#262)",
		"t [4]\n",
		"deref [4]\n",
		"grown [4, 5]\n",
		"assigned [9]\n",
		"filled [9, 7, 7]\n",
		"set [4, 5]\n",
		"global [1]\n",
	)
	t.RunWant("test-slice-pointer-index.cx", valuetypes.CXSuccess, "(*p)[i] and len(*p) on a pointer to a slice (skycoin-cx#49, skycoin-cx#267)",
		"3\n2\n3\n3\nlen 2 sum 8\nbump [3, 15]\nvar 15 plus 4\nf64 1.5 2.5 2\nui8 [7] 7\nstruct 5 40 2\n")
	t.RunWant("test-slice-pointer-index-errors.cx", valuetypes.CXCompilationError, "p[i] and *p[i] on a pointer to a slice are invalid indexing (skycoin-cx#49)",
		"test-slice-pointer-index-errors.cx:9 invalid indexing",
		"test-slice-pointer-index-errors.cx:10 invalid indexing")
	t.RunWant("test-append-slice-pointer.cx", valuetypes.CXCompilationError, "append, resize and remove of a pointer to a slice are compile errors (skycoin-cx#52)",
		"test-append-slice-pointer.cx:6 function 'append' expected a slice as input argument; '*[]i32' was provided",
		"test-append-slice-pointer.cx:7 function 'resize' expected a slice as input argument; '*[]i32' was provided",
		"test-append-slice-pointer.cx:8 function 'remove' expected a slice as input argument; '*[]i32' was provided")
	t.RunWant("test-append-slice-pointer-deref.cx", valuetypes.CXSuccess, "append(*p, v) appends to the slice p points to (skycoin-cx#52)",
		"3 [4, 5, 6] [4, 5, 6]\n")
	t.RunWant("test-slice-literal-struct.cx", valuetypes.CXSuccess, "slice literals of structs and empty slice literals (skycoin-cx#47)",
		"len 1\nlen 1 hmmmm\n4 [{x: 3, y: 1.5}, {x: 9, y: 0}, {x: 7, y: 0}, {x: 0, y: 0}]\nloop 1 2\nmany 7 28\ndecl 1 8\nglobal 2 [{x: 1, y: 0}, {x: 2, y: 2.5}]\nempty 0 0\n")
	t.RunWant("test-slice-literal-struct-errors.cx", valuetypes.CXCompilationError, "elements of the wrong type in a struct slice literal (skycoin-cx#47)",
		"test-slice-literal-struct-errors.cx:20:11 cannot use value of type 'U' as type 'T' in slice literal",
		"test-slice-literal-struct-errors.cx:21:11 cannot use value of type 'U' as type 'T' in slice literal",
		"test-slice-literal-struct-errors.cx:22:11 cannot use value of type 'i32' as type 'T' in slice literal",
		"test-slice-literal-struct-errors.cx:23:11 cannot use value of type 'U' as type 'T' in slice literal",
		"test-slice-literal-struct-errors.cx:24 field 'n' in struct literal of type 'T' expected argument of type 'i32'; 'str' was provided",
		"test-slice-literal-struct-errors.cx:25 field 'n' in struct literal of type 'T' expected argument of type 'i32'; 'str' was provided",
		"test-slice-literal-struct-errors.cx:26:13 cannot use value of type 'T' as type 'i32' in slice literal",
		"test-slice-literal-struct-errors.cx:27:20 cannot use value of type 'U' as type 'T' in slice literal",
		"test-slice-literal-struct-errors.cx:29:3 cannot use value of type 'U' as type 'T' in slice literal")
	t.RunWant("test-call-index-operator.cx", valuetypes.CXSuccess, "f()[i] of an array or slice result as an operator operand",
		"7\n9\n42\n2\n6 14 2.5\ncond\n")
	t.RunWant("test-import-global-array-number.cx test-import-global-array.cx", valuetypes.CXSuccess, "pkg.A[i] reads and writes a global array or slice of another package",
		"4 7 14\n1.5 3\n20 2 12\n[0, 4, 7] [11, 20] 7\n")
	t.RunWant("test-slice-resize-grow.cx", valuetypes.CXSuccess, "resize past twice the capacity allocates the whole slice (audit A-04)",
		"64 1111 2222 2\n[100, 101, 102, 103, 104, 105, 106, 107, 108, 109] [5, 5, 5, 5, 5, 5]\n100 2 3\n")
	t.RunWant("test-slice-resize-huge.cx", valuetypes.CXRuntimeHeapExhaustedError, "a slice too large for the heap is a heap exhausted error (audit A-05)",
		"test-slice-resize-huge.cx:5, CX_RUNTIME_HEAP_EXHAUSTED_ERROR")
	t.RunWant("--enable=file test-os-read-slice-grow.cx", valuetypes.CXSuccess, "os.ReadI32Slice allocates the whole slice and returns false for a count past the heap (audit B-01, B-12)",
		"read true len 10 same true\nhuge false len 0\n")
	t.RunWant("test-test-line.cx", valuetypes.CXAssert, "a failing test() reports its own line (skycoin-cx#254)",
		"test-test-line.cx: 4: result was not equal to the expected value; first statement",
		"test-test-line.cx: 6: result was not equal to the expected value; after a declaration",
		"test-test-line.cx: 8: result was not equal to the expected value; after an assignment",
	)
	t.RunWant("test-vm-divide-callee.cx", valuetypes.CXRuntimeError, "a division by zero in a callee prints its line and the call stack (skycoin-cx#408)",
		"5\nerror: test-vm-divide-callee.cx:8, CX_RUNTIME_ERROR, runtime error: integer divide by zero\n===Callstack===\n>>> divide()",
		">>> middle()",
		">>> main()",
	)
	t.RunWant("test-vm-test-callee.cx", valuetypes.CXAssert, "a failed test in a callee prints its message (skycoin-cx#408)",
		"1\ntest-vm-test-callee.cx: 8: result was not equal to the expected value; local message\n",
		"test-vm-test-callee.cx: 9: result was not equal to the expected value\n",
		"test-vm-test-callee.cx: 10: result was not equal to the expected value; literal message\n2\n",
	)
	t.RunWant("test-vm-stack-overflow.cx", valuetypes.CXRuntimeStackOverflowError, "large frames overflow the stack (skycoin-cx#408)",
		"error: test-vm-stack-overflow.cx:143, CX_RUNTIME_STACK_OVERFLOW_ERROR, stack overflow\n===Callstack===\n>>> recur()",
	)
	t.RunWant("test-vm-call-depth.cx", valuetypes.CXRuntimeStackOverflowError, "the call depth limit is a stack overflow (skycoin-cx#408)",
		"1\nerror: test-vm-call-depth.cx:7, CX_RUNTIME_STACK_OVERFLOW_ERROR, stack overflow\n===Callstack===\n>>> recur()",
	)
	t.RunWant("test-vm-init-divide.cx", valuetypes.CXRuntimeError, "a division by zero in a global initializer is a runtime error (skycoin-cx#408)",
		"error: test-vm-init-divide.cx:7, CX_RUNTIME_ERROR, runtime error: integer divide by zero\n===Callstack===\n>>> *init()",
	)
	t.RunWant("test-vm-array.cx", valuetypes.CXSuccess, "arrays in the stack and data segment (skycoin-cx#408)",
		"16\n30\n23\n23\n7\n5\n",
	)
	t.RunWant("test-vm-struct.cx", valuetypes.CXSuccess, "struct fields, arrays of structs and struct parameters (skycoin-cx#408)",
		"12\n9\n6\n11\n12\n5\n9\n",
	)
	t.RunWant("test-vm-pointer.cx", valuetypes.CXSuccess, "pointers to scalars, fields and elements in the stack and data segment (skycoin-cx#408)",
		"42\n43\n2\n33\n7\n9\n8\n",
	)
	t.RunWant("test-vm-print.cx", valuetypes.CXSuccess, "string.PrintLine and printf (skycoin-cx#408)",
		"hello\nfrom a global\nfrom a field\n\n",
		"-12 1234567890123 200\n1.5000000 2.2500000000000000\n-12 1.5 2.25 true from a field\n",
		"from a field has 3\ntrue false\n100% done, a \\n escape\nmissing -12 and %!d(MISSING)\n",
		"-12 on a float is empty: []\nelement 77\nthrough a pointer 77 77\n",
	)
	t.RunWant("test-array-index-negative.cx", valuetypes.CXRuntimeInvalidCast, "a negative array index is a runtime error and the call stack prints without a panic",
		"1\nerror: test-array-index-negative.cx:7, CX_RUNTIME_INVALID_CAST, 11\n===Callstack===\n>>> store()\n",
		"\ttest-array-index-negative.cx:7 : identity() : <index out of range>\n\n>>> main()\n",
	)
	for _, length := range []string{"128", "100"} {
		for _, scope := range []string{"local", "global"} {
			line, after := "7", ""
			if scope == "global" {
				line, after = "11", "0\n"
			}
			name := "test-array-bounds-" + scope + "-" + length
			t.RunWant(name+"-end.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "an index at the length of a "+scope+" ["+length+"]i32 is a runtime error",
				"5\n"+after+"error: "+name+"-end.cx:"+line+", CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, 9\n")
			t.RunWant(name+"-negative.cx", valuetypes.CXRuntimeInvalidCast, "a negative index of a "+scope+" ["+length+"]i32 is a runtime error",
				"5\n"+after+"error: "+name+"-negative.cx:"+line+", CX_RUNTIME_INVALID_CAST, 11\n")
			if length == "100" {
				t.RunWant(name+"-gap.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "index 127 of a "+scope+" [100]i32, below the next power of two, is a runtime error",
					"5\n"+after+"error: "+name+"-gap.cx:"+line+", CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, 9\n")
			}
		}
	}
	t.RunWant("test-array-bounds-2d.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "the inner index of a [3][5]i32 is checked against 5",
		"5\n0\nerror: test-array-bounds-2d.cx:7, CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE, 9\n")
	t.RunWant("test-stack-print-uninit.cx", valuetypes.CXRuntimeError, "the call stack prints a str local whose declaration never ran without a panic (skycoin-cx#407)",
		"error: test-stack-print-uninit.cx:20, CX_RUNTIME_ERROR, runtime error: integer divide by zero\n===Callstack===\n>>> fail()\n",
		"\ttest-stack-print-uninit.cx:26 : printf() : <invalid address>\n",
		"\n>>> main()\n",
	)
	t.RunWant("test-printf-extra.cx", valuetypes.CXSuccess, "printf prints values past the last verb as %!(EXTRA ...)",
		"none\n%!(EXTRA i32=1, i32=2)5\n%!(EXTRA i64=1234567890123, bool=true, str=text)5 and\n%!(EXTRA i32=7, i32=9, f64=2.5)",
	)
	t.RunWant("test-printf-extra-values.cx", valuetypes.CXSuccess, "printf prints arrays, structs and fields past the last verb as %!(EXTRA ...)",
		"1\n%!(EXTRA i32=[0, 7, 0], point={x: 0, y: 9}, i32=9)",
	)
	t.RunWant("test-test-type-line.cx", valuetypes.CXCompilationError, "a test(), panic or operator type error names the line of the call (skycoin-cx#259)",
		"test-test-type-line.cx:8 first and second input arguments' types are not equal in 'test' call",
		"test-test-type-line.cx:11 first and second input arguments' types are not equal in 'panic' call",
		"test-test-type-line.cx:15 invalid argument types for 'add' operator",
	)

	t.Run("--cxpath test-workspace test-workspace-a.cx", valuetypes.CXSuccess, "Testing if CX can set a workspace and then import a library, taking that workspace as the new relative path.")
	t.Run("--cxpath test-workspace test-workspace-b.cx", valuetypes.CXSuccess, "Testing if CX can set a workspace and then import a nested library, taking that workspace as the new relative path.")
	t.Run("--cxpath test-workspace test-workspace-c.cx test-workspace-d.cx", valuetypes.CXSuccess, "Testing if files supplied to the CLI override libraries in the workspace.")
	t.RunWant("--cxpath test-workspace test-import-twice-a.cx test-import-twice-b.cx", valuetypes.CXSuccess, "two files that import math/geometry compile it once",
		"38.4844780\n6.0000000\n")
	t.RunWant("--cxpath test-workspace test-import-twice-c.cx", valuetypes.CXSuccess, "main and an imported package that both import math/geometry compile it once",
		"3.1415901 12.5663605\n")
	t.RunWant("--cxpath test-workspace test-struct-literal-external-ref.cx", valuetypes.CXSuccess, "&pkg.T{...} in an assignment, argument, return, append, field and index",
		"assign 1.0000000 2.0000000 2.5000000\nargument 6.0000000\nreturn 4.0000000\nappend 5.0000000\nfield 6.0000000 7.0000000\nindex 9.0000000\nvalues 3.0000000 3.0000000\nmethod 3.1415901 8.0000000\n")
	t.RunWant("test-struct-literal-method.cx", valuetypes.CXSuccess, "a selector, method call or index after a struct literal, T{a: 1}.m()",
		"10\n16\n5\n3\n6\n12\n18\n6\n8\n")
	t.RunWant("test-struct-literal-operand.cx", valuetypes.CXSuccess, "a struct literal postfix is an operand of operators, calls, literals and returns, and parenthesized in a header",
		"11\n-2\n6\n31\n54\n59\n5\n6\n6\n12\n11\n12\n0\n1\n")
	t.RunWant("test-struct-literal-header.cx", valuetypes.CXCompilationError, "a bare struct literal in an if header is not an operand, so { opens the body (skycoin-cx#21)",
		"test-struct-literal-header.cx:8:12 syntax error: unexpected .")
	t.RunWant("test-single-line-forms.cx", valuetypes.CXSuccess, "struct, function and block forms on one line, an unparenthesized result type and a comment holding a newline",
		"5\n6\n7\n10\n42\n9\n5\ntrue\n3\n0\n6\n1\n")
	t.RunWant("test-result-unnamed-type.cx", valuetypes.CXCompilationError, "a value of the wrong type returned as an unnamed result",
		"test-result-unnamed-type.cx:3:23 cannot use value of type 'bool' as type 'i32' in return statement")
	t.RunWant("test-array-literal-element-type.cx", valuetypes.CXCompilationError, "an array literal element of the wrong type is reported at the element",
		"test-array-literal-element-type.cx:18:3 cannot use value of type 'U' as type 'T' in array literal",
		"test-array-literal-element-type.cx:19:29 cannot use value of type 'U' as type 'T' in array literal",
		"test-array-literal-element-type.cx:20:29 cannot use value of type 'U' as type 'T' in array literal",
		"test-array-literal-element-type.cx:22:3 cannot use value of type 'U' as type 'T' in slice literal")
	t.RunWant("test-method-pointer-literal.cx", valuetypes.CXCompilationError, "a pointer method of a literal or a call result, which is not addressable",
		"test-method-pointer-literal.cx:20:20 cannot call pointer method p on T",
		"test-method-pointer-literal.cx:21:17 cannot call pointer method p on T",
		"test-method-pointer-literal.cx:22:28 cannot call pointer method p on T")
	t.RunWant("test-method-pointer-addressable.cx", valuetypes.CXSuccess, "a pointer method of &T{}, a variable, a field and a value reached through a pointer",
		"105\n106\n107\n102\n104\n108\n")
	t.RunWant("test-method-pointer-value-receiver.cx", valuetypes.CXSuccess, "a method of value receiver called on a pointer gets a copy of the value",
		"104 4\n5\n107 7\n8\n101 1\n101 1\n105\n101\n")
	t.Run("test-slices-index-out-of-range-a.cx", valuetypes.CXRuntimeInvalidCast, "Test index < 0")
	t.Run("test-slices-index-out-of-range-b.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "Test index >= len")
	t.Run("test-slices-resize-out-of-range-a.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "Test out of range after resize")
	t.Run("test-slices-resize-out-of-range-b.cx", valuetypes.CXRuntimeInvalidCast, "Test resize with count < 0")
	t.Run("test-slices-insert-out-of-range-a.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "Test insert with index > len")
	t.Run("test-slices-insert-out-of-range-b.cx", valuetypes.CXRuntimeInvalidCast, "Test insert with index < 0")
	t.Run("test-slices-remove-out-of-range-a.cx", valuetypes.CXRuntimeInvalidCast, "Test remove with index < 0")
	t.Run("test-slices-remove-out-of-range-b.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "Test remove with index >= len")
	t.Run("test-slices-remove-out-of-range-c.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "Test remove with index == 0 && len == 0")
	t.RunWant("test-slices-remove-other-slice.cx", valuetypes.CXSuccess, "remove, insert and resize into a different slice", "remove other slice ok\n")
	t.Run("test-slices-remove-out-of-range-d.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "remove into a different slice with index >= len")
	t.Run("test-slices-remove-out-of-range-e.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "len of chained remove past the end")
	t.Run("test-slices-remove-out-of-range-f.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "remove of an empty str slice into a different slice")
	t.Run("test-str-substr-out-of-range-a.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "string.SubstringByByteRange with a negative start")
	t.Run("test-str-substr-out-of-range-b.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "string.SubstringByByteRange with a negative end")
	t.Run("test-str-substr-out-of-range-c.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "string.SubstringByByteRange with start > end")
	t.Run("test-str-substr-out-of-range-d.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "string.SubstringByByteRange with end > len")
	t.Run("test-short-declarations.cx", valuetypes.CXSuccess, "short declarations")
	t.Run("test-parse.cx", valuetypes.CXSuccess, "parse")
	t.Run("test-collection-functions.cx", valuetypes.CXSuccess, "collection functions")
	t.Run("test-scopes.cx", valuetypes.CXSuccess, "ProgramError in scopes.")
	t.RunWant("test-scope-block-starts-with-block.cx", valuetypes.CXSuccess, "a block whose first statement is a block", "2\n")
	t.RunWant("test-scope-shadow-after-block.cx", valuetypes.CXSuccess, "shadowing after a nested block", "2\n1\n")
	t.RunWant("test-scope-one-statement-block.cx", valuetypes.CXSuccess, "a one statement block has its own scope", "0\n")
	t.RunWant("test-scope-method-call-first.cx", valuetypes.CXSuccess, "a method call as the first statement of a block", "7\n1\n2\n")
	t.RunWant("test-scope-nested-shadow.cx", valuetypes.CXSuccess, "shadowing in nested if, for and else blocks", "3\n4\n2\n5\n1\n")
	t.RunWant("test-scope-sibling-blocks.cx", valuetypes.CXSuccess, "sibling blocks declare the same names (skycoin-cx#16)", "1\n2\n")
	t.RunWant("test-loop-break-nested.cx", valuetypes.CXSuccess, "break leaves only the innermost loop (skycoin-cx#39)",
		"j: 0\nout of j loop\nj: 0\nout of j loop\nout of i loop\nj: 0\nout of j loop\nj: 0\nout of j loop\nout of i loop\nout of h loop\n")
	t.RunWant("test-loop-continue-nested.cx", valuetypes.CXSuccess, "continue goes on with the innermost loop (skycoin-cx#39)",
		"j: 0\nj: 1\nj: 2\nout of j loop\nj: 0\nj: 1\nj: 2\nout of j loop\nout of h loop\n")
	t.RunWant("test-loop-three-deep.cx", valuetypes.CXSuccess, "break and continue three loops deep (skycoin-cx#39)",
		"break 0 0 0\nbreak 0 1 0\nbreak 1 0 0\nbreak 1 1 0\n"+
			"continue 0 0 0\ncontinue 0 0 2\ncontinue 0 1 0\ncontinue 0 1 2\nafter c 0 1\n"+
			"continue 1 0 0\ncontinue 1 0 2\ncontinue 1 1 0\ncontinue 1 1 2\nafter c 1 1\nafter b 1\n")
	t.RunWant("test-loop-if-else.cx", valuetypes.CXSuccess, "break and continue inside if and else (skycoin-cx#39)", "odd 1\nodd 3\n3\n")
	t.RunWant("test-loop-goto.cx", valuetypes.CXSuccess, "goto out of and back into a labelled loop with break and continue", "1 0 0\n2 2 0\n3 0 0\n4\n")
	t.RunWant("test-loop-for-true.cx", valuetypes.CXSuccess, "for true runs its body (skycoin-cx#20)", "55\n4\n")
	t.RunWant("test-loop-for-bool.cx", valuetypes.CXSuccess, "for on a bool variable or expression, an empty condition and an assignment post statement (skycoin-cx#21)", "3\n20\n6\n6\n6\n")
	t.RunWant("test-shortcircuit-assign.cx", valuetypes.CXSuccess, "&& and || in assignments, nested, with ! and in a global initializer short-circuit (skycoin-cx#26)",
		"T0 T1 = true\nF3 T4 = true\nF5 = false\nT7 F8 = false\nF9 F10 T11 = true\nT13 T14 F15 = false\nT17 F18 T19 = true\nF20 = false\nF23 T25 T26 = true\nF27 = true\nT29 F30 T31 = false\nT32 = false\n= true\n= false\nF36 = true\n= true\n= false\nearly true\n26\n")
	t.RunWant("test-shortcircuit-control.cx", valuetypes.CXSuccess, "&& and || in if and for conditions skip a division by zero and an out of range index (skycoin-cx#26)",
		"T1 then\nF3 else\nF5 T6 F7 then\nT0 T1 T2 n 3\nT10 T11 \nno divide\na[0] 5\n10\n")
	t.RunWant("test-shortcircuit-call.cx", valuetypes.CXSuccess, "&& and || in arguments and return values short-circuit, bool.and and bool.or evaluate both (skycoin-cx#26)",
		"T1 F3 show true false\nF5 T100 show true true\nT200 show false true\nT301 pair true true\nT300 pair true false\nT6 F7 T8 F9 T10 F11 show true false\nF0 T0 show false true\nshow true false\nF2 T2 show false true\nT12 T13 = true\nF14 T15 = false\n")
	t.RunWant("test-shortcircuit-slice.cx", valuetypes.CXSuccess, "&& and || guard slice indexes, including a bare indexed operand (skycoin-cx#26)",
		"s[0] 5\nin range check\nfalse\ntrue\n")
	t.RunWant("test-loop-break-outside.cx", valuetypes.CXCompilationError, "break outside a loop (skycoin-cx#60)",
		`test-loop-break-outside.cx:5 break is not in a loop`)
	t.RunWant("test-loop-continue-outside.cx", valuetypes.CXCompilationError, "continue outside a loop (skycoin-cx#60)",
		`test-loop-continue-outside.cx:6 continue is not in a loop`)
	t.RunWant("test-operator-result-type.cx", valuetypes.CXCompilationError, "an arithmetic result of another type is not cast implicitly (skycoin-cx#27)",
		"test-operator-result-type.cx:4 cannot assign value of type 'i32' to identifier 'f' of type 'f32'",
		"test-operator-result-type.cx:6 cannot assign value of type 'i32' to identifier 'h' of type 'f32'",
		"test-operator-result-type.cx:8 cannot assign value of type 'i32' to identifier 'y' of type 'f32'",
		"test-operator-result-type.cx:9 cannot assign value of type 'i32' to identifier 'l' of type 'i64'",
		"test-operator-result-type.cx:11 cannot assign value of type 'i32' to identifier 'a' of type 'f32'",
		"test-operator-result-type.cx:13 cannot assign value of type 'f32' to identifier 'n' of type 'i32'")
	t.RunWant("test-operator-result-type-global.cx", valuetypes.CXCompilationError, "a global initialized with an arithmetic result of another type (skycoin-cx#27)",
		"test-operator-result-type-global.cx:3 cannot assign value of type 'i32' to identifier 'g' of type 'f32'")
	t.RunWant("test-operator-result-type-valid.cx", valuetypes.CXSuccess, "negative literals and arithmetic results of the receiver's type (skycoin-cx#27)",
		"-1 -1 -1 7 2 -1\n")
	t.Run("issue-520.cx", valuetypes.CXCompilationError, "a negative i32 literal initializes an f32 (skycoin-cx#27)")
	t.RunWant("test-slice-builtin-element-type.cx", valuetypes.CXCompilationError, "element and result types of append, insert, resize, remove and copy (skycoin-cx#55)",
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
	t.RunWant("test-slice-field-multidimensional.cx", valuetypes.CXSuccess, "append, insert, remove, resize and indexing on multi-dimensional slice struct fields (skycoin-cx#55)",
		"3\n5\n6\n1.5\n8\n10\n12\n13\n")
	t.RunWant("test-global-redeclaration.cx", valuetypes.CXCompilationError, "a global declared twice in one package (skycoin-cx#57)",
		"test-global-redeclaration.cx:4 'i' redeclared")
	t.RunWant("test-index-comma.cx", valuetypes.CXCompilationError, "a comma inside an index is a syntax error (skycoin-cx#59)",
		"test-index-comma.cx:7 syntax error: unexpected comma, expected ]")
	t.Run("issue-22.cx", valuetypes.CXCompilationError, "commas inside slice indexers of a struct field (skycoin-cx#59)")
	t.RunWant("test-goto-label-missing.cx", valuetypes.CXCompilationError, "goto to a label that does not exist",
		"test-goto-label-missing.cx:4 label 'nowhere' is not defined")
	t.RunWant("test-field-access-primitive.cx", valuetypes.CXCompilationError, "a field of a variable that is not a struct",
		"test-field-access-primitive.cx:5 illegal method call or field access on identifier 'x' of primitive type 'i32'",
		"test-field-access-primitive.cx:7 illegal method call or field access on identifier 'f' of primitive type 'f32'")
	t.RunWant("test-method-call-value.cx", valuetypes.CXSuccess, "method calls in operands, arguments, return values, if and for conditions and chains (skycoin-cx#24)",
		"7\n8\n8\n77\n8\n13\n10\n18\np(3,4)\n7\n7\n5\n6\nd(3,4)\nif sum\nif positive\nboth positive\nloop 1\nloop 2\nloop 3\n14\n15\n4\n9\n10\n7\n7\n8\n14\n103\n8\n7 5\n7 2 13\n-7\nfalse\n7 x(3,4)\n")
	t.RunWant("test-method-pointer-receiver.cx", valuetypes.CXSuccess, "a pointer receiver method on a value, a pointer and a struct field",
		"11 12\n111 112\n0 0 6 1\n")
	t.RunWant("test-method-result-missing.cx", valuetypes.CXCompilationError, "a method with no output used as a value",
		"test-method-result-missing.cx:15 method 'Point.show' returns no value")
	t.RunWant("test-call-result-call.cx", valuetypes.CXCompilationError, "calling the result of a call (skycoin-cx#22)",
		"test-call-result-call.cx:19 the result of a call is not a function",
		"test-call-result-call.cx:20 the result of a call is not a function",
		"test-call-result-call.cx:21 the result of a call is not a function",
		"test-call-result-call.cx:23 the result of a call is not a function")
	t.RunWant("test-package-function-qualified.cx", valuetypes.CXCompilationError, "pkg.f does not find a function of an import of pkg (skycoin/cx#9)",
		"test-package-function-qualified.cx:25 identifier 'main.test1' does not exist")
	t.RunWant("test-package-function-value.cx", valuetypes.CXCompilationError, "pkg.F without parentheses is not a call (skycoin-cx#48)",
		"test-package-function-value.cx:8 cannot assign value of type 'func()(i32)' to identifier 'x' of type 'i32'",
		"test-package-function-value.cx:9 function 'i32.print' expected input argument of type 'i32'; 'func()(i32)' was provided")
	t.RunWant("issue-136.cx", valuetypes.CXCompilationError, "pkg.F without parentheses is not a call (skycoin-cx#48)",
		"issue-136.cx:7 invalid expression")
	t.RunWant("test-function-shadows-builtin.cx", valuetypes.CXSuccess, "a function shadows a built-in of the same name in its package (skycoin-cx#294)",
		"-1\n99\n7\n20\n7\n20\n")
	t.RunWant("issue-23.cx", valuetypes.CXSuccess, "logical not on a call result (skycoin-cx#58)", "false\n")
	t.Run("issue-121.cx", valuetypes.CXSuccess, "unary minus on a call result (skycoin-cx#28)")
	t.RunWant("issue-25.cx", valuetypes.CXSuccess, "unary minus on struct fields (skycoin-cx#56)", "-12\n15\n")
	t.RunWant("test-unary-value.cx", valuetypes.CXSuccess, "! and - on calls, method calls and parenthesized expressions (skycoin-cx#58)",
		"false\ntrue\ntrue\nside 1\ntrue\ntrue\n-4\n-8\n-7\n3\n-1.5\n-3\nfalse\n1\n-1\n")
	t.RunWant("test-unary-field.cx", valuetypes.CXSuccess, "! and - on nested, pointer and indexed struct fields (skycoin-cx#56)",
		"-5\n-1.5\nfalse\n-9\n-5\n-5\n5\n")
	t.RunWant("issue-450.cx", valuetypes.CXSuccess, "++ and -- on every numeric type (skycoin-cx#45)", "2\n0\n2\n2\n0\n2\n")
	t.RunWant("issue-450b.cx", valuetypes.CXSuccess, "++ and -- on every numeric type (skycoin-cx#45)", "2\n0\n2\n2\n2\n")
	t.RunWant("test-incdec-operand.cx", valuetypes.CXSuccess, "++, -- and compound assignment on unsigned types, fields, indexes and pointers (skycoin-cx#45)",
		"0\n65535\n42\n0\n5\n9\n15\n4\n6\n3.5\n2\n14\n8\n3\n3\n2\n1\n")
	t.RunWant("test-operator-operand-line.cx", valuetypes.CXCompilationError, "an operand type mismatch is reported at its line (skycoin-cx#45)",
		"test-operator-operand-line.cx:8 operands are not of the same type: i32 != i8")
	t.RunWant("issue-68.cx", valuetypes.CXCompilationError, "i++ is a statement, not an argument (skycoin-cx#18)", "issue-68.cx:7 syntax error")
	t.RunWant("test-incdec-value.cx", valuetypes.CXCompilationError, "i++ is a statement, not an operand (skycoin-cx#18)", "test-incdec-value.cx:5 syntax error")
	t.RunWant("test-incdec-prefix.cx", valuetypes.CXCompilationError, "there is no prefix ++", "test-incdec-prefix.cx:5 syntax error")
	t.RunWant("issue-62.cx", valuetypes.CXCompilationError, "a comma list is not a statement (skycoin-cx#23)", "issue-62.cx:15 syntax error: unexpected newline")
	t.Run("issue-34.cx", valuetypes.CXSuccess, "multiple results of a call in := (skycoin-cx#44)")
	t.RunWant("issue-512.cx", valuetypes.CXSuccess, "multiple results of a call in := (skycoin-cx#44)", "2\n1\n")
	t.RunWant("issue-512b.cx", valuetypes.CXSuccess, "several values in := (skycoin-cx#44)", "20\n10\n")
	t.RunWant("cassignfunc.cx", valuetypes.CXSuccess, "multiple results of a call in := (skycoin-cx#44)", "20\n10\n")
	t.RunWant("cassigntwovarialbe.cx", valuetypes.CXSuccess, "several values in := (skycoin-cx#44)", "20\n10\n")
	t.RunWant("multi-return-from-function.cx", valuetypes.CXSuccess, "multiple results of a call in =", "20\n10\n")
	t.Run("multi-return-from-function-new-varaible.cx", valuetypes.CXSuccess, "multiple results of a call in := (skycoin-cx#44)")
	t.RunWant("test-multi-assign.cx", valuetypes.CXSuccess, "multiple assignment from values, calls and method calls, with swaps and short-circuit values (skycoin-cx#44)",
		"2 1\n7 seven\n7 seven\nside 1\nside 2\ntrue true\n9 2.5\n0 9\n2 1\n1 two 3.5\n5000000000\n0.25\ntrue\n3\n-1 2 -2\n2 5000000000\ni 0 j 5\ni 1 j 4\ni 2 j 3\n")
	t.RunWant("test-multi-assign-count.cx", valuetypes.CXCompilationError, "a multiple assignment with the wrong number of values",
		"test-multi-assign-count.cx:16 assignment mismatch: 2 variables but 3 values",
		"test-multi-assign-count.cx:17 assignment mismatch: 2 variables but 1 value",
		"test-multi-assign-count.cx:18 assignment mismatch: 3 variables but 2 values",
		"test-multi-assign-count.cx:19 assignment mismatch: 2 variables but 1 value")
	t.RunWant("test-multi-assign-operator.cx", valuetypes.CXCompilationError, "a multiple assignment takes only = and :=", "test-multi-assign-operator.cx:6 syntax error")
	t.RunWant("test-struct-literal-negative.cx", valuetypes.CXSuccess, "negative and computed values in struct literal fields", "-1\n-2.5\n-4\n4\n1.5\n-3\n")
	t.RunWant("test-short-declaration-copy.cx", valuetypes.CXSuccess, "a short declaration copies an i64, an f64, a field or a struct",
		"5000000000\n7\n0.125\n8\n6000000000\n9\n1\n2\n10\n6000000000\n")
	t.RunWant("issue-35.cx", valuetypes.CXSuccess, "slice literal as an argument (skycoin-cx#51)", "3\n3\n")
	t.RunWant("issue-539.cx", valuetypes.CXSuccess, "array literal of structs (skycoin-cx#40)", "1\n")
	t.RunWant("issue-526.cx", valuetypes.CXSuccess, "[N]*T parameters and variables (skycoin-cx#42)", "1\n")
	t.RunWant("issue-521.cx", valuetypes.CXSuccess, "[N]pkg.T variables (skycoin-cx#43)", "5\n")
	t.RunWant("issue-521b.cx", valuetypes.CXCompilationError, "[N]pkg.T of a package that is not imported (skycoin-cx#43)",
		"issue-521b.cx:12 package 'pack' not imported")
	t.RunWant("issue-37.cx", valuetypes.CXSuccess, "nested slice literal without inner types (skycoin-cx#50)", "3\n")
	t.RunWant("issue-37b.cx", valuetypes.CXSuccess, "nested slice literal with inner types (skycoin-cx#50)", "3\n")
	t.RunWant("test-array-pointer-element.cx", valuetypes.CXSuccess, "arrays of pointers as variables, fields, globals, arguments and results (skycoin-cx#42)",
		"4\n4\n6\n6\n9\n9\n6\n")
	t.RunWant("test-array-package-type.cx", valuetypes.CXSuccess, "arrays of a struct of another package as variables, fields, globals, arguments and results (skycoin-cx#43)",
		"3\n7\n3\n5\n6\n8\n")
	t.RunWant("test-array-2d-result.cx", valuetypes.CXSuccess, "2-D and 3-D array results and arguments", "7\n9\n9\n5\n")
	t.RunWant("test-array-2d-errors.cx", valuetypes.CXCompilationError, "2-D array type errors name the dimensions outermost first",
		"test-array-2d-errors.cx:12 function 'mk' expected receiving variable of type '[3][2]i32'; '[2][3]i32' was provided",
		"function 'take' expected input argument of type '[2][3]i32'; '[3][2]i32' was provided")
	t.RunWant("test-literal-elided.cx", valuetypes.CXSuccess, "array and slice literals with elided element types (skycoin-cx#50)",
		"5\n3\n4\n3\n3\n8\n9\n0\n6\n5\n6\n11\n10\n6\n6\n7\n2\n4\n8\n4\n")
	t.RunWant("test-literal-elided-errors.cx", valuetypes.CXCompilationError, "elided literal elements of a type that cannot take one",
		"test-literal-elided-errors.cx:9 invalid composite literal type 'i32'",
		"test-literal-elided-errors.cx:10 struct literal of type 'Point' must name its fields",
		"test-literal-elided-errors.cx:11 invalid composite literal type '[]i32'",
		"test-literal-elided-errors.cx:12 invalid composite literal type 'i32'",
		"test-literal-elided-errors.cx:13 struct literal of type 'Point' must name its fields")
	t.RunWant("test-pointer-copy.cx", valuetypes.CXSuccess, "copying a pointer copies its value, and p[i].f indexes a *[N]T and a [N]*T",
		"11\n11\n21\n12\n1\n3\n5\n6\n11\n9\n12\n32\n")
	t.RunWant("test-runtime-error-empty-callee.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "a runtime error in the argument of an empty function prints the call site",
		"test-runtime-error-empty-callee.cx:8, CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE", "===Callstack===\n>>> main()")
	t.RunWant("test-pointer-field-chain.cx", valuetypes.CXSuccess, "b.p.x reads and writes through the pointer field p, and b.p = &x stores the pointer",
		"7\n9\n9\n9\n7\n9\n3\n9\n70\n71\n72\n72\n72\n5\n9\n1\n")
	t.RunWant("test-pointer-array-index.cx", valuetypes.CXSuccess, "p[i] on a *[N]T indexes *p as in Go, for atomic, struct and pointer elements",
		"5\n19\n13\n42\n5\n8\n7\n16\n15\n31\n41\n16\n31\n70\n2.5\nhi\n1\n")
	t.RunWant("test-struct-field-pointer-atomic.cx", valuetypes.CXSuccess, "a struct field of type *i32 has the type *i32",
		"4\n4\n6\n9\n")
	t.RunWant("test-runtime-error-input-copy.cx", valuetypes.CXRuntimeInvalidCast, "a runtime error in an argument stops the caller, not the callee",
		"test-runtime-error-input-copy.cx:12, CX_RUNTIME_INVALID_CAST", "===Callstack===\n>>> main()")
	t.RunWant("test-escape-return.cx", valuetypes.CXSuccess, "a returned pointer to a local, an input, a struct, an array or a &T{} is not overwritten by a later call",
		"5\n5\n6\n7\n8\n41\n2\n3\n4\n60\n10\n")
	t.RunWant("test-escape-store.cx", valuetypes.CXSuccess, "a pointer to a local stored in a global, a struct field, a slice or an array outlives the call",
		"11\n12\n13\n21\n22\n31\n32\n33\n32\n51\n")
	t.RunWant("test-escape-loop.cx", valuetypes.CXSuccess, "each loop pass gets its own &T{} and escaping local",
		"0\n100\n200\n10\n101\n201\n20\n102\n202\n10\n4321\n")
	t.RunWant("-heap-initial 0 test-escape-gc.cx", valuetypes.CXSuccess, "the GC keeps reachable escaped locals and local slices and moves the pointers to them",
		"0\n2\n4\n6\n7\n321\n41\n42\n")
	t.RunWant("-heap-initial 0 test-gc-array-pointer.cx", valuetypes.CXSuccess, "the GC follows a pointer to an array, in a local, a global and a field, to the pointers and strs in its elements", "gc ok\n")
	t.RunWant("-heap-initial 0 test-gc-array-field.cx", valuetypes.CXSuccess, "the GC follows the pointers, strs and slices in the array fields of a struct on the heap", "gc ok\n")
	t.RunWant("-heap-initial 0 test-gc-slice-struct.cx", valuetypes.CXSuccess, "the GC follows the pointers, slices and strs in the struct elements of a slice, and a pointer into a slice moves with it", "gc ok\n")
	t.RunWant("-heap-initial 0 test-gc-nested.cx", valuetypes.CXSuccess, "the GC follows arrays of structs holding slices of structs holding pointers and str arrays, and a ring of pointers", "gc ok\n")
	t.RunWant("--executor=vm test-escape-frame.cx", valuetypes.CXSuccess, "a local whose address does not outlive the call stays in the frame",
		"10\n12\n13\n4\n6\n14\n")
	t.RunWant("test-address-literal.cx", valuetypes.CXSuccess, "&T{} as a var initializer, :=, an array element, a field, a struct literal field, append, an argument and a return gives a new T each time",
		"2\n7\n210\n5\n6\n321\n2\n15\n10\n21\n31\n30\n")
	t.RunWant("--executor=old test-address-literal-frame.cx", valuetypes.CXSuccess, "&T{} that stays in the frame, on the old loop", "2\n9\n6\n15\n9\n10\n")
	t.RunWant("--executor=vm test-address-literal-frame.cx", valuetypes.CXSuccess, "&T{} that stays in the frame, on vm", "2\n9\n6\n15\n9\n10\n")
	t.RunWant("test-call-result-field.cx", valuetypes.CXSuccess, "f().x and f().p.y on struct, *T and method call results",
		"1\n4\n3\n8\n6\n6\n16\n109\n18\n110\n")
	t.RunWant("--executor=old test-call-result-field-frame.cx", valuetypes.CXSuccess, "f().x on a struct result, on the old loop", "3\n12\n5\n36\n15\n42\n")
	t.RunWant("--executor=vm test-call-result-field-frame.cx", valuetypes.CXSuccess, "f().x on a struct result, on vm", "3\n12\n5\n36\n15\n42\n")
	t.RunWant("--executor=old test-pointer-to-pointer.cx", valuetypes.CXSuccess, "&p of a *i32 is a **i32, on the old loop", "5\n8\n9\n")
	t.RunWant("--executor=vm test-pointer-to-pointer.cx", valuetypes.CXSuccess, "&p of a *i32 is a **i32, on vm", "5\n8\n9\n")
	t.RunWant("test-array-struct-pointer-field.cx", valuetypes.CXSuccess, "a[i].p on an array of structs with a pointer field, from a variable, a call and &x",
		"2\n10\n4\n11\n6\n12\n14\n2\n21\n")
	t.RunWant("--executor=old test-array-struct-pointer-field-frame.cx", valuetypes.CXSuccess, "a[i].p = &x keeps each element apart, on the old loop", "5\n6\n5\n12\n")
	t.RunWant("--executor=vm test-array-struct-pointer-field-frame.cx", valuetypes.CXSuccess, "a[i].p = &x keeps each element apart, on vm", "5\n6\n5\n12\n")
	t.RunMissing("test-debug-go-stack.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "a runtime error prints no Go stack by default", "goroutine")
	t.RunWant("--debug-go-stack test-debug-go-stack.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "--debug-go-stack prints the Go stack of a runtime error", "goroutine")
	t.RunWant("test-pointer-array-2d.cx", valuetypes.CXSuccess, "p[i][j] indexes a *[N][M]T and a *[N][M]*T",
		"12\n101\n110\n77\n11\n212\n201\n211\n55\n211\n211\n9\n4\n")
	t.RunWant("test-runtime-error-neg-argument.cx", valuetypes.CXRuntimeInvalidCast, "the temporary of g(-1) has the line of the call in the call stack",
		"test-runtime-error-neg-argument.cx:12 : i32.neg() : -1")
	t.RunWant("test-runtime-error-return-line.cx", valuetypes.CXRuntimeSliceIndexOutOfRange, "a runtime error in return s[0][1] prints the line of the return",
		"test-runtime-error-return-line.cx:4, CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE")
	t.RunWant("test-runtime-error-return-line-index.cx", valuetypes.CXRuntimeInvalidCast, "a runtime error in return a[i] prints the line of the return",
		"test-runtime-error-return-line-index.cx:5, CX_RUNTIME_INVALID_CAST")
	t.Run("test-struct-literal-value.cx", valuetypes.CXSuccess, "a struct literal is a value: var initializer, T{} zeroes, argument, result and a swap of fields")
	t.Run("test-struct-literal-nested.cx", valuetypes.CXSuccess, "struct, slice and array literals nested in each other, with calls in the fields")
	t.Run("test-struct-literal-package.cx", valuetypes.CXSuccess, "struct literals in global initializers and of another package")
	t.Run("test-array-literal-value.cx", valuetypes.CXSuccess, "array literals of structs, as arguments, results and short declarations")
	t.RunWant("test-slice-literal-argument.cx", valuetypes.CXSuccess, "slice literals as call arguments (skycoin-cx#51)", "2 1\n")
	t.Run("test-string-slice-pointer.cx", valuetypes.CXSuccess, "p = &s on a *[]str takes the address of the slice")
	t.RunWant("test-composite-literal-errors.cx", valuetypes.CXCompilationError, "an array literal longer than its array and a field named twice",
		"test-composite-literal-errors.cx:9 array literal of length 2 has 3 elements",
		"test-composite-literal-errors.cx:11 array literal of length 1 has 2 elements",
		"test-composite-literal-errors.cx:13 duplicate field name 'a' in struct literal")
	t.Run("issue-540a.cx", valuetypes.CXSuccess, "continue inside three nested loops (skycoin-cx#39)")
	t.Run("issue-540b.cx", valuetypes.CXSuccess, "break inside three nested loops (skycoin-cx#39)")
	t.Run("issue-537.cx", valuetypes.CXSuccess, "Crash when a block starts with a for loop and declares a variable")
	t.Run("issue-31.cx", valuetypes.CXSuccess, "len of a slice after append, read into a short declaration")
	t.RunWant("issue-157.cx", valuetypes.CXSuccess, "a call result as a slice index in a bubble sort (skycoin-cx#29)", "[0, 1, 2, 3, 4, 5, 6, 8, 9, 10]\n")
	t.RunWantInput("issue-110.cx", "abc\ndef\n", valuetypes.CXSuccess, "a second read() on a piped stdin (skycoin/cx#110)", "Hello, abc!\ndef\n")
	t.RunWantInput("test-read-eof.cx", "one\ntwo", valuetypes.CXSuccess, "read() returns an unterminated last line, then \"\" at the end of input", "[one]\n[two]\n[]\n[]\n")
	t.RunWant("issue-432.cx", valuetypes.CXCompilationError, "sys/gl in a cx built without cxfx (skycoin/cx#432)", "issue-432.cx:6 package \"sys/gl\" needs a GUI build of cx, build it with make build-gui")
	t.RunWantEx("--enable=file test-cxfx-handles.cx", valuetypes.CXSuccess, "cxfx handles and compile time callbacks (skycoin-cx#432)", testrunner.TestGui,
		"window 64 48\n", "start callback 0 1\n", "callback sum 42\n", "pixel 1.0000000 0.8509804 0.2862745 1.0000000\n", "gif 3 2 4 2\n", "gif frame 20 2 error 0\n",
		"metrics 152 45\n", "shader log length 0\n", "shader compiled 1\n", "program linked 1 attrib 3 uniform 0\n", "closing false error 0\n", "closing true\n")
	t.RunWantEx("test-cxfx-events.cx", valuetypes.CXSuccess, "cxfx window callbacks get the window handle (skycoin-cx#432)", testrunner.TestGui, "pos 0 30 40\n", "size 0 matches\n", "framebuffer 0 matches\n")
	t.RunWantEx("test-cxfx-callback-name.cx", valuetypes.CXCompilationError, "a cxfx callback takes a function name (skycoin-cx#432)", testrunner.TestGui, "test-cxfx-callback-name.cx:10 function 'glfw.SetKeyboardCallback' expected a function name as input argument 2; 'str' was provided")
	t.RunWant("-h", valuetypes.CXSuccess, "cx -h names the real flags (skycoin/cx#567)", "alias for -lexer\n", "alias for -heap-initial\n", "alias for -heap-max\n", "-lexer-output string")
	t.RunWant("test-append-struct-field.cx", valuetypes.CXSuccess, "append and len on a slice field of a struct and of a struct pointer",
		"1 7 9000000000 cee\n2 7 9000000000 cee\n2 2.5 3.5\none two\n4 5\n3 3 9000000000 cee\n2 9000000000 cee\n")
	t.RunWant("--executor=old test-operator-precedence.cx", valuetypes.CXSuccess, "binary operators bind as in Go, on the old loop",
		"true\ntrue\n17\n12\n3\n8\n4\n6\n8\n6\ntrue\ntrue\n1\n0\n")
	t.RunWant("--executor=vm test-operator-precedence.cx", valuetypes.CXSuccess, "binary operators bind as in Go, on vm",
		"true\ntrue\n17\n12\n3\n8\n4\n6\n8\n6\ntrue\ntrue\n1\n0\n")
	t.RunWant("test-error-cascade.cx", valuetypes.CXCompilationError, "a compile error in one function leaves the locals of another declared",
		"test-error-cascade.cx:6 operands are not of the same type: i32 != ui64")
	t.RunMissing("test-error-cascade.cx", valuetypes.CXCompilationError, "a compile error in one function leaves the locals of another declared", "does not exist")
	t.RunWant("test-undeclared-target.cx", valuetypes.CXCompilationError, "assignments to undeclared variables are compile errors, not a panic",
		"test-undeclared-target.cx:12 identifier 'x' does not exist")
	t.RunMissing("test-undeclared-target.cx", valuetypes.CXCompilationError, "assignments to undeclared variables are compile errors, not a panic", "panic")
	t.RunWant("test-global-pointer-slice-field.cx", valuetypes.CXSuccess, "slice fields through a global pointer to a struct (skycoin/skycoin-cx#13)",
		"b\na\n7\n4\n2\n9\nb\ng\n2\n8\n")
	t.RunWant("test-array-of-slices.cx", valuetypes.CXSuccess, "arrays of slices as locals, globals and struct fields",
		"3\n2\n6\n4\n3\n4\n3\n40\n8\n3\nthree\n11\n21\nnm\n32\n2\n2\n9\n0\n4\n40\n")
	t.RunWant("-heap-initial 0 test-gc-array-of-slices.cx", valuetypes.CXSuccess, "the GC follows the slices in arrays of slices", "gc ok\n")
	t.RunWant("--executor=old test-array-struct-field-sum.cx", valuetypes.CXSuccess, "a sum of fields of local array elements, on the old loop", "11\n7\n17\n")
	t.RunWant("--executor=vm test-array-struct-field-sum.cx", valuetypes.CXSuccess, "a sum of fields of local array elements, on vm", "11\n7\n17\n")
	t.Run("-heap-initial 0 test-gc.cx", valuetypes.CXSuccess, "Stress-testing the garbage collector, with slice fields through a global pointer (skycoin/skycoin-cx#13)")
	t.RunWant("-heap-initial 0 test-gc-slice-resize.cx", valuetypes.CXSuccess, "append keeps the elements of a slice the GC moves while it grows", "local 200 2\n")
	t.RunWant("--executor=old test-short-decl-array.cx", valuetypes.CXSuccess, "x := v of an array, an array element, a field or a call result, on the old loop", "7 0 9\n9 6 5\n8 11 11\n4 5\n2 3\n")
	t.RunWant("--executor=vm test-short-decl-array.cx", valuetypes.CXSuccess, "x := v of an array, an array element, a field or a call result, on vm", "7 0 9\n9 6 5\n8 11 11\n4 5\n2 3\n")
	t.RunWant("-heap-initial 0 test-short-decl-slice.cx", valuetypes.CXSuccess, "x := v of a slice, a slice element, a field or a call result", "2 2\n2 1\n2\na b 2\n2\n1 b\n2 8 a\nb\n")
	t.RunWant("test-slice-of-arrays.cx", valuetypes.CXSuccess, "slices of arrays and of arrays of structs", "2 2 1 2 9\n3 9\n40 30 39 b5 seven b39\n2 77 b39\nz1 1 600\n")
	t.RunWant("-heap-initial 0 test-slice-of-arrays.cx", valuetypes.CXSuccess, "the GC follows slices of arrays", "2 2 1 2 9\n3 9\n40 30 39 b5 seven b39\n2 77 b39\nz1 1 600\n")
	t.RunWant("-ast test-ast-order.cx", valuetypes.CXSuccess, "-ast prints functions in declaration order",
		"\t\t0.- Function: A.m (1:a A) ()\n\t\t1.- Function: zeta () ()\n\t\t2.- Function: alpha () ()\n\t\t3.- Function: mid () ()\n\t\t4.- Function: omega () ()\n\t\t5.- Function: beta () ()\n\t\t6.- Function: main () ()\n")
	t.RunWant("test-error-outputless-method.cx", valuetypes.CXCompilationError, "an outputless call in an assignment leaves the later errors reported",
		"test-error-outputless-method.cx:14 trying to use an outputless operator in an assignment", "test-error-outputless-method.cx:12 type 'A' has no method 'missing'")
	t.RunWant("test-error-outputless-declare.cx", valuetypes.CXCompilationError, "y := noout() declares y, so its uses report nothing more",
		"test-error-outputless-declare.cx:14 trying to use an outputless operator in an assignment",
		"test-error-outputless-declare.cx:25 cannot assign value of type 'str' to identifier 'late' of type 'i32'")
	t.RunMissing("test-error-outputless-declare.cx", valuetypes.CXCompilationError, "y := noout() declares y, so its uses report nothing more",
		"does not exist", "invalid type", "has no fields", "same type", "invalid argument types", ":15 ", ":17 ", ":18 ", ":19 ", ":21 ", ":23 ", ":24 ")
	t.RunWant("test-error-method-missing-continue.cx", valuetypes.CXCompilationError, "a call of a missing method leaves the later errors reported",
		"test-error-method-missing-continue.cx:12 type 'A' has no method 'missing'",
		"test-error-method-missing-continue.cx:14 type 'A' has no method 'gone'",
		"test-error-method-missing-continue.cx:16 illegal method call or field access on identifier 'n' of primitive type 'i32'",
		"test-error-method-missing-continue.cx:17 method 'A.show' returns no value",
		"test-error-method-missing-continue.cx:19 identifier 'b' does not exist",
		"test-error-method-missing-continue.cx:20 cannot assign value of type 'str' to identifier 'late' of type 'i32'")
	t.RunMissing("test-error-method-missing-continue.cx", valuetypes.CXCompilationError, "a call of a missing method leaves the later errors reported",
		"*mrs", "not found", "has no fields", "invalid type", ":13 ", ":18 ")
	t.RunWant("test-error-assign-once.cx", valuetypes.CXCompilationError, "an assignment of the wrong type is one error",
		"test-error-assign-once.cx:5 cannot assign value of type 'str' to identifier 'z' of type 'i32'",
		"test-error-assign-once.cx:7 cannot assign value of type 'i32' to identifier 'b' of type 'bool'")
	t.RunMissing("test-error-assign-once.cx", valuetypes.CXCompilationError, "an assignment of the wrong type is one error",
		"trying to assign argument")
	t.RunWant("test-error-order.cx", valuetypes.CXCompilationError, "compile errors print in line order",
		"test-error-order.cx:12 type 'A' has no method 'missing'\nerror: test-error-order.cx:14 trying to use an outputless operator in an assignment\nerror: test-error-order.cx:15 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunWant("test-append-element-type-full.cx", valuetypes.CXCompilationError, "append checks the full element type, call results and literals included",
		"test-append-element-type-full.cx:27 function 'append' expected argument of type '[2]i32'; 'i32' was provided",
		"test-append-element-type-full.cx:28 function 'append' expected argument of type '[2]i32'; '[3]i32' was provided",
		"test-append-element-type-full.cx:29 function 'append' expected argument of type '[2]i32'; 'i32' was provided",
		"test-append-element-type-full.cx:30 function 'append' expected argument of type '[2]i32'; '[3]i32' was provided",
		"test-append-element-type-full.cx:33 function 'append' expected argument of type 'P'; 'Q' was provided",
		"test-append-element-type-full.cx:34 function 'append' expected argument of type 'P'; 'Q' was provided",
		"test-append-element-type-full.cx:37 function 'append' expected argument of type '[]i64'; '[]i32' was provided",
		"test-append-element-type-full.cx:38 function 'append' expected argument of type '[]i64'; '[]i32' was provided")
	t.RunMissing("test-append-element-type-full.cx", valuetypes.CXCompilationError, "append checks the full element type, call results and literals included",
		":40 ", ":41 ", ":42 ", ":13 ", ":17 ", ":21 ")
	t.RunWant("test-error-call-count-continue.cx", valuetypes.CXCompilationError, "a call with a wrong number of inputs or receivers leaves the later errors reported",
		"test-error-call-count-continue.cx:12 operator 'f' expects 1 input/s, but 0 input arguments were provided",
		"test-error-call-count-continue.cx:14 operator 'g' expects to return 2 output/s, but 1 receiving argument/s were provided",
		"test-error-call-count-continue.cx:16 cannot assign value of type 'str' to identifier 'late' of type 'i32'")
	t.RunWant("test-error-function-missing-continue.cx", valuetypes.CXCompilationError, "a call of a missing function leaves the later errors reported",
		"test-error-function-missing-continue.cx:8:2 function 'foo' not found in package 'main'\nerror: test-error-function-missing-continue.cx:9:7 function 'foo' not found in package 'main'\nerror: test-error-function-missing-continue.cx:12:4 function 'foo' not found in package 'main'\nerror: test-error-function-missing-continue.cx:13:12 function 'bar' not found in package 'main'\nerror: test-error-function-missing-continue.cx:14:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("test-error-function-missing-continue.cx", valuetypes.CXCompilationError, "a call of a missing function leaves the later errors reported",
		"expects", "invalid type", "does not exist", ":11 ")
	t.RunWant("test-error-native-missing-continue.cx", valuetypes.CXCompilationError, "a call of a missing or renamed native leaves the later errors reported",
		"test-error-native-missing-continue.cx:4:2 function 'i32.nope' does not exist\nerror: test-error-native-missing-continue.cx:5:7 function 'i32.nope2' does not exist\nerror: test-error-native-missing-continue.cx:6:2 str.print was renamed to string.PrintLine",
		"test-error-native-missing-continue.cx:8:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'")
	t.RunMissing("test-error-native-missing-continue.cx", valuetypes.CXCompilationError, "a call of a missing or renamed native leaves the later errors reported",
		":7 ", "invalid type", "expects")
	t.RunWant("test-error-package-member-continue.cx", valuetypes.CXCompilationError, "a missing package member or an unimported built-in package leaves the later errors reported",
		"test-error-package-member-continue.cx:5:2 identifier 'os.Nope' does not exist\nerror: test-error-package-member-continue.cx:6:7 identifier 'os.Nope2' does not exist\nerror: test-error-package-member-continue.cx:7:7 identifier 'os.NoVal' does not exist\nerror: test-error-package-member-continue.cx:9:2 built-in package time is used without importing it, add import \"sys/time\"\nerror: test-error-package-member-continue.cx:10:2 identifier 'os.ReadAllHmm' does not exist\nerror: test-error-package-member-continue.cx:11:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("test-error-package-member-continue.cx", valuetypes.CXCompilationError, "a missing package member or an unimported built-in package leaves the later errors reported",
		":8 ", "not found", "invalid type", "expects")
	t.RunWant("test-error-package-unimported-continue.cx", valuetypes.CXCompilationError, "each use of an unimported built-in package is reported",
		"test-error-package-unimported-continue.cx:4:2 built-in package os is used without importing it, add import \"sys/os\"\nerror: test-error-package-unimported-continue.cx:5:7 built-in package os is used without importing it",
		"test-error-package-unimported-continue.cx:6:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'")
	t.RunWant("test-error-loop-indirection-continue.cx", valuetypes.CXCompilationError, "a loop condition without a value and an invalid indirection leave the later errors reported",
		"test-error-loop-indirection-continue.cx:7:6 loop condition has no value\nerror: test-error-loop-indirection-continue.cx:10:7 invalid indirection\nerror: test-error-loop-indirection-continue.cx:12:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("test-error-loop-indirection-continue.cx", valuetypes.CXCompilationError, "a loop condition without a value and an invalid indirection leave the later errors reported",
		"outputless", "expects", ":11 ", ":9 ")
	t.RunWant("test-error-preliminary-continue.cx", valuetypes.CXCompilationError, "a missing type in a declaration leaves the errors in the bodies reported once",
		"test-error-preliminary-continue.cx:4:4 struct 'Missing2' not found in package 'main'\nerror: test-error-preliminary-continue.cx:7:10 struct 'Missing3' not found in package 'main'\nerror: test-error-preliminary-continue.cx:11:8 struct 'Missing' not found in package 'main'\nerror: test-error-preliminary-continue.cx:12:2 function 'foo' not found in package 'main'\nerror: test-error-preliminary-continue.cx:13:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunWant("test-error-syntax-a.cx test-error-syntax-b.cx", valuetypes.CXCompilationError, "a syntax error is a record, sorted with the errors of the other files",
		"error: test-error-syntax-a.cx:4:4 struct 'Missing' not found in package 'main'\nerror: test-error-syntax-b.cx:5:4 syntax error: unexpected name y, expected } or semicolon or newline\n")
	t.RunWant("test-error-syntax-body.cx", valuetypes.CXCompilationError, "a syntax error names the unexpected and the expected token",
		"error: test-error-syntax-body.cx:6:5 syntax error: unexpected comma, expected ]\n")
	t.RunWant("test-error-column.cx", valuetypes.CXCompilationError, "compile errors print file:line:column, in column order on one line",
		"error: test-error-column.cx:4:7 function 'foo' not found in package 'main'\nerror: test-error-column.cx:4:16 function 'bar' not found in package 'main'\nerror: test-error-column.cx:5:14 cannot assign value of type 'str' to identifier 'y' of type 'i32'\n")
	t.RunWant("test-dereference-non-pointer.cx", valuetypes.CXCompilationError, "dereferencing a non-pointer is one error that names the type (skycoin-cx#15)",
		"test-dereference-non-pointer.cx:9:8 cannot dereference non-pointer 'x' of type 'i32'\nerror: test-dereference-non-pointer.cx:11:15 cannot dereference non-pointer 'x' of type 'i32'\nerror: test-dereference-non-pointer.cx:13:8 cannot dereference non-pointer 'p' of type 'P'\nerror: test-dereference-non-pointer.cx:17:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("test-dereference-non-pointer.cx", valuetypes.CXCompilationError, "dereferencing a non-pointer is one error that names the type (skycoin-cx#15)",
		"'32'", "invalid type", ":10 ", ":15 ", ":16 ")
	t.RunWant("test-slice-literal-nested-type.cx", valuetypes.CXSuccess, "nested slice literal elements have their full type",
		"[[1 2] [3] [7]] 1\n[[{x: 1}] []]\n[[1] [2 3]]\n")
	t.RunWant("test-slice-literal-nested-type-errors.cx", valuetypes.CXCompilationError, "a nested slice literal element of the wrong type is reported",
		"error: test-slice-literal-nested-type-errors.cx:4:15 cannot use value of type '[]i64' as type '[]i32' in slice literal\nerror: test-slice-literal-nested-type-errors.cx:4:21 cannot use value of type 'i32' as type 'i64' in slice literal\nerror: test-slice-literal-nested-type-errors.cx:5:15 cannot use value of type 'i32' as type '[]i32' in slice literal\nerror: test-slice-literal-nested-type-errors.cx:6:36 cannot use value of type '[]i32' as type '[][]i32' in slice literal\nerror: test-slice-literal-nested-type-errors.cx:7:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("test-slice-literal-nested-type-errors.cx", valuetypes.CXCompilationError, "a nested slice literal element of the wrong type is reported",
		"function 'append'")
	t.RunWant("test-error-call-count-once.cx", valuetypes.CXCompilationError, "a call with the wrong number of inputs reports only that",
		"test-error-call-count-once.cx:8:2 operator 'f' expects 1 input/s, but 2 input arguments were provided\nerror: test-error-call-count-once.cx:9:2 operator 'f' expects to return 1 output/s, but 0 receiving argument/s were provided\nerror: test-error-call-count-once.cx:10:2 operator 'f' expects 1 input/s, but 0 input arguments were provided\nerror: test-error-call-count-once.cx:11:17")
	t.RunMissing("test-error-call-count-once.cx", valuetypes.CXCompilationError, "a call with the wrong number of inputs reports only that",
		"test-error-call-count-once.cx:8:2 operator 'f' expects to return", "test-error-call-count-once.cx:10:2 operator 'f' expects to return")
	t.RunWant("test-error-string-unterminated.cx", valuetypes.CXCompilationError, "an unterminated string is one error at its start, and the other errors print",
		"error: test-error-string-unterminated.cx:4:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\nerror: test-error-string-unterminated.cx:5:14 newline in string\n")
	t.RunMissing("test-error-string-unterminated.cx", valuetypes.CXCompilationError, "an unterminated string is one error at its start, and the other errors print",
		"invalid string literal")
	t.RunWant("test-error-string-newline.cx", valuetypes.CXCompilationError, "a string ends at its line, and the next lines compile",
		"error: test-error-string-newline.cx:4:14 newline in string\nerror: test-error-string-newline.cx:5:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\nerror: test-error-string-newline.cx:6:14 newline in string\nerror: test-error-string-newline.cx:7:18 cannot assign value of type 'str' to identifier 'later' of type 'i32'\n")
	t.RunMissing("test-error-string-newline.cx", valuetypes.CXCompilationError, "a string ends at its line, and the next lines compile",
		"invalid string literal", "syntax error")
	t.RunWant("test-error-package-clause-missing.cx", valuetypes.CXCompilationError, "a file without a package clause is one error at line 1",
		"error: test-error-package-clause-missing.cx:1:1 no package clause\n")
	t.RunMissing("test-error-package-clause-missing.cx", valuetypes.CXCompilationError, "a file without a package clause is one error at line 1",
		"panic", "tcp", "not checked", ":6")
	t.RunWant("test-error-package-clause-missing.cx test-error-package-clause-main.cx", valuetypes.CXCompilationError, "a file without a package clause leaves the other files compiled",
		"error: test-error-package-clause-missing.cx:1:1 no package clause\nerror: test-error-package-clause-main.cx:4:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunWant("test-error-redeclared.cx", valuetypes.CXCompilationError, "a type, function, global or method declared twice is reported with both positions, and the uses take the first",
		"error: test-error-redeclared.cx:7:6 'T' redeclared in this block, other declaration at test-error-redeclared.cx:3:6\nerror: test-error-redeclared.cx:9:4 struct 'Missing' not found in package 'main'\nerror: test-error-redeclared.cx:16:6 'f' redeclared in this block, other declaration at test-error-redeclared.cx:12:6\nerror: test-error-redeclared.cx:17:16 cannot assign value of type 'str' to identifier 'bad' of type 'i32'\nerror: test-error-redeclared.cx:21:5 'g' redeclared in this block, other declaration at test-error-redeclared.cx:20:5\nerror: test-error-redeclared.cx:27:12 method T.m already declared at test-error-redeclared.cx:23:12\nerror: test-error-redeclared.cx:28:17 cannot assign value of type 'str' to identifier 'bad2' of type 'i32'\nerror: test-error-redeclared.cx:37:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("test-error-redeclared.cx", valuetypes.CXCompilationError, "a type, function, global or method declared twice is reported with both positions, and the uses take the first",
		"panic", ":33", ":34", ":35", ":36", "function redeclaration")
	t.RunWant("test-error-redeclared-a.cx test-error-redeclared-b.cx", valuetypes.CXCompilationError, "a type or function declared in two files of a package is reported with both files",
		"error: test-error-redeclared-b.cx:3:6 'T' redeclared in this block, other declaration at test-error-redeclared-a.cx:3:6\nerror: test-error-redeclared-b.cx:7:6 'h' redeclared in this block, other declaration at test-error-redeclared-a.cx:7:6\n")
	t.RunWant("test-slice-literal-element-errors.cx", valuetypes.CXCompilationError, "every slice literal element of the wrong type is one error at the element",
		"error: test-slice-literal-element-errors.cx:17:16 cannot use value of type 'str' as type 'i32' in slice literal\nerror: test-slice-literal-element-errors.cx:17:21 cannot use value of type 'f32' as type 'i32' in slice literal\nerror: test-slice-literal-element-errors.cx:18:11 cannot use value of type 'U' as type 'T' in slice literal\nerror: test-slice-literal-element-errors.cx:19:25 cannot use value of type '[]i64' as type '[]i32' in slice literal\nerror: test-slice-literal-element-errors.cx:19:36 cannot use value of type 'i32' as type '[]i32' in slice literal\nerror: test-slice-literal-element-errors.cx:20:17 cannot use value of type 'U' as type 'T' in slice literal\nerror: test-slice-literal-element-errors.cx:20:21 cannot use value of type '[]U' as type '[]T' in slice literal\nerror: test-slice-literal-element-errors.cx:21:31 cannot use value of type 'i64' as type 'i32' in slice literal\nerror: test-slice-literal-element-errors.cx:21:37 cannot use value of type '[]i64' as type '[]i32' in slice literal\nerror: test-slice-literal-element-errors.cx:22:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("test-slice-literal-element-errors.cx", valuetypes.CXCompilationError, "every slice literal element of the wrong type is one error at the element",
		"function 'append'", "'[]U' as type 'T'")
	t.RunWant("test-error-import-missing.cx", valuetypes.CXCompilationError, "a package that cannot be loaded is one error at its import, and its members report nothing more",
		"error: test-error-import-missing.cx:3:8 could not import nope ($CXPATH/src/nope: source file or library not found)\nerror: test-error-import-missing.cx:15:2 identifier 'os.Nope' does not exist\nerror: test-error-import-missing.cx:16:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("test-error-import-missing.cx", valuetypes.CXCompilationError, "a package that cannot be loaded is one error at its import, and its members report nothing more",
		"panic", "'nope.", "struct 'T'", "'ident'", ":6:", ":9:", ":10:", ":11:", ":12:", ":13:", ":14:")
	t.RunWant("test-error-invalid-field-continue.cx", valuetypes.CXCompilationError, "a field of a value of no valid type reports nothing more",
		"error: test-error-invalid-field-continue.cx:7:2 trying to use an outputless operator in an assignment\nerror: test-error-invalid-field-continue.cx:10:14 function 'foo' not found in package 'main'\nerror: test-error-invalid-field-continue.cx:11:6 function 'foo' not found in package 'main'\nerror: test-error-invalid-field-continue.cx:12:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("test-error-invalid-field-continue.cx", valuetypes.CXCompilationError, "a field of a value of no valid type reports nothing more",
		"'ident'", ":0 ", ":8:", ":9:")
	t.RunWant("--cxpath test-import-errors test-error-import-compile.cx", valuetypes.CXCompilationError, "a package with compile errors prints them with the errors of the program, and its members report nothing more",
		"error: test-import-errors/src/broken/broken.cx:10:16 cannot assign value of type 'str' to identifier 'bad' of type 'i32'\nerror: test-import-errors/src/broken/broken.cx:18:6 'Foo' redeclared in this block, other declaration at test-import-errors/src/broken/broken.cx:9:6\nerror: test-import-errors/src/syntaxbad/syntaxbad.cx:4:8 syntax error: unexpected comma, expected ]\nerror: test-error-import-compile.cx:15:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("--cxpath test-import-errors test-error-import-compile.cx", valuetypes.CXCompilationError, "a package with compile errors prints them with the errors of the program, and its members report nothing more",
		"panic", "could not import", "'broken.", "'syntaxbad.", "invalid type", "compile.cx:6:", "compile.cx:9:", "compile.cx:10:", "compile.cx:11:", "compile.cx:12:", "compile.cx:13:", "compile.cx:14:")
	t.RunWant("--cxpath test-import-errors test-error-import-twice-a.cx test-error-import-twice-b.cx", valuetypes.CXCompilationError, "a failed import is reported once per package, whatever its path, relative to the program",
		"error: test-error-import-twice-a.cx:3:8 could not import nope (test-import-errors/src/nope: source file or library not found)\nerror: test-error-import-twice-a.cx:9:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\nerror: test-import-errors/src/lib/broken/broken.cx:4:16 cannot assign value of type 'str' to identifier 'bad' of type 'i32'\nerror: test-error-import-twice-b.cx:9:18 cannot assign value of type 'str' to identifier 'late2' of type 'i32'\n")
	t.RunMissing("--cxpath test-import-errors test-error-import-twice-a.cx test-error-import-twice-b.cx", valuetypes.CXCompilationError, "a failed import is reported once per package, whatever its path, relative to the program",
		"a/nope", "test-error-import-twice-a.cx:4:", "test-error-import-twice-b.cx:3:", "test-error-import-twice-b.cx:4:", "src/broken/broken.cx")
	t.RunWant("--cxpath test-import-errors test-error-invalid-method-call.cx", valuetypes.CXCompilationError, "a method call on a value of no valid type reports nothing more",
		"error: test-error-invalid-method-call.cx:3:8 could not import nope (test-import-errors/src/nope: source file or library not found)\nerror: test-error-invalid-method-call.cx:12:2 trying to use an outputless operator in an assignment\nerror: test-error-invalid-method-call.cx:14:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("--cxpath test-import-errors test-error-invalid-method-call.cx", valuetypes.CXCompilationError, "a method call on a value of no valid type reports nothing more",
		"invalid type", ":10:", ":11:", ":13:")
	t.RunWant("test-error-method-receivers.cx", valuetypes.CXCompilationError, "a method with more than one receiver is one error at the second, and the first receiver is used",
		"error: test-error-method-receivers.cx:7:12 method has multiple receivers\nerror: test-error-method-receivers.cx:10:12 method has multiple receivers\nerror: test-error-method-receivers.cx:23:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("test-error-method-receivers.cx", valuetypes.CXCompilationError, "a method with more than one receiver is one error at the second, and the first receiver is used",
		"panic", ":21:", ":22:")
	t.RunWant("test-error-method-receiver-type.cx", valuetypes.CXCompilationError, "a method receiver that is not a struct, or a struct that does not exist, is a compile error (skycoin-cx#515)",
		"error: test-error-method-receiver-type.cx:5:7 method receiver must be a struct or a pointer to one; 'i32' was provided\nerror: test-error-method-receiver-type.cx:9:9 struct 'Circle' not found in package 'main'\nerror: test-error-method-receiver-type.cx:13:9 struct 'Circle' not found in package 'main'\nerror: test-error-method-receiver-type.cx:18:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunWant("test-error-redeclared-kinds.cx", valuetypes.CXCompilationError, "a name declared as two of a type, a function and a global is reported at the later declaration",
		"error: test-error-redeclared-kinds.cx:7:6 'T' redeclared in this block, other declaration at test-error-redeclared-kinds.cx:3:6\nerror: test-error-redeclared-kinds.cx:10:5 'T' redeclared in this block, other declaration at test-error-redeclared-kinds.cx:3:6\nerror: test-error-redeclared-kinds.cx:14:6 'U' redeclared in this block, other declaration at test-error-redeclared-kinds.cx:12:5\nerror: test-error-redeclared-kinds.cx:21:5 'f' redeclared in this block, other declaration at test-error-redeclared-kinds.cx:18:6\nerror: test-error-redeclared-kinds.cx:27:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("test-error-redeclared-kinds.cx", valuetypes.CXCompilationError, "a name declared as two of a type, a function and a global is reported at the later declaration",
		"panic", ":23:")
	t.RunWant("test-error-redeclared-kinds-a.cx test-error-redeclared-kinds-b.cx test-error-redeclared-kinds-c.cx", valuetypes.CXCompilationError, "a type and a function or global in two files of a package are reported in file order",
		"error: test-error-redeclared-kinds-b.cx:3:6 'V' redeclared in this block, other declaration at test-error-redeclared-kinds-a.cx:3:6\nerror: test-error-redeclared-kinds-b.cx:9:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\nerror: test-error-redeclared-kinds-c.cx:3:6 'W' redeclared in this block, other declaration at test-error-redeclared-kinds-b.cx:6:5\n")
	t.RunWant("test-error-redeclared-kinds-c.cx test-error-redeclared-kinds-b.cx test-error-redeclared-kinds-a.cx", valuetypes.CXCompilationError, "a type and a function or global in two files of a package are reported in file order",
		"error: test-error-redeclared-kinds-b.cx:6:5 'W' redeclared in this block, other declaration at test-error-redeclared-kinds-c.cx:3:6\nerror: test-error-redeclared-kinds-b.cx:9:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\nerror: test-error-redeclared-kinds-a.cx:3:6 'V' redeclared in this block, other declaration at test-error-redeclared-kinds-b.cx:3:6\n")
	t.RunWant("test-slice-literal-call-errors.cx", valuetypes.CXCompilationError, "a call element of the wrong type in a struct slice literal is 7051 at the element",
		"error: test-slice-literal-call-errors.cx:21:20 cannot use value of type 'U' as type 'T' in slice literal\nerror: test-slice-literal-call-errors.cx:22:3 cannot use value of type 'U' as type 'T' in slice literal\nerror: test-slice-literal-call-errors.cx:23:12 cannot use value of type 'U' as type 'T' in slice literal\nerror: test-slice-literal-call-errors.cx:25:18 cannot use value of type 'U' as type 'T' in slice literal\nerror: test-slice-literal-call-errors.cx:26:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'\n")
	t.RunMissing("test-slice-literal-call-errors.cx", valuetypes.CXCompilationError, "a call element of the wrong type in a struct slice literal is 7051 at the element",
		"receiving variable", ":21:11")
	t.Run("testlib/args.cx test-args.cx", valuetypes.CXSuccess, "ProgramError in args lib.")
	t.Run("test-regexp-must-compile-fail.cx", valuetypes.CXRuntimeError, "ProgramError in regexp lib - MustCompile should have thrown an error.")
	t.Run("test-regexp-compile-fail.cx", valuetypes.CXSuccess, "ProgramError in regexp lib - error thrown by regexp.Compile does not matches expected error.")
	t.Run("test-regexp.cx", valuetypes.CXSuccess, "ProgramError in regexp lib.")
	t.Run("-heap-initial 0 test-regexp.cx", valuetypes.CXSuccess, "regexp.MustCompile writes its expression str after the heap grows (#485)")
	t.Run("test-regexp-not-imported.cx", valuetypes.CXCompilationError, "regexp used without importing it")
	t.Run("test-regexp-find-failed-compile.cx", valuetypes.CXRuntimeError, "Find on a Regexp that did not compile")
	t.Run("test-regexp-find-loop.cx", valuetypes.CXSuccess, "Find in a loop")
	t.Run("test-unknown-package-function.cx", valuetypes.CXCompilationError, "unknown function in a built in package")
	t.Run("--enable=file test-sys-import.cx", valuetypes.CXSuccess, "built-in packages imported by their sys/ path (skycoin-cx#380)")
	t.RunWant("test-sys-import-unknown.cx", valuetypes.CXCompilationError, "unknown sys/ import path (skycoin-cx#380)",
		`test-sys-import-unknown.cx:3 unknown built-in package "sys/nosuchpackage"`)
	t.RunWant("test-sys-import-missing.cx", valuetypes.CXCompilationError, "built-in package used without its import (skycoin-cx#380)",
		`test-sys-import-missing.cx:4 built-in package os is used without importing it, add import "sys/os"`)
	t.RunWant("test-sys-import-bare.cx", valuetypes.CXCompilationError, "built-in package imported without sys/ (skycoin-cx#380)",
		`test-sys-import-bare.cx:3 built-in package os is imported as "sys/os"`)
	t.Run("test-sys-string.cx", valuetypes.CXSuccess, "sys/string functions (skycoin-cx#382)")
	t.RunWant("test-str-renamed.cx", valuetypes.CXCompilationError, "a renamed str function names its new name (skycoin-cx#382)",
		`test-str-renamed.cx:4 str.concat was renamed to string.Concatenation in language version 2611200002, import "sys/string"`)
	t.RunWant("test-sys-string-not-imported.cx", valuetypes.CXCompilationError, "sys/string used without its import (skycoin-cx#382)",
		`test-sys-string-not-imported.cx:4 built-in package string is used without importing it, add import "sys/string"`)
	t.Run("test-unknown-native-function.cx", valuetypes.CXCompilationError, "unknown function in an imported native package")
	t.RunWant("test-unknown-type-declarations.cx", valuetypes.CXCompilationError, "every unknown type in a declaration is reported, without a crash",
		`test-unknown-type-declarations.cx:4 struct 'st' not found in package 'main'`,
		`test-unknown-type-declarations.cx:7 struct 'i' not found in package 'main'`,
		`test-unknown-type-declarations.cx:9 struct 'foo' not found in package 'main'`,
		`test-unknown-type-declarations.cx:13 struct 'Nope' not found in package 'main'`,
		`test-unknown-type-declarations.cx:14 struct 'Nope' not found in package 'main'`,
		`test-unknown-type-declarations.cx:15 struct 'Nope' not found in package 'main'`)
	t.RunWant("test-unknown-type-imported-array.cx", valuetypes.CXCompilationError, "an array of a type from a package that is not imported",
		`test-unknown-type-imported-array.cx:4 package 'other' not imported`)
	t.RunWant("test-index-too-many.cx", valuetypes.CXCompilationError, "more indexes than dimensions, and indexing a scalar",
		`test-index-too-many.cx:5 invalid indexing`,
		`test-index-too-many.cx:8 invalid indexing`,
		`test-index-too-many.cx:10 invalid indexing`,
		`test-index-too-many.cx:12 invalid indexing`)
	t.RunWant("test-struct-recursive.cx", valuetypes.CXCompilationError, "a struct that holds itself",
		`test-struct-recursive.cx:4 invalid recursive type 'T'`)
	t.RunWant("test-struct-recursive-mutual.cx", valuetypes.CXCompilationError, "two structs that hold each other",
		`test-struct-recursive-mutual.cx:8 invalid recursive type 'B'`)
	t.RunWant("test-method-missing.cx", valuetypes.CXCompilationError, "a call of a method the struct does not have",
		`test-method-missing.cx:9 type 'T' has no method 'Nope'`)
	t.RunWant("test-return-count.cx", valuetypes.CXCompilationError, "return with the wrong number of values",
		`test-return-count.cx:4 function 'f' expects to return 2 arguments, but 1 output argument was provided`,
		`test-return-count.cx:8 function 'main' expects to return 0 argument, but 1 output argument was provided`)
	t.RunWant("test-array-too-large.cx", valuetypes.CXCompilationError, "a struct and a global past the 32 bit size range",
		`test-array-too-large.cx:5 'b' is too large`,
		`test-array-too-large.cx:8 'g' is too large`)
	t.RunWant("test-array-too-large-local.cx", valuetypes.CXCompilationError, "a local array whose size wraps",
		`test-array-too-large-local.cx:4 'l' is too large`)
	t.RunWant("test-array-too-large-frame.cx", valuetypes.CXCompilationError, "locals whose frame passes the 32 bit size range",
		`test-array-too-large-frame.cx:5 'c' is too large`)
	t.RunWant("test-function-missing-in-expression.cx", valuetypes.CXCompilationError, "an unknown native inside an operator",
		`test-function-missing-in-expression.cx:4 function 'i32.nope' does not exist`)
	t.RunWant("test-function-missing-in-condition.cx", valuetypes.CXCompilationError, "an unknown function in a loop condition",
		`test-function-missing-in-condition.cx:4 function 'nope' not found in package 'main'`)
	t.RunWant("test-append-missing-value.cx", valuetypes.CXCompilationError, "append without a value",
		`test-append-missing-value.cx:5 operator 'append' expects 2 input/s, but 1 input argument`)
	t.RunWant("test-operator-argument-types.cx", valuetypes.CXCompilationError, "a type inferred operator on a type it does not support",
		`test-operator-argument-types.cx:4 invalid argument types for 'lt' operator`,
		`test-operator-argument-types.cx:5 invalid argument types for 'bitand' operator`,
		`test-operator-argument-types.cx:6 invalid argument types for 'lt' operator`)
	t.Run("test-function-return-array.cx", valuetypes.CXSuccess, "functions that return arrays")
	t.RunWant("--executor=old test-function-result-array-forms.cx", valuetypes.CXSuccess, "array results named and unnamed, from literals, of structs and pointers, on the old loop", "2 3 6 7 9\n21 5\n")
	t.RunWant("--executor=vm test-function-result-array-forms.cx", valuetypes.CXSuccess, "array results named and unnamed, from literals, of structs and pointers, on vm", "2 3 6 7 9\n21 5\n")
	t.RunWant("test-function-result-array-heap.cx", valuetypes.CXSuccess, "return &a of a local array and slices of arrays as results", "3 13 14 13\n1 9 8\n")
	t.RunWant("--executor=old test-function-results-unnamed.cx", valuetypes.CXSuccess, "unnamed results in parentheses, on the old loop", "3 4 x 14 y 13\n")
	t.RunWant("--executor=vm test-function-results-unnamed.cx", valuetypes.CXSuccess, "unnamed results in parentheses, on vm", "3 4 x 14 y 13\n")
	t.RunWant("test-function-results-unnamed-errors.cx", valuetypes.CXCompilationError, "a return that does not match unnamed results",
		"test-function-results-unnamed-errors.cx:4 function 'short' expects to return 2 arguments, but 1 output argument was provided",
		"test-function-results-unnamed-errors.cx:8:12 cannot use value of type 'i32' as type 'str' in return statement")
	t.RunWant("--executor=old test-var-grouped.cx", valuetypes.CXSuccess, "var a, b T, var a, b = x, y and var ( ... ) as globals and locals, on the old loop", "10 37 140\nf g j e 4 2.5\n")
	t.RunWant("--executor=vm test-var-grouped.cx", valuetypes.CXSuccess, "var a, b T, var a, b = x, y and var ( ... ) as globals and locals, on vm", "10 37 140\nf g j e 4 2.5\n")
	t.RunWant("test-var-grouped-errors.cx", valuetypes.CXCompilationError, "a grouped var with the wrong number of values, and a global whose value has no type",
		"test-var-grouped-errors.cx:6:5 cannot infer the type of global 'g' from its value, declare it with a type",
		"test-var-grouped-errors.cx:7:5 assignment mismatch: 2 variables but 1 value",
		"test-var-grouped-errors.cx:8:5 assignment mismatch: 2 variables but 3 values",
		"test-var-grouped-errors.cx:11:6 assignment mismatch: 2 variables but 1 value",
		"test-var-grouped-errors.cx:12:6 assignment mismatch: 2 variables but 3 values")
	t.RunMissing("test-var-grouped-errors.cx", valuetypes.CXCompilationError, "the names of a grouped var with the wrong number of values are declared, globals and locals", ":13 ", "does not exist")
	t.RunWant("test-if-composite-literal.cx", valuetypes.CXCompilationError, "a bare struct literal in an if header gets Go's hint",
		"test-if-composite-literal.cx:11:11 syntax error: unexpected ., composite literal in condition must be parenthesized")
	t.RunWant("test-if-composite-literal-parenthesized.cx", valuetypes.CXSuccess, "a parenthesized struct literal in an if or for header", "first\nloop 0\nloop 1\nsecond\n")
	t.RunWant("test-struct-literal-missing.cx", valuetypes.CXCompilationError, "a literal of a struct that does not exist is an error at its name, not a panic",
		"test-struct-literal-missing.cx:4:7 struct 'Missing' not found in package 'main'",
		"test-struct-literal-missing.cx:5:8 struct 'Missing' not found in package 'main'",
		"test-struct-literal-missing.cx:6:17 struct 'Missing' not found in package 'main'")
	t.RunMissing("test-struct-literal-missing.cx", valuetypes.CXCompilationError, "a literal of a struct that does not exist is an error, not a panic", "panic", "goroutine", ":7 ")
	t.RunWant("test-global-inferred.cx", valuetypes.CXSuccess, "a global without a type takes the type of any value, globals initialize after the globals they refer to, and a method of a global is called",
		"3 1 2 4\n2 4 7 3\n2.5000000 10 8 true\n6 x 42 6 12\n")
	t.RunWant("test-global-inferred-errors.cx", valuetypes.CXCompilationError, "an initialization cycle among globals without a type, and a value with no type",
		"test-global-inferred-errors.cx:6:5 initialization cycle for 'a': a refers to b, b refers to c, c refers to a",
		"test-global-inferred-errors.cx:9:5 initialization cycle for 'self': self refers to itself",
		"test-global-inferred-errors.cx:10:5 cannot infer the type of global 'nothing' from its value, declare it with a type")
	t.RunMissing("test-global-inferred-errors.cx", valuetypes.CXCompilationError, "the globals of a cycle report nothing more", ":11 ", ":14 ", ":15 ", "cxGlobal")
	t.RunWant("test-global-init-cycle.cx", valuetypes.CXCompilationError, "an initialization cycle through a function, and one of globals with a type",
		"test-global-init-cycle.cx:7:5 initialization cycle for 'loop': loop refers to back, back refers to loop",
		"test-global-init-cycle.cx:8:5 initialization cycle for 'typed': typed refers to typedToo, typedToo refers to typed")
	t.RunWant("--cxpath test-workspace test-global-inferred-imported.cx", valuetypes.CXSuccess, "a global without a type of a struct of an imported package, and a method call of a global in an initializer", "2.0000000 6.0000000\n")
	t.RunWant("test-global-field-failed-import.cx", valuetypes.CXCompilationError, "a field of a struct of a failed import reports nothing, and a field of an i32 names the i32",
		"test-global-field-failed-import.cx:3:8 could not import math/geometry",
		"test-global-field-failed-import.cx:13:21 illegal method call or field access on identifier 'n' of primitive type 'i32'")
	t.RunMissing("test-global-field-failed-import.cx", valuetypes.CXCompilationError, "a field of a struct of a failed import reports nothing", ":12 ", "Holder")
	t.RunWant("test-struct-literal-field-position.cx", valuetypes.CXCompilationError, "the errors of a literal are at its field or its start, not at its closing brace",
		"test-struct-literal-field-position.cx:8:15 duplicate field name 'a' in struct literal",
		"test-struct-literal-field-position.cx:9:9 field 'b' not found in struct 'T'",
		"test-struct-literal-field-position.cx:11:3 field 'b' not found in struct 'T'",
		"test-struct-literal-field-position.cx:12:7 array literal of length 2 has 3 elements",
		"test-struct-literal-field-position.cx:13:7 struct 'Missing' not found in package 'main'",
		"test-struct-literal-field-position.cx:14:7 struct 'Missing' not found in package 'main'")
	t.RunMissing("test-struct-literal-field-position.cx", valuetypes.CXCompilationError, "a field that does not exist reports nothing more", "expected argument", ":15 ")
	t.RunWant("test-one-line-function-errors.cx", valuetypes.CXCompilationError, "an error of a one line function is not at the token after it",
		"test-one-line-function-errors.cx:3 operands are not of the same type: f32 != i32",
		"test-one-line-function-errors.cx:5:20 function 'count' expects to return 1 argument, but 2 output arguments were provided",
		"test-one-line-function-errors.cx:7 label 'nowhere' is not defined",
		"test-one-line-function-errors.cx:9 break is not in a loop")
	t.RunMissing("test-one-line-function-errors.cx", valuetypes.CXCompilationError, "an error of a one line function is not at the token after it", ":3:", ":7:", ":9:")
	t.RunWant("test-short-decl-chained.cx", valuetypes.CXSuccess, "x := a * b + c declares x with the type of the last operator", "true\ntrue\nfalse\n4\n7\ntrue\n")
	t.RunWant("test-global-composite-forms.cx", valuetypes.CXSuccess, "globals whose literal spans lines or sits in a var group, with and without a type", "1 2 3 4\n6 8 9 10\n11 12 13\n")
	t.RunMissing("-ast test-pointers.cx", valuetypes.CXCompilationError, "-ast of a program missing its struct file reports errors, not a panic", "panic", "goroutine")
	t.RunWant("-ast ./examples/packages.cx", valuetypes.CXSuccess, "-ast prints packages after the packages they import, the same on every run",
		"0.- Package: foo\n", "1.- Package: nosal\n", "2.- Package: bar\n", "3.- Package: main\n")
	t.RunWant("-ast --cxpath test-workspace test-struct-literal-external-ref.cx", valuetypes.CXSuccess, "-ast prints an imported package before the program",
		"0.- Package: geometry\n", "1.- Package: main\n")
	t.Run("test-imported-global-assign.cx", valuetypes.CXSuccess, "assignment to a global of an imported package")
	t.RunWant("--executor=old test-compound-results.cx", valuetypes.CXSuccess, "x := f() and globals of *str, *[]i32, []*i32, [2]*i32 and [2][]i32 results, on the old loop", "s 3 7 7 5 1 2\ns 3 1 7 7 5\n1 2 9 2\n")
	t.RunWant("test-compound-results.cx", valuetypes.CXSuccess, "x := f() and globals of *str, *[]i32, []*i32, [2]*i32 and [2][]i32 results", "s 3 7 7 5 1 2\ns 3 1 7 7 5\n1 2 9 2\n")
	t.RunWant("--executor=old test-local-infer-pointer.cx", valuetypes.CXSuccess, "var c = p and c := p of pointer and compound locals, on the old loop", "3 4 5 5 5 5 6\n")
	t.RunWant("test-local-infer-pointer.cx", valuetypes.CXSuccess, "var c = p and c := p of pointer and compound locals", "3 4 5 5 5 5 6\n")
	t.RunWant("test-error-outputless-value.cx", valuetypes.CXCompilationError, "a call with no results used as a slice, array or struct element, an argument or an operand",
		"test-error-outputless-value.cx:14 function 'none' returns no value",
		"test-error-outputless-value.cx:15 function 'none' returns no value",
		"test-error-outputless-value.cx:16 function 'none' returns no value",
		"test-error-outputless-value.cx:17 function 'none' returns no value",
		"test-error-outputless-value.cx:18 function 'none' returns no value",
		"test-error-outputless-value.cx:20:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'")
	t.RunMissing("test-error-outputless-value.cx", valuetypes.CXCompilationError, "a call with no results used as a value reports nothing more", "panic", ":19 ")
	t.RunWant("test-error-missing-type-local.cx", valuetypes.CXCompilationError, "locals of a struct that does not exist",
		"test-error-missing-type-local.cx:4:12 struct 'Missing' not found in package 'main'",
		"test-error-missing-type-local.cx:7:11 struct 'Missing' not found in package 'main'",
		"test-error-missing-type-local.cx:12:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'")
	t.RunMissing("test-error-missing-type-local.cx", valuetypes.CXCompilationError, "the locals of a struct that does not exist report nothing more", ":8 ", ":9 ", ":10 ", ":11 ", "does not exist")
	t.RunWant("test-error-short-decl-count.cx", valuetypes.CXCompilationError, "a := with the wrong number of values",
		"test-error-short-decl-count.cx:10:2 assignment mismatch: 2 variables but 3 values",
		"test-error-short-decl-count.cx:12:2 assignment mismatch: 3 variables but 2 values",
		"test-error-short-decl-count.cx:15:2 assignment mismatch: 2 variables but 3 values",
		"test-error-short-decl-count.cx:17:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'")
	t.RunMissing("test-error-short-decl-count.cx", valuetypes.CXCompilationError, "the names of a := with the wrong number of values report nothing more, and an existing name is not redeclared", ":11 ", ":13 ", ":16 ", "redeclared")
	t.RunWant("test-trailing-comma.cx", valuetypes.CXSuccess, "a trailing comma in composite literals, call arguments, parameters and results", "2 2 4 2 3 8 2 3\n2 1 5 7\n")
	t.RunWant("test-trailing-comma-errors.cx", valuetypes.CXCompilationError, "a struct literal of no fields may not end in a comma", "test-trailing-comma-errors.cx:8 syntax error: unexpected comma, expected name")
	t.RunWant("test-printf-pointer.cx", valuetypes.CXSuccess, "%v prints a pointer in hexadecimal, nil as <nil> and a pointer to a struct as &{...}, and %+v as %v",
		"0x\n<nil> <nil>\n{x: 1, s: a} {x: 1, s: a}\n&{x: 1, s: a}\n[]\n[1, 2] [1, 2]\n{p: <nil>, n: []} {p: <nil>, n: []}\n&{x: 1, s: a}\n0x\n")
	t.RunWant("test-global-init-method.cx", valuetypes.CXSuccess, "a global read only in a method initializes before the global whose initializer calls the method", "40 42 40 41\n")
	t.RunWant("test-global-init-method-cycle.cx", valuetypes.CXCompilationError, "an initialization cycle through a method",
		"test-global-init-method-cycle.cx:13:5 initialization cycle for 'loop': loop refers to T.get, T.get refers to loop")
	t.RunWant("--executor=old test-global-multiple-results.cx", valuetypes.CXSuccess, "var a, b = f() and var a, b T = f() as globals and locals, on the old loop", "4 x 4 x\n9 3 3 4 4 x\n")
	t.RunWant("test-global-multiple-results.cx", valuetypes.CXSuccess, "var a, b = f() and var a, b T = f() as globals and locals", "4 x 4 x\n9 3 3 4 4 x\n")
	t.RunWant("test-global-multiple-results-errors.cx", valuetypes.CXCompilationError, "var a, b = f() with the wrong number of results",
		"test-global-multiple-results-errors.cx:11:5 assignment mismatch: 2 variables but 1 value",
		"test-global-multiple-results-errors.cx:12:5 assignment mismatch: 3 variables but 2 values",
		"test-global-multiple-results-errors.cx:13:5 assignment mismatch: 2 variables but 1 value")
	t.RunMissing("test-global-multiple-results-errors.cx", valuetypes.CXCompilationError, "the globals of var a, b = f() with the wrong number of results report nothing more", ":16 ")
	t.RunWant("test-printf-escapes.cx", valuetypes.CXSuccess, "printf and sprintf print the bytes of a %s argument and their format as they are, and %% prints %",
		"2 [\\n]\n4 [50\\%] [50\\%]\n7 \\n|50\\%\n100% x\\%!y(MISSING)\n")
	t.RunWant("--executor=old test-printf-escapes.cx", valuetypes.CXSuccess, "printf and sprintf print a %s argument as it is, on the old loop",
		"2 [\\n]\n4 [50\\%] [50\\%]\n7 \\n|50\\%\n100% x\\%!y(MISSING)\n")
	t.RunWant("test-string-escapes.cx", valuetypes.CXSuccess, "string literals take the escapes of Go, \\x, octal, \\u and \\U among them",
		"9: 7 8 12 10 13 9 11 92 34\n4: 65 127 255 0\n4: 65 0 255 127\n6: 65 195 169 226 130 172\n5: 65 240 159 152 128\n4: 97 92 110 98\n0:\n")
	t.RunWant("test-error-string-escapes.cx", valuetypes.CXCompilationError, "an invalid escape is the error of Go and the errors after it are found",
		"test-error-string-escapes.cx:4:16 unknown escape",
		"test-error-string-escapes.cx:5:18 invalid character 'g' in hexadecimal escape",
		"test-error-string-escapes.cx:6:19 octal escape value 256 > 255",
		"test-error-string-escapes.cx:7:21 escape is invalid Unicode code point U+D800",
		"test-error-string-escapes.cx:8:25 escape is invalid Unicode code point U+110000",
		"test-error-string-escapes.cx:9:16 unknown escape",
		"test-error-string-escapes.cx:10:16 unknown escape",
		"test-error-string-escapes.cx:11:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'")
	t.RunMissing("test-error-string-escapes.cx", valuetypes.CXCompilationError, "an invalid escape is reported once", "invalid string literal")
	t.RunWant("test-string-bytes.cx", valuetypes.CXSuccess, "string.BytesOfString and string.StringOfBytes copy between str and []ui8",
		"3 104 255 105\n4\n0 []\n[]\nabc xbc\n")
	t.RunWant("test-pointer-pointer-field.cx", valuetypes.CXSuccess, "(*pp).f on a **T dereferences pp twice",
		"1 2\n9 9 9 9\n1 9\n5\n5 9\n")
	t.RunWant("test-literal-compound.cx", valuetypes.CXSuccess, "composite literals of pointer and mixed array and slice element types, elided elements and &T elided",
		"a 1 2\nb 7 8\nc 5\nd 2 2\ne 3 3 2\nf 3\ng 9\nh 0\ni 2 3\n")
	t.RunWant("--executor=old test-literal-compound.cx", valuetypes.CXSuccess, "composite literals of pointer and mixed element types, on the old loop",
		"a 1 2\nb 7 8\nc 5\nd 2 2\ne 3 3 2\nf 3\ng 9\nh 0\ni 2 3\n")
	t.RunWant("test-index-three-level.cx", valuetypes.CXSuccess, "indexes of three levels of arrays, slices and pointers",
		"a 8 1 0\nb 7 6\nc 9\nd 7\ne 7\nf 7\ng 7\n")
	t.RunWant("test-global-str-zero.cx", valuetypes.CXSuccess, "a global str without a value is empty, also read through a pointer",
		"[][][][x] 0\n")
	t.RunWant("--executor=old test-global-str-zero.cx", valuetypes.CXSuccess, "a global str without a value is empty, on the old loop",
		"[][][][x] 0\n")
	t.RunWant("test-short-declaration-reuse.cx", valuetypes.CXSuccess, "a := of several names assigns the names its scope has and declares the others",
		"5 2 6\n7 8\n5\n3 4\ny 1\n")
	t.RunWant("test-error-short-declaration-no-new.cx", valuetypes.CXCompilationError, "a := whose names its scope has all",
		"test-error-short-declaration-no-new.cx:5:2 no new variables on left side of :=",
		"test-error-short-declaration-no-new.cx:7:2 cannot assign value of type 'str' to identifier 'x' of type 'i32'")
	t.RunMissing("test-error-short-declaration-no-new.cx", valuetypes.CXCompilationError, "a := of several names reports no name redeclared", "redeclared")
	t.RunWant("issue-51.cx", valuetypes.CXCompilationError, "a local named like a global is a compile error (skycoin-cx#25, #486)",
		"issue-51.cx:8:6 'i' shadows global declared at issue-51.cx:5:5")
	t.RunWant("test-global-declaration.cx", valuetypes.CXSuccess, "global statements of the package and of an imported package (#486)",
		"2 1 9 7 14\n")
	t.RunWant("test-global-declaration-errors.cx", valuetypes.CXCompilationError, "errors of the global statement (#486)",
		"test-global-declaration-errors.cx:13:2 function 'undeclared' uses global 'counter' without declaring it, add global counter at the top of its body",
		"test-global-declaration-errors.cx:14:12 function 'undeclared' uses global 'config.Limit' without declaring it",
		"test-global-declaration-errors.cx:18:18 global 'spare' declared and not used in function 'unused'",
		"test-global-declaration-errors.cx:24:2 global declarations must be the first statements of a function body",
		"test-global-declaration-errors.cx:29:3 global declarations must be the first statements of a function body",
		"test-global-declaration-errors.cx:34:9 'missing' is not a global of package 'main'",
		"test-global-declaration-errors.cx:34:18 'config.Missing' is not a global of package 'config'",
		"test-global-declaration-errors.cx:34:34 package 'other' of global 'other.Limit' is not imported",
		"test-global-declaration-errors.cx:34:56 global 'counter' declared twice")
	t.RunWant("test-shadow-errors.cx", valuetypes.CXCompilationError, "a name that shadows another is a compile error (#486)",
		"test-shadow-errors.cx:11:16 'counter' shadows global declared at test-shadow-errors.cx:5:5",
		"test-shadow-errors.cx:14:16 'helper' shadows function declared at test-shadow-errors.cx:9:6",
		"test-shadow-errors.cx:19:6 'os' shadows package imported at test-shadow-errors.cx:3:8",
		"test-shadow-errors.cx:20:2 'Point' shadows type declared at test-shadow-errors.cx:7:6",
		"test-shadow-errors.cx:23:3 'x' shadows variable declared at test-shadow-errors.cx:21:2",
		"test-shadow-errors.cx:26:6 'counter' shadows global declared at test-shadow-errors.cx:5:5")
	t.RunWant("test-import-scoped-module.cx", valuetypes.CXSuccess, "import . brings every name of a package into the package, as types, calls, literals and globals (#487)",
		"7 7 7 5 8 0\n")
	t.RunWant("test-import-scoped-names.cx", valuetypes.CXSuccess, "import (names) into the package and the file, of a package and of a built-in package, with a shorthand global statement (#487)",
		"6 9 9\ndone\n")
	t.RunWant("test-import-scoped-file-a.cx test-import-scoped-file-b.cx", valuetypes.CXSuccess, "import file . brings the names into its file, and another file of the package reaches them qualified (#487)",
		"3\n8\n")
	t.RunWant("test-import-scoped-file-a.cx test-import-scoped-file-c.cx", valuetypes.CXCompilationError, "import file . does not bring the names into another file of the package (#487)",
		"test-import-scoped-file-c.cx:4:12 function 'Distance' not found in package 'main'",
		"test-import-scoped-file-c.cx:4:21 struct 'Point' not found in package 'main'")
	t.RunWant("test-import-scoped-file-b.cx", valuetypes.CXCompilationError, "a struct literal of a package the file does not import is a compile error, where it panicked",
		"test-import-scoped-file-b.cx:4:39 package 'geometry' not imported")
	t.RunWant("--cxpath test-workspace test-import-scoped-workspace.cx", valuetypes.CXSuccess, "import file (name) of a workspace package (#487)", "3.14159\n")
	t.RunWant("test-import-masking-errors.cx", valuetypes.CXCompilationError, "a name that masks a name an import brings into scope, and the invalid import forms (#487)",
		"test-import-masking-errors.cx:20:8 'physics' masks the name imported at test-import-masking-errors.cx:21:10",
		"test-import-masking-errors.cx:22:9 'Count' masks the name imported at test-import-masking-errors.cx:21:10",
		"test-import-masking-errors.cx:23:9 'Missing' is not a global, function or type of package 'geometry'",
		"test-import-masking-errors.cx:23:28 invalid import: 'Distance' is listed twice",
		"test-import-masking-errors.cx:24:8 invalid import: 'module' cannot stand before an import, only file can",
		"test-import-masking-errors.cx:26:5 'Size' masks the name imported at test-import-masking-errors.cx:21:10",
		"test-import-masking-errors.cx:28:6 'Area' masks the name imported at test-import-masking-errors.cx:21:10",
		"test-import-masking-errors.cx:30:6 'Point' masks the name imported at test-import-masking-errors.cx:21:10",
		"test-import-masking-errors.cx:34:10 'Distance' masks the name imported at test-import-masking-errors.cx:21:10",
		"test-import-masking-errors.cx:35:2 'Count' masks the name imported at test-import-masking-errors.cx:21:10")
	t.RunWant("test-import-masking-warning.cx", valuetypes.CXSuccess, "//cx:masking warning makes masking in its function a warning, and the local hides the import there (#487)",
		"warning: test-import-masking-warning.cx:14:12 'Distance' masks the name imported at test-import-masking-warning.cx:11:16",
		"warning: test-import-masking-warning.cx:15:6 'Count' masks the name imported at test-import-masking-warning.cx:11:9",
		"10 11 7 3\n")
	t.RunWant("test-error-single-value.cx", valuetypes.CXCompilationError, "a call of several results or of none used as one value",
		"test-error-single-value.cx:12 multiple-value two() (value of type (a i32, b i32)) in single-value context",
		"test-error-single-value.cx:13 function 'none' returns no value",
		"test-error-single-value.cx:14:17 cannot assign value of type 'str' to identifier 'late' of type 'i32'")
	t.RunMissing("test-error-single-value.cx", valuetypes.CXCompilationError, "the single value errors report nothing more", "redeclared", "expects to return")
	t.RunWant("test-short-circuit-fields.cx", valuetypes.CXSuccess, "&& and || on field, element and pointer field operands, on either side",
		"true true true\nfalse false false\ntrue true true\ntrue false true\ntrue true\nfalse false\nfalse true\ncalls 9\n")
	t.RunWant("--executor=vm test-short-circuit-fields-vm.cx", valuetypes.CXSuccess, "&& and || on field, element and pointer field operands, on the vm",
		"true true true\nfalse false false\ntrue true true\ntrue false true\ntrue true\nfalse false\ncalls 10\n")
	t.RunWant("--executor=old test-short-circuit-fields-vm.cx", valuetypes.CXSuccess, "&& and || on field, element and pointer field operands, on the old loop",
		"true true true\nfalse false false\ntrue true true\ntrue false true\ntrue true\nfalse false\ncalls 10\n")
	// t.Run("test-cipher.cx", types.CXSuccess, "ProgramError in cipher lib.")

	// issues
	t.Run("issue-207.cx", valuetypes.CXCompilationError, "Type casting error not reported.")
	// FAILING: panics, gl and glfw only exist with -tags cxfx; see #30
	// t.RunEx("issue-208.cx", types.CXCompilationError, "Panic if return value is not used.", runner.TestGui|runner.TestStable, 0)
	t.Run("issue-214.cx", valuetypes.CXSuccess, "String not working across packages")
	t.Run("issue-215.cx issue-215a.cx", valuetypes.CXSuccess, "Order of files matters for structs")
	t.Run("issue-215a.cx issue-215.cx", valuetypes.CXSuccess, "Order of files matters for structs")
	// FAILING: panics, gl and glfw only exist with -tags cxfx; see #31
	// t.RunEx("issue-216.cx", types.CXCompilationError, "Panic when calling gl.BindBuffer with only one argument.", runner.TestGui|runner.TestStable, 0)
	// FAILING: gl.BufferData takes []ui8, so a []f32 is a compile error.
	t.RunEx("issue-217.cx", valuetypes.CXSuccess, "Panic when giving []f32 argument to gl.BufferData", testrunner.TestGui|testrunner.TestIssue, 0)
	t.Run("issue-218.cx", valuetypes.CXSuccess, "Struct field crushed")
	t.Run("issue-219.cx", valuetypes.CXSuccess, "Failed to modify value in an array")
	t.Run("issue-220.cx", valuetypes.CXSuccess, "Panic when trying to index (using a var) an array, member of a struct passed as a function argument")
	t.Run("issue-27.cx", valuetypes.CXSuccess, "Failed to use shorthand operator-assign (+=, etc.) for arithmetic statements")
	t.Run("issue-221.cx", valuetypes.CXSuccess, "Can't call method from package")
	t.Run("issue-222.cx", valuetypes.CXSuccess, "Can't call method if it has a parameter")
	t.Run("issue-223.cx", valuetypes.CXSuccess, "Panic when using arithmetic to index an array field of a struct")
	t.Run("issue-224.cx", valuetypes.CXSuccess, "Panic if return value is used in an expression")
	t.Run("issue-225.cx", valuetypes.CXSuccess, "Using a variable to store the return boolean value of a function doesnt work with an if statement")
	t.Run("issue-226.cx", valuetypes.CXSuccess, "Panic when accessing property of struct array passed in as argument to func")
	t.Run("issue-227.cx", valuetypes.CXSuccess, "Unexpected results when accessing arrays of structs in a struct")
	t.Run("issue-230.cx", valuetypes.CXSuccess, "Inline initializations and arrays")
	t.Run("issue-231.cx", valuetypes.CXSuccess, "Slice keeps growing though it's cleared inside the loop")
	t.Run("issue-232.cx", valuetypes.CXSuccess, "Scope not working in loops")
	t.Run("issue-233.cx", valuetypes.CXSuccess, "Interdependant Structs")
	t.Run("issue-234.cx", valuetypes.CXCompilationError, "Panic when trying to access an invalid field. (skycoin-cx#14)")
	t.Run("issue-235.cx", valuetypes.CXCompilationError, "No compilation error when using an using an invalid identifier")
	t.Run("issue-236a.cx issue-236.cx", valuetypes.CXSuccess, "Silent name clash between packages")
	t.Run("issue-236.cx issue-236a.cx", valuetypes.CXSuccess, "Silent name clash between packages")
	t.Run("issue-237.cx", valuetypes.CXCompilationError, "Invalid implicit cast.")
	t.Run("issue-238.cx", valuetypes.CXCompilationError, "Panic when using +* in an expression")
	t.Run("issue-239.cx", valuetypes.CXCompilationError, "No compilation error when defining a struct with duplicate fields.")
	t.Run("issue-240.cx", valuetypes.CXSuccess, "Can't define struct with a single character identifier.")
	t.Run("issue-241.cx", valuetypes.CXSuccess, "Panic when variable used in if statement without parenthesis.")
	t.Run("issue-242.cx", valuetypes.CXSuccess, "Struct field stomped")
	t.Run("issue-243.cx", valuetypes.CXCompilationError, "No compilation error when indexing an array with a non integral var.")
	t.Run("issue-244a.cx", valuetypes.CXSuccess, "Panic when a field of a struct returned by a function is used in an expression")
	t.Run("issue-244b.cx", valuetypes.CXSuccess, "Panic when a field of a struct returned by a function is used in an expression")
	t.Run("issue-245a.cx issue-245.cx", valuetypes.CXCompilationError, "No compilation error when using var without package qualification.")
	t.Run("issue-246.cx", valuetypes.CXSuccess, "No compilation error when passing *i32 as an i32 arg and conversely")
	t.Run("issue-246a.cx", valuetypes.CXCompilationError, "No compilation error when passing *i32 as an i32 arg and conversely")
	t.RunWant("issue-247.cx", valuetypes.CXCompilationError, "dereferencing an i32 var is a compile error (skycoin-cx#15)", "issue-247.cx:7:24 cannot dereference non-pointer 'i' of type 'i32'")
	t.Run("issue-248.cx", valuetypes.CXSuccess, "Wrong pointer behaviour.")
	t.Run("issue-249.cx", valuetypes.CXSuccess, "Return from a function doesnt work")
	t.Run("issue-249b.cx", valuetypes.CXCompilationError, "Mismatched number of returning arguments is not throwing an error")
	t.Run("issue-250.cx", valuetypes.CXCompilationError, "No compilation error when var is accessed outside of its declaring scope")
	t.RunWant("issue-251.cx", valuetypes.CXSuccess, "Panic when a str var is shadowed by a struct var in another scope", "test\n")
	// FAILING: fails, glfw only exists with -tags cxfx; see #32
	// t.RunEx("issue-252.cx", types.CXSuccess, "glfw.GetCursorPos() throws error", runner.TestGui|runner.TestStable, 0)
	t.Run("issue-253.cx", valuetypes.CXSuccess, "Inline field and index 'dereferences' to function calls' outputs")
	t.Run("issue-254.cx", valuetypes.CXCompilationError, "No compilation error when redeclaring a variable")
	t.Run("issue-255.cx", valuetypes.CXSuccess, "Multi-dimensional slices don't work")
	t.Run("issue-256.cx", valuetypes.CXSuccess, "can't prefix a (f32) variable with minus to flip it's signedness")
	t.Run("issue-257.cx", valuetypes.CXCompilationError, "Using int literal 0 where 0.0 was needed gave no error")
	t.Run("issue-258.cx", valuetypes.CXSuccess, "error with sending references of structs to functions")
	t.Run("issue-258b.cx", valuetypes.CXSuccess, "error with references to struct literals")
	t.Run("issue-259.cx", valuetypes.CXCompilationError, "struct identifier (when initializing fields) can be with or without a '&' prefix, with no CX error")
	t.Run("issue-260.cx", valuetypes.CXCompilationError, "can assign to previously undeclared vars with just '='")
	t.Run("issue-261.cx", valuetypes.CXSuccess, "empty code blocks (even if they contain commented-out lines) crash like this")
	t.Run("issue-262.cx", valuetypes.CXSuccess, "increment operator ++ does not work")
	t.Run("issue-263.cx", valuetypes.CXSuccess, "Method does not work")
	t.Run("issue-264.cx", valuetypes.CXSuccess, "Cannot use bool variable in if expression")
	t.Run("issue-265.cx", valuetypes.CXSuccess, "CX Parser does not recognize method")
	t.Run("issue-266.cx", valuetypes.CXSuccess, "Goto not working on windows")
	t.Run("issue-267.cx", valuetypes.CXSuccess, "Methods with pointer receivers don't work")
	t.RunEx("issue-268.cx", valuetypes.CXSuccess, "when using 2 f32 out parameters, only the value of the 2nd gets through", testrunner.TestGui, 0)
	t.Run("issue-269.cx", valuetypes.CXCompilationError, "Variable redeclaration should not be allowed")
	t.Run("issue-270.cx", valuetypes.CXSuccess, "Short variable declarations are not working with calls to methods or functions")
	t.Run("issue-271.cx", valuetypes.CXCompilationError, "Panic when using equality operator between a bool and an i32")
	t.Run("issue-272.cx", valuetypes.CXSuccess, "String concatenation using the + operator doesn't work")
	t.Run("issue-273.cx", valuetypes.CXSuccess, "Argument list is not parsed correctly")
	t.Run("issue-274.cx", valuetypes.CXSuccess, "Dubious error message when indexing an array with a substraction expression")
	t.Run("issue-275.cx", valuetypes.CXSuccess, "Dubious error message when inline initializing a slice")
	t.Run("issue-276a.cx issue-276.cx", valuetypes.CXSuccess, "Troubles when accessing a global var from another package")
	t.Run("issue-277.cx", valuetypes.CXSuccess, "same func names (but in different packages) collide")
	t.Run("issue-278.cx", valuetypes.CXCompilationError, "can use vars from other packages without a 'packageName.' prefix")
	t.Run("issue-279.cx", valuetypes.CXSuccess, "False positive when detecting variable redeclaration.")
	t.Run("issue-279a.cx", valuetypes.CXSuccess, "False positive when detecting variable redeclaration.")
	t.Run("issue-280.cx", valuetypes.CXSuccess, "Problem with struct literals in short variable declarations")
	t.Run("issue-281.cx", valuetypes.CXSuccess, "Panic when using the return value of a function in a short declaration")
	t.Run("issue-282a.cx", valuetypes.CXCompilationError, "Panic when inserting a new line in a string literal")
	t.Run("issue-282b.cx", valuetypes.CXSuccess, "Panic when inserting a new line in a string literal")
	t.Run("issue-283.cx", valuetypes.CXCompilationError, "Panic when declaring a variable of an unknown type")
	t.Run("issue-284.cx", valuetypes.CXCompilationError, "No compilation error when using arithmetic operators on struct instances")
	t.RunEx("issue-285.cx", valuetypes.CXSuccess, "Parser gets confused with `2 -2`", testrunner.TestStable, 0)
	t.Run("issue-286.cx", valuetypes.CXSuccess, "Panic in when assigning an empty initializer list to a []i32 variable")
	t.Run("issue-287.cx", valuetypes.CXSuccess, "Cx stack overflow when appending to a slice passed by address")
	t.Run("issue-288.cx", valuetypes.CXCompilationError, "Panic when trying to assign return value of a function returning void")
	t.Run("issue-289.cx", valuetypes.CXCompilationError, "Panic when using a function declared in another package without importing the package")
	t.Run("issue-290.cx", valuetypes.CXSuccess, "Cx memory stomped")
	t.Run("issue-291.cx", valuetypes.CXSuccess, "Invalid offset calculation of non literal strings when appended to a slice")
	t.Run("issue-292.cx", valuetypes.CXCompilationError, "Panic when calling a function from another package where the package name alias a local variable name")
	t.Run("issue-293.cx", valuetypes.CXSuccess, "Garbage memory when passing the address of slice element to a function")
	t.Run("issue-294.cx", valuetypes.CXSuccess, "Type deduction of struct field fails")
	t.Run("issue-295.cx", valuetypes.CXCompilationError, "No compilation error when assigning a i32 value to a []i32 variable")
	t.Run("issue-296.cx", valuetypes.CXCompilationError, "No compilation error when comparing value of different types")
	t.Run("-stack-size 30 issue-297a.cx", valuetypes.CXRuntimeStackOverflowError, "No stack overflow error")
	t.Run("-heap-initial 100 -heap-max 110 issue-297b.cx", valuetypes.CXRuntimeHeapExhaustedError, "No heap exhausted error")
	t.RunWant("-heap-max 64K test-heap-cap-exceeded.cx", valuetypes.CXRuntimeHeapExhaustedError, "a slice past the heap cap is an out of memory error that names the request and the cap (D42)",
		"test-heap-cap-exceeded.cx:7, CX_RUNTIME_HEAP_EXHAUSTED_ERROR, out of memory: ", " do not fit the heap cap of 65536 bytes, raise it with -heap-max\n")
	t.RunWant("-heap-max 512K test-heap-cap-near.cx", valuetypes.CXSuccess, "a live heap near the heap cap collects before the heap is full (D42)", "60000 59999 1999 nx garbagestr\n")
	t.Run("issue-298.cx", valuetypes.CXSuccess, "Argument type deduction failed when passing address of an i32 struct field to a function accepting *i32 argument.")
	t.Run("issue-299.cx", valuetypes.CXCompilationError, "Type checking is not working with receiving variables of unexpected types")
	t.Run("issue-300.cx", valuetypes.CXSuccess, "Crash when using a constant expression in a slice literal expression")
	t.Run("issue-301-a.cx", valuetypes.CXCompilationError, "Can redeclare variables if they are inline initialized")
	t.Run("issue-301-b.cx", valuetypes.CXCompilationError, "Can redeclare variables if they are inline initialized")
	t.Run("issue-302.cx", valuetypes.CXSuccess, "Trying to determine the length of a slice of struct instances throws an error.")
	t.Run("issue-67.cx", valuetypes.CXCompilationError, "Panic when using void return value of a function in a for loop expression")
	t.Run("issue-66.cx", valuetypes.CXSuccess, "Wrong sprintf behaviour when printing boolean values with %v")
	t.Run("issue-65.cx", valuetypes.CXSuccess, "for true {} loop scope is not executed")
	t.Run("issue-64.cx", valuetypes.CXSuccess, "for loop using boolean value is not compiling")
	t.Run("issue-303.cx", valuetypes.CXSuccess, "Concatenation of str variables with + operator doesn't work")
	t.Run("issue-304.cx", valuetypes.CXSuccess, "Short declaration doesn't compile with opcode return value")
	t.Run("issue-305.cx", valuetypes.CXSuccess, "Compilation error when struct field is named 'input' or 'output'")
	t.RunWant("issue-63.cx", valuetypes.CXCompilationError, "No compilation error when using empty argument list after function call (skycoin-cx#22)",
		"issue-63.cx:10 the result of a call is not a function")
	t.Run("issue-306.cx", valuetypes.CXCompilationError, "No compilation error when using float value in place of boolean expression")
	t.Run("issue-308.cx", valuetypes.CXCompilationError, "Panic when package contains duplicate function signature")
	t.Run("issue-61-a.cx", valuetypes.CXSuccess, "Compilation error when using return value of a member method in a expression (skycoin-cx#24)")
	t.Run("issue-61-b.cx", valuetypes.CXSuccess, "Compilation error when using return value of a member method in a expression (skycoin-cx#24)")
	t.Run("issue-309.cx", valuetypes.CXSuccess, "Compilation error when left hand side of an assignment expression is a struct field")
	t.Run("--heap-initial=1M issue-60.cx", valuetypes.CXSuccess, "Crash when INIT_HEAP_SIZE limit is reached ")
	t.Run("--heap-initial=1M issue-59-a.cx", valuetypes.CXSuccess, "Crash in garbage collector when heap is resized")
	t.Run("--heap-initial=1M issue-59-b.cx", valuetypes.CXSuccess, "Crash in garbage collector when heap is resized")
	t.RunEx("issue-53-a.cx", valuetypes.CXSuccess, "Issues with slice of type T where sizeof T is different than 4 ", testrunner.TestStable, 0)
	t.RunEx("issue-53-b.cx", valuetypes.CXSuccess, "Issues with slice of type T where sizeof T is different than 4 ", testrunner.TestStable, 0)
	t.RunEx("issue-53-c.cx", valuetypes.CXSuccess, "Issues with slice of type T where sizeof T is different than 4 ", testrunner.TestStable, 0)
	t.Run("issue-49.cx", valuetypes.CXCompilationError, "No compilation error when assigning an literal which overflow the receiving type")
	t.Run("issue-48.cx", valuetypes.CXSuccess, "Cx is not supporting short-circuit evaluation")
	t.Run("issue-39.cx", valuetypes.CXCompilationError, "func defined with no arguments, called WITH arguments causes PANIC")
	t.Run("issue-2.cx", valuetypes.CXSuccess, "multi-dimensional arrays are not working")
	t.Run("issue-1.cx", valuetypes.CXSuccess, "multi-dimensional slices are not working")
	t.Run("issue-177/structofslicechar.cx", valuetypes.CXSuccess, "len of a struct slice literal does not depend on its strings (skycoin-cx#47)")
	t.Run("issue-177/structofsliceint32.cx", valuetypes.CXSuccess, "len of a struct slice literal (skycoin-cx#47)")
	t.Run("issue-177/structofslicestring.cx", valuetypes.CXSuccess, "len of a struct slice literal with a str field (skycoin-cx#47)")
	t.Run("issue-177/structofslicestrings.cx", valuetypes.CXSuccess, "len of a struct slice literal with another str field (skycoin-cx#47)")
	t.Run("issue-120-a.cx", valuetypes.CXCompilationError, "Invalid implicit cast when assigning the result of a math operator to a variable. (skycoin-cx#27)")
	t.Run("issue-120-b.cx", valuetypes.CXCompilationError, "Invalid implicit cast when assigning the result of a math operator to a variable. (skycoin-cx#27)")
	t.Run("issue-120-c.cx", valuetypes.CXCompilationError, "Invalid implicit cast when assigning the result of a math operator to a variable. (skycoin-cx#27)")
	t.Run("issue-131.cx", valuetypes.CXSuccess, "Panic when using arithmetic operations.")
	t.Run("test-ar-1.cx", valuetypes.CXSuccess, "Panic when using string to pointer array.")
	t.Run("issue-650.cx", valuetypes.CXSuccess, "test should not fail")

	t.RunWant("test-serialize.cx", valuetypes.CXSuccess, "serialize returns a []ui8 that deserialize accepts", "true\n7")
	t.RunWant("test-serialize-invalid.cx", valuetypes.CXRuntimeInvalidArgument, "deserialize of bytes that are not a program is a runtime error",
		"test-serialize-invalid.cx:9, CX_RUNTIME_INVALID_ARGUMENT")
	// These issues of skycoin/cx have no test file here.
	// t.RunEx("issue-309.cx", types.CXSuccess, "Serialization is not taking into account non-default stack sizes.", TestIssue, 0)
	// t.RunEx("issue-310.cx", types.CXSuccess, "Splitting a serialized program into its blockchain and transaction parts.", TestIssue, 0)
	// t.RunEx("issue-311.cx", types.CXSuccess, "`CurrentFunction` and `CurrentStruct` are causing errors in programs with more than 1 package.", TestIssue, 0)
	// t.RunEx("issue-312.cx", types.CXSuccess, "Deserialization is not setting correctly the sizes for the CallStack, HeapStartsAt and StackSize fields of the CXProgram structure.", TestIssue, 0)

	t.Run("issue-28-f32.cx", valuetypes.CXSuccess, "f32")
	t.Run("issue-28-f64.cx", valuetypes.CXSuccess, "f64")
	t.Run("issue-28-i8.cx", valuetypes.CXSuccess, "i8")
	t.Run("issue-28-i16.cx", valuetypes.CXSuccess, "i16")
	t.Run("issue-28-i32.cx", valuetypes.CXSuccess, "i32")
	t.Run("issue-28-i64.cx", valuetypes.CXSuccess, "i64")
	t.Run("issue-28-ui8.cx", valuetypes.CXSuccess, "ui8")
	t.Run("issue-28-ui16.cx", valuetypes.CXSuccess, "ui16")
	t.Run("issue-28-ui32.cx", valuetypes.CXSuccess, "ui32")
	t.Run("issue-28-ui64.cx", valuetypes.CXSuccess, "ui64")

	// // New tests added starting 05/10/2022
	t.Run("issue-struct-a.cx", valuetypes.CXSuccess, "struct printf")
	t.Run("issue-struct-b.cx", valuetypes.CXSuccess, "struct printf uninitialized")

	// Examples
	t.RunEx("./examples/bubble-sort.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("--enable=cipher ./examples/cipher.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/concatenating-strings.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/conversions.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/convert-time.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/digital-root.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/errors.cx", valuetypes.CXRuntimeError, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/even-or-odd.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/expressions.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/factorial.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/fibonacci-recursive.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/function-to-function-call.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/hello-world.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/nth-pow.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/nth-recursive-pow.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/packages.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	// t.RunEx("./examples/program.cx", types.CXSuccess, "", runner.TestExamples, 0)
	// t.RunEx("./examples/read-string.cx", types.CXSuccess, "", runner.TestExamples, 0)
	t.RunEx("./examples/regexp.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/return.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/sum-digits-integer.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/temp-converter.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/testing-global-1.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/testing-globals.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunEx("./examples/testing-packages.cx", valuetypes.CXSuccess, "", testrunner.TestExamples, 0)
	t.RunWant("./examples/testing-pointers-to-arrays-of-structs.cx", valuetypes.CXSuccess, "a pointer to a local array stays valid after a later call",
		"10\n20\n30\n40\n50\n60\n10\n20\n30\n40\n50\n60\n")
}

// runFileWriteTests runs the tests that write files in a sandbox directory it
// removes afterwards, so they leave nothing in workingDir.
func runFileWriteTests(t *testrunner.TestRunner, workingDir string) {
	sandboxDir, err := os.MkdirTemp(workingDir, "file-write-")
	if err != nil {
		fmt.Printf("file write tests: %v\n", err)
		t.TestCount++
		return
	}
	defer os.RemoveAll(sandboxDir)

	sandbox := "--enable=file --sandbox=" + filepath.Base(sandboxDir) + " "
	t.RunWant(sandbox+"test-os-write-slice-size.cx", valuetypes.CXSuccess, "os.Write*Slice writes len(s) elements of the slice type", "size 31\n", "1 2 3 -4 5 6 7 0.5 1.5\n")
	t.RunWant(sandbox+"test-os-read-write-sync-remove.cx", valuetypes.CXSuccess, "os.OpenReadWrite keeps and overwrites, os.Sync and os.Remove (skycoin-cx D19)", "first 10\n", "second 21\nthird 30\n")
}
