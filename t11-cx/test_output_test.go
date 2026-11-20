package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A failing test() printed the raw bytes of both values before its message.
// See #239.
func TestFailingTestPrintsOnlyItsMessage(t *testing.T) {
	file := filepath.Join(t.TempDir(), "t.cx")
	src := "package main\n\nfunc main() {\n\ttest(1, 2, \"one is not two\")\n\ttest(\"a\", \"bc\", \"\")\n\ttest(5, 5, \"equal\")\n\ti32.print(3)\n}\n"
	if err := os.WriteFile(file, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], file)
	cmd.Env = append(os.Environ(), "CX_TEST_RUN_MAIN=1")
	out, _ := cmd.Output()
	lines := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	want := []string{
		"result was not equal to the expected value; one is not two",
		"result was not equal to the expected value",
		"3",
	}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines of stdout, want %d:\n%s", len(lines), len(want), out)
	}
	for i, line := range lines {
		if !strings.HasSuffix(line, want[i]) {
			t.Errorf("line %d = %q, want it to end with %q", i+1, line, want[i])
		}
	}
}
