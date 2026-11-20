package bolt

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var testDBPath string

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	dir, err := os.MkdirTemp("", "plbolt")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(dir)
	testDBPath = filepath.Join(dir, "program_list.db")
	if err := Init(testDBPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer Close()
	return m.Run()
}
