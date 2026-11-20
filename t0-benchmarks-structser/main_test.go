package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// The benchmark used to write files with os.FileMode(644), which is 01204.
// See #155.
func TestWriteFileMode(t *testing.T) {
	name := filepath.Join(t.TempDir(), "serial")
	if err := writeFile(name, []byte("x")); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSticky != 0 || fi.Mode().Perm()&0400 == 0 {
		t.Fatalf("mode %v, want 0644 before umask", fi.Mode())
	}
}

// The structs had only unexported fields, which the encoder skips, so the
// benchmark serialized and wrote nothing. See #178.
func TestStructsSerializeToBytes(t *testing.T) {
	for name, v := range map[string]interface{}{"Point": Point{}, "Arrays": Arrays{}, "Structs": Structs{}} {
		if n := len(encoder.Serialize(v)); n == 0 {
			t.Errorf("%s serialized to 0 bytes", name)
		}
	}
}
