package tree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-util"
	"github.com/skycoin/skycoin-cx/t1-plloader"
)

// loadSource stores the files, given as path and content, as one program.
func loadSource(t *testing.T, name string, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	for path, content := range files {
		path = filepath.Join(dir, "src", path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	_, sourceCodes, _, err := util.ParseArgsForCX([]string{filepath.Join(dir, "src")}, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := loader.LoadCXProgram(name, sourceCodes); err != nil {
		t.Fatal(err)
	}
}

func TestTreeSkipsCommentedImport(t *testing.T) {
	loadSource(t, "TreeComment", map[string]string{
		"m.cx": "package main\nimport \"os\"\n/*\nimport \"fromcomment\"\n*/\nfunc main() {\n}\n",
	})
	got, err := GetImportTree("TreeComment")
	if err != nil {
		t.Fatal(err)
	}
	if want := "main\n`--os\n"; got != want {
		t.Errorf("GetImportTree:\n got %q\nwant %q", got, want)
	}
}
