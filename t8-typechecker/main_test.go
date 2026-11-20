package type_checker_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-plbolt"
)

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
	if err := bolt.Init(filepath.Join(dir, "program_list.db")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer bolt.Close()
	return m.Run()
}
