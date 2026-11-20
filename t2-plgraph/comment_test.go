package graph

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
	closeFiles(t, sourceCodes)
	if err := loader.LoadCXProgram(name, sourceCodes); err != nil {
		t.Fatal(err)
	}
}

func TestGraphSkipsCommentedImport(t *testing.T) {
	loadSource(t, "GraphComment", map[string]string{
		"m.cx": "package main\nimport \"os\"\n/*\nimport \"fromcomment\"\n*/\nfunc main() {\n}\n",
	})
	got, err := GetImportGraph("GraphComment")
	if err != nil {
		t.Fatal(err)
	}
	if want := "id=0, module=main, imports=1,\nid=1, module=os, imports=,\n"; got != want {
		t.Errorf("GetImportGraph:\n got %q\nwant %q", got, want)
	}
}

func TestGraphExternalImportOnce(t *testing.T) {
	loadSource(t, "GraphExternal", map[string]string{
		"main.cx": "package main\nimport \"os\"\nimport \"b\"\nfunc main() {\n}\n",
		"b/b.cx":  "package b\nimport \"os\"\n",
	})
	got, err := GetImportGraph("GraphExternal")
	if err != nil {
		t.Fatal(err)
	}
	if want := "id=0, module=main, imports=2,1,\nid=1, module=b, imports=2,\nid=2, module=os, imports=,\n"; got != want {
		t.Errorf("GetImportGraph:\n got %q\nwant %q", got, want)
	}
}

// closeFiles closes the source files when the test ends, before its temporary
// directory is removed, which Windows refuses while a file is open.
func closeFiles(t *testing.T, files []*os.File) {
	t.Cleanup(func() {
		for _, file := range files {
			file.Close()
		}
	})
}
