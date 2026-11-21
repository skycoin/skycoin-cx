package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-cxtestrunner"
	"github.com/skycoin/skycoin-cx/t0-types"
)

// runGuideSnippetTests compiles each ```cx block of LanguageGuide.md that has a func main with cx -ast.
// A block fenced ```cx compile-error must fail to compile instead.
func runGuideSnippetTests(t *runner.TestRunner, workingDir string) {
	guide, err := os.ReadFile(filepath.Join(workingDir, "..", "LanguageGuide.md"))
	if err != nil {
		fmt.Printf("guide snippets: %v\n", err)
		t.TestCount++
		return
	}

	snippetDir, err := os.MkdirTemp(workingDir, "guide-snippets-")
	if err != nil {
		fmt.Printf("guide snippets: %v\n", err)
		t.TestCount++
		return
	}
	defer os.RemoveAll(snippetDir)

	var block []string
	var fence string
	var startLine int
	for index, line := range strings.Split(string(guide), "\n") {
		line = strings.TrimRight(line, "\r")
		if fence == "" {
			if line == "```cx" || line == "```cx compile-error" {
				fence = line
				startLine = index + 2
				block = block[:0]
			}
			continue
		}
		if line != "```" {
			block = append(block, line)
			continue
		}

		source := strings.Join(block, "\n") + "\n"
		expectedCode := types.CX_SUCCESS
		if fence == "```cx compile-error" {
			expectedCode = types.CX_COMPILATION_ERROR
		}
		fence = ""
		if !strings.Contains(source, "func main") {
			continue
		}

		name := fmt.Sprintf("guide-L%04d.cx", startLine)
		if err := os.WriteFile(filepath.Join(snippetDir, name), []byte(source), 0o644); err != nil {
			fmt.Printf("guide snippets: %v\n", err)
			t.TestCount++
			continue
		}
		path := filepath.Base(snippetDir) + "/" + name
		t.Run("-ast "+path, expectedCode, fmt.Sprintf("LanguageGuide.md line %d", startLine))
	}
}
