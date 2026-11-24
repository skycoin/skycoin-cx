package types

import "testing"

// TestCodeNameBytes fails when NameBytes differs from Name or can be appended
// to in place.
func TestCodeNameBytes(t *testing.T) {
	for code := Code(0); int(code) < len(definitions); code++ {
		name := code.NameBytes()
		if string(name) != code.Name() || cap(name) != len(name) {
			t.Errorf("code %d: NameBytes %q with cap %d, want %q with no spare capacity", code, name, cap(name), code.Name())
		}
	}
}

// TestCodeNameBytesAllocs fails when NameBytes allocates.
func TestCodeNameBytesAllocs(t *testing.T) {
	if allocs := testing.AllocsPerRun(100, func() { _ = I32.NameBytes() }); allocs != 0 {
		t.Errorf("%.0f allocations, want 0", allocs)
	}
}
