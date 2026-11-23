package ast

import (
	"slices"
	"testing"

	stringpool "github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
)

func TestNativeNameIndex(t *testing.T) {
	var index NativeNameIndex
	index.NativeNameIndexAdd(1, "len")
	index.NativeNameIndexAdd(2, "os.ReadFile")
	index.NativeNameIndexAdd(3, "os")
	index.NativeNameIndexAdd(4, "osx.Open")
	index.NativeNameIndexAdd(5, "os.Point.Scale")
	index.NativeNameIndexAdd(6, "os.Exit")
	index.NativeNameIndexAdd(7, "os.ReadFile")
	if got := index.NativeNameIndexLen(); got != 6 {
		t.Fatalf("Len = %d, want 6, a name added twice is kept once", got)
	}
	if entry, found := index.NativeNameIndexText("os.ReadFile"); !found || entry.NativeNameEntryCode != 7 {
		t.Errorf("os.ReadFile = %+v, %v, want the later code 7", entry, found)
	}
	generation := index.NativeNameIndexGeneration
	index.NativeNameIndexAdd(7, "os.ReadFile")
	if index.NativeNameIndexGeneration != generation {
		t.Error("registering a name again with its code changed the generation")
	}

	var members []string
	for _, entry := range index.NativeNameIndexPackage("os") {
		members = append(members, entry.NativeNameEntryText)
	}
	if want := []string{"os.Exit", "os.Point.Scale", "os.ReadFile"}; !slices.Equal(members, want) {
		t.Errorf("package os = %q, want %q", members, want)
	}
	if got := index.NativeNameIndexCodeText(5); got != "os.Point.Scale" {
		t.Errorf("CodeText(5) = %q", got)
	}
	if got := index.NativeNameIndexCodeText(99); got != "" {
		t.Errorf("CodeText(99) = %q, want empty", got)
	}

	prgrm := &CXProgram{CxIrState: &CxIrState{}}
	prgrm.CxIrState.StringPool.StringPoolInit(16, 64)
	intern := prgrm.CXProgramNameIntern
	none := stringpool.StringGidNil
	cases := []struct {
		pkg, receiver, name stringpool.StringGid
		code                int
		found               bool
	}{
		{none, none, intern("len"), 1, true},
		{intern("os"), none, intern("ReadFile"), 7, true},
		{none, none, intern("os.ReadFile"), 7, true},
		{intern("os"), intern("Point"), intern("Scale"), 5, true},
		{intern("os"), none, none, 3, true},
		{intern("os"), none, intern("Read"), 0, false},
		{intern("o"), none, intern("s"), 0, false},
		{intern("os"), none, intern("ReadFileX"), 0, false},
		{none, none, intern("zzz"), 0, false},
	}
	for round := 0; round < 2; round++ {
		for _, c := range cases {
			entry, found := prgrm.CXProgramNativeFind(&index, c.pkg, c.receiver, c.name)
			if found != c.found || entry.NativeNameEntryCode != c.code {
				t.Errorf("round %d: find %q %q %q = %d, %v, want %d, %v", round, prgrm.CXProgramNameText(c.pkg), prgrm.CXProgramNameText(c.receiver), prgrm.CXProgramNameText(c.name), entry.NativeNameEntryCode, found, c.code, c.found)
			}
		}
	}

	// A name added after a program cached a miss is found.
	zzz := intern("zzz")
	index.NativeNameIndexAdd(8, "zzz")
	if entry, found := prgrm.CXProgramNativeFind(&index, none, none, zzz); !found || entry.NativeNameEntryCode != 8 {
		t.Errorf("zzz after Add = %+v, %v, want code 8", entry, found)
	}
	index.NativeNameIndexReset()
	if _, found := prgrm.CXProgramNativeFind(&index, none, none, zzz); found {
		t.Error("zzz found after Reset")
	}
}
