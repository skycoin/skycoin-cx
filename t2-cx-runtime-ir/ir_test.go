package ir

import (
	"testing"
	"unsafe"
)

func TestInstrSize(t *testing.T) {
	if size := unsafe.Sizeof(Instr{}); size != InstrSize {
		t.Fatalf("Instr is %d bytes, want %d", size, InstrSize)
	}
}

func TestLineAt(t *testing.T) {
	image := Image{
		ImageFuncs: []Func{
			{FuncCode: make([]Instr, 1)},
			{FuncCode: make([]Instr, 6), FuncLineFirst: 0, FuncLineCount: 2},
		},
		ImageLines: []Line{{LinePc: 1, LineNumber: 10}, {LinePc: 4, LineNumber: 12}},
	}
	cases := []struct {
		function, pc uint32
		line         uint32
		found        bool
	}{
		{0, 0, 0, false},
		{1, 0, 0, false},
		{1, 1, 10, true},
		{1, 3, 10, true},
		{1, 4, 12, true},
		{1, 5, 12, true},
	}
	for _, c := range cases {
		line, found := LineAt(&image, c.function, c.pc)
		if found != c.found || line.LineNumber != c.line {
			t.Errorf("LineAt(%d, %d) = %d %v, want %d %v", c.function, c.pc, line.LineNumber, found, c.line, c.found)
		}
	}
}
