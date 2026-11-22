package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-execute"
)

// serializeDeclarationSource declares a local variable, which Deserialize
// used to crash on. See #292.
const serializeDeclarationSource = `package main
func main() {
	var x i32 = 7
	i32.print(x)
}
`

// serializeCompile compiles source alone and returns the program.
func serializeCompile(t *testing.T, name string, source string) *ast.CXProgram {
	t.Helper()
	initProgram()
	compileRunSources(t, []string{name}, []string{source})
	return actions.AST
}

func TestSerializeDeclaration(t *testing.T) {
	prgrm := serializeCompile(t, "declaration.cx", serializeDeclarationSource)
	for _, includeMemory := range []bool{false, true} {
		b, err := ast.SerializeCXProgram(prgrm, includeMemory, nil)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ast.Deserialize(b, nil)
		if err != nil {
			t.Fatalf("includeMemory %v: %v", includeMemory, err)
		}
		if !includeMemory {
			continue
		}
		actions.AST = got
		out := lowerCompareCaptureStdout(t, func() {
			if err := execute.RunCompiled(got, 0, nil); err != nil {
				t.Fatal(err)
			}
		})
		if string(out) != "7\n" {
			t.Fatalf("the deserialized program printed %q, want %q", out, "7\n")
		}
	}
}

// The section sizes add up to the serialized program and none is empty. See
// #293.
func TestSerializeDebugInfoSections(t *testing.T) {
	prgrm := serializeCompile(t, "declaration.cx", serializeDeclarationSource)
	b, err := ast.SerializeCXProgram(prgrm, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	info, err := ast.SerializeDebugInfo(prgrm, true, false)
	if err != nil {
		t.Fatal(err)
	}
	sections := []int{info.Header, info.Program, info.Strings, info.Packages, info.Structs, info.Functions,
		info.Expressions, info.Arguments, info.TypeSignatures, info.Memory}
	sum := 0
	for i, size := range sections {
		if size == 0 {
			t.Errorf("section %d is empty in %+v", i, info)
		}
		sum += size
	}
	if sum != len(b) {
		t.Fatalf("sections add up to %d, the program is %d bytes: %+v", sum, len(b), info)
	}
}

// A program from a build with another pointer size, format or language
// version is refused with an error that names it. See #295.
func TestSerializeHeaderMismatch(t *testing.T) {
	prgrm := serializeCompile(t, "declaration.cx", serializeDeclarationSource)
	b, err := ast.SerializeCXProgram(prgrm, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		offset int
		value  uint32
		want   string
	}{
		{8, 1, "format 1"},
		{12, uint32(16 - types.POINTER_SIZE), "byte pointers"},
		{16, 1, "language version"},
	}
	for _, tc := range cases {
		patched := append([]byte(nil), b...)
		binary.LittleEndian.PutUint32(patched[tc.offset:], tc.value)
		_, err := ast.Deserialize(patched, nil)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("header field at %d set to %d: got %v, want an error with %q", tc.offset, tc.value, err, tc.want)
		}
	}
	if _, err := ast.Deserialize([]byte("not a program at all"), nil); err == nil {
		t.Error("want an error for bytes without the magic")
	}
}

// Truncations of a serialized program are errors, and no flipped byte makes
// Deserialize panic. Each Deserialize sets up every manager, so the test
// samples about 300 offsets.
func TestSerializeMalformed(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "tests", "test-escape-gc.cx"))
	if err != nil {
		t.Fatal(err)
	}
	prgrm := serializeCompile(t, "test-escape-gc.cx", string(source))
	b, err := ast.SerializeCXProgram(prgrm, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	stride := max(1, len(b)/300)
	for length := 0; length < len(b); length += stride {
		if _, err := ast.Deserialize(b[:length], nil); err == nil {
			t.Fatalf("Deserialize of the first %d of %d bytes gave no error", length, len(b))
		}
	}
	flipped := append([]byte(nil), b...)
	for i := 24; i < len(flipped); i += stride {
		flipped[i] ^= 0xa5
		ast.Deserialize(flipped, nil)
		flipped[i] = b[i]
	}
	if !bytes.Equal(flipped, b) {
		t.Fatal("the bytes were not restored")
	}
}
