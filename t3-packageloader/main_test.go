package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("PACKAGELOADER_RUN_MAIN") == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// packageloader runs main in a child process with args and returns its
// combined output and whether it exited 0.
func packageloader(t *testing.T, args ...string) (string, bool) {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), "PACKAGELOADER_RUN_MAIN=1")
	out, err := cmd.CombinedOutput()
	return string(out), err == nil
}

func TestLoadTreeGraphSave(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "p.db")
	if out, ok := packageloader(t, "-db", db, "-load", "-name", "P", "-path", "../t2-pltree/test_tree"); !ok {
		t.Fatalf("-load: %s", out)
	}
	out, ok := packageloader(t, "-db", db, "-tree", "-name", "P")
	if !ok || !strings.HasPrefix(out, "main\n") || !strings.Contains(out, "testimport1") {
		t.Errorf("-tree: ok %v, output %q", ok, out)
	}
	out, ok = packageloader(t, "-db", db, "-graph", "-name", "P")
	if !ok || !strings.HasPrefix(out, "id=0, module=main, ") {
		t.Errorf("-graph: ok %v, output %q", ok, out)
	}
	saved := filepath.Join(dir, "saved")
	if out, ok := packageloader(t, "-db", db, "-save", "-name", "P", "-path", saved); !ok {
		t.Fatalf("-save: %s", out)
	}
	if _, err := os.Stat(filepath.Join(saved, "src", "testmain.cx")); err != nil {
		t.Errorf("-save did not write the main file: %v", err)
	}
}

func TestStrayArguments(t *testing.T) {
	db := filepath.Join(t.TempDir(), "p.db")
	if out, ok := packageloader(t, "-db", db, "-load", "-name", "P", "-path", "../t2-pltree/test_tree"); !ok {
		t.Fatalf("-load: %s", out)
	}
	for _, args := range [][]string{
		{"-db", db, "-tree", "-name", "P", "stray"},
		{"-db", db, "-graph", "-name", "P", "a", "b"},
	} {
		if out, ok := packageloader(t, args...); ok {
			t.Errorf("%q: want an error for the stray arguments, got exit 0 and %q", args, out)
		}
	}
}
