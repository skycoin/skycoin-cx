package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// sandboxCxQuote quotes s as a CX string literal. CX takes octal but not hex
// escapes, so NUL is written \000.
func sandboxCxQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\x00", `\000`)
	return `"` + s + `"`
}

// Program paths stay inside the --sandbox directory for every file opcode in
// sys/os, and for StartCPUProfile. See #433.
func TestProgramFileOpcodesStayInSandbox(t *testing.T) {
	dir := t.TempDir()
	sandbox := filepath.Join(dir, "sandbox")
	outside := filepath.Join(dir, "outside")
	for _, d := range []string{sandbox, outside} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(outside, "secret.txt"): "secret",
		filepath.Join(sandbox, "in.txt"):     "hello",
	}
	for name, content := range files {
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	rejected := []string{"../outside/secret.txt", "a/../../outside/secret.txt", "/etc/passwd", `C:\x`, `\\server\x`, "in\x00.txt", "../outside/new.txt"}
	symlinkErr := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(sandbox, "link.txt"))
	if symlinkErr == nil {
		symlinkErr = os.Symlink(outside, filepath.Join(sandbox, "linkdir"))
	}
	if symlinkErr == nil {
		rejected = append(rejected, "link.txt", "linkdir/secret.txt", "linkdir/new.txt")
	} else {
		t.Logf("symlink cases skipped: %v", symlinkErr)
	}

	var src strings.Builder
	src.WriteString("package main\n\nimport \"sys/os\"\n\nfunc main() {\n\tvar text str\n\tvar ok bool\n\tvar h i32\n")
	for _, path := range rejected {
		q := sandboxCxQuote(path)
		src.WriteString("\ttest(os.Open(" + q + "), -1, " + sandboxCxQuote("os.Open "+path) + ")\n")
		src.WriteString("\ttest(os.Create(" + q + "), -1, " + sandboxCxQuote("os.Create "+path) + ")\n")
		src.WriteString("\ttext, ok = os.ReadAllText(" + q + ")\n\ttest(ok, false, " + sandboxCxQuote("os.ReadAllText "+path) + ")\n")
		src.WriteString("\tStartCPUProfile(" + q + ", 100)\n\tStopCPUProfile(" + q + ")\n")
	}
	src.WriteString(`	text, ok = os.ReadAllText("in.txt")
	test(ok, true, "os.ReadAllText in.txt")
	h = os.Open("./in.txt")
	test(h >= 0, true, "os.Open in.txt")
	test(os.Close(h), true, "os.Close in.txt")
	h = os.Create("made.txt")
	test(h >= 0, true, "os.Create made.txt")
	test(os.Close(h), true, "os.Close made.txt")
	StartCPUProfile("cpu.pprof", 100)
	StopCPUProfile("cpu.pprof")
}
`)
	program := filepath.Join(dir, "p.cx")
	if err := os.WriteFile(program, []byte(src.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "--enable=file", "--sandbox", sandbox, program)
	cmd.Env = append(os.Environ(), "CX_TEST_RUN_MAIN=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cx: %v\n%s", err, out)
	}

	for _, name := range []string{"made.txt", "cpu.pprof"} {
		if _, err := os.Stat(filepath.Join(sandbox, name)); err != nil {
			t.Errorf("%s not created in the sandbox: %v", name, err)
		}
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("outside directory holds %d entries, want only secret.txt", len(entries))
	}
	if content, err := os.ReadFile(filepath.Join(outside, "secret.txt")); err != nil || string(content) != "secret" {
		t.Errorf("outside secret.txt is %q, %v", content, err)
	}
}
