package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// StartCPUProfile wrote <argv0>_<name>_cpu.pprof next to the executable
// instead of the path it was given. See #118.
func TestStartCPUProfileWritesTheGivenPath(t *testing.T) {
	dir := t.TempDir()
	prof := filepath.Join(dir, "run.pprof")
	file := filepath.Join(dir, "p.cx")
	src := "package main\n\nimport \"os\"\n\nfunc main() {\n\tStartCPUProfile(" + strconv.Quote(prof) + ", 100)\n\tvar i i32\n\tfor i = 0; i < 100000; i++ {\n\t}\n\tStopCPUProfile(" + strconv.Quote(prof) + ")\n}\n"
	if err := os.WriteFile(file, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], file)
	cmd.Env = append(os.Environ(), "CX_TEST_RUN_MAIN=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cx: %v\n%s", err, out)
	}
	if info, err := os.Stat(prof); err != nil {
		t.Errorf("no profile at the given path: %v", err)
	} else if info.Size() == 0 {
		t.Error("the profile at the given path is empty")
	}
	stray := os.Args[0] + "_" + prof + "_cpu.pprof"
	if _, err := os.Stat(stray); err == nil {
		os.Remove(stray)
		t.Errorf("profile written to %s", stray)
	}
}
