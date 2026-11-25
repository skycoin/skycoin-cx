package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// make release-archives stamps the tag into cx with -ldflags -X, so a cx
// built that way prints it.
func TestVersionLdflags(t *testing.T) {
	cx := filepath.Join(t.TempDir(), "cx")
	if runtime.GOOS == "windows" {
		cx += ".exe"
	}
	build := exec.Command("go", "build", "-ldflags", "-X main.VERSION=9.9.9-test1", "-o", cx, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	out, err := exec.Command(cx, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("cx --version: %v\n%s", err, out)
	}
	want := "CX version 9.9.9-test1 " + runtime.GOOS + "/" + runtime.GOARCH
	if got := strings.TrimSpace(string(out)); got != want {
		t.Fatalf("cx --version printed %q, want %q", got, want)
	}
}
