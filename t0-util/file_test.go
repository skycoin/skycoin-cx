package util

import (
	"os"
	"path/filepath"
	"testing"
)

// Each helper joined the working dir differently. See #89.
func TestFileHelpersUseWorkingDir(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir("wd", 0o755); err != nil {
		t.Fatal(err)
	}
	CXSetWorkingDir("wd")
	defer CXSetWorkingDir("")

	f, err := CXCreateFile("a.txt")
	if err != nil {
		t.Fatalf("CXCreateFile: %v", err)
	}
	if _, err := f.WriteString("hi"); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := os.Stat(filepath.Join("wd", "a.txt")); err != nil {
		t.Errorf("CXCreateFile did not create wd/a.txt: %v", err)
	}

	f, err = CXOpenFile("a.txt")
	if err != nil {
		t.Errorf("CXOpenFile: %v", err)
	} else {
		f.Close()
	}
	if _, err := CXStatFile("a.txt"); err != nil {
		t.Errorf("CXStatFile: %v", err)
	}
	if b, err := CXReadFile("a.txt"); err != nil || string(b) != "hi" {
		t.Errorf("CXReadFile: %q, %v", b, err)
	}
	if err := CXMkdirAll("d", 0o755); err != nil {
		t.Errorf("CXMkdirAll: %v", err)
	}
	if _, err := os.Stat(filepath.Join("wd", "d")); err != nil {
		t.Errorf("CXMkdirAll did not create wd/d: %v", err)
	}
	if err := CXRemoveFile("a.txt"); err != nil {
		t.Errorf("CXRemoveFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join("wd", "a.txt")); !os.IsNotExist(err) {
		t.Errorf("CXRemoveFile left wd/a.txt: %v", err)
	}
}
