package constants

import "testing"

// TestLibraryTablesConsistent fails when a library or function number breaks
// the numbering rules in libraries.go.
func TestLibraryTablesConsistent(t *testing.T) {
	names := map[string]bool{}
	for i, library := range LibraryTable {
		if library.LibraryNumber != i+1 || library.LibraryNumber >= LibraryCount || library.LibraryName == "" || names[library.LibraryName] {
			t.Errorf("library %+v: numbers must start at 1 and be dense below LibraryCount, names unique", library)
		}
		names[library.LibraryName] = true
		if !LanguageVersionValid(library.LibraryVersionAdded) || library.LibraryVersionAdded > LanguageVersionCurrent {
			t.Errorf("library %+v: bad version added", library)
		}
	}
	if len(LibraryTable) != LibraryCount-1 {
		t.Errorf("%d libraries, LibraryCount is %d", len(LibraryTable), LibraryCount)
	}

	live := map[string]bool{}
	previousLibrary, previousNumber := 0, 0
	for _, row := range LibraryFunctionLifecycleTable {
		if row.LibraryNumber != previousLibrary {
			if row.LibraryNumber < previousLibrary {
				t.Errorf("row %+v: libraries out of order", row)
			}
			previousLibrary, previousNumber = row.LibraryNumber, 0
		}
		if LibraryNameOf(row.LibraryNumber) == "" || row.Number <= 0 || row.Number < previousNumber || row.Name == "" {
			t.Errorf("row %+v: unknown library, or numbers not positive and in order", row)
		}
		previousNumber = row.Number
		if !LanguageVersionValid(row.VersionAdded) || row.VersionAdded > LanguageVersionCurrent {
			t.Errorf("row %+v: bad version added", row)
		}
		if row.VersionDeprecated == 0 {
			if row.VersionReassignable != 0 || live[row.Name] {
				t.Errorf("row %+v: live row with a reassignable version, or a name live twice", row)
			}
			live[row.Name] = true
			if opcode, found := OpcodeLifecycleLiveByName(row.Name); found {
				t.Errorf("row %+v: name is also the live opcode %d", row, opcode.Number)
			}
			continue
		}
		if row.VersionDeprecated < row.VersionAdded || row.VersionReassignable != LanguageVersionReassignable(row.VersionDeprecated) {
			t.Errorf("row %+v: bad version deprecated or reassignable", row)
		}
	}
}

// TestLibraryFunctionLiveNameBytes fails when the bytes of a live function
// name differ from LibraryFunctionLiveByNumber, or allocate.
func TestLibraryFunctionLiveNameBytes(t *testing.T) {
	for _, row := range LibraryFunctionLifecycleTable {
		live, found := LibraryFunctionLiveByNumber(row.LibraryNumber, row.Number)
		name := LibraryFunctionLiveNameBytes(row.LibraryNumber, row.Number)
		if !found {
			live.Name = ""
		}
		if string(name) != live.Name || cap(name) != len(name) {
			t.Errorf("library %d function %d: %q with cap %d, want %q with no spare capacity", row.LibraryNumber, row.Number, name, cap(name), live.Name)
		}
	}
	if name := LibraryFunctionLiveNameBytes(LibraryCount, 1); name != nil {
		t.Errorf("unknown library: %q, want nil", name)
	}
	row := LibraryFunctionLifecycleTable[0]
	if allocs := testing.AllocsPerRun(100, func() { _ = LibraryFunctionLiveNameBytes(row.LibraryNumber, row.Number) }); allocs != 0 {
		t.Errorf("%.0f allocations, want 0", allocs)
	}
}
