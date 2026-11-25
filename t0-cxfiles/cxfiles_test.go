package cxfiles

import (
	"os"
	"testing"
)

// FileHandleSync makes no Go allocation, for an open file and for a handle
// that is not open, as the vm runs it.
func TestFileHandleSyncAllocs(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "sync")
	if err != nil {
		t.Fatal(err)
	}
	handle := FileHandleTake(file)
	defer FileHandleClose(handle)
	for _, h := range []int32{handle, -1, 1000} {
		allocs := testing.AllocsPerRun(100, func() { FileHandleSync(h) })
		if allocs != 0 {
			t.Errorf("handle %d: %v allocations", h, allocs)
		}
		if synced := FileHandleSync(h); synced != (h == handle) {
			t.Errorf("handle %d: synced %v", h, synced)
		}
	}
}
