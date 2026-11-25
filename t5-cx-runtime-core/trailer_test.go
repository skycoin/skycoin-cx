package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-util"
)

// A program appended with CoreTrailerAppend reads back from the file, and a
// file without a trailer reads as no program.
func TestCoreTrailerRoundTrip(t *testing.T) {
	dir := t.TempDir()
	coreBinary := []byte("core binary bytes")
	program := []byte("CXPROGRM serialized program")
	built := filepath.Join(dir, "built")
	if err := os.WriteFile(built, CoreTrailerAppend(coreBinary, program), 0o755); err != nil {
		t.Fatal(err)
	}
	got, fileError := CoreTrailerRead(built)
	if fileError != nil || string(got) != string(program) {
		t.Fatalf("read %q, %v, want %q", got, fileError, program)
	}

	plain := filepath.Join(dir, "plain")
	if err := os.WriteFile(plain, append(coreBinary, coreBinary...), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, fileError := CoreTrailerRead(plain); got != nil || fileError != nil {
		t.Fatalf("file without a trailer read as %q, %v", got, fileError)
	}
}

// A trailer whose offset, length, version or checksum does not fit is
// FileErrorCodeContentInvalid.
func TestCoreTrailerDamaged(t *testing.T) {
	dir := t.TempDir()
	good := CoreTrailerAppend([]byte("core"), []byte("program"))
	tail := len(good) - CoreTrailerSize
	cases := map[string]func(b []byte){
		"offset":   func(b []byte) { b[tail]++ },
		"length":   func(b []byte) { b[tail+8]++ },
		"checksum": func(b []byte) { b[tail+16]++ },
		"version":  func(b []byte) { b[tail+20]++ },
		"program":  func(b []byte) { b[tail-1]++ },
		"prefixed": func(b []byte) {},
	}
	for name, damage := range cases {
		b := append([]byte(nil), good...)
		damage(b)
		if name == "prefixed" {
			b = append([]byte("x"), b...)
		}
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatal(err)
		}
		got, fileError := CoreTrailerRead(path)
		if got != nil || fileError == nil || fileError.FileErrorCode != util.FileErrorCodeContentInvalid {
			t.Errorf("%s: read %q, %+v, want FileErrorCodeContentInvalid", name, got, fileError)
		}
	}
}
