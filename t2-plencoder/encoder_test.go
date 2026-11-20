package encoder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-plbolt"
	"github.com/skycoin/skycoin-cx/t0-util"
	"github.com/skycoin/skycoin-cx/t1-plloader"
)

func TestSavePackage(t *testing.T) {
	_, sourceCodes, _, perr := util.ParseArgsForCX([]string{"test_tree"}, true)
	if perr != nil {
		t.Fatal(perr)
	}
	if err := loader.LoadCXProgram("Test", sourceCodes); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "out") + "/"
	if err := SavePackagesToDisk("Test", dir); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "src", "testimport2", "testimport2file.cx"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("test_tree", "src", "testimport2", "testimport2file.cx"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("saved file differs from the source:\n got %q\nwant %q", got, want)
	}
}

func TestSavePackageRejectsUnsafeNames(t *testing.T) {
	put := func(key string, v interface{ MarshalBinary() ([]byte, error) }) {
		b, err := v.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if err := bolt.Add(key, b); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"../escape.cx", "..", "a/b.cx", `a\b.cx`, ""} {
		put("unsafe-file", loader.File{FileName: name, Content: []byte("package main\n")})
		put("unsafe-pkg", loader.Package{PackageName: "main", Files: []string{"unsafe-file"}})
		put("UnsafeProgram", loader.PackageList{Packages: []string{"unsafe-pkg"}})
		dir := filepath.Join(t.TempDir(), "out")
		if err := SavePackagesToDisk("UnsafeProgram", dir); err == nil {
			t.Errorf("file name %q: want an error, got nil", name)
		}
	}
	put("unsafe-pkg", loader.Package{PackageName: "../up", Files: nil})
	if err := SavePackagesToDisk("UnsafeProgram", filepath.Join(t.TempDir(), "out")); err == nil {
		t.Error("package name ../up: want an error, got nil")
	}
}
