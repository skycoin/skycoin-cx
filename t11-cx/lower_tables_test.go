package main

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	"github.com/skycoin/skycoin-cx/t4-actions"
	lower "github.com/skycoin/skycoin-cx/t4-cx-compiler-lower"
)

// TestLowerTables checks the side tables of decision D8: the extent, name
// and type of each frame variable, and the literals of the data segment.
func TestLowerTables(t *testing.T) {
	initProgram()
	compileRunSources(t, []string{"tables.cx"}, []string{`package main

type point struct {
	x i32
	y i32
}

func add(a i32, b i32) (c i32) {
	c = a + b
}

func main() {
	var x i32
	var p point
	var q *i32
	x = add(1234567, 7654321)
	p.x = x
	q = &x
	i32.print(*q + p.x)
}
`})
	image, failures := lower.Program(actions.AST)
	if len(failures) > 0 {
		t.Fatalf("does not lower: %+v", failures)
	}
	locals := func(name string) map[string]ir.Local {
		for id, funcName := range image.ImageFuncNames {
			if funcName != name {
				continue
			}
			fn := &image.ImageFuncs[id]
			byName := map[string]ir.Local{}
			previous := uint32(0)
			for _, local := range image.ImageLocals[fn.FuncLocalFirst : fn.FuncLocalFirst+fn.FuncLocalCount] {
				if local.LocalOffset < previous {
					t.Errorf("%s: locals not sorted by offset", name)
				}
				previous = local.LocalOffset
				if local.LocalOffset+local.LocalSize > fn.FuncFrameSize {
					t.Errorf("%s: local %s outside the frame", name, image.ImageLocalNames[local.LocalName])
				}
				byName[image.ImageLocalNames[local.LocalName]] = local
			}
			return byName
		}
		t.Fatalf("no function %s", name)
		return nil
	}
	check := func(function, name string, size uint32, code types.Code, typeName string) {
		local, found := locals(function)[name]
		switch {
		case !found:
			t.Errorf("%s: no local %s", function, name)
		case local.LocalSize != size || local.LocalType != code || image.ImageLocalNames[local.LocalTypeName] != typeName:
			t.Errorf("%s: local %s is %d bytes of %d %q, want %d bytes of %d %q", function, name,
				local.LocalSize, local.LocalType, image.ImageLocalNames[local.LocalTypeName], size, code, typeName)
		}
	}
	check("add", "a", 4, types.I32, "i32")
	check("add", "b", 4, types.I32, "i32")
	check("add", "c", 4, types.I32, "i32")
	check("main", "x", 4, types.I32, "i32")
	check("main", "p", 8, types.UNDEFINED, "point")
	check("main", "q", uint32(types.POINTER_SIZE), types.UNDEFINED, "*i32")

	readOnly := func(value uint32) bool {
		for _, span := range image.ImageReadOnly {
			for address := span.SpanStart; address+4 <= span.SpanEnd; address++ {
				at := image.ImageData[address-image.ImageDataStart:]
				if uint32(at[0])|uint32(at[1])<<8|uint32(at[2])<<16|uint32(at[3])<<24 == value {
					return true
				}
			}
		}
		return false
	}
	for _, literal := range []uint32{1234567, 7654321} {
		if !readOnly(literal) {
			t.Errorf("literal %d is not in a read only span", literal)
		}
	}
	for i, span := range image.ImageReadOnly {
		if span.SpanEnd <= span.SpanStart || span.SpanStart < image.ImageDataStart || span.SpanEnd > ir.ImageDataEnd(&image) {
			t.Errorf("span %d %+v outside the data segment", i, span)
		}
		if i > 0 && span.SpanStart <= image.ImageReadOnly[i-1].SpanEnd {
			t.Errorf("spans %d and %d not sorted and apart", i-1, i)
		}
	}
}
