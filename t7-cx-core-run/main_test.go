package main

import (
	"os/exec"
	"strings"
	"testing"
)

// coreImportsAllowed are the packages of this module the core may import.
// Every other import must be of the Go standard library, so no parser, no
// file system opcodes and no outside module can enter the core.
var coreImportsAllowed = map[string]bool{
	"t0-cx-core-assert":     true,
	"t0-packages":           true,
	"t0-types":              true,
	"t0-util":               true,
	"t1-constants":          true,
	"t1-cx-core-objmodel":   true,
	"t2-cx-core-stringpool": true,
	"t2-cx-runtime-gc":      true,
	"t2-cx-runtime-ir":      true,
	"t2-globals":            true,
	"t3-ast":                true,
	"t3-cx-runtime-vm":      true,
	"t4-cx-compiler-lower":  true,
	"t4-execute":            true,
	"t4-opcodes":            true,
	"t5-cx-runtime-printer": true,
	"t5-cxos-core":          true,
	"t5-regexp":             true,
	"t5-tcp":                true,
	"t6-cx-runtime-core":    true,
	"t7-cx-core-run":        true,
}

// TestCoreImports fails when the core's transitive imports, with the default
// and the ptr64 tags, include a package outside the Go standard library and
// coreImportsAllowed.
func TestCoreImports(t *testing.T) {
	const module = "github.com/skycoin/skycoin-cx/"
	for _, tags := range []string{"", "ptr64"} {
		out, err := exec.Command("go", "list", "-deps", "-tags", tags, "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", ".").CombinedOutput()
		if err != nil {
			t.Fatalf("go list -tags %q: %v\n%s", tags, err, out)
		}
		for _, path := range strings.Fields(string(out)) {
			if name, found := strings.CutPrefix(path, module); !found || !coreImportsAllowed[name] {
				t.Errorf("tags %q: the core imports %s, which is outside the standard library and the allowed packages", tags, path)
			}
		}
	}
}
