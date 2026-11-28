package main

import (
	"os/exec"
	"strings"
	"testing"
)

// coreImportsAllowed are the packages of this module the core without a
// system library may import. Every other import must be of the Go standard
// library, so no parser, no system library and no outside module can enter
// the core.
var coreImportsAllowed = map[string]bool{
	"t0-cx-core-assert":             true,
	"t0-cx-core-floatparse":         true,
	"t0-cx-runtime-filelog":         true,
	"t0-cx-core-syspackages":        true,
	"t0-cx-core-valuetypes":         true,
	"t0-cx-library-terminal":        true,
	"t0-cx-runtime-output":          true,
	"t1-cx-library-cipherstate":     true,
	"t1-cx-core-opcodetable":        true,
	"t1-cx-core-objmodel":           true,
	"t1-cx-runtime-sandbox":         true,
	"t2-cx-library-reflecttable":    true,
	"t2-cx-core-stringpool":         true,
	"t2-cx-runtime-gc":              true,
	"t2-cx-runtime-ir":              true,
	"t2-cx-core-environment":        true,
	"t3-cx-core-ast":                true,
	"t3-cx-runtime-vm":              true,
	"t4-cx-compiler-lower":          true,
	"t4-cx-runtime-interpreter":     true,
	"t4-cx-runtime-opcodes":         true,
	"t4-cx-runtime-printer":         true,
	"t4-cx-library-osopcodes":       true,
	"t4-cx-library-reflectopcodes":  true,
	"t4-cx-library-regexp":          true,
	"t4-cx-library-terminalopcodes": true,
	"t5-cx-runtime-core":            true,
	"t6-cx-command-core":            true,
}

// coreStandardDenied are packages of the Go standard library the core must not
// import, as they reach files, processes or the network. A system library
// brings them in.
var coreStandardDenied = map[string]bool{"net": true, "os/exec": true, "net/http": true, "net/rpc": true, "crypto/tls": true}

// coreLibraryImports are the packages of this module each system library
// adds to the core, by its build tag.
var coreLibraryImports = map[string][]string{
	"cxcorelibraryfile":    {"t0-cx-library-filetable", "t1-cx-runtime-profiling", "t5-cx-library-fileopcodes"},
	"cxcorelibrarynet":     {"t0-cx-core-textsections", "t1-cx-library-networkbatch", "t4-cx-library-netopcodes"},
	"cxcorelibraryprocess": {"t0-cx-library-filetable", "t1-cx-runtime-profiling", "t5-cx-library-fileopcodes"},
}

// TestCoreImports fails when the transitive imports of the core, with the
// default and the ptr64 tags, include a package outside the Go standard
// library and coreImportsAllowed, or a package of coreStandardDenied. With the
// tag of a system library the core may import its coreLibraryImports and
// the standard library it needs.
func TestCoreImports(t *testing.T) {
	const module = "github.com/skycoin/skycoin-cx/"
	for _, tags := range []string{"", "ptr64", "cxcorelibraryfile", "cxcorelibrarynet", "cxcorelibraryprocess ptr64", "cxcorelibraryfile cxcorelibrarynet cxcorelibraryprocess"} {
		allowed := map[string]bool{}
		library := false
		for _, tag := range strings.Fields(tags) {
			for _, name := range coreLibraryImports[tag] {
				allowed[name], library = true, true
			}
		}
		out, err := exec.Command("go", "list", "-deps", "-tags", "cxcore "+tags, "-f", "{{.Standard}} {{.ImportPath}}", ".").CombinedOutput()
		if err != nil {
			t.Fatalf("go list -tags %q: %v\n%s", tags, err, out)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			standard, path, _ := strings.Cut(line, " ")
			if standard == "true" {
				if coreStandardDenied[path] && !library {
					t.Errorf("tags %q: the core imports %s, which only a system library may bring in", tags, path)
				}
				continue
			}
			if name, found := strings.CutPrefix(path, module); !found || !coreImportsAllowed[name] && !allowed[name] {
				t.Errorf("tags %q: the core imports %s, which is outside the standard library and the allowed packages", tags, path)
			}
		}
	}
}
