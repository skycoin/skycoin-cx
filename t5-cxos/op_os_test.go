//go:build !nocxos

package cxos

import (
	"os"
	"reflect"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
)

// os.Run split the command on single spaces. See #118.
func TestSplitCommand(t *testing.T) {
	cases := []struct {
		in   string
		want []string
		ok   bool
	}{
		{"", nil, true},
		{"   ", nil, true},
		{"echo a", []string{"echo", "a"}, true},
		{"echo  a\tb", []string{"echo", "a", "b"}, true},
		{"sh -c 'echo hi'", []string{"sh", "-c", "echo hi"}, true},
		{`printf "%s|" "a b" c`, []string{"printf", "%s|", "a b", "c"}, true},
		{`echo ""`, []string{"echo", ""}, true},
		{`echo a'b c'd`, []string{"echo", "ab cd"}, true},
		{`C:\bin\tool.exe x`, []string{`C:\bin\tool.exe`, "x"}, true},
		{"echo 'a", nil, false},
	}
	for _, c := range cases {
		got, err := splitCommand(c.in)
		if (err == nil) != c.ok {
			t.Errorf("splitCommand(%q) error %v, want ok=%v", c.in, err, c.ok)
			continue
		}
		if c.ok && !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitCommand(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// os.Sync on vm makes no Go allocation, for an open file and for a handle
// that is not open.
func TestLibraryFileSyncVmAllocs(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "sync")
	if err != nil {
		t.Fatal(err)
	}
	handle := getFileHandle(file)
	defer fileClose(handle)
	memory := make([]byte, 8)
	for _, h := range []int32{handle, -1, 1000} {
		types.Write_i32(memory, 0, h)
		allocs := testing.AllocsPerRun(100, func() { LibraryFileSyncVm(memory, 0) })
		if allocs != 0 {
			t.Errorf("handle %d: %v allocations", h, allocs)
		}
		if synced := types.Read_bool(memory, 4); synced != (h == handle) {
			t.Errorf("handle %d: synced %v", h, synced)
		}
	}
}
