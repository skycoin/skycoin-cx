//go:build !noregexp

package regexp

import (
	"fmt"
	"os"
	"regexp"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes" // thin: only StrRead, STR, Write_str; no depth gain
	"github.com/skycoin/skycoin-cx/t3-cx-core-ast"
)

// VIOLATION: state outside State. Owner: t4-cx-library-regexp; moves to RegexpMng in phase 1.
var regexps map[string]*regexp.Regexp = make(map[string]*regexp.Regexp, 0)

// expOffset returns the offset of the `exp` field of the CX `regexp.Regexp`
// held by value. The field holds the expression, which keys `regexps`. exp is
// the only field PackageRegister gives Regexp, so it starts the struct.
func expOffset(value ast.CXValue) valuetypes.Pointer {
	return value.Offset
}

// regexpCompile is a helper function for `OpRegexpMustCompile` and
// `OpRegexpCompile`. `regexpCompile` compiles a `regexp.Regexp` structure
// and adds it to global `regexps`. It also writes CX structure `regexp.Regexp`.
// GCREACH: allocates the expression str, which adds to the GC pressure,
// see docs/GarbageCollector.md.
// VIOLATION STRHOT-142: B, hands a path or text to Go or the OS, #412.
// VIOLATION VMALLOC-017: compiles and matches with the Go regexp package, #407.
func regexpCompile(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) error {
	exp := inputs[0].StrGet(prgrm)

	// The CX Regexp only holds the expression. The same expression in two
	// parts of a program shares one compiled instance.
	ast.StrAllocateAt(prgrm, expOffset(outputs[0]), exp)

	r, err := regexp.Compile(exp)
	if err != nil {
		return err
	}
	regexps[exp] = r
	return nil
}

// OpRegexpMustCompile is a wrapper for golang's `regexp`'s `MustCompile`.
func OpRegexpMustCompile(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	err := regexpCompile(prgrm, inputs, outputs)

	if err != nil {
		regexpMustCompileFailureReport(err)
	}
}

// regexpMustCompileFailureReport writes the error of a failed MustCompile to
// standard error and stops the program with CXRuntimeError.
// STRHOT REPORTER: builds the text of the Go regexp error, #412.
func regexpMustCompileFailureReport(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	panic(valuetypes.CXRuntimeError)
}

// OpRegexpCompile is a wrapper for golang's `regexp`'s `MustCompile`.
// VIOLATION STRHOT-143: B, hands a path or text to Go or the OS, #412.
func OpRegexpCompile(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	// We're only interested in `out1`, which represents the
	// returned error.
	err := regexpCompile(prgrm, inputs, outputs)

	// Writing error message to `out1`.
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	outputs[1].StrSet(prgrm, errStr)
}

// OpRegexpFind is a wrapper for golang's `regexp`'s `Find`.
// VIOLATION STRHOT-144: B, hands a path or text to Go or the OS, #412.
// VIOLATION VMALLOC-017: compiles and matches with the Go regexp package, #407.
func OpRegexpFind(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	exp := valuetypes.StrRead(prgrm.Memory, expOffset(inputs[0]))
	r, ok := regexps[exp]
	if !ok {
		fmt.Fprintln(os.Stderr, "regexp: Find on a Regexp that did not compile: "+exp)
		panic(valuetypes.CXRuntimeError)
	}

	outputs[0].StrSet(prgrm, string(r.Find([]byte(inputs[1].StrGet(prgrm)))))
}
